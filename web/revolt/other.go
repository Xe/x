package revolt

import (
	"fmt"
	"os"
)

// Similar to message, but created for send message function.
type SendMessage struct {
	Content     string          `json:"content"`
	Nonce       string          `json:"nonce,omitempty"`
	Attachments []string        `json:"attachments,omitempty"`
	Replies     []Replies       `json:"replies,omitempty"`
	Embeds      []SendableEmbed `json:"embeds,omitempty"`
	DeleteAfter uint            `json:"-"`
}

type SendableEmbed struct {
	Type        string `json:"type"`
	IconUrl     string `json:"icon_url,omitempty"`
	Url         string `json:"url,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Media       string `json:"media,omitempty"`
	Colour      string `json:"colour,omitempty"`
}

type Replies struct {
	Id      string `json:"id"`
	Mention bool   `json:"mention"`
}

// Set content.
func (sms *SendMessage) SetContent(content string) *SendMessage {
	sms.Content = content
	return sms
}

// Set and format content.
func (sms *SendMessage) SetContentf(format string, values ...interface{}) *SendMessage {
	sms.Content = fmt.Sprintf(format, values...)
	return sms
}

// Set delete after option.
func (sms *SendMessage) SetDeleteAfter(second uint) *SendMessage {
	sms.DeleteAfter = second
	return sms
}

// Add a new attachment.
func (sms *SendMessage) AddAttachment(attachment string) *SendMessage {
	sms.Attachments = append(sms.Attachments, attachment)
	return sms
}

// Add a new reply.
func (sms *SendMessage) AddReply(id string, mention bool) *SendMessage {
	sms.Replies = append(sms.Replies, Replies{
		Id:      id,
		Mention: mention,
	})

	return sms
}

// Create a unique nonce.
func (sms *SendMessage) CreateNonce() *SendMessage {
	sms.Nonce = genULID()
	return sms
}

// Edit channel struct.
// Please see: https://developers.revolt.chat/api/#tag/Channel-Information/paths/~1channels~1:channel/patch for more information.
type EditChannel struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Remove      string `json:"remove,omitempty"`
}

// Set name for struct.
func (ec *EditChannel) SetName(name string) *EditChannel {
	ec.Name = name
	return ec
}

// Set description for struct.
func (ec *EditChannel) SetDescription(desc string) *EditChannel {
	ec.Description = desc
	return ec
}

// Set icon for struct.
func (ec *EditChannel) SetIcon(autumn_id string) *EditChannel {
	ec.Icon = autumn_id
	return ec
}

// Set remove item.
func (ec *EditChannel) RemoveItem(item string) *EditChannel {
	ec.Remove = item
	return ec
}

// Edit server struct.
// Please see https://developers.revolt.chat/api/#tag/Server-Information/paths/~1servers~1:server/patch for more detail.
type EditServer struct {
	Name           string                  `json:"name,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Icon           string                  `json:"icon,omitempty"`
	Banner         string                  `json:"banner,omitempty"`
	Categories     []*ServerCategory       `json:"categories,omitempty"`
	SystemMessages *ServerSystemMessages   `json:"system_messages,omitempty"`
	Remove         string                  `json:"remove,omitempty"`
}

// Set name for struct
func (es *EditServer) SetName(name string) *EditServer {
	es.Name = name
	return es
}

// Set description for struct.
func (es *EditServer) SetDescription(desc string) *EditServer {
	es.Description = desc
	return es
}

// Set icon for struct.
func (es *EditServer) SetIcon(autumn_id string) *EditServer {
	es.Icon = autumn_id
	return es
}

// Set banner for struct.
func (es *EditServer) SetBanner(autumn_id string) *EditServer {
	es.Banner = autumn_id
	return es
}

// Add a new category for struct.
func (es *EditServer) AddCategory(category *ServerCategory) *EditServer {
	es.Categories = append(es.Categories, category)
	return es
}

// Set system messages for struct.
func (es *EditServer) SetSystemMessages(sm *ServerSystemMessages) *EditServer {
	es.SystemMessages = sm
	return es
}

// Set remove item.
func (es *EditServer) RemoveItem(item string) *EditServer {
	es.Remove = item
	return es
}

// Edit member struct.
// Please see https://developers.revolt.chat/api/#tag/Server-Members/paths/~1servers~1:server~1members~1:member/patch for more information.
type EditMember struct {
	Nickname string   `json:"nickname,omitempty"`
	Avatar   string   `json:"avatar,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	Remove   string   `json:"remove,omitempty"`
}

// Set nickname for struct.
func (em *EditMember) SetNickname(nick string) *EditMember {
	em.Nickname = nick
	return em
}

// Set avatar for struct.
func (em *EditMember) SetAvatar(autumn_id string) *EditMember {
	em.Avatar = autumn_id
	return em
}

// Add role for struct.
func (em *EditMember) AddRole(role_id string) *EditMember {
	em.Roles = append(em.Roles, role_id)
	return em
}

// Set remove item.
func (em *EditMember) RemoveItem(item string) *EditMember {
	em.Remove = item
	return em
}

// Edit role struct.
type EditRole struct {
	Name   string `json:"name,omitempty"`
	Colour string `json:"colour,omitempty"`
	Hoist  bool   `json:"hoist,omitempty"`
	Rank   int    `json:"rank,omitempty"`
	Remove string `json:"remove,omitempty"`
}

// Set name for struct.
func (er *EditRole) SetName(name string) *EditRole {
	er.Name = name
	return er
}

// Set valid HTML color for struct.
func (er *EditRole) SetColour(color string) *EditRole {
	er.Colour = color
	return er
}

// Set hoist boolean value for struct.
func (er *EditRole) IsHoist(hoist bool) *EditRole {
	er.Hoist = hoist
	return er
}

// Set role ranking for struct.
func (er *EditRole) SetRank(rank int) *EditRole {
	er.Rank = rank
	return er
}

// Set role ranking for struct.
func (er *EditRole) RemoveColour() *EditRole {
	er.Remove = "Colour"
	return er
}

// Edit client user struct.
// See https://developers.revolt.chat/api/#tag/User-Information/paths/~1users~1@me/patch for more information.
type EditUser struct {
	Status struct {
		Text     string `json:"text,omitempty"`
		Presence string `json:"presence,omitempty"`
	} `json:"status,omitempty"`
	Profile struct {
		Content    string `json:"content,omitempty"`
		Background string `json:"background,omitempty"`
	} `json:"profile,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Remove string `json:"remove,omitempty"`
}

// Set status for struct.
func (eu *EditUser) SetStatus(text, presence string) *EditUser {
	eu.Status = struct {
		Text     string "json:\"text,omitempty\""
		Presence string "json:\"presence,omitempty\""
	}{
		Text:     text,
		Presence: presence,
	}
	return eu
}

// Set profile informations for struct.
func (eu *EditUser) SetProfile(content, background string) *EditUser {
	eu.Profile = struct {
		Content    string "json:\"content,omitempty\""
		Background string "json:\"background,omitempty\""
	}{
		Content:    content,
		Background: background,
	}
	return eu
}

// Set avatar for struct.
func (eu *EditUser) SetAvatar(autumn_id string) *EditUser {
	eu.Avatar = autumn_id
	return eu
}

// Set remove item for struct.
func (eu *EditUser) SetRemove(item string) *EditUser {
	eu.Remove = item
	return eu
}

// revolt binary struct.
type Binary struct {
	Data []byte
}

// Save data to the given path.
func (b Binary) Save(path string) error {
	return os.WriteFile(path, b.Data, 0666)
}

// Edit bot struct
// Please see https://developers.revolt.chat/api/#tag/Bots/paths/~1bots~1:bot/patch for more information.
type EditBot struct {
	Name            string `json:"name,omitempty"`
	Public          bool   `json:"public,omitempty"`
	InteractionsUrl string `json:"interactionsURL,omitempty"`
	Remove          string `json:"remove,omitempty"`
}

// Set name for struct.
func (eb *EditBot) SetName(name string) *EditBot {
	eb.Name = name
	return eb
}

// Set public value for struct.
func (eb *EditBot) SetPublicValue(is_public bool) *EditBot {
	eb.Public = is_public
	return eb
}

// Set interaction url for struct.
func (eb *EditBot) SetInteractionsUrl(url string) *EditBot {
	eb.InteractionsUrl = url
	return eb
}

// Remove interaction url for struct.
func (eb *EditBot) RemoveInteractionsUrl() *EditBot {
	eb.Remove = "InteractionsURL"
	return eb
}
