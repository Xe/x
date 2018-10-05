package main // christine.website/go/tools/dokku

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hypersleep/easyssh"
	gcfg "gopkg.in/gcfg.v1"
)

var (
	cfgPath    = flag.String("cfg", "", "configuration path, default is ~/.dokku.cfg")
	serverName = flag.String("server", "default", "server to use out of dokku config")
)

func main() {
	flag.Parse()

	if *cfgPath == "" {
		*cfgPath = os.Getenv("HOME") + "/.dokku.cfg"
	}

	var cfg Config
	err := gcfg.ReadFileInto(&cfg, *cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	var server *Server
	var ok bool

	if server, ok = cfg.Server[*serverName]; !ok {
		log.Fatalf("server %s not defined in configuration file %s", *serverName, *cfgPath)
	}

	if server.User == "" {
		server.User = "dokku"
	}

	if server.SSHKey == "" {
		server.SSHKey = "/.ssh/id_rsa"
	}

	ssh := &easyssh.MakeConfig{
		User:   server.User,
		Server: server.Host,
		Key:    server.SSHKey,
	}

	command := strings.Join(flag.Args(), " ")

	stdout, stderr, _, err := ssh.Run(command, 360)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(stdout)
	fmt.Println()
	fmt.Print(stderr)
	fmt.Println()
}
