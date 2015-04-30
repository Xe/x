//+build openbsd

package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/mgutz/ansi"
)

func main() {
	color := ansi.ColorCode("white+b:green")

	fmt.Print(color)
	fmt.Printf(
		logo,
		"",
		getPackageCount(),
		getCPUName(),
		getUptime(),
		getUsername(),
		getHostname(),
	)
	fmt.Println(ansi.ColorCode("reset"))
}

func getUsername() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}

	return u.Username
}

func getHostname() string {
	name, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	return name
}
