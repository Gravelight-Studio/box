package router

import (
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// HandlerFactory is a function that creates an HTTP handler
// It receives database and logger dependencies and returns the handler function
type HandlerFactory func(db *pgxpool.Pool, logger *zap.Logger) http.HandlerFunc

// HandlerRegistry maps package.function names to handler factories
type HandlerRegistry struct {
	handlers map[string]HandlerFactory
	db       *pgxpool.Pool
	logger   *zap.Logger
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry(db *pgxpool.Pool, logger *zap.Logger) *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]HandlerFactory),
		db:       db,
		logger:   logger,
	}
}

// Register adds a handler to the registry
func (r *HandlerRegistry) Register(packageName, functionName string, factory HandlerFactory) {
	key := fmt.Sprintf("%s.%s", packageName, functionName)
	r.handlers[key] = factory
	r.logger.Debug("Handler registered", zap.String("handler", key))
}

// GetHandler retrieves a handler by package and function name
func (r *HandlerRegistry) GetHandler(packageName, functionName string) (http.HandlerFunc, error) {
	key := fmt.Sprintf("%s.%s", packageName, functionName)

	factory, exists := r.handlers[key]
	if !exists {
		return nil, fmt.Errorf("handler not found: %s", key)
	}

	// Create handler instance with dependencies
	return factory(r.db, r.logger), nil
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
