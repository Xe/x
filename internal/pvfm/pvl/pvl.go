/*
Package pvl grabs Ponyville Live data.
*/
package pvl

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

var (
	latestInfo Wrapper

	bugTime = flag.Int64("pvl-poke-delay", 666, "how stale pvl info can get")
)

type Wrapper struct {
	Age  time.Time
	Info Calendar
}

type Calendar struct {
	Result []struct {
		Body       interface{} `json:"body"`
		EndTime    int64       `json:"end_time"`
		Guid       string      `json:"guid"`
		ID         float64     `json:"id"`
		ImageURL   string      `json:"image_url"`
		IsAllDay   bool        `json:"is_all_day"`
		IsPromoted bool        `json:"is_promoted"`
		Location   interface{} `json:"location"`
		Range      string      `json:"range"`
		StartTime  int64       `json:"start_time"`
		StationID  int64       `json:"station_id"`
		Title      string      `json:"title"`
		WebURL     string      `json:"web_url"`
	} `json:"result"`
	Status string `json:"status"`
}

// Get grabs the station schedule from Ponyville Live.
func Get() (Calendar, error) {
	now := time.Now()
	if now.Before(latestInfo.Age.Add(time.Second * time.Duration(*bugTime))) {
		return latestInfo.Info, nil
	}

	c := Calendar{}
	resp, err := http.Get("http://ponyvillelive.com/api/schedule/index/station/ponyvillefm")
	if err != nil {
		return Calendar{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 == 5 {
		return Calendar{}, errors.New("pvl: API returned " + strconv.Itoa(resp.StatusCode) + " " + resp.Status)
	}

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Calendar{}, err
	}

	err = json.Unmarshal(content, &c)
	if err != nil {
		return Calendar{}, err
	}

	latestInfo.Info = c
	latestInfo.Age = now
	return latestInfo.Info, nil
}
