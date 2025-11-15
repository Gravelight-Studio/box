import { GeneratorConfig, Logger } from './types';
/**
 * Main build generator that orchestrates all artifact generation
 */
export declare class Generator {
    private config;
    private logger;
    private functionGenerator;
    private containerGenerator;
    private gatewayGenerator;
    private terraformGenerator;
    constructor(config: GeneratorConfig, logger: Logger);
    /**
     * Generate all deployment artifacts
     */
    generate(): Promise<void>;
    /**
     * Clean output directory
     */
    private cleanOutputDir;
    /**
     * Generate functions only
     */
    generateFunctions(): Promise<void>;
    /**
     * Generate containers only
     */
    generateContainers(): Promise<void>;
    /**
     * Generate gateway only
     */
    generateGateway(): Promise<void>;
    /**
     * Generate terraform only
     */
    generateTerraform(): Promise<void>;
}
//# sourceMappingURL=generator.d.ts.map