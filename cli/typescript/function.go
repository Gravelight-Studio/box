package typescript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gravelight-studio/box/go/annotations"
	"go.uber.org/zap"
)

// FunctionGenerator generates Cloud Functions for TypeScript handlers
type FunctionGenerator struct {
	handlers   []annotations.Handler
	outputDir  string
	moduleName string
	logger     *zap.Logger
}

// NewFunctionGenerator creates a new function generator
func NewFunctionGenerator(handlers []annotations.Handler, outputDir, moduleName string, logger *zap.Logger) *FunctionGenerator {
	return &FunctionGenerator{
		handlers:   handlers,
		outputDir:  outputDir,
		moduleName: moduleName,
		logger:     logger,
	}
}

// Generate generates all function packages
func (g *FunctionGenerator) Generate() error {
	// Filter function handlers
	var functionHandlers []annotations.Handler
	for _, h := range g.handlers {
		if h.DeploymentType == annotations.DeploymentFunction {
			functionHandlers = append(functionHandlers, h)
		}
	}

	if len(functionHandlers) == 0 {
		g.logger.Info("No function handlers to generate")
		return nil
	}

	// Create output directory
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate each function
	for _, handler := range functionHandlers {
		if err := g.generateFunction(handler); err != nil {
			return fmt.Errorf("failed to generate function %s: %w", handler.FunctionName, err)
		}
	}

	g.logger.Info("Generated cloud functions",
		zap.Int("count", len(functionHandlers)),
		zap.String("outputDir", g.outputDir))

	return nil
}

// generateFunction generates a single function package
func (g *FunctionGenerator) generateFunction(handler annotations.Handler) error {
	functionName := g.getFunctionName(handler)
	functionDir := filepath.Join(g.outputDir, functionName)

	g.logger.Info("Generating cloud function",
		zap.String("name", functionName),
		zap.String("path", handler.Route.Path))

	// Create function directory
	if err := os.MkdirAll(functionDir, 0755); err != nil {
		return err
	}

	// Generate files
	if err := g.generatePackageJson(functionDir, handler); err != nil {
		return err
	}
	if err := g.generateIndexJs(functionDir, handler); err != nil {
		return err
	}
	if err := g.generateFunctionYaml(functionDir, handler); err != nil {
		return err
	}
	if err := g.generateGcloudIgnore(functionDir); err != nil {
		return err
	}

	return nil
}

// generatePackageJson generates package.json
func (g *FunctionGenerator) generatePackageJson(dir string, handler annotations.Handler) error {
	pkg := map[string]interface{}{
		"name":        g.getFunctionName(handler),
		"version":     "1.0.0",
		"description": fmt.Sprintf("Cloud Function for %s %s", handler.Route.Method, handler.Route.Path),
		"main":        "index.js",
		"scripts": map[string]string{
			"start": "node index.js",
		},
		"dependencies": map[string]string{
			"@google-cloud/functions-framework": "^3.3.0",
			"express":                           "^4.18.2",
			"cors":                              "^2.8.5",
			"express-rate-limit":                "^7.1.5",
		},
		"engines": map[string]string{
			"node": ">=18.0.0",
		},
	}

	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "package.json"), data, 0644)
}

// generateIndexJs generates index.js entry point
func (g *FunctionGenerator) generateIndexJs(dir string, handler annotations.Handler) error {
	var sb strings.Builder

	sb.WriteString("const functions = require('@google-cloud/functions-framework');\n")
	sb.WriteString("const cors = require('cors');\n")
	sb.WriteString("const rateLimit = require('express-rate-limit');\n\n")

	// CORS configuration
	if handler.CORS != nil {
		sb.WriteString("// CORS configuration\n")
		origins := "['*']"
		if len(handler.CORS.AllowedOrigins) > 0 && handler.CORS.AllowedOrigins[0] != "*" {
			originsJSON, _ := json.Marshal(handler.CORS.AllowedOrigins)
			origins = string(originsJSON)
		}
		sb.WriteString(fmt.Sprintf(`const corsMiddleware = cors({
  origin: %s,
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization'],
  credentials: false
});

`, origins))
	}

	// Rate limit configuration
	if handler.RateLimit != nil {
		sb.WriteString("// Rate limit configuration\n")
		windowMs := int(handler.RateLimit.Period.Milliseconds())
		sb.WriteString(fmt.Sprintf(`const rateLimiter = rateLimit({
  windowMs: %d,
  max: %d,
  message: { error: 'Too many requests, please try again later. Limit: %s' },
  standardHeaders: true,
  legacyHeaders: false
});

`, windowMs, handler.RateLimit.Count, handler.RateLimit.Raw))
	}

	// Main handler function
	sb.WriteString("// Main handler function\n")
	sb.WriteString(fmt.Sprintf("functions.http('%s', async (req, res) => {\n", handler.FunctionName))

	// Apply CORS
	if handler.CORS != nil {
		sb.WriteString("  corsMiddleware(req, res, () => {});\n\n")
	}

	// Authentication
	if handler.Auth.Type == annotations.AuthRequired {
		sb.WriteString(`  // Authentication
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Unauthorized: Missing or invalid authorization header' });
  }
  const token = authHeader.split(' ')[1];
  if (!token) {
    return res.status(401).json({ error: 'Unauthorized: Invalid token' });
  }
  req.user = { token };

`)
	}

	// Rate limiting
	if handler.RateLimit != nil {
		sb.WriteString("  // Rate limiting\n")
		sb.WriteString("  rateLimiter(req, res, () => {});\n\n")
	}

	// Handle request
	sb.WriteString(`  // Handle request
  try {
    // TODO: Import and call your actual handler logic here
    // For now, return a placeholder response
    res.status(200).json({
      message: '` + handler.FunctionName + ` executed successfully',
      method: '` + handler.Route.Method + `',
      path: '` + handler.Route.Path + `'
    });
  } catch (error) {
    console.error('Error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});
`)

	return os.WriteFile(filepath.Join(dir, "index.js"), []byte(sb.String()), 0644)
}

// generateFunctionYaml generates function.yaml configuration
func (g *FunctionGenerator) generateFunctionYaml(dir string, handler annotations.Handler) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Cloud Function configuration for %s\n", handler.FunctionName))
	sb.WriteString("runtime: nodejs20\n")
	sb.WriteString(fmt.Sprintf("entryPoint: %s\n\n", handler.FunctionName))

	sb.WriteString("# Resource configuration\n")
	if handler.Memory != "" {
		// Memory is a string like "256MB", extract the number
		memory := strings.TrimSuffix(handler.Memory, "MB")
		sb.WriteString(fmt.Sprintf("availableMemoryMb: %s\n", memory))
	} else {
		sb.WriteString("availableMemoryMb: 256\n")
	}

	if handler.Timeout > 0 {
		timeoutSeconds := int(handler.Timeout.Seconds())
		sb.WriteString(fmt.Sprintf("timeout: %ds\n", timeoutSeconds))
	} else {
		sb.WriteString("timeout: 60s\n")
	}

	sb.WriteString("maxInstances: 100\n\n")

	sb.WriteString("# Environment variables\n")
	sb.WriteString("environmentVariables:\n")
	sb.WriteString("  NODE_ENV: production\n")
	sb.WriteString(fmt.Sprintf("  FUNCTION_NAME: %s\n", handler.FunctionName))
	sb.WriteString(fmt.Sprintf("  FUNCTION_PATH: %s\n", handler.Route.Path))
	sb.WriteString(fmt.Sprintf("  FUNCTION_METHOD: %s\n", handler.Route.Method))

	return os.WriteFile(filepath.Join(dir, "function.yaml"), []byte(sb.String()), 0644)
}

// generateGcloudIgnore generates .gcloudignore
func (g *FunctionGenerator) generateGcloudIgnore(dir string) error {
	ignore := `.gcloudignore
.git
.gitignore
node_modules/
*.log
.DS_Store
`
	return os.WriteFile(filepath.Join(dir, ".gcloudignore"), []byte(ignore), 0644)
}

// getFunctionName generates a function name from handler
func (g *FunctionGenerator) getFunctionName(handler annotations.Handler) string {
	// Convert CamelCase to kebab-case
	name := handler.FunctionName
	var result strings.Builder

	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
