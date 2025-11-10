package build

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// ContainerGenerator generates Cloud Run container deployment packages
type ContainerGenerator struct {
	handlers   []annotations.Handler
	outputDir  string
	moduleName string
	logger     *zap.Logger
}

// ServiceGroup represents a group of handlers that will be deployed together
type ServiceGroup struct {
	Name     string
	Handlers []annotations.Handler
}

// Generate creates deployment packages for all container services
func (cg *ContainerGenerator) Generate() error {
	if len(cg.handlers) == 0 {
		cg.logger.Info("No container handlers to generate")
		return nil
	}

	// Create containers output directory
	if err := os.MkdirAll(cg.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create containers directory: %w", err)
	}

	// Group handlers by service
	serviceGroups := cg.groupHandlers()

	cg.logger.Info("Grouped container handlers",
		zap.Int("total_handlers", len(cg.handlers)),
		zap.Int("service_groups", len(serviceGroups)))

	// Generate package for each service group
	for _, group := range serviceGroups {
		if err := cg.generateService(group); err != nil {
			return fmt.Errorf("failed to generate service %s: %w", group.Name, err)
		}
	}

	cg.logger.Info("Generated all container services",
		zap.Int("count", len(serviceGroups)),
		zap.String("output_dir", cg.outputDir))

	return nil
}

// groupHandlers groups handlers by service name from annotations
func (cg *ContainerGenerator) groupHandlers() []ServiceGroup {
	serviceMap := make(map[string][]annotations.Handler)

	for _, handler := range cg.handlers {
		// Get service name from handler metadata
		// If handler has service specified in some future field, use it
		// For now, use package name as service grouping
		serviceName := handler.PackageName
		if serviceName == "" {
			serviceName = "default"
		}

		serviceMap[serviceName] = append(serviceMap[serviceName], handler)
	}

	// Convert map to slice
	var groups []ServiceGroup
	for name, handlers := range serviceMap {
		groups = append(groups, ServiceGroup{
			Name:     name,
			Handlers: handlers,
		})
	}

	return groups
}

// generateService creates a complete deployment package for a service group
func (cg *ContainerGenerator) generateService(group ServiceGroup) error {
	// Create service directory (kebab-case)
	serviceDir := filepath.Join(cg.outputDir, toKebabCase(group.Name))
	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	cg.logger.Info("Generating container service",
		zap.String("service", group.Name),
		zap.Int("handlers", len(group.Handlers)),
		zap.String("output_dir", serviceDir))

	// Generate files
	if err := cg.generateServerMain(serviceDir, group); err != nil {
		return fmt.Errorf("failed to generate main.go: %w", err)
	}

	if err := cg.generateDockerfile(serviceDir, group); err != nil {
		return fmt.Errorf("failed to generate Dockerfile: %w", err)
	}

	if err := cg.generateCloudBuild(serviceDir, group); err != nil {
		return fmt.Errorf("failed to generate cloudbuild.yaml: %w", err)
	}

	if err := cg.generateDeployScript(serviceDir, group); err != nil {
		return fmt.Errorf("failed to generate deploy script: %w", err)
	}

	return nil
}

// generateServerMain creates the main.go file for the multi-handler server
func (cg *ContainerGenerator) generateServerMain(dir string, group ServiceGroup) error {
	tmpl := template.Must(template.New("servermain").Parse(serverMainTemplate))

	file, err := os.Create(filepath.Join(dir, "main.go"))
	if err != nil {
		return err
	}
	defer file.Close()

	// Get unique package imports
	packageImports := make(map[string]string)
	for _, h := range group.Handlers {
		if h.PackagePath != "" {
			packageImports[h.PackageName] = h.PackagePath
		}
	}

	data := struct {
		ServiceName    string
		ModuleName     string
		Handlers       []annotations.Handler
		PackageImports map[string]string
	}{
		ServiceName:    group.Name,
		ModuleName:     cg.moduleName,
		Handlers:       group.Handlers,
		PackageImports: packageImports,
	}

	return tmpl.Execute(file, data)
}

// generateDockerfile creates a multi-stage Dockerfile
func (cg *ContainerGenerator) generateDockerfile(dir string, group ServiceGroup) error {
	tmpl := template.Must(template.New("dockerfile").Parse(dockerfileTemplate))

	file, err := os.Create(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		ServiceName string
		ModuleName  string
	}{
		ServiceName: group.Name,
		ModuleName:  cg.moduleName,
	}

	return tmpl.Execute(file, data)
}

// generateCloudBuild creates cloudbuild.yaml for GCP Cloud Build
func (cg *ContainerGenerator) generateCloudBuild(dir string, group ServiceGroup) error {
	tmpl := template.Must(template.New("cloudbuild").Parse(cloudBuildTemplate))

	file, err := os.Create(filepath.Join(dir, "cloudbuild.yaml"))
	if err != nil {
		return err
	}
	defer file.Close()

	serviceName := toKebabCase(group.Name)

	data := struct {
		ServiceName string
		Region      string
	}{
		ServiceName: serviceName,
		Region:      "us-central1",
	}

	return tmpl.Execute(file, data)
}

// generateDeployScript creates a deployment script
func (cg *ContainerGenerator) generateDeployScript(dir string, group ServiceGroup) error {
	tmpl := template.Must(template.New("deploy").Parse(containerDeployScriptTemplate))

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

	serviceName := toKebabCase(group.Name)

	data := struct {
		ServiceName string
		Region      string
	}{
		ServiceName: serviceName,
		Region:      "us-central1",
	}

	return tmpl.Execute(file, data)
}

// Templates

const serverMainTemplate = `// Code generated by Wylla build system. DO NOT EDIT.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
{{range $pkg, $path := .PackageImports}}
	"{{$.ModuleName}}/{{$path}}"
{{end}}
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

	logger.Info("Container service initialized",
		zap.String("service", "{{.ServiceName}}"))
}

func main() {
	// Create router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Register handlers
{{range .Handlers}}
	r.Method("{{.Route.Method}}", "{{.Route.Path}}", http.HandlerFunc({{.PackageName}}.{{.FunctionName}}))
{{end}}

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", zap.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// Close database connection
	db.Close()

	logger.Info("Server stopped")
}
`

const dockerfileTemplate = `# Multi-stage Dockerfile for {{.ServiceName}} service
# Generated by Wylla build system

# Stage 1: Build
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY ../../../go.mod ../../../go.sum ./
RUN go mod download

# Copy source code
COPY ../../../ .

# Build the service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./build/containers/{{.ServiceName}}/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /app/server

# Use non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the service
ENTRYPOINT ["/app/server"]
`

const cloudBuildTemplate = `# Cloud Build configuration for {{.ServiceName}}
# Generated by Wylla build system

steps:
  # Build the container image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '-t'
      - 'gcr.io/$PROJECT_ID/{{.ServiceName}}:$SHORT_SHA'
      - '-t'
      - 'gcr.io/$PROJECT_ID/{{.ServiceName}}:latest'
      - '-f'
      - './build/containers/{{.ServiceName}}/Dockerfile'
      - '.'

  # Push the container image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/$PROJECT_ID/{{.ServiceName}}:$SHORT_SHA'

  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/$PROJECT_ID/{{.ServiceName}}:latest'

  # Deploy to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: gcloud
    args:
      - 'run'
      - 'deploy'
      - '{{.ServiceName}}'
      - '--image=gcr.io/$PROJECT_ID/{{.ServiceName}}:$SHORT_SHA'
      - '--region={{.Region}}'
      - '--platform=managed'
      - '--allow-unauthenticated'
      - '--set-env-vars=DATABASE_URL=$$DATABASE_URL'
    secretEnv: ['DATABASE_URL']

availableSecrets:
  secretManager:
    - versionName: projects/$PROJECT_ID/secrets/database-url/versions/latest
      env: 'DATABASE_URL'

images:
  - 'gcr.io/$PROJECT_ID/{{.ServiceName}}:$SHORT_SHA'
  - 'gcr.io/$PROJECT_ID/{{.ServiceName}}:latest'

options:
  machineType: 'N1_HIGHCPU_8'
  logging: CLOUD_LOGGING_ONLY
`

const containerDeployScriptTemplate = `#!/bin/bash
# Deploy script for {{.ServiceName}} container
# Generated by Wylla build system

set -e

# Configuration
SERVICE_NAME="{{.ServiceName}}"
REGION="{{.Region}}"

# Get project ID
PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
    echo "Error: GCP project not set. Run: gcloud config set project PROJECT_ID"
    exit 1
fi

echo "Deploying container service: $SERVICE_NAME to project: $PROJECT_ID"

# Submit build to Cloud Build
gcloud builds submit \
    --config=cloudbuild.yaml \
    --substitutions=SHORT_SHA=$(git rev-parse --short HEAD) \
    ../../..

echo "Service deployed successfully!"
echo "URL: https://$SERVICE_NAME-$(gcloud run services describe $SERVICE_NAME --region=$REGION --format='value(status.url)' | sed 's/https\?:\/\///')"
`
