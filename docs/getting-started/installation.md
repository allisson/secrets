# 💾 Install Pre-compiled Binaries

> Last updated: 2026-02-28

If you don't want to build from source or use Docker, you can download pre-compiled binaries for your operating system and architecture directly from GitHub.

## 1) Download from GitHub Releases

Visit the [GitHub Releases](https://github.com/allisson/secrets/releases) page and find the latest version (currently `v0.20.0`).

### Asset Naming Convention

Select the asset that matches your platform:

| Platform | OS | Arch | Filename Pattern |
| --- | --- | --- | --- |
| macOS (Intel) | `Darwin` | `x86_64` | `secrets_Darwin_x86_64.tar.gz` |
| macOS (Apple Silicon) | `Darwin` | `arm64` | `secrets_Darwin_arm64.tar.gz` |
| Linux (64-bit) | `Linux` | `x86_64` | `secrets_Linux_x86_64.tar.gz` |
| Linux (Arm64) | `Linux` | `arm64` | `secrets_Linux_arm64.tar.gz` |
| Windows (64-bit) | `Windows` | `x86_64` | `secrets_Windows_x86_64.zip` |

## 2) Extract and Install

### Linux and macOS

```bash
# Extract the archive (e.g., for Linux x86_64)
tar -xvzf secrets_Linux_x86_64.tar.gz

# Move the binary to your PATH (e.g., /usr/local/bin)
sudo mv secrets /usr/local/bin/
```

### Windows

1. Extract the `.zip` file.
2. Move `secrets.exe` to a directory in your `PATH` (e.g., `C:\Windows\System32` or a custom tools folder).

## 3) Verify Installation

Run the following command to verify the installation:

```bash
secrets --version
```

Expected output:

```text
Version:    v0.20.0
Build Date: 2026-02-28T...
Commit SHA: ...
```

## 4) Verify Checksum (Optional but Recommended)

For security, you should verify the integrity of the downloaded file using the provided `checksums.txt` file.

```bash
# Download checksums.txt
curl -LO https://github.com/allisson/secrets/releases/download/v0.20.0/checksums.txt

# Verify checksum (Linux)
sha256sum --ignore-missing -c checksums.txt

# Verify checksum (macOS)
shasum -a 256 --ignore-missing -c checksums.txt
```

## Next Steps

- 🐳 [Run with Docker](docker.md)
- 💻 [Local Development](local-development.md)
- 🧭 [Day 0 Walkthrough](day-0-walkthrough.md)
- ⚙️ [Configuration Guide](../configuration.md)
