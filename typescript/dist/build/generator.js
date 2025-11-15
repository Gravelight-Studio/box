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
exports.Generator = void 0;
const fs = __importStar(require("fs"));
const path = __importStar(require("path"));
const function_1 = require("./function");
const container_1 = require("./container");
const gateway_1 = require("./gateway");
const terraform_1 = require("./terraform");
const annotations_1 = require("../annotations");
/**
 * Main build generator that orchestrates all artifact generation
 */
class Generator {
    config;
    logger;
    functionGenerator;
    containerGenerator;
    gatewayGenerator;
    terraformGenerator;
    constructor(config, logger) {
        this.config = {
            ...config,
            region: config.region || 'us-central1',
            environment: config.environment || 'dev',
            cleanBuildDir: config.cleanBuildDir ?? false
        };
        this.logger = logger;
        // Initialize sub-generators
        this.functionGenerator = new function_1.FunctionGenerator(config.handlers, path.join(config.outputDir, 'functions'), config.moduleName, logger);
        this.containerGenerator = new container_1.ContainerGenerator(config.handlers, path.join(config.outputDir, 'containers'), config.moduleName, logger);
        this.gatewayGenerator = new gateway_1.GatewayGenerator(config.handlers, path.join(config.outputDir, 'gateway'), config.moduleName, config.projectId, this.config.region, logger);
        this.terraformGenerator = new terraform_1.TerraformGenerator(config.handlers, path.join(config.outputDir, 'terraform'), config.moduleName, config.projectId, this.config.region, logger);
    }
    /**
     * Generate all deployment artifacts
     */
    async generate() {
        this.logger.info('Starting build generation', {
            totalHandlers: this.config.handlers.length,
            outputDir: this.config.outputDir
        });
        // Clean build directory if requested
        if (this.config.cleanBuildDir) {
            await this.cleanOutputDir();
        }
        // Create output directory
        await fs.promises.mkdir(this.config.outputDir, { recursive: true });
        // Count handlers by type
        const functionCount = this.config.handlers.filter(h => h.deploymentType === annotations_1.DeploymentType.Function).length;
        const containerCount = this.config.handlers.filter(h => h.deploymentType === annotations_1.DeploymentType.Container).length;
        // Generate Cloud Functions
        this.logger.info('Generating cloud functions', { count: functionCount });
        await this.functionGenerator.generate();
        // Generate Cloud Run containers
        this.logger.info('Generating cloud run containers', { count: containerCount });
        await this.containerGenerator.generate();
        // Generate API Gateway configuration
        this.logger.info('Generating API Gateway configuration', {
            handlers: this.config.handlers.length
        });
        await this.gatewayGenerator.generate();
        // Generate Terraform infrastructure
        this.logger.info('Generating Terraform infrastructure', {
            handlers: this.config.handlers.length
        });
        await this.terraformGenerator.generate();
        this.logger.info('Build generation complete', {
            functionsGenerated: functionCount,
            containersGenerated: containerCount,
            totalEndpoints: this.config.handlers.length
        });
    }
    /**
     * Clean output directory
     */
    async cleanOutputDir() {
        try {
            await fs.promises.rm(this.config.outputDir, { recursive: true, force: true });
            this.logger.info('Cleaned build directory', { dir: this.config.outputDir });
        }
        catch (error) {
            this.logger.warn('Failed to clean build directory', { error });
        }
    }
    /**
     * Generate functions only
     */
    async generateFunctions() {
        await this.functionGenerator.generate();
    }
    /**
     * Generate containers only
     */
    async generateContainers() {
        await this.containerGenerator.generate();
    }
    /**
     * Generate gateway only
     */
    async generateGateway() {
        await this.gatewayGenerator.generate();
    }
    /**
     * Generate terraform only
     */
    async generateTerraform() {
        await this.terraformGenerator.generate();
    }
}
exports.Generator = Generator;
//# sourceMappingURL=generator.js.map