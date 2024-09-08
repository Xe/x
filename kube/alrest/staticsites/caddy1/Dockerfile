FROM debian:bookworm AS build

WORKDIR /app
RUN mkdir -p /app
ADD https://github.com/caddyserver/caddy/releases/download/v1.0.4/caddy_v1.0.4_linux_amd64.tar.gz /app/caddy1.tgz
RUN tar xf /app/caddy1.tgz

FROM debian:bookworm
LABEL org.opencontainers.image.source="https://github.com/Xe/x"
WORKDIR /app
RUN mkdir -p /app/bin
COPY --from=build /app/caddy /app/bin/caddy
CMD ["/app/bin/caddy", "-port", "8080", "-agree", "-conf", "/app/etc/Caddyfile"]