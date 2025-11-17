#!/usr/bin/env node

/**
 * Sync version from root VERSION file to package.json
 * This runs automatically before publishing to npm
 */

const fs = require('fs');
const path = require('path');

const versionFile = path.join(__dirname, '../../VERSION');
const packageFile = path.join(__dirname, '../package.json');

// Read version
const version = fs.readFileSync(versionFile, 'utf8').trim();

// Read and update package.json
const pkg = JSON.parse(fs.readFileSync(packageFile, 'utf8'));
pkg.version = version;

// Write back
fs.writeFileSync(packageFile, JSON.stringify(pkg, null, 2) + '\n');

console.log(`✓ Synced version to ${version}`);
