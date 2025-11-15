package typescript

import (
	"fmt"

	"github.com/gravelight-studio/box/annotations"
	"go.uber.org/zap"
)

// Generator orchestrates all TypeScript artifact generation
type Generator struct {
	handlers          []annotations.Handler
	outputDir         string
	moduleName        string
	projectID         string
	region            string
	environment       string
	cleanBuildDir     bool
	logger            *zap.Logger
	functionGenerator *FunctionGenerator
}

// NewGenerator creates a new TypeScript generator
func NewGenerator(handlers []annotations.Handler, outputDir, moduleName, projectID, region, environment string, cleanBuildDir bool, logger *zap.Logger) *Generator {
	return &Generator{
		handlers:          handlers,
		outputDir:         outputDir,
		moduleName:        moduleName,
		projectID:         projectID,
		region:            region,
		environment:       environment,
		cleanBuildDir:     cleanBuildDir,
		logger:            logger,
		functionGenerator: NewFunctionGenerator(handlers, outputDir+"/functions", moduleName, logger),
	}
}

// Generate generates all deployment artifacts
func (g *Generator) Generate() error {
	g.logger.Info("Starting TypeScript build generation",
		zap.Int("totalHandlers", len(g.handlers)),
		zap.String("outputDir", g.outputDir))

	// Count handlers by type
	functionCount := 0
	containerCount := 0
	for _, h := range g.handlers {
		if h.DeploymentType == annotations.DeploymentFunction {
			functionCount++
		} else if h.DeploymentType == annotations.DeploymentContainer {
			containerCount++
		}
	}

	// Generate Cloud Functions
	g.logger.Info("Generating cloud functions", zap.Int("count", functionCount))
	if err := g.functionGenerator.Generate(); err != nil {
		return fmt.Errorf("failed to generate functions: %w", err)
	}

	// TODO: Generate Cloud Run containers, API Gateway, and Terraform
	// For now, we're focusing on functions to get the CLI working

	g.logger.Info("TypeScript build generation complete",
		zap.Int("functionsGenerated", functionCount),
		zap.Int("containersGenerated", containerCount))

	return nil
}
