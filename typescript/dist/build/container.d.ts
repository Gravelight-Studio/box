import { Handler } from '../annotations';
import { Logger } from './types';
/**
 * Generator for Cloud Run containers
 */
export declare class ContainerGenerator {
    private handlers;
    private outputDir;
    private moduleName;
    private logger;
    constructor(handlers: Handler[], outputDir: string, moduleName: string, logger: Logger);
    /**
     * Generate all container services
     */
    generate(): Promise<void>;
    /**
     * Group handlers by package name
     */
    private groupByService;
    /**
     * Generate a container service
     */
    private generateService;
    /**
     * Generate Dockerfile
     */
    private generateDockerfile;
    /**
     * Generate package.json
     */
    private generatePackageJson;
    /**
     * Generate server.js
     */
    private generateServerJs;
    /**
     * Generate handler code
     */
    private generateHandlerCode;
    /**
     * Generate .dockerignore
     */
    private generateDockerIgnore;
    /**
     * Generate cloudbuild.yaml
     */
    private generateCloudBuild;
}
//# sourceMappingURL=container.d.ts.map