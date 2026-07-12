FROM golang:1.25 AS build
WORKDIR /app

# Cache module download
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build the iamd binary
COPY . .
RUN --mount=type=cache,target=/root/.cache \
    GOBIN=/app/bin \
    go install ./cmd/iamd

FROM debian:bookworm AS runtime
WORKDIR /app

# Minimal runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && cp /etc/ssl/certs/ca-certificates.crt . \
    && mkdir -p /data

COPY --from=build /app/bin/iamd /app/bin/iamd

# SQLite database lives on /data so it survives container replacement.
VOLUME /data

# 9080: IAM/STS Twirp API, 9081: Prometheus metrics
EXPOSE 9080 9081

CMD ["/app/bin/iamd", "-db-loc", "/data/iamd.db"]

LABEL org.opencontainers.image.source="https://github.com/Xe/x"
