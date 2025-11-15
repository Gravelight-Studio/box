import * as fs from 'fs';
import * as path from 'path';
import { GeneratorConfig, Logger } from './types';
import { FunctionGenerator } from './function';
import { ContainerGenerator } from './container';
import { GatewayGenerator } from './gateway';
import { TerraformGenerator } from './terraform';
import { DeploymentType } from '../annotations';

/**
 * Main build generator that orchestrates all artifact generation
 */
export class Generator {
  private config: GeneratorConfig;
  private logger: Logger;
  private functionGenerator: FunctionGenerator;
  private containerGenerator: ContainerGenerator;
  private gatewayGenerator: GatewayGenerator;
  private terraformGenerator: TerraformGenerator;

  constructor(config: GeneratorConfig, logger: Logger) {
    this.config = {
      ...config,
      region: config.region || 'us-central1',
      environment: config.environment || 'dev',
      cleanBuildDir: config.cleanBuildDir ?? false
    };
    this.logger = logger;

    // Initialize sub-generators
    this.functionGenerator = new FunctionGenerator(
      config.handlers,
      path.join(config.outputDir, 'functions'),
      config.moduleName,
      logger
    );

    this.containerGenerator = new ContainerGenerator(
      config.handlers,
      path.join(config.outputDir, 'containers'),
      config.moduleName,
      logger
    );

    this.gatewayGenerator = new GatewayGenerator(
      config.handlers,
      path.join(config.outputDir, 'gateway'),
      config.moduleName,
      config.projectId,
      this.config.region!,
      logger
    );

    this.terraformGenerator = new TerraformGenerator(
      config.handlers,
      path.join(config.outputDir, 'terraform'),
      config.moduleName,
      config.projectId,
      this.config.region!,
      logger
    );
  }

  /**
   * Generate all deployment artifacts
   */
  async generate(): Promise<void> {
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
    const functionCount = this.config.handlers.filter(
      h => h.deploymentType === DeploymentType.Function
    ).length;
    const containerCount = this.config.handlers.filter(
      h => h.deploymentType === DeploymentType.Container
    ).length;

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
  private async cleanOutputDir(): Promise<void> {
    try {
      await fs.promises.rm(this.config.outputDir, { recursive: true, force: true });
      this.logger.info('Cleaned build directory', { dir: this.config.outputDir });
    } catch (error) {
      this.logger.warn('Failed to clean build directory', { error });
    }
  }

  /**
   * Generate functions only
   */
  async generateFunctions(): Promise<void> {
    await this.functionGenerator.generate();
  }

  /**
   * Generate containers only
   */
  async generateContainers(): Promise<void> {
    await this.containerGenerator.generate();
  }

  /**
   * Generate gateway only
   */
  async generateGateway(): Promise<void> {
    await this.gatewayGenerator.generate();
  }

  /**
   * Generate terraform only
   */
  async generateTerraform(): Promise<void> {
    await this.terraformGenerator.generate();
  }
}
