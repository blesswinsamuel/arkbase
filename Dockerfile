# ==========================================
# Stage 1: Build React Frontend (shadcn/ui + Tailwind v4)
# ==========================================
FROM node:24-alpine AS frontend-builder
WORKDIR /app/frontend

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm build

# ==========================================
# Stage 2: Build Golang Binary (Static)
# ==========================================
FROM golang:1.26-alpine AS backend-builder
WORKDIR /app

# Install git for version injection
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

# Copy source code and frontend build output
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

ARG VERSION=dev
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -ldflags "-s -w -X main.Version=${VERSION}" \
    -o /bin/arkbase ./cmd/arkbase

# ==========================================
# Stage 3: Minimal Production Runtime
# ==========================================
FROM alpine:3.21

# Install database clients and SSL certs
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    postgresql17-client

# Create unprivileged application user
RUN addgroup -g 10001 -S arkbase && \
    adduser -u 10001 -S arkbase -G arkbase

COPY --from=backend-builder /bin/arkbase /usr/local/bin/arkbase

# Default data directories
RUN mkdir -p /var/lib/arkbase /backups /etc/arkbase && \
    chown -R arkbase:arkbase /var/lib/arkbase /backups /etc/arkbase

USER arkbase

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/status || exit 1

ENTRYPOINT ["/usr/local/bin/arkbase"]
CMD ["run", "--config", "/etc/arkbase/config.yaml"]
