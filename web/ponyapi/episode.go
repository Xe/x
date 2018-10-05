package ponyapi

// Episode is an episode of My Little Pony: Friendship is Magic.
type Episode struct {
	AirDate int    `json:"air_date"`
	Episode int    `json:"episode"`
	IsMovie bool   `json:"is_movie"`
	Name    string `json:"name"`
	Season  int    `json:"season"`
}

type episodeWrapper struct {
	Episode *Episode `json:"episode"`
}

type episodes struct {
	Episodes []Episode `json:"episodes"`
}
