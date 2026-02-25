# 🏷️ OCI Image Labels Reference

> **Document version**: v0.13.0  
> Last updated: 2026-02-25  
> **Audience**: DevOps engineers, security teams, compliance officers

## Table of Contents

- [Overview](#overview)

- [Label Schema](#label-schema)

- [Querying Image Labels](#querying-image-labels)

- [Using Labels for Security and Compliance](#using-labels-for-security-and-compliance)

- [Build-Time Label Injection](#build-time-label-injection)

- [Label Verification](#label-verification)

- [Label Maintenance](#label-maintenance)

- [Best Practices](#best-practices)

- [Troubleshooting](#troubleshooting)

- [Related Documentation](#related-documentation)

- [References](#references)

This document describes the OCI (Open Container Initiative) image labels used in the Secrets container image. These labels provide metadata for security scanning, SBOM generation, container registries, and operational tooling.

## Overview

The Secrets container image follows the [OCI Image Format Specification](https://github.com/opencontainers/image-spec/blob/main/annotations.md) for image annotations. Labels are embedded at build time and provide essential metadata about the image's content, provenance, and security characteristics.

**Use Cases**:

- **Security Scanning**: Tools like Trivy, Grype, and Snyk use labels to identify versions and vulnerabilities

- **SBOM Generation**: Software Bill of Materials (SBOM) tools use labels for component tracking

- **Container Registries**: Docker Hub, GitHub Container Registry, and others display label information

- **Operational Tooling**: Monitoring tools and CI/CD pipelines use labels for automation

## Label Schema

The image uses the standard `org.opencontainers.image.*` namespace defined by the OCI specification.

### Basic Information Labels

| Label | Description | Example Value | Source |
|-------|-------------|---------------|--------|
| `org.opencontainers.image.title` | Human-readable image title | `Secrets` | Static |
| `org.opencontainers.image.description` | Brief description of the application | `Lightweight secrets manager with envelope encryption, transit encryption, and audit logs` | Static |
| `org.opencontainers.image.url` | Project homepage URL | `https://github.com/allisson/secrets` | Static |
| `org.opencontainers.image.source` | Source code repository URL | `https://github.com/allisson/secrets` | Static |
| `org.opencontainers.image.documentation` | Documentation URL | `https://github.com/allisson/secrets/tree/main/docs` | Static |

### Version and Build Metadata

| Label | Description | Example Value | Source |
|-------|-------------|---------------|--------|
| `org.opencontainers.image.version` | Application version | `v0.10.0` | Build arg (`VERSION`) |
| `org.opencontainers.image.created` | ISO 8601 build timestamp | `2026-02-21T10:30:00Z` | Build arg (`BUILD_DATE`) |
| `org.opencontainers.image.revision` | Git commit hash | `23d48a137821f9428304e9929cf470adf8c3dee6` | Build arg (`COMMIT_SHA`) |

**Note**: These labels are injected at build time via Docker build arguments. Local builds without build args will show default values (`version=dev`, `created` and `revision` empty).

### License and Authorship

| Label | Description | Example Value | Source |
|-------|-------------|---------------|--------|
| `org.opencontainers.image.licenses` | SPDX license identifier | `MIT` | Static |
| `org.opencontainers.image.vendor` | Organization or individual name | `Allisson Azevedo` | Static |
| `org.opencontainers.image.authors` | Contact information | `Allisson Azevedo <allisson@gmail.com>` | Static |

### Base Image Metadata

| Label | Description | Example Value | Source |
|-------|-------------|---------------|--------|
| `org.opencontainers.image.base.name` | Base image name | `gcr.io/distroless/static-debian13` | Static |
| `org.opencontainers.image.base.digest` | Base image SHA256 digest | `sha256:d90359c7a3ad67b3c11ca44fd5f3f5208cbef546f2e692b0dc3410a869de46bf` | Static |

**Purpose**: Base image metadata enables:

- **Supply Chain Security**: Track the provenance of the base image

- **Vulnerability Scanning**: Identify vulnerabilities in the base layer

- **SBOM Generation**: Create complete software bill of materials

- **Immutable Builds**: Verify that the expected base image was used

## Querying Image Labels

### Docker CLI

**View all labels**:

```bash
docker inspect allisson/secrets:latest | jq '.[0].Config.Labels'

```

**View specific label**:

```bash
docker inspect allisson/secrets:latest \
  --format '{{ index .Config.Labels "org.opencontainers.image.version" }}'

```

**View version information**:

```bash
docker inspect allisson/secrets:latest | jq -r '
  .[0].Config.Labels |
  "Version: \(.["org.opencontainers.image.version"])
Build Date: \(.["org.opencontainers.image.created"])
Commit SHA: \(.["org.opencontainers.image.revision"])"
'

```

### Docker Compose

```yaml
services:
  secrets-api:
    image: allisson/secrets:latest
    # Labels are automatically inherited from the image
    # You can also add container-specific labels:
    labels:
      - "com.mycompany.environment=production"

      - "com.mycompany.team=platform"

```

## Using Labels for Security and Compliance

### SBOM Generation

**Generate CycloneDX SBOM**:

```bash
# Using Syft
syft allisson/secrets:latest -o cyclonedx-json > sbom.json

# Using Trivy
trivy image --format cyclonedx allisson/secrets:latest > sbom.json

```

**Generate SPDX SBOM**:

```bash
# Using Syft
syft allisson/secrets:latest -o spdx-json > sbom.spdx.json

# Using Trivy
trivy image --format spdx-json allisson/secrets:latest > sbom.spdx.json

```

The OCI labels provide metadata that enriches SBOM reports with:

- Application name and version

- Build timestamp and commit hash

- Base image provenance

- License information

- Author and vendor details

### Vulnerability Scanning

**Trivy scan with label context**:

```bash
trivy image --severity HIGH,CRITICAL allisson/secrets:latest

# Trivy uses labels to:
# - Identify the application version for CVE correlation

# - Track base image vulnerabilities via base.name and base.digest

# - Generate detailed reports with build metadata

```

**Grype scan with label context**:

```bash
grype allisson/secrets:latest

# Grype uses labels to:
# - Match package versions against vulnerability databases

# - Track base image components

# - Provide remediation guidance based on version metadata

```

### Container Registry Display

**Docker Hub**:

- Labels appear under "Image Details"

- Version, description, and source URL are prominently displayed

- Automated builds can use labels for tagging strategies

**GitHub Container Registry (ghcr.io)**:

- Labels are displayed in the package details page

- Source URL creates automatic linking to the repository

- Version labels enable automated vulnerability alerts

**AWS ECR / Google Artifact Registry**:

- Labels are indexed for searching and filtering

- Lifecycle policies can use labels for retention rules

- Security scanning services use labels for CVE tracking

## Build-Time Label Injection

### Manual Builds

```bash
# Build with version metadata
docker build -t allisson/secrets:v0.14.0 \
  --build-arg VERSION=v0.13.0 \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) .

```

### CI/CD Builds

**GitHub Actions** (automatic injection):

```yaml

- name: Build Docker image

  run: |
    VERSION=$(git describe --tags --always --dirty)
    BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    COMMIT_SHA=${{ github.sha }}
    
    docker build -t allisson/secrets:latest \
      --build-arg VERSION=${VERSION} \
      --build-arg BUILD_DATE=${BUILD_DATE} \
      --build-arg COMMIT_SHA=${COMMIT_SHA} .

```

**GitLab CI**:

```yaml
build:
  script:
    - export VERSION=$(git describe --tags --always --dirty)

    - export BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    - export COMMIT_SHA=${CI_COMMIT_SHA}

    - docker build -t allisson/secrets:latest

        --build-arg VERSION=${VERSION}
        --build-arg BUILD_DATE=${BUILD_DATE}
        --build-arg COMMIT_SHA=${COMMIT_SHA} .

```

**Makefile** (using `make docker-build`):

```makefile
# Automatic version detection and injection
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_SHA := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

docker-build:
    docker build -t $(DOCKER_IMAGE):latest \
      --build-arg VERSION=$(VERSION) \
      --build-arg BUILD_DATE=$(BUILD_DATE) \
      --build-arg COMMIT_SHA=$(COMMIT_SHA) .

```

## Label Verification

### Automated Verification Script

Create a script to verify that all required labels are present:

```bash
#!/bin/bash
# verify-oci-labels.sh

IMAGE="allisson/secrets:latest"

REQUIRED_LABELS=(
  "org.opencontainers.image.title"
  "org.opencontainers.image.description"
  "org.opencontainers.image.version"
  "org.opencontainers.image.created"
  "org.opencontainers.image.revision"
  "org.opencontainers.image.licenses"
  "org.opencontainers.image.source"
  "org.opencontainers.image.base.name"
  "org.opencontainers.image.base.digest"
)

echo "Verifying OCI labels for: $IMAGE"
echo "============================================"

MISSING=0
for label in "${REQUIRED_LABELS[@]}"; do
  value=$(docker inspect "$IMAGE" \
    --format "{{ index .Config.Labels \"$label\" }}" 2>/dev/null)
  
  if [ -z "$value" ] || [ "$value" = "<no value>" ]; then
    echo "❌ MISSING: $label"
    MISSING=$((MISSING + 1))
  else
    echo "✅ $label: $value"
  fi
done

echo "============================================"
if [ $MISSING -eq 0 ]; then
  echo "All required labels present"
  exit 0
else
  echo "Missing $MISSING required labels"
  exit 1
fi

```

### Integration Tests

Add label verification to your CI/CD pipeline:

```yaml
# .github/workflows/ci.yml

- name: Verify OCI labels

  run: |
    docker inspect allisson/secrets:latest | jq -e '
      .[0].Config.Labels |
      select(
        .["org.opencontainers.image.version"] != null and
        .["org.opencontainers.image.created"] != null and
        .["org.opencontainers.image.revision"] != null and
        .["org.opencontainers.image.licenses"] == "MIT"
      )
    ' || (echo "Missing required OCI labels" && exit 1)

```

## Label Maintenance

### When to Update Labels

| Scenario | Labels to Update | Action |
|----------|------------------|--------|
| **New release** | `version`, `created`, `revision` | Automatic (build args) |
| **Base image update** | `base.name`, `base.digest` | Manual update in Dockerfile |
| **License change** | `licenses` | Manual update in Dockerfile |
| **Repository move** | `url`, `source`, `documentation` | Manual update in Dockerfile |
| **Author change** | `authors`, `vendor` | Manual update in Dockerfile |
| **Description change** | `title`, `description` | Manual update in Dockerfile |

### Updating Base Image Digest

When updating the distroless base image:

```bash
# 1. Pull the latest base image
docker pull gcr.io/distroless/static-debian13:latest

# 2. Get the SHA256 digest
docker inspect gcr.io/distroless/static-debian13:latest \
  --format '{{index .RepoDigests 0}}'

# Output: gcr.io/distroless/static-debian13@sha256:d90359c7...

# 3. Update Dockerfile:
#    - FROM statement with new digest

#    - org.opencontainers.image.base.digest label with new digest

```

## Best Practices

### 1. Always Inject Build Metadata

**Bad** (local builds without metadata):

```bash
docker build -t allisson/secrets:latest .
# Labels show: version=dev, created=<empty>, revision=<empty>

```

**Good** (production builds with metadata):

```bash
docker build -t allisson/secrets:latest \
  --build-arg VERSION=$(git describe --tags) \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) .
# Labels show: version=v0.13.0, created=2026-02-21T10:30:00Z, revision=23d48a1...

```

### 2. Verify Labels in CI/CD

Add label verification to your pipeline to catch missing metadata:

```bash
# Fail the build if version is not set correctly
VERSION=$(docker inspect allisson/secrets:latest \
  --format '{{ index .Config.Labels "org.opencontainers.image.version" }}')

if [ "$VERSION" = "dev" ] || [ -z "$VERSION" ]; then
  echo "ERROR: Image version not set correctly"
  exit 1
fi

```

### 3. Use Labels for Automation

**Example**: Automatic vulnerability scanning based on version:

```bash
# Scan only production releases (not dev builds)
VERSION=$(docker inspect "$IMAGE" \
  --format '{{ index .Config.Labels "org.opencontainers.image.version" }}')

if [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  trivy image --severity HIGH,CRITICAL "$IMAGE"
else
  echo "Skipping scan for non-release build: $VERSION"
fi

```

### 4. Document Label Schema

Keep this documentation up-to-date when adding or removing labels. All label changes should be:

- Documented in this file

- Reviewed for compliance with OCI specification

- Tested in CI/CD pipelines

- Announced in release notes

## Troubleshooting

### Labels Not Appearing

**Symptom**: `docker inspect` shows empty labels

**Cause**: Build arguments not passed during build

**Solution**:

```bash
# Verify build arguments were passed
docker history allisson/secrets:latest | grep ARG

# Rebuild with build arguments
docker build -t allisson/secrets:latest \
  --build-arg VERSION=v0.13.0 \
  --build-arg BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) .

```

### Labels Show Default Values

**Symptom**: Labels show `version=dev`, `created=<empty>`, `revision=<empty>`

**Cause**: Build arguments were not provided (using default values from ARG statements)

**Solution**: Always provide build arguments in production builds (see "Build-Time Label Injection" section)

### Base Image Digest Mismatch

**Symptom**: Security scanner reports base image mismatch

**Cause**: Dockerfile `FROM` statement uses different digest than `base.digest` label

**Solution**:

```bash
# 1. Check actual base image digest
docker inspect allisson/secrets:latest \
  --format '{{index .RootFS.Layers 0}}'

# 2. Verify it matches the label
docker inspect allisson/secrets:latest \
  --format '{{ index .Config.Labels "org.opencontainers.image.base.digest" }}'

# 3. Update Dockerfile if they don't match

```

## Related Documentation

- [Dockerfile](../../../Dockerfile) - Source of OCI labels

- [Container Security Guide](docker-hardened.md) - Security hardening and verification

- [Security Scanning Guide](../security/scanning.md) - Vulnerability scanning and SBOM generation

- [Multi-Architecture Builds](multi-arch-builds.md) - Building for multiple platforms

- [Docker Getting Started](../../getting-started/docker.md) - Basic Docker usage

## References

- [OCI Image Format Specification](https://github.com/opencontainers/image-spec/blob/main/annotations.md)

- [Docker LABEL Instruction](https://docs.docker.com/engine/reference/builder/#label)

- [Best Practices for Writing Dockerfiles](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/#label)

- [Container Structure Tests](https://github.com/GoogleContainerTools/container-structure-test)
