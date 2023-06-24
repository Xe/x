package localca

import (
	"context"
	"crypto/tls"
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"golang.org/x/crypto/acme/autocert"
)

func TestLocalCA(t *testing.T) {
	dir, err := ioutil.TempDir("", "localca-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	cache := autocert.DirCache(dir)

	keyFile := path.Join(dir, "key.pem")
	certFile := path.Join(dir, "cert.pem")
	const suffix = "club"

	m, err := New(keyFile, certFile, suffix, cache)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("local", func(t *testing.T) {
		_, err = m.GetCertificate(&tls.ClientHelloInfo{
			ServerName: "foo.local.cetacean.club",
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("network", func(t *testing.T) {
		t.Skip("no")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		tc := &tls.Config{
			GetCertificate: m.GetCertificate,
		}

		ch := make(chan struct{})

		go func() {
			lis, err := tls.Listen("tcp", ":9293", tc)
			if err != nil {
				t.Error(err)
				return
			}
			defer lis.Close()
			ch <- struct{}{}

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				cli, err := lis.Accept()
				if err != nil {
					t.Error(err)
					return
				}
				defer cli.Close()

				go io.Copy(cli, cli)
			}
		}()

		<-ch
		cli, err := tls.Dial("tcp", "localhost:9293", &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		defer cli.Close()

		cli.Write([]byte("butts"))
	})
}
