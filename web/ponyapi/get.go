package ponyapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	endpoint = "https://ponyapi.apps.xeserv.us/"
)

func getJSON(fragment string) *http.Request {
	req, err := http.NewRequest(http.MethodGet, endpoint+fragment, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	return req
}

func readData(resp *http.Response) ([]byte, error) {
	if resp.StatusCode%100 != 2 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return data, nil
}

// ReadEpisode reads information about an invididual episode from an HTTP response.
func ReadEpisode(resp *http.Response) (*Episode, error) {
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var ewr episodeWrapper
	err := json.NewDecoder(resp.Body).Decode(&ewr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ewr.Episode, nil
}

// ReadEpisodes reads a slice of episode information out of a HTTP response.
func ReadEpisodes(resp *http.Response) ([]Episode, error) {
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	var eswr episodes
	err := json.NewDecoder(resp.Body).Decode(&eswr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return eswr.Episodes, nil
}

// Newest returns information on the newest episode or an error.
func Newest() *http.Request {
	return getJSON("/newest")
}

// LastAired returns information on the most recently aried episode
// or an error.
func LastAired() *http.Request {
	return getJSON("/last_aired")
}

// Random returns information on a random episode.
func Random() *http.Request {
	return getJSON("/random")
}

// GetEpisode gets information about season x episode y or an error.
func GetEpisode(season, episode int) *http.Request {
	return getJSON(fmt.Sprintf("/season/%d/episode/%d", season, episode))
}

// AllEpisodes gets all information on all episodes or returns an error.
func AllEpisodes() *http.Request {
	return getJSON("/all")
}

// GetSeason returns all information on season x or returns an error.
func GetSeason(season int) *http.Request {
	return getJSON(fmt.Sprintf("/season/%d", season))
}

// Search takes the give search terms and uses that to search the
// list of episodes.
func Search(query string) *http.Request {
	path, err := url.Parse("/search")
	if err != nil {
		panic(err)
	}

	q := path.Query()
	q.Set("q", query)

	path.RawQuery = q.Encode()

	return getJSON(path.String())
}
