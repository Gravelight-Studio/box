package annotations

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected *Handler
		wantErr  bool
	}{
		{
			name: "function with all annotations",
			source: `package test

// CreateAccount creates a new account
// @box:function
// @box:path POST /api/v1/accounts
// @box:auth required
// @box:ratelimit 100/hour
// @box:timeout 30s
// @box:memory 256MB
func CreateAccount(w http.ResponseWriter, r *http.Request) {
	// implementation
}
`,
			expected: &Handler{
				FunctionName:   "CreateAccount",
				PackageName:    "test",
				DeploymentType: DeploymentFunction,
				Route: Route{
					Method: "POST",
					Path:   "/api/v1/accounts",
				},
				Auth: AuthConfig{
					Type: AuthRequired,
				},
				RateLimit: &RateLimitConfig{
					Count:  100,
					Period: time.Hour,
					Raw:    "100/hour",
				},
				Timeout: 30 * time.Second,
				Memory:  "256MB",
			},
			wantErr: false,
		},
		{
			name: "container with concurrency",
			source: `package test

// StreamChat streams chat messages
// @box:container
// @box:path GET /api/v1/chat/{id}/stream
// @box:auth required
// @box:concurrency 100
func StreamChat(w http.ResponseWriter, r *http.Request) {
	// implementation
}
`,
			expected: &Handler{
				FunctionName:   "StreamChat",
				PackageName:    "test",
				DeploymentType: DeploymentContainer,
				Route: Route{
					Method: "GET",
					Path:   "/api/v1/chat/{id}/stream",
				},
				Auth: AuthConfig{
					Type: AuthRequired,
				},
				Concurrency: 100,
			},
			wantErr: false,
		},
		{
			name: "optional auth",
			source: `package test

// GetProfile gets user profile
// @box:function
// @box:path GET /api/v1/profile
// @box:auth optional
func GetProfile(w http.ResponseWriter, r *http.Request) {
	// implementation
}
`,
			expected: &Handler{
				FunctionName:   "GetProfile",
				PackageName:    "test",
				DeploymentType: DeploymentFunction,
				Route: Route{
					Method: "GET",
					Path:   "/api/v1/profile",
				},
				Auth: AuthConfig{
					Type: AuthOptional,
				},
			},
			wantErr: false,
		},
		{
			name: "no auth",
			source: `package test

// HealthCheck health check endpoint
// @box:function
// @box:path GET /health
// @box:auth none
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	// implementation
}
`,
			expected: &Handler{
				FunctionName:   "HealthCheck",
				PackageName:    "test",
				DeploymentType: DeploymentFunction,
				Route: Route{
					Method: "GET",
					Path:   "/health",
				},
				Auth: AuthConfig{
					Type: AuthNone,
				},
			},
			wantErr: false,
		},
		{
			name: "cors configuration",
			source: `package test

// GetPublicData gets public data
// @box:function
// @box:path GET /api/v1/public
// @box:cors origins=*
func GetPublicData(w http.ResponseWriter, r *http.Request) {
	// implementation
}
`,
			expected: &Handler{
				FunctionName:   "GetPublicData",
				PackageName:    "test",
				DeploymentType: DeploymentFunction,
				Route: Route{
					Method: "GET",
					Path:   "/api/v1/public",
				},
				Auth: AuthConfig{
					Type: AuthNone,
				},
				CORS: &CORSConfig{
					AllowedOrigins: []string{"*"},
					Raw:            "origins=*",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.go")

			err := os.WriteFile(tmpFile, []byte(tt.source), 0644)
			if err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			// Parse the file
			parser := NewParser()
			result, err := parser.ParseFile(tmpFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expected == nil {
				return
			}

			if len(result.Handlers) != 1 {
				t.Fatalf("Expected 1 handler, got %d", len(result.Handlers))
			}

			handler := result.Handlers[0]

			// Compare fields
			if handler.FunctionName != tt.expected.FunctionName {
				t.Errorf("FunctionName = %v, want %v", handler.FunctionName, tt.expected.FunctionName)
			}

			if handler.DeploymentType != tt.expected.DeploymentType {
				t.Errorf("DeploymentType = %v, want %v", handler.DeploymentType, tt.expected.DeploymentType)
			}

			if handler.Route.Method != tt.expected.Route.Method {
				t.Errorf("Route.Method = %v, want %v", handler.Route.Method, tt.expected.Route.Method)
			}

			if handler.Route.Path != tt.expected.Route.Path {
				t.Errorf("Route.Path = %v, want %v", handler.Route.Path, tt.expected.Route.Path)
			}

			if handler.Auth.Type != tt.expected.Auth.Type {
				t.Errorf("Auth.Type = %v, want %v", handler.Auth.Type, tt.expected.Auth.Type)
			}

			if tt.expected.RateLimit != nil {
				if handler.RateLimit == nil {
					t.Error("Expected RateLimit to be set")
				} else {
					if handler.RateLimit.Count != tt.expected.RateLimit.Count {
						t.Errorf("RateLimit.Count = %v, want %v", handler.RateLimit.Count, tt.expected.RateLimit.Count)
					}
					if handler.RateLimit.Period != tt.expected.RateLimit.Period {
						t.Errorf("RateLimit.Period = %v, want %v", handler.RateLimit.Period, tt.expected.RateLimit.Period)
					}
				}
			}

			if tt.expected.CORS != nil {
				if handler.CORS == nil {
					t.Error("Expected CORS to be set")
				} else {
					if len(handler.CORS.AllowedOrigins) != len(tt.expected.CORS.AllowedOrigins) {
						t.Errorf("CORS origins count = %v, want %v", len(handler.CORS.AllowedOrigins), len(tt.expected.CORS.AllowedOrigins))
					}
				}
			}

			if handler.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %v, want %v", handler.Timeout, tt.expected.Timeout)
			}

			if handler.Memory != tt.expected.Memory {
				t.Errorf("Memory = %v, want %v", handler.Memory, tt.expected.Memory)
			}

			if handler.Concurrency != tt.expected.Concurrency {
				t.Errorf("Concurrency = %v, want %v", handler.Concurrency, tt.expected.Concurrency)
			}
		})
	}
}

func TestParsePathVariations(t *testing.T) {
	tests := []struct {
		name     string
		pathLine string
		expected Route
		wantErr  bool
	}{
		{
			name:     "simple path",
			pathLine: "GET /api/v1/accounts",
			expected: Route{Method: "GET", Path: "/api/v1/accounts"},
			wantErr:  false,
		},
		{
			name:     "path with single parameter",
			pathLine: "GET /api/v1/accounts/{id}",
			expected: Route{Method: "GET", Path: "/api/v1/accounts/{id}"},
			wantErr:  false,
		},
		{
			name:     "path with multiple parameters",
			pathLine: "GET /api/v1/accounts/{accountId}/chats/{chatId}",
			expected: Route{Method: "GET", Path: "/api/v1/accounts/{accountId}/chats/{chatId}"},
			wantErr:  false,
		},
		{
			name:     "POST method",
			pathLine: "POST /api/v1/accounts",
			expected: Route{Method: "POST", Path: "/api/v1/accounts"},
			wantErr:  false,
		},
		{
			name:     "DELETE method",
			pathLine: "DELETE /api/v1/accounts/{id}",
			expected: Route{Method: "DELETE", Path: "/api/v1/accounts/{id}"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{}
			parser := NewParser()

			err := parser.parsePath(handler, tt.pathLine)

			if (err != nil) != tt.wantErr {
				t.Errorf("parsePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if handler.Route.Method != tt.expected.Method {
					t.Errorf("Method = %v, want %v", handler.Route.Method, tt.expected.Method)
				}
				if handler.Route.Path != tt.expected.Path {
					t.Errorf("Path = %v, want %v", handler.Route.Path, tt.expected.Path)
				}
			}
		})
	}
}

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected *RateLimitConfig
		wantErr  bool
	}{
		{
			name:  "per hour",
			value: "100/hour",
			expected: &RateLimitConfig{
				Count:  100,
				Period: time.Hour,
				Raw:    "100/hour",
			},
			wantErr: false,
		},
		{
			name:  "per minute",
			value: "60/minute",
			expected: &RateLimitConfig{
				Count:  60,
				Period: time.Minute,
				Raw:    "60/minute",
			},
			wantErr: false,
		},
		{
			name:  "per second",
			value: "10/second",
			expected: &RateLimitConfig{
				Count:  10,
				Period: time.Second,
				Raw:    "10/second",
			},
			wantErr: false,
		},
		{
			name:    "invalid format",
			value:   "100",
			wantErr: true,
		},
		{
			name:    "invalid count",
			value:   "abc/hour",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &Handler{}
			parser := NewParser()

			err := parser.parseRateLimit(handler, tt.value)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseRateLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expected != nil {
				if handler.RateLimit.Count != tt.expected.Count {
					t.Errorf("Count = %v, want %v", handler.RateLimit.Count, tt.expected.Count)
				}
				if handler.RateLimit.Period != tt.expected.Period {
					t.Errorf("Period = %v, want %v", handler.RateLimit.Period, tt.expected.Period)
				}
			}
		})
	}
}

func TestValidator(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		handler     Handler
		wantErrors  int
		errorContains string
	}{
		{
			name: "valid function handler",
			handler: Handler{
				FunctionName:   "Test",
				DeploymentType: DeploymentFunction,
				Route:          Route{Method: "GET", Path: "/test"},
				Auth:           AuthConfig{Type: AuthRequired},
			},
			wantErrors: 0,
		},
		{
			name: "missing deployment type",
			handler: Handler{
				FunctionName: "Test",
				Route:        Route{Method: "GET", Path: "/test"},
			},
			wantErrors:    1,
			errorContains: "deployment type",
		},
		{
			name: "missing route",
			handler: Handler{
				FunctionName:   "Test",
				DeploymentType: DeploymentFunction,
			},
			wantErrors:    1,
			errorContains: "path",
		},
		{
			name: "invalid memory for function",
			handler: Handler{
				FunctionName:   "Test",
				DeploymentType: DeploymentFunction,
				Route:          Route{Method: "GET", Path: "/test"},
				Memory:         "99MB",
			},
			wantErrors:    1,
			errorContains: "memory",
		},
		{
			name: "concurrency on function (warning)",
			handler: Handler{
				FunctionName:   "Test",
				DeploymentType: DeploymentFunction,
				Route:          Route{Method: "GET", Path: "/test"},
				Concurrency:    10,
			},
			wantErrors:    1,
			errorContains: "Concurrency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateHandler(tt.handler)

			if len(errors) != tt.wantErrors {
				t.Errorf("validateHandler() got %d errors, want %d", len(errors), tt.wantErrors)
				for _, err := range errors {
					t.Logf("  Error: %s", err.Reason)
				}
			}

			if tt.errorContains != "" && len(errors) > 0 {
				found := false
				for _, err := range errors {
					if containsString(err.Reason, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing %q, but not found in: %v", tt.errorContains, errors)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
