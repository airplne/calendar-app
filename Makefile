.PHONY: dev build test lint clean install-deps

# Development: run both Go server and React dev server
dev:
	@echo "Starting Calendar-app in development mode..."
	@echo "Go server will run on http://localhost:8080"
	@echo "React dev server will run on http://localhost:5173"
	@echo ""
	@bash scripts/dev.sh

# Build: compile Go binary with embedded React dist
build:
	@echo "Building React frontend..."
	cd web && npm run build
	@echo "Building Go backend..."
	cd server && go build -o ../calendar-app ./cmd/calendarapp
	@echo "Build complete: ./calendar-app"

# Test: run both Go and React tests
test:
	@echo "Running Go tests..."
	cd server && go test ./...
	@echo "Running React tests..."
	cd web && npm run test

# Lint: run linters for both projects
lint:
	@echo "Linting Go code..."
	cd server && golangci-lint run ./...
	@echo "Linting React code..."
	cd web && npm run lint

# Clean: remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f calendar-app
	rm -rf web/dist
	rm -rf server/calendar-app

# Install dependencies
install-deps:
	@echo "Installing Go dependencies..."
	cd server && go mod download
	@echo "Installing Node dependencies..."
	cd web && npm install
