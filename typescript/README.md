# Box - TypeScript/JavaScript

**Box** is a TypeScript/JavaScript framework for building annotation-driven APIs that deploy seamlessly to GCP Cloud Functions and Cloud Run.

## Installation

```bash
npm install @gravelight-studio/box
```

## Quick Start

### 1. Create an Annotated Handler

```typescript
import { Request, Response, HandlerFactory, Logger, Pool } from '@gravelight-studio/box';

// @box:function
// @box:path POST /api/v1/users
// @box:auth required
// @box:ratelimit 100/hour
export const createUser: HandlerFactory = (db: Pool | null, logger: Logger) => {
  return (req: Request, res: Response) => {
    const user = req.body;
    // Your business logic here
    res.status(201).json(user);
  };
};
```

> **Note:** All types (`Request`, `Response`, `Pool`, etc.) are re-exported from Box for convenience. You don't need to import from Express or pg directly.

### 2. Create Router and Register Handlers

```typescript
import { createRouter, HandlerRegistry } from '@gravelight-studio/box';
import pino from 'pino';
import * as handlers from './handlers';

async function main() {
  const logger = pino();

  // Create annotation-driven router
  const router = await createRouter({
    handlersDir: './src/handlers',
    db: null,
    logger
  });

  // Create registry and register handlers
  const registry = new HandlerRegistry(null, logger);
  registry.register('users', 'createUser', handlers.createUser);

  // Register with router
  router.registerHandlers(registry);

  // Start server
  const app = router.getApp();
  app.listen(8080, () => logger.info('Server started on :8080'));
}

main();
```

## Annotations Reference

### Deployment Type

```typescript
// @box:function      - Deploy as Cloud Function (serverless)
// @box:container     - Deploy as Cloud Run (always-on)
```

### Routing

```typescript
// @box:path GET /api/v1/users
// @box:path POST /api/v1/users
// @box:path GET /api/v1/users/:id
```

### Middleware

```typescript
// @box:auth none|optional|required
// @box:cors origins=*
// @box:ratelimit 100/minute
// @box:timeout 10s
```

### Resources

```typescript
// @box:memory 256MB        - Memory for functions
// @box:concurrency 100     - Max concurrent requests for containers
```

## Examples

See [examples/typescript](../examples/typescript/) for a complete working application.

## API Documentation

### HandlerFactory

A function that returns an Express request handler:

```typescript
type HandlerFactory = (db: Pool | null, logger: Logger) => RequestHandler;
```

### createRouter(config)

Creates a Box router that automatically parses annotations:

```typescript
interface RouterConfig {
  handlersDir: string;
  db?: Pool | null;
  logger: Logger;
}

const router = await createRouter(config);
```

### HandlerRegistry

Registry for storing handler implementations:

```typescript
const registry = new HandlerRegistry(db, logger);
registry.register(packageName, functionName, handlerFactory);
```

## Development

```bash
# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Lint
npm run lint

# Format
npm run format
```

## License

MIT
