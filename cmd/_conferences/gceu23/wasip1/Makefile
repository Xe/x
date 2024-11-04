GOBIN = gowasi

export GOARCH = wasm
export GOOS = wasip1

SOURCES := $(wildcard *.go)
OBJECTS := $(patsubst %.go, %.wasm, $(SOURCES))

%.wasm: %.rs
	rustc --target=wasm32-wasi $^

%.wasm: %.go
	$(GOBIN) build -o $@ $^

all: $(OBJECTS)
