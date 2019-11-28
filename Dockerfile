FROM xena/go:1.12.6 AS build
WORKDIR /x
ENV GOPROXY=https://cache.greedo.xeserv.us
ENV CGO_ENABLED=0
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go test ./...
RUN GOBIN=/x/bin go install -v ./...

FROM xena/alpine
COPY --from=build /x/bin/ /usr/local/bin/
RUN apk --no-cache add man
