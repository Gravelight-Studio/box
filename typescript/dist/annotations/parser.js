"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.Parser = void 0;
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const types_1 = require("./types");
/**
 * Parser for extracting Box annotations from TypeScript/JavaScript files
 */
class Parser {
    /**
     * Parse all handlers in a directory
     */
    async parseDirectory(dir) {
        const result = {
            handlers: [],
            errors: []
        };
        await this.walkDirectory(dir, result);
        return result;
    }
    /**
     * Parse a single file for handlers
     */
    async parseFile(filePath) {
        const result = {
            handlers: [],
            errors: []
        };
        try {
            const content = await fs.promises.readFile(filePath, 'utf-8');
            const handlers = this.extractHandlers(content, filePath);
            result.handlers.push(...handlers);
        }
        catch (error) {
            result.errors.push({
                filePath,
                lineNumber: 0,
                message: `Failed to read file: ${error}`,
                annotation: ''
            });
        }
        return result;
    }
    /**
     * Recursively walk directory and parse files
     */
    async walkDirectory(dir, result) {
        const entries = await fs.promises.readdir(dir, { withFileTypes: true });
        for (const entry of entries) {
            const fullPath = path.join(dir, entry.name);
            if (entry.isDirectory()) {
                // Skip node_modules and hidden directories
                if (entry.name === 'node_modules' || entry.name.startsWith('.')) {
                    continue;
                }
                await this.walkDirectory(fullPath, result);
            }
            else if (entry.isFile()) {
                // Only parse TypeScript and JavaScript files
                if (entry.name.endsWith('.ts') || entry.name.endsWith('.js')) {
                    // Skip test files and type definition files
                    if (!entry.name.endsWith('.test.ts') &&
                        !entry.name.endsWith('.test.js') &&
                        !entry.name.endsWith('.d.ts')) {
                        const fileResult = await this.parseFile(fullPath);
                        result.handlers.push(...fileResult.handlers);
                        result.errors.push(...fileResult.errors);
                    }
                }
            }
        }
    }
    /**
     * Extract handlers from file content
     */
    extractHandlers(content, filePath) {
        const handlers = [];
        const lines = content.split('\n');
        for (let i = 0; i < lines.length; i++) {
            // Look for function declarations or const assignments
            // Handles: function foo(), export function foo(), const foo = ..., export const foo: Type = ...
            const functionMatch = lines[i].match(/(?:export\s+)?(?:async\s+)?function\s+(\w+)|(?:export\s+)?const\s+(\w+)\s*(?::\s*\w+\s*)?=/);
            if (functionMatch) {
                const functionName = functionMatch[1] || functionMatch[2];
                // Look backwards for annotations
                const annotations = this.extractAnnotations(lines, i);
                if (annotations.deploymentType) {
                    const handler = this.buildHandler(functionName, filePath, annotations, i + 1);
                    if (handler) {
                        handlers.push(handler);
                    }
                }
            }
        }
        return handlers;
    }
    /**
     * Extract annotations from comments above a function
     */
    extractAnnotations(lines, functionLineIndex) {
        const annotations = {};
        // Look backwards from function line
        for (let i = functionLineIndex - 1; i >= 0; i--) {
            const line = lines[i].trim();
            // Stop at empty lines or non-comment lines
            if (!line || (!line.startsWith('//') && !line.startsWith('*') && !line.startsWith('/*'))) {
                break;
            }
            // Extract annotation
            const annotationMatch = line.match(/@box:(\w+)\s*(.*)/);
            if (annotationMatch) {
                const [, key, value] = annotationMatch;
                this.parseAnnotation(key, value.trim(), annotations);
            }
        }
        return annotations;
    }
    /**
     * Parse individual annotation
     */
    parseAnnotation(key, value, annotations) {
        switch (key) {
            case 'function':
                annotations.deploymentType = types_1.DeploymentType.Function;
                break;
            case 'container':
                annotations.deploymentType = types_1.DeploymentType.Container;
                break;
            case 'path':
                const pathMatch = value.match(/(\w+)\s+(.+)/);
                if (pathMatch) {
                    annotations.route = {
                        method: pathMatch[1].toUpperCase(),
                        path: pathMatch[2]
                    };
                }
                break;
            case 'auth':
                if (value === 'none')
                    annotations.auth = types_1.AuthType.None;
                else if (value === 'optional')
                    annotations.auth = types_1.AuthType.Optional;
                else if (value === 'required')
                    annotations.auth = types_1.AuthType.Required;
                break;
            case 'cors':
                const originsMatch = value.match(/origins=(.+)/);
                if (originsMatch) {
                    const origins = originsMatch[1].split(',').map(o => o.trim());
                    annotations.cors = { origins };
                }
                break;
            case 'ratelimit':
                const rateLimitMatch = value.match(/(\d+)\/(second|minute|hour|day)/);
                if (rateLimitMatch) {
                    const requests = parseInt(rateLimitMatch[1]);
                    const period = rateLimitMatch[2];
                    const windowMs = this.periodToMs(period);
                    annotations.rateLimit = { requests, period, windowMs };
                }
                break;
            case 'timeout':
                const timeoutMatch = value.match(/(\d+)(s|m|h)/);
                if (timeoutMatch) {
                    const duration = parseInt(timeoutMatch[1]);
                    const unit = timeoutMatch[2];
                    annotations.timeout = {
                        duration: this.durationToMs(duration, unit)
                    };
                }
                break;
            case 'memory':
                const memoryMatch = value.match(/(\d+)MB/);
                if (memoryMatch) {
                    annotations.memory = {
                        size: parseInt(memoryMatch[1])
                    };
                }
                break;
            case 'concurrency':
                const concurrencyValue = parseInt(value);
                if (!isNaN(concurrencyValue)) {
                    annotations.concurrency = {
                        max: concurrencyValue
                    };
                }
                break;
        }
    }
    /**
     * Build handler object from annotations
     */
    buildHandler(functionName, filePath, annotations, lineNumber) {
        if (!annotations.route) {
            return null; // Must have a route
        }
        const packageName = path.basename(path.dirname(filePath));
        return {
            packageName,
            functionName,
            filePath,
            deploymentType: annotations.deploymentType,
            route: annotations.route,
            auth: annotations.auth || types_1.AuthType.None,
            cors: annotations.cors,
            rateLimit: annotations.rateLimit,
            timeout: annotations.timeout,
            memory: annotations.memory,
            concurrency: annotations.concurrency
        };
    }
    /**
     * Convert period to milliseconds
     */
    periodToMs(period) {
        switch (period) {
            case 'second': return 1000;
            case 'minute': return 60 * 1000;
            case 'hour': return 60 * 60 * 1000;
            case 'day': return 24 * 60 * 60 * 1000;
        }
    }
    /**
     * Convert duration to milliseconds
     */
    durationToMs(value, unit) {
        switch (unit) {
            case 's': return value * 1000;
            case 'm': return value * 60 * 1000;
            case 'h': return value * 60 * 60 * 1000;
            default: return value * 1000;
        }
    }
}
exports.Parser = Parser;
//# sourceMappingURL=parser.js.map