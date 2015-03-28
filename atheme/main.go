package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Xe/Tetra/atheme"
	"github.com/howeyc/gopass"
)

var (
	client       *atheme.Atheme
	serverString = flag.String("server", "http://127.0.0.1:8080/xmlrpc", "what http://server:port to connect to")
	cookie       = flag.String("cookie", "", "authentication cookie to use")
	user         = flag.String("user", "", "username to use")
)

func main() {
	flag.Parse()

	command := flag.Arg(0)
	if command == "" {
		flag.Usage()
		os.Exit(1)
	}

	var rest []string

	if n := flag.NArg(); n > 1 {
		rest = flag.Args()[1:]
	}

	_ = rest

	var err error

	client, err = atheme.NewAtheme(*serverString)
	if err != nil {
		log.Fatal(err)
	}

	switch command {
	case "login":
		var username string
		var password string
		scanner := bufio.NewScanner(os.Stdin)

		fmt.Print("Username: ")

		for {
			scanner.Scan()
			username = scanner.Text()

			if scanner.Err() == nil {
				break
			}

			fmt.Print("Username: ")
		}

		fmt.Print("Password: ")
		for {
			password = string(gopass.GetPasswdMasked())
			if password != "" {
				break
			}
			fmt.Print("Password: ")
		}

		err = client.Login(username, password)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Authcookie: %s\n", client.Authcookie)

	case "command":
		if *cookie == "" {
			log.Fatal("specify cookie")
		}

		client.Authcookie = *cookie
		client.Account = *user

		output, err := client.Command(rest...)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(output)
	}
}
