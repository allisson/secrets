# 📦 Run with Pre-compiled Binary

> Last updated: 2026-02-28

This guide covers how to install and run Secrets using pre-compiled binaries from GitHub Releases. This is a great option if you don't want to use Docker but still want a quick and easy installation.

## 📥 Download

1. Go to the [GitHub Releases](https://github.com/allisson/secrets/releases) page.
2. Download the archive for your operating system and architecture (e.g., `secrets_Linux_x86_64.tar.gz`).
3. Extract the archive:

   ```bash
   tar -xzf secrets_Linux_x86_64.tar.gz
   ```

4. Move the `secrets` binary to a directory in your `PATH` (e.g., `/usr/local/bin`):

   ```bash
   sudo mv secrets /usr/local/bin/
   ```

## ✅ Verify

It's recommended to verify the checksum of the downloaded archive:

1. Download the `checksums.txt` file from the same release.
2. Run the verification command:

   ```bash
   sha256sum --ignore-missing -c checksums.txt
   ```

   Expected output: `secrets_Linux_x86_64.tar.gz: OK`

## 🚀 Quick Start

1. **Initialize environment**:

   ```bash
   # Generate a 32-byte base64 key for localsecrets KMS
   export KMS_KEY=$(openssl rand -base64 32)
   export KMS_PROVIDER=localsecrets
   export KMS_KEY_URI=base64key://$KMS_KEY
   ```

2. **Generate master key**:

   ```bash
   secrets create-master-key --id default --kms-provider localsecrets --kms-key-uri "$KMS_KEY_URI" > .env
   ```

3. **Configure database**:
   Edit the `.env` file to include your database connection string:

   ```bash
   echo "DB_DRIVER=postgres" >> .env
   echo "DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable" >> .env
   ```

4. **Bootstrap**:

   ```bash
   set -a; source .env; set +a
   secrets migrate
   secrets create-kek --algorithm aes-gcm
   ```

5. **Start the server**:

   ```bash
   secrets server
   ```

## See also

- [Docker getting started](docker.md)
- [Local development](local-development.md)
- [Configuration reference](../configuration.md)
- [CLI commands reference](../cli-commands.md)
