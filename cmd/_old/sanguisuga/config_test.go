package main

import (
	"testing"

	"go.jetpack.io/tyson"
)

func TestDefaultConfig(t *testing.T) {
	var c Config
	if err := tyson.Unmarshal("./config.default.ts", &c); err != nil {
		t.Fatal(err)
	}
}
