//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("你叫什么名字？")
	os.Stdout.Sync()
	名字, _ := reader.ReadString('\n')
	fmt.Printf("你好%s!", strings.TrimSpace(名字))
}
