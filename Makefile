.PHONY: all account auditlog docker-up docker-stop docker-clean nats-sub docker-stats stress-test

# Run all services without Docker
all: account auditlog

# Run account service
account:
	@echo "Starting account service..."
	@cd internal/account/cmd && go run main.go

# Run auditlog service
auditlog:
	@echo "Starting auditlog service..."
	@cd internal/auditlog/cmd && go run main.go

# Start services with Docker Compose
docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up --build --attach-dependencies

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down

# Remove Docker containers, volumes, and networks
docker-clean:
	@echo "Cleaning Docker environment..."
	@docker-compose down -v --remove-orphans

# Show resource usage of specific Docker containers
docker-stats:
	@echo "Displaying Docker container stats..."
	@docker stats omi-backend-assignment-auditlog-1 omi-backend-assignment-account-1

# Run tests with race condition detection
test-race:
	@echo "Running tests with race detection..."
	@go test -race ./internal/auditlog/...

# Stress test: Send multiple PATCH requests
stress-test:
	@echo "Running stress test..."
	@for i in {1..100000}; do \
		curl -X PATCH http://localhost:8080/accounts/4eaa2b93-c0e2-4556-83a3-ecfbc7d60fa3 \
		-H "Content-Type: application/json" \
		-d '{"name": "John Doe", "email": "john.doe@example.com"}' & \
	done; \
	wait
