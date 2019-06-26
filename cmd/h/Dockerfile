ARG X_VERSION
FROM xena/xperimental:$X_VERSION AS x

FROM xena/alpine AS build
WORKDIR /wabt
RUN apk --no-cache add build-base cmake git python \
 && git clone --recursive https://github.com/WebAssembly/wabt /wabt \
 && mkdir build \
 && cd build \
 && cmake .. \
 && make && make install
RUN ldd $(which wat2wasm)

FROM xena/alpine
COPY --from=build /usr/local/bin/wat2wasm /usr/local/bin/wat2wasm
COPY --from=x /usr/local/bin/h /usr/local/bin/h
ENV PORT 5000
CMD /usr/local/bin/h
