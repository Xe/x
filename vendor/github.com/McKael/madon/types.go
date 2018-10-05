/*
Copyright 2017-2018 Mikael Berthe
Copyright 2017 Ollivier Robert

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"time"
)

// ActivityTime is a custom type for the time returned by instance/activity
type ActivityTime struct {
	time.Time
}

// Client contains data for a madon client application
type Client struct {
	Name        string // Name of the client
	ID          string // Application ID
	Secret      string // Application secret
	APIBase     string // API prefix URL
	InstanceURL string // Instance base URL

	UserToken *UserToken // User token
}

/*
Entities - Everything manipulated/returned by the API
*/

// DomainName is a domain name string, as returned by the domain_blocks API
type DomainName string

// InstancePeer is a peer name, as returned by the instance/peers API
type InstancePeer string

// Account represents a Mastodon account entity
type Account struct {
	ID             int64     `json:"id,string"`
	Username       string    `json:"username"`
	Acct           string    `json:"acct"`
	DisplayName    string    `json:"display_name"`
	Note           string    `json:"note"`
	URL            string    `json:"url"`
	Avatar         string    `json:"avatar"`
	AvatarStatic   string    `json:"avatar_static"`
	Header         string    `json:"header"`
	HeaderStatic   string    `json:"header_static"`
	Locked         bool      `json:"locked"`
	CreatedAt      time.Time `json:"created_at"`
	FollowersCount int64     `json:"followers_count"`
	FollowingCount int64     `json:"following_count"`
	StatusesCount  int64     `json:"statuses_count"`
	Moved          *Account  `json:"moved"`
	Source         *struct { // Used for verify_credentials
		Privacy   string `json:"privacy"`
		Sensitive bool   `json:"sensitive"`
		Note      string `json:"note"`
	} `json:"source"`
}

// Application represents a Mastodon application entity
type Application struct {
	Name    string `json:"name"`
	Website string `json:"website"`
}

// Attachment represents a Mastodon media attachment entity
type Attachment struct {
	ID         int64   `json:"id,string"`
	Type       string  `json:"type"`
	URL        string  `json:"url"`
	RemoteURL  *string `json:"remote_url"`
	PreviewURL string  `json:"preview_url"`
	TextURL    *string `json:"text_url"`
	Meta       *struct {
		Original struct {
			Size   string  `json:"size"`
			Aspect float64 `json:"aspect"`
			Width  int     `json:"width"`
			Height int     `json:"height"`
		} `json:"original"`
		Small struct {
			Size   string  `json:"size"`
			Aspect float64 `json:"aspect"`
			Width  int     `json:"width"`
			Height int     `json:"height"`
		} `json:"small"`
	} `json:"meta"`
	Description *string `json:"description"`
}

// Card represents a Mastodon preview card entity
type Card struct {
	URL          string  `json:"url"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Image        string  `json:"image"`
	Type         *string `json:"type"`
	AuthorName   *string `json:"author_name"`
	AuthorURL    *string `json:"author_url"`
	ProviderName *string `json:"provider_name"`
	ProviderURL  *string `json:"provider_url"`
	EmbedURL     *string `json:"embed_url"`
	HTML         *string `json:"html"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
}

// Context represents a Mastodon context entity
type Context struct {
	Ancestors   []Status `json:"ancestors"`
	Descendants []Status `json:"descendants"`
}

// Emoji represents a Mastodon emoji entity
type Emoji struct {
	ShortCode       string `json:"shortcode"`
	URL             string `json:"url"`
	StaticURL       string `json:"static_url"`
	VisibleInPicker bool   `json:"visible_in_picker"`
}

// Error represents a Mastodon error entity
type Error struct {
	Text string `json:"error"`
}

// Instance represents a Mastodon instance entity
type Instance struct {
	URI         string `json:"uri"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Email       string `json:"email"`
	Version     string `json:"version"`

	URLs struct {
		SteamingAPI string `json:"streaming_api"`
	} `json:"urls"`
	Stats struct {
		UserCount   int64 `json:"user_count"`
		StatusCount int64 `json:"status_count"`
		DomainCount int64 `json:"domain_count"`
	} `json:"stats"`
	Thumbnail      *string  `json:"thumbnail"`
	Languages      []string `json:"languages"`
	ContactAccount *Account `json:"contact_account"`
}

// List represents a Mastodon list entity
type List struct {
	ID    int64  `json:"id,string"`
	Title string `json:"title"`
}

// Mention represents a Mastodon mention entity
type Mention struct {
	ID       int64  `json:"id,string"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Acct     string `json:"acct"`
}

// Notification represents a Mastodon notification entity
type Notification struct {
	ID        int64     `json:"id,string"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Account   *Account  `json:"account"`
	Status    *Status   `json:"status"`
}

// Relationship represents a Mastodon relationship entity
type Relationship struct {
	ID        int64 `json:"id,string"`
	Following bool  `json:"following"`
	//ShowingReblogs      bool  `json:"showing_reblogs"` // Incoherent type
	FollowedBy          bool `json:"followed_by"`
	Blocking            bool `json:"blocking"`
	Muting              bool `json:"muting"`
	Requested           bool `json:"requested"`
	DomainBlocking      bool `jsin:"domain_blocking"`
	MutingNotifications bool `json:"muting_notifications"`
}

// Report represents a Mastodon report entity
type Report struct {
	ID          int64  `json:"id,string"`
	ActionTaken string `json:"action_taken"`
}

// Results represents a Mastodon search results entity
type Results struct {
	Accounts []Account `json:"accounts"`
	Statuses []Status  `json:"statuses"`
	Hashtags []string  `json:"hashtags"`
}

// Status represents a Mastodon status entity
type Status struct {
	ID                 int64        `json:"id,string"`
	URI                string       `json:"uri"`
	URL                string       `json:"url"`
	Account            *Account     `json:"account"`
	InReplyToID        *int64       `json:"in_reply_to_id,string"`
	InReplyToAccountID *int64       `json:"in_reply_to_account_id,string"`
	Reblog             *Status      `json:"reblog"`
	Content            string       `json:"content"`
	CreatedAt          time.Time    `json:"created_at"`
	ReblogsCount       int64        `json:"reblogs_count"`
	FavouritesCount    int64        `json:"favourites_count"`
	Reblogged          bool         `json:"reblogged"`
	Favourited         bool         `json:"favourited"`
	Muted              bool         `json:"muted"`
	Pinned             bool         `json:"pinned"`
	Sensitive          bool         `json:"sensitive"`
	SpoilerText        string       `json:"spoiler_text"`
	Visibility         string       `json:"visibility"`
	MediaAttachments   []Attachment `json:"media_attachments"`
	Mentions           []Mention    `json:"mentions"`
	Tags               []Tag        `json:"tags"`
	Emojis             []Emoji      `json:"emojis"`
	Application        *Application `json:"application"`
	Language           *string      `json:"language"`
}

// Tag represents a Mastodon tag entity
type Tag struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// WeekActivity represents a Mastodon instance activity "week" entity
type WeekActivity struct {
	Week          ActivityTime `json:"week"`
	Statuses      int64        `json:"statuses,string"`
	Logins        int64        `json:"logins,string"`
	Registrations int64        `json:"registrations,string"`
}
