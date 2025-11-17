import { resolve } from 'node:path';
import pino from 'pino';
import express, { Express, RequestHandler, Request as ExpressRequest, Response as ExpressResponse } from 'express';
import { Parser, AuthType } from '../annotations';
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
  handlersDirs: string[];
  logger?: Logger
}

/**
 * Box router with annotation-driven route registration
 */
export class BoxRouter {
  private app: Express;
  private config: Required<RouterConfig>;

  constructor(config: RouterConfig) {
    
    const logger = config.logger ?? pino({
      transport: {
        target: 'pino-pretty',
        options: {
          colorize: true,
          ignore: 'pid,hostname'
        }
      }
    });

    this.app = express();
    this.config = {...config, logger};

    // Add body parsing middleware
    this.app.use(express.json());
    this.app.use(express.urlencoded({ extended: true }));
  }

  /**
   * Initialize router by parsing annotations and registering handlers
   */
  async initialize(): Promise<void> {
    const parser = new Parser();
    const {errors, handlers} = (await Promise.all(
      this.config.handlersDirs.map(handlerDir => parser.parseDirectory(handlerDir))
    )).reduce((final, parsedAnnotations) => {
      return {
        errors: [...final.errors, ...parsedAnnotations.errors],
        handlers: [...final.handlers, ...parsedAnnotations.handlers]
      }
    });

    if (errors.length > 0) {
      this.config.logger.warn(`Found ${errors.length} parse errors`);
      for (const error of errors) {
        this.config.logger.warn(`Parse error in ${error.filePath}:${error.lineNumber} - ${error.message}`);
      }
    }

    this.config.logger.info(`Parsed ${handlers.length} handlers from ${this.config.handlersDirs.join(',')}`);

    // Register all provided handler implementations
    for (const handler of handlers) {
      // Resolve path relative to current working directory
      const absolutePath = resolve(process.cwd(), handler.filePath);

      this.config.logger.debug(`Loading handler from: ${absolutePath}`);

      let handlerFn;
      try {
        handlerFn = require(absolutePath)[handler.functionName];
      } catch (error) {
        this.config.logger.error(
          `Failed to load handler file ${absolutePath}: ${error}`
        );
        continue;
      }

      if (!handlerFn) {
        this.config.logger.warn(
          `Handler function '${handler.functionName}' not found in ${absolutePath}`
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

      // Convert Chi-style path parameters {id} to Express-style :id
      const expressPath = handler.route.path.replace(/\{(\w+)\}/g, ':$1');

      // Register route with Express
      const method = handler.route.method.toLowerCase() as 'get' | 'post' | 'put' | 'delete' | 'patch';
      this.app[method](expressPath, ...middleware);

      this.config.logger.info(
        `Registered ${handler.route.method} ${handler.route.path} -> ${handler.functionName} [${handler.deploymentType}]`
      );
    }

    // Add error handling middleware last
    this.app.use(notFoundMiddleware);
    this.app.use(errorMiddleware);
  }

  /**
   * Get the Express app
   */
  listen(port: string | number, callback?: () => void): void {
    this.app.listen(port, () => {
      this.config.logger.info(`Server listening on port ${port}`);
      if (callback) callback();
    });
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
