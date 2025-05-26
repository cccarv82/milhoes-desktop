# =============================================================================
# 🏗️ MULTI-STAGE BUILD DOCKERFILE
# Lottery Optimizer - AI-Powered Brazilian Lottery Strategy Optimizer
# =============================================================================

# =====================================================
# 📦 BUILD STAGE
# =====================================================
FROM golang:1.22-alpine AS builder

# 🏷️ Metadata
LABEL stage=builder
LABEL description="Build stage for Lottery Optimizer"

# 🔧 Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# 👤 Create non-root user for security
RUN adduser -D -g '' appuser

# 📁 Set working directory
WORKDIR /build

# 📥 Copy dependency files first (for better caching)
COPY go.mod go.sum ./

# 📦 Download dependencies
RUN go mod download
RUN go mod verify

# 📋 Copy source code
COPY . .

# 🏗️ Build the application
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o lottery-optimizer .

# 🧪 Quick test to ensure binary works
RUN ./lottery-optimizer --help

# =====================================================
# 🐳 PRODUCTION STAGE
# =====================================================
FROM scratch AS production

# 🏷️ Metadata
LABEL org.opencontainers.image.title="Lottery Optimizer"
LABEL org.opencontainers.image.description="AI-powered lottery strategy optimizer for Brazilian lotteries"
LABEL org.opencontainers.image.vendor="Lottery Optimizer Team"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.documentation="https://github.com/seu-usuario/lottery-optimizer"
LABEL org.opencontainers.image.source="https://github.com/seu-usuario/lottery-optimizer"

# 📄 Copy certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# 👤 Copy the non-root user
COPY --from=builder /etc/passwd /etc/passwd

# 📋 Copy documentation
COPY --from=builder /build/README.md /
COPY --from=builder /build/LICENSE /

# 🎯 Copy the binary
COPY --from=builder /build/lottery-optimizer /lottery-optimizer

# 👤 Use non-root user
USER appuser

# 🚪 Expose port (if needed for future web interface)
# EXPOSE 8080

# 🔧 Health check (basic binary execution test)
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/lottery-optimizer", "--help"]

# ⚡ Set entrypoint
ENTRYPOINT ["/lottery-optimizer"]

# 🎯 Default command
CMD ["--help"]

# =====================================================
# 🎯 DEVELOPMENT STAGE (Optional)
# =====================================================
FROM golang:1.22-alpine AS development

# 🏷️ Metadata
LABEL stage=development
LABEL description="Development environment for Lottery Optimizer"

# 🔧 Install development tools
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    curl \
    bash \
    make \
    && update-ca-certificates

# 📁 Set working directory
WORKDIR /app

# 📥 Copy dependency files
COPY go.mod go.sum ./

# 📦 Download dependencies
RUN go mod download

# 📋 Copy source code
COPY . .

# 🔧 Install development tools
RUN go install github.com/air-verse/air@latest
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 🚪 Expose port for development
EXPOSE 8080

# ⚡ Development entrypoint with hot reload
CMD ["air", "-c", ".air.toml"]

# =====================================================
# 🧪 TESTING STAGE (Optional)
# =====================================================
FROM golang:1.22-alpine AS testing

# 🏷️ Metadata
LABEL stage=testing
LABEL description="Testing environment for Lottery Optimizer"

# 🔧 Install test dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# 📁 Set working directory
WORKDIR /app

# 📥 Copy dependency files
COPY go.mod go.sum ./

# 📦 Download dependencies
RUN go mod download

# 📋 Copy source code
COPY . .

# 🧪 Run tests
RUN go test -v ./...

# 🔍 Run linting
RUN go vet ./...

# 🏗️ Test build
RUN go build -o lottery-optimizer .

# ✅ Test binary
RUN ./lottery-optimizer --help

# =====================================================
# 📝 BUILD INSTRUCTIONS
# =====================================================

# 🏗️ Build production image:
# docker build --target production -t lottery-optimizer:latest .

# 🧪 Build and run tests:
# docker build --target testing -t lottery-optimizer:test .

# 🔧 Development environment:
# docker build --target development -t lottery-optimizer:dev .
# docker run -it -v $(pwd):/app lottery-optimizer:dev

# 🚀 Run production container:
# docker run -e CLAUDE_API_KEY="your-key" lottery-optimizer:latest

# 📊 Multi-platform build:
# docker buildx build --platform linux/amd64,linux/arm64 -t lottery-optimizer:latest .

# 🔍 Image analysis:
# docker run --rm -it wagoodman/dive lottery-optimizer:latest 