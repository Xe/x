GOBIN = gowasi

export GOARCH = wasm
export GOOS = wasip1

SOURCES := $(wildcard *.go)
OBJECTS := $(patsubst %.go, %.wasm, $(SOURCES))

%.wasm: %.go
	$(GOBIN) build -o $@ $^

all: $(OBJECTS)
