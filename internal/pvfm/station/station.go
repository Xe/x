/*
Package station grabs fallback data from the radio station.
*/
package station

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	latestInfo Wrapper

	bugTime = flag.Int("station-poke-delay", 15, "how stale the info can get")
)

type Wrapper struct {
	Age  time.Time
	Info Info
}

type Info struct {
	Icestats struct {
		Admin            string `json:"admin"`
		BannedIPs        int    `json:"banned_IPs"`
		Build            string `json:"build"`
		Host             string `json:"host"`
		Location         string `json:"location"`
		OutgoingKbitrate int    `json:"outgoing_kbitrate"`
		ServerID         string `json:"server_id"`
		ServerStart      string `json:"server_start"`
		StreamKbytesRead int    `json:"stream_kbytes_read"`
		StreamKbytesSent int    `json:"stream_kbytes_sent"`
		Source           []struct {
			Artist             string `json:"artist"`
			AudioBitrate       int    `json:"audio_bitrate,omitempty"`
			AudioChannels      int    `json:"audio_channels,omitempty"`
			AudioInfo          string `json:"audio_info"`
			AudioSamplerate    int    `json:"audio_samplerate,omitempty"`
			Bitrate            int    `json:"bitrate"`
			Connected          int    `json:"connected"`
			Genre              string `json:"genre"`
			IceBitrate         int    `json:"ice-bitrate,omitempty"`
			IncomingBitrate    int    `json:"incoming_bitrate"`
			ListenerPeak       int    `json:"listener_peak"`
			Listeners          int    `json:"listeners"`
			Listenurl          string `json:"listenurl"`
			MetadataUpdated    string `json:"metadata_updated"`
			OutgoingKbitrate   int    `json:"outgoing_kbitrate"`
			QueueSize          int    `json:"queue_size"`
			ServerDescription  string `json:"server_description"`
			ServerName         string `json:"server_name"`
			ServerType         string `json:"server_type"`
			ServerURL          string `json:"server_url"`
			StreamStart        string `json:"stream_start"`
			Subtype            string `json:"subtype,omitempty"`
			Title              string `json:"title"`
			TotalMbytesSent    int    `json:"total_mbytes_sent"`
			YpCurrentlyPlaying string `json:"yp_currently_playing"`
		} `json:"source"`
	} `json:"icestats"`
}

func GetStats() (Info, error) {
	now := time.Now()
	if now.Before(latestInfo.Age.Add(time.Second * time.Duration(*bugTime))) {
		return latestInfo.Info, nil
	}

	i := Info{}

	resp, err := http.Get("http://dj.bronyradio.com:8000/status-json.xsl")
	if err != nil {
		return Info{}, err
	}

	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Info{}, err
	}

	err = json.Unmarshal(content, &i)
	if err != nil {
		return Info{}, err
	}

	latestInfo.Info = i
	latestInfo.Age = now

	return latestInfo.Info, nil
}
