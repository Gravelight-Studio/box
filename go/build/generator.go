package build

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// Generator orchestrates the build process for cloud deployments
type Generator struct {
	handlers            []annotations.Handler
	outputDir           string
	moduleName          string // e.g., "github.com/gravelight-studio/box"
	logger              *zap.Logger
	funcGenerator       *FunctionGenerator
	containerGenerator  *ContainerGenerator
	gatewayGenerator    *GatewayGenerator
	terraformGenerator  *TerraformGenerator
	cleanBuildDir       bool
}

// Config holds generator configuration
type Config struct {
	Handlers      []annotations.Handler
	OutputDir     string // e.g., "./build"
	ModuleName    string // e.g., "github.com/gravelight-studio/box"
	ProjectID     string // GCP project ID (e.g., "my-project-123")
	Region        string // GCP region (e.g., "us-central1")
	Environment   string // Environment name (e.g., "dev", "staging", "production")
	Logger        *zap.Logger
	CleanBuildDir bool // If true, removes existing build directory before generating
}

// NewGenerator creates a new build generator
func NewGenerator(config Config) *Generator {
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}

	if config.OutputDir == "" {
		config.OutputDir = "./build"
	}

	if config.ProjectID == "" {
		config.ProjectID = "PROJECT_ID" // Placeholder
	}

	if config.Region == "" {
		config.Region = "us-central1" // Default region
	}

	if config.Environment == "" {
		config.Environment = "dev" // Default environment
	}

	g := &Generator{
		handlers:      config.Handlers,
		outputDir:     config.OutputDir,
		moduleName:    config.ModuleName,
		logger:        config.Logger,
		cleanBuildDir: config.CleanBuildDir,
	}

	// Initialize function generator
	g.funcGenerator = &FunctionGenerator{
		handlers:   filterFunctionHandlers(config.Handlers),
		outputDir:  filepath.Join(config.OutputDir, "functions"),
		moduleName: config.ModuleName,
		logger:     config.Logger,
	}

	// Initialize container generator
	g.containerGenerator = &ContainerGenerator{
		handlers:   filterContainerHandlers(config.Handlers),
		outputDir:  filepath.Join(config.OutputDir, "containers"),
		moduleName: config.ModuleName,
		logger:     config.Logger,
	}

	// Initialize gateway generator
	g.gatewayGenerator = &GatewayGenerator{
		handlers:   config.Handlers, // Gateway needs all handlers
		outputDir:  filepath.Join(config.OutputDir, "gateway"),
		moduleName: config.ModuleName,
		projectID:  config.ProjectID,
		region:     config.Region,
		logger:     config.Logger,
	}

	// Initialize terraform generator
	g.terraformGenerator = &TerraformGenerator{
		handlers:    config.Handlers,
		outputDir:   filepath.Join(config.OutputDir, "terraform"),
		moduleName:  config.ModuleName,
		projectID:   config.ProjectID,
		region:      config.Region,
		environment: config.Environment,
		logger:      config.Logger,
	}

	return g
}

// Generate runs the complete build process
func (g *Generator) Generate() error {
	g.logger.Info("Starting build generation",
		zap.Int("total_handlers", len(g.handlers)),
		zap.String("output_dir", g.outputDir))

	// Clean build directory if requested
	if g.cleanBuildDir {
		if err := os.RemoveAll(g.outputDir); err != nil {
			return fmt.Errorf("failed to clean build directory: %w", err)
		}
		g.logger.Info("Cleaned build directory", zap.String("dir", g.outputDir))
	}

	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate cloud functions
	functionCount := len(g.funcGenerator.handlers)
	if functionCount > 0 {
		g.logger.Info("Generating cloud functions", zap.Int("count", functionCount))
		if err := g.funcGenerator.Generate(); err != nil {
			return fmt.Errorf("failed to generate cloud functions: %w", err)
		}
	} else {
		g.logger.Info("No cloud functions to generate")
	}

	// Generate cloud run containers
	containerCount := len(g.containerGenerator.handlers)
	if containerCount > 0 {
		g.logger.Info("Generating cloud run containers", zap.Int("handlers", containerCount))
		if err := g.containerGenerator.Generate(); err != nil {
			return fmt.Errorf("failed to generate cloud run containers: %w", err)
		}
	} else {
		g.logger.Info("No cloud run containers to generate")
	}

	// Generate API Gateway configuration
	totalHandlers := len(g.handlers)
	if totalHandlers > 0 {
		g.logger.Info("Generating API Gateway configuration", zap.Int("handlers", totalHandlers))
		if err := g.gatewayGenerator.Generate(); err != nil {
			return fmt.Errorf("failed to generate API Gateway configuration: %w", err)
		}
	} else {
		g.logger.Info("No handlers to generate API Gateway configuration")
	}

	// Generate Terraform infrastructure configuration
	if totalHandlers > 0 {
		g.logger.Info("Generating Terraform infrastructure", zap.Int("handlers", totalHandlers))
		if err := g.terraformGenerator.Generate(); err != nil {
			return fmt.Errorf("failed to generate Terraform infrastructure: %w", err)
		}
	} else {
		g.logger.Info("No handlers to generate Terraform infrastructure")
	}

	g.logger.Info("Build generation complete",
		zap.Int("functions_generated", functionCount),
		zap.Int("container_handlers", containerCount),
		zap.Int("total_api_endpoints", totalHandlers))

	return nil
}

// GenerateFunctions generates only cloud function packages
func (g *Generator) GenerateFunctions() error {
	return g.funcGenerator.Generate()
}

// GenerateContainers generates only cloud run container packages
func (g *Generator) GenerateContainers() error {
	return g.containerGenerator.Generate()
}

// GenerateGateway generates only API Gateway configuration
func (g *Generator) GenerateGateway() error {
	return g.gatewayGenerator.Generate()
}

// GenerateTerraform generates only Terraform infrastructure configuration
func (g *Generator) GenerateTerraform() error {
	return g.terraformGenerator.Generate()
}

// GetFunctionHandlers returns handlers marked for cloud function deployment
func (g *Generator) GetFunctionHandlers() []annotations.Handler {
	return g.funcGenerator.handlers
}

// GetContainerHandlers returns handlers marked for container deployment
func (g *Generator) GetContainerHandlers() []annotations.Handler {
	return filterContainerHandlers(g.handlers)
}

// GetAllHandlers returns all handlers
func (g *Generator) GetAllHandlers() []annotations.Handler {
	return g.handlers
}

// filterFunctionHandlers returns only handlers marked for function deployment
func filterFunctionHandlers(handlers []annotations.Handler) []annotations.Handler {
	var functions []annotations.Handler
	for _, h := range handlers {
		if h.DeploymentType == annotations.DeploymentFunction {
			functions = append(functions, h)
		}
	}
	return functions
}

// filterContainerHandlers returns only handlers marked for container deployment
func filterContainerHandlers(handlers []annotations.Handler) []annotations.Handler {
	var containers []annotations.Handler
	for _, h := range handlers {
		if h.DeploymentType == annotations.DeploymentContainer {
			containers = append(containers, h)
		}
	}
	return containers
}
