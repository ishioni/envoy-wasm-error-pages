# Envoy WASM Error Pages

This project provides an Envoy WASM extension written in Go that intercepts backend error responses (4xx and 5xx status codes) and replaces them with custom HTML error pages.

## Features

- **Automatic Error Interception**: Detects and handles all 4xx and 5xx HTTP status codes
- **Custom Error Pages**: Provides distinct error pages for client errors (4xx) and server errors (5xx)
- **Template-Based Design**: HTML error pages are stored in separate files for easy customization without Go knowledge
- **Version Tracking**: Automatically embeds the git commit SHA into the plugin for easy version identification
- **Lightweight**: Compiled to WASM for minimal overhead

## Prerequisites

- Docker
- Go 1.25+ (for local development)
- Git (for automatic version tagging)
- Make (optional, for convenience commands)

## Building

### Using Make (Recommended)

```bash
# Build WASM plugin locally (auto-uses git SHA as version)
make build

# Build with specific version
make build VERSION=1.0.0

# Build Docker image (auto-uses git SHA)
make build-docker

# Build Docker image with specific version
make build-docker VERSION=1.0.0

# Show current version
make version

# Clean build artifacts
make clean

# Show all available commands
make help
```

### Using Docker Directly

```bash
# Build with automatic git SHA version
docker build --build-arg VERSION=$(git rev-parse --short HEAD) -t envoy-wasm-error-pages:latest .

# Build with custom version
docker build --build-arg VERSION=1.0.0 -t envoy-wasm-error-pages:1.0.0 .

# Build with default version (dev)
docker build -t envoy-wasm-error-pages:latest .
```

### Using Go Directly

```bash
# Build with git SHA version
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared \
  -ldflags "-X main.version=$(git rev-parse --short HEAD)" \
  -o main.wasm main.go

# Build with custom version
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared \
  -ldflags "-X main.version=1.0.0" \
  -o main.wasm main.go
```

## Local Development

The easiest way to develop and test the plugin is using the provided docker-compose setup:

```bash
# Start the full development environment
make up
# or
docker-compose up --build

# This will start:
# 1. WASM builder - builds the plugin from source
# 2. http-debug - backend server that returns various status codes
# 3. envoy - proxy with the WASM plugin loaded
```

The http-debug service provides endpoints that return different status codes:
- `http://localhost:10000/200` - Returns 200 OK (passes through)
- `http://localhost:10000/400` - Returns 400 (shows 4xx error page)
- `http://localhost:10000/404` - Returns 404 (shows 4xx error page)
- `http://localhost:10000/500` - Returns 500 (shows 5xx error page)
- `http://localhost:10000/503` - Returns 503 (shows 5xx error page)

### Quick Testing

```bash
# Test all error pages
make test-errors

# View in browser (best way to see the full styling)
open http://localhost:10000/500
open http://localhost:10000/404
```

### Development Workflow

1. Edit the HTML templates in `templates/error-4xx.html` or `templates/error-5xx.html`
2. Restart the environment: `make restart` or `docker-compose up --build`
3. Test your changes: `curl http://localhost:10000/500` or visit in browser
4. Check Envoy logs: `make logs` or `docker-compose logs -f envoy`

### Stopping the Environment

```bash
make down
# or
docker-compose down
```

## Extracting the WASM File

To extract the WASM file from the Docker image for standalone use:

```bash
docker run --rm --entrypoint cat envoy-wasm-error-pages:latest /plugin.wasm > plugin.wasm
```

## Running with Envoy

This extension requires Envoy >= 1.33.0.

### Using Envoy Directly

1. Extract the WASM file (see above)
2. Run Envoy with the provided configuration:

```bash
docker run --rm -it \
  -v $(pwd)/envoy.yaml:/etc/envoy/envoy.yaml \
  -v $(pwd)/plugin.wasm:/etc/envoy/plugin.wasm \
  -p 10000:10000 \
  envoyproxy/envoy:v1.33.0 \
  -c /etc/envoy/envoy.yaml
```

## Testing

The docker-compose setup includes a test backend (http-debug) that makes testing easy:

```bash
# Start the development environment
make up

# In another terminal, test the error pages
make test-errors

# Or test manually
curl http://localhost:10000/200   # Normal response (passes through)
curl http://localhost:10000/400   # Client error (custom 4xx page)
curl http://localhost:10000/404   # Not found (custom 4xx page)
curl http://localhost:10000/500   # Server error (custom 5xx page)
curl http://localhost:10000/503   # Service unavailable (custom 5xx page)

# View in browser for full styling
open http://localhost:10000/500
```

You can also access the Envoy admin interface at http://localhost:9901

## How It Works

### Response Processing

1. **Interception**: The plugin monitors all HTTP response headers
2. **Detection**: When it detects a 4xx or 5xx status code, it flags the response for modification
3. **Replacement**: The original response body is replaced with a custom HTML error page
4. **Headers**: Content-Type, Content-Length, and Content-Encoding headers are updated appropriately

### Supported Error Codes

- **4xx (Client Errors)**: 400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, etc.
  - Displays an orange-themed "Client Error" page
- **5xx (Server Errors)**: 500, 501, 502, 503, 504, 505, etc.
  - Displays a red-themed "Server Error" page

## Customization

### Modifying Error Pages

The error pages are stored as HTML files in the `templates/` directory:

- `templates/error-4xx.html` - Page shown for 4xx client errors
- `templates/error-5xx.html` - Page shown for 5xx server errors

You can edit these HTML files directly without any Go knowledge! The files are embedded into the WASM binary at compile time using Go's `embed` package, so after editing the templates, you'll need to rebuild:

```bash
make build
# or
make build-docker
```

The templates include:
- Modern, responsive design
- Gradient backgrounds
- Action buttons (Go Back, Return Home, Retry)
- Mobile-friendly layout
- Customizable colors, text, and styling

### Adding Status-Specific Pages

To handle specific status codes differently, modify the `GetErrorPage()` function in `internal/errorpages/errorpages.go`:

```go
func (h *Handler) GetErrorPage(status string) []byte {
    switch status {
    case "404":
        return error404HTML
    case "500":
        return error500HTML
    case "503":
        return error503HTML
    default:
        if status[0] == '4' {
            return h.error4xxHTML
        }
        return h.error5xxHTML
    }
}
```

### Excluding Certain Error Codes

Modify the `IsErrorStatus()` function in `internal/errorpages/errorpages.go` to exclude specific status codes from being intercepted.

## Development

### Project Structure

```
.
├── main.go                    # Entry point and WASM contexts
├── config.yaml                # Configuration file (reserved for future use)
├── internal/                  # Internal packages
│   └── errorpages/           # Error page handling
│       └── errorpages.go
├── templates/                 # HTML error page templates
│   ├── error-4xx.html        # Client error page (4xx)
│   ├── error-5xx.html        # Server error page (5xx)
│   └── README.md             # Template customization guide
├── Makefile                   # Build automation
├── Dockerfile                 # Multi-stage Docker build
├── Dockerfile.debug           # Debug build configuration
├── docker-compose.yaml        # Local testing setup
├── envoy.yaml                 # Envoy configuration
├── go.mod                     # Go module dependencies
└── README.md                  # This file
```

### Code Structure

**Main Package (`main.go`):**
- `vmContext`: VM-level context for the plugin
- `pluginContext`: Plugin-level context, handles initialization
- `httpContext`: HTTP request/response context, handles error interception
- `error4xxHTML` / `error5xxHTML`: Embedded HTML templates

**Internal Packages:**
- `internal/errorpages`: Error detection and page template management

### Logging

The plugin uses different log levels:
- `LogInfo`: Plugin initialization and error interception events
- `LogDebug`: Detailed status codes and operation confirmations
- `LogWarn`: Non-critical issues
- `LogError`: Critical failures

View logs in real-time:
```bash
docker-compose logs -f envoy
```

## License

Apache License 2.0 - See the license header in source files for details.

## References

- [Proxy-WASM Go SDK](https://github.com/proxy-wasm/proxy-wasm-go-sdk)
- [Envoy WASM Documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/wasm_filter)