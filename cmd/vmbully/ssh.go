package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// isUsableIPv4 reports if addr is an IPv4 address that a remote client can reach.
func isUsableIPv4(addr string) bool {
	ip := net.ParseIP(addr)
	return ip != nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

// sshDial logs in to host with a password. The connection closes when ctx ends.
func sshDial(ctx context.Context, host, user, password string) (*ssh.Client, error) {
	addr := net.JoinHostPort(host, "22")

	d := net.Dialer{Timeout: 10 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	sc, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(password)},
		// The VM is seconds old, so there is no known host key to check. Lab only.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	})
	if err != nil {
		conn.Close()
		return nil, err
	}

	client := ssh.NewClient(sc, chans, reqs)
	context.AfterFunc(ctx, func() { client.Close() })

	return client, nil
}

// sshRun runs cmd on the remote host and returns its stdout. The stderr of cmd
// goes to the local stderr. A nonzero exit status is an error.
func sshRun(client *ssh.Client, cmd string) (string, error) {
	sess, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = io.MultiWriter(os.Stderr, &stderr)

	if err := sess.Run(cmd); err != nil {
		var exitErr *ssh.ExitError
		if errors.As(err, &exitErr) {
			return stdout.String(), fmt.Errorf("command %q: exit status %d: %s", cmd, exitErr.ExitStatus(), strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), fmt.Errorf("command %q: %w", cmd, err)
	}

	return stdout.String(), nil
}

// shellQuote quotes s as one word for a POSIX shell.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sshCopy copies the local file source to dest on the remote host with the
// scp protocol. dest is a file path or a directory.
func sshCopy(client *ssh.Client, source, dest string) error {
	fin, err := os.Open(source)
	if err != nil {
		return err
	}
	defer fin.Close()

	st, err := fin.Stat()
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", source)
	}

	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	sess.Stderr = &stderr

	if err := sess.Start("scp -t -- " + shellQuote(dest)); err != nil {
		return err
	}

	// The remote scp sends one status reply at the start and one after each
	// message: 0 for OK, 1 or 2 for an error with a text line.
	replies := bufio.NewReader(stdout)
	ack := func() error {
		code, err := replies.ReadByte()
		if err != nil {
			return fmt.Errorf("read scp reply: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		if code == 0 {
			return nil
		}
		msg, _ := replies.ReadString('\n')
		return fmt.Errorf("scp: %s", strings.TrimSpace(msg))
	}

	if err := ack(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdin, "C%04o %d %s\n", st.Mode().Perm(), st.Size(), filepath.Base(source)); err != nil {
		return err
	}
	if err := ack(); err != nil {
		return err
	}

	if _, err := io.CopyN(stdin, fin, st.Size()); err != nil {
		return err
	}
	if _, err := stdin.Write([]byte{0}); err != nil {
		return err
	}
	if err := ack(); err != nil {
		return err
	}

	if err := stdin.Close(); err != nil {
		return err
	}

	return sess.Wait()
}
