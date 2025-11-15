package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gravelight-studio/box-cli/typescript"
	"github.com/manifoldco/promptui"
	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
	"github.com/gravelight-studio/box/build"
)

//go:embed templates/*
var templatesFS embed.FS

const version = "0.1.0"

type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		initCommand()
	case "build":
		buildCommand()
	case "version", "--version", "-v":
		fmt.Printf("Box version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Box - Universal deployment framework for serverless applications

Usage:
  box <command> [options]

Commands:
  init     Initialize a new Box project
  build    Build deployment artifacts from an existing project
  version  Show version information
  help     Show this help message

Examples:
  box init my-app --lang go
  box init my-api --lang typescript
  box build --project my-gcp-project

Run 'box <command> --help' for more information on a command.
`)
}

func initCommand() {
	var projectName string
	var langFlag string
	var pathFlag string
	var githubUserFlag string

	// Parse init-specific flags
	initFlags := flag.NewFlagSet("init", flag.ExitOnError)
	initFlags.StringVar(&langFlag, "lang", "", "Project language (go|typescript)")
	initFlags.StringVar(&pathFlag, "path", "", "Project path (default: ./project-name)")
	initFlags.StringVar(&githubUserFlag, "github-user", "", "GitHub username or organization (for Go projects)")
	initFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: box init [project-name] [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		initFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  box init my-app --lang go --github-user myusername\n")
		fmt.Fprintf(os.Stderr, "  box init my-api --lang typescript --path ./projects/my-api\n\n")
	}

	// Parse arguments
	args := os.Args[2:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		projectName = args[0]
		args = args[1:]
	}
	initFlags.Parse(args)

	// Get project name interactively if not provided
	if projectName == "" {
		prompt := promptui.Prompt{
			Label:   "Project name",
			Default: "my-box-app",
		}
		result, err := prompt.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Prompt failed: %v\n", err)
			os.Exit(1)
		}
		projectName = result
	}

	// Get language interactively if not provided
	var lang Language
	if langFlag != "" {
		lang = Language(strings.ToLower(langFlag))
		if lang != LanguageGo && lang != LanguageTypeScript {
			fmt.Fprintf(os.Stderr, "Error: Invalid language. Choose 'go' or 'typescript'\n")
			os.Exit(1)
		}
	} else {
		prompt := promptui.Select{
			Label: "Select project language",
			Items: []string{"Go", "TypeScript"},
		}
		_, result, err := prompt.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Prompt failed: %v\n", err)
			os.Exit(1)
		}
		if result == "Go" {
			lang = LanguageGo
		} else {
			lang = LanguageTypeScript
		}
	}

	// Get GitHub username for Go projects (needed for module name)
	var githubUsername string
	if lang == LanguageGo {
		if githubUserFlag != "" {
			githubUsername = githubUserFlag
		} else {
			prompt := promptui.Prompt{
				Label:   "GitHub username or organization",
				Default: "your-username",
			}
			result, err := prompt.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Prompt failed: %v\n", err)
				os.Exit(1)
			}
			githubUsername = result
		}
	}

	// Determine project path
	projectPath := pathFlag
	if projectPath == "" {
		projectPath = filepath.Join(".", projectName)
	}

	// Create project
	fmt.Printf("🎯 Creating %s project '%s' at %s\n\n", lang, projectName, projectPath)

	if err := createProject(projectName, lang, projectPath, githubUsername); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create project: %v\n", err)
		os.Exit(1)
	}

	// Print success message
	fmt.Printf("\n✅ Project created successfully!\n\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  cd %s\n", projectPath)
	if lang == LanguageGo {
		fmt.Printf("  go mod tidy\n")
		fmt.Printf("  go run .\n")
	} else {
		fmt.Printf("  npm install\n")
		fmt.Printf("  npm run dev\n")
	}
	fmt.Printf("\nTo build deployment artifacts:\n")
	fmt.Printf("  box build --project <your-gcp-project-id>\n\n")
}

func createProject(name string, lang Language, path string, githubUsername string) error {
	// Create project directory
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Template data
	var moduleName string
	if lang == LanguageGo {
		moduleName = fmt.Sprintf("github.com/%s/%s", githubUsername, name)
	} else {
		moduleName = name
	}

	data := map[string]interface{}{
		"ProjectName": name,
		"ModuleName":  moduleName,
	}

	// Copy templates based on language
	templateDir := fmt.Sprintf("templates/%s", lang)

	return fs.WalkDir(templatesFS, templateDir, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip the template directory itself
		if filePath == templateDir {
			return nil
		}

		// Get relative path from template dir
		relPath, err := filepath.Rel(templateDir, filePath)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(path, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Read template
		content, err := templatesFS.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", filePath, err)
		}

		// Get file info for mode
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		// Process as template if it's a .tmpl file
		if strings.HasSuffix(filePath, ".tmpl") {
			targetPath = strings.TrimSuffix(targetPath, ".tmpl")

			tmpl, err := template.New(filepath.Base(filePath)).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", filePath, err)
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}
			defer file.Close()

			if err := tmpl.Execute(file, data); err != nil {
				return fmt.Errorf("failed to execute template %s: %w", filePath, err)
			}

			fmt.Printf("  ✓ Created %s\n", strings.TrimSuffix(relPath, ".tmpl"))
		} else {
			// Copy file directly
			if err := os.WriteFile(targetPath, content, info.Mode()); err != nil {
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			fmt.Printf("  ✓ Created %s\n", relPath)
		}

		return nil
	})
}

func buildCommand() {
	// Parse build-specific flags
	buildFlags := flag.NewFlagSet("build", flag.ExitOnError)
	handlersDir := buildFlags.String("handlers", "./handlers", "Path to handlers directory")
	outputDir := buildFlags.String("output", "./build", "Path to output directory")
	projectID := buildFlags.String("project", "", "GCP project ID (required)")
	region := buildFlags.String("region", "us-central1", "GCP region")
	environment := buildFlags.String("env", "dev", "Environment (dev, staging, production)")
	moduleName := buildFlags.String("module", "", "Module name (auto-detected if not provided)")
	clean := buildFlags.Bool("clean", false, "Clean build directory before generating")
	verbose := buildFlags.Bool("verbose", false, "Enable verbose logging")

	buildFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: box build [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		buildFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  box build --project my-gcp-project\n")
		fmt.Fprintf(os.Stderr, "  box build --handlers ./handlers --output ./build --project my-gcp-project\n\n")
	}

	buildFlags.Parse(os.Args[2:])

	// Validate required flags
	if *projectID == "" {
		fmt.Fprintf(os.Stderr, "Error: --project flag is required\n\n")
		buildFlags.Usage()
		os.Exit(1)
	}

	// Detect project language
	lang, err := detectLanguage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Detected %s project\n", lang)

	// Create logger
	var logger *zap.Logger
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

	// Delegate to language-specific build
	switch lang {
	case LanguageGo:
		buildGo(*handlersDir, *outputDir, *projectID, *region, *environment, *moduleName, *clean, logger)
	case LanguageTypeScript:
		buildTypeScript(*handlersDir, *outputDir, *projectID, *region, *environment, *moduleName, *clean, logger)
	}
}

func detectLanguage() (Language, error) {
	// Check for go.mod
	if _, err := os.Stat("go.mod"); err == nil {
		return LanguageGo, nil
	}

	// Check for package.json
	if _, err := os.Stat("package.json"); err == nil {
		return LanguageTypeScript, nil
	}

	return "", fmt.Errorf("could not detect project language. Make sure you're in a Go (go.mod) or TypeScript (package.json) project directory")
}

func buildGo(handlersDir, outputDir, projectID, region, environment, moduleName string, clean bool, logger *zap.Logger) {
	logger.Info("Building Go project",
		zap.String("version", version),
		zap.String("handlers", handlersDir),
		zap.String("output", outputDir),
		zap.String("project", projectID),
		zap.String("region", region),
		zap.String("environment", environment))

	// Parse annotations
	logger.Info("Parsing handlers", zap.String("directory", handlersDir))
	parser := annotations.NewParser()
	parsed, err := parser.ParseDirectory(handlersDir)
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
	if moduleName == "" {
		detectedModule, err := detectGoModuleName()
		if err != nil {
			logger.Fatal("Failed to detect module name. Please specify --module flag", zap.Error(err))
		}
		moduleName = detectedModule
		logger.Info("Detected module name", zap.String("module", moduleName))
	}

	// Create generator
	logger.Info("Creating deployment artifacts")
	generator := build.NewGenerator(build.Config{
		Handlers:      parsed.Handlers,
		OutputDir:     outputDir,
		ModuleName:    moduleName,
		ProjectID:     projectID,
		Region:        region,
		Environment:   environment,
		Logger:        logger,
		CleanBuildDir: clean,
	})

	// Generate all artifacts
	if err := generator.Generate(); err != nil {
		logger.Fatal("Failed to generate artifacts", zap.Error(err))
	}

	logger.Info("✓ Deployment artifacts generated successfully",
		zap.String("output", outputDir))

	printBuildSummary(outputDir)
}

func buildTypeScript(handlersDir, outputDir, projectID, region, environment, moduleName string, clean bool, logger *zap.Logger) {
	logger.Info("Building TypeScript project",
		zap.String("version", version),
		zap.String("handlers", handlersDir),
		zap.String("output", outputDir),
		zap.String("project", projectID),
		zap.String("region", region),
		zap.String("environment", environment))

	// Parse annotations using TypeScript parser
	logger.Info("Parsing TypeScript/JavaScript handlers", zap.String("directory", handlersDir))
	parser := typescript.NewParser()
	parsed, err := parser.ParseDirectory(handlersDir)
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
	if moduleName == "" {
		detectedModule, err := detectTypeScriptModuleName()
		if err != nil {
			logger.Fatal("Failed to detect module name. Please specify --module flag", zap.Error(err))
		}
		moduleName = detectedModule
		logger.Info("Detected module name", zap.String("module", moduleName))
	}

	// Create generator
	logger.Info("Creating deployment artifacts")
	generator := typescript.NewGenerator(
		parsed.Handlers,
		outputDir,
		moduleName,
		projectID,
		region,
		environment,
		clean,
		logger,
	)

	// Generate all artifacts
	if err := generator.Generate(); err != nil {
		logger.Fatal("Failed to generate artifacts", zap.Error(err))
	}

	logger.Info("✓ Deployment artifacts generated successfully",
		zap.String("output", outputDir))

	printBuildSummary(outputDir)
}

func detectGoModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("go.mod not found: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

func detectTypeScriptModuleName() (string, error) {
	data, err := os.ReadFile("package.json")
	if err != nil {
		return "", fmt.Errorf("package.json not found: %w", err)
	}

	// Simple JSON parsing for name field
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"name"`) {
			// Extract value between quotes after "name":
			parts := strings.Split(line, `"name"`)
			if len(parts) > 1 {
				valuePart := strings.TrimSpace(parts[1])
				valuePart = strings.TrimPrefix(valuePart, ":")
				valuePart = strings.TrimSpace(valuePart)
				valuePart = strings.Trim(valuePart, `",`)
				return valuePart, nil
			}
		}
	}

	return "box-app", nil // Default name
}

func countFunctions(handlers []annotations.Handler) int {
	count := 0
	for _, h := range handlers {
		if h.DeploymentType == annotations.DeploymentFunction {
			count++
		}
	}
	return count
}

func countContainers(handlers []annotations.Handler) int {
	count := 0
	for _, h := range handlers {
		if h.DeploymentType == annotations.DeploymentContainer {
			count++
		}
	}
	return count
}

func printBuildSummary(outputDir string) {
	fmt.Printf("\n✅ Success! Generated deployment artifacts:\n")
	fmt.Printf("  • Cloud Functions: %s/functions/\n", outputDir)
	fmt.Printf("  • Cloud Run Containers: %s/containers/\n", outputDir)
	fmt.Printf("  • API Gateway: %s/gateway/\n", outputDir)
	fmt.Printf("  • Terraform IaC: %s/terraform/\n", outputDir)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review generated files in %s/\n", outputDir)
	fmt.Printf("  2. Deploy with: cd %s/terraform && terraform init && terraform apply\n", outputDir)
}
