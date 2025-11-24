.PHONY: goose_up goose_reset goose_create db_create run setup help api-build api-run worker-build worker-run worker-dev

# Database connection string
DB_DSN := root:password@tcp(localhost:3306)/cinestream?parseTime=true

# Create database
db_create:
	@echo "Creating database cinestream_db..."
	@mysql -u root -ppassword -e "CREATE DATABASE IF NOT EXISTS cinestream_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
	@echo "Database created successfully!"

# Run migrations
goose_up:
	cd ./migration && goose mysql "$(DB_DSN)" up

# Reset migrations
goose_reset:
	cd ./migration && goose mysql "$(DB_DSN)" reset

# Create new migration
goose_create:
	cd ./migration && goose create $(name) sql

# Run the API server
run:
	go run cmd/api/*.go

# Build API server
api-build:
	@echo "Building API server..."
	@go build -o bin/api ./cmd/api
	@echo "API server built successfully: bin/api"

# Run API server from binary
api-run: api-build
	@echo "Starting API server..."
	@./bin/api

# Build Worker service
worker-build:
	@echo "Building Worker service..."
	@go build -o bin/worker ./cmd/worker
	@echo "Worker service built successfully: bin/worker"

# Run Worker service from binary
worker-run: worker-build
	@echo "Starting Worker service..."
	@./bin/worker

# Run Worker service in development mode
worker-dev:
	@echo "Starting Worker service in dev mode..."
	@go run cmd/worker/*.go

# Setup everything (create db and run migrations)
setup: db_create goose_up
	@echo "Setup complete!"

# Help command
help:
	@echo "Available commands:"
	@echo "  make db_create     - Create the database"
	@echo "  make goose_up      - Run migrations"
	@echo "  make goose_reset   - Reset migrations"
	@echo "  make goose_create  - Create a new migration"
	@echo "  make run           - Run the API server (dev mode)"
	@echo "  make api-build     - Build API server binary"
	@echo "  make api-run       - Build and run API server"
	@echo "  make worker-build  - Build Worker service binary"
	@echo "  make worker-run    - Build and run Worker service"
	@echo "  make worker-dev    - Run Worker service in dev mode"
	@echo "  make setup         - Create database and run migrations"
