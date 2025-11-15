"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createCORSMiddleware = createCORSMiddleware;
exports.createAuthMiddleware = createAuthMiddleware;
exports.createRateLimitMiddleware = createRateLimitMiddleware;
exports.createTimeoutMiddleware = createTimeoutMiddleware;
exports.errorMiddleware = errorMiddleware;
exports.notFoundMiddleware = notFoundMiddleware;
const cors_1 = __importDefault(require("cors"));
const express_rate_limit_1 = __importDefault(require("express-rate-limit"));
const annotations_1 = require("../annotations");
/**
 * Create CORS middleware from configuration
 */
function createCORSMiddleware(config) {
    return (0, cors_1.default)({
        origin: config.origins.includes('*') ? '*' : config.origins,
        methods: config.methods || ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
        allowedHeaders: config.allowedHeaders || ['Content-Type', 'Authorization'],
        exposedHeaders: config.exposedHeaders || [],
        credentials: config.credentials ?? false,
        maxAge: config.maxAge
    });
}
/**
 * Create authentication middleware
 */
function createAuthMiddleware(authType) {
    return (req, res, next) => {
        if (authType === annotations_1.AuthType.None) {
            return next();
        }
        const authHeader = req.headers.authorization;
        if (!authHeader) {
            if (authType === annotations_1.AuthType.Required) {
                return res.status(401).json({ error: 'Unauthorized: Missing authorization header' });
            }
            // Optional auth - continue without user
            return next();
        }
        // Parse Bearer token
        const parts = authHeader.split(' ');
        if (parts.length !== 2 || parts[0] !== 'Bearer') {
            return res.status(401).json({ error: 'Unauthorized: Invalid authorization format' });
        }
        const token = parts[1];
        // Basic token validation (replace with your actual auth logic)
        if (!token || token.length === 0) {
            return res.status(401).json({ error: 'Unauthorized: Invalid token' });
        }
        // Attach user info to request (in real app, verify JWT and extract user)
        req.user = { token };
        next();
    };
}
/**
 * Create rate limiting middleware
 */
function createRateLimitMiddleware(config) {
    return (0, express_rate_limit_1.default)({
        windowMs: config.windowMs,
        max: config.requests,
        message: {
            error: `Too many requests, please try again later. Limit: ${config.requests}/${config.period}`
        },
        standardHeaders: true,
        legacyHeaders: false
    });
}
/**
 * Create timeout middleware
 */
function createTimeoutMiddleware(config) {
    return (req, res, next) => {
        const timeout = setTimeout(() => {
            if (!res.headersSent) {
                res.status(504).json({ error: 'Request timeout' });
            }
        }, config.duration);
        // Clear timeout when response is sent
        res.on('finish', () => clearTimeout(timeout));
        res.on('close', () => clearTimeout(timeout));
        next();
    };
}
/**
 * Error handling middleware
 */
function errorMiddleware(err, req, res, next) {
    console.error('Error:', err);
    if (res.headersSent) {
        return next(err);
    }
    res.status(500).json({
        error: 'Internal server error',
        message: process.env.NODE_ENV === 'development' ? err.message : undefined
    });
}
/**
 * 404 Not Found middleware
 */
function notFoundMiddleware(req, res) {
    res.status(404).json({
        error: 'Not found',
        path: req.path
    });
}
//# sourceMappingURL=middleware.js.map