/**
 * Deployment types for Box handlers
 */
export declare enum DeploymentType {
    Function = "function",
    Container = "container"
}
/**
 * Authentication requirement levels
 */
export declare enum AuthType {
    None = "none",
    Optional = "optional",
    Required = "required"
}
/**
 * Route definition for HTTP endpoints
 */
export interface Route {
    method: string;
    path: string;
}
/**
 * CORS configuration
 */
export interface CORSConfig {
    origins: string[];
    methods?: string[];
    allowedHeaders?: string[];
    exposedHeaders?: string[];
    credentials?: boolean;
    maxAge?: number;
}
/**
 * Rate limit configuration
 */
export interface RateLimitConfig {
    requests: number;
    period: 'second' | 'minute' | 'hour' | 'day';
    windowMs: number;
}
/**
 * Timeout configuration
 */
export interface TimeoutConfig {
    duration: number;
}
/**
 * Memory configuration (for functions)
 */
export interface MemoryConfig {
    size: number;
}
/**
 * Concurrency configuration (for containers)
 */
export interface ConcurrencyConfig {
    max: number;
}
/**
 * Handler metadata parsed from annotations
 */
export interface Handler {
    packageName: string;
    functionName: string;
    filePath: string;
    deploymentType: DeploymentType;
    route: Route;
    auth: AuthType;
    cors?: CORSConfig;
    rateLimit?: RateLimitConfig;
    timeout?: TimeoutConfig;
    memory?: MemoryConfig;
    concurrency?: ConcurrencyConfig;
}
/**
 * Parse error information
 */
export interface ParseError {
    filePath: string;
    lineNumber: number;
    message: string;
    annotation: string;
}
/**
 * Result of annotation parsing
 */
export interface ParsedAnnotations {
    handlers: Handler[];
    errors: ParseError[];
}
//# sourceMappingURL=types.d.ts.map