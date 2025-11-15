import { Request, Response, NextFunction, RequestHandler } from 'express';
import { AuthType, CORSConfig, RateLimitConfig, TimeoutConfig } from '../annotations';
/**
 * Create CORS middleware from configuration
 */
export declare function createCORSMiddleware(config: CORSConfig): RequestHandler;
/**
 * Create authentication middleware
 */
export declare function createAuthMiddleware(authType: AuthType): RequestHandler;
/**
 * Create rate limiting middleware
 */
export declare function createRateLimitMiddleware(config: RateLimitConfig): RequestHandler;
/**
 * Create timeout middleware
 */
export declare function createTimeoutMiddleware(config: TimeoutConfig): RequestHandler;
/**
 * Error handling middleware
 */
export declare function errorMiddleware(err: Error, req: Request, res: Response, next: NextFunction): void;
/**
 * 404 Not Found middleware
 */
export declare function notFoundMiddleware(req: Request, res: Response): void;
//# sourceMappingURL=middleware.d.ts.map