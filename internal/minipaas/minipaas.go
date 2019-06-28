package minipaas

import (
	"flag"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func getAgent() (agent.Agent, error) {
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	return agent.NewClient(agentConn), err
}

var (
	minipaasAddr = flag.String("minipaas-addr", "minipaas.xeserv.us:22", "address of minipaas")
	minipaasUser = flag.String("minipaas-user", "dokku", "username on minipaas")
)

// Dial opens a SSH client to minipaas as the dokku user.
func Dial() (*ssh.Client, error) {
	agent, err := getAgent()
	if err != nil {
		return nil, err
	}

	client, err := ssh.Dial("tcp", *minipaasAddr, &ssh.ClientConfig{
		User: *minipaasUser,
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

// Exec runs an arbitrary dokku command with OS standard input, output and error.
func Exec(args string) error {
	mp, err := Dial()
	if err != nil {
		return err
	}
	defer mp.Close()

	sess, err := mp.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	sess.Stdin = os.Stdin
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	err = sess.Run(args)
	if err != nil {
		return err
	}

	return nil
}
