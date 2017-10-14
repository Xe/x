package gluassh

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
)

// SSH connection config.
type Config struct {
	Host          string  `json:"host"`
	Port          string  `json:"port"`
	User          string  `json:"user"`
	Key           string  `json:"key"`
	KeyPassphrase string  `json:"key_passphrase"`
	UseAgent      bool    `json:"use_agent"`
	ForwardAgent  bool    `json:"forward_agent"`
	Pty           bool    `json:"pty"`
	Password      string  `json:"password"`
	Proxy         *Config `json:"proxy"`
}

// NewConfig creates new config instance.
func NewConfig() *Config {
	c := &Config{}

	return c
}

// Pop the last proxy server
func (c *Config) PopProxyConfig() *Config {
	proxy := c.Proxy
	if proxy == nil {
		return nil
	}

	if proxy.Proxy != nil {
		return proxy.PopProxyConfig()
	}

	c.Proxy = nil

	return proxy
}

func (c *Config) UpdateWithJSON(data []byte) error {
	err := json.Unmarshal(data, c)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) ConfigsChain() []*Config {
	var configs []*Config

	configs = append(configs, c)

	if c.Proxy != nil {
		proxyConfigs := c.Proxy.ConfigsChain()
		configs = append(proxyConfigs, configs...)
	}

	return configs
}

func (c *Config) HostOrDefault() string {
	if c.Host == "" {
		return "localhost"
	} else {
		return c.Host
	}
}

func (c *Config) PortOrDefault() string {
	if c.Port == "" {
		return "22"
	} else {
		return c.Port
	}
}

func (c *Config) KeyOrDefault() string {
	if c.Key == "" {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		return filepath.Join(home, ".ssh/id_rsa")
	}

	return c.Key
}

func (c *Config) UserOrDefault() string {
	if c.User == "" {
		u, err := user.Current()
		if err == nil {
			return u.Username
		}
	}

	return c.User
}
