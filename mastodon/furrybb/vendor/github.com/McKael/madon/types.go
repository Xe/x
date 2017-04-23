/*
Copyright 2017 Ollivier Robert
Copyright 2017 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"time"
)

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

// Account represents a Mastodon account entity
type Account struct {
	ID             int       `json:"id"`
	Username       string    `json:"username"`
	Acct           string    `json:"acct"`
	DisplayName    string    `json:"display_name"`
	Note           string    `json:"note"`
	URL            string    `json:"url"`
	Avatar         string    `json:"avatar"`
	Header         string    `json:"header"`
	Locked         bool      `json:"locked"`
	CreatedAt      time.Time `json:"created_at"`
	FollowersCount int       `json:"followers_count"`
	FollowingCount int       `json:"following_count"`
	StatusesCount  int       `json:"statuses_count"`
}

// Application represents a Mastodon application entity
type Application struct {
	Name    string `json:"name"`
	Website string `json:"website"`
}

// Attachment represents a Mastodon attachement entity
type Attachment struct {
	ID         int    `json:"iD"`
	Type       string `json:"type"`
	URL        string `json:"url"`
	RemoteURL  string `json:"remote_url"`
	PreviewURL string `json:"preview_url"`
	TextURL    string `json:"text_url"`
}

// Card represents a Mastodon card entity
type Card struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

// Context represents a Mastodon context entity
type Context struct {
	Ancestors   []Status `json:"ancestors"`
	Descendents []Status `json:"descendents"`
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
}

// Mention represents a Mastodon mention entity
type Mention struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Acct     string `json:"acct"`
}

// Notification represents a Mastodon notification entity
type Notification struct {
	ID        int       `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Account   *Account  `json:"account"`
	Status    *Status   `json:"status"`
}

// Relationship represents a Mastodon relationship entity
type Relationship struct {
	ID         int  `json:"id"`
	Following  bool `json:"following"`
	FollowedBy bool `json:"followed_by"`
	Blocking   bool `json:"blocking"`
	Muting     bool `json:"muting"`
	Requested  bool `json:"requested"`
}

// Report represents a Mastodon report entity
type Report struct {
	ID          int    `json:"iD"`
	ActionTaken string `json:"action_taken"`
}

// Results represents a Mastodon results entity
type Results struct {
	Accounts []Account `json:"accounts"`
	Statuses []Status  `json:"statuses"`
	Hashtags []string  `json:"hashtags"`
}

// Status represents a Mastodon status entity
type Status struct {
	ID                 int          `json:"id"`
	URI                string       `json:"uri"`
	URL                string       `json:"url"`
	Account            *Account     `json:"account"`
	InReplyToID        int          `json:"in_reply_to_id"`
	InReplyToAccountID int          `json:"in_reply_to_account_id"`
	Reblog             *Status      `json:"reblog"`
	Content            string       `json:"content"`
	CreatedAt          time.Time    `json:"created_at"`
	ReblogsCount       int          `json:"reblogs_count"`
	FavouritesCount    int          `json:"favourites_count"`
	Reblogged          bool         `json:"reblogged"`
	Favourited         bool         `json:"favourited"`
	Sensitive          bool         `json:"sensitive"`
	SpoilerText        string       `json:"spoiler_text"`
	Visibility         string       `json:"visibility"`
	MediaAttachments   []Attachment `json:"media_attachments"`
	Mentions           []Mention    `json:"mentions"`
	Tags               []Tag        `json:"tags"`
	Application        Application  `json:"application"`
}

// Tag represents a Mastodon tag entity
type Tag struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
