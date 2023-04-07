// Package cryptocompare fetches the latest USD price of a given crpytocurrency from CryptoCompare
package cryptocompare

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"within.website/x/web"
	_ "within.website/x/web/useragent"
)

func Get(symbol string, currencies []string) (map[string]float64, error) {
	u, err := url.Parse("https://min-api.cryptocompare.com/data/price")
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("fsym", symbol)
	q.Set("tsyms", strings.Join(currencies, ","))

	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("cryptocompare: can't fetch result: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}

	defer resp.Body.Close()

	result := make(map[string]float64)
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("cryptocompare: can't decode result: %w", err)
	}

	return result, nil
}
