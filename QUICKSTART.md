# Quick Start Guide

Get up and running with the WASM Error Pages plugin in under 5 minutes.

## Prerequisites

- Docker and Docker Compose
- Make (optional, but recommended)

## 1. Start the Development Environment

```bash
# Clone and enter the directory
cd envoy-wasm-error-pages

# Start everything (builds WASM plugin, debug backend, and Envoy)
make up
```

Wait for the services to start (should take 5-10 seconds). The plugin will be built with the current git SHA as the version.

You'll see:
```
Starting development environment (version: abc1234)...
✔ Container wasm-builder  Exited    
✔ Container envoy-wasm    Started
```

## 2. Test the Error Pages

**In your browser** (best way to see styling):
- http://localhost:10000/500 - Server Error page (red theme)
- http://localhost:10000/404 - Client Error page (orange theme)
- http://localhost:10000/200 - Normal response (no error page)

**From command line:**
```bash
make test-errors
```

## 3. Customize the Error Pages

Edit the HTML templates (no Go knowledge needed):
```bash
# Edit 4xx error page (400, 404, etc.)
open templates/error-4xx.html

# Edit 5xx error page (500, 503, etc.)
open templates/error-5xx.html
```

Change text, colors, or styling - it's just HTML/CSS!

## 4. See Your Changes

```bash
# Rebuild and restart
make restart

# Test again
open http://localhost:10000/500

# The version is displayed in the footer of error pages
```

## 5. Stop the Environment

```bash
make down
```

## Common Commands

| Command | Description |
|---------|-------------|
| `make up` | Start development environment |
| `make down` | Stop and clean up |
| `make restart` | Rebuild and restart |
| `make test-errors` | Test all error codes |
| `make logs` | View all logs |
| `make build` | Build WASM locally |
| `make build-docker` | Build Docker image |
| `make help` | Show all commands |

## Quick Customization Examples

### Change the Error Message

Edit `templates/error-5xx.html`:
```html
<p class="error-message">
    We're experiencing technical difficulties.  <!-- Change this! -->
</p>
```

### Change the Color Scheme

Edit the `<style>` section:
```css
.error-container {
    border-top: 4px solid #f44336;  /* Change to your brand color */
}

h1 {
    color: #c62828;  /* Change to match */
}
```

### Set a Custom Version

```bash
# Use a specific version instead of git SHA
VERSION=1.0.0 make up
```

The version will appear in the footer of error pages.

### Add Your Logo

Add after `<div class="error-container">`:
```html
<img src="data:image/svg+xml;base64,..." alt="Logo" 
     style="width: 100px; margin: 0 auto 20px; display: block;">
```

## Test Backend Endpoints

The http-debug service responds to:
- `/200` - OK
- `/400` - Bad Request
- `/401` - Unauthorized
- `/403` - Forbidden
- `/404` - Not Found
- `/500` - Internal Server Error
- `/502` - Bad Gateway
- `/503` - Service Unavailable
- `/504` - Gateway Timeout

All accessed via: `http://localhost:10000/<endpoint>`

## Debugging

**View Envoy logs:**
```bash
docker-compose logs -f envoy
```

**Access Envoy admin interface:**
http://localhost:9901

**Check if WASM is loaded:**
```bash
docker-compose logs envoy | grep "WASM Error Pages Plugin initialized"
```

**Rebuild from scratch:**
```bash
make down
docker-compose build --no-cache
make up
```

## Production Deployment

Once you're happy with your error pages:

```bash
# Build production image with version
make build-docker VERSION=1.0.0

# Tag and push to your registry
docker tag envoy-wasm-error-pages:1.0.0 your-registry/error-pages:1.0.0
docker push your-registry/error-pages:1.0.0

# Extract WASM file for EnvoyPatchPolicy
docker run --rm --entrypoint cat envoy-wasm-error-pages:1.0.0 /plugin.wasm > plugin.wasm
```

## Need More Help?

- Full documentation: [README.md](README.md)
- Template customization guide: [templates/README.md](templates/README.md)
- Detailed customization examples: [CUSTOMIZING.md](CUSTOMIZING.md)
- Recent changes: [CHANGELOG.md](CHANGELOG.md)

## Troubleshooting

**Problem:** Changes don't appear after editing templates
- **Fix:** Run `make down && make up` to rebuild with new templates

**Problem:** Port 10000 already in use
- **Fix:** Stop other services using that port or edit `docker-compose.yaml`

**Problem:** WASM plugin not loading
- **Fix:** Check build logs with `docker-compose logs wasm-builder` and Envoy logs with `docker-compose logs envoy`

**Problem:** Envoy fails to start
- **Fix:** Ensure WASM builder completed successfully with `docker-compose logs wasm-builder`

---

**Quick tip:** Keep this file open while developing - it has everything you need! 🚀