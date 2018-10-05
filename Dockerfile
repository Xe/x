FROM xena/go:1.11.1 AS build

WORKDIR /x
COPY . .
ENV CGO_ENABLED=0
RUN GOBIN=/x/bin go install -v -mod=vendor ./...

FROM xena/alpine
COPY --from=build /x/bin/ /usr/local/bin/
