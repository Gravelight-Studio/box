# Box Example Application

This example demonstrates how to use the **Box** framework to build an annotation-driven API with hybrid deployment support.

## What This Example Shows

- ✅ Using `@box:function` annotations for serverless functions
- ✅ Using `@box:container` annotations for long-running containers
- ✅ Automatic middleware from annotations (auth, CORS, rate limiting)
- ✅ Dynamic route registration from annotations
- ✅ Handler registry pattern
- ✅ Server-Sent Events (SSE) for streaming

## Project Structure

```
examples/go/
├── main.go                    # Entry point with Box router setup
├── go.mod                     # Module with local Box dependency
├── handlers/
│   ├── health/
│   │   └── health.go         # Health check endpoint (@box:function)
│   └── users/
│       └── users.go          # User CRUD + streaming (@box:function + @box:container)
└── README.md                  # This file
```

## Handlers Overview

### Function Deployment (Serverless)

These handlers are marked with `@box:function` and are suitable for short-lived, low-traffic endpoints:

- **GET /health** - Health check (no auth)
- **GET /api/v1/users** - List users (requires auth, rate limited 100/min)
- **GET /api/v1/users/{id}** - Get single user (requires auth, rate limited 200/min)
- **POST /api/v1/users** - Create user (requires auth, rate limited 50/hour, 256MB memory)

### Container Deployment (Always-On)

This handler is marked with `@box:container` for long-running connections:

- **GET /api/v1/users/{id}/events** - Stream user events via SSE (requires auth, 5min timeout, 100 concurrency)

## Running the Example

### 1. Install Dependencies

```bash
go mod download
```

### 2. Run the Server

```bash
go run main.go
```

The server starts on `http://localhost:8080`

### 3. Test the Endpoints

**Health Check (No Auth):**
```bash
curl http://localhost:8080/health
```

**List Users (Requires Auth):**
```bash
curl -H "Authorization: Bearer test-token" \
     http://localhost:8080/api/v1/users
```

**Get Single User (Requires Auth):**
```bash
curl -H "Authorization: Bearer test-token" \
     http://localhost:8080/api/v1/users/user-1
```

**Create User (Requires Auth):**
```bash
curl -X POST \
     -H "Authorization: Bearer test-token" \
     -H "Content-Type: application/json" \
     -d '{"email":"new@example.com","name":"New User"}' \
     http://localhost:8080/api/v1/users
```

**Stream Events (Requires Auth, Container Deployment):**
```bash
curl -H "Authorization: Bearer test-token" \
     http://localhost:8080/api/v1/users/123/events
```

This will stream 5 events over 10 seconds using Server-Sent Events.

## How Box Works

### 1. Annotation Parsing

When you create the router, Box automatically scans your handlers directory:

```go
r, err := router.New(router.Config{
    HandlersDir: "./handlers",
    DB:          nil,
    Logger:      logger,
})
```

Box parses all `@box:` annotations and builds a routing table.

### 2. Handler Registration

You register your handler functions with the registry:

```go
registry := router.NewHandlerRegistry(nil, logger)
registry.Register("users", "ListUsers", users.ListUsers)
registry.Register("users", "GetUser", users.GetUser)
// ...
```

### 3. Automatic Middleware

Box applies middleware based on annotations:

- `@box:auth required` → Validates Bearer token
- `@box:cors origins=*` → Adds CORS headers
- `@box:ratelimit 100/minute` → Rate limits requests
- `@box:timeout 10s` → Times out slow requests

### 4. Route Registration

Finally, register handlers with the router:

```go
err = r.RegisterHandlers(registry)
```

Box automatically:
- Creates routes based on `@box:path` annotations
- Applies middleware based on other annotations
- Logs all registered routes

## Annotations Used

### Deployment Annotations

```go
// @box:function      - Deploy as Cloud Function (serverless)
// @box:container     - Deploy as Cloud Run (always-on)
```

### Routing Annotations

```go
// @box:path GET /api/v1/users
// @box:path POST /api/v1/users
// @box:path GET /api/v1/users/{id}
```

### Middleware Annotations

```go
// @box:auth required      - Require Bearer token
// @box:auth optional      - Accept token if present
// @box:auth none          - No authentication
// @box:cors origins=*     - Allow all origins
// @box:ratelimit 100/min  - Rate limit to 100 requests/minute
// @box:timeout 10s        - Timeout after 10 seconds
```

### Resource Annotations

```go
// @box:memory 256MB       - Allocate 256MB (for functions)
// @box:concurrency 100    - Max 100 concurrent requests (for containers)
```

## Building Deployment Artifacts

The Box CLI generates deployment artifacts (Cloud Functions, Cloud Run containers, API Gateway specs, and Terraform IaC) for GCP.

### Install the Box CLI

**Option 1: Go Install (if you have Go installed)**
```bash
go install github.com/gravelight-studio/box/cmd/box@latest
```

**Option 2: Download Binary (coming soon)**
```bash
curl -sSL https://raw.githubusercontent.com/gravelight-studio/box/main/install.sh | sh
```

### Generate Deployment Artifacts

From your project root:

```bash
box --handlers ./handlers \
    --output ./build \
    --project my-gcp-project \
    --region us-central1
```

**Options:**
- `--handlers` - Path to your handlers directory (default: `./handlers`)
- `--output` - Output directory for generated files (default: `./build`)
- `--project` - GCP project ID (required)
- `--region` - GCP region (default: `us-central1`)
- `--env` - Environment name: dev, staging, production (default: `dev`)
- `--clean` - Clean build directory before generating
- `--verbose` - Enable verbose logging

### Generated Files

```
build/
├── functions/          # Cloud Function packages (serverless)
│   ├── list-users/
│   ├── get-user/
│   └── create-user/
├── containers/         # Cloud Run Dockerfiles (always-on)
│   └── stream-user-events/
├── gateway/           # API Gateway OpenAPI specs
│   └── openapi.yaml
└── terraform/         # Infrastructure as Code
    ├── main.tf
    ├── functions.tf
    ├── containers.tf
    └── gateway.tf
```

### Deploy to GCP

```bash
cd build/terraform
terraform init
terraform plan
terraform apply
```

## Next Steps

1. **Add Database**: Initialize a PostgreSQL connection and pass it to the router
2. **Add Real Auth**: Implement JWT validation in the auth middleware
3. **Add More Handlers**: Create additional endpoints with different annotations
4. **Deploy to GCP**: Use the build system to generate deployment artifacts
5. **Add Tests**: Write tests for your handlers

## Learn More

- [Box Library Documentation](../../go/README.md)
- [Annotation Reference](../../go/README.md#annotation-reference)
- [GCP Cloud Functions](https://cloud.google.com/functions)
- [GCP Cloud Run](https://cloud.google.com/run)
