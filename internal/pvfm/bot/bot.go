package bot

import (
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrRateLimitExceeded = errors.New("bot: per-command rate limit exceeded")
)

type command struct {
	aliases  []string
	verb     string
	helptext string
}

func (c *command) Verb() string {
	return c.verb
}

func (c *command) Helptext() string {
	return c.helptext
}

// Handler is the type that bot command functions need to implement. Errors
// should be returned.
type Handler func(*discordgo.Session, *discordgo.Message, []string) error

// CommandHandler is a generic interface for types that implement a bot
// command. It is akin to http.Handler, but more comprehensive.
type CommandHandler interface {
	Verb() string
	Helptext() string

	Handler(*discordgo.Session, *discordgo.Message, []string) error
	Permissions(*discordgo.Session, *discordgo.Message, []string) error
}

type basicCommand struct {
	*command
	handler     Handler
	permissions Handler
	limiter     *rate.Limiter
}

func (bc *basicCommand) Handler(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	return bc.handler(s, m, parv)
}

func (bc *basicCommand) Permissions(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	if !bc.limiter.Allow() {
		return ErrRateLimitExceeded
	}

	return bc.permissions(s, m, parv)
}

// The "default" command set, useful for simple bot projects.
var (
	DefaultCommandSet = NewCommandSet()
)

// Command handling errors.
var (
	ErrAlreadyExists     = errors.New("bot: command already exists")
	ErrNoSuchCommand     = errors.New("bot: no such command exists")
	ErrNoPermissions     = errors.New("bot: you do not have permissions for this command")
	ErrParvCountMismatch = errors.New("bot: parameter count mismatch")
)

// The default command prefix. Command `foo` becomes `.foo` in chat, etc.
const (
	DefaultPrefix = "."
)

// NewCommand creates an anonymous command and adds it to the default CommandSet.
func NewCommand(verb, helptext string, handler, permissions Handler) error {
	return DefaultCommandSet.Add(NewBasicCommand(verb, helptext, handler, permissions))
}

// NewBasicCommand creates a CommandHandler instance using the implementation
// functions supplied as arguments.
func NewBasicCommand(verb, helptext string, permissions, handler Handler) CommandHandler {
	return &basicCommand{
		command: &command{
			verb:     verb,
			helptext: helptext,
		},
		handler:     handler,
		permissions: permissions,
		limiter:     rate.NewLimiter(rate.Every(5*time.Second), 1),
	}
}

// CommandSet is a group of bot commands similar to an http.ServeMux.
type CommandSet struct {
	sync.Mutex
	cmds map[string]CommandHandler

	Prefix string
}

// NewCommandSet creates a new command set with the `help` command pre-loaded.
func NewCommandSet() *CommandSet {
	cs := &CommandSet{
		cmds:   map[string]CommandHandler{},
		Prefix: DefaultPrefix,
	}

	cs.AddCmd("help", "Shows help for the bot", NoPermissions, cs.help)

	return cs
}

// NoPermissions is a simple middelware function that allows all command invocations
// to pass the permissions check.
func NoPermissions(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	return nil
}

// AddCmd is syntactic sugar for cs.Add(NewBasicCommand(args...))
func (cs *CommandSet) AddCmd(verb, helptext string, permissions, handler Handler) error {
	return cs.Add(NewBasicCommand(verb, helptext, permissions, handler))
}

// Add adds a single command handler to the CommandSet. This can be done at runtime
// but it is suggested to only add commands on application boot.
func (cs *CommandSet) Add(h CommandHandler) error {
	cs.Lock()
	defer cs.Unlock()

	v := strings.ToLower(h.Verb())

	if _, ok := cs.cmds[v]; ok {
		return ErrAlreadyExists
	}

	cs.cmds[v] = h

	return nil
}

// Run makes a CommandSet compatible with discordgo event dispatching.
func (cs *CommandSet) Run(s *discordgo.Session, msg *discordgo.Message) error {
	cs.Lock()
	defer cs.Unlock()

	if strings.HasPrefix(msg.Content, cs.Prefix) {
		params := strings.Fields(msg.Content)
		verb := strings.ToLower(params[0][1:])

		cmd, ok := cs.cmds[verb]
		if !ok {
			return ErrNoSuchCommand
		}

		err := cmd.Permissions(s, msg, params)
		if err != nil {
			log.Printf("Permissions error: %s: %v", msg.Author.Username, err)
			s.ChannelMessageSend(msg.ChannelID, "You don't have permissions for that, sorry.")
			return ErrNoPermissions
		}

		err = cmd.Handler(s, msg, params)
		if err != nil {
			log.Printf("command handler error: %v", err)
			s.ChannelMessageSend(msg.ChannelID, "error when running that command: "+err.Error())
			return err
		}
	}

	return nil
}
