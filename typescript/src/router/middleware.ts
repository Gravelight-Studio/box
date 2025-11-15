import { Request, Response, NextFunction, RequestHandler } from 'express';
import cors from 'cors';
import rateLimit from 'express-rate-limit';
import { AuthType, CORSConfig, RateLimitConfig, TimeoutConfig } from '../annotations';

/**
 * Create CORS middleware from configuration
 */
export function createCORSMiddleware(config: CORSConfig): RequestHandler {
  return cors({
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
export function createAuthMiddleware(authType: AuthType): RequestHandler {
  return (req: Request, res: Response, next: NextFunction) => {
    if (authType === AuthType.None) {
      return next();
    }

    const authHeader = req.headers.authorization;

    if (!authHeader) {
      if (authType === AuthType.Required) {
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
    (req as any).user = { token };

    next();
  };
}

/**
 * Create rate limiting middleware
 */
export function createRateLimitMiddleware(config: RateLimitConfig): RequestHandler {
  return rateLimit({
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
export function createTimeoutMiddleware(config: TimeoutConfig): RequestHandler {
  return (req: Request, res: Response, next: NextFunction) => {
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
export function errorMiddleware(
  err: Error,
  req: Request,
  res: Response,
  next: NextFunction
): void {
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
export function notFoundMiddleware(req: Request, res: Response): void {
  res.status(404).json({
    error: 'Not found',
    path: req.path
  });
}
