FROM golang:1.24 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache GOBIN=/app/bin go install -ldflags="-X within.website/x.Version=$(git describe --tags --always --dirty)" ./cmd/mimi

FROM debian:bookworm AS runtime

WORKDIR /app

RUN apt-get update && apt-get install -y \
  imagemagick \
  && rm -rf /var/lib/apt/lists/* 

COPY --from=build /app/bin/mimi /app/bin/mimi
CMD ["/app/bin/mimi"]

LABEL org.opencontainers.image.source="https://github.com/Xe/x"