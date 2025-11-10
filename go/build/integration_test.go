package build

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// Integration tests for the build system - tests at the boundary with file system

func TestIntegration_GeneratorCreation(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Memory:  "256MB",
			Timeout: 30 * time.Second,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      handlers,
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	assert.NotNil(t, gen)
	assert.Equal(t, tmpDir, gen.outputDir)
	assert.Equal(t, "github.com/gravelight-studio/box", gen.moduleName)
	assert.Len(t, gen.GetFunctionHandlers(), 1)
	assert.Len(t, gen.GetContainerHandlers(), 0)
}

func TestIntegration_FilterHandlers(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			DeploymentType: annotations.DeploymentFunction,
		},
		{
			FunctionName:   "GetUsers",
			DeploymentType: annotations.DeploymentContainer,
		},
		{
			FunctionName:   "DeleteAccount",
			DeploymentType: annotations.DeploymentFunction,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		Logger:     zap.NewNop(),
	})

	funcHandlers := gen.GetFunctionHandlers()
	containerHandlers := gen.GetContainerHandlers()

	assert.Len(t, funcHandlers, 2)
	assert.Len(t, containerHandlers, 1)

	assert.Equal(t, "CreateAccount", funcHandlers[0].FunctionName)
	assert.Equal(t, "DeleteAccount", funcHandlers[1].FunctionName)
	assert.Equal(t, "GetUsers", containerHandlers[0].FunctionName)
}

func TestIntegration_GenerateFunctionPackage(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "CreateAccount",
		PackageName:    "accounts",
		PackagePath:    "internal/handlers/accounts",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "POST",
			Path:   "/api/v1/accounts",
		},
		Memory:  "512MB",
		Timeout: 60 * time.Second,
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      []annotations.Handler{handler},
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.GenerateFunctions()
	require.NoError(t, err)

	// Verify function directory was created
	funcDir := filepath.Join(tmpDir, "functions", "create-account")
	assert.DirExists(t, funcDir)

	// Verify required files exist
	mainGo := filepath.Join(funcDir, "main.go")
	goMod := filepath.Join(funcDir, "go.mod")
	functionYAML := filepath.Join(funcDir, "function.yaml")
	deployScript := filepath.Join(funcDir, "deploy.sh")

	assert.FileExists(t, mainGo, "main.go should exist")
	assert.FileExists(t, goMod, "go.mod should exist")
	assert.FileExists(t, functionYAML, "function.yaml should exist")
	assert.FileExists(t, deployScript, "deploy.sh should exist")

	// Verify main.go contains expected content
	mainContent, err := os.ReadFile(mainGo)
	require.NoError(t, err)
	mainStr := string(mainContent)

	assert.Contains(t, mainStr, "package main")
	assert.Contains(t, mainStr, "func CreateAccount(w http.ResponseWriter, r *http.Request)")
	assert.Contains(t, mainStr, "accounts.CreateAccount(w, r)")
	assert.Contains(t, mainStr, "github.com/gravelight-studio/box/internal/handlers/accounts")

	// Verify go.mod contains expected content
	goModContent, err := os.ReadFile(goMod)
	require.NoError(t, err)
	goModStr := string(goModContent)

	assert.Contains(t, goModStr, "module github.com/gravelight-studio/box/build/functions/create-account")
	assert.Contains(t, goModStr, "github.com/GoogleCloudPlatform/functions-framework-go")
	assert.Contains(t, goModStr, "github.com/jackc/pgx/v5")
	assert.Contains(t, goModStr, "replace github.com/gravelight-studio/box => ../../..")

	// Verify function.yaml contains expected config
	yamlContent, err := os.ReadFile(functionYAML)
	require.NoError(t, err)
	yamlStr := string(yamlContent)

	assert.Contains(t, yamlStr, "name: CreateAccount")
	assert.Contains(t, yamlStr, "runtime: go122")
	assert.Contains(t, yamlStr, "entryPoint: CreateAccount")
	assert.Contains(t, yamlStr, "availableMemoryMb: 512Mi")
	assert.Contains(t, yamlStr, "timeout: 60s")

	// Verify deploy script exists and has proper permissions (on Unix)
	info, err := os.Stat(deployScript)
	require.NoError(t, err)
	// Skip executable check on Windows
	if os.PathSeparator == '/' {
		assert.True(t, info.Mode()&0111 != 0, "deploy.sh should be executable on Unix")
	}

	// Verify deploy script content
	deployContent, err := os.ReadFile(deployScript)
	require.NoError(t, err)
	deployStr := string(deployContent)

	assert.Contains(t, deployStr, "#!/bin/bash")
	assert.Contains(t, deployStr, "FUNCTION_NAME=\"create-account\"")
	assert.Contains(t, deployStr, "ENTRY_POINT=\"CreateAccount\"")
	assert.Contains(t, deployStr, "gcloud functions deploy")
}

func TestIntegration_GenerateMultipleFunctions(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Memory:  "256MB",
			Timeout: 30 * time.Second,
		},
		{
			FunctionName:   "GetAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/accounts/{id}",
			},
			Memory:  "128MB",
			Timeout: 15 * time.Second,
		},
		{
			FunctionName:   "DeleteAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "DELETE",
				Path:   "/api/v1/accounts/{id}",
			},
			Memory:  "256MB",
			Timeout: 30 * time.Second,
		},
		{
			FunctionName:   "ListUsers",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer, // Should be filtered out
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/users",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      handlers,
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.Generate()
	require.NoError(t, err)

	// Verify only function handlers were generated
	functionsDir := filepath.Join(tmpDir, "functions")
	assert.DirExists(t, functionsDir)

	// Should have 3 function directories
	entries, err := os.ReadDir(functionsDir)
	require.NoError(t, err)
	assert.Len(t, entries, 3)

	// Verify each function directory
	expectedDirs := []string{"create-account", "get-account", "delete-account"}
	for _, dirName := range expectedDirs {
		dirPath := filepath.Join(functionsDir, dirName)
		assert.DirExists(t, dirPath, "Function directory %s should exist", dirName)

		// Each should have all required files
		assert.FileExists(t, filepath.Join(dirPath, "main.go"))
		assert.FileExists(t, filepath.Join(dirPath, "go.mod"))
		assert.FileExists(t, filepath.Join(dirPath, "function.yaml"))
		assert.FileExists(t, filepath.Join(dirPath, "deploy.sh"))
	}
}

func TestIntegration_CleanBuildDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some existing files in the build directory
	existingFile := filepath.Join(tmpDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("old content"), 0644)
	require.NoError(t, err)

	handler := annotations.Handler{
		FunctionName:   "TestFunction",
		PackageName:    "test",
		PackagePath:    "internal/handlers/test",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "GET",
			Path:   "/test",
		},
	}

	gen := NewGenerator(Config{
		Handlers:      []annotations.Handler{handler},
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err = gen.Generate()
	require.NoError(t, err)

	// Existing file should be removed
	assert.NoFileExists(t, existingFile, "Existing file should be cleaned")

	// New function should exist
	funcDir := filepath.Join(tmpDir, "functions", "test-function")
	assert.DirExists(t, funcDir)
}

func TestIntegration_NoFunctionsToGenerate(t *testing.T) {
	// All handlers are containers
	handlers := []annotations.Handler{
		{
			FunctionName:   "ContainerHandler",
			DeploymentType: annotations.DeploymentContainer,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		Logger:     zap.NewNop(),
	})

	err := gen.Generate()
	require.NoError(t, err)

	// Build directory should be created but empty
	assert.DirExists(t, tmpDir)

	// No functions directory should exist since no functions were generated
	functionsDir := filepath.Join(tmpDir, "functions")
	if _, err := os.Stat(functionsDir); err == nil {
		// If it exists, it should be empty
		entries, err := os.ReadDir(functionsDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	}
}

func TestIntegration_ToKebabCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"CreateAccount", "create-account"},
		{"GetAccountByID", "get-account-by-i-d"},
		{"Test", "test"},
		{"SimpleFunction", "simple-function"},
		{"HTTPHandler", "h-t-t-p-handler"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toKebabCase(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIntegration_DefaultValues(t *testing.T) {
	// Handler with no memory or timeout set
	handler := annotations.Handler{
		FunctionName:   "DefaultsFunction",
		PackageName:    "test",
		PackagePath:    "internal/handlers/test",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "GET",
			Path:   "/test",
		},
		// Memory and Timeout not set
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   []annotations.Handler{handler},
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateFunctions()
	require.NoError(t, err)

	// Check function.yaml for default values
	yamlPath := filepath.Join(tmpDir, "functions", "defaults-function", "function.yaml")
	yamlContent, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	yamlStr := string(yamlContent)

	// Should have default memory
	assert.Contains(t, yamlStr, "availableMemoryMb: 256Mi")

	// Should have default timeout
	assert.Contains(t, yamlStr, "timeout: 60s")
}

// Container Generator Tests

func TestIntegration_GenerateContainerPackage(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "GetUsers",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/users",
			},
		},
		{
			FunctionName:   "CreateUser",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/users",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      handlers,
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.GenerateContainers()
	require.NoError(t, err)

	// Verify container directory was created
	containerDir := filepath.Join(tmpDir, "containers", "users")
	assert.DirExists(t, containerDir)

	// Verify required files exist
	mainGo := filepath.Join(containerDir, "main.go")
	dockerfile := filepath.Join(containerDir, "Dockerfile")
	cloudBuild := filepath.Join(containerDir, "cloudbuild.yaml")
	deployScript := filepath.Join(containerDir, "deploy.sh")

	assert.FileExists(t, mainGo, "main.go should exist")
	assert.FileExists(t, dockerfile, "Dockerfile should exist")
	assert.FileExists(t, cloudBuild, "cloudbuild.yaml should exist")
	assert.FileExists(t, deployScript, "deploy.sh should exist")

	// Verify main.go contains both handlers
	mainContent, err := os.ReadFile(mainGo)
	require.NoError(t, err)
	mainStr := string(mainContent)

	assert.Contains(t, mainStr, "package main")
	assert.Contains(t, mainStr, `r.Method("GET", "/api/v1/users"`)
	assert.Contains(t, mainStr, `r.Method("POST", "/api/v1/users"`)
	assert.Contains(t, mainStr, "users.GetUsers")
	assert.Contains(t, mainStr, "users.CreateUser")
	assert.Contains(t, mainStr, "github.com/gravelight-studio/box/internal/handlers/users")

	// Verify Dockerfile is multi-stage
	dockerContent, err := os.ReadFile(dockerfile)
	require.NoError(t, err)
	dockerStr := string(dockerContent)

	assert.Contains(t, dockerStr, "FROM golang:1.22-alpine AS builder")
	assert.Contains(t, dockerStr, "FROM alpine:latest")
	assert.Contains(t, dockerStr, "CGO_ENABLED=0")
	assert.Contains(t, dockerStr, "HEALTHCHECK")

	// Verify cloudbuild.yaml
	cloudBuildContent, err := os.ReadFile(cloudBuild)
	require.NoError(t, err)
	cloudBuildStr := string(cloudBuildContent)

	assert.Contains(t, cloudBuildStr, "gcr.io/cloud-builders/docker")
	assert.Contains(t, cloudBuildStr, "gcloud")
	assert.Contains(t, cloudBuildStr, "'deploy'")
	assert.Contains(t, cloudBuildStr, "users")

	// Verify deploy script exists
	deployContent, err := os.ReadFile(deployScript)
	require.NoError(t, err)
	deployStr := string(deployContent)

	assert.Contains(t, deployStr, "#!/bin/bash")
	assert.Contains(t, deployStr, "SERVICE_NAME=\"users\"")
	assert.Contains(t, deployStr, "gcloud builds submit")
}

func TestIntegration_GenerateMultipleContainerServices(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "GetUsers",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/users",
			},
		},
		{
			FunctionName:   "CreateUser",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/users",
			},
		},
		{
			FunctionName:   "GetAccounts",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/accounts",
			},
		},
		{
			FunctionName:   "CreateMessage",
			PackageName:    "messages",
			PackagePath:    "internal/handlers/messages",
			DeploymentType: annotations.DeploymentFunction, // Should be filtered out
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/messages",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      handlers,
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.Generate()
	require.NoError(t, err)

	// Verify container services were generated (2 services: users, accounts)
	containersDir := filepath.Join(tmpDir, "containers")
	assert.DirExists(t, containersDir)

	entries, err := os.ReadDir(containersDir)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "Should have 2 container services")

	// Verify each service directory
	expectedServices := []string{"users", "accounts"}
	for _, serviceName := range expectedServices {
		serviceDir := filepath.Join(containersDir, serviceName)
		assert.DirExists(t, serviceDir, "Service directory %s should exist", serviceName)

		// Each should have all required files
		assert.FileExists(t, filepath.Join(serviceDir, "main.go"))
		assert.FileExists(t, filepath.Join(serviceDir, "Dockerfile"))
		assert.FileExists(t, filepath.Join(serviceDir, "cloudbuild.yaml"))
		assert.FileExists(t, filepath.Join(serviceDir, "deploy.sh"))
	}

	// Verify function was also generated separately
	functionsDir := filepath.Join(tmpDir, "functions")
	assert.DirExists(t, functionsDir)

	functionEntries, err := os.ReadDir(functionsDir)
	require.NoError(t, err)
	assert.Len(t, functionEntries, 1, "Should have 1 function")
}

func TestIntegration_ContainerServiceGrouping(t *testing.T) {
	// Test that handlers from the same package are grouped together
	handlers := []annotations.Handler{
		{
			FunctionName:   "Handler1",
			PackageName:    "service1",
			PackagePath:    "internal/handlers/service1",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/service1/a",
			},
		},
		{
			FunctionName:   "Handler2",
			PackageName:    "service1",
			PackagePath:    "internal/handlers/service1",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/service1/b",
			},
		},
		{
			FunctionName:   "Handler3",
			PackageName:    "service1",
			PackagePath:    "internal/handlers/service1",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "DELETE",
				Path:   "/api/v1/service1/c",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateContainers()
	require.NoError(t, err)

	// Should have only 1 service directory
	containersDir := filepath.Join(tmpDir, "containers")
	entries, err := os.ReadDir(containersDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "All handlers from same package should be in one service")

	// Verify all 3 handlers are in the main.go
	mainGo := filepath.Join(containersDir, "service1", "main.go")
	mainContent, err := os.ReadFile(mainGo)
	require.NoError(t, err)
	mainStr := string(mainContent)

	assert.Contains(t, mainStr, "Handler1")
	assert.Contains(t, mainStr, "Handler2")
	assert.Contains(t, mainStr, "Handler3")
	assert.Contains(t, mainStr, "/api/v1/service1/a")
	assert.Contains(t, mainStr, "/api/v1/service1/b")
	assert.Contains(t, mainStr, "/api/v1/service1/c")
}

func TestIntegration_NoContainersToGenerate(t *testing.T) {
	// All handlers are functions
	handlers := []annotations.Handler{
		{
			FunctionName:   "FunctionHandler",
			DeploymentType: annotations.DeploymentFunction,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		Logger:     zap.NewNop(),
	})

	err := gen.Generate()
	require.NoError(t, err)

	// Containers directory might not exist or be empty
	containersDir := filepath.Join(tmpDir, "containers")
	if _, err := os.Stat(containersDir); err == nil {
		// If it exists, it should be empty
		entries, err := os.ReadDir(containersDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	}
}

func TestIntegration_MixedDeploymentTypes(t *testing.T) {
	// Mix of functions and containers
	handlers := []annotations.Handler{
		{
			FunctionName:   "LightweightFunction",
			PackageName:    "lightweight",
			PackagePath:    "internal/handlers/lightweight",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/lightweight",
			},
		},
		{
			FunctionName:   "HeavyContainer",
			PackageName:    "heavy",
			PackagePath:    "internal/handlers/heavy",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/heavy",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		Logger:     zap.NewNop(),
	})

	err := gen.Generate()
	require.NoError(t, err)

	// Should have both functions and containers directories
	functionsDir := filepath.Join(tmpDir, "functions")
	containersDir := filepath.Join(tmpDir, "containers")

	assert.DirExists(t, functionsDir)
	assert.DirExists(t, containersDir)

	// Check function was generated
	functionEntries, err := os.ReadDir(functionsDir)
	require.NoError(t, err)
	assert.Len(t, functionEntries, 1)

	// Check container was generated
	containerEntries, err := os.ReadDir(containersDir)
	require.NoError(t, err)
	assert.Len(t, containerEntries, 1)
}

// Gateway Generator Integration Tests

func TestIntegration_GenerateBasicGateway(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "CreateAccount",
		PackageName:    "accounts",
		PackagePath:    "internal/handlers/accounts",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "POST",
			Path:   "/api/v1/accounts",
		},
		Auth: annotations.AuthConfig{
			Type: annotations.AuthNone,
		},
		Timeout: 30 * time.Second,
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      []annotations.Handler{handler},
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		ProjectID:     "test-project",
		Region:        "us-central1",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Verify gateway directory was created
	gatewayDir := filepath.Join(tmpDir, "gateway")
	assert.DirExists(t, gatewayDir)

	// Verify required files exist
	openAPIFile := filepath.Join(gatewayDir, "openapi.yaml")
	gatewayConfigFile := filepath.Join(gatewayDir, "gateway-config.yaml")
	deployScript := filepath.Join(gatewayDir, "deploy.sh")

	assert.FileExists(t, openAPIFile, "openapi.yaml should exist")
	assert.FileExists(t, gatewayConfigFile, "gateway-config.yaml should exist")
	assert.FileExists(t, deployScript, "deploy.sh should exist")

	// Verify OpenAPI spec contains expected content
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	assert.Contains(t, openAPIStr, "openapi: 3.0.0")
	assert.Contains(t, openAPIStr, "title: Wylla API")
	assert.Contains(t, openAPIStr, "/api/v1/accounts:")
	assert.Contains(t, openAPIStr, "post:")
	assert.Contains(t, openAPIStr, "operationId: CreateAccount")
	assert.Contains(t, openAPIStr, "tags:")
	assert.Contains(t, openAPIStr, "- accounts")
	assert.Contains(t, openAPIStr, "x-google-backend:")
	assert.Contains(t, openAPIStr, "address: https://us-central1-test-project.cloudfunctions.net/create-account")

	// Verify gateway config contains expected content
	gatewayConfigContent, err := os.ReadFile(gatewayConfigFile)
	require.NoError(t, err)
	gatewayConfigStr := string(gatewayConfigContent)

	assert.Contains(t, gatewayConfigStr, "apiVersion: apigateway.cnrm.cloud.google.com/v1beta1")
	assert.Contains(t, gatewayConfigStr, "kind: ApiGatewayAPI")
	assert.Contains(t, gatewayConfigStr, "name: wylla-api")
	assert.Contains(t, gatewayConfigStr, "projectRef:")
	assert.Contains(t, gatewayConfigStr, "external: test-project")
}

func TestIntegration_GenerateGatewayWithAuth(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthRequired,
			},
		},
		{
			FunctionName:   "ListAccounts",
			PackageName:    "accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/accounts",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthOptional,
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		ProjectID:  "test-project",
		Region:     "us-central1",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Verify OpenAPI spec includes security schemes
	openAPIFile := filepath.Join(tmpDir, "gateway", "openapi.yaml")
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	// Should have security schemes defined
	assert.Contains(t, openAPIStr, "securitySchemes:")
	assert.Contains(t, openAPIStr, "bearerAuth:")
	assert.Contains(t, openAPIStr, "type: http")
	assert.Contains(t, openAPIStr, "scheme: bearer")
	assert.Contains(t, openAPIStr, "bearerFormat: JWT")

	// Both operations should reference bearer auth
	assert.Contains(t, openAPIStr, "security:")
	assert.Contains(t, openAPIStr, "- bearerAuth: []")

	// Should have 401/403 responses for auth
	assert.Contains(t, openAPIStr, "'401':")
	assert.Contains(t, openAPIStr, "'403':")
}

func TestIntegration_GenerateGatewayWithPathParameters(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "GetAccount",
		PackageName:    "accounts",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "GET",
			Path:   "/api/v1/accounts/{id}",
		},
		Auth: annotations.AuthConfig{
			Type: annotations.AuthRequired,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   []annotations.Handler{handler},
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		ProjectID:  "test-project",
		Region:     "us-central1",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Verify path parameter is extracted and documented
	openAPIFile := filepath.Join(tmpDir, "gateway", "openapi.yaml")
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	assert.Contains(t, openAPIStr, "/api/v1/accounts/{id}:")
	assert.Contains(t, openAPIStr, "parameters:")
	assert.Contains(t, openAPIStr, "- name: id")
	assert.Contains(t, openAPIStr, "in: path")
	assert.Contains(t, openAPIStr, "required: true")
	assert.Contains(t, openAPIStr, "type: string")
}

func TestIntegration_GenerateGatewayWithRateLimit(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "CreateAccount",
		PackageName:    "accounts",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "POST",
			Path:   "/api/v1/accounts",
		},
		RateLimit: &annotations.RateLimitConfig{
			Count:  100,
			Period: 1 * time.Hour,
			Raw:    "100/hour",
		},
		Auth: annotations.AuthConfig{
			Type: annotations.AuthNone,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   []annotations.Handler{handler},
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		ProjectID:  "test-project",
		Region:     "us-central1",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Verify rate limit is included in OpenAPI spec
	openAPIFile := filepath.Join(tmpDir, "gateway", "openapi.yaml")
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	assert.Contains(t, openAPIStr, "x-google-quota:")
	assert.Contains(t, openAPIStr, "metricCosts:")
	assert.Contains(t, openAPIStr, ": 100") // The quota value

	// Should have 429 response
	assert.Contains(t, openAPIStr, "'429':")
	assert.Contains(t, openAPIStr, "rate limit")
}

func TestIntegration_GenerateGatewayMixedBackends(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthNone,
			},
		},
		{
			FunctionName:   "ListUsers",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/users",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthNone,
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		ProjectID:  "test-project",
		Region:     "us-central1",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Verify both function and container backends are correctly mapped
	openAPIFile := filepath.Join(tmpDir, "gateway", "openapi.yaml")
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	// Function backend URL
	assert.Contains(t, openAPIStr, "address: https://us-central1-test-project.cloudfunctions.net/create-account")

	// Container backend URL
	assert.Contains(t, openAPIStr, "address: https://users-us-central1.run.app")
}

func TestIntegration_GenerateGatewayMultipleMethodsSamePath(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthRequired,
			},
		},
		{
			FunctionName:   "ListAccounts",
			PackageName:    "accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/accounts",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthOptional,
			},
		},
		{
			FunctionName:   "DeleteAllAccounts",
			PackageName:    "accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "DELETE",
				Path:   "/api/v1/accounts",
			},
			Auth: annotations.AuthConfig{
				Type: annotations.AuthRequired,
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   handlers,
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		ProjectID:  "test-project",
		Region:     "us-central1",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Verify all methods are grouped under same path
	openAPIFile := filepath.Join(tmpDir, "gateway", "openapi.yaml")
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	// Path should appear once
	assert.Contains(t, openAPIStr, "/api/v1/accounts:")

	// All three methods should be present
	assert.Contains(t, openAPIStr, "post:")
	assert.Contains(t, openAPIStr, "get:")
	assert.Contains(t, openAPIStr, "delete:")

	// Each method should have its operation ID
	assert.Contains(t, openAPIStr, "operationId: CreateAccount")
	assert.Contains(t, openAPIStr, "operationId: ListAccounts")
	assert.Contains(t, openAPIStr, "operationId: DeleteAllAccounts")
}

func TestIntegration_GenerateGatewayNoHandlers(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:   []annotations.Handler{},
		OutputDir:  tmpDir,
		ModuleName: "github.com/gravelight-studio/box",
		ProjectID:  "test-project",
		Region:     "us-central1",
		Logger:     zap.NewNop(),
	})

	err := gen.GenerateGateway()
	require.NoError(t, err)

	// Gateway directory should not be created or be empty
	gatewayDir := filepath.Join(tmpDir, "gateway")
	if _, err := os.Stat(gatewayDir); err == nil {
		// If it exists, it should be empty
		entries, err := os.ReadDir(gatewayDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	}
}

func TestIntegration_GenerateCompleteWithGateway(t *testing.T) {
	// Test that Generate() creates functions, containers, and gateway
	handlers := []annotations.Handler{
		{
			FunctionName:   "FunctionHandler",
			PackageName:    "func",
			PackagePath:    "internal/handlers/func",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/func",
			},
			Memory:  "256MB",
			Timeout: 30 * time.Second,
		},
		{
			FunctionName:   "ContainerHandler",
			PackageName:    "cont",
			PackagePath:    "internal/handlers/cont",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/cont",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      handlers,
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		ProjectID:     "test-project",
		Region:        "us-central1",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.Generate()
	require.NoError(t, err)

	// Should have all three directories
	functionsDir := filepath.Join(tmpDir, "functions")
	containersDir := filepath.Join(tmpDir, "containers")
	gatewayDir := filepath.Join(tmpDir, "gateway")

	assert.DirExists(t, functionsDir)
	assert.DirExists(t, containersDir)
	assert.DirExists(t, gatewayDir)

	// Verify gateway includes both handlers
	openAPIFile := filepath.Join(gatewayDir, "openapi.yaml")
	openAPIContent, err := os.ReadFile(openAPIFile)
	require.NoError(t, err)
	openAPIStr := string(openAPIContent)

	assert.Contains(t, openAPIStr, "/api/v1/func:")
	assert.Contains(t, openAPIStr, "/api/v1/cont:")
	assert.Contains(t, openAPIStr, "operationId: FunctionHandler")
	assert.Contains(t, openAPIStr, "operationId: ContainerHandler")
}

// Terraform Generator Integration Tests

func TestIntegration_GenerateTerraformBasic(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "CreateAccount",
		PackageName:    "accounts",
		PackagePath:    "internal/handlers/accounts",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "POST",
			Path:   "/api/v1/accounts",
		},
		Memory:  "256MB",
		Timeout: 30 * time.Second,
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      []annotations.Handler{handler},
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		ProjectID:     "test-project",
		Region:        "us-central1",
		Environment:   "dev",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Verify terraform directory was created
	terraformDir := filepath.Join(tmpDir, "terraform")
	assert.DirExists(t, terraformDir)

	// Verify required files exist
	mainTF := filepath.Join(terraformDir, "main.tf")
	variablesTF := filepath.Join(terraformDir, "variables.tf")
	outputsTF := filepath.Join(terraformDir, "outputs.tf")
	gitignore := filepath.Join(terraformDir, ".gitignore")
	readme := filepath.Join(terraformDir, "README.md")

	assert.FileExists(t, mainTF, "main.tf should exist")
	assert.FileExists(t, variablesTF, "variables.tf should exist")
	assert.FileExists(t, outputsTF, "outputs.tf should exist")
	assert.FileExists(t, gitignore, ".gitignore should exist")
	assert.FileExists(t, readme, "README.md should exist")

	// Verify main.tf contains expected content
	mainContent, err := os.ReadFile(mainTF)
	require.NoError(t, err)
	mainStr := string(mainContent)

	assert.Contains(t, mainStr, "terraform {")
	assert.Contains(t, mainStr, "required_version = \">= 1.0\"")
	assert.Contains(t, mainStr, "required_providers {")
	assert.Contains(t, mainStr, "google = {")
	assert.Contains(t, mainStr, "module \"cloud_functions\" {")
	assert.Contains(t, mainStr, "source = \"./modules/cloud-functions\"")

	// Verify .gitignore contains Terraform files
	gitignoreContent, err := os.ReadFile(gitignore)
	require.NoError(t, err)
	gitignoreStr := string(gitignoreContent)

	assert.Contains(t, gitignoreStr, "*.tfstate")
	assert.Contains(t, gitignoreStr, ".terraform/")
}

func TestIntegration_GenerateTerraformCloudFunctions(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Memory:  "256MB",
			Timeout: 30 * time.Second,
		},
		{
			FunctionName:   "GetAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/accounts/{id}",
			},
			Memory:  "128MB",
			Timeout: 15 * time.Second,
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:    handlers,
		OutputDir:   tmpDir,
		ModuleName:  "github.com/gravelight-studio/box",
		ProjectID:   "test-project",
		Region:      "us-central1",
		Environment: "dev",
		Logger:      zap.NewNop(),
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Verify cloud-functions module was created
	modulePath := filepath.Join(tmpDir, "terraform", "modules", "cloud-functions", "main.tf")
	assert.FileExists(t, modulePath)

	moduleContent, err := os.ReadFile(modulePath)
	require.NoError(t, err)
	moduleStr := string(moduleContent)

	// Should have service account
	assert.Contains(t, moduleStr, "resource \"google_service_account\" \"accounts\" {")
	assert.Contains(t, moduleStr, "account_id   = \"wylla-accounts-$${var.environment}\"")

	// Should have IAM bindings
	assert.Contains(t, moduleStr, "resource \"google_project_iam_member\" \"accounts_cloudsql\" {")
	assert.Contains(t, moduleStr, "role    = \"roles/cloudsql.client\"")

	// Should have storage bucket for function source
	assert.Contains(t, moduleStr, "resource \"google_storage_bucket\" \"functions\" {")

	// Should have function resources
	assert.Contains(t, moduleStr, "resource \"google_cloudfunctions_function\" \"create_account\" {")
	assert.Contains(t, moduleStr, "name                  = \"wylla-$${var.environment}-create-account\"")
	assert.Contains(t, moduleStr, "runtime              = \"go122\"")
	assert.Contains(t, moduleStr, "available_memory_mb = 256")
	assert.Contains(t, moduleStr, "timeout             = 30")
}

func TestIntegration_GenerateTerraformCloudRun(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "GetUsers",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/users",
			},
		},
		{
			FunctionName:   "CreateUser",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/users",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:    handlers,
		OutputDir:   tmpDir,
		ModuleName:  "github.com/gravelight-studio/box",
		ProjectID:   "test-project",
		Region:      "us-central1",
		Environment: "staging",
		Logger:      zap.NewNop(),
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Verify cloud-run module was created
	modulePath := filepath.Join(tmpDir, "terraform", "modules", "cloud-run", "main.tf")
	assert.FileExists(t, modulePath)

	moduleContent, err := os.ReadFile(modulePath)
	require.NoError(t, err)
	moduleStr := string(moduleContent)

	// Should have service account for users
	assert.Contains(t, moduleStr, "resource \"google_service_account\" \"users\" {")
	assert.Contains(t, moduleStr, "account_id   = \"wylla-users-$${var.environment}\"")

	// Should have Cloud Run service
	assert.Contains(t, moduleStr, "resource \"google_cloud_run_service\" \"users\" {")
	assert.Contains(t, moduleStr, "name     = \"wylla-$${var.environment}-users\"")
	assert.Contains(t, moduleStr, "location = var.region")

	// Should have container config
	assert.Contains(t, moduleStr, "image = \"gcr.io/$${var.project_id}/users:latest\"")

	// Should have IAM for public access
	assert.Contains(t, moduleStr, "resource \"google_cloud_run_service_iam_member\" \"users_invoker\" {")
	assert.Contains(t, moduleStr, "role     = \"roles/run.invoker\"")
}

func TestIntegration_GenerateTerraformAPIGateway(t *testing.T) {
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:    handlers,
		OutputDir:   tmpDir,
		ModuleName:  "github.com/gravelight-studio/box",
		ProjectID:   "test-project",
		Region:      "us-central1",
		Environment: "production",
		Logger:      zap.NewNop(),
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Verify api-gateway module was created
	modulePath := filepath.Join(tmpDir, "terraform", "modules", "api-gateway", "main.tf")
	assert.FileExists(t, modulePath)

	moduleContent, err := os.ReadFile(modulePath)
	require.NoError(t, err)
	moduleStr := string(moduleContent)

	// Should have API resource
	assert.Contains(t, moduleStr, "resource \"google_api_gateway_api\" \"api\" {")
	assert.Contains(t, moduleStr, "api_id       = \"wylla-api-$${var.environment}\"")

	// Should have API config
	assert.Contains(t, moduleStr, "resource \"google_api_gateway_api_config\" \"api_config\" {")
	assert.Contains(t, moduleStr, "api           = google_api_gateway_api.api.api_id")

	// Should reference OpenAPI spec
	assert.Contains(t, moduleStr, "openapi_documents {")
	assert.Contains(t, moduleStr, "path     = \"openapi.yaml\"")
	assert.Contains(t, moduleStr, "contents = filebase64(\"$${path.module}/../../gateway/openapi.yaml\")")

	// Should have Gateway resource
	assert.Contains(t, moduleStr, "resource \"google_api_gateway_gateway\" \"gateway\" {")
	assert.Contains(t, moduleStr, "gateway_id   = \"wylla-gateway-$${var.environment}\"")
}

func TestIntegration_GenerateTerraformNetworking(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "TestHandler",
		PackageName:    "test",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "GET",
			Path:   "/test",
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:    []annotations.Handler{handler},
		OutputDir:   tmpDir,
		ModuleName:  "github.com/gravelight-studio/box",
		ProjectID:   "test-project",
		Region:      "us-central1",
		Environment: "dev",
		Logger:      zap.NewNop(),
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Verify networking module was created
	modulePath := filepath.Join(tmpDir, "terraform", "modules", "networking", "main.tf")
	assert.FileExists(t, modulePath)

	moduleContent, err := os.ReadFile(modulePath)
	require.NoError(t, err)
	moduleStr := string(moduleContent)

	// Should have VPC
	assert.Contains(t, moduleStr, "resource \"google_compute_network\" \"vpc\" {")
	assert.Contains(t, moduleStr, "name                    = \"wylla-vpc-$${var.environment}\"")

	// Should have subnet
	assert.Contains(t, moduleStr, "resource \"google_compute_subnetwork\" \"subnet\" {")
	assert.Contains(t, moduleStr, "ip_cidr_range = \"10.0.0.0/24\"")

	// Should have VPC connector
	assert.Contains(t, moduleStr, "resource \"google_vpc_access_connector\" \"connector\" {")
	assert.Contains(t, moduleStr, "name          = \"wylla-connector-$${var.environment}\"")

	// Should have Cloud SQL instance
	assert.Contains(t, moduleStr, "resource \"google_sql_database_instance\" \"main\" {")
	assert.Contains(t, moduleStr, "name             = \"wylla-db-$${var.environment}\"")
	assert.Contains(t, moduleStr, "database_version = \"POSTGRES_15\"")
}

func TestIntegration_GenerateTerraformEnvironmentVariables(t *testing.T) {
	handler := annotations.Handler{
		FunctionName:   "TestHandler",
		PackageName:    "test",
		DeploymentType: annotations.DeploymentFunction,
		Route: annotations.Route{
			Method: "GET",
			Path:   "/test",
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:    []annotations.Handler{handler},
		OutputDir:   tmpDir,
		ModuleName:  "github.com/gravelight-studio/box",
		ProjectID:   "test-project",
		Region:      "us-central1",
		Environment: "dev",
		Logger:      zap.NewNop(),
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Verify environment-specific tfvars files exist
	envDir := filepath.Join(tmpDir, "terraform", "environments")
	assert.DirExists(t, envDir)

	devTfvars := filepath.Join(envDir, "dev.tfvars")
	stagingTfvars := filepath.Join(envDir, "staging.tfvars")
	productionTfvars := filepath.Join(envDir, "production.tfvars")

	assert.FileExists(t, devTfvars, "dev.tfvars should exist")
	assert.FileExists(t, stagingTfvars, "staging.tfvars should exist")
	assert.FileExists(t, productionTfvars, "production.tfvars should exist")

	// Verify dev.tfvars content
	devContent, err := os.ReadFile(devTfvars)
	require.NoError(t, err)
	devStr := string(devContent)

	assert.Contains(t, devStr, "environment = \"dev\"")
	assert.Contains(t, devStr, "project_id  = \"YOUR_PROJECT_ID\"") // Template generates placeholder
	assert.Contains(t, devStr, "region      = \"us-central1\"")
	assert.Contains(t, devStr, "database_password = \"CHANGE_ME_DEV\"")

	// Verify staging.tfvars content
	stagingContent, err := os.ReadFile(stagingTfvars)
	require.NoError(t, err)
	stagingStr := string(stagingContent)

	assert.Contains(t, stagingStr, "environment = \"staging\"")
	assert.Contains(t, stagingStr, "database_password = \"CHANGE_ME_STAGING\"")

	// Verify production.tfvars content
	productionContent, err := os.ReadFile(productionTfvars)
	require.NoError(t, err)
	productionStr := string(productionContent)

	assert.Contains(t, productionStr, "environment = \"production\"")
	assert.Contains(t, productionStr, "database_password = \"CHANGE_ME_PRODUCTION\"")
}

func TestIntegration_GenerateTerraformNoHandlers(t *testing.T) {
	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:    []annotations.Handler{},
		OutputDir:   tmpDir,
		ModuleName:  "github.com/gravelight-studio/box",
		ProjectID:   "test-project",
		Region:      "us-central1",
		Environment: "dev",
		Logger:      zap.NewNop(),
	})

	err := gen.GenerateTerraform()
	require.NoError(t, err)

	// Terraform directory should not be created or be empty
	terraformDir := filepath.Join(tmpDir, "terraform")
	if _, err := os.Stat(terraformDir); err == nil {
		// If it exists, check that basic files exist but modules are minimal
		mainTF := filepath.Join(terraformDir, "main.tf")
		if _, err := os.Stat(mainTF); err == nil {
			mainContent, _ := os.ReadFile(mainTF)
			mainStr := string(mainContent)
			// Should still have terraform block but no handlers
			assert.Contains(t, mainStr, "terraform {")
		}
	}
}

func TestIntegration_GenerateTerraformComplete(t *testing.T) {
	// Test complete Terraform generation with mixed handlers
	handlers := []annotations.Handler{
		{
			FunctionName:   "CreateAccount",
			PackageName:    "accounts",
			PackagePath:    "internal/handlers/accounts",
			DeploymentType: annotations.DeploymentFunction,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/accounts",
			},
			Memory:  "256MB",
			Timeout: 30 * time.Second,
		},
		{
			FunctionName:   "GetUsers",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "GET",
				Path:   "/api/v1/users",
			},
		},
		{
			FunctionName:   "CreateUser",
			PackageName:    "users",
			PackagePath:    "internal/handlers/users",
			DeploymentType: annotations.DeploymentContainer,
			Route: annotations.Route{
				Method: "POST",
				Path:   "/api/v1/users",
			},
		},
	}

	tmpDir := t.TempDir()

	gen := NewGenerator(Config{
		Handlers:      handlers,
		OutputDir:     tmpDir,
		ModuleName:    "github.com/gravelight-studio/box",
		ProjectID:     "test-project",
		Region:        "us-central1",
		Environment:   "dev",
		Logger:        zap.NewNop(),
		CleanBuildDir: true,
	})

	// Generate everything
	err := gen.Generate()
	require.NoError(t, err)

	// Should have all directories including terraform
	functionsDir := filepath.Join(tmpDir, "functions")
	containersDir := filepath.Join(tmpDir, "containers")
	gatewayDir := filepath.Join(tmpDir, "gateway")
	terraformDir := filepath.Join(tmpDir, "terraform")

	assert.DirExists(t, functionsDir)
	assert.DirExists(t, containersDir)
	assert.DirExists(t, gatewayDir)
	assert.DirExists(t, terraformDir)

	// Verify Terraform structure is complete
	assert.DirExists(t, filepath.Join(terraformDir, "modules"))
	assert.DirExists(t, filepath.Join(terraformDir, "modules", "cloud-functions"))
	assert.DirExists(t, filepath.Join(terraformDir, "modules", "cloud-run"))
	assert.DirExists(t, filepath.Join(terraformDir, "modules", "api-gateway"))
	assert.DirExists(t, filepath.Join(terraformDir, "modules", "networking"))
	assert.DirExists(t, filepath.Join(terraformDir, "environments"))

	// Verify main.tf references all modules
	mainTF := filepath.Join(terraformDir, "main.tf")
	mainContent, err := os.ReadFile(mainTF)
	require.NoError(t, err)
	mainStr := string(mainContent)

	assert.Contains(t, mainStr, "module \"cloud_functions\" {")
	assert.Contains(t, mainStr, "module \"cloud_run\" {")
	assert.Contains(t, mainStr, "module \"api_gateway\" {")
	assert.Contains(t, mainStr, "module \"networking\" {")

	// Verify outputs.tf has useful outputs
	outputsTF := filepath.Join(terraformDir, "outputs.tf")
	outputsContent, err := os.ReadFile(outputsTF)
	require.NoError(t, err)
	outputsStr := string(outputsContent)

	assert.Contains(t, outputsStr, "output \"function_urls\" {")
	assert.Contains(t, outputsStr, "output \"service_urls\" {")
	assert.Contains(t, outputsStr, "output \"api_gateway_url\" {")
	assert.Contains(t, outputsStr, "output \"database_connection_name\" {")
}
