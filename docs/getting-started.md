# 🚀 Getting Started

This guide will help you set up and run the Go project template on your local machine.

## 📋 Prerequisites

Before you begin, ensure you have the following installed:

- ✅ **Go 1.25 or higher** - [Download Go](https://golang.org/dl/)
- ✅ **PostgreSQL 12+** or **MySQL 8.0+** - For development
- ✅ **Docker and Docker Compose** - For testing and optional development
- ✅ **Make** (optional) - For convenience commands

## 📥 Installation

### 1. Clone the repository

```bash
git clone https://github.com/allisson/go-project-template.git
cd go-project-template
```

### 2. Customize the module path

After cloning, update the import paths to match your project.

#### Option 1: Using find and sed (Linux/macOS)

```bash
# Replace with your actual module path
NEW_MODULE="github.com/yourname/yourproject"

# Update go.mod
sed -i "s|github.com/allisson/go-project-template|$NEW_MODULE|g" go.mod

# Update all Go files
find . -type f -name "*.go" -exec sed -i "s|github.com/allisson/go-project-template|$NEW_MODULE|g" {} +
```

#### Option 2: Using PowerShell (Windows)

```powershell
# Replace with your actual module path
$NEW_MODULE = "github.com/yourname/yourproject"

# Update go.mod
(Get-Content go.mod) -replace 'github.com/allisson/go-project-template', $NEW_MODULE | Set-Content go.mod

# Update all Go files
Get-ChildItem -Recurse -Filter *.go | ForEach-Object {
    (Get-Content $_.FullName) -replace 'github.com/allisson/go-project-template', $NEW_MODULE | Set-Content $_.FullName
}
```

#### Option 3: Manually

1. Update the module name in `go.mod`
2. Search and replace `github.com/allisson/go-project-template` with your module path in all `.go` files

After updating, verify the changes and tidy dependencies:

```bash
go mod tidy
```

**Important**: Also update the `.golangci.yml` file to match your new module path:

```yaml
formatters:
  settings:
    goimports:
      local-prefixes:
        - github.com/yourname/yourproject  # Update this line
```

This ensures the linter correctly groups your local imports.

### 3. Install dependencies

```bash
go mod download
```

## ⚙️ Configuration

### Environment Variables

The application automatically loads environment variables from a `.env` file. Create a `.env` file in your project root:

```bash
# Database configuration
DB_DRIVER=postgres  # or mysql
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable
DB_MAX_OPEN_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONN_MAX_LIFETIME=5

# Server configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Logging
LOG_LEVEL=info

# Worker configuration
WORKER_INTERVAL=5
WORKER_BATCH_SIZE=10
WORKER_MAX_RETRIES=3
WORKER_RETRY_INTERVAL=1
```

**Note**: The application searches for the `.env` file recursively from the current working directory up to the root directory. This allows you to run the application from any subdirectory.

### Configuration Options

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | HTTP server host | `0.0.0.0` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `DB_DRIVER` | Database driver (`postgres`/`mysql`) | `postgres` |
| `DB_CONNECTION_STRING` | Database connection string | - |
| `DB_MAX_OPEN_CONNECTIONS` | Max open connections | `25` |
| `DB_MAX_IDLE_CONNECTIONS` | Max idle connections | `5` |
| `DB_CONN_MAX_LIFETIME` | Connection max lifetime (minutes) | `5` |
| `LOG_LEVEL` | Log level (`debug`/`info`/`warn`/`error`) | `info` |
| `WORKER_INTERVAL` | Worker poll interval (seconds) | `5` |
| `WORKER_BATCH_SIZE` | Events to process per batch | `10` |
| `WORKER_MAX_RETRIES` | Max retry attempts | `3` |
| `WORKER_RETRY_INTERVAL` | Retry interval (seconds) | `1` |

## 🗄️ Database Setup

### Using Docker (Recommended)

#### PostgreSQL

```bash
make dev-postgres
```

This starts PostgreSQL on port `5432` with the following credentials:
- **User**: `postgres`
- **Password**: `postgres`
- **Database**: `mydb`

#### MySQL

```bash
make dev-mysql
```

This starts MySQL on port `3306` with the following credentials:
- **User**: `root`
- **Password**: `root`
- **Database**: `mydb`

### Using Local Installation

If you have PostgreSQL or MySQL installed locally, create a database and update your `.env` file with the appropriate connection string.

**PostgreSQL**:
```sql
CREATE DATABASE mydb;
```

**MySQL**:
```sql
CREATE DATABASE mydb CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## 🔄 Database Migrations

Run database migrations to create the required tables:

```bash
make run-migrate
```

This command:
1. Connects to your database using the `DB_CONNECTION_STRING`
2. Runs all pending migrations from the appropriate directory (`migrations/postgresql` or `migrations/mysql`)
3. Creates the `users` and `outbox_events` tables

## ▶️ Running the Application

### Start the HTTP Server

```bash
make run-server
```

The server will be available at http://localhost:8080

**Health Check**:
```bash
curl http://localhost:8080/health
```

**Readiness Check**:
```bash
curl http://localhost:8080/ready
```

### Start the Background Worker

In another terminal, start the outbox event processor:

```bash
make run-worker
```

The outbox event processor handles asynchronous event processing from the outbox table using the transactional outbox pattern.

## 🧪 Testing the API

### Register a User

```bash
curl -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "SecurePass123!"
  }'
```

**Success Response** (201 Created):
```json
{
  "id": "01936a99-8c2f-7890-b123-456789abcdef",
  "name": "John Doe",
  "email": "john@example.com",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Password Requirements**:
- ✅ Minimum 8 characters
- ✅ At least one uppercase letter
- ✅ At least one lowercase letter
- ✅ At least one number
- ✅ At least one special character

**Validation Error Response** (422 Unprocessable Entity):
```json
{
  "error": "invalid_input",
  "message": "email: must be a valid email address; password: password must contain at least one uppercase letter."
}
```

## 🐳 Docker Deployment

### Build Docker Image

```bash
make docker-build
```

### Run Server in Docker

```bash
make docker-run-server
```

### Run Worker in Docker

```bash
make docker-run-worker
```

### Run Migrations in Docker

```bash
make docker-run-migrate
```

## 🔧 CLI Commands

The binary supports three commands via `urfave/cli`:

### Start HTTP Server
```bash
./bin/app server
```

### Run Database Migrations
```bash
./bin/app migrate
```

### Run Event Worker
```bash
./bin/app worker
```

This starts the outbox event processor which handles asynchronous event processing using the transactional outbox pattern.

## 📚 Next Steps

Now that you have the application running, you can:

- 📖 Learn about the [Architecture](architecture.md)
- 🛠️ Set up your [Development Environment](development.md)
- ✅ Learn about [Testing](testing.md)
- ⚠️ Understand [Error Handling](error-handling.md)
- ➕ Learn how to [Add New Domains](adding-domains.md)

## 🆘 Troubleshooting

### Database Connection Issues

**Problem**: `failed to connect to database`

**Solutions**:
- ✅ Verify database is running: `docker ps` (if using Docker)
- ✅ Check connection string in `.env` file
- ✅ Ensure database credentials are correct
- ✅ Check firewall settings

### Port Already in Use

**Problem**: `bind: address already in use`

**Solutions**:
- ✅ Change `SERVER_PORT` in `.env` file
- ✅ Stop other services using the same port
- ✅ Use `lsof -i :8080` (Linux/macOS) or `netstat -ano | findstr :8080` (Windows) to find the process

### Migration Failures

**Problem**: `migration failed`

**Solutions**:
- ✅ Check database connectivity
- ✅ Verify you're using the correct database driver (`postgres` or `mysql`)
- ✅ Ensure database user has sufficient permissions
- ✅ Check migration files for syntax errors

### Module Import Errors

**Problem**: `cannot find package`

**Solutions**:
- ✅ Run `go mod tidy` to sync dependencies
- ✅ Verify you updated all import paths after cloning
- ✅ Check `.golangci.yml` has correct module path
- ✅ Clear Go cache: `go clean -modcache`
