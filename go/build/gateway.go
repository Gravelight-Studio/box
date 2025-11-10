package build

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// GatewayGenerator generates OpenAPI specifications and GCP API Gateway configurations
type GatewayGenerator struct {
	handlers   []annotations.Handler
	outputDir  string
	moduleName string
	projectID  string // GCP project ID
	region     string // GCP region for backends
	logger     *zap.Logger
}

// OpenAPIPath represents a path in the OpenAPI spec with its operations
type OpenAPIPath struct {
	Path       string
	Operations map[string]*OpenAPIOperation // method -> operation
}

// OpenAPIOperation represents a single operation (method) on a path
type OpenAPIOperation struct {
	OperationID string
	Summary     string
	Tags        []string
	Security    []map[string][]string
	Parameters  []OpenAPIParameter
	Responses   map[string]OpenAPIResponse
	XGoogle     map[string]interface{} // GCP extensions
}

// OpenAPIParameter represents a path/query parameter
type OpenAPIParameter struct {
	Name     string
	In       string // "path", "query", "header"
	Required bool
	Schema   map[string]string
}

// OpenAPIResponse represents a response definition
type OpenAPIResponse struct {
	Description string
	Content     map[string]interface{}
}

// Generate creates OpenAPI spec and API Gateway configuration
func (gg *GatewayGenerator) Generate() error {
	if len(gg.handlers) == 0 {
		gg.logger.Info("No handlers to generate gateway configuration")
		return nil
	}

	// Create gateway output directory
	if err := os.MkdirAll(gg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create gateway directory: %w", err)
	}

	gg.logger.Info("Generating API Gateway configuration",
		zap.Int("handlers", len(gg.handlers)),
		zap.String("output_dir", gg.outputDir))

	// Generate OpenAPI specification
	if err := gg.generateOpenAPISpec(); err != nil {
		return fmt.Errorf("failed to generate OpenAPI spec: %w", err)
	}

	// Generate API Gateway config
	if err := gg.generateGatewayConfig(); err != nil {
		return fmt.Errorf("failed to generate gateway config: %w", err)
	}

	// Generate deployment script
	if err := gg.generateDeployScript(); err != nil {
		return fmt.Errorf("failed to generate deploy script: %w", err)
	}

	gg.logger.Info("Generated API Gateway configuration",
		zap.String("openapi_spec", filepath.Join(gg.outputDir, "openapi.yaml")))

	return nil
}

// generateOpenAPISpec creates the OpenAPI 3.0 specification
func (gg *GatewayGenerator) generateOpenAPISpec() error {
	tmpl := template.Must(template.New("openapi").Funcs(template.FuncMap{
		"join":           strings.Join,
		"formatSecurity": gg.formatSecurity,
		"hasParameters":  gg.hasPathParameters,
		"extractParams":  gg.extractPathParameters,
		"backendURL":     gg.getBackendURL,
		"getRateLimit":   gg.getRateLimitQuota,
		"getTimeout":     gg.getTimeoutSeconds,
	}).Parse(openAPITemplate))

	file, err := os.Create(filepath.Join(gg.outputDir, "openapi.yaml"))
	if err != nil {
		return err
	}
	defer file.Close()

	// Group handlers by path
	paths := gg.groupHandlersByPath()

	// Get all unique tags (package names)
	tags := gg.extractTags()

	// Determine if we need security definitions
	needsAuth := gg.hasAuthentication()

	data := struct {
		Title      string
		Version    string
		Paths      []OpenAPIPath
		Tags       []string
		NeedsAuth  bool
		ProjectID  string
		Region     string
		ModuleName string
	}{
		Title:      "Wylla API",
		Version:    "1.0.0",
		Paths:      paths,
		Tags:       tags,
		NeedsAuth:  needsAuth,
		ProjectID:  gg.projectID,
		Region:     gg.region,
		ModuleName: gg.moduleName,
	}

	return tmpl.Execute(file, data)
}

// groupHandlersByPath groups handlers by their route path
func (gg *GatewayGenerator) groupHandlersByPath() []OpenAPIPath {
	pathMap := make(map[string]*OpenAPIPath)

	for _, handler := range gg.handlers {
		path := handler.Route.Path
		if _, exists := pathMap[path]; !exists {
			pathMap[path] = &OpenAPIPath{
				Path:       path,
				Operations: make(map[string]*OpenAPIOperation),
			}
		}

		// Create operation for this method
		method := strings.ToLower(handler.Route.Method)
		pathMap[path].Operations[method] = &OpenAPIOperation{
			OperationID: handler.FunctionName,
			Summary:     fmt.Sprintf("%s %s", handler.Route.Method, handler.Route.Path),
			Tags:        []string{handler.PackageName},
			Security:    gg.buildSecurityRequirement(handler),
			Parameters:  gg.buildParameters(handler),
			Responses:   gg.buildResponses(handler),
			XGoogle:     gg.buildGCPExtensions(handler),
		}
	}

	// Convert map to sorted slice
	paths := make([]OpenAPIPath, 0, len(pathMap))
	for _, path := range pathMap {
		paths = append(paths, *path)
	}

	// Sort by path for consistent output
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Path < paths[j].Path
	})

	return paths
}

// buildSecurityRequirement creates security requirements based on auth config
func (gg *GatewayGenerator) buildSecurityRequirement(handler annotations.Handler) []map[string][]string {
	if handler.Auth.Type == annotations.AuthNone {
		return nil
	}

	// Both required and optional auth use bearerAuth scheme
	// The difference is handled at the middleware level
	return []map[string][]string{
		{"bearerAuth": []string{}},
	}
}

// buildParameters extracts path parameters from the route
func (gg *GatewayGenerator) buildParameters(handler annotations.Handler) []OpenAPIParameter {
	var params []OpenAPIParameter

	// Extract path parameters (e.g., {id} from /api/v1/accounts/{id})
	pathParams := extractPathParams(handler.Route.Path)
	for _, param := range pathParams {
		params = append(params, OpenAPIParameter{
			Name:     param,
			In:       "path",
			Required: true,
			Schema: map[string]string{
				"type": "string",
			},
		})
	}

	return params
}

// buildResponses creates standard response definitions
func (gg *GatewayGenerator) buildResponses(handler annotations.Handler) map[string]OpenAPIResponse {
	responses := map[string]OpenAPIResponse{
		"200": {
			Description: "Successful response",
		},
		"400": {
			Description: "Bad request",
		},
		"500": {
			Description: "Internal server error",
		},
	}

	// Add auth-specific responses
	if handler.Auth.Type != annotations.AuthNone {
		responses["401"] = OpenAPIResponse{
			Description: "Unauthorized - missing or invalid authentication",
		}
		responses["403"] = OpenAPIResponse{
			Description: "Forbidden - insufficient permissions",
		}
	}

	// Add rate limit response
	if handler.RateLimit != nil {
		responses["429"] = OpenAPIResponse{
			Description: "Too many requests - rate limit exceeded",
		}
	}

	// POST requests typically return 201 for creation
	if handler.Route.Method == "POST" {
		responses["201"] = OpenAPIResponse{
			Description: "Resource created successfully",
		}
	}

	return responses
}

// buildGCPExtensions creates GCP-specific OpenAPI extensions
func (gg *GatewayGenerator) buildGCPExtensions(handler annotations.Handler) map[string]interface{} {
	extensions := make(map[string]interface{})

	// Backend address
	extensions["backend"] = map[string]interface{}{
		"address": gg.getBackendURL(handler),
	}

	// Rate limiting (if configured)
	if handler.RateLimit != nil {
		extensions["quota"] = map[string]interface{}{
			"limit": handler.RateLimit.Count,
			"interval": handler.RateLimit.Period.String(),
		}
	}

	// Timeout (if configured)
	if handler.Timeout > 0 {
		extensions["timeout"] = fmt.Sprintf("%.0fs", handler.Timeout.Seconds())
	}

	// CORS (if configured)
	if handler.CORS != nil {
		extensions["cors"] = map[string]interface{}{
			"allowOrigins": handler.CORS.AllowedOrigins,
			"allowMethods": []string{handler.Route.Method},
		}
	}

	return extensions
}

// getBackendURL determines the backend URL based on deployment type
func (gg *GatewayGenerator) getBackendURL(handler annotations.Handler) string {
	functionName := toKebabCase(handler.FunctionName)

	switch handler.DeploymentType {
	case annotations.DeploymentFunction:
		// Cloud Function URL format
		return fmt.Sprintf("https://%s-%s.cloudfunctions.net/%s",
			gg.region, gg.projectID, functionName)
	case annotations.DeploymentContainer:
		// Cloud Run URL format (service name is package name)
		serviceName := toKebabCase(handler.PackageName)
		return fmt.Sprintf("https://%s-%s.run.app",
			serviceName, gg.region)
	default:
		return ""
	}
}

// extractTags gets unique package names as tags
func (gg *GatewayGenerator) extractTags() []string {
	tagMap := make(map[string]bool)
	for _, handler := range gg.handlers {
		if handler.PackageName != "" {
			tagMap[handler.PackageName] = true
		}
	}

	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// hasAuthentication checks if any handler requires authentication
func (gg *GatewayGenerator) hasAuthentication() bool {
	for _, handler := range gg.handlers {
		if handler.Auth.Type != annotations.AuthNone {
			return true
		}
	}
	return false
}

// formatSecurity formats security requirement for template
func (gg *GatewayGenerator) formatSecurity(handler annotations.Handler) string {
	if handler.Auth.Type == annotations.AuthNone {
		return "[]"
	}
	return "- bearerAuth: []"
}

// hasPathParameters checks if a path contains parameters
func (gg *GatewayGenerator) hasPathParameters(path string) bool {
	return strings.Contains(path, "{")
}

// extractPathParameters extracts parameter definitions for template
func (gg *GatewayGenerator) extractPathParameters(handler annotations.Handler) []OpenAPIParameter {
	return gg.buildParameters(handler)
}

// getRateLimitQuota formats rate limit for display
func (gg *GatewayGenerator) getRateLimitQuota(handler annotations.Handler) string {
	if handler.RateLimit == nil {
		return "none"
	}
	return fmt.Sprintf("%d requests per %s", handler.RateLimit.Count, handler.RateLimit.Period)
}

// getTimeoutSeconds returns timeout in seconds
func (gg *GatewayGenerator) getTimeoutSeconds(handler annotations.Handler) int {
	if handler.Timeout > 0 {
		return int(handler.Timeout.Seconds())
	}
	return 60 // default
}

// extractPathParams extracts parameter names from a path
func extractPathParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			param := strings.TrimPrefix(part, "{")
			param = strings.TrimSuffix(param, "}")
			params = append(params, param)
		}
	}
	return params
}

// generateGatewayConfig creates the GCP API Gateway configuration
func (gg *GatewayGenerator) generateGatewayConfig() error {
	tmpl := template.Must(template.New("gatewayconfig").Parse(gatewayConfigTemplate))

	file, err := os.Create(filepath.Join(gg.outputDir, "gateway-config.yaml"))
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		ProjectID string
		Region    string
		APIName   string
	}{
		ProjectID: gg.projectID,
		Region:    gg.region,
		APIName:   "wylla-api",
	}

	return tmpl.Execute(file, data)
}

// generateDeployScript creates deployment script for API Gateway
func (gg *GatewayGenerator) generateDeployScript() error {
	tmpl := template.Must(template.New("deploy").Parse(gatewayDeployScriptTemplate))

	scriptPath := filepath.Join(gg.outputDir, "deploy.sh")
	file, err := os.Create(scriptPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return err
	}

	data := struct {
		APIName string
		Region  string
	}{
		APIName: "wylla-api",
		Region:  gg.region,
	}

	return tmpl.Execute(file, data)
}

// Templates

const openAPITemplate = `# OpenAPI 3.0 Specification
# Generated by Wylla build system
# DO NOT EDIT - This file is auto-generated

openapi: 3.0.0
info:
  title: {{.Title}}
  description: API specification generated from Wylla handler annotations
  version: {{.Version}}
  contact:
    name: API Support

servers:
  - url: https://{{.Region}}-{{.ProjectID}}.gateway.dev
    description: Production API Gateway

{{if .NeedsAuth}}
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT Bearer token authentication
{{end}}

{{if .Tags}}
tags:
{{range .Tags}}  - name: {{.}}
    description: {{.}} endpoints
{{end}}
{{end}}

paths:
{{range .Paths}}
  {{.Path}}:
{{range $method, $op := .Operations}}    {{$method}}:
      operationId: {{$op.OperationID}}
      summary: {{$op.Summary}}
      tags:
{{range $op.Tags}}        - {{.}}
{{end}}
{{if $op.Security}}      security:
{{range $op.Security}}        - bearerAuth: []
{{end}}
{{end}}
{{if $op.Parameters}}      parameters:
{{range $op.Parameters}}        - name: {{.Name}}
          in: {{.In}}
          required: {{.Required}}
          schema:
            type: {{index .Schema "type"}}
{{end}}
{{end}}
      responses:
{{range $code, $response := $op.Responses}}        '{{$code}}':
          description: {{$response.Description}}
{{end}}
      x-google-backend:
        address: {{index $op.XGoogle "backend" "address"}}
        deadline: {{if index $op.XGoogle "timeout"}}{{index $op.XGoogle "timeout"}}{{else}}60.0{{end}}
{{if index $op.XGoogle "quota"}}      x-google-quota:
        metricCosts:
          "{{$op.OperationID}}-quota": {{index $op.XGoogle "quota" "limit"}}
{{end}}
{{end}}
{{end}}
`

const gatewayConfigTemplate = `# GCP API Gateway Configuration
# Generated by Wylla build system

apiVersion: apigateway.cnrm.cloud.google.com/v1beta1
kind: ApiGatewayAPI
metadata:
  name: {{.APIName}}
spec:
  projectRef:
    external: {{.ProjectID}}
---
apiVersion: apigateway.cnrm.cloud.google.com/v1beta1
kind: ApiGatewayAPIConfig
metadata:
  name: {{.APIName}}-config
spec:
  projectRef:
    external: {{.ProjectID}}
  apiRef:
    name: {{.APIName}}
  openapiDocuments:
    - document:
        path: openapi.yaml
  gatewayServiceAccount:
    serviceAccountRef:
      name: api-gateway-sa
---
apiVersion: apigateway.cnrm.cloud.google.com/v1beta1
kind: ApiGatewayGateway
metadata:
  name: {{.APIName}}-gateway
spec:
  projectRef:
    external: {{.ProjectID}}
  location: {{.Region}}
  apiConfigRef:
    name: {{.APIName}}-config
`

const gatewayDeployScriptTemplate = `#!/bin/bash
# Deploy script for API Gateway
# Generated by Wylla build system

set -e

# Configuration
API_NAME="{{.APIName}}"
REGION="{{.Region}}"

# Get project ID
PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
    echo "Error: GCP project not set. Run: gcloud config set project PROJECT_ID"
    exit 1
fi

echo "Deploying API Gateway: $API_NAME to project: $PROJECT_ID"

# Create API if it doesn't exist
echo "Creating API..."
gcloud api-gateway apis describe "$API_NAME" --project="$PROJECT_ID" 2>/dev/null || \
  gcloud api-gateway apis create "$API_NAME" \
    --project="$PROJECT_ID"

# Create API config
echo "Creating API config..."
CONFIG_ID="$API_NAME-config-$(date +%s)"
gcloud api-gateway api-configs create "$CONFIG_ID" \
  --api="$API_NAME" \
  --openapi-spec=openapi.yaml \
  --project="$PROJECT_ID" \
  --backend-auth-service-account="api-gateway@$PROJECT_ID.iam.gserviceaccount.com"

# Create or update gateway
echo "Creating/updating gateway..."
gcloud api-gateway gateways describe "$API_NAME-gateway" \
  --location="$REGION" \
  --project="$PROJECT_ID" 2>/dev/null && \
  gcloud api-gateway gateways update "$API_NAME-gateway" \
    --api="$API_NAME" \
    --api-config="$CONFIG_ID" \
    --location="$REGION" \
    --project="$PROJECT_ID" || \
  gcloud api-gateway gateways create "$API_NAME-gateway" \
    --api="$API_NAME" \
    --api-config="$CONFIG_ID" \
    --location="$REGION" \
    --project="$PROJECT_ID"

echo ""
echo "✓ API Gateway deployed successfully!"
echo ""
echo "Gateway URL:"
gcloud api-gateway gateways describe "$API_NAME-gateway" \
  --location="$REGION" \
  --project="$PROJECT_ID" \
  --format="value(defaultHostname)"
`
