package plex

type Webhook struct {
	Event    string    `json:"event"`
	User     bool      `json:"user"`
	Owner    bool      `json:"owner"`
	Account  Account   `json:"Account"`
	Server   *Server   `json:"Server"`
	Player   *Player   `json:"Player"`
	Metadata *Metadata `json:"Metadata"`
}

type Account struct {
	ID    int    `json:"id"`
	Thumb string `json:"thumb"`
	Title string `json:"title"`
}

type Server struct {
	Title string `json:"title"`
	UUID  string `json:"uuid"`
}

type Player struct {
	Local         bool   `json:"local"`
	PublicAddress string `json:"publicAddress"`
	Title         string `json:"title"`
	UUID          string `json:"uuid"`
}

type GUID0 struct {
	ID string `json:"id"`
}

type Rating struct {
	Image string  `json:"image"`
	Value float64 `json:"value"`
	Type  string  `json:"type"`
}

type Director struct {
	ID     int    `json:"id"`
	Filter string `json:"filter"`
	Tag    string `json:"tag"`
	TagKey string `json:"tagKey"`
}

type Writer struct {
	ID     int    `json:"id"`
	Filter string `json:"filter"`
	Tag    string `json:"tag"`
	TagKey string `json:"tagKey"`
}

type Role struct {
	ID     int    `json:"id"`
	Filter string `json:"filter"`
	Tag    string `json:"tag"`
	TagKey string `json:"tagKey"`
	Role   string `json:"role"`
	Thumb  string `json:"thumb"`
}

type Metadata struct {
	LibrarySectionType    string     `json:"librarySectionType"`
	RatingKey             string     `json:"ratingKey"`
	Key                   string     `json:"key"`
	ParentRatingKey       string     `json:"parentRatingKey"`
	GrandparentRatingKey  string     `json:"grandparentRatingKey"`
	GUID                  string     `json:"guid"`
	ParentGUID            string     `json:"parentGuid"`
	GrandparentGUID       string     `json:"grandparentGuid"`
	Type                  string     `json:"type"`
	Title                 string     `json:"title"`
	GrandparentKey        string     `json:"grandparentKey"`
	ParentKey             string     `json:"parentKey"`
	LibrarySectionTitle   string     `json:"librarySectionTitle"`
	LibrarySectionID      int        `json:"librarySectionID"`
	LibrarySectionKey     string     `json:"librarySectionKey"`
	GrandparentTitle      string     `json:"grandparentTitle"`
	ParentTitle           string     `json:"parentTitle"`
	OriginalTitle         string     `json:"originalTitle"`
	ContentRating         string     `json:"contentRating"`
	Summary               string     `json:"summary"`
	Index                 int        `json:"index"`
	ParentIndex           int        `json:"parentIndex"`
	AudienceRating        float64    `json:"audienceRating"`
	ViewOffset            int        `json:"viewOffset"`
	LastViewedAt          int        `json:"lastViewedAt"`
	Year                  int        `json:"year"`
	Thumb                 string     `json:"thumb"`
	Art                   string     `json:"art"`
	ParentThumb           string     `json:"parentThumb"`
	GrandparentThumb      string     `json:"grandparentThumb"`
	GrandparentArt        string     `json:"grandparentArt"`
	Duration              int        `json:"duration"`
	OriginallyAvailableAt string     `json:"originallyAvailableAt"`
	AddedAt               int        `json:"addedAt"`
	UpdatedAt             int        `json:"updatedAt"`
	AudienceRatingImage   string     `json:"audienceRatingImage"`
	GUID0                 []GUID0    `json:"Guid"`
	Rating                []Rating   `json:"Rating"`
	Director              []Director `json:"Director"`
	Writer                []Writer   `json:"Writer"`
	Role                  []Role     `json:"Role"`
}
