# 🏗️ Multi-Architecture Build Guide

> **Document version**: v0.x
> Last updated: 2026-02-25
> **Audience**: DevOps engineers, release managers, CI/CD maintainers

## Table of Contents

- [Overview](#overview)

- [Quick Start](#quick-start)

- [Docker Buildx Setup](#docker-buildx-setup)

- [Building Multi-Arch Images](#building-multi-arch-images)

- [Verifying Multi-Arch Images](#verifying-multi-arch-images)

- [CI/CD Integration](#cicd-integration)

- [Troubleshooting](#troubleshooting)

- [Best Practices](#best-practices)

- [FAQ](#faq)

- [See Also](#see-also)

## Overview

This guide covers building multi-architecture (multi-arch) Docker images for Secrets, supporting multiple CPU architectures from a single image manifest. This enables seamless deployment across different hardware platforms (x86_64 servers, ARM-based cloud instances, Raspberry Pi, Apple Silicon Macs, etc.).

**Supported architectures** (v0.10.0+):

- **`linux/amd64`** (x86_64) - Intel/AMD servers, most cloud VMs

- **`linux/arm64`** (aarch64) - AWS Graviton, Google Tau T2A, Azure Cobalt, Apple Silicon

**Why multi-arch matters:**

1. **Cloud cost optimization**: ARM instances (AWS Graviton2/3, Google Tau T2A) are 20-40% cheaper than x86 equivalents
2. **Performance**: Native ARM execution (no emulation overhead)
3. **Developer experience**: Run production images locally on Apple Silicon Macs (M1/M2/M3)
4. **Future-proofing**: ARM adoption is growing (cloud providers, edge computing, IoT)

---

## Quick Start

### Building Multi-Arch Images

**Prerequisites:**

- Docker 19.03+ with BuildKit enabled

- Docker Buildx plugin (included in Docker Desktop)

- Authenticated to Docker registry (`docker login`)

**Build and push multi-arch images:**

```bash
# Build for both amd64 and arm64, push to registry
make docker-build-multiarch

# Outputs:
#   allisson/secrets:latest (multi-arch manifest)
#   allisson/secrets:<VERSION> (multi-arch manifest)

```

**Build specific architecture locally:**

```bash
# Build for amd64 only (load locally, don't push)
docker buildx build --platform linux/amd64 --load -t secrets:amd64 .

# Build for arm64 only (load locally, don't push)
docker buildx build --platform linux/arm64 --load -t secrets:arm64 .

```

**Verify multi-arch manifest:**

```bash
# Inspect manifest (shows all supported architectures)
docker manifest inspect allisson/secrets:<VERSION>

# Example output:
# {
#   "manifests": [
#     {
#       "platform": {
#         "architecture": "amd64",
#         "os": "linux"
#       },
#       "digest": "sha256:abc123..."
#     },
#     {
#       "platform": {
#         "architecture": "arm64",
#         "os": "linux"
#       },
#       "digest": "sha256:def456..."
#     }
#   ]
# }

```

---

## Docker Buildx Setup

### Installing Docker Buildx

**Docker Desktop** (macOS, Windows): Buildx is pre-installed.

**Linux** (manual installation):

```bash
# Check if buildx is available
docker buildx version
# docker buildx version github.com/docker/buildx v0.12.1

# If not installed, install manually
mkdir -p ~/.docker/cli-plugins/
curl -Lo ~/.docker/cli-plugins/docker-buildx \
  https://github.com/docker/buildx/releases/download/v0.12.1/buildx-v0.12.1.linux-amd64
chmod +x ~/.docker/cli-plugins/docker-buildx

# Verify installation
docker buildx version

```

### Creating a Builder Instance

Buildx uses "builder instances" to build multi-arch images. Create a builder with multi-platform support:

```bash
# Create new builder instance (only needed once)
docker buildx create --name multiarch-builder --use

# Inspect builder (shows supported platforms)
docker buildx inspect multiarch-builder --bootstrap

# Example output:
# Name:   multiarch-builder
# Driver: docker-container
# Status: running
# Platforms: linux/amd64, linux/amd64/v2, linux/amd64/v3, linux/arm64, linux/riscv64, ...

```

**Using the default builder:**

```bash
# Use default builder
docker buildx use default

# Verify current builder
docker buildx ls
# NAME/NODE              DRIVER/ENDPOINT             STATUS  BUILDKIT PLATFORMS
# multiarch-builder *    docker-container
#   multiarch-builder0   unix:///var/run/docker.sock running v0.12.1  linux/amd64*, linux/arm64, ...
# default                docker
#   default              default                     running v0.11.0  linux/amd64, ...

```

**Note**: The `*` indicates the currently active builder.

### QEMU for Cross-Platform Builds

To build ARM images on x86 hosts (and vice versa), Docker uses QEMU for emulation. Install QEMU binfmt support:

```bash
# Install QEMU emulation support (Linux)
docker run --privileged --rm tonistiigi/binfmt --install all

# Verify QEMU is installed
docker buildx inspect --bootstrap | grep Platforms
# Platforms: linux/amd64, linux/arm64, linux/riscv64, linux/ppc64le, ...

# Test ARM emulation on x86 host
docker run --rm --platform linux/arm64 alpine uname -m
# aarch64

```

**macOS/Windows**: QEMU is pre-configured in Docker Desktop.

---

## Building Multi-Arch Images

### Method 1: Using Makefile (Recommended)

The Makefile provides a simple interface for multi-arch builds:

```bash
# Build and push multi-arch images (amd64 + arm64)
make docker-build-multiarch

# Build with custom version tag
make docker-build-multiarch VERSION=v1.0.0-rc1

# Build with custom registry
make docker-build-multiarch DOCKER_REGISTRY=myregistry.io/myorg

```

**What it does:**

1. Builds images for `linux/amd64` and `linux/arm64` platforms
2. Creates multi-arch manifest (single image tag, multiple architectures)
3. Pushes images and manifest to registry
4. Tags images with both `:latest` and `:$VERSION`

**Output:**

```text
Building multi-platform Docker image...
  Version: v0.10.0
  Build Date: 2026-02-21T10:30:00Z
  Commit SHA: abc123def456...
  Platforms: linux/amd64, linux/arm64
[+] Building 45.2s (24/24) FINISHED
...
Multi-platform images pushed: allisson/secrets:latest and allisson/secrets:<VERSION>

```

### Method 2: Using Docker Buildx Directly

For advanced use cases, use `docker buildx` directly:

```bash
# Build and push multi-arch images
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v0.13.0 \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) \
  -t allisson/secrets:<VERSION> \
  -t allisson/secrets:latest \
  --push \
  .

# Build for specific platform (load locally, don't push)
docker buildx build \
  --platform linux/arm64 \
  --load \
  -t secrets:arm64-local \
  .

# Build without pushing (create manifest only)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t secrets:multiarch \
  --output type=docker \
  .

```

**Important flags:**

- `--platform`: Comma-separated list of target platforms

- `--push`: Push images to registry (required for multi-arch manifests)

- `--load`: Load single-platform image into local Docker (cannot be used with `--push`)

- `--output type=docker`: Save images to local Docker daemon (single platform only)

- `--output type=registry`: Push to registry (enables multi-platform manifests)

### Method 3: Build Locally, Push Separately

For air-gapped environments or offline builds:

```bash
# Step 1: Build multi-arch images to local cache
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v0.13.0 \
  -t allisson/secrets:<VERSION> \
  --output type=oci,dest=secrets-v0.13.0.tar \
  .

# Step 2: Transfer OCI archive to target environment (USB, network copy, etc.)
# secrets-v0.13.0.tar contains all platform images

# Step 3: Load and push from target environment
docker load < secrets-v0.13.0.tar
docker push allisson/secrets:<VERSION>

```

---

## Verifying Multi-Arch Images

### Inspecting Manifest Lists

Multi-arch images use **manifest lists** (also called "fat manifests") that point to platform-specific images:

```bash
# Inspect multi-arch manifest
docker manifest inspect allisson/secrets:<VERSION>

# Example output (simplified):
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
  "manifests": [
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1234,
      "digest": "sha256:abc123...",
      "platform": {
        "architecture": "amd64",
        "os": "linux"
      }
    },
    {
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "size": 1234,
      "digest": "sha256:def456...",
      "platform": {
        "architecture": "arm64",
        "os": "linux"
      }
    }
  ]
}

```

**Extract specific platform digest:**

```bash
# Get amd64 digest
docker manifest inspect allisson/secrets:<VERSION> | \
  jq -r '.manifests[] | select(.platform.architecture=="amd64") | .digest'
# sha256:abc123...

# Get arm64 digest
docker manifest inspect allisson/secrets:<VERSION> | \
  jq -r '.manifests[] | select(.platform.architecture=="arm64") | .digest'
# sha256:def456...

```

### Pulling Platform-Specific Images

Docker automatically pulls the correct platform image based on the host architecture:

```bash
# On x86_64 host: pulls amd64 image
docker pull allisson/secrets:<VERSION>

# On ARM64 host: pulls arm64 image
docker pull allisson/secrets:<VERSION>

# Force pull specific platform (regardless of host)
docker pull --platform linux/arm64 allisson/secrets:<VERSION>
docker pull --platform linux/amd64 allisson/secrets:<VERSION>

```

### Testing Platform-Specific Images

**Verify correct architecture:**

```bash
# Run on x86_64 host (native execution)
docker run --rm allisson/secrets:<VERSION> uname -m
# x86_64

# Run ARM image on x86_64 host (QEMU emulation)
docker run --rm --platform linux/arm64 allisson/secrets:<VERSION> uname -m
# aarch64

# Verify application works on both platforms
docker run --rm --platform linux/amd64 allisson/secrets:<VERSION> --version
docker run --rm --platform linux/arm64 allisson/secrets:<VERSION> --version
# Both should output: Version: v0.10.0

```

**Compare image sizes:**

```bash
# Pull both platforms
docker pull --platform linux/amd64 allisson/secrets:<VERSION>
docker pull --platform linux/arm64 allisson/secrets:<VERSION>

# Compare sizes
docker images allisson/secrets:<VERSION>
# REPOSITORY         TAG       IMAGE ID       CREATED        SIZE
# allisson/secrets   v0.10.0   abc123...      2 hours ago    12.5 MB (amd64)
# allisson/secrets   v0.10.0   def456...      2 hours ago    12.3 MB (arm64)

```

---

## CI/CD Integration

### GitHub Actions (Recommended)

Secrets uses GitHub Actions for automated multi-arch builds on every release:

```yaml
# ../../../.github/workflows/docker-push.yml
name: Docker Multi-Arch Build

on:
  push:
    tags:
      - 'v*.*.*'

  workflow_dispatch:

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code

        uses: actions/checkout@v4

      - name: Set up QEMU

        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx

        uses: docker/setup-buildx-action@v3

      - name: Login to Docker Hub

        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Extract metadata (tags, labels)

        id: meta
        uses: docker/metadata-action@v5
        with:
          images: allisson/secrets
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=semver,pattern={{major}}
            type=raw,value=latest

      - name: Build and push multi-arch image

        uses: docker/build-push-action@v5
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            VERSION=${{ github.ref_name }}
            BUILD_DATE=${{ steps.meta.outputs.created }}
            COMMIT_SHA=${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

```

**Benefits:**

- ✅ Automated builds on every git tag push

- ✅ Multi-arch manifest published automatically

- ✅ Semantic versioning tags (`:latest`, `:v0.10.0`, `:v0.10`, `:v0`)

- ✅ Build caching (GitHub Actions cache) reduces build time by 50-80%

### GitLab CI

```yaml
# .gitlab-ci.yml
docker-multiarch:
  image: docker:latest
  services:
    - docker:dind

  variables:
    DOCKER_DRIVER: overlay2
    DOCKER_TLS_CERTDIR: "/certs"
  before_script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY

    - docker buildx create --use --name multiarch-builder

  script:
    - |

      docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --build-arg VERSION=$CI_COMMIT_TAG \
        --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
        --build-arg COMMIT_SHA=$CI_COMMIT_SHA \
        -t $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG \
        -t $CI_REGISTRY_IMAGE:latest \
        --push \
        .
  only:
    - tags

```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    environment {
        DOCKER_REGISTRY = 'allisson'
        IMAGE_NAME = 'secrets'
    }
    stages {
        stage('Build Multi-Arch') {
            steps {
                script {
                    sh '''
                        docker buildx create --use --name multiarch-builder || true
                        docker buildx build \
                            --platform linux/amd64,linux/arm64 \
                            --build-arg VERSION=${GIT_TAG} \
                            --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
                            --build-arg COMMIT_SHA=${GIT_COMMIT} \
                            -t ${DOCKER_REGISTRY}/${IMAGE_NAME}:${GIT_TAG} \
                            -t ${DOCKER_REGISTRY}/${IMAGE_NAME}:latest \
                            --push \
                            .
                    '''
                }
            }
        }
    }
}

```

---

## Troubleshooting

### Issue: "multiple platforms feature is currently not supported"

**Cause**: Using `--load` with multiple platforms (Docker can only load one platform at a time).

**Solution**: Use `--push` to push multi-arch images to registry, or build single platform with `--load`:

```bash
# Wrong (fails with error)
docker buildx build --platform linux/amd64,linux/arm64 --load -t secrets .

# Correct (push to registry)
docker buildx build --platform linux/amd64,linux/arm64 --push -t allisson/secrets:<VERSION> .

# Correct (load single platform locally)
docker buildx build --platform linux/amd64 --load -t secrets:amd64 .

```

### Issue: "exec user process caused: exec format error"

**Cause**: Running wrong platform image (e.g., ARM64 image on x86_64 host without QEMU).

**Solution**: Install QEMU emulation or pull correct platform image:

```bash
# Install QEMU
docker run --privileged --rm tonistiigi/binfmt --install all

# Or force pull correct platform
docker pull --platform linux/amd64 allisson/secrets:<VERSION>

```

### Issue: Slow multi-arch builds (> 10 minutes)

**Cause**: Cross-platform compilation uses QEMU emulation (slow).

**Solutions:**

1. **Use build cache** (GitHub Actions cache, BuildKit cache):

   ```bash
   docker buildx build \
     --cache-from type=registry,ref=allisson/secrets:buildcache \
     --cache-to type=registry,ref=allisson/secrets:buildcache,mode=max \
     ...
   ```

2. **Use native builders** (build each platform on native hardware):

   ```bash
   # On x86_64 host
   docker buildx build --platform linux/amd64 --push -t allisson/secrets:<VERSION>-amd64 .
   
   # On ARM64 host
   docker buildx build --platform linux/arm64 --push -t allisson/secrets:<VERSION>-arm64 .
   
   # Create manifest manually
   docker manifest create allisson/secrets:<VERSION> \
     allisson/secrets:<VERSION>-amd64 \
     allisson/secrets:<VERSION>-arm64
   docker manifest push allisson/secrets:<VERSION>
   ```

3. **Enable BuildKit inline cache**:

   ```dockerfile
   # Dockerfile
   # syntax=docker/dockerfile:1
   ```

### Issue: "failed to solve: failed to push: unexpected status: 401 Unauthorized"

**Cause**: Not authenticated to Docker registry.

**Solution**: Login to registry before building:

```bash
# Docker Hub
docker login

# GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# AWS ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# Google Container Registry
gcloud auth configure-docker

```

### Issue: ARM64 builds fail on CI (GitHub Actions, GitLab CI)

**Cause**: QEMU not installed in CI environment.

**Solution**: Install QEMU in CI pipeline:

```yaml
# GitHub Actions

- name: Set up QEMU

  uses: docker/setup-qemu-action@v3

# GitLab CI
before_script:
  - docker run --privileged --rm tonistiigi/binfmt --install all

```

---

## Best Practices

### 1. Always Pin Distroless Digest for Both Platforms

**Bad** (floating tag, no digest):

```dockerfile
FROM gcr.io/distroless/static-debian13:nonroot

```

**Good** (pinned digest, but only supports one platform):

```dockerfile
FROM gcr.io/distroless/static-debian13:nonroot@sha256:abc123...
# Problem: This digest might only support amd64

```

**Best** (use tag with digest, supports multi-arch):

```dockerfile
# Use tag + digest for multi-platform support
FROM gcr.io/distroless/static-debian13:nonroot@sha256:d90359c7...
# Distroless publishes multi-arch manifests, so this works for both amd64 and arm64

```

**Verify distroless supports both platforms:**

```bash
docker manifest inspect gcr.io/distroless/static-debian13:nonroot@sha256:d90359c7... | \
  jq '.manifests[].platform.architecture'
# "amd64"
# "arm64"

```

### 2. Test Both Platforms Before Release

```bash
# Test amd64 build
docker buildx build --platform linux/amd64 --load -t secrets:test-amd64 .
docker run --rm secrets:test-amd64 --version

# Test arm64 build (uses QEMU emulation on x86_64 host)
docker buildx build --platform linux/arm64 --load -t secrets:test-arm64 .
docker run --rm secrets:test-arm64 --version

# Run integration tests on both platforms
docker run --rm secrets:test-amd64 server &
# Run tests...
docker run --rm secrets:test-arm64 server &
# Run tests...

```

### 3. Use Build Cache to Speed Up Builds

```bash
# Enable BuildKit cache
export DOCKER_BUILDKIT=1

# Use GitHub Actions cache
docker buildx build \
  --cache-from type=gha \
  --cache-to type=gha,mode=max \
  --platform linux/amd64,linux/arm64 \
  ...

# Use registry cache
docker buildx build \
  --cache-from type=registry,ref=allisson/secrets:buildcache \
  --cache-to type=registry,ref=allisson/secrets:buildcache,mode=max \
  --platform linux/amd64,linux/arm64 \
  ...

```

### 4. Document Supported Platforms

Add supported platforms to README and release notes:

```markdown
## Supported Platforms

- `linux/amd64` (x86_64) - Intel/AMD servers

- `linux/arm64` (aarch64) - AWS Graviton, Google Tau T2A, Apple Silicon

```

### 5. Monitor Build Times and Costs

Multi-arch builds take 2-3x longer than single-platform builds (due to QEMU emulation). Monitor CI/CD costs:

```bash
# GitHub Actions: Check "billable time" in Actions tab
# GitLab CI: Check "CI/CD minutes" in project settings
# Jenkins: Monitor build duration trends

```

**Optimization tips:**

- Use build caching (reduces build time by 50-80%)

- Build multi-arch only on releases (not every commit)

- Use native builders for critical builds (no emulation overhead)

---

## FAQ

### Q: Do I need to build multi-arch images for every deployment?

**A**: No. Build multi-arch images for releases only. For development/testing, build single-platform images:

```bash
# Development: build for local platform only
docker build -t secrets:dev .

# Release: build multi-arch
make docker-build-multiarch VERSION=v1.0.0

```

### Q: Can I use multi-arch images with Docker Compose?

**A**: Yes, Docker Compose automatically pulls the correct platform image:

```yaml
# docker-compose.yml
services:
  secrets:
    image: allisson/secrets:<VERSION>  # Pulls amd64 on x86_64, arm64 on ARM64
    ports:
      - "8080:8080"

```

### Q: How do I know which platform image Docker pulled?

**A**: Inspect the image after pulling:

```bash
docker pull allisson/secrets:<VERSION>
docker inspect allisson/secrets:<VERSION> --format='{{.Architecture}}'
# amd64 (on x86_64 host)
# arm64 (on ARM64 host)

```

### Q: What's the performance difference between amd64 and arm64?

**A**: For Go applications (like Secrets), ARM64 performance is comparable to amd64:

- **CPU-bound workloads**: ARM64 (Graviton3) is 10-20% faster than x86_64 (Intel Xeon) for some workloads

- **Memory-bound workloads**: Similar performance

- **Cost**: ARM instances are 20-40% cheaper (AWS Graviton2/3, Google Tau T2A)

**Recommendation**: Use ARM64 for cost savings, unless you have specific x86_64 requirements.

---

## See Also

- [Dockerfile Reference](../../../Dockerfile) - Multi-stage build configuration

- [Container Security Guide](docker-hardened.md) - Security best practices

- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/) - Official buildx docs

- [GitHub Actions Multi-Arch Example](../../../.github/workflows/docker-push.yml) - CI/CD workflow
