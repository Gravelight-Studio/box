"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.BoxRouter = void 0;
exports.createRouter = createRouter;
const node_path_1 = require("node:path");
const pino_1 = __importDefault(require("pino"));
const express_1 = __importDefault(require("express"));
const annotations_1 = require("../annotations");
const middleware_1 = require("./middleware");
/**
 * Box router with annotation-driven route registration
 */
class BoxRouter {
    app;
    config;
    constructor(config) {
        const logger = config.logger ?? (0, pino_1.default)({
            transport: {
                target: 'pino-pretty',
                options: {
                    colorize: true,
                    ignore: 'pid,hostname'
                }
            }
        });
        this.app = (0, express_1.default)();
        this.config = { ...config, logger };
        // Add body parsing middleware
        this.app.use(express_1.default.json());
        this.app.use(express_1.default.urlencoded({ extended: true }));
    }
    /**
     * Initialize router by parsing annotations and registering handlers
     */
    async initialize() {
        const parser = new annotations_1.Parser();
        const { errors, handlers } = (await Promise.all(this.config.handlersDirs.map(handlerDir => parser.parseDirectory(handlerDir)))).reduce((final, parsedAnnotations) => {
            return {
                errors: [...final.errors, ...parsedAnnotations.errors],
                handlers: [...final.handlers, ...parsedAnnotations.handlers]
            };
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
            const handlerFn = require((0, node_path_1.resolve)(handler.filePath))[handler.functionName];
            if (!handlerFn) {
                this.config.logger.warn(`Handler not found: ${handler.packageName}.${handler.functionName}`);
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
     * Get the Express app
     */
    listen(port, callback) {
        this.app.listen(port, () => {
            this.config.logger.info(`Server listening on port ${port}`);
            if (callback)
                callback();
        });
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