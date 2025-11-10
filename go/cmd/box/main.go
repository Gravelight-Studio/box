package main

import (
	"flag"
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
	"github.com/gravelight-studio/box/build"
)

const version = "0.1.0"

func main() {
	// Define flags
	handlersDir := flag.String("handlers", "./handlers", "Path to handlers directory")
	outputDir := flag.String("output", "./build", "Path to output directory")
	projectID := flag.String("project", "", "GCP project ID (required)")
	region := flag.String("region", "us-central1", "GCP region")
	environment := flag.String("env", "dev", "Environment (dev, staging, production)")
	moduleName := flag.String("module", "", "Go module name (auto-detected from go.mod if not provided)")
	clean := flag.Bool("clean", false, "Clean build directory before generating")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Box - Deployment artifact generator for annotation-driven Go applications\n\n")
		fmt.Fprintf(os.Stderr, "Usage: box [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  box --handlers ./handlers --output ./build --project my-gcp-project\n\n")
	}

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("Box version %s\n", version)
		os.Exit(0)
	}

	// Validate required flags
	if *projectID == "" {
		fmt.Fprintf(os.Stderr, "Error: --project flag is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create logger
	var logger *zap.Logger
	var err error
	if *verbose {
		logger, err = zap.NewDevelopment()
	} else {
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logger, err = config.Build()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Box deployment generator",
		zap.String("version", version),
		zap.String("handlers", *handlersDir),
		zap.String("output", *outputDir),
		zap.String("project", *projectID),
		zap.String("region", *region),
		zap.String("environment", *environment))

	// Parse annotations from handlers directory
	logger.Info("Parsing handlers", zap.String("directory", *handlersDir))
	parser := annotations.NewParser()
	parsed, err := parser.ParseDirectory(*handlersDir)
	if err != nil {
		logger.Fatal("Failed to parse handlers", zap.Error(err))
	}

	// Check for parse errors
	if len(parsed.Errors) > 0 {
		logger.Warn("Encountered parse errors", zap.Int("count", len(parsed.Errors)))
		for _, parseErr := range parsed.Errors {
			logger.Warn("Parse error",
				zap.String("file", parseErr.FilePath),
				zap.Int("line", parseErr.LineNumber),
				zap.String("message", parseErr.Message))
		}
	}

	logger.Info("Found handlers",
		zap.Int("total", len(parsed.Handlers)),
		zap.Int("functions", countFunctions(parsed.Handlers)),
		zap.Int("containers", countContainers(parsed.Handlers)))

	if len(parsed.Handlers) == 0 {
		logger.Warn("No handlers found with @box: annotations")
		os.Exit(0)
	}

	// Auto-detect module name if not provided
	if *moduleName == "" {
		detectedModule, err := detectModuleName()
		if err != nil {
			logger.Fatal("Failed to detect module name. Please specify --module flag", zap.Error(err))
		}
		*moduleName = detectedModule
		logger.Info("Detected module name", zap.String("module", *moduleName))
	}

	// Create generator
	logger.Info("Creating deployment artifacts")
	generator := build.NewGenerator(build.Config{
		Handlers:      parsed.Handlers,
		OutputDir:     *outputDir,
		ModuleName:    *moduleName,
		ProjectID:     *projectID,
		Region:        *region,
		Environment:   *environment,
		Logger:        logger,
		CleanBuildDir: *clean,
	})

	// Generate all artifacts
	if err := generator.Generate(); err != nil {
		logger.Fatal("Failed to generate artifacts", zap.Error(err))
	}

	logger.Info("✓ Deployment artifacts generated successfully",
		zap.String("output", *outputDir))

	// Print summary
	fmt.Printf("\n✓ Success! Generated deployment artifacts:\n")
	fmt.Printf("  • Cloud Functions: %s/functions/\n", *outputDir)
	fmt.Printf("  • Cloud Run Containers: %s/containers/\n", *outputDir)
	fmt.Printf("  • API Gateway: %s/gateway/\n", *outputDir)
	fmt.Printf("  • Terraform IaC: %s/terraform/\n", *outputDir)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review generated files in %s/\n", *outputDir)
	fmt.Printf("  2. Deploy with: cd %s/terraform && terraform init && terraform apply\n", *outputDir)
}

// detectModuleName reads the go.mod file to determine the module name
func detectModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("go.mod not found: %w", err)
	}

	// Parse module line
	lines := []byte(data)
	for i := 0; i < len(lines); i++ {
		if lines[i] == '\n' || i == len(lines)-1 {
			line := string(lines[:i])
			if len(line) > 7 && line[:7] == "module " {
				return line[7:], nil
			}
			lines = lines[i+1:]
			i = 0
		}
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

// countFunctions counts handlers with function deployment type
func countFunctions(handlers []annotations.Handler) int {
	count := 0
	for _, h := range handlers {
		if h.DeploymentType == annotations.DeploymentFunction {
			count++
		}
	}
	return count
}

// countContainers counts handlers with container deployment type
func countContainers(handlers []annotations.Handler) int {
	count := 0
	for _, h := range handlers {
		if h.DeploymentType == annotations.DeploymentContainer {
			count++
		}
	}
	return count
}
