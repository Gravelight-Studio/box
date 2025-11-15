import { Express, RequestHandler, Request as ExpressRequest, Response as ExpressResponse } from 'express';
import { Pool as PgPool } from 'pg';
import { Handler } from '../annotations';
/**
 * HTTP Request type (re-exported from Express for convenience)
 */
export type Request = ExpressRequest;
/**
 * HTTP Response type (re-exported from Express for convenience)
 */
export type Response = ExpressResponse;
/**
 * PostgreSQL connection pool type (re-exported from pg for convenience)
 */
export type Pool = PgPool;
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
 * Handler function type
 */
export type HandlerFunction = (req: Request, res: Response) => void | Promise<void>;
/**
 * Handler factory function type
 * Returns a function compatible with Express middleware
 */
export type HandlerFactory = (db: PgPool | null, logger: Logger) => RequestHandler;
/**
 * Router configuration
 */
export interface RouterConfig {
    handlersDir: string;
    db?: PgPool | null;
    logger: Logger;
}
/**
 * Handler registry for storing handler implementations
 */
export declare class HandlerRegistry {
    private db;
    private logger;
    private handlers;
    constructor(db: PgPool | null, logger: Logger);
    /**
     * Register a handler implementation
     */
    register(packageName: string, functionName: string, factory: HandlerFactory): void;
    /**
     * Get a handler implementation
     */
    get(packageName: string, functionName: string): HandlerFactory | undefined;
    /**
     * Get the database pool
     */
    getDB(): PgPool | null;
    /**
     * Get the logger
     */
    getLogger(): Logger;
}
/**
 * Box router with annotation-driven route registration
 */
export declare class BoxRouter {
    private app;
    private handlers;
    private config;
    constructor(config: RouterConfig);
    /**
     * Initialize router by parsing annotations
     */
    initialize(): Promise<void>;
    /**
     * Register all handlers with their middleware
     */
    registerHandlers(registry: HandlerRegistry): void;
    /**
     * Get all registered handlers
     */
    getHandlers(): Handler[];
    /**
     * Get the Express app
     */
    getApp(): Express;
}
/**
 * Create a Box router
 */
export declare function createRouter(config: RouterConfig): Promise<BoxRouter>;
//# sourceMappingURL=router.d.ts.map