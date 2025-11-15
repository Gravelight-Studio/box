package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gravelight-studio/box/go/annotations"
)

// Parser parses TypeScript/JavaScript files for Box annotations
type Parser struct{}

// NewParser creates a new TypeScript parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseDirectory recursively parses all TypeScript/JavaScript files in a directory
func (p *Parser) ParseDirectory(dir string) (*annotations.ParsedAnnotations, error) {
	result := &annotations.ParsedAnnotations{
		Handlers: []annotations.Handler{},
		Errors:   []annotations.ParseError{},
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip node_modules and hidden directories
			if info.Name() == "node_modules" || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only parse .ts and .js files (excluding test and type definition files)
		if (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".js")) &&
			!strings.HasSuffix(path, ".test.ts") &&
			!strings.HasSuffix(path, ".test.js") &&
			!strings.HasSuffix(path, ".d.ts") {

			fileResult, err := p.ParseFile(path)
			if err != nil {
				result.Errors = append(result.Errors, annotations.ParseError{
					FilePath:   path,
					LineNumber: 0,
					Message:    fmt.Sprintf("Failed to parse file: %v", err),
				})
				return nil
			}

			result.Handlers = append(result.Handlers, fileResult.Handlers...)
			result.Errors = append(result.Errors, fileResult.Errors...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ParseFile parses a single TypeScript/JavaScript file
func (p *Parser) ParseFile(filePath string) (*annotations.ParsedAnnotations, error) {
	result := &annotations.ParsedAnnotations{
		Handlers: []annotations.Handler{},
		Errors:   []annotations.ParseError{},
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")

	// Pattern to match function declarations and const assignments
	// Handles: function foo(), export function foo(), const foo = ..., export const foo: Type = ...
	functionPattern := regexp.MustCompile(`(?:export\s+)?(?:async\s+)?function\s+(\w+)|(?:export\s+)?const\s+(\w+)\s*(?::\s*\w+\s*)?=`)

	for i, line := range lines {
		matches := functionPattern.FindStringSubmatch(line)
		if matches != nil {
			functionName := matches[1]
			if functionName == "" {
				functionName = matches[2]
			}

			// Look backwards for annotations
			annotationData := p.extractAnnotationsAbove(lines, i)

			// Only create handler if it has a deployment type
			if annotationData["deploymentType"] != "" {
				handler := p.buildHandler(functionName, filePath, annotationData, i+1)
				if handler != nil {
					result.Handlers = append(result.Handlers, *handler)
				}
			}
		}
	}

	return result, nil
}

// extractAnnotationsAbove looks backwards from a function declaration for @box: annotations
func (p *Parser) extractAnnotationsAbove(lines []string, functionLineIndex int) map[string]string {
	annotations := make(map[string]string)

	// Look backwards from function line
	for i := functionLineIndex - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Stop at empty lines or non-comment lines
		if line == "" || (!strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "*") && !strings.HasPrefix(line, "/*")) {
			break
		}

		// Extract annotation
		annotationPattern := regexp.MustCompile(`@box:(\w+)\s*(.*)`)
		matches := annotationPattern.FindStringSubmatch(line)
		if matches != nil {
			key := matches[1]
			value := strings.TrimSpace(matches[2])
			p.parseAnnotation(key, value, annotations)
		}
	}

	return annotations
}

// parseAnnotation parses a single annotation key-value pair
func (p *Parser) parseAnnotation(key, value string, annotations map[string]string) {
	switch key {
	case "function":
		annotations["deploymentType"] = "function"

	case "container":
		annotations["deploymentType"] = "container"

	case "service":
		annotations["service"] = value

	case "path":
		pathPattern := regexp.MustCompile(`(\w+)\s+(.+)`)
		matches := pathPattern.FindStringSubmatch(value)
		if matches != nil {
			annotations["method"] = strings.ToUpper(matches[1])
			annotations["path"] = matches[2]
		}

	case "auth":
		annotations["auth"] = value

	case "cors":
		if strings.Contains(value, "origins=") {
			originsPattern := regexp.MustCompile(`origins=(.+)`)
			matches := originsPattern.FindStringSubmatch(value)
			if matches != nil {
				annotations["corsOrigins"] = matches[1]
			}
		}

	case "ratelimit":
		rateLimitPattern := regexp.MustCompile(`(\d+)\s*(?:requests?)?/?(second|minute|hour|day)`)
		matches := rateLimitPattern.FindStringSubmatch(value)
		if matches != nil {
			annotations["rateLimitRequests"] = matches[1]
			annotations["rateLimitPeriod"] = matches[2]
		}

	case "timeout":
		timeoutPattern := regexp.MustCompile(`(\d+)(s|m|h)`)
		matches := timeoutPattern.FindStringSubmatch(value)
		if matches != nil {
			duration, _ := strconv.Atoi(matches[1])
			unit := matches[2]
			ms := p.durationToMs(duration, unit)
			annotations["timeout"] = fmt.Sprintf("%d", ms)
		}

	case "memory":
		memoryPattern := regexp.MustCompile(`(\d+)MB`)
		matches := memoryPattern.FindStringSubmatch(value)
		if matches != nil {
			annotations["memory"] = matches[1]
		}

	case "concurrency":
		annotations["concurrency"] = value
	}
}

// buildHandler constructs a Handler from parsed annotations
func (p *Parser) buildHandler(functionName, filePath string, annotationData map[string]string, lineNumber int) *annotations.Handler {
	// Must have a path
	if annotationData["path"] == "" {
		return nil
	}

	// Parse deployment type
	var deploymentType annotations.DeploymentType
	switch annotationData["deploymentType"] {
	case "function":
		deploymentType = annotations.DeploymentFunction
	case "container":
		deploymentType = annotations.DeploymentContainer
	default:
		return nil
	}

	// Parse auth type
	authType := annotations.AuthNone
	switch annotationData["auth"] {
	case "optional":
		authType = annotations.AuthOptional
	case "required":
		authType = annotations.AuthRequired
	}

	// Build handler
	handler := &annotations.Handler{
		PackageName:    filepath.Base(filepath.Dir(filePath)),
		FunctionName:   functionName,
		FilePath:       filePath,
		LineNumber:     lineNumber,
		DeploymentType: deploymentType,
		Route: annotations.Route{
			Method: annotationData["method"],
			Path:   annotationData["path"],
		},
		Auth: annotations.AuthConfig{
			Type: authType,
		},
	}

	// Add service name for containers
	if serviceName := annotationData["service"]; serviceName != "" {
		handler.ServiceName = serviceName
	}

	// Add CORS if specified
	if corsOrigins := annotationData["corsOrigins"]; corsOrigins != "" {
		origins := strings.Split(corsOrigins, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		handler.CORS = &annotations.CORSConfig{
			AllowedOrigins: origins,
			Raw:            corsOrigins,
		}
	}

	// Add rate limit if specified
	if requests := annotationData["rateLimitRequests"]; requests != "" {
		reqCount, _ := strconv.Atoi(requests)
		period := annotationData["rateLimitPeriod"]
		periodDuration := p.periodToDuration(period)

		handler.RateLimit = &annotations.RateLimitConfig{
			Count:  reqCount,
			Period: periodDuration,
			Raw:    fmt.Sprintf("%s/%s", requests, period),
		}
	}

	// Add timeout if specified
	if timeout := annotationData["timeout"]; timeout != "" {
		timeoutMs, _ := strconv.Atoi(timeout)
		handler.Timeout = time.Duration(timeoutMs) * time.Millisecond
	}

	// Add memory if specified
	if memory := annotationData["memory"]; memory != "" {
		handler.Memory = memory + "MB"
	}

	// Add concurrency if specified
	if concurrency := annotationData["concurrency"]; concurrency != "" {
		maxConcurrency, _ := strconv.Atoi(concurrency)
		handler.Concurrency = maxConcurrency
	}

	return handler
}

// periodToDuration converts a period string to time.Duration
func (p *Parser) periodToDuration(period string) time.Duration {
	switch period {
	case "second":
		return time.Second
	case "minute":
		return time.Minute
	case "hour":
		return time.Hour
	case "day":
		return 24 * time.Hour
	default:
		return time.Second
	}
}

// durationToMs converts a duration value and unit to milliseconds
func (p *Parser) durationToMs(value int, unit string) int {
	switch unit {
	case "s":
		return value * 1000
	case "m":
		return value * 60 * 1000
	case "h":
		return value * 60 * 60 * 1000
	default:
		return value * 1000
	}
}
