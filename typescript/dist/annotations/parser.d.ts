import { ParsedAnnotations } from './types';
/**
 * Parser for extracting Box annotations from TypeScript/JavaScript files
 */
export declare class Parser {
    /**
     * Parse all handlers in a directory
     */
    parseDirectory(dir: string): Promise<ParsedAnnotations>;
    /**
     * Parse a single file for handlers
     */
    parseFile(filePath: string): Promise<ParsedAnnotations>;
    /**
     * Recursively walk directory and parse files
     */
    private walkDirectory;
    /**
     * Extract handlers from file content
     */
    private extractHandlers;
    /**
     * Extract annotations from comments above a function
     */
    private extractAnnotations;
    /**
     * Parse individual annotation
     */
    private parseAnnotation;
    /**
     * Build handler object from annotations
     */
    private buildHandler;
    /**
     * Convert period to milliseconds
     */
    private periodToMs;
    /**
     * Convert duration to milliseconds
     */
    private durationToMs;
}
//# sourceMappingURL=parser.d.ts.map