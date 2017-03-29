package splattus

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

func Lookup() ([]SplatoonData, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var sd []SplatoonData
	err = json.Unmarshal(body, &sd)
	if err != nil {
		return nil, err
	}

	return sd, nil
}
