package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	derpiSearch "within.website/x/cmd/aerial/derpi"
	"within.website/x/internal/pvfm"
	pvfmschedule "within.website/x/internal/pvfm/schedule"
	"within.website/x/internal/pvfm/station"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// randomRange gives a random whole integer between the given integers [min, max)
func randomRange(min, max int) int {
	return rand.Intn(max-min) + min
}

func pesterLink(s *discordgo.Session, m *discordgo.MessageCreate) {
	if musicLinkRegex.Match([]byte(m.Content)) {
		i, err := pvfm.GetStats()
		if err != nil {
			log.Println(err)
			return
		}

		if i.IsDJLive() && m.ChannelID == youtubeSpamRoomID {
			s.ChannelMessageSend(m.ChannelID, "Please be mindful sharing links to music when a DJ is performing. Thanks!")
		}
	}
}

func stats(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	i, err := pvfm.GetStats()
	if err != nil {
		log.Printf("Error getting the station info: %v, falling back to plan b", err)
		return doStatsFromStation(s, m, parv)
	}

	st, err := station.GetStats()
	if err != nil {
		return err
	}

	var l int
	var peak int

	for _, source := range st.Icestats.Source {
		l = l + source.Listeners
		peak = peak + source.ListenerPeak
	}

	// checks if the event is currently happening
	outputEmbed := NewEmbed().
		SetTitle("Listener Statistics").
		SetDescription("Use `;streams` if you need a link to the radio!\nTotal listeners across all stations: " + strconv.Itoa(i.Listeners.Listeners) + " with a maximum  of " + strconv.Itoa(peak) + ".")

	outputEmbed.AddField("🎵 Main", strconv.Itoa(i.Main.Listeners)+" listeners.\n"+i.Main.Nowplaying)
	outputEmbed.AddField("🎵 Chill", strconv.Itoa(i.Secondary.Listeners)+" listeners.\n"+i.Secondary.Nowplaying)
	outputEmbed.AddField("🎵 Free! (no DJ sets)", strconv.Itoa(i.MusicOnly.Listeners)+" listeners.\n"+i.MusicOnly.Nowplaying)

	outputEmbed.InlineAllFields()

	s.ChannelMessageSendEmbed(m.ChannelID, outputEmbed.MessageEmbed)

	return nil
}

func schedule(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	schEntries, err := pvfmschedule.Get()
	if err != nil {
		return err
	}

	// Create embed object
	outputEmbed := NewEmbed().
		SetTitle("Upcoming Shows").
		SetDescription("These are the upcoming shows and events airing soon on PVFM 1.\n[Convert to your timezone](https://www.worldtimebuddy.com/?pl=1&lid=100&h=100)")

	for _, entry := range schEntries {

		// Format countdown timer
		startTimeUnix := time.Unix(int64(entry.StartUnix), 0)
		nowWithoutNanoseconds := time.Unix(time.Now().Unix(), 0)
		dur := startTimeUnix.Sub(nowWithoutNanoseconds)

		// Show "Live Now!" if the timer is less than 0h0m0s
		if dur > 0 {
			outputEmbed.AddField(":musical_note:  "+entry.Host+" - "+entry.Name, entry.StartTime+" "+entry.Timezone+"\nAirs in "+dur.String())
		} else {
			outputEmbed.AddField(":musical_note:  "+entry.Host+" - "+entry.Name, "Live now!")
		}
	}

	s.ChannelMessageSendEmbed(m.ChannelID, outputEmbed.MessageEmbed)
	return nil
}

func doStationRequest(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	stats, err := station.GetStats()
	if err != nil {
		return err
	}

	result := fmt.Sprintf(
		"Now playing: %s - %s on Ponyville FM!",
		stats.Icestats.Source[0].Title,
		stats.Icestats.Source[0].Artist,
	)

	s.ChannelMessageSend(m.ChannelID, result)
	return nil
}

func doStatsFromStation(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	st, err := station.GetStats()
	if err != nil {
		return err
	}

	var l int
	var peak int

	for _, source := range st.Icestats.Source {
		l = l + source.Listeners
		peak = peak + source.ListenerPeak
	}

	result := []string{
		fmt.Sprintf("Current listeners: %d with a maximum of %d!", l, peak),
	}

	s.ChannelMessageSend(m.ChannelID, strings.Join(result, "\n"))
	return nil
}

func curTime(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("The time currently is %s\nUse <https://www.worldtimebuddy.com/?pl=1&lid=100&h=100> to convert UTC to your local timezone.", time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))

	return nil
}

const pvfmList = `SSL SAFE Streams
PonyvilleFM Europe OGG Stream:
https://dj.bronyradio.com/pvfm1.ogg
PVFM AAC+ 3G/4G Mobile Stream:
https://dj.bronyradio.com/pvfm1mobile.aac
PonyvilleFM Free MP3 24/7 Pony Stream:
https://dj.bronyradio.com/pvfmfree.mp3
PonyvilleFM Free OGG 24/7 Pony Stream:
https://dj.bronyradio.com/pvfmfree.ogg
PVFM OPUS Stream:
https://dj.bronyradio.com/pvfmopus.ogg
PonyvilleFM Europe Stream:
https://dj.bronyradio.com/stream.mp3
PonyvilleFM High Quality Europe Stream:
https://dj.bronyradio.com/streamhq.mp3

Legacy Streams (non https)
PonyvilleFM Europe OGG Stream:
http://dj.bronyradio.com:8000/pvfm1.ogg
PonyvilleFM Europe Stream:
http://dj.bronyradio.com:8000/stream.mp3
PonyvilleFM Free MP3 24/7 Pony Stream:
http://dj.bronyradio.com:8000/pvfmfree.mp3
PonyvilleFM Free OGG 24/7 Pony Stream:
http://dj.bronyradio.com:8000/pvfmfree.ogg
PVFM AAC+ 3G/4G Mobile Stream:
http://dj.bronyradio.com:8000/pvfm1mobile.aac`

func streams(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	// start building custom embed
	outputEmbed := NewEmbed().
		SetTitle("Stream Links").
		SetDescription("These are direct feeds of the live streams; most browsers and media players can play them!")

	// PVFM
	outputEmbed.AddField(":musical_note:  PVFM Servers", pvfmList)
	// Luna Radio
	outputEmbed.AddField(":musical_note:  Luna Radio Servers", "Luna Radio MP3 128Kbps Stream:\n<http://radio.ponyvillelive.com:8002/stream.mp3>\nLuna Radio Mobile MP3 64Kbps Stream:\n<http://radio.ponyvillelive.com:8002/mobile?;stream.mp3>\n")
	// Recordings
	outputEmbed.AddField(":cd:  DJ Recordings", "Archive\n<https://pvfm.within.lgbt/var/93252527679639552/>\nLegacy Archive\n<https://pvfm.within.lgbt/BronyRadio/>")

	s.ChannelMessageSendEmbed(m.ChannelID, outputEmbed.MessageEmbed)

	// no errors yay!!!!
	return nil
}

func derpi(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	if m.ChannelID == "292755043684450304" {

		searchResults, err := derpiSearch.SearchDerpi(m.Content[7:len(m.Content)]) // Safe tag will be added in derpi/derpi.go
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An error occured.")
			return err
		}
		if len(searchResults.Search) < 1 {
			s.ChannelMessageSend(m.ChannelID, "Error: No results")
			return nil
		}
		derpiImage := searchResults.Search[randomRange(0, len(searchResults.Search))]

		tags := strings.Split(derpiImage.Tags, ", ") // because this isn't an array for some reason

		// Check for artist tag
		artist := ""
		for _, tag := range tags {
			if strings.Contains(tag, "artist:") {
				artist = tag[7:]
			}
		}

		outputEmbed := NewEmbed().
			SetTitle("Derpibooru Image").
			SetURL("https://derpibooru.org/" + derpiImage.ID).
			SetDescription(derpiImage.Description).
			SetImage("http:" + derpiImage.Image).
			SetFooter("Image score: " + strconv.Itoa(derpiImage.Score) + " | Uploaded: " + derpiImage.CreatedAt.String())

		// Credit the artist!
		if artist == "" {
			outputEmbed.SetAuthor("No artist")
		} else {
			outputEmbed.SetAuthor("Artist: " + artist)
		}

		s.ChannelMessageSendEmbed(m.ChannelID, outputEmbed.MessageEmbed)
	} else {
		s.ChannelMessageSend(m.ChannelID, "Please use this command in <#292755043684450304> only.")
	}
	return nil
}

func weather(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	responses := []string{
		"Cloudy with a chance of meatballs.",
		"It's currently pouring down even more than Pinkie.",
		"It's the most overcast I've ever seen. In other words, same as always.",
		"Do you have a better conversation starter than that?",
		"There's at least 5 or 6 weather right now, my dude.",
		"It's soggy enough for Rainbow Dash to get fired, if she didn't have a literal deity keeping her in charge.",
		"Surprisingly, the weather is pretty alright.",
		"You'd be happy to know that it's hot enough to make a phoenix sweat.",
		"The weather right now is like you took London and stuck it in a dishwasher.",
		"The Crystal Empire is warmer than this weather.",
	}

	s.ChannelMessageSend(m.ChannelID, responses[randomRange(0, len(responses))])

	return nil
}
