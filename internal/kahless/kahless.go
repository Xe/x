package kahless

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func getAgent() (agent.Agent, error) {
	agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	return agent.NewClient(agentConn), err
}

const (
	greedoAddr = `kahless.wg.akua:22`
	greedoUser = `xena`
)

// Dial opens a SSH client to greedo.
func Dial() (*ssh.Client, error) {
	agent, err := getAgent()
	if err != nil {
		return nil, err
	}

	client, err := ssh.Dial("tcp", greedoAddr, &ssh.ClientConfig{
		User: greedoUser,
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

// Copy copies a local file reader to the remote destination path. This copies the enitre contents of contents into ram. Don't use this function if doing so is a bad idea. Only works on files less than 2 GB.
func Copy(mode os.FileMode, fileName string, contents io.Reader, destinationPath string) error {
	data, err := ioutil.ReadAll(contents)
	if err != nil {
		return err
	}

	client, err := Dial()
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}

	err = scp.Copy(int64(len(data)), mode, fileName, bytes.NewBuffer(data), destinationPath, session)
	if err != nil {
		return err
	}

	return nil
}

// CopyFile copies a file to Greedo's public files folder and returns its public-facing URL.
func CopyFile(fileName string, contents io.Reader) (string, error) {
	err := Copy(0644, fileName, contents, filepath.Join("public_html", "files", "slugs", fileName))
	if err != nil {
		return "", err
	}

	return WebURL(fileName), nil
}

// WebURL constructs a public-facing URL for a given resource by fragment.
func WebURL(fragment string) string {
	return "https://xena.greedo.xeserv.us/files/slugs/" + fragment
}
