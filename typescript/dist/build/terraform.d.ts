import { Handler } from '../annotations';
import { Logger } from './types';
/**
 * Generator for Terraform infrastructure as code
 */
export declare class TerraformGenerator {
    private handlers;
    private outputDir;
    private moduleName;
    private projectId;
    private region;
    private logger;
    constructor(handlers: Handler[], outputDir: string, moduleName: string, projectId: string, region: string, logger: Logger);
    /**
     * Generate Terraform configuration
     */
    generate(): Promise<void>;
    /**
     * Generate main.tf
     */
    private generateMain;
    /**
     * Generate variables.tf
     */
    private generateVariables;
    /**
     * Generate outputs.tf
     */
    private generateOutputs;
    /**
     * Generate functions module
     */
    private generateFunctionsModule;
    /**
     * Generate Terraform resource for a function
     */
    private generateFunctionResource;
    /**
     * Generate containers module
     */
    private generateContainersModule;
    /**
     * Generate Cloud Run service resource
     */
    private generateContainerService;
    /**
     * Get normalized function name
     */
    private getFunctionName;
}
//# sourceMappingURL=terraform.d.ts.map