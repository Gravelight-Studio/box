package router

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

// handlerRegistry maps package.function names to HTTP handlers
type handlerRegistry struct {
	handlers map[string]http.HandlerFunc
	logger   *zap.Logger
}

// newHandlerRegistry creates a new handler registry
func newHandlerRegistry(logger *zap.Logger) *handlerRegistry {
	return &handlerRegistry{
		handlers: make(map[string]http.HandlerFunc),
		logger:   logger,
	}
}

// register adds a handler to the registry
func (r *handlerRegistry) register(packageName, functionName string, handler http.HandlerFunc) {
	key := fmt.Sprintf("%s.%s", packageName, functionName)
	r.handlers[key] = handler
	if r.logger != nil {
		r.logger.Debug("Handler registered", zap.String("handler", key))
	}
}

// getHandler retrieves a handler by package and function name
func (r *handlerRegistry) getHandler(packageName, functionName string) (http.HandlerFunc, error) {
	key := fmt.Sprintf("%s.%s", packageName, functionName)

	handler, exists := r.handlers[key]
	if !exists {
		return nil, fmt.Errorf("handler not found: %s", key)
	}

	return handler, nil
}
