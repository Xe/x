package main

import (
	"context"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Xe/ln"
	"github.com/digitalocean/godo"
	_ "github.com/joho/godotenv/autoload"
	"golang.org/x/oauth2"
	"gopkg.in/irc.v1"
)

// TokenSource is needed for oauth2 munging.
type TokenSource struct {
	AccessToken string
}

// Token returns the access token as an oauth2 token.
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func main() {
	tokenSource := &TokenSource{
		AccessToken: os.Getenv("DO_TOKEN"),
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	client := godo.NewClient(oauthClient)

	t := time.NewTicker(20 * time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-t.C:
			opt := &godo.ListOptions{
				Page:    1,
				PerPage: 200,
			}

			ln.Log(ln.F{"action": "listing droplets"})
			droplets, _, err := client.Droplets.List(ctx, opt)
			if err != nil {
				ln.Error(err, ln.F{"action": "listing droplets"})
				continue
			}

			toRestart := []int{}

			for _, d := range droplets {
				for _, t := range d.Tags {
					if t == "ircd" {
						f := ln.F{
							"droplet_id":   d.ID,
							"droplet_name": d.Name,
						}

						ln.Log(f, ln.F{"action": "investigating droplet"})

						ipv4, err := d.PublicIPv4()
						if err != nil {
							ln.Error(err, f, ln.F{"action": "getting droplet ipv4"})
							continue
						}

						f["host"] = ipv4

						dur := 30 * time.Second
						ln.Log(f, ln.F{"action": "dialing", "timeout": dur})

						conn, err := net.DialTimeout("tcp", ipv4+":6667", 30*time.Second)
						if err != nil {
							ln.Error(err, f, ln.F{"action": "dialing plain tcp"})
							toRestart = append(toRestart, d.ID)
							continue
						}

						config := irc.ClientConfig{
							Nick: "irc_tester_" + randStringRunes(5),
							Pass: "password",
							User: "irctestr",
							Name: "IRC tester bot" + randStringRunes(10),
							Handler: irc.HandlerFunc(func(c *irc.Client, m *irc.Message) {
								if m.Command == "001" {
									conn.Close()
								}
							}),
						}

						client := irc.NewClient(conn, config)
						err = client.Run()
						if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
							ln.Error(err, f, ln.F{"action": "testing RFC 1459 protocol"})
							toRestart = append(toRestart, d.ID)
							continue
						}
					}
				}
			}

			for _, did := range toRestart {
				ln.Log(ln.F{"intent": "restarting droplet", "droplet_id": did})
				_, _, err := client.DropletActions.Reboot(ctx, did)
				if err != nil {
					ln.Error(err, ln.F{"action": "restarting droplet", "droplet_id": did})
				}
			}
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
