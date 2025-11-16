// Main entry point for Box framework
export * from './annotations';
export * from './router';
export * from './build';

// Explicitly re-export Request and Response types for convenience
export type { Request, Response } from './router';
