package config

import (
	"context"
	"errors"
)

type Autocert struct {
	AcceptTOS bool   `hcl:"accept_tos"`
	Email     string `hcl:"email"`

	Directory string   `hcl:"directory,optional"`
	S3        *S3Cache `hcl:"s3,block"`
}

type S3Cache struct {
	Bucket string `hcl:"bucket"`
	Prefix string `hcl:"prefix"`
}

type Toplevel struct {
	Autocert *Autocert `hcl:"autocert,block"`
	Bind     Bind      `hcl:"bind,block"`
	Domains  []Domain  `hcl:"domain,block"`
}

func (t Toplevel) HostPolicy(_ context.Context, host string) error {
	err := errors.New("host not found")

	for _, d := range t.Domains {
		if host == d.Name && d.TLS.Autocert {
			return nil
		}
	}

	return err
}

type Bind struct {
	HTTP    string `hcl:"http"`
	HTTPS   string `hcl:"https"`
	Metrics string `hcl:"metrics"`
}

type Domain struct {
	Name   string  `hcl:"name,label"`
	TLS    TLS     `hcl:"tls,block"`
	Routes []Route `hcl:"route,block"`
}

type TLS struct {
	Autocert bool   `hcl:"autocert,optional"`
	Cert     string `hcl:"cert,optional"`
	Key      string `hcl:"key,optional"`
}

type Route struct {
	Path string `hcl:"path,label"`

	Folder       string        `hcl:"folder,optional"`
	ReverseProxy *ReverseProxy `hcl:"reverse_proxy,block"`
}

type ReverseProxy struct {
	Target       string `hcl:"target"`
	HealthTarget string `hcl:"health_target"`
}
