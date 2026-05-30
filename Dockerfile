# Lintasan Go — multi-stage build producing a single self-contained image.
#
# Stage 1 builds the SvelteKit dashboard into a static SPA.
# Stage 2 embeds that SPA into the Go binary via go:embed and compiles it.
# Stage 3 is a tiny runtime image with just the binary + SQLite libs.
#
# Result: one container that serves the full dashboard UI + API on :20180.
# No separate Node process, no nginx required for a basic deployment.

# ---- Stage 1: frontend (SvelteKit static SPA) ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# ---- Stage 2: backend (Go, embeds the SPA) ----
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Drop in the freshly-built dashboard so go:embed picks it up.
RUN rm -rf internal/web/dist && mkdir -p internal/web/dist
COPY --from=frontend /app/frontend/build/ internal/web/dist/
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /app/bin/lintasan ./cmd/lintasan/

# ---- Stage 3: runtime ----
FROM alpine:3.20
RUN apk add --no-cache sqlite-libs ca-certificates
WORKDIR /app
COPY --from=builder /app/bin/lintasan /app/lintasan
RUN mkdir -p /app/data
EXPOSE 20180
ENV PORT=20180
ENV LINTASAN_DATA_DIR=/app/data
# A master key is required to reach ACTIVE state. Override at runtime:
#   docker run -e LINTASAN_MASTER_KEY=... -p 20180:20180 lintasan
ENTRYPOINT ["/app/lintasan"]
CMD ["start"]
