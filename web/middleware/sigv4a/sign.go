package sigv4a

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"within.website/x/web/middleware/internal/awssig"
)

// Signer signs outgoing HTTP requests with AWS Signature Version 4A. It is
// the client-side counterpart to Verifier and shares its canonicalization
// through awssig, so the two agree by construction.
//
// The ECDSA keypair is derived once at construction; the secret access key
// is not retained. Application code usually wants the sigv4aclient
// subpackage, which wraps a Signer in an http.RoundTripper.
type Signer struct {
	accessKeyID string
	priv        *ecdsa.PrivateKey
	region      string
	service     string

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time
}

// NewSigner derives the SigV4A keypair for the credential and returns a
// Signer that signs for the given region (the X-Amz-Region-Set value; a
// single region name for within.website services) and service.
func NewSigner(accessKeyID, secretAccessKey, region, service string) (*Signer, error) {
	priv, err := DeriveKeyPair(accessKeyID, secretAccessKey)
	if err != nil {
		return nil, err
	}
	return &Signer{accessKeyID: accessKeyID, priv: priv, region: region, service: service}, nil
}

// Sign signs r in place: it hashes body (which must be the full request
// payload; nil means empty), declares it in X-Amz-Content-Sha256, and signs
// the host, x-amz-content-sha256, x-amz-date, and x-amz-region-set headers.
// It does not touch r.Body.
func (s *Signer) Sign(r *http.Request, body []byte) error {
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	r.Header.Set("X-Amz-Content-Sha256", payloadHash)
	return s.sign(r, []string{"host", "x-amz-content-sha256", "x-amz-date", "x-amz-region-set"}, payloadHash)
}

// sign stamps X-Amz-Date and X-Amz-Region-Set, builds the canonical request
// over exactly signedHeaders, and writes the Authorization header. It is
// split from Sign so tests can reproduce the AWS test-suite vectors, which
// sign without x-amz-content-sha256.
func (s *Signer) sign(r *http.Request, signedHeaders []string, payloadHash string) error {
	if r.Host == "" {
		r.Host = r.URL.Host
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	amzDate := now().UTC().Format(amzTimeFormat)
	r.Header.Set("X-Amz-Date", amzDate)
	r.Header.Set("X-Amz-Region-Set", s.region)

	headers := append([]string(nil), signedHeaders...)
	sort.Strings(headers)

	canonReq := awssig.BuildCanonicalRequest(r, headers, payloadHash, false)
	// SigV4A credential scope: date/service/aws4_request — no region. The
	// region set is bound to the signature as a signed header instead.
	scope := strings.Join([]string{amzDate[:len(shortDateFormat)], s.service, terminator}, "/")
	hashed := sha256.Sum256([]byte(canonReq))
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		scope,
		hex.EncodeToString(hashed[:]),
	}, "\n")

	digest := sha256.Sum256([]byte(stringToSign))
	sig, err := ecdsa.SignASN1(rand.Reader, s.priv, digest[:])
	if err != nil {
		return fmt.Errorf("sigv4a: signing: %w", err)
	}

	r.Header.Set("Authorization", fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, s.accessKeyID, scope, strings.Join(headers, ";"), hex.EncodeToString(sig)))
	return nil
}
