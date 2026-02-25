# syntax=docker/dockerfile:1
# Dockerfile for Secrets - Secure secrets manager with envelope encryption
#
# This multi-stage build produces a minimal, secure container image based on
# Google Distroless for reduced attack surface and improved security posture.
#
# Key Features:
#   - Multi-architecture support (linux/amd64, linux/arm64)
#   - Distroless base image (no shell, package manager, or system utilities)
#   - SHA256 digest pinning for immutable builds
#   - Non-root user execution (UID 65532)
#   - Static binary with no runtime dependencies
#   - Build-time version injection via ldflags
#   - Comprehensive OCI labels for SBOM and security scanning
#
# Build Command:
#   docker build -t allisson/secrets:latest \
#     --build-arg VERSION=v0.12.0 \
#     --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
#     --build-arg COMMIT_SHA=$(git rev-parse HEAD) .
#
# Multi-Architecture Build:
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     -t allisson/secrets:latest .
#
# Documentation:
#   - Getting Started: docs/getting-started/docker.md
#   - Security Guide: docs/operations/security/container-security.md
#   - Health Checks: docs/operations/observability/health-checks.md

# ==============================================================================
# Build Arguments (Global)
# ==============================================================================

# Go version for builder stage (matches go.mod)
ARG GO_VERSION=1.25.5

# ==============================================================================
# Stage 1: Builder
# ==============================================================================
# Purpose: Compile the Go application into a static binary
# Base: golang:1.25.5-trixie (Debian 13 Trixie for glibc version consistency)
# Output: /app/bin/app (static binary with version metadata injected)

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-trixie AS builder

# Build arguments for cross-compilation and versioning
# These are automatically provided by Docker buildx for multi-arch builds
ARG TARGETOS      # Target OS (e.g., linux)
ARG TARGETARCH    # Target architecture (e.g., amd64, arm64)

# Version metadata (injected at build time via ldflags)
ARG VERSION=dev         # Application version (e.g., v0.10.0, or "dev" for local builds)
ARG BUILD_DATE          # ISO 8601 build timestamp (e.g., 2026-02-21T10:30:00Z)
ARG COMMIT_SHA          # Full git commit hash (e.g., abc123def456...)

# Set working directory for build
WORKDIR /app

# Copy dependency files first for better Docker layer caching
# If go.mod/go.sum haven't changed, this layer is reused
COPY go.mod go.sum ./

# Download and verify dependencies
# This layer is cached and only re-run if go.mod/go.sum change
RUN go mod download && go mod verify

# Copy application source code
# This layer changes frequently, so we do it after dependency download
COPY . .

# Build static binary with version injection
# Flags explained:
#   CGO_ENABLED=0           - Disable CGO for fully static binary (no libc dependency)
#   GOOS=${TARGETOS}        - Target operating system (linux)
#   GOARCH=${TARGETARCH}    - Target architecture (amd64, arm64, etc.)
#   -a                      - Force rebuild of all packages
#   -installsuffix cgo      - Add suffix to package directory (avoids conflicts)
#   -ldflags="-w -s ..."    - Linker flags:
#     -w                    - Omit DWARF symbol table (reduces binary size)
#     -s                    - Omit symbol table and debug info (reduces binary size)
#     -X main.version       - Inject version string into main.version variable
#     -X main.buildDate     - Inject build timestamp into main.buildDate variable
#     -X main.commitSHA     - Inject git commit hash into main.commitSHA variable
#   -o /app/bin/app         - Output binary path
#   ./cmd/app               - Main package path
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -installsuffix cgo \
    -ldflags="-w -s \
    -X main.version=${VERSION} \
    -X main.buildDate=${BUILD_DATE} \
    -X main.commitSHA=${COMMIT_SHA}" \
    -o /app/bin/app ./cmd/app

# ==============================================================================
# Stage 2: Final Runtime Image
# ==============================================================================
# Purpose: Minimal runtime environment with only the compiled binary
# Base: gcr.io/distroless/static-debian13 (Google Distroless - Debian 13 Trixie)
# Size: ~2-3 MB (base) + ~15-20 MB (binary) = ~17-23 MB total
# Security: No shell, package manager, or system utilities (minimal attack surface)

# Distroless static image (Debian 13 Trixie) pinned by SHA256 digest
# Digest pinning ensures immutable builds and prevents supply chain attacks
# To update digest: docker pull gcr.io/distroless/static-debian13:latest && docker inspect
FROM gcr.io/distroless/static-debian13@sha256:d90359c7a3ad67b3c11ca44fd5f3f5208cbef546f2e692b0dc3410a869de46bf

# Distroless static image (Debian 13 Trixie) pinned by SHA256 digest
# Digest pinning ensures immutable builds and prevents supply chain attacks
# To update digest: docker pull gcr.io/distroless/static-debian13:latest && docker inspect
FROM gcr.io/distroless/static-debian13@sha256:d90359c7a3ad67b3c11ca44fd5f3f5208cbef546f2e692b0dc3410a869de46bf

# ==============================================================================
# OCI Labels (Image Metadata)
# ==============================================================================
# Purpose: Provide metadata for security scanning, SBOM generation, and registries
# Follows Open Container Initiative (OCI) Image Format Specification
# Reference: https://github.com/opencontainers/image-spec/blob/main/annotations.md
# Documentation: docs/operations/deployment/oci-labels.md

# Basic image information
LABEL org.opencontainers.image.title="Secrets"
LABEL org.opencontainers.image.description="Lightweight secrets manager with envelope encryption, transit encryption, and audit logs"
LABEL org.opencontainers.image.url="https://github.com/allisson/secrets"
LABEL org.opencontainers.image.source="https://github.com/allisson/secrets"
LABEL org.opencontainers.image.documentation="https://github.com/allisson/secrets/tree/main/docs"

# Version and build metadata (injected from build args)
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"

# License and authorship
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.vendor="Allisson Azevedo"
LABEL org.opencontainers.image.authors="Allisson Azevedo <allisson@gmail.com>"

# Base image metadata (for security scanning and provenance)
LABEL org.opencontainers.image.base.name="gcr.io/distroless/static-debian13"
LABEL org.opencontainers.image.base.digest="sha256:d90359c7a3ad67b3c11ca44fd5f3f5208cbef546f2e692b0dc3410a869de46bf"

# ==============================================================================
# Runtime Configuration
# ==============================================================================

# Copy compiled binary from builder stage
# Source: /app/bin/app (builder stage)
# Destination: /app (final image root)
COPY --from=builder /app/bin/app /app

# Expose HTTP API port
# Note: EXPOSE is documentation only; does not actually publish the port
# Use -p 8080:8080 when running the container to bind the port
EXPOSE 8080

# ==============================================================================
# Security Configuration
# ==============================================================================

# Run as non-root user for enhanced security
# User: nonroot (UID 65532, GID 65532)
# This is the default user in distroless/static, but we make it explicit
# Benefits:
#   - Prevents privilege escalation attacks
#   - Limits filesystem access to writable directories only
#   - Required for some security policies (PodSecurityPolicy, PodSecurityStandards)
#
# ⚠️ BREAKING CHANGE (v0.10.0): Previous versions ran as root (UID 0)
# Volume permissions may need adjustment when upgrading from v0.9.0
# See: docs/operations/troubleshooting/volume-permissions.md
USER nonroot:nonroot

# ==============================================================================
# Health Check Configuration
# ==============================================================================
# The application exposes two HTTP endpoints for health monitoring:
#
# Endpoints:
#   GET /health - Liveness probe (basic health check, < 10ms response time)
#     Purpose: Detect if application is running and responsive
#     Returns: 200 OK with {"status":"healthy"}
#     Use: Kubernetes livenessProbe, restart triggers
#
#   GET /ready - Readiness probe (database connectivity check, < 100ms response time)
#     Purpose: Detect if application can handle requests (includes DB check)
#     Returns: 200 OK with {"status":"ready","database":"ok"}
#     Use: Kubernetes readinessProbe, load balancer target health
#
# ⚠️ Docker HEALTHCHECK Not Supported:
#   Distroless images have no shell (/bin/sh) or utilities (curl, wget)
#   Docker's built-in HEALTHCHECK directive does NOT work:
#
#   ❌ This will fail:
#     HEALTHCHECK --interval=30s --timeout=3s \
#       CMD curl -f http://localhost:8080/health || exit 1
#
# ✅ Recommended Solutions:
#
# 1. Kubernetes (native HTTP probes):
#    livenessProbe:
#      httpGet:
#        path: /health
#        port: 8080
#      initialDelaySeconds: 10
#      periodSeconds: 30
#      timeoutSeconds: 3
#      failureThreshold: 3
#
#    readinessProbe:
#      httpGet:
#        path: /ready
#        port: 8080
#      initialDelaySeconds: 5
#      periodSeconds: 10
#      timeoutSeconds: 3
#      failureThreshold: 2
#
# 2. Docker Compose (healthcheck sidecar):
#    services:
#      secrets-api:
#        image: allisson/secrets:latest
#
#      healthcheck:
#        image: curlimages/curl:latest
#        command: >
#          sh -c 'while true; do
#            curl -f http://secrets-api:8080/health || exit 1;
#            sleep 30;
#          done'
#
# 3. Production (external monitoring):
#    - Prometheus Blackbox Exporter
#    - Datadog Synthetic Monitoring
#    - Uptime Kuma
#    - AWS ALB Target Health Checks
#    - Google Cloud Run Health Checks
#
# For complete health check documentation and examples:
#   docs/operations/observability/health-checks.md
#   docs/getting-started/docker.md
#   docs/operations/security/container-security.md

# ==============================================================================
# Runtime Security Notes
# ==============================================================================
# This image is designed for secure production deployments:
#
# 1. Non-Root User:
#    - Runs as UID 65532 (nonroot:nonroot)
#    - Cannot write to most filesystem locations
#    - Prevents privilege escalation
#
# 2. Static Binary:
#    - No libc or dynamic library dependencies
#    - Self-contained executable
#    - Minimal runtime requirements
#
# 3. Read-Only Filesystem Support:
#    - Can run with --read-only flag
#    - No filesystem writes needed at runtime
#    - Example: docker run --read-only -p 8080:8080 allisson/secrets
#
# 4. Minimal Attack Surface:
#    - No shell (no /bin/sh, /bin/bash)
#    - No package manager (no apt, apk, yum)
#    - No system utilities (no curl, wget, nc)
#    - Only the application binary and CA certificates
#
# 5. Immutable Base Image:
#    - SHA256 digest pinning prevents tampering
#    - Regular security patches from Google Distroless
#    - Automated vulnerability scanning via Trivy/Grype
#
# 6. Included Components:
#    - CA certificates (from distroless base)
#    - Timezone data (from distroless base)
#    - Static application binary (compiled in builder stage)
#
# 7. Security Scanning:
#    - OCI labels enable SBOM generation
#    - Compatible with Trivy, Grype, Snyk, Anchore
#    - Scan command: trivy image allisson/secrets:latest
#
# For complete security hardening guide:
#   docs/operations/security/container-security.md
#   docs/operations/security/hardening.md

# ==============================================================================
# Container Entrypoint and Command
# ==============================================================================

# Entrypoint: Path to the application binary
# This is the main executable that runs when the container starts
ENTRYPOINT ["/app"]

# Default command: Start the HTTP API server
# This can be overridden when running the container
# Examples:
#   docker run allisson/secrets server              # Default (HTTP API)
#   docker run allisson/secrets migrate              # Run database migrations
#   docker run allisson/secrets create-kek           # Create encryption key
#   docker run allisson/secrets --version            # Show version info
#   docker run allisson/secrets --help               # Show help
CMD ["server"]
