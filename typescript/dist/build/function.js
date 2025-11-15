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
exports.FunctionGenerator = void 0;
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const annotations_1 = require("../annotations");
/**
 * Generator for Cloud Functions
 */
class FunctionGenerator {
    handlers;
    outputDir;
    moduleName;
    logger;
    constructor(handlers, outputDir, moduleName, logger) {
        this.handlers = handlers;
        this.outputDir = outputDir;
        this.moduleName = moduleName;
        this.logger = logger;
    }
    /**
     * Generate all function packages
     */
    async generate() {
        const functionHandlers = this.handlers.filter(h => h.deploymentType === annotations_1.DeploymentType.Function);
        if (functionHandlers.length === 0) {
            this.logger.info('No function handlers to generate');
            return;
        }
        // Create output directory
        await fs.promises.mkdir(this.outputDir, { recursive: true });
        // Generate each function
        for (const handler of functionHandlers) {
            await this.generateFunction(handler);
        }
        this.logger.info(`Generated ${functionHandlers.length} cloud functions`, {
            outputDir: this.outputDir
        });
    }
    /**
     * Generate a single function package
     */
    async generateFunction(handler) {
        const functionName = this.getFunctionName(handler);
        const functionDir = path.join(this.outputDir, functionName);
        this.logger.info(`Generating cloud function: ${functionName}`, {
            path: handler.route.path,
            outputDir: functionDir
        });
        // Create function directory
        await fs.promises.mkdir(functionDir, { recursive: true });
        // Generate package.json
        await this.generatePackageJson(functionDir, handler);
        // Generate index.js (entry point)
        await this.generateIndexJs(functionDir, handler);
        // Generate function.yaml (Cloud Functions config)
        await this.generateFunctionYaml(functionDir, handler);
        // Generate .gcloudignore
        await this.generateGcloudIgnore(functionDir);
    }
    /**
     * Generate package.json for function
     */
    async generatePackageJson(dir, handler) {
        const pkg = {
            name: this.getFunctionName(handler),
            version: '1.0.0',
            description: `Cloud Function for ${handler.route.method} ${handler.route.path}`,
            main: 'index.js',
            scripts: {
                start: 'node index.js'
            },
            dependencies: {
                '@google-cloud/functions-framework': '^3.3.0',
                'express': '^4.18.2',
                'cors': '^2.8.5',
                'express-rate-limit': '^7.1.5'
            },
            engines: {
                node: '>=18.0.0'
            }
        };
        await fs.promises.writeFile(path.join(dir, 'package.json'), JSON.stringify(pkg, null, 2));
    }
    /**
     * Generate index.js entry point
     */
    async generateIndexJs(dir, handler) {
        const code = `const functions = require('@google-cloud/functions-framework');
const cors = require('cors');
const rateLimit = require('express-rate-limit');

// CORS configuration
${handler.cors ? `const corsMiddleware = cors({
  origin: ${JSON.stringify(handler.cors.origins.includes('*') ? '*' : handler.cors.origins)},
  methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'Authorization'],
  credentials: false
});` : ''}

// Rate limit configuration
${handler.rateLimit ? `const rateLimiter = rateLimit({
  windowMs: ${handler.rateLimit.windowMs},
  max: ${handler.rateLimit.requests},
  message: { error: 'Too many requests, please try again later. Limit: ${handler.rateLimit.requests}/${handler.rateLimit.period}' },
  standardHeaders: true,
  legacyHeaders: false
});` : ''}

// Main handler function
functions.http('${handler.functionName}', async (req, res) => {
  ${handler.cors ? 'corsMiddleware(req, res, () => {});' : ''}

  // Authentication
  ${handler.auth === 'required' ? `
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Unauthorized: Missing or invalid authorization header' });
  }
  const token = authHeader.split(' ')[1];
  if (!token) {
    return res.status(401).json({ error: 'Unauthorized: Invalid token' });
  }
  req.user = { token };
  ` : ''}

  // Rate limiting
  ${handler.rateLimit ? 'rateLimiter(req, res, () => {});' : ''}

  // Handle request
  try {
    // TODO: Import and call your actual handler logic here
    // For now, return a placeholder response
    res.status(200).json({
      message: '${handler.functionName} executed successfully',
      method: '${handler.route.method}',
      path: '${handler.route.path}'
    });
  } catch (error) {
    console.error('Error:', error);
    res.status(500).json({ error: 'Internal server error' });
  }
});
`;
        await fs.promises.writeFile(path.join(dir, 'index.js'), code);
    }
    /**
     * Generate function.yaml configuration
     */
    async generateFunctionYaml(dir, handler) {
        const yaml = `# Cloud Function configuration for ${handler.functionName}
runtime: nodejs20
entryPoint: ${handler.functionName}

# Resource configuration
${handler.memory ? `availableMemoryMb: ${handler.memory.size}` : 'availableMemoryMb: 256'}
${handler.timeout ? `timeout: ${Math.floor(handler.timeout.duration / 1000)}s` : 'timeout: 60s'}
maxInstances: 100

# Environment variables
environmentVariables:
  NODE_ENV: production
  FUNCTION_NAME: ${handler.functionName}
  FUNCTION_PATH: ${handler.route.path}
  FUNCTION_METHOD: ${handler.route.method}
`;
        await fs.promises.writeFile(path.join(dir, 'function.yaml'), yaml);
    }
    /**
     * Generate .gcloudignore
     */
    async generateGcloudIgnore(dir) {
        const ignore = `.gcloudignore
.git
.gitignore
node_modules/
*.log
.DS_Store
`;
        await fs.promises.writeFile(path.join(dir, '.gcloudignore'), ignore);
    }
    /**
     * Get normalized function name
     */
    getFunctionName(handler) {
        return handler.functionName
            .replace(/([A-Z])/g, '-$1')
            .toLowerCase()
            .replace(/^-/, '');
    }
}
exports.FunctionGenerator = FunctionGenerator;
//# sourceMappingURL=function.js.map