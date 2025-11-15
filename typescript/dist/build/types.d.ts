import { Handler } from '../annotations';
/**
 * Configuration for build generator
 */
export interface GeneratorConfig {
    handlers: Handler[];
    outputDir: string;
    moduleName: string;
    projectId: string;
    region?: string;
    environment?: string;
    cleanBuildDir?: boolean;
}
/**
 * Logger interface
 */
export interface Logger {
    info(msg: string, ...args: any[]): void;
    warn(msg: string, ...args: any[]): void;
    error(msg: string, ...args: any[]): void;
    debug(msg: string, ...args: any[]): void;
}
//# sourceMappingURL=types.d.ts.map