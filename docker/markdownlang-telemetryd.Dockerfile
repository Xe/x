FROM golang:1.25 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache GOBIN=/app/bin go install -ldflags="-X within.website/x.Version=$(git describe --tags --always --dirty)" ./cmd/markdownlang-telemetryd

FROM debian:bookworm AS runtime

WORKDIR /app

RUN apt-get update && apt-get install -y \
  ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && cp /etc/ssl/certs/ca-certificates.crt .

COPY --from=build /app/bin/markdownlang-telemetryd /app/bin/markdownlang-telemetryd

# Run as non-root user
USER nobody:nogroup

CMD ["/app/bin/markdownlang-telemetryd"]

LABEL org.opencontainers.image.source="https://github.com/Xe/x"
