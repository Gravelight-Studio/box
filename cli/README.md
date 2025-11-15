# Box CLI

Universal command-line interface for the Box serverless framework. Scaffold and build applications in multiple languages with a single tool.

## Features

- **Multi-language support**: Create projects in Go or TypeScript
- **Interactive project initialization**: Guided setup with prompts
- **Auto-detection**: Automatically detects project language for builds
- **Single binary**: No runtime dependencies, just one executable
- **Embedded templates**: Project templates are built into the CLI

## Installation

### Quick Install

#### Linux / macOS
```bash
curl -sSL https://raw.githubusercontent.com/gravelight-studio/box/main/install.sh | sh
```

#### Windows (PowerShell)
```powershell
iwr -useb https://raw.githubusercontent.com/gravelight-studio/box/main/install.ps1 | iex
```

### From Source

```bash
cd cli
go build -o bin/box ./cmd/box

# Add to PATH (optional)
export PATH=$PATH:$(pwd)/bin
```

### Binary Releases

Download pre-built binaries from [GitHub Releases](https://github.com/gravelight-studio/box/releases).

## Commands

### `box init` - Initialize a new project

Create a new Box project with an interactive wizard:

```bash
box init my-app
```

Or specify options directly:

```bash
# Create a Go project
box init my-go-api --lang go

# Create a TypeScript project
box init my-ts-api --lang typescript --path ./projects/api
```

**Options:**
- `--lang <language>` - Project language: `go` or `typescript`
- `--path <path>` - Custom project path (default: `./<project-name>`)

**What it creates:**

For **Go** projects:
- `go.mod` - Go module file
- `main.go` - Local development server
- `handlers/health.go` - Example health check handlers
- `README.md` - Project documentation
- `.gitignore` - Git ignore file

For **TypeScript** projects:
- `package.json` - NPM package file
- `tsconfig.json` - TypeScript configuration
- `src/index.ts` - Local development server
- `handlers/health.ts` - Example health check handlers
- `README.md` - Project documentation
- `.gitignore` - Git ignore file

### `box build` - Build deployment artifacts

Generate deployment artifacts for Google Cloud Platform:

```bash
# Must be run from project directory
cd my-app
box build --project my-gcp-project-id
```

**Options:**
- `--project <id>` - GCP project ID (required)
- `--handlers <path>` - Path to handlers directory (default: `./handlers`)
- `--output <path>` - Output directory (default: `./build`)
- `--region <region>` - GCP region (default: `us-central1`)
- `--env <environment>` - Environment name (default: `dev`)
- `--module <name>` - Module name (auto-detected from go.mod/package.json)
- `--clean` - Clean build directory before generating
- `--verbose` - Enable verbose logging

**Language Detection:**

The CLI automatically detects your project language:
- Looks for `go.mod` → Go project
- Looks for `package.json` → TypeScript project

**What it generates:**

```
build/
├── functions/          # Cloud Functions (one per @box:function)
│   ├── get-health/
│   │   ├── function.go or index.js
│   │   ├── go.mod or package.json
│   │   └── function.yaml
│   └── ...
├── containers/         # Cloud Run containers (grouped by @box:service)
│   ├── api/
│   │   ├── Dockerfile
│   │   ├── server.go or server.js
│   │   ├── go.mod or package.json
│   │   └── cloudbuild.yaml
│   └── ...
├── gateway/            # API Gateway
│   └── openapi.yaml
└── terraform/          # Infrastructure as Code
    ├── main.tf
    ├── variables.tf
    └── outputs.tf
```

### `box version` - Show version

```bash
box version
```

## Quick Start

### Create and run a Go project:

```bash
# 1. Create project
box init my-go-api --lang go
cd my-go-api

# 2. Install dependencies
go mod tidy

# 3. Run locally
go run .

# 4. Generate deployment artifacts
box build --project my-gcp-project

# 5. Deploy to GCP
cd build/terraform
terraform init
terraform apply
```

### Create and run a TypeScript project:

```bash
# 1. Create project
box init my-ts-api --lang typescript
cd my-ts-api

# 2. Install dependencies
npm install

# 3. Run locally
npm run dev

# 4. Generate deployment artifacts
box build --project my-gcp-project

# 5. Deploy to GCP
cd build/terraform
terraform init
terraform apply
```

## Project Structure

Both Go and TypeScript projects follow the same structure:

```
my-app/
├── handlers/           # HTTP handlers with @box: annotations
│   └── health.{go,ts}  # Example handlers
├── main.{go,ts}        # Local dev server (in src/ for TS)
├── go.mod              # Go: Module file
├── package.json        # TypeScript: Package file
└── README.md           # Documentation
```

## Handler Annotations

Box uses comment annotations to configure deployment:

```go
// Go example
// @box:function
// @box:path GET /api/users
// @box:auth required
// @box:ratelimit 100 requests/second
func GetUsers(db router.DB, logger *zap.Logger) http.HandlerFunc {
    // ...
}
```

```typescript
// TypeScript example
// @box:container
// @box:service api
// @box:path POST /api/users
// @box:auth required
// @box:memory 512MB
export const createUser: HandlerFactory = (db, logger) => {
    // ...
};
```

### Available Annotations

| Annotation | Values | Description |
|------------|--------|-------------|
| `@box:function` | - | Deploy as Cloud Function |
| `@box:container` | - | Deploy as Cloud Run container |
| `@box:service` | name | Service name (for grouping containers) |
| `@box:path` | METHOD /path | HTTP route (GET, POST, PUT, DELETE, etc.) |
| `@box:auth` | none \| optional \| required | Authentication mode |
| `@box:cors` | enabled \| disabled | CORS support |
| `@box:ratelimit` | N requests/second | Rate limiting |
| `@box:timeout` | Ns \| Nm | Request timeout |
| `@box:memory` | NMB \| NGB | Memory allocation |

## Architecture

The unified CLI is built in Go and uses:

- **`embed.FS`** - Embeds project templates in the binary
- **`promptui`** - Interactive command-line prompts
- **Template engine** - Go's `text/template` for scaffolding
- **Language delegation** - Calls language-specific build tools
  - Go: Direct integration with Box Go library
  - TypeScript: Shells out to Node.js to run Box TypeScript CLI

## Development

### Project Structure

```
cli/
├── cmd/
│   └── box/
│       ├── main.go         # CLI entry point
│       └── templates/      # Embedded project templates
│           ├── go/
│           └── typescript/
├── go.mod
└── README.md
```

### Building

```bash
go build -o bin/box ./cmd/box
```

### Testing

```bash
# Test Go project scaffolding
./bin/box init test-go --lang go
cd test-go && go mod tidy && go run .

# Test TypeScript project scaffolding
./bin/box init test-ts --lang typescript
cd test-ts && npm install && npm run dev

# Test build command
./bin/box build --project test-project-id
```

## Why Go for the CLI?

1. **Single binary distribution** - No runtime dependencies
2. **Cross-platform** - Compile for any OS
3. **Fast startup** - Native performance
4. **Embedded assets** - Templates bundled in binary
5. **Small footprint** - ~10MB binary includes everything

## Learn More

- [Box Documentation](https://github.com/gravelight-studio/box)
- [Box Go Library](../go/)
- [Box TypeScript Library](../typescript/)

## License

MIT
