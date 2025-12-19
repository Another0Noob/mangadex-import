.PHONY: help dev dev-frontend dev-backend build build-frontend build-backend run clean install

# Default target
help:
	@echo "Available targets:"
	@echo "  make dev            - Run both frontend and backend in development mode"
	@echo "  make dev-frontend   - Run only Vite dev server"
	@echo "  make dev-backend    - Run only Go backend in dev mode"
	@echo "  make build          - Build production bundle"
	@echo "  make build-frontend - Build only frontend"
	@echo "  make build-backend  - Build only Go backend"
	@echo "  make run            - Run production server"
	@echo "  make install        - Install frontend dependencies"
	@echo "  make clean          - Clean build artifacts"

# Install frontend dependencies
install:
	cd frontend-vite && pnpm install

# Development mode - runs both frontend and backend concurrently
dev:
	@echo "Starting development servers..."
	@make -j2 dev-frontend dev-backend

# Run Vite dev server
dev-frontend:
	@echo "Starting Vite dev server on http://localhost:5173"
	cd frontend-vite && pnpm run dev

# Run Go backend in dev mode
dev-backend:
	@echo "Starting Go backend on http://localhost:3939"
	go run main.go -dev

# Build everything for production
build: build-frontend build-backend

# Build frontend assets
build-frontend:
	@echo "Building frontend..."
	cd frontend-vite && pnpm run build

# Build Go binary
build-backend:
	@echo "Building backend..."
	go build -o web main.go

# Run production server
run:
	@echo "Starting production server on http://localhost:3939"
	go run main.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf frontend-vite/dist
	rm web
