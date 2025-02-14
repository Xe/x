VERSION 0.8
FROM debian:bookworm
WORKDIR /app

runtime:
    FROM debian:bookworm

    WORKDIR /app

    RUN apt-get update && apt-get install -y \
        curl \
        wget \
        unzip \
        ca-certificates \
        && rm -rf /var/lib/apt/lists/* \
        && cp /etc/ssl/certs/ca-certificates.crt .

    SAVE ARTIFACT ca-certificates.crt

deps:
    FROM golang:1.24
    WORKDIR /app

    COPY go.mod go.sum ./
    RUN go mod download

    SAVE ARTIFACT go.mod

build:
    FROM +deps
    WORKDIR /app

    ARG PROGRAM=anubis
    ARG GOOS=linux
    ARG GOARCH=amd64
    ARG GOARM

    COPY . .
    ARG VERSION=$(git describe --tags --always --dirty)
    RUN --mount=type=cache,target=/root/.cache go build -o /app/bin/${PROGRAM} -ldflags="-X within.website/x.Version=${VERSION}" ./cmd/${PROGRAM}

    SAVE ARTIFACT bin

ship:
    ARG PROGRAM=anubis
    ARG GOARCH

    FROM --platform=linux/${GOARCH} debian:bookworm
    COPY --platform=${TARGETPLATFORM} (+runtime/ca-certificates.crt) /etc/ssl/certs/ca-certificates.crt
    COPY --platform=${TARGETPLATFORM} (+build/bin/${PROGRAM} --GOARCH=${GOARCH} --PROGRAM=${PROGRAM}) /app/bin/${PROGRAM}

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

everything:
    FROM +deps

    COPY . .
    ARG VERSION=$(git describe --tags --always --dirty)
    RUN mkdir -p bin
    RUN --mount=type=cache,target=/root/.cache GOBIN=$(pwd)/bin go install -ldflags="-X within.website/x.Version="${VERSION} ./...

    SAVE ARTIFACT bin

aerial:
    BUILD +ship --PROGRAM=aerial --GOARCH=amd64

amano:
    BUILD +ship --PROGRAM=amano --GOARCH=amd64

anubis-amd64:
    FROM +ship --PROGRAM=anubis --GOARCH=amd64
    CMD ["/app/bin/anubis"]
    USER 1000:1000

    SAVE IMAGE --push ghcr.io/xe/x/anubis:latest

anubis-arm64:
    FROM +ship --PROGRAM=anubis --GOARCH=arm64
    CMD ["/app/bin/anubis"]
    USER 1000:1000

    SAVE IMAGE --push ghcr.io/xe/x/anubis:latest

anubis:
    BUILD +anubis-amd64
    BUILD +anubis-arm64

aura:
    FROM +runtime

    RUN apt-get update && apt-get install -y \
        streamripper vim jq \
        && rm -rf /var/lib/apt/lists/*

    COPY +everything/bin/aura /app/bin/aura
    CMD ["/app/bin/aura"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/aura:latest

future-sight:
    FROM +runtime

    COPY +everything/bin/future-sight /app/bin/future-sight
    CMD ["/app/bin/future-sight"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/future-sight:latest

hdrwtch:
    FROM +runtime

    COPY +everything/bin/hdrwtch /app/bin/hdrwtch
    CMD ["/app/bin/hdrwtch", "--port=8080", "--database-loc=/data/hdrwtch.db"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/hdrwtch:latest

hlang:
    FROM +runtime

    COPY +everything/bin/hlang /app/bin/hlang
    CMD ["/app/bin/hlang", "--port=8080"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/hlang:latest

httpdebug:
    FROM +ship --PROGRAM=httpdebug --GOARCH=amd64
    CMD ["/app/bin/httpdebug"]
    SAVE IMAGE --push ghcr.io/xe/x/httpdebug:latest

mi:
    FROM +runtime

    COPY +everything/bin/mi /app/bin/mi
    CMD ["/app/bin/mi"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/mi:latest

mimi:
    FROM +runtime

    RUN apt-get update && apt-get install -y \
        imagemagick \
        && rm -rf /var/lib/apt/lists/*

    COPY +everything/bin/mimi /app/bin/mimi
    CMD ["/app/bin/mimi"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/mimi:latest

relayd:
    FROM +runtime

    COPY +everything/bin/relayd /app/bin/relayd
    CMD ["/app/bin/relayd"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/relayd:latest

sapientwindex:
    FROM +runtime

    COPY +everything/bin/sapientwindex /app/bin/sapientwindex
    CMD ["/app/bin/sapientwindex"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/sapientwindex:latest

stealthmountain:
    FROM +runtime

    COPY +everything/bin/stealthmountain /app/bin/stealthmountain
    CMD ["/app/bin/stealthmountain"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/stealthmountain:latest

stickers:
    FROM +runtime

    COPY +everything/bin/stickers /app/bin/stickers
    CMD ["/app/bin/stickers"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/stickers:latest

todayinmarch2020:
    FROM +runtime

    COPY +everything/bin/todayinmarch2020 /app/bin/todayinmarch2020
    CMD ["/app/bin/todayinmarch2020", "--port=8080"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/todayinmarch2020:latest

uncle-ted:
    FROM +runtime

    COPY +everything/bin/uncle-ted /app/bin/uncle-ted
    CMD ["/app/bin/uncle-ted"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/uncle-ted:latest

within-website:
    FROM +runtime

    COPY +everything/bin/within.website /app/bin/within.website
    CMD ["/app/bin/within.website", "--port=8080"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/within-website:latest

xedn-static:
    RUN apt-get update && apt-get install -y \
        tar \
        gzip \ 
        wget \
        curl \
        && rm -rf /var/lib/apt/lists/*

    RUN mkdir -p /app/static

    RUN wget https://registry.npmjs.org/@xeserv/xeact/-/xeact-0.69.71.tgz \
     && tar -xzf xeact-0.69.71.tgz \
     && mkdir -p /app/static/pkg/xeact/0.69.71 \
     && cp -r package/* /app/static/pkg/xeact/0.69.71 \
     && rm -rf xeact-0.69.71.tgz package

    RUN wget https://registry.npmjs.org/@xeserv/xeact/-/xeact-0.70.0.tgz \
     && tar -xzf xeact-0.70.0.tgz \
     && mkdir -p /app/static/pkg/xeact/0.70.0 \
     && cp -r package/* /app/static/pkg/xeact/0.70.0 \
     && rm -rf xeact-0.70.0.tgz package

    RUN wget https://registry.npmjs.org/@xeserv/xeact/-/xeact-0.71.0.tgz \
     && tar -xzf xeact-0.71.0.tgz \
     && mkdir -p /app/static/pkg/xeact/0.71.0 \
     && cp -r package/* /app/static/pkg/xeact/0.71.0 \
     && rm -rf xeact-0.71.0.tgz package

    RUN mkdir -p /app/static/css /app/static/pkg/iosevka

    RUN wget https://cdn.xeiaso.net/file/christine-static/dl/iosevka-iaso.tgz \
     && (cd /app/static/pkg/iosevka && tar -xzf /app/iosevka-iaso.tgz) \
     && ln -s /app/static/pkg/iosevka /app/static/css/iosevka \
     && rm -f /iosevka-iaso.tgz

    COPY ./xess/dist/*.css /app/static/pkg/xess/
    COPY ./xess/xess.css /app/static/css/xess.css
    COPY ./xess/xess.css /app/static/pkg/xess/xess.css
    COPY ./xess/static/podkova.css /app/static/pkg/podkova/family.css
    COPY ./xess/static/podkova.woff2 /app/static/pkg/podkova/podkova.woff2
    RUN ln -s /app/static/pkg/podkova /app/static/css/podkova

    SAVE ARTIFACT /app/static

xedn:
    FROM +runtime

    COPY +everything/bin/xedn /app/bin/xedn
    COPY +everything/bin/uplodr /app/bin/uplodr
    COPY +xedn-static/static /app/static
    CMD ["/app/bin/xedn"]
    ENV XEDN_STATIC=/app
    ENV UPLODR_BINARY=/app/bin/uplodr

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/xedn:latest

all:
    BUILD --pass-args --platform=linux/amd64 +aerial
    BUILD --pass-args --platform=linux/amd64 +amano
    BUILD --pass-args --platform=linux/amd64 +anubis
    BUILD --pass-args --platform=linux/amd64 +aura
    BUILD --pass-args --platform=linux/amd64 +future-sight
    BUILD --pass-args --platform=linux/amd64 +hdrwtch
    BUILD --pass-args --platform=linux/amd64 +hlang
    BUILD --pass-args --platform=linux/amd64 +httpdebug
    BUILD --pass-args --platform=linux/amd64 +mi
    BUILD --pass-args --platform=linux/amd64 +mimi
    BUILD --pass-args --platform=linux/amd64 +relayd
    BUILD --pass-args --platform=linux/amd64 +sapientwindex
    BUILD --pass-args --platform=linux/amd64 +stealthmountain
    BUILD --pass-args --platform=linux/amd64 +stickers
    BUILD --pass-args --platform=linux/amd64 +todayinmarch2020
    BUILD --pass-args --platform=linux/amd64 +uncle-ted
    BUILD --pass-args --platform=linux/amd64 +within-website
    BUILD --pass-args --platform=linux/amd64 +xedn

    BUILD --pass-args --platform=linux/amd64 ./kube/alrest/staticsites/caddy1+all
    BUILD --pass-args --platform=linux/amd64 ./migroserbices/falin+run