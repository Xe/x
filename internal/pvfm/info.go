/*
Package pvfm grabs information about PonyvilleFM from the station servers.
*/
package pvfm

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

var (
	latestInfo Wrapper

	bugTime = flag.Int("pvfm-poke-delay", 15, "how stale the info can get")
)

// The regex to check for Aerial's name.
var (
	AerialRegex = regexp.MustCompile(`Aerial`)
)

// Wrapper is a time, info pair. This is used to invalidate the cache of
// data from ponyvillefm.com.
type Wrapper struct {
	Age  time.Time
	Info Info
}

// Info is the actual information we care about. It contains information about the
// available streams.
type Info struct {
	Listeners Listeners   `json:"all"`
	Main      RadioStream `json:"one"`
	Secondary RadioStream `json:"two"`
	MusicOnly RadioStream `json:"free"`
}

// RadioStream contains data about an individual stream.
type RadioStream struct {
	Listeners  int    `json:"listeners"`
	Nowplaying string `json:"nowplaying"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	Title      string `json:"title"`
	Onair      string `json:"onair"`
	Artwork    string `json:"artwork"`
}

// Listeners contains a single variable Listeners.
type Listeners struct {
	Listeners int `json:"listeners"`
}

// GetStats returns an Info, error pair representing the latest (or cached)
// version of the statistics from the ponyvillefm servers. If there is an error
// anywhere
func GetStats() (Info, error) {
	now := time.Now()

	// If right now is before the age of the latestInfo plus the pacing time,
	// return the latestInfo Info.
	if now.Before(latestInfo.Age.Add(time.Second * time.Duration(*bugTime))) {
		return latestInfo.Info, nil
	}

	i := Info{}

	// Grab stuff from the internet
	c := &http.Client{
		Timeout: time.Second * 15,
	}

	resp, err := c.Get("http://ponyvillefm.com/data/nowplaying")
	if err != nil {
		return Info{}, fmt.Errorf("http fetch: %s %d: %v", resp.Status, resp.StatusCode, err)
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Info{}, err
	}

	err = json.Unmarshal(content, &i)
	if err != nil {
		return Info{}, fmt.Errorf("json unmarshal: %v", err)
	}

	// Update the age/contents of the latestInfo
	latestInfo.Info = i
	latestInfo.Age = now

	return latestInfo.Info, nil
}

// IsDJLive returns true if a human DJ is live or false if the auto DJ (and any
// of its playlists) is playing music.
func (i Info) IsDJLive() bool {
	return !AerialRegex.Match([]byte(i.Main.Onair))
}
