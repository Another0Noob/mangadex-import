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
	@make -j2 dev-frontend dev-backend
	@echo "Server running at: http://localhost:3939"

# Run Vite dev server
dev-frontend:
	cd frontend-vite && pnpm run dev

# Run Go backend in dev mode
dev-backend:
	go run main.go -dev

# Build everything for production
build: build-frontend build-backend

# Build frontend assets
build-frontend:
	cd frontend-vite && pnpm run build

# Build Go binary
build-backend:
	go build -o web main.go

# Run production server
run: build-frontend
	go run main.go

# Clean build artifacts
clean:
	rm -rf frontend-vite/dist
	rm web
