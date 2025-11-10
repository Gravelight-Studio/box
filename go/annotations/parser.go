package annotations

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Parser handles parsing of Go source files for annotations
type Parser struct {
	fset *token.FileSet
}

// NewParser creates a new annotation parser
func NewParser() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
	}
}

// ParseDirectory parses all Go files in a directory recursively
func (p *Parser) ParseDirectory(dir string) (*ParsedAnnotations, error) {
	result := &ParsedAnnotations{
		Handlers: make([]Handler, 0),
		Errors:   make([]ParseError, 0),
	}

	// Walk the directory tree
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the file
		fileResult, err := p.ParseFile(path)
		if err != nil {
			result.Errors = append(result.Errors, ParseError{
				FilePath: path,
				Message:  fmt.Sprintf("Failed to parse file: %v", err),
			})
			return nil // Continue parsing other files
		}

		// Merge results
		result.Handlers = append(result.Handlers, fileResult.Handlers...)
		result.Errors = append(result.Errors, fileResult.Errors...)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return result, nil
}

// ParseFile parses a single Go file for annotations
func (p *Parser) ParseFile(filePath string) (*ParsedAnnotations, error) {
	result := &ParsedAnnotations{
		Handlers: make([]Handler, 0),
		Errors:   make([]ParseError, 0),
	}

	// Parse the file
	file, err := parser.ParseFile(p.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	// Extract package name
	packageName := file.Name.Name

	// Find all function declarations with annotations
	ast.Inspect(file, func(n ast.Node) bool {
		// Look for function declarations
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}

		// Skip methods (we only want top-level functions)
		if funcDecl.Recv != nil {
			return true
		}

		// Get function name
		funcName := funcDecl.Name.Name

		// Get function position
		position := p.fset.Position(funcDecl.Pos())

		// Extract annotations from doc comments
		if funcDecl.Doc == nil {
			return true // No comments, skip
		}

		// Parse annotations from comments
		handler, parseErrs := p.parseAnnotations(funcDecl.Doc, funcName, packageName, absPath, position.Line)

		if handler != nil {
			result.Handlers = append(result.Handlers, *handler)
		}

		result.Errors = append(result.Errors, parseErrs...)

		return true
	})

	return result, nil
}

// parseAnnotations extracts @wylla:* annotations from comment group
func (p *Parser) parseAnnotations(doc *ast.CommentGroup, funcName, packageName, filePath string, lineNumber int) (*Handler, []ParseError) {
	handler := &Handler{
		FunctionName: funcName,
		PackageName:  packageName,
		FilePath:     filePath,
		LineNumber:   lineNumber,
		Auth: AuthConfig{
			Type: AuthNone, // Default to no auth
		},
	}

	var errors []ParseError
	hasBoxAnnotation := false

	// Process each comment line
	for _, comment := range doc.List {
		text := strings.TrimSpace(comment.Text)

		// Remove comment markers
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)

		// Check if it's a Box annotation
		if !strings.HasPrefix(text, "@box:") {
			continue
		}

		hasBoxAnnotation = true

		// Parse the annotation
		parts := strings.SplitN(text, ":", 2)
		if len(parts) != 2 {
			errors = append(errors, ParseError{
				FilePath:   filePath,
				LineNumber: lineNumber,
				Message:    fmt.Sprintf("Invalid annotation format: %s", text),
				Annotation: text,
			})
			continue
		}

		annotationType := strings.TrimSpace(parts[1])
		var annotationValue string

		// Split annotation type and value
		if spaceIdx := strings.Index(annotationType, " "); spaceIdx > 0 {
			annotationValue = strings.TrimSpace(annotationType[spaceIdx+1:])
			annotationType = annotationType[:spaceIdx]
		}

		// Parse based on annotation type
		switch annotationType {
		case "function":
			handler.DeploymentType = DeploymentFunction

		case "container":
			handler.DeploymentType = DeploymentContainer
			// Parse optional service=name parameter
			if annotationValue != "" {
				if err := p.parseContainerService(handler, annotationValue); err != nil {
					errors = append(errors, ParseError{
						FilePath:   filePath,
						LineNumber: lineNumber,
						Message:    fmt.Sprintf("Invalid container annotation: %v", err),
						Annotation: text,
					})
				}
			}

		case "path":
			if err := p.parsePath(handler, annotationValue); err != nil {
				errors = append(errors, ParseError{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Message:    fmt.Sprintf("Invalid path annotation: %v", err),
					Annotation: text,
				})
			}

		case "auth":
			if err := p.parseAuth(handler, annotationValue); err != nil {
				errors = append(errors, ParseError{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Message:    fmt.Sprintf("Invalid auth annotation: %v", err),
					Annotation: text,
				})
			}

		case "ratelimit":
			if err := p.parseRateLimit(handler, annotationValue); err != nil {
				errors = append(errors, ParseError{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Message:    fmt.Sprintf("Invalid ratelimit annotation: %v", err),
					Annotation: text,
				})
			}

		case "cors":
			if err := p.parseCORS(handler, annotationValue); err != nil {
				errors = append(errors, ParseError{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Message:    fmt.Sprintf("Invalid cors annotation: %v", err),
					Annotation: text,
				})
			}

		case "timeout":
			if err := p.parseTimeout(handler, annotationValue); err != nil {
				errors = append(errors, ParseError{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Message:    fmt.Sprintf("Invalid timeout annotation: %v", err),
					Annotation: text,
				})
			}

		case "memory":
			handler.Memory = annotationValue

		case "concurrency":
			var concurrency int
			if _, err := fmt.Sscanf(annotationValue, "%d", &concurrency); err != nil {
				errors = append(errors, ParseError{
					FilePath:   filePath,
					LineNumber: lineNumber,
					Message:    fmt.Sprintf("Invalid concurrency value: %s", annotationValue),
					Annotation: text,
				})
			} else {
				handler.Concurrency = concurrency
			}

		default:
			errors = append(errors, ParseError{
				FilePath:   filePath,
				LineNumber: lineNumber,
				Message:    fmt.Sprintf("Unknown annotation type: %s", annotationType),
				Annotation: text,
			})
		}
	}

	// If no Box annotations found, return nil
	if !hasBoxAnnotation {
		return nil, errors
	}

	return handler, errors
}

// parsePath parses @box:path METHOD /path/to/resource
func (p *Parser) parsePath(handler *Handler, value string) error {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("path must be in format 'METHOD /path', got: %s", value)
	}

	method := strings.ToUpper(strings.TrimSpace(parts[0]))
	path := strings.TrimSpace(parts[1])

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true,
		"PATCH": true, "OPTIONS": true, "HEAD": true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	// Validate path starts with /
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /, got: %s", path)
	}

	handler.Route = Route{
		Method: method,
		Path:   path,
	}

	return nil
}

// parseAuth parses @wylla:auth required|optional|none
func (p *Parser) parseAuth(handler *Handler, value string) error {
	value = strings.ToLower(strings.TrimSpace(value))

	switch value {
	case "required":
		handler.Auth.Type = AuthRequired
	case "optional":
		handler.Auth.Type = AuthOptional
	case "none":
		handler.Auth.Type = AuthNone
	default:
		return fmt.Errorf("auth must be 'required', 'optional', or 'none', got: %s", value)
	}

	return nil
}

// parseRateLimit parses @wylla:ratelimit 100/hour
func (p *Parser) parseRateLimit(handler *Handler, value string) error {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return fmt.Errorf("ratelimit must be in format 'count/period', got: %s", value)
	}

	var count int
	if _, err := fmt.Sscanf(parts[0], "%d", &count); err != nil {
		return fmt.Errorf("invalid count in ratelimit: %s", parts[0])
	}

	period := strings.ToLower(strings.TrimSpace(parts[1]))
	var duration int64

	switch period {
	case "second", "sec", "s":
		duration = 1
	case "minute", "min", "m":
		duration = 60
	case "hour", "hr", "h":
		duration = 3600
	case "day", "d":
		duration = 86400
	default:
		return fmt.Errorf("invalid period in ratelimit: %s (use second/minute/hour/day)", period)
	}

	handler.RateLimit = &RateLimitConfig{
		Count:  count,
		Period: parseDuration(duration),
		Raw:    value,
	}

	return nil
}

// parseCORS parses @wylla:cors origins=*
func (p *Parser) parseCORS(handler *Handler, value string) error {
	if !strings.HasPrefix(value, "origins=") {
		return fmt.Errorf("cors must be in format 'origins=*' or 'origins=url1,url2', got: %s", value)
	}

	originsStr := strings.TrimPrefix(value, "origins=")
	var origins []string

	if originsStr == "*" {
		origins = []string{"*"}
	} else {
		origins = strings.Split(originsStr, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
	}

	handler.CORS = &CORSConfig{
		AllowedOrigins: origins,
		Raw:            value,
	}

	return nil
}

// parseTimeout parses @wylla:timeout 30s
func (p *Parser) parseTimeout(handler *Handler, value string) error {
	// Parse duration string (e.g., "30s", "5m", "1h")
	var num int
	var unit string

	if _, err := fmt.Sscanf(value, "%d%s", &num, &unit); err != nil {
		return fmt.Errorf("invalid timeout format: %s (use format like '30s', '5m', '1h')", value)
	}

	var multiplier int64
	switch unit {
	case "s", "sec", "second":
		multiplier = 1
	case "m", "min", "minute":
		multiplier = 60
	case "h", "hr", "hour":
		multiplier = 3600
	default:
		return fmt.Errorf("invalid timeout unit: %s (use s/m/h)", unit)
	}

	handler.Timeout = parseDuration(int64(num) * multiplier)
	return nil
}

// parseContainerService parses @wylla:container service=name
func (p *Parser) parseContainerService(handler *Handler, value string) error {
	if !strings.HasPrefix(value, "service=") {
		return fmt.Errorf("container parameter must be in format 'service=name', got: %s", value)
	}

	serviceName := strings.TrimPrefix(value, "service=")
	serviceName = strings.TrimSpace(serviceName)

	if serviceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	handler.ServiceName = serviceName
	return nil
}

// Helper function to create time.Duration from seconds
func parseDuration(seconds int64) time.Duration {
	return time.Duration(seconds * 1000000000) // Convert to nanoseconds
}
