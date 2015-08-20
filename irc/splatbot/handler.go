package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/belak/irc"
)

func handlePrivmsg(c *irc.Client, e *irc.Event) {
	if strings.HasPrefix(e.Trailing(), "!splatoon") {
		splatoonLookup(c, e)
	}
}

func splatoonLookup(c *irc.Client, e *irc.Event) {
	resp, err := http.Get(url)
	if err != nil {
		c.Reply(e, "Couldn't look up splatoon maps: %s", err.Error())
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.Reply(e, "Couldn't look up splatoon maps: %s", err.Error())
		return
	}

	defer resp.Body.Close()

	var sd []SplatoonData
	err = json.Unmarshal(body, &sd)
	if err != nil {
		c.Reply(e, "Couldn't look up splatoon maps: %s", err.Error())
		return
	}

	data := sd[0]

	stage1 := data.Stages[0]
	stage2 := data.Stages[1]
	c.Reply(
		e,
		"From %s to %s, the stage rotation is %s and %s",
		data.DatetimeTermBegin, data.DatetimeTermEnd,
		englishIfy(stage1), englishIfy(stage2),
	)

	log.Printf("%s asked me to look up data", e.Identity.Nick)
}
