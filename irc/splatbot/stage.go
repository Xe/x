package main

type SplatoonData struct {
	DatetimeTermBegin string   `json:"datetime_term_begin"`
	DatetimeTermEnd   string   `json:"datetime_term_end"`
	Stages            []*Stage `json:"stages"`
}

type Stage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
