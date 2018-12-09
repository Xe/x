FROM xena/go:1.11.2 AS build
WORKDIR /x
COPY . .
ENV GOPROXY=https://cache.greedo.xeserv.us
ENV CGO_ENABLED=0
RUN GOBIN=/x/bin go install -v ./...
RUN apk --no-cache add mdocml && cd ./docs/man && ./prepare.sh

FROM xena/alpine
COPY --from=build /x/bin/ /usr/local/bin/
COPY --from=build /x/docs/man /usr/share/man/man1
RUN apk --no-cache add man
