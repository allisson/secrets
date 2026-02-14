# 💻 Run Locally (Development)

> Last updated: 2026-02-14

Use this path if you want to modify the source code and run from your workstation.

## Prerequisites

- Go 1.25+
- Docker (for local database)

## 1) Clone and install dependencies

```bash
git clone https://github.com/allisson/secrets.git
cd secrets
go mod download
```

## 2) Build

```bash
make build
```

## 3) Generate master key and set `.env`

```bash
./bin/app create-master-key --id default
cp .env.example .env
```

Paste generated `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID` into `.env`.

## 4) Start PostgreSQL

```bash
make dev-postgres
```

Default connection in `.env` can be:

```dotenv
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable
```

## 5) Migrate and create KEK

```bash
./bin/app migrate
./bin/app create-kek --algorithm aes-gcm
```

## 6) Start server

```bash
./bin/app server
```

## 7) Smoke test

```bash
curl http://localhost:8080/health
```
