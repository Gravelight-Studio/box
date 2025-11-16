# Box

**Box** is a Go framework for building annotation-driven APIs that deploy seamlessly to GCP Cloud Functions and Cloud Run. Write handlers once, deploy anywhere.

## Features

- 🏷️ **Annotation-Driven** - Configure deployment, routing, and middleware with simple comments
- 🚀 **Hybrid Deployment** - Deploy handlers as serverless functions or containers based on annotations
- 🔄 **Build Automation** - Automatically generate deployment artifacts, OpenAPI specs, and Terraform configs
- 🎯 **Type-Safe** - Full Go type safety with compile-time checks
- 🔐 **Built-in Middleware** - CORS, authentication, rate limiting, and timeouts out of the box
- 📊 **Production Ready** - Battle-tested with comprehensive test coverage

## Installation

```bash
go get github.com/gravelight-studio/box
```

## Quick Start

### 1. Write an Annotated Handler

```go
package handlers

import (
    "encoding/json"
    "net/http"
)

// @box:function
// @box:path POST /api/v1/users
// @box:auth required
// @box:ratelimit 100/hour
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    json.NewDecoder(r.Body).Decode(&user)

    // Your business logic here

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(user)
}
```

### 2. Parse Annotations and Build Routes

```go
package main

import (
    "github.com/gravelight-studio/box/go/annotations"
    "github.com/gravelight-studio/box/go/router"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()

    // Create annotation-driven router
    r, err := router.New(router.Config{
        HandlersDir: "./handlers",
        DB:          dbPool,
        Logger:      logger,
    })

    // Register your handlers
    registry := router.NewHandlerRegistry(dbPool, logger)
    registry.Register("handlers", "CreateUser", handlers.CreateUser)
    r.RegisterHandlers(registry)

    // Start server
    http.ListenAndServe(":8080", r)
}
```

### 3. Generate Deployment Artifacts

Install the Box CLI:

```bash
# Option 1: Using Go
go install github.com/gravelight-studio/box/cmd/box@latest

# Option 2: Download binary (coming soon)
curl -sSL https://raw.githubusercontent.com/gravelight-studio/box/main/install.sh | sh
```

Generate deployment artifacts for GCP:

```bash
box --handlers ./handlers \
    --output ./build \
    --project my-gcp-project \
    --region us-central1 \
    --env production
```

This generates:
- `build/functions/` - Cloud Function packages for serverless handlers
- `build/containers/` - Cloud Run Dockerfiles for container handlers
- `build/gateway/` - OpenAPI 3.0 specs for API Gateway
- `build/terraform/` - Infrastructure as Code (Terraform)

Deploy to GCP:

```bash
cd build/terraform
terraform init
terraform apply
```

> **Advanced:** You can also use the programmatic API directly. See [Build Package API](#build-package) for details.

## Core Concepts

### Annotations

Box uses special comments to configure handlers. All annotations start with `@box:`.

#### Deployment Type

Choose where your handler deploys:

```go
// @box:function    - Deploy as GCP Cloud Function (serverless)
// @box:container   - Deploy as GCP Cloud Run (always-on container)
```

**When to use `@box:function`:**
- Low or sporadic traffic
- Fast execution (<60s)
- Cost optimization (scales to zero)
- Stateless operations

**When to use `@box:container`:**
- High or consistent traffic
- Long-running operations (>60s)
- WebSocket/SSE connections
- Persistent state needed

#### Routing

Define HTTP routes:

```go
// @box:path GET /api/v1/users
// @box:path POST /api/v1/users
// @box:path GET /api/v1/users/{id}
// @box:path DELETE /api/v1/users/{id}
```

Supported methods: `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `OPTIONS`, `HEAD`

#### Authentication

Configure authentication requirements:

```go
// @box:auth required   - Reject requests without valid Bearer token
// @box:auth optional   - Accept token if present, continue if not
// @box:auth none       - No authentication (default)
```

#### Rate Limiting

Limit request rates:

```go
// @box:ratelimit 100/hour
// @box:ratelimit 1000/minute
// @box:ratelimit 10/second
// @box:ratelimit 50000/day
```

#### CORS

Configure cross-origin resource sharing:

```go
// @box:cors origins=*                           - Allow all origins
// @box:cors origins=https://example.com         - Single origin
// @box:cors origins=https://a.com,https://b.com - Multiple origins
```

#### Timeouts

Set request timeouts:

```go
// @box:timeout 30s    - 30 seconds
// @box:timeout 5m     - 5 minutes
// @box:timeout 1h     - 1 hour
```

#### Resource Configuration

**Cloud Functions:**
```go
// @box:memory 128MB
// @box:memory 256MB
// @box:memory 512MB
// @box:memory 1GB
// @box:memory 2GB
```

**Cloud Run:**
```go
// @box:concurrency 80    - Max concurrent requests per instance
// @box:concurrency 1000
```

## Package Reference

### `annotations`

Parse and validate handler annotations.

```go
import "github.com/gravelight-studio/box/go/annotations"

// Create parser
parser := annotations.NewParser()

// Parse single file
result, err := parser.ParseFile("./handlers/users.go")

// Parse directory recursively
result, err := parser.ParseDirectory("./handlers")

// Validate handlers
validator := annotations.NewValidator()
errors := validator.Validate(result.Handlers)
pathErrors := validator.ValidateUniquePaths(result.Handlers)
```

**Key Types:**

```go
type Handler struct {
    FunctionName   string
    PackageName    string
    PackagePath    string
    DeploymentType DeploymentType
    Route          Route
    Auth           AuthConfig
    RateLimit      *RateLimitConfig
    CORS           *CORSConfig
    Timeout        time.Duration
    Memory         string
    Concurrency    int
}
```

### `router`

Create annotation-driven HTTP routers.

```go
import "github.com/gravelight-studio/box/go/router"

// Create router
r, err := router.New(router.Config{
    HandlersDir: "./internal/handlers",
    DB:          pgxPool,
    Logger:      zapLogger,
})

// Create handler registry
registry := router.NewHandlerRegistry(pgxPool, zapLogger)

// Register handlers
registry.Register("users", "CreateUser", users.CreateUser)
registry.Register("users", "GetUser", users.GetUser)
registry.Register("accounts", "CreateAccount", accounts.CreateAccount)

// Register all handlers with router
r.RegisterHandlers(registry)

// Use as http.Handler
http.ListenAndServe(":8080", r)
```

**Middleware:**

Middleware is automatically applied based on annotations:
- **CORS** - Applied when `@box:cors` is present
- **Auth** - Applied when `@box:auth required|optional`
- **RateLimit** - Applied when `@box:ratelimit` is present
- **Timeout** - Applied when `@box:timeout` is present

### `build`

Generate deployment artifacts.

```go
import "github.com/gravelight-studio/box/go/build"

gen := build.NewGenerator(build.Config{
    Handlers:      handlers,
    ModuleName:    "github.com/mycompany/myapi",
    OutputDir:     "./build",
    ProjectID:     "my-gcp-project",
    Region:        "us-central1",
    Environment:   "production",
    Logger:        logger,
    CleanBuildDir: true,
})

// Generate everything
gen.Generate()

// Or generate selectively
gen.GenerateFunctions()
gen.GenerateContainers()
gen.GenerateGateway()
gen.GenerateTerraform()
```

## Complete Example

```go
package handlers

import (
    "encoding/json"
    "net/http"
)

// CreateAccount handles account creation
// @box:function
// @box:path POST /api/v1/accounts
// @box:auth required
// @box:ratelimit 10/minute
// @box:cors origins=*
// @box:timeout 30s
// @box:memory 256MB
func CreateAccount(w http.ResponseWriter, r *http.Request) {
    var req CreateAccountRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Create account logic...
    account := Account{
        ID:    generateID(),
        Email: req.Email,
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(account)
}

// StreamChat handles real-time chat streaming
// @box:container
// @box:path GET /api/v1/chat/{id}/stream
// @box:auth required
// @box:timeout 5m
// @box:concurrency 100
func StreamChat(w http.ResponseWriter, r *http.Request) {
    // SSE implementation for long-running connections
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Stream chat messages...
}
```

## Build Output Structure

When you run `gen.Generate()`, Box creates:

```
build/
├── functions/
│   ├── create-account/
│   │   ├── main.go           # Function entry point
│   │   ├── go.mod            # Standalone module
│   │   ├── function.yaml     # GCP config
│   │   └── deploy.sh         # Deployment script
│   └── ...
│
├── containers/
│   ├── chat-service/
│   │   ├── main.go           # Server with multiple handlers
│   │   ├── Dockerfile        # Multi-stage build
│   │   ├── cloudbuild.yaml   # CI/CD config
│   │   └── deploy.sh         # Deployment script
│   └── ...
│
├── gateway/
│   ├── openapi.yaml          # OpenAPI 3.0 spec
│   ├── gateway-config.yaml   # API Gateway config
│   └── deploy.sh             # Gateway deployment
│
└── terraform/
    ├── main.tf               # Root module
    ├── variables.tf          # Input variables
    ├── outputs.tf            # Outputs
    ├── modules/
    │   ├── cloud-functions/
    │   ├── cloud-run/
    │   ├── api-gateway/
    │   └── networking/
    └── environments/
        ├── dev.tfvars
        ├── staging.tfvars
        └── production.tfvars
```

## Deployment

### Deploy Cloud Functions

```bash
cd build/functions/create-account
./deploy.sh
```

### Deploy Cloud Run

```bash
cd build/containers/chat-service
./deploy.sh
```

### Deploy with Terraform

```bash
cd build/terraform
terraform init
terraform plan -var-file=environments/production.tfvars
terraform apply -var-file=environments/production.tfvars
```

## Architecture

### Local Development

In local development, all handlers run in a single server process:

```
┌─────────────────────────┐
│   Single Server         │
│   All Handlers          │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│   PostgreSQL            │
└─────────────────────────┘
```

### Production (GCP)

In production, handlers are deployed based on annotations:

```
        ┌────────────┐
        │ API Gateway│
        └──────┬─────┘
               │
       ┌───────┴───────┐
       │               │
       ▼               ▼
┌─────────────┐ ┌─────────────┐
│Cloud Function│ │  Cloud Run  │
│(Low Traffic) │ │(High Traffic)│
└──────┬──────┘ └──────┬──────┘
       │               │
       └───────┬───────┘
               ▼
       ┌──────────────┐
       │  Cloud SQL   │
       └──────────────┘
```

## Best Practices

### 1. Choose Deployment Type Wisely

```go
// ✅ Good: Lightweight CRUD operation
// @box:function
func CreateUser(w http.ResponseWriter, r *http.Request) { ... }

// ✅ Good: Long-running operation
// @box:container
func ProcessVideo(w http.ResponseWriter, r *http.Request) { ... }

// ❌ Bad: Heavy operation on function
// @box:function  // This will timeout!
func ProcessLargeFile(w http.ResponseWriter, r *http.Request) { ... }
```

### 2. Set Appropriate Rate Limits

```go
// ✅ Good: Protect expensive operations
// @box:ratelimit 10/minute
func GenerateReport(w http.ResponseWriter, r *http.Request) { ... }

// ✅ Good: Allow high throughput for reads
// @box:ratelimit 1000/minute
func GetUser(w http.ResponseWriter, r *http.Request) { ... }
```

### 3. Use Timeouts

```go
// ✅ Good: Fast operations get short timeouts
// @box:timeout 5s
func GetStatus(w http.ResponseWriter, r *http.Request) { ... }

// ✅ Good: Long operations get longer timeouts
// @box:timeout 5m
func ExportData(w http.ResponseWriter, r *http.Request) { ... }
```

### 4. Group Related Container Handlers

```go
// Group related handlers in the same container service
// @box:container service=chat
func CreateChat(w http.ResponseWriter, r *http.Request) { ... }

// @box:container service=chat
func StreamChat(w http.ResponseWriter, r *http.Request) { ... }
```

## Testing

All Box packages include comprehensive tests:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test github.com/gravelight-studio/box/go/annotations
```

## Contributing

Contributions are welcome! Please open an issue or submit a PR.

## License

MIT License - See LICENSE file for details

## Resources

- [GCP Cloud Functions Documentation](https://cloud.google.com/functions)
- [GCP Cloud Run Documentation](https://cloud.google.com/run)
- [Chi Router Documentation](https://go-chi.io/)
- [OpenAPI Specification](https://swagger.io/specification/)

---

**Built for hybrid cloud-native Go applications** 📦
