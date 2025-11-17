# Box Scripts

Utility scripts for maintaining the Box repository.

## Version Management

The repository uses a single `VERSION` file at the root as the source of truth for all version numbers.

### How it works

1. **VERSION file** - Contains the current version (e.g., `0.1.8`)
2. **CLI** - Version injected at build time via `-ldflags` from VERSION file
3. **TypeScript** - Runs `npm run sync-version` before publishing to update `package.json`
4. **Templates** - Use `{{.Version}}` template variable from the CLI

### Release Process

1. **Update VERSION file:**
   ```bash
   echo "0.2.0" > VERSION
   ```

2. **Rebuild CLI:**

   **Windows:**
   ```bash
   cd cli && build.bat
   ```

   **Unix/macOS:**
   ```bash
   cd cli && ./build.sh
   ```

   The build script automatically reads the VERSION file and injects it via `-ldflags`.

3. **Test version:**
   ```bash
   ./bin/box.exe version  # Should show 0.2.0
   ```

4. **Commit and tag:**
   ```bash
   git add .
   git commit -m "Bump version to v0.2.0"
   git tag -a v0.2.0 -m "Release v0.2.0"
   git push && git push origin v0.2.0
   ```

The GitHub Actions workflow will automatically build release binaries.

**Note:** The TypeScript package version is automatically synced during `npm publish` via the `prepublishOnly` hook.
