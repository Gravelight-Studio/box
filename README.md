# Box

Box is an annotation-driven framework for building APIs that deploy seamlessly to hybrid cloud infrastructure. Write your handlers once with simple comment annotations, and Box automatically generates the deployment artifacts, OpenAPI specs, and infrastructure-as-code for Google Cloud Platform.

The framework supports both serverless (Cloud Functions) and container (Cloud Run) deployments from the same codebase. Use `@box:function` for short-lived, stateless endpoints and `@box:container` for long-running connections like WebSockets or Server-Sent Events. Box handles routing, middleware (CORS, auth, rate limiting), and deployment configuration through annotations.

Currently supports Go, with Python, Node.js, and other languages planned. Each language implementation provides the same annotation syntax and generates compatible deployment artifacts, making it easy to build polyglot microservices.

## Installation

### Box CLI

**Option 1: Install Script**
```bash
curl -sSL https://raw.githubusercontent.com/gravelight-studio/box/main/install.sh | sh
```

**Option 2: Go Install**
```bash
go install github.com/gravelight-studio/box/cmd/box@latest
```

### Go Library

```bash
go get github.com/gravelight-studio/box
```

## Quick Example

```go
// @box:function
// @box:path POST /api/users
// @box:auth required
// @box:ratelimit 100/hour
func CreateUser(db *pgxpool.Pool, logger *zap.Logger) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Your handler logic
    }
}
```

Generate deployment artifacts:

```bash
box --handlers ./handlers --project my-gcp-project
```

## Documentation

- **[Go Library Documentation](go/README.md)** - Full API reference and annotation guide
- **[Example Application](examples/go/)** - Complete working example with multiple deployment types
- **[Installation Methods](TODO/Install%20Methods.md)** - Planned distribution methods (NPM, Homebrew)

## Repository Structure

```
Box/
├── go/                     # Go implementation
│   ├── annotations/        # Annotation parser
│   ├── router/            # HTTP router with middleware
│   ├── build/             # Deployment artifact generator
│   └── cmd/box/           # CLI tool
├── examples/go/           # Example application
├── install.sh             # Cross-platform install script
└── .github/workflows/     # CI/CD for releases
```

## Contributors

Thanks to all contributors who have helped make Box better!

<a href="https://github.com/gravelight-studio/box/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=gravelight-studio/box" />
</a>

## Funding

Box is open source and free to use. If you find it valuable, consider supporting its development:

- **[GitHub Sponsors](https://github.com/sponsors/gravelight-studio)** - Sponsor ongoing development
- **[Open Collective](https://opencollective.com/box)** - One-time or recurring contributions

Your support helps maintain the project, add new language implementations, and improve documentation.

## License

MIT
