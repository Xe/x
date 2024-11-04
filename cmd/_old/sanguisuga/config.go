package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/netip"

	"within.website/x/internal/key2hex"
)

type IRC struct {
	Server   string `json:"server"`
	Password string `json:"password"`
	Channel  string `json:"channel"`
	Regex    string `json:"regex"`
	Nick     string `json:"nick"`
	User     string `json:"user"`
	Real     string `json:"real"`
}

type Show struct {
	Title    string `json:"title"`
	DiskPath string `json:"diskPath"`
	Quality  string `json:"quality"`
}

func (s Show) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("title", s.Title),
		slog.String("disk_path", s.DiskPath),
		slog.String("quality", s.Quality),
	)
}

type Transmission struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type Tailscale struct {
	Hostname string  `json:"hostname"`
	Authkey  string  `json:"authkey"`
	DataDir  *string `json:"dataDir,omitempty"`
}

type Telegram struct {
	Token       string `json:"token"`
	MentionUser int64  `json:"mentionUser"`
}

type WireGuard struct {
	PrivateKey string          `json:"privateKey"`
	Address    []netip.Addr    `json:"address"`
	DNS        netip.Addr      `json:"dns"`
	Peers      []WireGuardPeer `json:"peers"`
}

type WireGuardPeer struct {
	PublicKey  string   `json:"publicKey"`
	AllowedIPs []string `json:"allowedIPs"`
	Endpoint   string   `json:"endpoint"`
}

func (w WireGuard) UAPI(out io.Writer) error {
	pkey, err := key2hex.Convert(w.PrivateKey)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "private_key=%s\n", pkey)
	fmt.Fprintln(out, "listen_port=0")
	fmt.Fprintln(out, "replace_peers=true")
	for _, peer := range w.Peers {
		pkey, err := key2hex.Convert(peer.PublicKey)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "public_key=%s\n", pkey)
		fmt.Fprintf(out, "endpoint=%s\n", peer.Endpoint)
		for _, ip := range peer.AllowedIPs {
			fmt.Fprintf(out, "allowed_ip=%s\n", ip)
		}
		fmt.Fprintln(out, "persistent_keepalive_interval=25")
	}
	return nil
}

type Config struct {
	IRC          IRC          `json:"irc"`
	XDCC         IRC          `json:"xdcc"`
	Transmission Transmission `json:"transmission"`
	Shows        []Show       `json:"shows"`
	RSSKey       string       `json:"rssKey"`
	Tailscale    Tailscale    `json:"tailscale"`
	BaseDiskPath string       `json:"baseDiskPath"`
	Telegram     Telegram     `json:"telegram"`
	WireGuard    WireGuard    `json:"wireguard"`
}
