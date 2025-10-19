FROM golang:1.24 AS build
WORKDIR /app

# Cache module download
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build the httpdebug binary
COPY . .
RUN --mount=type=cache,target=/root/.cache \
    GOBIN=/app/bin \
    go install -ldflags="-X within.website/x.Version=$(git describe --tags --always --dirty)" \
    ./cmd/httpdebug

FROM debian:bookworm AS runtime
WORKDIR /app

# Minimal runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && cp /etc/ssl/certs/ca-certificates.crt .

COPY --from=build /app/bin/httpdebug /app/bin/httpdebug
CMD ["/app/bin/httpdebug"]

LABEL org.opencontainers.image.source="https://github.com/Xe/x"

