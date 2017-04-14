package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env"
	_ "github.com/joho/godotenv/autoload"
	yaml "gopkg.in/yaml.v1"
)

// Config is the configuration for kcpd
type Config struct {
	Mode string `env:"KCPD_MODE,required" envDefault:"server" yaml:"mode"`

	// Client mode config

	// What IP the client should connect to
	ClientServerAddress string `env:"KCPD_SERVER_ADDRESS" yaml:"server"`
	// Administrator's NickServ username
	ClientUsername string `env:"KCPD_ADMIN_USERNAME" yaml:"admin_username"`
	// Administrator's NickServ password
	ClientPassword string `env:"KCPD_ADMIN_PASSWORD" yaml:"admin_password"`
	// Local bindaddr
	ClientBindaddr string `env:"KCPD_CLIENT_BINDADDR" yaml:"client_bind"`

	// Server mode config

	// What UDP port/address should kcpd bind on?
	ServerBindAddr string `env:"KCPD_BIND" yaml:"bind"`
	// Atheme URL for Nickserv authentication of the administrator for setting up KCP sessions
	ServerAthemeURL string `env:"KCPD_ATHEME_URL" yaml:"atheme_url"`
	// URL endpoint for allowing/denying users
	ServerAllowListEndpoint string `env:"KCPD_ALLOWLIST_ENDPOINT" yaml:"allow_list_endpoint"`
	// local ircd (unsecure) endpoint
	ServerLocalIRCd string `env:"KCPD_LOCAL_IRCD" yaml:"local_ircd"`
	// WEBIRC password to use for local sockets
	ServerWEBIRCPassword string `env:"KCPD_WEBIRC_PASSWORD" yaml:"webirc_password"`
	// ServerTLSCert is the TLS cert file
	ServerTLSCert string `env:"KCPD_TLS_CERT" yaml:"tls_cert"`
	// ServerTLSKey is the TLS key file
	ServerTLSKey string `env:"KCPD_TLS_KEY" yaml:"tls_key"`
}

var (
	configFname = flag.String("config", "", "configuration file to use (if unset config will be pulled from the environment)")
)

func main() {
	flag.Parse()

	cfg := &Config{}

	if *configFname != "" {
		fin, err := os.Open(*configFname)
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()

		data, err := ioutil.ReadAll(fin)
		if err != nil {
			log.Fatal(err)
		}

		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		err := env.Parse(cfg)
		if err != nil {
			log.Fatal(err)
		}
	}

	switch cfg.Mode {
	case "client":
		c, err := NewClient(cfg)
		if err != nil {
			log.Fatal(err)
		}

		for {
			err = c.Dial()
			if err != nil {
				log.Println(err)
			}

			time.Sleep(time.Second)
		}

	case "server":
		s, err := NewServer(cfg)
		if err != nil {
			log.Fatal(err)
		}

		err = s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatal(ErrBadConfig)
	}
}
