package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

var r *rand.Rand

func main() {
	accessKeySwitch := flag.Bool("a", false, "Gen a fake AWS style access key")
	secretKeySwitch := flag.Bool("s", false, "Gen a fake AWS style secret key")
	flag.Parse()

	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	accessKey := randomAccessKey(20)

	formattedAccessKey := mergePrefixWithKey("AKIA", accessKey)
	secretKey := computeHmac256(string(time.Now().String()), accessKey)

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
	fmt.Printf("ACCESS KEY: %s\n", accessKey)
	fmt.Printf("SECRET KEY: %s\n", secretKey)
}

func randomAccessKey(length int) string {
	const chars string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	buffer := make([]byte, length)
	for i := range buffer {
		buffer[i] = chars[r.Intn(len(chars))]
	}
	return string(buffer)
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

func computeHmac256(message string, secret string) string {
	key := []byte(secret)
	h := hmac.New(sha256.New, key)
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}