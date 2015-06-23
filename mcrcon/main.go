package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/bearbin/mcgorcon"
)

var (
	host     = flag.String("host", "127.0.0.1", "server hostname")
	port     = flag.Int("port", 25575, "rcon port")
	password = flag.String("pass", "swag", "rcon password")
)

func main() {
	flag.Parse()

	client, err := mcgorcon.Dial(*host, *port, *password)
	if err != nil {
		log.Fatal(err)
	}

	data, err := client.SendCommand(strings.Join(flag.Args(), " "))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(data)
}
