package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"go.uber.org/zap"

	"github.com/gravelight-studio/box/router"

	"github.com/gravelight-studio/box-example/handlers/health"
	"github.com/gravelight-studio/box-example/handlers/users"
)

func main() {
	// Create logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Box Example API")

	// Create Box router
	// This automatically parses annotations from handlers directory
	r, err := router.New(router.Config{
		HandlersDir: "./handlers",
		DB:          nil, // In a real app, initialize database connection here
		Logger:      logger,
	})
	if err != nil {
		logger.Fatal("Failed to create router", zap.Error(err))
	}

	// Create handler registry
	registry := router.NewHandlerRegistry(nil, logger)

	// Register health handlers
	registry.Register("health", "GetHealth", health.GetHealth)

	// Register user handlers
	registry.Register("users", "ListUsers", users.ListUsers)
	registry.Register("users", "GetUser", users.GetUser)
	registry.Register("users", "CreateUser", users.CreateUser)
	registry.Register("users", "StreamUserEvents", users.StreamUserEvents)

	// Register all handlers with router
	// This applies middleware based on annotations
	err = r.RegisterHandlers(registry)
	if err != nil {
		logger.Fatal("Failed to register handlers", zap.Error(err))
	}

	// Log registered routes
	handlers := r.GetHandlers()
	logger.Info("Registered handlers",
		zap.Int("count", len(handlers)))

	for _, h := range handlers {
		logger.Info("Route registered",
			zap.String("method", h.Route.Method),
			zap.String("path", h.Route.Path),
			zap.String("deployment", string(h.DeploymentType)),
			zap.String("function", h.FunctionName))
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	logger.Info("Server starting",
		zap.String("address", addr),
		zap.String("url", fmt.Sprintf("http://localhost%s", addr)))

	logger.Info("Example endpoints:",
		zap.String("health", fmt.Sprintf("http://localhost%s/health", addr)),
		zap.String("users", fmt.Sprintf("http://localhost%s/api/v1/users", addr)),
		zap.String("stream", fmt.Sprintf("http://localhost%s/api/v1/users/123/events", addr)))

	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Fatal("Server failed", zap.Error(err))
	}
}
