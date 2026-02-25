# Use standard Go 1.25 for building WASM
FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY . .

# Version can be overridden at build time with --build-arg VERSION=x.y.z
# If not provided, defaults to 'dev'
ARG VERSION=dev

# Build the WASM binary using the new Go WASIP1 target
# We use -buildmode=c-shared as recommended by the SDK
RUN env GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -ldflags "-X main.version=${VERSION}" -o main.wasm main.go

# Use a minimal base image for the OCI artifact
FROM scratch

# Envoy looks for the wasm file
COPY --from=builder /src/main.wasm ./plugin.wasm
