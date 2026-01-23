# CineStream

A video streaming platform with transcoding capabilities built with Go.

## Prerequisites

- Go 1.24.9 or higher
- MySQL 5.7 or higher
- Redis (for caching and job queue)
- MinIO (for object storage)
- Goose (for database migrations)

## Getting Started

### 1. Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install Goose for migrations (if not installed)
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### 2. Configure Application

Edit `app-config.yaml` to match your environment:

```yaml
server:
  port: "8080"

database:
  host: "localhost"
  port: "3306"
  user: "root"
  password: "password"
  dbname: "test"
  max_idle_conns: 10
  max_open_conns: 100

# ... other configurations
```

### 3. Setup Database

```bash
# Create database and run migrations
make setup
```

Or manually:

```bash
# Create database
make db_create

# Run migrations
make goose_up
```

### 4. Run the Application

```bash
# Using Make
make run

# Or directly with Go
go run cmd/api/*.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Health Check
```
GET /health
```

### User Registration
```
POST /api/v1/users/register
Content-Type: application/json

{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

**Response (201 Created):**
```json
{
  "status": "success",
  "code": 201,
  "message": "user_registered_successfully",
  "data": {
    "ext_id": "user_2XYZ123ABC...",
    "name": "John Doe",
    "email": "john@example.com"
  }
}
```

## Available Make Commands

- `make help` - Show available commands
- `make db_create` - Create the database
- `make goose_up` - Run migrations
- `make goose_reset` - Reset migrations
- `make goose_create name=<migration_name>` - Create a new migration
- `make run` - Run the API server
- `make setup` - Create database and run migrations

## Project Structure

```
cinestream/
├── cmd/
│   ├── api/              # API server entry point
│   └── worker/           # Background worker
├── internal/
│   ├── domain/           # Business logic layer
│   │   ├── users/
│   │   └── orders/
│   └── platform/         # Infrastructure layer
│       ├── config/
│       ├── database/
│       ├── queue/
│       └── storage/
├── pkg/                  # Shared packages
│   ├── jwt/
│   └── response/
├── migration/            # Database migrations
```

## Development

### Creating a New Migration

```bash
make goose_create name=create_new_table
```

### Running Tests

```bash
go test ./...
```

## Error Handling

The API uses a standardized error response format:

```json
{
  "status": "error",
  "code": 400,
  "message": "error_message",
  "errors": {
    // error details
  }
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `409` - Conflict (e.g., email already exists)
- `500` - Internal Server Error

## License

MIT
