package annotations

import (
	"time"
)

// DeploymentType indicates where a handler should be deployed
type DeploymentType string

const (
	DeploymentFunction  DeploymentType = "function"  // GCP Cloud Function
	DeploymentContainer DeploymentType = "container" // GCP Cloud Run
)

// AuthType indicates the authentication requirement for a handler
type AuthType string

const (
	AuthRequired AuthType = "required" // Bearer token required
	AuthOptional AuthType = "optional" // Bearer token optional (check if present)
	AuthNone     AuthType = "none"     // No authentication
)

// Handler represents a parsed HTTP handler with its annotations
type Handler struct {
	// Source code metadata
	FunctionName string // Go function name (e.g., "CreateAccount")
	PackagePath  string // Import path (e.g., "github.com/gravelight-studio/box/internal/handlers/accounts")
	PackageName  string // Package name (e.g., "accounts")
	FilePath     string // Absolute file path
	LineNumber   int    // Line number of function declaration

	// Deployment configuration
	DeploymentType DeploymentType // function or container
	ServiceName    string          // Service group name for containers (e.g., "chat-service")

	// HTTP routing
	Route Route

	// Middleware configuration
	Auth      AuthConfig
	RateLimit *RateLimitConfig // nil if not specified
	CORS      *CORSConfig      // nil if not specified
	Timeout   time.Duration    // 0 if not specified

	// Resource configuration (Cloud Functions)
	Memory string // e.g., "128MB", "256MB", "512MB"

	// Resource configuration (Cloud Run)
	Concurrency int // Max concurrent requests per instance (1-1000)
}

// Route represents an HTTP route
type Route struct {
	Method string // GET, POST, PUT, DELETE, PATCH, OPTIONS
	Path   string // e.g., "/api/v1/accounts", "/api/v1/accounts/{id}"
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type AuthType
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Count  int           // Number of requests
	Period time.Duration // Time period (e.g., 1 hour, 1 minute)
	Raw    string        // Original string (e.g., "100/hour")
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string // e.g., ["*"], ["https://example.com"]
	Raw            string   // Original string (e.g., "origins=*")
}

// ParsedAnnotations represents all annotations found in a directory/file
type ParsedAnnotations struct {
	Handlers []Handler
	Errors   []ParseError
}

// ParseError represents an error encountered during parsing
type ParseError struct {
	FilePath   string
	LineNumber int
	Message    string
	Annotation string // The problematic annotation line
}

// Error implements the error interface
func (e ParseError) Error() string {
	return e.Message
}

// AnnotationError represents validation errors for annotations
type AnnotationError struct {
	Handler    string
	Annotation string
	Reason     string
}

// Error implements the error interface
func (e AnnotationError) Error() string {
	return e.Reason
}
