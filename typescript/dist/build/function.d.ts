import { Handler } from '../annotations';
import { Logger } from './types';
/**
 * Generator for Cloud Functions
 */
export declare class FunctionGenerator {
    private handlers;
    private outputDir;
    private moduleName;
    private logger;
    constructor(handlers: Handler[], outputDir: string, moduleName: string, logger: Logger);
    /**
     * Generate all function packages
     */
    generate(): Promise<void>;
    /**
     * Generate a single function package
     */
    private generateFunction;
    /**
     * Generate package.json for function
     */
    private generatePackageJson;
    /**
     * Generate index.js entry point
     */
    private generateIndexJs;
    /**
     * Generate function.yaml configuration
     */
    private generateFunctionYaml;
    /**
     * Generate .gcloudignore
     */
    private generateGcloudIgnore;
    /**
     * Get normalized function name
     */
    private getFunctionName;
}
//# sourceMappingURL=function.d.ts.map