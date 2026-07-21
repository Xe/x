package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
)

func main() {
	accessKeySwitch := flag.Bool("a", false, "Gen a fake AWS style access key")
	secretKeySwitch := flag.Bool("s", false, "Gen a fake AWS style secret key")
	flag.Parse()

	accessKey := rand.Text()

	formattedAccessKey := mergePrefixWithKey("AKIA", accessKey)
	secretKey := randomSecretAccessKey()

	if *accessKeySwitch == true {
		fmt.Printf("%v", formattedAccessKey)
		os.Exit(0)
	}

	if *secretKeySwitch == true {
		fmt.Printf("%v", secretKey)
		os.Exit(0)
	}

	printEverything(formattedAccessKey, secretKey)
}

func printEverything(accessKey string, secretKey string) {
	fmt.Printf("ACCESS KEY ID:     %s\n", accessKey)
	fmt.Printf("SECRET ACCESS KEY: %s\n", secretKey)
}

func mergePrefixWithKey(prefix string, key string) string {
	// can't do this, prefix should be less than the key
	if len(prefix) > len(key) {
		return key
	}

	built := []rune(key)
	for index, rune := range prefix {
		built[index] = rune
	}
	return string(built)
}

func randomSecretAccessKey() string {
	src := make([]byte, 42)
	rand.Read(src)
	return base64.StdEncoding.EncodeToString(src)
}
