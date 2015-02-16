package main // christine.website/go/tools/dokku

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"code.google.com/p/gcfg"
	"github.com/hypersleep/easyssh"
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

	res, err := ssh.Run(command)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(res)
}
