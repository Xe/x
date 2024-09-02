/*
Package schedule grabs DJ schedule data from Ponyville FM's servers.
*/
package schedule

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// ScheduleResult is a wrapper for a list of ScheduleEntry records.
type ScheduleResult struct {
	Result []ScheduleEntry `json:"result"`
}

// ScheduleEntry is an individual schedule datum.
type ScheduleEntry struct {
	StartTime   string `json:"start_time"`
	StartUnix   int    `json:"start_unix"`
	Duration    int    `json:"duration"` // minutes
	EndTime     string `json:"end_time"`
	EndUnix     int    `json:"end_unix"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Description string `json:"description"`
	Showcard    string `json:"showcard"`
	Background  string `json:"background"`
	Timezone    string `json:"timezone"`
	Status      string `json:"status"`
}

func (s ScheduleEntry) String() string {
	startTimeUnix := time.Unix(int64(s.StartUnix), 0)
	nowWithoutNanoseconds := time.Unix(time.Now().Unix(), 0)
	dur := startTimeUnix.Sub(nowWithoutNanoseconds)

	if dur > 0 {
		return fmt.Sprintf(
			"In %s (%v %v): %s - %s",
			dur, s.StartTime, s.Timezone, s.Host, s.Name,
		)
	} else {
		return fmt.Sprintf(
			"Now: %s - %s",
			s.Host, s.Name,
		)
	}
}

var (
	latestInfo Wrapper

	bugTime = flag.Int("pvfm-schedule-poke-delay", 15, "how stale the info can get")
)

// Wrapper is a time, info pair. This is used to invalidate the cache of
// data from ponyvillefm.com.
type Wrapper struct {
	Age  time.Time
	Info *ScheduleResult
}

// Get returns schedule entries, only fetching new data at most every n
// seconds, where n is defined above.
func Get() ([]ScheduleEntry, error) {
	now := time.Now()

	if now.Before(latestInfo.Age.Add(time.Second * time.Duration(*bugTime))) {
		return latestInfo.Info.Result, nil
	}

	s := &ScheduleResult{}
	c := http.Client{
		Timeout: time.Duration(time.Second * 15),
	}

	resp, err := c.Get("http://ponyvillefm.com/data/schedule")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(content, s)
	if err != nil {
		return nil, err
	}

	// Update the age/contents of the latestInfo
	latestInfo.Info = s
	latestInfo.Age = now

	return s.Result, nil
}
