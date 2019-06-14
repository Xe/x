FROM xena/go:1.12.6 AS build
WORKDIR /x
COPY . .
ENV GOPROXY=https://cache.greedo.xeserv.us
ENV CGO_ENABLED=0
RUN go mod download
RUN go test ./...
RUN GOBIN=/x/bin go install -v ./...
RUN apk --no-cache add upx \
 && upx /x/bin/*

FROM xena/alpine
COPY --from=build /x/bin/ /usr/local/bin/
RUN apk --no-cache add man
