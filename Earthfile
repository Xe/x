VERSION 0.8
FROM debian:bookworm
WORKDIR /app

runtime:
    FROM debian:bookworm

    RUN apt-get update && apt-get install -y \
        curl \
        wget \
        unzip \
        ca-certificates \
        && rm -rf /var/lib/apt/lists/*

    WORKDIR /app

deps:
    FROM golang:1.23
    WORKDIR /app

    COPY go.mod go.sum ./
    RUN go mod download

    SAVE ARTIFACT go.mod

everything:
    FROM +deps

    COPY . .
    RUN mkdir -p bin
    RUN --mount=type=cache,target=/root/.cache GOBIN=$(pwd)/bin go install ./... 

    SAVE ARTIFACT bin

azurda:
    FROM +runtime
    
    COPY +everything/bin/azurda /app/bin/azurda
    CMD ["/app/bin/azurda"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/azurda:latest

aerial:
    FROM +runtime

    COPY +everything/bin/aerial /app/bin/aerial
    CMD ["/app/bin/aerial"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/aerial:latest

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

    SAVE IMAGE --push registry.fly.io/hdrwtch:latest

hlang:
    FROM +runtime

    COPY +everything/bin/hlang /app/bin/hlang
    CMD ["/app/bin/hlang", "--port=8080"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/hlang:latest

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

sanguisuga:
    FROM +runtime

    COPY +everything/bin/sanguisuga /app/bin/sanguisuga
    CMD ["/app/bin/sanguisuga"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/sanguisuga:latest

sapientwindex:
    FROM +runtime

    COPY +everything/bin/sapientwindex /app/bin/sapientwindex
    CMD ["/app/bin/sapientwindex"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/sapientwindex:latest

todayinmarch2020:
    FROM +runtime

    COPY +everything/bin/todayinmarch2020 /app/bin/todayinmarch2020
    CMD ["/app/bin/todayinmarch2020", "--port=8080"]

    LABEL org.opencontainers.image.source="https://github.com/Xe/x"

    SAVE IMAGE --push ghcr.io/xe/x/todayinmarch2020:latest

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

    SAVE IMAGE --push registry.fly.io/xedn:latest

all:
    BUILD --platform=linux/amd64 +azurda
    BUILD --platform=linux/amd64 +future-sight
    BUILD --platform=linux/amd64 +hdrwtch
    BUILD --platform=linux/amd64 +hlang
    BUILD --platform=linux/amd64 +mi
    BUILD --platform=linux/amd64 +mimi
    BUILD --platform=linux/amd64 +sanguisuga
    BUILD --platform=linux/amd64 +sapientwindex
    BUILD --platform=linux/amd64 +todayinmarch2020
    BUILD --platform=linux/amd64 +within-website
    BUILD --platform=linux/amd64 +xedn