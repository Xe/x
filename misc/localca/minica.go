package localca

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

type issuer struct {
	key  crypto.Signer
	cert *x509.Certificate
}

func getIssuer(keyFile, certFile string, autoCreate bool) (*issuer, error) {
	keyContents, keyErr := ioutil.ReadFile(keyFile)
	certContents, certErr := ioutil.ReadFile(certFile)
	if os.IsNotExist(keyErr) && os.IsNotExist(certErr) {
		err := makeIssuer(keyFile, certFile)
		if err != nil {
			return nil, err
		}
		return getIssuer(keyFile, certFile, false)
	} else if keyErr != nil {
		return nil, fmt.Errorf("%s (but %s exists)", keyErr, certFile)
	} else if certErr != nil {
		return nil, fmt.Errorf("%s (but %s exists)", certErr, keyFile)
	}
	key, err := readPrivateKey(keyContents)
	if err != nil {
		return nil, fmt.Errorf("reading private key from %s: %s", keyFile, err)
	}

	cert, err := readCert(certContents)
	if err != nil {
		return nil, fmt.Errorf("reading CA certificate from %s: %s", certFile, err)
	}

	equal, err := publicKeysEqual(key.Public(), cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("comparing public keys: %s", err)
	} else if !equal {
		return nil, fmt.Errorf("public key in CA certificate %s doesn't match private key in %s",
			certFile, keyFile)
	}
	return &issuer{key, cert}, nil
}

func readPrivateKey(keyContents []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyContents)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	} else if block.Type != "RSA PRIVATE KEY" && block.Type != "ECDSA PRIVATE KEY" {
		return nil, fmt.Errorf("incorrect PEM type %s", block.Type)
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func readCert(certContents []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certContents)
	if block == nil {
		return nil, fmt.Errorf("no PEM found")
	} else if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("incorrect PEM type %s", block.Type)
	}
	return x509.ParseCertificate(block.Bytes)
}

func makeIssuer(keyFile, certFile string) error {
	keyData, key, err := makeKey()
	if err != nil {
		return err
	}
	ioutil.WriteFile(keyFile, keyData, 0600)
	certData, _, err := makeRootCert(key, certFile)
	if err != nil {
		return err
	}
	ioutil.WriteFile(certFile, certData, 0600)
	return nil
}

func makeKey() ([]byte, *rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	if err != nil {
		return nil, nil, err
	}
	buf := bytes.NewBuffer([]byte{})
	err = pem.Encode(buf, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), key, nil
}

func makeRootCert(key crypto.Signer, filename string) ([]byte, *x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "localca root ca " + hex.EncodeToString(serial.Bytes()[:3]),
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(100, 0, 0),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	if err != nil {
		return nil, nil, err
	}
	buf := bytes.NewBuffer([]byte{})
	err = pem.Encode(buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}
	result, err := x509.ParseCertificate(der)
	return buf.Bytes(), result, err
}

func parseIPs(ipAddresses []string) ([]net.IP, error) {
	var parsed []net.IP
	for _, s := range ipAddresses {
		p := net.ParseIP(s)
		if p == nil {
			return nil, fmt.Errorf("invalid IP address %s", s)
		}
		parsed = append(parsed, p)
	}
	return parsed, nil
}

func publicKeysEqual(a, b interface{}) (bool, error) {
	aBytes, err := x509.MarshalPKIXPublicKey(a)
	if err != nil {
		return false, err
	}
	bBytes, err := x509.MarshalPKIXPublicKey(b)
	if err != nil {
		return false, err
	}
	return bytes.Compare(aBytes, bBytes) == 0, nil
}

func (iss *issuer) sign(domains []string, ipAddresses []string) ([]byte, *x509.Certificate, error) {
	var cn string
	if len(domains) > 0 {
		cn = domains[0]
	} else if len(ipAddresses) > 0 {
		cn = ipAddresses[0]
	} else {
		return nil, nil, fmt.Errorf("must specify at least one domain name or IP address")
	}
	keyData, key, err := makeKey()
	if err != nil {
		return nil, nil, err
	}
	buf := bytes.NewBuffer([]byte{})
	buf.Write(keyData)

	parsedIPs, err := parseIPs(ipAddresses)
	if err != nil {
		return nil, nil, err
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		DNSNames:    domains,
		IPAddresses: parsedIPs,
		Subject: pkix.Name{
			CommonName: cn,
		},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(90, 0, 0),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, iss.cert, key.Public(), iss.key)
	if err != nil {
		return nil, nil, err
	}
	err = pem.Encode(buf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: der,
	})
	if err != nil {
		return nil, nil, err
	}
	result, err := x509.ParseCertificate(der)
	return buf.Bytes(), result, err
}

func split(s string) (results []string) {
	if len(s) > 0 {
		return strings.Split(s, ",")
	}
	return nil
}
