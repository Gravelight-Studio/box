package annotations

import (
	"fmt"
	"strings"
)

// Validator validates parsed annotations for correctness and completeness
type Validator struct {
	// Configuration for validation rules
}

// NewValidator creates a new annotation validator
func NewValidator() *Validator {
	return &Validator{}
}

// Validate checks if all handlers have valid and complete annotations
func (v *Validator) Validate(handlers []Handler) []AnnotationError {
	var errors []AnnotationError

	for _, handler := range handlers {
		errors = append(errors, v.validateHandler(handler)...)
	}

	return errors
}

// validateHandler validates a single handler
func (v *Validator) validateHandler(handler Handler) []AnnotationError {
	var errors []AnnotationError

	// Check deployment type is set
	if handler.DeploymentType == "" {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:function or @wylla:container",
			Reason:     "Missing deployment type annotation. Add @wylla:function or @wylla:container",
		})
	}

	// Check route is set
	if handler.Route.Method == "" || handler.Route.Path == "" {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:path",
			Reason:     "Missing path annotation. Add @wylla:path METHOD /path",
		})
	}

	// Validate route path format
	if handler.Route.Path != "" {
		errors = append(errors, v.validatePath(handler)...)
	}

	// Validate deployment-specific config
	if handler.DeploymentType == DeploymentFunction {
		errors = append(errors, v.validateFunctionConfig(handler)...)
	} else if handler.DeploymentType == DeploymentContainer {
		errors = append(errors, v.validateContainerConfig(handler)...)
	}

	// Validate rate limit if present
	if handler.RateLimit != nil {
		errors = append(errors, v.validateRateLimit(handler)...)
	}

	// Validate CORS if present
	if handler.CORS != nil {
		errors = append(errors, v.validateCORS(handler)...)
	}

	// Validate timeout if present
	if handler.Timeout > 0 {
		errors = append(errors, v.validateTimeout(handler)...)
	}

	return errors
}

// validatePath validates the route path format
func (v *Validator) validatePath(handler Handler) []AnnotationError {
	var errors []AnnotationError

	path := handler.Route.Path

	// Path must start with /
	if !strings.HasPrefix(path, "/") {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:path",
			Reason:     fmt.Sprintf("Path must start with '/': %s", path),
		})
	}

	// Path segments should not end with /
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:path",
			Reason:     fmt.Sprintf("Path should not end with '/': %s", path),
		})
	}

	// Check for valid path parameter syntax {id}, {name}, etc.
	if strings.Contains(path, "{") || strings.Contains(path, "}") {
		if !v.hasValidPathParams(path) {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:path",
				Reason:     fmt.Sprintf("Invalid path parameter syntax: %s (use {paramName})", path),
			})
		}
	}

	return errors
}

// hasValidPathParams checks if path parameters are properly formatted
func (v *Validator) hasValidPathParams(path string) bool {
	// Simple validation: count opening and closing braces
	openCount := strings.Count(path, "{")
	closeCount := strings.Count(path, "}")

	if openCount != closeCount {
		return false
	}

	// Check each parameter is properly formatted
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.Contains(part, "{") {
			if !strings.HasPrefix(part, "{") || !strings.HasSuffix(part, "}") {
				return false
			}
			// Check parameter name is not empty
			paramName := strings.TrimPrefix(strings.TrimSuffix(part, "}"), "{")
			if len(paramName) == 0 {
				return false
			}
		}
	}

	return true
}

// validateFunctionConfig validates Cloud Function specific configuration
func (v *Validator) validateFunctionConfig(handler Handler) []AnnotationError {
	var errors []AnnotationError

	// Validate memory if specified
	if handler.Memory != "" {
		validMemory := map[string]bool{
			"128MB":  true,
			"256MB":  true,
			"512MB":  true,
			"1GB":    true,
			"2GB":    true,
			"4GB":    true,
			"8GB":    true,
			"16GB":   true,
		}

		if !validMemory[handler.Memory] {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:memory",
				Reason:     fmt.Sprintf("Invalid memory value: %s (valid: 128MB, 256MB, 512MB, 1GB, 2GB, 4GB, 8GB, 16GB)", handler.Memory),
			})
		}
	}

	// Warn if concurrency is set for a function (it's a container config)
	if handler.Concurrency > 0 {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:concurrency",
			Reason:     "Concurrency is not applicable to Cloud Functions, only Cloud Run containers",
		})
	}

	return errors
}

// validateContainerConfig validates Cloud Run specific configuration
func (v *Validator) validateContainerConfig(handler Handler) []AnnotationError {
	var errors []AnnotationError

	// Validate concurrency if specified
	if handler.Concurrency > 0 {
		if handler.Concurrency < 1 || handler.Concurrency > 1000 {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:concurrency",
				Reason:     fmt.Sprintf("Concurrency must be between 1 and 1000, got: %d", handler.Concurrency),
			})
		}
	}

	// Warn if memory is set for a container (it's less relevant than for functions)
	if handler.Memory != "" {
		// Note: Cloud Run also supports memory limits, but it's configured differently
		// This is more of a notice than an error
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:memory",
			Reason:     "Note: Memory for Cloud Run containers is typically configured at the service level, not per handler",
		})
	}

	return errors
}

// validateRateLimit validates rate limiting configuration
func (v *Validator) validateRateLimit(handler Handler) []AnnotationError {
	var errors []AnnotationError

	if handler.RateLimit.Count <= 0 {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:ratelimit",
			Reason:     fmt.Sprintf("Rate limit count must be positive, got: %d", handler.RateLimit.Count),
		})
	}

	// Check for reasonable limits (warn if too high or too low)
	if handler.RateLimit.Count > 10000 {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:ratelimit",
			Reason:     fmt.Sprintf("Rate limit seems very high: %s (consider if this is intentional)", handler.RateLimit.Raw),
		})
	}

	if handler.RateLimit.Count < 10 && handler.RateLimit.Period.Hours() >= 1 {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:ratelimit",
			Reason:     fmt.Sprintf("Rate limit seems very low: %s (consider if this is intentional)", handler.RateLimit.Raw),
		})
	}

	return errors
}

// validateCORS validates CORS configuration
func (v *Validator) validateCORS(handler Handler) []AnnotationError {
	var errors []AnnotationError

	if len(handler.CORS.AllowedOrigins) == 0 {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:cors",
			Reason:     "CORS must specify at least one origin",
		})
		return errors
	}

	// Validate origin format (if not wildcard)
	for _, origin := range handler.CORS.AllowedOrigins {
		if origin == "*" {
			continue
		}

		// Basic URL validation
		if !strings.HasPrefix(origin, "http://") && !strings.HasPrefix(origin, "https://") {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:cors",
				Reason:     fmt.Sprintf("CORS origin must start with http:// or https://, got: %s", origin),
			})
		}
	}

	return errors
}

// validateTimeout validates timeout configuration
func (v *Validator) validateTimeout(handler Handler) []AnnotationError {
	var errors []AnnotationError

	// Cloud Functions have max timeout of 540s (9 minutes)
	// Cloud Run has max timeout of 3600s (1 hour)
	if handler.DeploymentType == DeploymentFunction {
		maxTimeout := int64(540) // 9 minutes in seconds
		if handler.Timeout.Seconds() > float64(maxTimeout) {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:timeout",
				Reason:     fmt.Sprintf("Cloud Function timeout cannot exceed 540s (9 minutes), got: %v", handler.Timeout),
			})
		}
	} else if handler.DeploymentType == DeploymentContainer {
		maxTimeout := int64(3600) // 1 hour in seconds
		if handler.Timeout.Seconds() > float64(maxTimeout) {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:timeout",
				Reason:     fmt.Sprintf("Cloud Run timeout cannot exceed 3600s (1 hour), got: %v", handler.Timeout),
			})
		}
	}

	// Warn about very short timeouts
	if handler.Timeout.Seconds() < 5 {
		errors = append(errors, AnnotationError{
			Handler:    handler.FunctionName,
			Annotation: "@wylla:timeout",
			Reason:     fmt.Sprintf("Timeout is very short: %v (consider if this is intentional)", handler.Timeout),
		})
	}

	return errors
}

// ValidateUniquePaths checks if there are duplicate paths across handlers
func (v *Validator) ValidateUniquePaths(handlers []Handler) []AnnotationError {
	var errors []AnnotationError
	seen := make(map[string]string) // path+method -> handler name

	for _, handler := range handlers {
		key := fmt.Sprintf("%s %s", handler.Route.Method, handler.Route.Path)

		if existing, exists := seen[key]; exists {
			errors = append(errors, AnnotationError{
				Handler:    handler.FunctionName,
				Annotation: "@wylla:path",
				Reason:     fmt.Sprintf("Duplicate route: %s already defined in handler %s", key, existing),
			})
		} else {
			seen[key] = handler.FunctionName
		}
	}

	return errors
}
