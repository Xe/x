ARG ALPINE_VERSION=3.22
FROM alpine:${ALPINE_VERSION} AS build

RUN apk -U add build-base gyp pkgconfig python3 nodejs npm

WORKDIR /app
COPY package-lock.json package.json ./
RUN npm ci --include=dev

COPY . .
RUN npm run build \
  && npm prune --omit=dev


ARG ALPINE_VERSION=3.22
FROM alpine:${ALPINE_VERSION}
ENV NODE_ENV="production"
WORKDIR /app

RUN apk -U add nodejs
COPY --from=build /app /app
CMD [ "node", "src/main.js" ]

EXPOSE 3000
LABEL org.opencontainers.image.source="https://github.com/Xe/x"
