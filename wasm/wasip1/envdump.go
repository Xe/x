//go:build ignore

package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Environ()) == 0 {
		fmt.Println("No environment variables found")
		return
	}
	for _, kv := range os.Environ() {
		fmt.Println(kv)
	}
}
