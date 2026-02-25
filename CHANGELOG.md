# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Template-based error page architecture using Go's `embed` package
  - Error pages now stored in `templates/` directory as separate HTML files
  - Non-developers can customize error pages without touching Go code
  - Templates embedded at compile-time for zero runtime overhead
- Modern, responsive error page designs
  - Material Design-inspired styling
  - Mobile-friendly responsive layouts
  - Distinct visual themes for 4xx (orange) and 5xx (red) errors
  - Action buttons (Go Back, Retry, Return Home)
  - Emoji icons for visual feedback
- Comprehensive documentation
  - Detailed README with build instructions and customization guide
  - Template customization guide in `templates/README.md`
  - Examples for common customizations
- Support for all 4xx and 5xx HTTP error codes
  - Previously only handled 500 errors
  - Now intercepts all client (4xx) and server (5xx) errors
- Build automation with Makefile
  - `make build` - Build WASM plugin locally
  - `make build-docker` - Build Docker image
  - `make clean` - Clean build artifacts
  - `make help` - Show all available commands
- Automatic version tracking
  - Version embedded at compile-time using git commit SHA
  - Configurable via VERSION environment variable or build arg
  - Displayed in plugin initialization logs

### Changed
- **Code Refactoring**: Reorganized codebase into modular packages
  - Created `internal/errorpages` package for error detection and page handling
  - Simplified `main.go` to focus on WASM/Envoy integration
  - Improved separation of concerns and testability
- Refactored code structure for better maintainability
  - Renamed context types for clarity (`httpContext` vs `errorPageContext`)
  - Removed unused/commented code
  - Improved function naming and organization
- Enhanced logging
  - Uses appropriate log levels (Info, Debug, Warn, Error)
  - Less verbose output for normal operations
  - Better error messages and context

### Improved
- Error detection logic
  - More robust status code parsing
  - Supports all 3-digit 4xx and 5xx codes
  - Easy to extend for status-specific pages
- Build process
  - Dockerfile optimized for version injection
  - Multi-stage build keeps final image minimal
  - Consistent versioning across build methods

### Removed
- **X-App-Id Header Injection**: Removed domain matching and X-App-Id header generation functionality
  - Removed `internal/config` package
  - Removed `internal/matcher` package for domain pattern matching
  - Removed `internal/headers` package for header injection
  - Simplified `config.yaml` (now reserved for future use)
  - Removed `CONFIG.md` documentation file
  - Plugin now focuses solely on error page interception

## [0.1.0] - Initial Release

### Added
- Initial WASM plugin implementation
- Basic error page replacement for 500 errors
- Docker build support
- Envoy configuration example