ARG ALPINE_VERSION=3.22
FROM --platform=${BUILDPLATFORM} alpine:${ALPINE_VERSION} AS build

RUN apk add --no-cache go git build-base bash

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/root/go \
  go mod download

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=devel-docker

COPY . .
RUN --mount=type=cache,target=/root/.cache \
  --mount=type=cache,target=/root/go \
  GOOS=${TARGETOS} \
  GOARCH=${TARGETARCH} \
  CGO_ENABLED=0 \
  go build \
  -o /app/bin/sakurajima \
  -ldflags "-s -w -extldflags -static -X within.website/x.Version=$(git describe --tags --always --dirty)" \
  ./cmd/sakurajima

FROM alpine:${ALPINE_VERSION} AS run
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=build /app/bin/sakurajima /app/bin/sakurajima

CMD ["/app/bin/sakurajima"]

LABEL org.opencontainers.image.source="https://github.com/Xe/x" \
  org.opencontainers.image.description="Sakurajima application" \
  org.opencontainers.image.licenses="CC0"
