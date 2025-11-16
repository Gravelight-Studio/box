package router

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/gravelight-studio/box/go/annotations"
)

// Router wraps Chi router with annotation-driven routing
type Router struct {
	chi.Router
	handlers []annotations.Handler
	logger   *zap.Logger
}

// Config holds router configuration
type Config struct {
	HandlersDir string // Directory to scan for handlers (e.g., "./internal/handlers")
	Logger      *zap.Logger
}

// New creates a new annotation-driven router
func New(config Config) (*Router, error) {
	// Parse handlers from directory
	parser := annotations.NewParser()
	result, err := parser.ParseDirectory(config.HandlersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse handlers: %w", err)
	}

	// Log parse errors but continue
	if len(result.Errors) > 0 {
		config.Logger.Warn("Encountered parse errors",
			zap.Int("count", len(result.Errors)))
		for _, parseErr := range result.Errors {
			config.Logger.Warn("Parse error",
				zap.String("file", parseErr.FilePath),
				zap.Int("line", parseErr.LineNumber),
				zap.String("message", parseErr.Message))
		}
	}

	// Validate handlers
	validator := annotations.NewValidator()
	validationErrors := validator.Validate(result.Handlers)
	pathErrors := validator.ValidateUniquePaths(result.Handlers)
	validationErrors = append(validationErrors, pathErrors...)

	if len(validationErrors) > 0 {
		config.Logger.Error("Handler validation failed",
			zap.Int("count", len(validationErrors)))
		for _, err := range validationErrors {
			config.Logger.Error("Validation error",
				zap.String("handler", err.Handler),
				zap.String("annotation", err.Annotation),
				zap.String("reason", err.Reason))
		}
		return nil, fmt.Errorf("handler validation failed with %d errors", len(validationErrors))
	}

	config.Logger.Info("Handlers parsed and validated",
		zap.Int("count", len(result.Handlers)))

	// Create router
	r := &Router{
		Router:   chi.NewRouter(),
		handlers: result.Handlers,
		logger:   config.Logger,
	}

	return r, nil
}

// RegisterHandlers registers all parsed handlers with the router
func (r *Router) RegisterHandlers(registry *HandlerRegistry) error {
	for _, handler := range r.handlers {
		r.logger.Info("Registering handler",
			zap.String("function", handler.FunctionName),
			zap.String("method", handler.Route.Method),
			zap.String("path", handler.Route.Path),
			zap.String("deployment", string(handler.DeploymentType)))

		// Get the actual handler function from registry
		handlerFunc, err := registry.GetHandler(handler.PackageName, handler.FunctionName)
		if err != nil {
			r.logger.Error("Handler not found in registry",
				zap.String("package", handler.PackageName),
				zap.String("function", handler.FunctionName),
				zap.Error(err))
			return fmt.Errorf("handler %s.%s not found: %w", handler.PackageName, handler.FunctionName, err)
		}

		// Build middleware chain for this handler
		middlewares := buildMiddlewareChain(handler, r.logger)

		// Apply middleware and register route
		finalHandler := applyMiddleware(handlerFunc, middlewares)

		// Register based on HTTP method
		switch handler.Route.Method {
		case "GET":
			r.Get(handler.Route.Path, finalHandler)
		case "POST":
			r.Post(handler.Route.Path, finalHandler)
		case "PUT":
			r.Put(handler.Route.Path, finalHandler)
		case "DELETE":
			r.Delete(handler.Route.Path, finalHandler)
		case "PATCH":
			r.Patch(handler.Route.Path, finalHandler)
		case "OPTIONS":
			r.Options(handler.Route.Path, finalHandler)
		case "HEAD":
			r.Head(handler.Route.Path, finalHandler)
		default:
			return fmt.Errorf("unsupported HTTP method: %s", handler.Route.Method)
		}
	}

	r.logger.Info("All handlers registered successfully",
		zap.Int("count", len(r.handlers)))

	return nil
}

// GetHandlers returns the list of parsed handlers
func (r *Router) GetHandlers() []annotations.Handler {
	return r.handlers
}

// buildMiddlewareChain creates middleware chain based on annotations
func buildMiddlewareChain(handler annotations.Handler, logger *zap.Logger) []func(http.Handler) http.Handler {
	var middlewares []func(http.Handler) http.Handler

	// Add CORS middleware if specified
	if handler.CORS != nil {
		middlewares = append(middlewares, CORSMiddleware(handler.CORS))
	}

	// Add auth middleware if specified
	if handler.Auth.Type != annotations.AuthNone {
		middlewares = append(middlewares, AuthMiddleware(handler.Auth, logger))
	}

	// Add rate limiting middleware if specified
	if handler.RateLimit != nil {
		middlewares = append(middlewares, RateLimitMiddleware(handler.RateLimit, logger))
	}

	// Add timeout middleware if specified
	if handler.Timeout > 0 {
		middlewares = append(middlewares, TimeoutMiddleware(handler.Timeout))
	}

	return middlewares
}

// applyMiddleware applies middleware chain to handler
func applyMiddleware(handler http.HandlerFunc, middlewares []func(http.Handler) http.Handler) http.HandlerFunc {
	// Apply middleware in reverse order (last middleware wraps first)
	h := http.Handler(handler)
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h.ServeHTTP
}
