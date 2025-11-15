import { Handler } from '../annotations';
import { Logger } from './types';
/**
 * Generator for API Gateway configuration
 */
export declare class GatewayGenerator {
    private handlers;
    private outputDir;
    private moduleName;
    private projectId;
    private region;
    private logger;
    constructor(handlers: Handler[], outputDir: string, moduleName: string, projectId: string, region: string, logger: Logger);
    /**
     * Generate API Gateway configuration
     */
    generate(): Promise<void>;
    /**
     * Generate OpenAPI 3.0 specification
     */
    private generateOpenAPISpec;
    /**
     * Generate paths for OpenAPI spec
     */
    private generatePaths;
    /**
     * Generate operation for a handler
     */
    private generateOperation;
    /**
     * Generate gateway-config.yaml
     */
    private generateGatewayConfig;
}
//# sourceMappingURL=gateway.d.ts.map