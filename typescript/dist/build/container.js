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
exports.ContainerGenerator = void 0;
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const annotations_1 = require("../annotations");
/**
 * Generator for Cloud Run containers
 */
class ContainerGenerator {
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
     * Generate all container services
     */
    async generate() {
        const containerHandlers = this.handlers.filter(h => h.deploymentType === annotations_1.DeploymentType.Container);
        if (containerHandlers.length === 0) {
            this.logger.info('No container handlers to generate');
            return;
        }
        // Group handlers by package name (service)
        const serviceGroups = this.groupByService(containerHandlers);
        this.logger.info(`Grouped container handlers`, {
            totalHandlers: containerHandlers.length,
            serviceGroups: Object.keys(serviceGroups).length
        });
        // Create output directory
        await fs.promises.mkdir(this.outputDir, { recursive: true });
        // Generate each service
        for (const [serviceName, handlers] of Object.entries(serviceGroups)) {
            await this.generateService(serviceName, handlers);
        }
        this.logger.info(`Generated ${Object.keys(serviceGroups).length} container services`, {
            outputDir: this.outputDir
        });
    }
    /**
     * Group handlers by package name
     */
    groupByService(handlers) {
        const groups = {};
        for (const handler of handlers) {
            const serviceName = handler.packageName;
            if (!groups[serviceName]) {
                groups[serviceName] = [];
            }
            groups[serviceName].push(handler);
        }
        return groups;
    }
    /**
     * Generate a container service
     */
    async generateService(serviceName, handlers) {
        const serviceDir = path.join(this.outputDir, serviceName);
        this.logger.info(`Generating container service: ${serviceName}`, {
            handlers: handlers.length,
            outputDir: serviceDir
        });
        // Create service directory
        await fs.promises.mkdir(serviceDir, { recursive: true });
        // Generate Dockerfile
        await this.generateDockerfile(serviceDir, serviceName, handlers);
        // Generate package.json
        await this.generatePackageJson(serviceDir, serviceName);
        // Generate server.js
        await this.generateServerJs(serviceDir, handlers);
        // Generate .dockerignore
        await this.generateDockerIgnore(serviceDir);
        // Generate cloudbuild.yaml
        await this.generateCloudBuild(serviceDir, serviceName);
    }
    /**
     * Generate Dockerfile
     */
    async generateDockerfile(dir, serviceName, handlers) {
        const dockerfile = `# Multi-stage build for ${serviceName}
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Production stage
FROM node:20-alpine

WORKDIR /app

# Copy dependencies from builder
COPY --from=builder /app/node_modules ./node_modules

# Copy application code
COPY . .

# Set environment variables
ENV NODE_ENV=production
ENV PORT=8080

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \\
  CMD node -e "require('http').get('http://localhost:8080/health', (r) => {process.exit(r.statusCode === 200 ? 0 : 1)})"

# Run as non-root user
USER node

# Start server
CMD ["node", "server.js"]
`;
        await fs.promises.writeFile(path.join(dir, 'Dockerfile'), dockerfile);
    }
    /**
     * Generate package.json
     */
    async generatePackageJson(dir, serviceName) {
        const pkg = {
            name: `box-container-${serviceName}`,
            version: '1.0.0',
            description: `Cloud Run container for ${serviceName}`,
            main: 'server.js',
            scripts: {
                start: 'node server.js'
            },
            dependencies: {
                'express': '^4.18.2',
                'cors': '^2.8.5',
                'express-rate-limit': '^7.1.5',
                'pino': '^8.17.2'
            },
            engines: {
                node: '>=18.0.0'
            }
        };
        await fs.promises.writeFile(path.join(dir, 'package.json'), JSON.stringify(pkg, null, 2));
    }
    /**
     * Generate server.js
     */
    async generateServerJs(dir, handlers) {
        const code = `const express = require('express');
const cors = require('cors');
const rateLimit = require('express-rate-limit');
const pino = require('pino');

const app = express();
const logger = pino();

// Middleware
app.use(express.json());
app.use(express.urlencoded({ extended: true }));

${handlers.map(handler => this.generateHandlerCode(handler)).join('\n\n')}

// Health check
app.get('/health', (req, res) => {
  res.status(200).json({ status: 'healthy', timestamp: new Date() });
});

// Error handling
app.use((err, req, res, next) => {
  logger.error(err);
  res.status(500).json({ error: 'Internal server error' });
});

// 404 handler
app.use((req, res) => {
  res.status(404).json({ error: 'Not found', path: req.path });
});

// Start server
const port = process.env.PORT || 8080;
app.listen(port, () => {
  logger.info(\`Server listening on port \${port}\`);
});
`;
        await fs.promises.writeFile(path.join(dir, 'server.js'), code);
    }
    /**
     * Generate handler code
     */
    generateHandlerCode(handler) {
        const middleware = [];
        // CORS
        if (handler.cors) {
            middleware.push(`cors({
    origin: ${JSON.stringify(handler.cors.origins.includes('*') ? '*' : handler.cors.origins)},
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS']
  })`);
        }
        // Rate limiting
        if (handler.rateLimit) {
            middleware.push(`rateLimit({
    windowMs: ${handler.rateLimit.windowMs},
    max: ${handler.rateLimit.requests}
  })`);
        }
        // Auth middleware
        const authMiddleware = handler.auth === 'required' ? `
  const authHeader = req.headers.authorization;
  if (!authHeader || !authHeader.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'Unauthorized' });
  }
  const token = authHeader.split(' ')[1];
  if (!token) {
    return res.status(401).json({ error: 'Unauthorized' });
  }
  req.user = { token };
` : '';
        return `// ${handler.functionName} - ${handler.route.method} ${handler.route.path}
app.${handler.route.method.toLowerCase()}('${handler.route.path}'${middleware.length > 0 ? `, ${middleware.join(', ')}` : ''}, async (req, res) => {
  ${authMiddleware}
  try {
    // TODO: Import and call your actual handler logic here
    logger.info('${handler.functionName} called');
    res.status(200).json({
      message: '${handler.functionName} executed successfully',
      method: '${handler.route.method}',
      path: '${handler.route.path}'
    });
  } catch (error) {
    logger.error(error);
    res.status(500).json({ error: 'Internal server error' });
  }
});`;
    }
    /**
     * Generate .dockerignore
     */
    async generateDockerIgnore(dir) {
        const ignore = `node_modules
npm-debug.log
.git
.gitignore
.DS_Store
*.md
.env
.env.local
`;
        await fs.promises.writeFile(path.join(dir, '.dockerignore'), ignore);
    }
    /**
     * Generate cloudbuild.yaml
     */
    async generateCloudBuild(dir, serviceName) {
        const yaml = `# Cloud Build configuration for ${serviceName}
steps:
  # Build the container image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'build'
      - '-t'
      - 'gcr.io/$PROJECT_ID/${serviceName}:$SHORT_SHA'
      - '-t'
      - 'gcr.io/$PROJECT_ID/${serviceName}:latest'
      - '.'

  # Push the container image
  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/$PROJECT_ID/${serviceName}:$SHORT_SHA'

  - name: 'gcr.io/cloud-builders/docker'
    args:
      - 'push'
      - 'gcr.io/$PROJECT_ID/${serviceName}:latest'

  # Deploy to Cloud Run
  - name: 'gcr.io/google.com/cloudsdktool/cloud-sdk'
    entrypoint: gcloud
    args:
      - 'run'
      - 'deploy'
      - '${serviceName}'
      - '--image'
      - 'gcr.io/$PROJECT_ID/${serviceName}:$SHORT_SHA'
      - '--region'
      - '$_REGION'
      - '--platform'
      - 'managed'
      - '--allow-unauthenticated'

images:
  - 'gcr.io/$PROJECT_ID/${serviceName}:$SHORT_SHA'
  - 'gcr.io/$PROJECT_ID/${serviceName}:latest'

options:
  logging: CLOUD_LOGGING_ONLY

substitutions:
  _REGION: us-central1
`;
        await fs.promises.writeFile(path.join(dir, 'cloudbuild.yaml'), yaml);
    }
}
exports.ContainerGenerator = ContainerGenerator;
//# sourceMappingURL=container.js.map