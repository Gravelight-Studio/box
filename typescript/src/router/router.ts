import express, { Express, RequestHandler, Request as ExpressRequest, Response as ExpressResponse } from 'express';
import { Handler, Parser, AuthType } from '../annotations';
import {
  createCORSMiddleware,
  createAuthMiddleware,
  createRateLimitMiddleware,
  createTimeoutMiddleware,
  errorMiddleware,
  notFoundMiddleware
} from './middleware';

/**
 * HTTP Request type (re-exported from Express for convenience)
 */
export type Request = ExpressRequest;

/**
 * HTTP Response type (re-exported from Express for convenience)
 */
export type Response = ExpressResponse;

/**
 * Logger interface (compatible with pino)
 */
export interface Logger {
  info(msg: string, ...args: any[]): void;
  warn(msg: string, ...args: any[]): void;
  error(msg: string, ...args: any[]): void;
  debug(msg: string, ...args: any[]): void;
}

/**
 * Router configuration
 */
export interface RouterConfig {
  handlersDir: string;
  logger: Logger;
}

/**
 * Handler registry for storing handler implementations
 */
export class HandlerRegistry {
  private handlers: Map<string, RequestHandler> = new Map();

  constructor(private logger: Logger) {}

  /**
   * Register a handler implementation
   */
  register(packageName: string, functionName: string, handler: RequestHandler): void {
    const key = `${packageName}.${functionName}`;
    this.handlers.set(key, handler);
    this.logger.debug(`Registered handler: ${key}`);
  }

  /**
   * Get a handler implementation
   */
  get(packageName: string, functionName: string): RequestHandler | undefined {
    const key = `${packageName}.${functionName}`;
    return this.handlers.get(key);
  }
}

/**
 * Box router with annotation-driven route registration
 */
export class BoxRouter {
  private app: Express;
  private handlers: Handler[] = [];
  private config: RouterConfig;

  constructor(config: RouterConfig) {
    this.config = config;
    this.app = express();

    // Add body parsing middleware
    this.app.use(express.json());
    this.app.use(express.urlencoded({ extended: true }));
  }

  /**
   * Initialize router by parsing annotations
   */
  async initialize(): Promise<void> {
    const parser = new Parser();
    const result = await parser.parseDirectory(this.config.handlersDir);

    if (result.errors.length > 0) {
      this.config.logger.warn(`Found ${result.errors.length} parse errors`);
      for (const error of result.errors) {
        this.config.logger.warn(`Parse error in ${error.filePath}:${error.lineNumber} - ${error.message}`);
      }
    }

    this.handlers = result.handlers;
    this.config.logger.info(`Parsed ${this.handlers.length} handlers from ${this.config.handlersDir}`);
  }

  /**
   * Register all handlers with their middleware
   */
  registerHandlers(registry: HandlerRegistry): void {
    for (const handler of this.handlers) {
      const handlerFn = registry.get(handler.packageName, handler.functionName);

      if (!handlerFn) {
        this.config.logger.warn(
          `Handler not found in registry: ${handler.packageName}.${handler.functionName}`
        );
        continue;
      }

      // Build middleware chain
      const middleware: RequestHandler[] = [];

      // Add CORS middleware if configured
      if (handler.cors) {
        middleware.push(createCORSMiddleware(handler.cors));
      }

      // Add auth middleware
      if (handler.auth && handler.auth !== AuthType.None) {
        middleware.push(createAuthMiddleware(handler.auth));
      }

      // Add rate limit middleware if configured
      if (handler.rateLimit) {
        middleware.push(createRateLimitMiddleware(handler.rateLimit));
      }

      // Add timeout middleware if configured
      if (handler.timeout) {
        middleware.push(createTimeoutMiddleware(handler.timeout));
      }

      // Add the actual handler
      middleware.push(handlerFn);

      // Register route with Express
      const method = handler.route.method.toLowerCase() as 'get' | 'post' | 'put' | 'delete' | 'patch';
      this.app[method](handler.route.path, ...middleware);

      this.config.logger.info(
        `Registered ${handler.route.method} ${handler.route.path} -> ${handler.functionName} [${handler.deploymentType}]`
      );
    }

    // Add error handling middleware last
    this.app.use(notFoundMiddleware);
    this.app.use(errorMiddleware);
  }

  /**
   * Get all registered handlers
   */
  getHandlers(): Handler[] {
    return this.handlers;
  }

  /**
   * Get the Express app
   */
  getApp(): Express {
    return this.app;
  }
}

/**
 * Create a Box router
 */
export async function createRouter(config: RouterConfig): Promise<BoxRouter> {
  const router = new BoxRouter(config);
  await router.initialize();
  return router;
}
