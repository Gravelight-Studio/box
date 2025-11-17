import { Request as ExpressRequest, Response as ExpressResponse } from 'express';
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
    logger?: Logger;
}
/**
 * Box router with annotation-driven route registration
 */
export declare class BoxRouter {
    private app;
    private config;
    constructor(config: RouterConfig);
    /**
     * Initialize router by parsing annotations and registering handlers
     */
    initialize(): Promise<void>;
    /**
     * Get the Express app
     */
    listen(port: string | number, callback?: () => void): void;
}
/**
 * Create a Box router
 */
export declare function createRouter(config: RouterConfig): Promise<BoxRouter>;
//# sourceMappingURL=router.d.ts.map