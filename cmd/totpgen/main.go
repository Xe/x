package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/creachadair/otp"
	"within.website/x/internal"
)

var (
	totpKey = flag.String("totp-key", "", "TOTP secret key (base32 encoded)")
)

func main() {
	internal.HandleStartup()

	if *totpKey == "" {
		fmt.Fprintln(os.Stderr, "error: --totp-key is required")
		os.Exit(1)
	}

	code, err := otp.DefaultTOTP(*totpKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating TOTP: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(code)
}
