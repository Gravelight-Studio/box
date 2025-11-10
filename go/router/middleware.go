package router

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"github.com/gravelight-studio/box/annotations"
)

// CORSMiddleware creates CORS middleware from annotation config
func CORSMiddleware(config *annotations.CORSConfig) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	})
}

// AuthMiddleware creates authentication middleware
func AuthMiddleware(config annotations.AuthConfig, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Authorization header
			authHeader := r.Header.Get("Authorization")

			if config.Type == annotations.AuthRequired {
				// Auth required: reject if no valid token
				if authHeader == "" {
					logger.Warn("Missing authorization header", zap.String("path", r.URL.Path))
					http.Error(w, `{"error":"Authorization required"}`, http.StatusUnauthorized)
					return
				}

				// TODO: Validate Bearer token
				// For now, just check it starts with "Bearer "
				if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
					logger.Warn("Invalid authorization format", zap.String("path", r.URL.Path))
					http.Error(w, `{"error":"Invalid authorization format"}`, http.StatusUnauthorized)
					return
				}

				// TODO: Validate token with auth provider
				// For now, accept any Bearer token
				logger.Debug("Auth token present (validation stubbed)", zap.String("path", r.URL.Path))
			} else if config.Type == annotations.AuthOptional {
				// Auth optional: check if present, but don't reject if missing
				if authHeader != "" {
					logger.Debug("Optional auth token present", zap.String("path", r.URL.Path))
					// TODO: Validate token if present
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware(config *annotations.RateLimitConfig, logger *zap.Logger) func(http.Handler) http.Handler {
	// Create in-memory rate limiter
	limiter := NewInMemoryRateLimiter(config.Count, config.Period)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use IP address as key (in production, use user ID from auth)
			key := r.RemoteAddr

			allowed, remaining, resetTime := limiter.Allow(key)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Count))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				logger.Warn("Rate limit exceeded",
					zap.String("key", key),
					zap.String("path", r.URL.Path))

				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
				http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware creates timeout middleware
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			// Create a channel to signal completion
			done := make(chan struct{})

			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Request completed successfully
				return
			case <-ctx.Done():
				// Request timed out
				http.Error(w, `{"error":"Request timeout"}`, http.StatusGatewayTimeout)
				return
			}
		})
	}
}

// InMemoryRateLimiter implements a simple in-memory rate limiter
type InMemoryRateLimiter struct {
	mu      sync.RWMutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

type bucket struct {
	count     int
	resetTime time.Time
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(limit int, window time.Duration) *InMemoryRateLimiter {
	limiter := &InMemoryRateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request is allowed for the given key
func (l *InMemoryRateLimiter) Allow(key string) (allowed bool, remaining int, resetTime time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// Get or create bucket
	b, exists := l.buckets[key]
	if !exists || now.After(b.resetTime) {
		// Create new bucket
		b = &bucket{
			count:     0,
			resetTime: now.Add(l.window),
		}
		l.buckets[key] = b
	}

	// Check if under limit
	if b.count < l.limit {
		b.count++
		return true, l.limit - b.count, b.resetTime
	}

	return false, 0, b.resetTime
}

// cleanup removes expired buckets periodically
func (l *InMemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, b := range l.buckets {
			if now.After(b.resetTime) {
				delete(l.buckets, key)
			}
		}
		l.mu.Unlock()
	}
}
