package iptoasn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"

	"within.website/x/web"
	"within.website/x/web/useragent"
)

var (
	ErrIPNotValid      = errors.New("ip not valid")
	ErrNoCountryCode   = errors.New("country_code not set")
	ErrNoDescription   = errors.New("description not set")
	ErrNoASNumber      = errors.New("as_number not set")
	ErrFirstIPNotSet   = errors.New("first_ip not set")
	ErrFirstIPNotValid = errors.New("first_ip not valid")
	ErrLastIPNotSet    = errors.New("last_ip not set")
	ErrLastIPNotValid  = errors.New("last_ip not valid")
)

type ASNInfo struct {
	IP          netip.Addr  `json:"ip"`
	Announced   bool        `json:"announced"`
	CountryCode *string     `json:"as_country_code,omitempty"`
	Description *string     `json:"as_description,omitempty"`
	ASNumber    *int        `json:"as_number,omitempty"`
	FirstIP     *netip.Addr `json:"first_ip,omitempty"`
	LastIP      *netip.Addr `json:"last_ip,omitempty"`
}

func (ai *ASNInfo) Valid() error {
	var errs []error

	if !ai.IP.IsValid() {
		errs = append(errs, ErrIPNotValid)
	}

	if ai.Announced {
		if ai.CountryCode == nil {
			errs = append(errs, ErrNoCountryCode)
		}

		if ai.Description == nil {
			errs = append(errs, ErrNoDescription)
		}

		if ai.ASNumber == nil {
			errs = append(errs, ErrNoASNumber)
		}

		if ai.FirstIP == nil {
			errs = append(errs, ErrFirstIPNotSet)
		}

		if ai.FirstIP != nil && !ai.FirstIP.IsValid() {
			errs = append(errs, ErrFirstIPNotValid)
		}

		if ai.LastIP == nil {
			errs = append(errs, ErrLastIPNotSet)
		}

		if ai.LastIP != nil && !ai.LastIP.IsValid() {
			errs = append(errs, ErrLastIPNotValid)
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("iptoasn: can't validate ASNInfo: %w", errors.Join(errs...))
	}

	return nil
}

type Client struct {
	baseURL   string
	cli       *http.Client
	userAgent string
}

func New(baseURL string) Client {
	return Client{
		baseURL:   baseURL,
		cli:       http.DefaultClient,
		userAgent: useragent.GenUserAgent("within.website/x/web/iptoasn", "https://xeiaso.net/contact"),
	}
}

func (c Client) WithClient(cli *http.Client) Client {
	c.cli = cli
	return c
}

func (c Client) Lookup(ctx context.Context, addr netip.Addr) (*ASNInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/as/ip/"+addr.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("can't create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't do request to %s: %w", c.baseURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	var result ASNInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("can't decode response: %w", err)
	}

	if err := result.Valid(); err != nil {
		return nil, err
	}

	return &result, nil
}
