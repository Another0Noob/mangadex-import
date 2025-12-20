# List recipes by default
default:
    just --list

build-cli:
    go build -o bin/cli ./cmd/cli/.

run-cli:
    go run ./cmd/cli/.

# Install frontend dependencies
install:
    cd web/frontend-vite && pnpm install

# Development mode (frontend + backend)
dev:
    just dev-frontend & just dev-backend

# Run Vite dev server
dev-frontend:
    cd web/frontend-vite && pnpm run dev

# Run Go backend in dev mode
dev-backend:
    go run -tags dev ./cmd/web/.

# Build everything
build-web: build-frontend build-backend

# Build frontend assets
build-frontend:
    cd web/frontend-vite && pnpm run build

# Build Go binary
build-backend:
    go build -o bin/web ./cmd/web/.

# Run production server
run-web: build-frontend
    go run ./cmd/web/.

# Clean build artifacts
clean:
    rm -rf web/frontend-vite/dist
    rm web
    rm cli
