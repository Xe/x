package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"within.website/x/internal"
)

func main() {
	internal.HandleStartup()

	if flag.NArg() != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run . <encrypt> <plaintext> <key>")
		fmt.Fprintln(os.Stderr, "usage: go run . <decrypt> <ciphertext> <key>")
		os.Exit(2)
	}

	action := flag.Arg(0)
	plaintext := flag.Arg(1)
	key := flag.Arg(2)

	switch action {
	case "encrypt":
		fmt.Println(Encrypt(key, plaintext))
	case "decrypt":
		fmt.Println(Decrypt(key, plaintext))
	default:
		fmt.Fprintln(os.Stderr, "usage: go run . <encrypt> <plaintext> <key>")
		fmt.Fprintln(os.Stderr, "usage: go run . <decrypt> <ciphertext> <key>")
		os.Exit(2)
	}
}

func ReplicateKey(key string, plaintextLen int) string {
	return strings.Repeat(key, plaintextLen/len(key)+1)[:plaintextLen]
}

func Encrypt(keyRaw, plaintext string) string {
	keyRaw = ReplicateKey(keyRaw, len(plaintext))
	plaintextDecoded := make([]int, len(plaintext))
	key := make([]int, len(plaintext))
	encoded := make([]byte, len(plaintext))

	for i := range plaintext {
		plaintextDecoded[i] = int(plaintext[i] - 'A')
		key[i] = int(keyRaw[i] - 'A')
		encoded[i] = byte(((plaintextDecoded[i] + key[i]) % 26) + 'A')
	}

	return string(encoded)
}

func Decrypt(keyRaw, ciphertext string) string {
	keyRaw = ReplicateKey(keyRaw, len(ciphertext))
	ciphertextDecoded := make([]int, len(ciphertext))
	key := make([]int, len(ciphertext))
	encoded := make([]byte, len(ciphertext))

	for i := range ciphertext {
		ciphertextDecoded[i] = int(ciphertext[i] - 'A')
		key[i] = int(keyRaw[i] - 'A')
		encoded[i] = byte((((ciphertextDecoded[i] - key[i]) + 26) % 26) + 'A')
	}

	return string(encoded)
}
