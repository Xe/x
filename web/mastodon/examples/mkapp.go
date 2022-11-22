//go:build ignore

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"within.website/x/web/mastodon"
)

func promptInput(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fmt.Fprint(os.Stderr, ": ")
	reader := bufio.NewReader(os.Stdin)
	// ReadString will block until the delimiter is entered
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// remove the delimeter from the string
	input = strings.TrimSuffix(input, "\n")
	return input, nil
}

func main() {
	instance, err := promptInput("URL of the mastodon server (incl https://)")
	if err != nil {
		log.Fatal(err)
	}

	cli, err := mastodon.Unauthenticated("Xe/x test", "https://within.website/.x.botinfo", instance)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	app, err := cli.CreateApplication(ctx, mastodon.CreateApplicationRequest{
		ClientName:   "Xe/x test",
		RedirectURIs: "urn:ietf:wg:oauth:2.0:oob", // default if not set
		Scopes:       "read write follow push",    // default if not set
		Website:      "https://within.website/.x.botinfo",
	})

	if err != nil {
		log.Fatal(err)
	}

	authURL, err := cli.AuthorizeURL(app, "read write follow push")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(authURL)

	code, err := promptInput("please paste the code")
	if err != nil {
		log.Fatal(err)
	}

	tokenInfo, err := cli.FetchToken(ctx, app, code, "read write follow push")
	if err != nil {
		log.Fatal(err)
	}

	cli, err = mastodon.Authenticated("Xe/x test", "https://within.website/.x.botinfo", instance, tokenInfo.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	if err := cli.VerifyCredentials(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("MASTODON_CLIENT_ID=%s\nMASTODON_CLIENT_SECRET=%s\nMASTODON_INSTANCE=%s\nMASTODON_TOKEN=%s", app.ClientID, app.ClientSecret, instance, tokenInfo.AccessToken)
}
