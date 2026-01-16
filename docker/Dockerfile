# Multi-stage Dockerfile for Calendar-app
# Stage 1: Build React frontend
# Stage 2: Build Go backend with embedded frontend
# Stage 3: Minimal runtime image (distroless)

# Stage 1: Build frontend
FROM node:18-alpine AS frontend-build

WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml* ./
RUN npm install -g pnpm && pnpm install --frozen-lockfile

COPY web/. .
RUN pnpm run build

# Stage 2: Build Go backend
FROM golang:1.22-alpine AS backend-build

WORKDIR /app
COPY server/go.mod server/go.sum ./server/
RUN cd server && go mod download

COPY server/. ./server/
COPY --from=frontend-build /app/web/dist ./server/web/dist

# Build static binary (CGO disabled for pure Go SQLite driver)
RUN cd server && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/calendar-app ./cmd/calendarapp

# Stage 3: Minimal runtime
FROM gcr.io/distroless/base-debian11

WORKDIR /app
COPY --from=backend-build /app/calendar-app /app/calendar-app

# Create data directory
VOLUME ["/data"]

# Run as non-root
USER nonroot:nonroot

# Expose HTTP port
EXPOSE 8080

# Set default environment variables
ENV CALENDARAPP_PORT=8080
ENV CALENDARAPP_DATA_DIR=/data

ENTRYPOINT ["/app/calendar-app"]
