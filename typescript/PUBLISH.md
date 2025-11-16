# Publishing @gravelight-studio/box to npm

This document explains how to publish the TypeScript package to npm.

## Prerequisites

1. **npm account** with access to `@gravelight-studio` organization
2. **Authentication**: Run `npm login` and sign in
3. **Version updated** in `package.json` (currently: 0.1.5)

## Pre-publish Checklist

- [x] Version updated in `package.json`
- [x] TypeScript built (`npm run build`)
- [x] `.npmignore` configured
- [x] Package contents verified (`npm pack --dry-run`)
- [ ] Tests passing (`npm test`)
- [ ] README.md up to date
- [ ] CHANGELOG.md updated (if exists)

## Publishing Commands

### 1. Test Package Locally

```bash
cd typescript

# Dry run to see what will be published
npm pack --dry-run

# Create actual tarball to inspect
npm pack
tar -tzf gravelight-studio-box-0.1.5.tgz
```

### 2. Authenticate with npm

```bash
npm login
# Enter credentials for @gravelight-studio
```

### 3. Publish to npm

```bash
# Publish as public package (required for scoped packages on free tier)
npm publish --access public

# Or for beta/preview releases
npm publish --access public --tag beta
```

### 4. Verify Publication

```bash
# Check on npm registry
npm view @gravelight-studio/box

# Install in test project
npm install @gravelight-studio/box@0.1.5
```

## Version Management

Update version using npm's built-in commands:

```bash
# Patch version (0.1.5 -> 0.1.6)
npm version patch

# Minor version (0.1.5 -> 0.2.0)
npm version minor

# Major version (0.1.5 -> 1.0.0)
npm version major
```

These commands automatically:
- Update `package.json`
- Create a git commit
- Create a git tag

## Package Contents

Current package includes:
- **dist/**: Compiled JavaScript + TypeScript definitions + source maps
- **README.md**: Package documentation
- **package.json**: Package metadata

Excluded via `.npmignore`:
- Source TypeScript files (`src/`)
- Tests
- Development configuration
- node_modules

## Troubleshooting

### "403 Forbidden"
- Ensure you're logged in: `npm whoami`
- Verify organization access: Contact @gravelight-studio owner

### "Package name too similar"
- Name `@gravelight-studio/box` is unique
- If blocked, might need to use different name

### "Version already published"
- Bump version: `npm version patch`
- Cannot overwrite existing versions (npm registry is immutable)

## Automated Publishing (Future)

Consider setting up GitHub Actions workflow:

```yaml
name: Publish to npm

on:
  release:
    types: [published]

jobs:
  publish:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
          registry-url: 'https://registry.npmjs.org'
      - run: cd typescript && npm ci
      - run: cd typescript && npm test
      - run: cd typescript && npm publish --access public
        env:
          NODE_AUTH_TOKEN: ${{ secrets.NPM_TOKEN }}
```

## Post-publish

After successful publish:

1. **Verify on npm**: https://www.npmjs.com/package/@gravelight-studio/box
2. **Test installation**: `npm install @gravelight-studio/box`
3. **Update main README.md** with installation instructions
4. **Announce** in release notes
5. **Tag release** in git if not done via `npm version`

## Support

For issues or questions:
- GitHub Issues: https://github.com/gravelight-studio/box/issues
- npm package: https://www.npmjs.com/package/@gravelight-studio/box
