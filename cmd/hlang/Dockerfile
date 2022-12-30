FROM xena/go:1.13.3 AS build
WORKDIR /hlang
COPY . .
RUN GOBIN=/usr/local/bin go install .

FROM xena/alpine AS wasm
WORKDIR /wabt
RUN apk --no-cache add build-base cmake git python \
 && git clone --recursive https://github.com/WebAssembly/wabt /wabt \
 && mkdir build \
 && cd build \
 && cmake .. \
 && make && make install
RUN ldd $(which wat2wasm)

FROM xena/alpine
COPY --from=wasm /usr/local/bin/wat2wasm /usr/local/bin/wat2wasm
COPY --from=build /usr/local/bin/hlang /usr/local/bin/hlang
ENV PORT 5000
RUN apk --no-cache add libstdc++
CMD /usr/local/bin/hlang
