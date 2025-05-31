FROM golang:1.24 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache GOBIN=/app/bin go install -ldflags="-X within.website/x.Version=$(git describe --tags --always --dirty)" ./cmd/aura

FROM debian:bookworm AS runtime

WORKDIR /app

RUN apt-get update && apt-get install -y \
  curl \
  wget \
  unzip \
  ca-certificates \
  streamripper \
  vim \
  jq \
  && rm -rf /var/lib/apt/lists/* \
  && cp /etc/ssl/certs/ca-certificates.crt .

COPY --from=build /app/bin/aura /app/bin/aura
CMD ["/app/bin/aura"]

LABEL org.opencontainers.image.source="https://github.com/Xe/x"