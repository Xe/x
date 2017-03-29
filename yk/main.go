package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/GeertJohan/yubigo"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kr/pretty"
)

func main() {
	yubiAuth, err := yubigo.NewYubiAuth(os.Getenv("YUBIKEY_CLIENT_ID"), os.Getenv("YUBIKEY_SECRET_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	_ = yubiAuth

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("yk> ")
	text, err := reader.ReadString(byte('\n'))
	if err != nil {
		log.Fatal(err)
	}

	text = strings.TrimSpace(text)

	resp, _, err := yubiAuth.Verify(text)
	if err != nil {
		log.Fatal(err)
	}

	pretty.Println(resp)

	if !resp.IsValidOTP() {
		log.Fatal("invalid OTP")
	}

	prefix, _, err := yubigo.ParseOTP(text)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("uid: %s", prefix)
}
