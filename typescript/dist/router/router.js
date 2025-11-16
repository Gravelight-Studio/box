"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.BoxRouter = exports.HandlerRegistry = void 0;
exports.createRouter = createRouter;
const express_1 = __importDefault(require("express"));
const annotations_1 = require("../annotations");
const middleware_1 = require("./middleware");
/**
 * Handler registry for storing handler implementations
 */
class HandlerRegistry {
    logger;
    handlers = new Map();
    constructor(logger) {
        this.logger = logger;
    }
    /**
     * Register a handler implementation
     */
    register(packageName, functionName, handler) {
        const key = `${packageName}.${functionName}`;
        this.handlers.set(key, handler);
        this.logger.debug(`Registered handler: ${key}`);
    }
    /**
     * Get a handler implementation
     */
    get(packageName, functionName) {
        const key = `${packageName}.${functionName}`;
        return this.handlers.get(key);
    }
}
exports.HandlerRegistry = HandlerRegistry;
/**
 * Box router with annotation-driven route registration
 */
class BoxRouter {
    app;
    handlers = [];
    config;
    constructor(config) {
        this.config = config;
        this.app = (0, express_1.default)();
        // Add body parsing middleware
        this.app.use(express_1.default.json());
        this.app.use(express_1.default.urlencoded({ extended: true }));
    }
    /**
     * Initialize router by parsing annotations
     */
    async initialize() {
        const parser = new annotations_1.Parser();
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
    registerHandlers(registry) {
        for (const handler of this.handlers) {
            const handlerFn = registry.get(handler.packageName, handler.functionName);
            if (!handlerFn) {
                this.config.logger.warn(`Handler not found in registry: ${handler.packageName}.${handler.functionName}`);
                continue;
            }
            // Build middleware chain
            const middleware = [];
            // Add CORS middleware if configured
            if (handler.cors) {
                middleware.push((0, middleware_1.createCORSMiddleware)(handler.cors));
            }
            // Add auth middleware
            if (handler.auth && handler.auth !== annotations_1.AuthType.None) {
                middleware.push((0, middleware_1.createAuthMiddleware)(handler.auth));
            }
            // Add rate limit middleware if configured
            if (handler.rateLimit) {
                middleware.push((0, middleware_1.createRateLimitMiddleware)(handler.rateLimit));
            }
            // Add timeout middleware if configured
            if (handler.timeout) {
                middleware.push((0, middleware_1.createTimeoutMiddleware)(handler.timeout));
            }
            // Add the actual handler
            middleware.push(handlerFn);
            // Register route with Express
            const method = handler.route.method.toLowerCase();
            this.app[method](handler.route.path, ...middleware);
            this.config.logger.info(`Registered ${handler.route.method} ${handler.route.path} -> ${handler.functionName} [${handler.deploymentType}]`);
        }
        // Add error handling middleware last
        this.app.use(middleware_1.notFoundMiddleware);
        this.app.use(middleware_1.errorMiddleware);
    }
    /**
     * Get all registered handlers
     */
    getHandlers() {
        return this.handlers;
    }
    /**
     * Get the Express app
     */
    getApp() {
        return this.app;
    }
}
exports.BoxRouter = BoxRouter;
/**
 * Create a Box router
 */
async function createRouter(config) {
    const router = new BoxRouter(config);
    await router.initialize();
    return router;
}
//# sourceMappingURL=router.js.map