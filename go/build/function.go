package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// FunctionGenerator generates Cloud Function deployment packages
type FunctionGenerator struct {
	handlers   []annotations.Handler
	outputDir  string
	moduleName string
	logger     *zap.Logger
}

// Generate creates deployment packages for all cloud functions
func (fg *FunctionGenerator) Generate() error {
	if len(fg.handlers) == 0 {
		fg.logger.Info("No function handlers to generate")
		return nil
	}

	// Create functions output directory
	if err := os.MkdirAll(fg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create functions directory: %w", err)
	}

	// Generate package for each function
	for _, handler := range fg.handlers {
		if err := fg.generateFunction(handler); err != nil {
			return fmt.Errorf("failed to generate function %s: %w", handler.FunctionName, err)
		}
	}

	fg.logger.Info("Generated all cloud functions",
		zap.Int("count", len(fg.handlers)),
		zap.String("output_dir", fg.outputDir))

	return nil
}

// generateFunction creates a complete deployment package for a single cloud function
func (fg *FunctionGenerator) generateFunction(handler annotations.Handler) error {
	// Create function directory (kebab-case from function name)
	functionDir := filepath.Join(fg.outputDir, toKebabCase(handler.FunctionName))
	if err := os.MkdirAll(functionDir, 0755); err != nil {
		return fmt.Errorf("failed to create function directory: %w", err)
	}

	fg.logger.Info("Generating cloud function",
		zap.String("function", handler.FunctionName),
		zap.String("path", handler.Route.Path),
		zap.String("output_dir", functionDir))

	// Generate files
	if err := fg.generateEntrypoint(functionDir, handler); err != nil {
		return fmt.Errorf("failed to generate entrypoint: %w", err)
	}

	if err := fg.generateGoMod(functionDir, handler); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	if err := fg.generateFunctionYAML(functionDir, handler); err != nil {
		return fmt.Errorf("failed to generate function.yaml: %w", err)
	}

	if err := fg.generateDeployScript(functionDir, handler); err != nil {
		return fmt.Errorf("failed to generate deploy script: %w", err)
	}

	return nil
}

// generateEntrypoint creates the main.go entry point for the cloud function
func (fg *FunctionGenerator) generateEntrypoint(dir string, handler annotations.Handler) error {
	tmpl := template.Must(template.New("entrypoint").Parse(entrypointTemplate))

	file, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		FunctionName string
		PackageName  string
		PackagePath  string
		ModuleName   string
	}{
		FunctionName: handler.FunctionName,
		PackageName:  handler.PackageName,
		PackagePath:  handler.PackagePath,
		ModuleName:   fg.moduleName,
	}

	return tmpl.Execute(file, data)
}

// generateGoMod creates the go.mod file for the cloud function
func (fg *FunctionGenerator) generateGoMod(dir string, handler annotations.Handler) error {
	tmpl := template.Must(template.New("gomod").Parse(goModTemplate))

	file, err := os.Create(filepath.Join(dir, "go.mod"))
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		FunctionName string
		ModuleName   string
	}{
		FunctionName: toKebabCase(handler.FunctionName),
		ModuleName:   fg.moduleName,
	}

	return tmpl.Execute(file, data)
}

// generateFunctionYAML creates the GCP function configuration file
func (fg *FunctionGenerator) generateFunctionYAML(dir string, handler annotations.Handler) error {
	tmpl := template.Must(template.New("function").Parse(functionYAMLTemplate))

	file, err := os.Create(filepath.Join(dir, "function.yaml"))
	if err != nil {
		return err
	}
	defer file.Close()

	// Extract numeric memory value (e.g., "256MB" -> "256Mi")
	memory := handler.Memory
	if memory == "" {
		memory = "256Mi" // Default
	} else {
		// Convert "256MB" to "256Mi" (GCP uses Mi not MB)
		memory = strings.Replace(memory, "MB", "Mi", 1)
	}

	// Convert timeout to seconds
	timeoutSeconds := int(handler.Timeout.Seconds())
	if timeoutSeconds == 0 {
		timeoutSeconds = 60 // Default 60 seconds
	}

	data := struct {
		FunctionName   string
		EntryPoint     string
		Memory         string
		TimeoutSeconds int
		Runtime        string
	}{
		FunctionName:   handler.FunctionName,
		EntryPoint:     handler.FunctionName,
		Memory:         memory,
		TimeoutSeconds: timeoutSeconds,
		Runtime:        "go122", // Go 1.22 runtime
	}

	return tmpl.Execute(file, data)
}

// generateDeployScript creates a deployment script for the function
func (fg *FunctionGenerator) generateDeployScript(dir string, handler annotations.Handler) error {
	tmpl := template.Must(template.New("deploy").Parse(deployScriptTemplate))

	scriptPath := filepath.Join(dir, "deploy.sh")
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
		FunctionName string
		Region       string
		EntryPoint   string
	}{
		FunctionName: toKebabCase(handler.FunctionName),
		Region:       "us-central1", // Default region
		EntryPoint:   handler.FunctionName,
	}

	return tmpl.Execute(file, data)
}

// toKebabCase converts "CreateAccount" to "create-account"
func toKebabCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '-')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// Templates

const entrypointTemplate = `// Code generated by Wylla build system. DO NOT EDIT.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"{{.ModuleName}}/{{.PackagePath}}"
)

var (
	db     *pgxpool.Pool
	logger *zap.Logger
)

func init() {
	var err error

	// Initialize logger
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize database connection pool
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL environment variable is required")
	}

	db, err = pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		logger.Fatal("Failed to create database pool", zap.Error(err))
	}

	logger.Info("Cloud function initialized",
		zap.String("function", "{{.FunctionName}}"))
}

// {{.FunctionName}} is the entry point for the cloud function
func {{.FunctionName}}(w http.ResponseWriter, r *http.Request) {
	// Call the actual handler from the package
	{{.PackageName}}.{{.FunctionName}}(w, r)
}

func main() {
	// Register the function
	funcframework.RegisterHTTPFunction("/", {{.FunctionName}})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Starting function server", zap.String("port", port))
	if err := funcframework.Start(port); err != nil {
		logger.Fatal("Function server failed", zap.Error(err))
	}
}
`

const goModTemplate = `module {{.ModuleName}}/build/functions/{{.FunctionName}}

go 1.22

require (
	github.com/GoogleCloudPlatform/functions-framework-go v1.8.0
	github.com/jackc/pgx/v5 v5.5.0
	go.uber.org/zap v1.26.0
	{{.ModuleName}} v0.0.0
)

replace {{.ModuleName}} => ../../..
`

const functionYAMLTemplate = `# GCP Cloud Function Configuration
# Generated by Wylla build system

name: {{.FunctionName}}
runtime: {{.Runtime}}
entryPoint: {{.EntryPoint}}

# Resource limits
availableMemoryMb: {{.Memory}}
timeout: {{.TimeoutSeconds}}s

# Environment
environmentVariables:
  GO111MODULE: "on"

# Trigger
httpsTrigger:
  securityLevel: SECURE_ALWAYS
`

const deployScriptTemplate = `#!/bin/bash
# Deploy script for {{.FunctionName}}
# Generated by Wylla build system

set -e

# Configuration
FUNCTION_NAME="{{.FunctionName}}"
REGION="{{.Region}}"
ENTRY_POINT="{{.EntryPoint}}"

# Get project ID
PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
    echo "Error: GCP project not set. Run: gcloud config set project PROJECT_ID"
    exit 1
fi

echo "Deploying function: $FUNCTION_NAME to project: $PROJECT_ID"

# Deploy the function
gcloud functions deploy "$FUNCTION_NAME" \
    --gen2 \
    --runtime=go122 \
    --region="$REGION" \
    --source=. \
    --entry-point="$ENTRY_POINT" \
    --trigger-http \
    --allow-unauthenticated \
    --set-env-vars="DATABASE_URL=$DATABASE_URL"

echo "Function deployed successfully!"
echo "URL: https://$REGION-$PROJECT_ID.cloudfunctions.net/$FUNCTION_NAME"
`
