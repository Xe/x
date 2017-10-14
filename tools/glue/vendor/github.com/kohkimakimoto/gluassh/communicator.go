package gluassh

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"io/ioutil"
	"net"
	"os"
	"github.com/yuin/gopher-lua"
)

//
// Refers to
//
// https://github.com/markpeek/packer/blob/53446bf713ed2866b451aa1d00813a90308ee9e9/communicator/ssh/communicator.go
// https://github.com/rapidloop/rtop
// https://github.com/rapidloop/rtop/blob/ba5b35e964135d50e0babedf0bd69b2fcb5dbcb4/src/sshhelper.go#L185
// https://gist.github.com/zxvdr/8229523
//

// SSH communicator
type Communicator struct {
	Config         *Config
	Agent          agent.Agent
	Auths          []ssh.AuthMethod
	ClientConfig   *ssh.ClientConfig
	UpstreamConfig *Config
	client         *ssh.Client
	clientConns    []interface {
		Close() error
	}

	OriginalConfig *Config
}

func NewComm(config *Config) (*Communicator, error) {
	// Has a proxy?
	originalConfig := config

	var upstreamConfig *Config = nil
	proxy := config.PopProxyConfig()
	if proxy != nil {
		upstreamConfig = config
		config = proxy
	}

	comm := &Communicator{
		Config:         config,
		UpstreamConfig: upstreamConfig,
		clientConns: make([]interface {
			Close() error
		}, 0),
		OriginalConfig: originalConfig,
	}

	// construct auths
	auths := []ssh.AuthMethod{}

	if config.UseAgent {
		// use ssh-agent
		if sock := os.Getenv("SSH_AUTH_SOCK"); len(sock) > 0 {
			agconn, err := net.Dial("unix", sock)
			if err != nil {
				return nil, err
			}
			ag := agent.NewClient(agconn)
			auth := ssh.PublicKeysCallback(ag.Signers)
			auths = append(auths, auth)

			comm.Agent = ag

		} else {
			return nil, errors.New("Could not get a socket from SSH_AUTH_SOCK")
		}
	} else {
		// use key file
		pemBytes, err := ioutil.ReadFile(config.KeyOrDefault())
		if err != nil {
			return nil, err
		}

		block, _ := pem.Decode(pemBytes)
		if block == nil {
			return nil, errors.New("no key found in " + config.KeyOrDefault())
		}

		// handle plain and encrypted keyfile
		if x509.IsEncryptedPEMBlock(block) {
			if config.KeyPassphrase == "" {
				return nil, errors.New("You have to set a key_passphrase for the encryped keyfile '" + config.KeyOrDefault() + "'")
			}

			block.Bytes, err = x509.DecryptPEMBlock(block, []byte(config.KeyPassphrase))
			if err != nil {
				return nil, err
			}
			key, err := parsePemBlock(block)
			if err != nil {
				return nil, err
			}
			signer, err := ssh.NewSignerFromKey(key)
			if err != nil {
				return nil, err
			}
			auths = append(auths, ssh.PublicKeys(signer))

		} else {
			signer, err := ssh.ParsePrivateKey(pemBytes)
			if err != nil {
				return nil, err
			}
			auths = append(auths, ssh.PublicKeys(signer))
		}
	}

	if config.Password != "" {
		// Use password
		auths = append(auths, ssh.Password(config.Password))
	}

	comm.Auths = auths

	// ssh client config
	comm.ClientConfig = &ssh.ClientConfig{
		User: config.UserOrDefault(),
		Auth: auths,
	}

	return comm, nil
}

// Exec runs command on remote server.
// It is a low-layer method to be used Run, Script and others method.
func (c *Communicator) Exec(cmd string, opt *Option, L *lua.LState) (*Result, error) {
	// get a client
	client, err := c.Client()
	if err != nil {
		return nil, err
	}

	// get a session
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	// RequestAgentForwarding
	if c.Config.ForwardAgent && c.Agent != nil {
		agent.ForwardToAgent(client, c.Agent)
		if err = agent.RequestAgentForwarding(sess); err != nil {
			return nil, err
		}
	}

	// Request a PTY
	if c.Config.Pty {
		// refers to https://github.com/markpeek/packer/blob/53446bf713ed2866b451aa1d00813a90308ee9e9/communicator/ssh/communicator.go
		termModes := ssh.TerminalModes{
			ssh.ECHO:          0,     // do not echo
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}
		err = sess.RequestPty("xterm", 80, 40, termModes)
		if err != nil {
			return nil, err
		}
	}

	// Compose sudo command line.
	if opt.Sudo {
		if opt.User != "" {
			cmd = "sudo -Sku " + opt.User + " /bin/sh -l -c '" + cmd + "'"
		} else {
			cmd = "sudo -Sk /bin/sh -l -c '" + cmd + "'"
		}
	}

	// apply io
	var outWriter io.Writer
	var errWriter io.Writer
	var inReader io.Reader

	// buffer
	var outBuffer = new(bytes.Buffer)
	var errBuffer = new(bytes.Buffer)
	var inBuffer = new(bytes.Buffer)


	// append stdio to buffer
	if opt.UseStdout {
		if opt.OutputFunc != nil {
			outWriter = io.MultiWriter(os.Stdout, outBuffer, NewLFuncWriter(1, opt.OutputFunc, L))
		} else {
			outWriter = io.MultiWriter(os.Stdout, outBuffer)
		}
	} else {
		outWriter = io.MultiWriter(outBuffer)
	}

	if opt.UseStderr {
		if opt.OutputFunc != nil {
			errWriter = io.MultiWriter(os.Stderr, errBuffer, NewLFuncWriter(2, opt.OutputFunc, L))
		} else {
			errWriter = io.MultiWriter(os.Stderr, errBuffer)
		}
	} else {
		errWriter = io.MultiWriter(errBuffer)
	}

	inReader = io.MultiReader(inBuffer, os.Stdin)

	sess.Stdin = inReader
	sess.Stdout = outWriter
	sess.Stderr = errWriter

	// write sudo password
	if opt.Password != "" {
		// If it try to use sudo password, write password to input buffer
		fmt.Fprintln(inBuffer, opt.Password)
	}

	err = sess.Run(cmd)
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return NewResult(outBuffer, errBuffer, exitErr.ExitStatus()), err
		} else {
			return NewResult(outBuffer, errBuffer, 1), err
		}
	}

	return NewResult(outBuffer, errBuffer, 0), nil
}

func (c *Communicator) Run(cmd string, opt *Option, L *lua.LState) (*Result, error) {
	return c.Exec(cmd, opt, L)
}

func (c *Communicator) Get(remote string, local string) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	sc, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sc.Close()

	fi, err := sc.Open(remote)
	if err != nil {
		return err
	}
	defer fi.Close()

	fo, err := os.Create(local)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = io.Copy(fo, fi)
	if err != nil {
		return err
	}

	return nil
}

func (c *Communicator) Put(local string, remote string) error {
	client, err := c.Client()
	if err != nil {
		return err
	}

	sc, err := sftp.NewClient(client)
	if err != nil {
		return err
	}
	defer sc.Close()

	b, err := ioutil.ReadFile(local)
	if err != nil {
		return err
	}

	fo, err := sc.Create(remote)
	if err != nil {
		return err
	}
	defer fo.Close()

	_, err = fo.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (c *Communicator) Client() (*ssh.Client, error) {
	if c.client != nil {
		return c.client, nil
	}

	// create ssh client.
	client, err := ssh.Dial("tcp", c.Config.HostOrDefault()+":"+c.Config.PortOrDefault(), c.ClientConfig)
	if err != nil {
		return nil, err
	}
	c.clientConns = append(c.clientConns, client)

	// If it has a upstream server?
	for c.UpstreamConfig != nil {
		// It is next server config to connect.
		var config *Config = nil

		// Does the upstream server need proxy to connet?
		proxy := c.UpstreamConfig.PopProxyConfig()
		if proxy != nil {
			config = proxy
		} else {
			config = c.UpstreamConfig
			c.UpstreamConfig = nil
		}

		// dial to ssh proxy
		connection, err := client.Dial("tcp", config.HostOrDefault()+":"+config.PortOrDefault())
		if err != nil {
			return nil, err
		}
		c.clientConns = append(c.clientConns, connection)

		conn, chans, reqs, err := ssh.NewClientConn(connection, config.HostOrDefault()+":"+config.PortOrDefault(), c.ClientConfig)
		if err != nil {
			return nil, err
		}
		client = ssh.NewClient(conn, chans, reqs)
		c.clientConns = append(c.clientConns, client)

		if err != nil {
			return nil, err
		}
	}

	c.client = client
	return c.client, nil
}

func (c *Communicator) Close() error {
	for _, conn := range c.clientConns {
		err := conn.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ref golang.org/x/crypto/ssh/keys.go#ParseRawPrivateKey.
func parsePemBlock(block *pem.Block) (interface{}, error) {
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "DSA PRIVATE KEY":
		return ssh.ParseDSAPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("rtop: unsupported key type %q", block.Type)
	}
}
