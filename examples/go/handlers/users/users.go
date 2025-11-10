package users

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ListUsers returns a list of users
// @box:function
// @box:path GET /api/v1/users
// @box:auth required
// @box:ratelimit 100/minute
// @box:cors origins=*
func ListUsers(db *pgxpool.Pool, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Listing users")

		// Mock data for example
		users := []User{
			{
				ID:        "user-1",
				Email:     "alice@example.com",
				Name:      "Alice",
				CreatedAt: time.Now().Add(-24 * time.Hour),
			},
			{
				ID:        "user-2",
				Email:     "bob@example.com",
				Name:      "Bob",
				CreatedAt: time.Now().Add(-12 * time.Hour),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(users)
	}
}

// GetUser returns a single user by ID
// @box:function
// @box:path GET /api/v1/users/{id}
// @box:auth required
// @box:ratelimit 200/minute
// @box:cors origins=*
func GetUser(db *pgxpool.Pool, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		logger.Info("Getting user", zap.String("id", id))

		// Mock data for example
		user := User{
			ID:        id,
			Email:     "user@example.com",
			Name:      "Example User",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	}
}

// CreateUser creates a new user
// @box:function
// @box:path POST /api/v1/users
// @box:auth required
// @box:ratelimit 50/hour
// @box:cors origins=*
// @box:timeout 10s
// @box:memory 256MB
func CreateUser(db *pgxpool.Pool, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("Failed to decode request", zap.Error(err))
			http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
			return
		}

		logger.Info("Creating user", zap.String("email", req.Email))

		// Mock creation for example
		user := User{
			ID:        "user-new",
			Email:     req.Email,
			Name:      req.Name,
			CreatedAt: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(user)
	}
}

// StreamUserEvents streams user events via Server-Sent Events
// @box:container
// @box:path GET /api/v1/users/{id}/events
// @box:auth required
// @box:timeout 5m
// @box:concurrency 100
// @box:cors origins=*
func StreamUserEvents(db *pgxpool.Pool, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		logger.Info("Streaming events for user", zap.String("id", id))

		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Flush headers immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Stream events for demonstration
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for i := 0; i < 5; i++ {
			select {
			case <-r.Context().Done():
				logger.Info("Client disconnected")
				return
			case <-ticker.C:
				event := map[string]interface{}{
					"event":     "user_activity",
					"user_id":   id,
					"timestamp": time.Now(),
					"data":      "User performed action",
				}

				data, _ := json.Marshal(event)
				_, err := w.Write([]byte("data: " + string(data) + "\n\n"))
				if err != nil {
					logger.Error("Failed to write event", zap.Error(err))
					return
				}

				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}

				logger.Info("Sent event", zap.Int("count", i+1))
			}
		}

		logger.Info("Event stream completed")
	}
}
