# 🔍 Security Scanning Guide

> **Document version**: v0.10.0  
> Last updated: 2026-02-21  
> **Audience**: DevOps engineers, security teams, release managers

## Table of Contents

- [Overview](#overview)

- [Quick Start](#quick-start)

- [Scanning Tools](#scanning-tools)

- [SBOM Generation](#sbom-generation)

- [CI/CD Integration](#cicd-integration)

- [Continuous Monitoring](#continuous-monitoring)

- [Vulnerability Triage and Response](#vulnerability-triage-and-response)

- [Best Practices](#best-practices)

- [Troubleshooting](#troubleshooting)

- [See Also](#see-also)

## Overview

This guide covers comprehensive security scanning practices for Secrets container images, including vulnerability detection, SBOM generation, supply chain security, and CI/CD integration.

**Why scan container images:**

1. **Detect vulnerabilities**: Find CVEs in base images, dependencies, and application code before deployment
2. **Compliance**: Meet security compliance requirements (SOC 2, PCI-DSS, HIPAA, ISO 27001)
3. **Supply chain security**: Verify image integrity and generate SBOMs for auditing
4. **Continuous monitoring**: Detect new vulnerabilities in deployed images (even after release)

**Security scanning tools covered:**

- **Trivy** (recommended) - Comprehensive, fast, open-source scanner

- **Docker Scout** - Built into Docker Desktop, commercial support

- **Grype** - Open-source alternative by Anchore

- **Snyk** - Commercial scanner with developer focus

- **Clair** - Open-source scanner for registry integration

---

## Quick Start

### Scan with Trivy (Recommended)

**Install Trivy:**

```bash
# macOS
brew install aquasecurity/trivy/trivy

# Linux
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -

echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update && sudo apt-get install trivy

# Docker (no installation required)
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  aquasec/trivy image allisson/secrets:v0.10.0

```

**Quick scan:**

```bash
# Scan for HIGH and CRITICAL vulnerabilities
trivy image --severity HIGH,CRITICAL allisson/secrets:v0.10.0

# Expected output for v0.10.0 (distroless base):
# allisson/secrets:v0.10.0 (debian 13)
# Total: 0 (HIGH: 0, CRITICAL: 0)

```

**If vulnerabilities found:**

1. Check if they affect your use case (e.g., server-side only, no user input)
2. Update base image digest (pull latest distroless image)
3. Rebuild and rescan
4. If vulnerability persists, check for workarounds or wait for upstream patch

---

## Scanning Tools

### Trivy (Recommended)

**Why Trivy:**

- ✅ Fast scanning (< 10 seconds for Secrets image)

- ✅ Detects OS packages, language-specific dependencies, and misconfigurations

- ✅ Supports SBOM generation (CycloneDX, SPDX)

- ✅ Can scan images, filesystems, and git repos

- ✅ Offline mode for air-gapped environments

- ✅ Free and open-source

**Basic usage:**

```bash
# Scan image
trivy image allisson/secrets:v0.10.0

# Filter by severity
trivy image --severity HIGH,CRITICAL allisson/secrets:v0.10.0

# Output formats
trivy image --format json -o results.json allisson/secrets:v0.10.0
trivy image --format sarif -o results.sarif allisson/secrets:v0.10.0  # GitHub Security tab
trivy image --format table allisson/secrets:v0.10.0  # Human-readable table

# Scan specific platforms
trivy image --platform linux/amd64 allisson/secrets:v0.10.0
trivy image --platform linux/arm64 allisson/secrets:v0.10.0

# Exit with error if vulnerabilities found (CI/CD)
trivy image --severity HIGH,CRITICAL --exit-code 1 allisson/secrets:v0.10.0

```

**Advanced options:**

```bash
# Ignore unfixed vulnerabilities (can't be patched yet)
trivy image --ignore-unfixed allisson/secrets:v0.10.0

# Scan with custom policy (fail on specific CVEs)
trivy image --severity HIGH,CRITICAL \
  --ignore-policy .trivyignore \
  allisson/secrets:v0.10.0

# Scan offline (air-gapped environments)
trivy image --download-db-only  # Download vulnerability database
trivy image --skip-update allisson/secrets:v0.10.0  # Scan without updating DB

# Generate SBOM
trivy image --format cyclonedx -o sbom.json allisson/secrets:v0.10.0
trivy image --format spdx-json -o sbom-spdx.json allisson/secrets:v0.10.0

```

**Ignore specific vulnerabilities (.trivyignore):**

```bash
# .trivyignore - ignore false positives or accepted risks

# CVE-2023-1234 - False positive, application doesn't use vulnerable code path

CVE-2023-1234

# CVE-2023-5678 - Accepted risk, workaround in place

CVE-2023-5678

# Scan with ignore file
trivy image --ignore-policy .trivyignore allisson/secrets:v0.10.0

```

---

### Docker Scout

**Why Docker Scout:**

- ✅ Integrated into Docker Desktop (no installation)

- ✅ Policy-based evaluation (commercial features)

- ✅ Image comparison (diff between versions)

- ✅ Recommendations for base image updates

**Setup:**

```bash
# Enable Docker Scout (Docker Desktop)
docker scout enroll

# Login if using Docker Hub
docker login

```

**Basic usage:**

```bash
# Quick scan
docker scout cves allisson/secrets:v0.10.0

# Compare with previous version
docker scout compare --to allisson/secrets:v0.9.0 allisson/secrets:v0.10.0

# Get recommendations
docker scout recommendations allisson/secrets:v0.10.0

# Generate SBOM
docker scout sbom allisson/secrets:v0.10.0 --format cyclonedx > sbom.json

# Policy evaluation (requires Docker Scout subscription)
docker scout policy allisson/secrets:v0.10.0

```

**CI/CD integration:**

```yaml
# GitHub Actions

- name: Docker Scout scan

  uses: docker/scout-action@v1
  with:
    command: cves
    image: allisson/secrets:${{ github.sha }}
    severity: high,critical
    exit-code: true

```

---

### Grype

**Why Grype:**

- ✅ Open-source alternative to commercial scanners

- ✅ Fast and accurate

- ✅ Supports multiple output formats

- ✅ Good for CI/CD pipelines

**Install:**

```bash
# macOS
brew tap anchore/grype
brew install grype

# Linux
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin

# Docker
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
  anchore/grype:latest allisson/secrets:v0.10.0

```

**Usage:**

```bash
# Scan image
grype allisson/secrets:v0.10.0

# Filter by severity
grype allisson/secrets:v0.10.0 --fail-on high

# Output formats
grype allisson/secrets:v0.10.0 -o json > results.json
grype allisson/secrets:v0.10.0 -o sarif > results.sarif

# Generate SBOM with Syft (Anchore's SBOM tool)
syft allisson/secrets:v0.10.0 -o cyclonedx-json > sbom.json
grype sbom:sbom.json  # Scan SBOM instead of image (faster)

```

---

### Snyk

**Why Snyk:**

- ✅ Developer-friendly UI

- ✅ Automated fix PRs

- ✅ Integrates with GitHub/GitLab/Bitbucket

- ✅ Commercial support

**Setup:**

```bash
# Install Snyk CLI
npm install -g snyk

# Authenticate
snyk auth

```

**Usage:**

```bash
# Scan image
snyk container test allisson/secrets:v0.10.0

# Monitor image (continuous scanning)
snyk container monitor allisson/secrets:v0.10.0

# Scan with custom Dockerfile
snyk container test allisson/secrets:v0.10.0 --file=Dockerfile

# CI/CD integration
snyk container test allisson/secrets:v0.10.0 \
  --severity-threshold=high \
  --fail-on=upgradable

```

---

### Clair

**Why Clair:**

- ✅ Registry-native scanning (integrates with Harbor, Quay)

- ✅ Open-source, RedHat-backed

- ✅ Good for private registries

**Setup:** (requires Clair server deployment)

```bash
# Use clairctl CLI
clairctl report allisson/secrets:v0.10.0

```

**Note**: Clair is typically integrated into container registries (Harbor, Quay) rather than used as a standalone CLI tool.

---

## SBOM Generation

**What is an SBOM:**

SBOM (Software Bill of Materials) is a complete inventory of all components, libraries, and dependencies in a software artifact. Required for:

- **Supply chain security**: Track dependencies for vulnerability monitoring

- **Compliance**: Meet NIST, CISA, and executive order requirements

- **Incident response**: Quickly identify affected systems during CVE disclosure

**Note**: The Secrets image includes comprehensive OCI labels that enrich SBOM reports with version metadata, base image provenance, license information, and build details. See [OCI Labels Reference](../deployment/oci-labels.md) for the complete label schema.

**Generate SBOM with Trivy:**

```bash
# CycloneDX format (recommended for vulnerability scanning)
trivy image --format cyclonedx -o sbom-cyclonedx.json allisson/secrets:v0.10.0

# SPDX format (recommended for compliance)
trivy image --format spdx-json -o sbom-spdx.json allisson/secrets:v0.10.0

# Human-readable SBOM
trivy image --format json -o sbom-full.json allisson/secrets:v0.10.0
cat sbom-full.json | jq '.Results[].Packages[] | {Name: .Name, Version: .Version}'

```

**Generate SBOM with Syft:**

```bash
# Install Syft
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Generate SBOM
syft allisson/secrets:v0.10.0 -o cyclonedx-json > sbom.json
syft allisson/secrets:v0.10.0 -o spdx-json > sbom-spdx.json
syft allisson/secrets:v0.10.0 -o table  # Human-readable

```

**Scan SBOM for vulnerabilities:**

```bash
# Generate SBOM once
syft allisson/secrets:v0.10.0 -o cyclonedx-json > sbom.json

# Scan SBOM multiple times (faster than scanning image)
grype sbom:sbom.json
trivy sbom --format cyclonedx sbom.json

```

**Store SBOM for compliance:**

```bash
# Attach SBOM to container image (OCI artifact)
oras attach --artifact-type application/vnd.cyclonedx+json \
  allisson/secrets:v0.10.0 sbom.json

# Upload to registry
docker scout sbom allisson/secrets:v0.10.0 --output sbom.json
# Store in artifact repository (Artifactory, Nexus)

```

---

## CI/CD Integration

### GitHub Actions

**Trivy integration:**

```yaml
name: Container Security Scan

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    # Scan daily for new vulnerabilities
    - cron: '0 0 * * *'

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      
      - name: Build image

        run: docker build -t secrets:${{ github.sha }} .
      
      - name: Run Trivy vulnerability scanner

        uses: aquasecurity/trivy-action@master
        with:
          image-ref: secrets:${{ github.sha }}
          format: sarif
          output: trivy-results.sarif
          severity: HIGH,CRITICAL
          exit-code: 1  # Fail build on HIGH/CRITICAL
      
      - name: Upload Trivy results to GitHub Security

        uses: github/codeql-action/upload-sarif@v3
        if: always()  # Upload even if scan fails
        with:
          sarif_file: trivy-results.sarif
      
      - name: Generate SBOM

        uses: aquasecurity/trivy-action@master
        with:
          image-ref: secrets:${{ github.sha }}
          format: cyclonedx
          output: sbom.json
      
      - name: Upload SBOM artifact

        uses: actions/upload-artifact@v4
        with:
          name: sbom
          path: sbom.json

```

**Docker Scout integration:**

```yaml
jobs:
  scout-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      
      - name: Build image

        run: docker build -t secrets:${{ github.sha }} .
      
      - name: Docker Scout scan

        uses: docker/scout-action@v1
        with:
          command: cves
          image: secrets:${{ github.sha }}
          severity: high,critical
          exit-code: true
          sarif-file: scout-results.sarif
      
      - name: Upload Scout results

        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: scout-results.sarif

```

---

### GitLab CI

```yaml
# .gitlab-ci.yml
container-scan:
  stage: test
  image: aquasec/trivy:latest
  services:
    - docker:dind

  variables:
    DOCKER_DRIVER: overlay2
    IMAGE: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
  before_script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY

  script:
    # Build image
    - docker build -t $IMAGE .

    
    # Scan with Trivy
    - trivy image --severity HIGH,CRITICAL --exit-code 1 $IMAGE

    
    # Generate SBOM
    - trivy image --format cyclonedx -o sbom.json $IMAGE

  artifacts:
    paths:
      - sbom.json

    reports:
      # GitLab Security Dashboard integration
      container_scanning: gl-container-scanning-report.json
  script:
    - trivy image --format gitlab $IMAGE > gl-container-scanning-report.json

```

---

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    environment {
        IMAGE_NAME = "allisson/secrets:${env.GIT_COMMIT}"
    }
    stages {
        stage('Build') {
            steps {
                sh 'docker build -t ${IMAGE_NAME} .'
            }
        }
        stage('Security Scan') {
            steps {
                script {
                    // Run Trivy scan
                    sh '''
                        docker run --rm \
                          -v /var/run/docker.sock:/var/run/docker.sock \
                          aquasec/trivy image \
                          --severity HIGH,CRITICAL \
                          --exit-code 1 \
                          ${IMAGE_NAME}
                    '''
                }
            }
        }
        stage('Generate SBOM') {
            steps {
                sh '''
                    docker run --rm \
                      -v /var/run/docker.sock:/var/run/docker.sock \
                      -v ${WORKSPACE}:/output \
                      aquasec/trivy image \
                      --format cyclonedx \
                      -o /output/sbom.json \
                      ${IMAGE_NAME}
                '''
                archiveArtifacts artifacts: 'sbom.json'
            }
        }
    }
}

```

---

## Continuous Monitoring

### Scheduled Scans (GitHub Actions)

```yaml
name: Scheduled Vulnerability Scan

on:
  schedule:
    # Run daily at 2 AM UTC
    - cron: '0 2 * * *'

  workflow_dispatch:  # Allow manual trigger

jobs:
  scan-latest:
    runs-on: ubuntu-latest
    steps:
      - name: Pull latest image

        run: docker pull allisson/secrets:latest
      
      - name: Scan with Trivy

        uses: aquasecurity/trivy-action@master
        with:
          image-ref: allisson/secrets:latest
          severity: HIGH,CRITICAL
          exit-code: 0  # Don't fail (just report)
      
      - name: Send alert if vulnerabilities found

        if: failure()
        uses: slackapi/slack-github-action@v1
        with:
          webhook-url: ${{ secrets.SLACK_WEBHOOK }}
          payload: |
            {
              "text": "🚨 New vulnerabilities detected in allisson/secrets:latest"
            }

```

---

### Registry Scanning

**Harbor registry integration:**

Harbor has built-in Trivy integration. Enable in Harbor admin panel:

1. **Administration** → **Interrogation Services** → **Scanners**
2. Add Trivy scanner
3. Set scan schedule: "Scan on push" or "Daily at 2 AM"
4. View scan results in Harbor UI

**Quay registry integration:**

Quay uses Clair for vulnerability scanning:

1. Enable Clair in Quay config
2. Scan results appear in Quay repository page
3. Set up webhook alerts for new vulnerabilities

---

## Vulnerability Triage and Response

### Severity Levels

| Severity | CVSS Score | Response Time | Action Required |
|----------|------------|---------------|-----------------|
| **CRITICAL** | 9.0-10.0 | < 24 hours | Immediate patching, deploy hotfix |
| **HIGH** | 7.0-8.9 | < 7 days | Scheduled patching, next release |
| **MEDIUM** | 4.0-6.9 | < 30 days | Include in monthly update |
| **LOW** | 0.1-3.9 | Best effort | Update during regular maintenance |

### Triage Workflow

**1. Scan detects vulnerability:**

```bash
trivy image --severity HIGH,CRITICAL allisson/secrets:v0.10.0

# Example output:
# CVE-2023-1234 (HIGH)
# Package: openssl
# Installed Version: 3.0.0
# Fixed Version: 3.0.1

```

**2. Assess impact:**

- **Does it affect Secrets?** Check if vulnerable code path is used

- **Is it exploitable?** Check CVSS score, exploit availability

- **Is a fix available?** Check "Fixed Version"

**3. Remediate:**

**Option A: Update base image (if CVE in distroless):**

```bash
# Pull latest distroless digest
docker pull gcr.io/distroless/static-debian13:nonroot

# Get new digest
docker inspect gcr.io/distroless/static-debian13:nonroot --format='{{index .RepoDigests 0}}'

# Update Dockerfile
FROM gcr.io/distroless/static-debian13:nonroot@sha256:NEW_DIGEST

# Rebuild and rescan
docker build -t secrets:patched .
trivy image --severity HIGH,CRITICAL secrets:patched

```

**Option B: Accept risk (if unfixable or false positive):**

```bash
# Document decision in .trivyignore
echo "CVE-2023-1234  # False positive - application doesn't use TLS 1.0" >> .trivyignore

# Scan with ignore file
trivy image --ignore-policy .trivyignore allisson/secrets:v0.10.0

```

**Option C: Implement workaround:**

- Disable vulnerable feature in configuration

- Add network-level mitigation (WAF, firewall rules)

- Document in security advisory

**4. Deploy patch:**

```bash
# Build patched image
docker build -t allisson/secrets:v0.10.1 .

# Verify vulnerability is fixed
trivy image --severity HIGH,CRITICAL allisson/secrets:v0.10.1
# Total: 0 (HIGH: 0, CRITICAL: 0)

# Deploy to production with Docker Compose
docker compose pull
docker compose up -d secrets-api

```

---

## Best Practices

### 1. Scan Early and Often

```bash
# Scan in CI/CD (every commit)
trivy image --severity HIGH,CRITICAL --exit-code 1 secrets:$CI_COMMIT_SHA

# Scan daily (detect new CVEs in deployed images)
# Use GitHub Actions scheduled workflow

# Scan before deployment
trivy image --severity HIGH,CRITICAL --exit-code 1 allisson/secrets:v0.10.0

```

### 2. Use Multiple Scanners

Different scanners have different vulnerability databases. Use at least two:

```bash
# Trivy (primary)
trivy image --severity HIGH,CRITICAL allisson/secrets:v0.10.0

# Grype (secondary)
grype allisson/secrets:v0.10.0 --fail-on high

# Docker Scout (tertiary, if available)
docker scout cves allisson/secrets:v0.10.0

```

### 3. Pin Base Image Digests

```dockerfile
# Bad: floating tag (vulnerabilities can be introduced)
FROM gcr.io/distroless/static-debian13:nonroot

# Good: pinned digest (immutable)
FROM gcr.io/distroless/static-debian13:nonroot@sha256:d90359c7...

```

### 4. Generate SBOMs for Every Release

```bash
# Generate SBOM during build
trivy image --format cyclonedx -o sbom-v0.10.0.json allisson/secrets:v0.10.0

# Store SBOM in artifact repository
# Upload to GitHub release
gh release upload v0.10.0 sbom-v0.10.0.json

# Scan SBOM regularly for new CVEs
trivy sbom sbom-v0.10.0.json

```

### 5. Automate Response

```yaml
# Automatically create GitHub issue when vulnerability detected

- name: Create issue if vulnerabilities found

  if: failure()
  uses: actions/github-script@v7
  with:
    script: |
      github.rest.issues.create({
        owner: context.repo.owner,
        repo: context.repo.repo,
        title: 'Security: New vulnerabilities detected',
        body: 'Trivy scan failed. Review scan results.',
        labels: ['security', 'vulnerability']
      })

```

### 6. Monitor Deployed Images

Don't just scan at build time - continuously monitor production images:

```bash
# Scan production images daily
trivy image --severity HIGH,CRITICAL allisson/secrets:latest

```

---

## Troubleshooting

### Trivy Fails with "database not found"

**Cause**: Vulnerability database not downloaded.

**Solution**:

```bash
# Download database
trivy image --download-db-only

# Retry scan
trivy image allisson/secrets:v0.10.0

```

### False Positives

**Cause**: Scanner detects vulnerability in package that's not actually used.

**Solution**: Add to `.trivyignore`:

```bash
# .trivyignore
CVE-2023-1234  # False positive - TLS 1.0 disabled in config

```

### Scan Timeout in CI/CD

**Cause**: Large image or slow network.

**Solution**:

```bash
# Increase timeout
trivy image --timeout 10m allisson/secrets:v0.10.0

# Use local cache
trivy image --cache-dir /tmp/trivy-cache allisson/secrets:v0.10.0

```

---

## See Also

- [Container Security Guide](../security/container-security.md) - Runtime security best practices

- [Base Image Migration Guide](../deployment/base-image-migration.md) - Migrate to distroless for fewer CVEs

- [Multi-Arch Builds Guide](../deployment/multi-arch-builds.md) - Build secure images for multiple architectures

- [Incident Response Guide](../observability/incident-response.md) - Respond to security incidents

- [Trivy Documentation](https://aquasecurity.github.io/trivy/) - Official Trivy docs

- [Docker Scout Documentation](https://docs.docker.com/scout/) - Official Docker Scout docs
