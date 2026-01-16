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
	cd web && pnpm run build
	@echo "Preparing embedded web UI assets..."
	@mkdir -p server/internal/webui/dist
	@find server/internal/webui/dist -mindepth 1 ! -name placeholder.txt -exec rm -rf {} +
	@cp -a web/dist/. server/internal/webui/dist/
	@echo "Building Go backend..."
	cd server && go build -o ../calendar-app ./cmd/calendarapp
	@echo "Build complete: ./calendar-app"

# Test: run both Go and React tests
test:
	@echo "Running Go tests..."
	cd server && go test ./...
	@echo "Running React tests..."
	cd web && pnpm run test

# Lint: run linters for both projects
lint:
	@echo "Linting Go code..."
	cd server && go vet ./...
	@echo "Linting React code..."
	cd web && pnpm run lint

# Clean: remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f calendar-app
	rm -rf web/dist
	rm -rf server/calendar-app
	@find server/internal/webui/dist -mindepth 1 ! -name placeholder.txt -exec rm -rf {} +

# Install dependencies
install-deps:
	@echo "Installing Go dependencies..."
	cd server && go mod download
	@echo "Installing Node dependencies..."
	cd web && pnpm install
