package minipaas

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func getAgent() (agent.Agent, error) {
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	return agent.NewClient(agentConn), err
}

const (
	minipaasAddr = `minipaas.xeserv.us:22`
	minipaasUser = `dokku`
)

// Dial opens a SSH client to minipaas as the dokku user.
func Dial() (*ssh.Client, error) {
	agent, err := getAgent()
	if err != nil {
		return nil, err
	}

	client, err := ssh.Dial("tcp", minipaasAddr, &ssh.ClientConfig{
		User: minipaasUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agent.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
