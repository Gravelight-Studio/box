import * as fs from 'fs';
import * as path from 'path';
import {
  Handler,
  ParsedAnnotations,
  ParseError,
  DeploymentType,
  AuthType,
  Route,
  CORSConfig,
  RateLimitConfig,
  TimeoutConfig,
  MemoryConfig,
  ConcurrencyConfig
} from './types';

/**
 * Parser for extracting Box annotations from TypeScript/JavaScript files
 */
export class Parser {
  /**
   * Parse all handlers in a directory
   */
  async parseDirectory(dir: string): Promise<ParsedAnnotations> {
    const result: ParsedAnnotations = {
      handlers: [],
      errors: []
    };

    await this.walkDirectory(dir, result);
    return result;
  }

  /**
   * Parse a single file for handlers
   */
  async parseFile(filePath: string): Promise<ParsedAnnotations> {
    const result: ParsedAnnotations = {
      handlers: [],
      errors: []
    };

    try {
      const content = await fs.promises.readFile(filePath, 'utf-8');
      const handlers = this.extractHandlers(content, filePath);
      result.handlers.push(...handlers);
    } catch (error) {
      result.errors.push({
        filePath,
        lineNumber: 0,
        message: `Failed to read file: ${error}`,
        annotation: ''
      });
    }

    return result;
  }

  /**
   * Recursively walk directory and parse files
   */
  private async walkDirectory(dir: string, result: ParsedAnnotations): Promise<void> {
    const entries = await fs.promises.readdir(dir, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);

      if (entry.isDirectory()) {
        // Skip node_modules and hidden directories
        if (entry.name === 'node_modules' || entry.name.startsWith('.')) {
          continue;
        }
        await this.walkDirectory(fullPath, result);
      } else if (entry.isFile()) {
        // Only parse TypeScript and JavaScript files
        if (entry.name.endsWith('.ts') || entry.name.endsWith('.js')) {
          // Skip test files and type definition files
          if (!entry.name.endsWith('.test.ts') &&
              !entry.name.endsWith('.test.js') &&
              !entry.name.endsWith('.d.ts')) {
            const fileResult = await this.parseFile(fullPath);
            result.handlers.push(...fileResult.handlers);
            result.errors.push(...fileResult.errors);
          }
        }
      }
    }
  }

  /**
   * Extract handlers from file content
   */
  private extractHandlers(content: string, filePath: string): Handler[] {
    const handlers: Handler[] = [];
    const lines = content.split('\n');

    for (let i = 0; i < lines.length; i++) {
      // Look for function declarations or const assignments
      // Handles: function foo(), export function foo(), const foo = ..., export const foo: Type = ...
      const functionMatch = lines[i].match(/(?:export\s+)?(?:async\s+)?function\s+(\w+)|(?:export\s+)?const\s+(\w+)\s*(?::\s*\w+\s*)?=/);

      if (functionMatch) {
        const functionName = functionMatch[1] || functionMatch[2];

        // Look backwards for annotations
        const annotations = this.extractAnnotationsAbove(lines, i);

        if (annotations.deploymentType) {
          const handler = this.buildHandler(
            functionName,
            filePath,
            annotations,
            i + 1
          );
          if (handler) {
            handlers.push(handler);
          }
        }
      }
    }

    return handlers;
  }

  /**
   * Extract annotations from comments above a function
   */
  private extractAnnotationsAbove(lines: string[], functionLineIndex: number): any {
    const annotations: any = {};

    // Look backwards from function line
    for (let i = functionLineIndex - 1; i >= 0; i--) {
      const line = lines[i].trim();

      // Stop at empty lines or non-comment lines
      if (!line || (!line.startsWith('//') && !line.startsWith('*') && !line.startsWith('/*'))) {
        break;
      }

      // Extract annotation
      const annotationMatch = line.match(/@box:(\w+)\s*(.*)/);
      if (annotationMatch) {
        const [, key, value] = annotationMatch;
        this.parseAnnotation(key, value.trim(), annotations);
      }
    }

    return annotations;
  }

  /**
   * Parse individual annotation
   */
  private parseAnnotation(key: string, value: string, annotations: any): void {
    switch (key) {
      case 'function':
        annotations.deploymentType = DeploymentType.Function;
        break;

      case 'container':
        annotations.deploymentType = DeploymentType.Container;
        break;

      case 'path':
        const pathMatch = value.match(/(\w+)\s+(.+)/);
        if (pathMatch) {
          annotations.route = {
            method: pathMatch[1].toUpperCase(),
            path: pathMatch[2]
          };
        }
        break;

      case 'auth':
        if (value === 'none') annotations.auth = AuthType.None;
        else if (value === 'optional') annotations.auth = AuthType.Optional;
        else if (value === 'required') annotations.auth = AuthType.Required;
        break;

      case 'cors':
        const originsMatch = value.match(/origins=(.+)/);
        if (originsMatch) {
          const origins = originsMatch[1].split(',').map(o => o.trim());
          annotations.cors = { origins };
        }
        break;

      case 'ratelimit':
        const rateLimitMatch = value.match(/(\d+)\/(second|minute|hour|day)/);
        if (rateLimitMatch) {
          const requests = parseInt(rateLimitMatch[1]);
          const period = rateLimitMatch[2] as 'second' | 'minute' | 'hour' | 'day';
          const windowMs = this.periodToMs(period);
          annotations.rateLimit = { requests, period, windowMs };
        }
        break;

      case 'timeout':
        const timeoutMatch = value.match(/(\d+)(s|m|h)/);
        if (timeoutMatch) {
          const duration = parseInt(timeoutMatch[1]);
          const unit = timeoutMatch[2];
          annotations.timeout = {
            duration: this.durationToMs(duration, unit)
          };
        }
        break;

      case 'memory':
        const memoryMatch = value.match(/(\d+)MB/);
        if (memoryMatch) {
          annotations.memory = {
            size: parseInt(memoryMatch[1])
          };
        }
        break;

      case 'concurrency':
        const concurrencyValue = parseInt(value);
        if (!isNaN(concurrencyValue)) {
          annotations.concurrency = {
            max: concurrencyValue
          };
        }
        break;
    }
  }

  /**
   * Build handler object from annotations
   */
  private buildHandler(
    functionName: string,
    filePath: string,
    annotations: any,
    lineNumber: number
  ): Handler | null {
    if (!annotations.route) {
      return null; // Must have a route
    }

    const packageName = path.basename(path.dirname(filePath));

    return {
      packageName,
      functionName,
      filePath,
      deploymentType: annotations.deploymentType,
      route: annotations.route,
      auth: annotations.auth || AuthType.None,
      cors: annotations.cors,
      rateLimit: annotations.rateLimit,
      timeout: annotations.timeout,
      memory: annotations.memory,
      concurrency: annotations.concurrency
    };
  }

  /**
   * Convert period to milliseconds
   */
  private periodToMs(period: 'second' | 'minute' | 'hour' | 'day'): number {
    switch (period) {
      case 'second': return 1000;
      case 'minute': return 60 * 1000;
      case 'hour': return 60 * 60 * 1000;
      case 'day': return 24 * 60 * 60 * 1000;
    }
  }

  /**
   * Convert duration to milliseconds
   */
  private durationToMs(value: number, unit: string): number {
    switch (unit) {
      case 's': return value * 1000;
      case 'm': return value * 60 * 1000;
      case 'h': return value * 60 * 60 * 1000;
      default: return value * 1000;
    }
  }
}
