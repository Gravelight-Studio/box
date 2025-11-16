package router

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// HandlerRegistry maps package.function names to HTTP handlers
type HandlerRegistry struct {
	handlers map[string]http.HandlerFunc
	logger   *zap.Logger
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry(logger *zap.Logger) *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]http.HandlerFunc),
		logger:   logger,
	}
}

// Register adds a handler to the registry
func (r *HandlerRegistry) Register(packageName, functionName string, handler http.HandlerFunc) {
	key := fmt.Sprintf("%s.%s", packageName, functionName)
	r.handlers[key] = handler
	if r.logger != nil {
		r.logger.Debug("Handler registered", zap.String("handler", key))
	}
}

// GetHandler retrieves a handler by package and function name
func (r *HandlerRegistry) GetHandler(packageName, functionName string) (http.HandlerFunc, error) {
	key := fmt.Sprintf("%s.%s", packageName, functionName)

	handler, exists := r.handlers[key]
	if !exists {
		return nil, fmt.Errorf("handler not found: %s", key)
	}

	return handler, nil
}

// ListHandlers returns all registered handler names
func (r *HandlerRegistry) ListHandlers() []string {
	handlers := make([]string, 0, len(r.handlers))
	for key := range r.handlers {
		handlers = append(handlers, key)
	}
	return handlers
}

// Count returns the number of registered handlers
func (r *HandlerRegistry) Count() int {
	return len(r.handlers)
}
