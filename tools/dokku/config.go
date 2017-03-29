package main

type Config struct {
	Server map[string]*Server
}

type Server struct {
	SSHKey string // if blank default key will be used.
	Host   string // hostname of the dokku server
	User   string // if blank username will be dokku
}
