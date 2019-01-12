package localca

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"within.website/ln"
	"within.website/ln/opname"
)

var (
	ErrBadData                = errors.New("localca: certificate data is bad")
	ErrDomainDoesntHaveSuffix = errors.New("localca: domain doesn't have the given suffix")
)

// Manager automatically provisions and caches TLS certificates in a given
// autocert Cache. If it cannot fetch a certificate on demand, the certificate
// is dynamically generated with a lifetime of 100 years, which should be good
// enough.
type Manager struct {
	Cache        autocert.Cache
	DomainSuffix string

	*issuer
}

// New creates a new Manager with the given key filename, certificate filename,
// allowed domain suffix and autocert cache. All given certificates will be
// created if they don't already exist.
func New(keyFile, certFile, suffix string, cache autocert.Cache) (Manager, error) {
	iss, err := getIssuer(keyFile, certFile, true)

	if err != nil {
		return Manager{}, err
	}

	result := Manager{
		DomainSuffix: suffix,
		Cache:        cache,
		issuer:       iss,
	}

	return result, nil
}

func (m Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	name := hello.ServerName
	if !strings.Contains(strings.Trim(name, "."), ".") {
		return nil, errors.New("localca: server name component count invalid")
	}

	if !strings.HasSuffix(name, m.DomainSuffix) {
		return nil, ErrDomainDoesntHaveSuffix
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = opname.With(ctx, "localca.Manager.GetCertificate")
	ctx = ln.WithF(ctx, ln.F{"server_name": name})

	data, err := m.Cache.Get(ctx, name)
	if err != nil && err != autocert.ErrCacheMiss {
		return nil, err
	}

	if err == autocert.ErrCacheMiss {
		data, _, err = m.issuer.sign([]string{name}, nil)
		if err != nil {
			return nil, err
		}
		err = m.Cache.Put(ctx, name, data)
		if err != nil {
			return nil, err
		}
	}

	cert, err := loadCertificate(name, data)
	if err != nil {
		return nil, err
	}

	ln.Log(ctx, ln.Info("returned cert successfully"))

	return cert, nil
}

func loadCertificate(name string, data []byte) (*tls.Certificate, error) {
	priv, pub := pem.Decode(data)
	if priv == nil || !strings.Contains(priv.Type, "PRIVATE") {
		return nil, ErrBadData
	}
	privKey, err := parsePrivateKey(priv.Bytes)
	if err != nil {
		return nil, err
	}

	// public
	var pubDER [][]byte
	for len(pub) > 0 {
		var b *pem.Block
		b, pub = pem.Decode(pub)
		if b == nil {
			break
		}
		pubDER = append(pubDER, b.Bytes)
	}
	if len(pub) > 0 {
		return nil, ErrBadData
	}

	// verify and create TLS cert
	leaf, err := validCert(name, pubDER, privKey, time.Now())
	if err != nil {
		return nil, err
	}
	tlscert := &tls.Certificate{
		Certificate: pubDER,
		PrivateKey:  privKey,
		Leaf:        leaf,
	}
	return tlscert, nil
}
