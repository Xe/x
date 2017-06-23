package finch

import (
	"bytes"
	"gopkg.in/telegram-bot-api.v4"
	"sync"
)

// Help contains information about a command,
// used for showing info in the help command.
type Help struct {
	Name        string
	Description string
	Example     string
	Botfather   [][]string
}

// String converts a Help struct into a pretty string.
//
// full makes each command item multiline with extra newlines.
func (h Help) String(full bool) string {
	b := &bytes.Buffer{}

	b.WriteString(h.Name)
	if full {
		b.WriteString("\n")
	} else {
		b.WriteString(" - ")
	}
	b.WriteString(h.Description)
	b.WriteString("\n")

	if full {
		b.WriteString("Example: ")
		b.WriteString(h.Example)
		b.WriteString("\n")
	}

	b.WriteString("\n")

	return b.String()
}

// BotfatherString formats a Help struct into something for Botfather.
func (h Help) BotfatherString() string {
	if len(h.Botfather) == 0 {
		return ""
	}

	b := bytes.Buffer{}

	for k, v := range h.Botfather {
		b.WriteString(v[0])
		b.WriteString(" - ")
		b.WriteString(v[1])
		if k+1 != len(h.Botfather) {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// Command contains the methods a must have.
type Command interface {
	Help() Help
	Init(*CommandState, *Finch) error
	ShouldExecute(tgbotapi.Message) bool
	Execute(tgbotapi.Message) error
	ExecuteWaiting(tgbotapi.Message) error
	ExecuteCallback(tgbotapi.CallbackQuery) error
	IsHighPriority(tgbotapi.Message) bool
}

// CommandBase is a default Command that handles various tasks for you,
// and allows for you to not have to write empty methods.
type CommandBase struct {
	*CommandState
	*Finch
}

// Help returns an empty Help struct.
func (CommandBase) Help() Help { return Help{} }

// Init sets CommandState equal to the current CommandState.
//
// If you overwrite this method, you must still set CommandState and Finch
// to the correct values!
func (cmd *CommandBase) Init(c *CommandState, f *Finch) error {
	cmd.CommandState = c
	cmd.Finch = f

	return nil
}

// ShouldExecute returns false, you should overwrite this method.
func (CommandBase) ShouldExecute(tgbotapi.Message) bool { return false }

// Execute returns nil to show no error, you should overwrite this method.
func (CommandBase) Execute(tgbotapi.Message) error { return nil }

// ExecuteWaiting returns nil to show no error, you may overwrite this
// when you are expecting to get a reply that is not a command.
func (CommandBase) ExecuteWaiting(tgbotapi.Message) error { return nil }

// ExecuteCallback returns nil to show no error, you may overwrite this
// when you are expecting to get a callback query.
func (CommandBase) ExecuteCallback(tgbotapi.CallbackQuery) error { return nil }

// IsHighPriority return false, you should overwrite this function to
// return true if your command needs to execute before checking for
// commands that are waiting for a reply or keyboard input.
func (CommandBase) IsHighPriority(tgbotapi.Message) bool { return false }

// Get fetches an item from the Config struct.
func (cmd CommandBase) Get(key string) interface{} {
	return cmd.Finch.Config[key]
}

// Set sets an item in the Config struct, then saves it.
func (cmd CommandBase) Set(key string, value interface{}) {
	cmd.Finch.Config[key] = value
	cmd.Finch.Config.Save()
}

type userWaitMap struct {
	mutex    *sync.Mutex
	userWait map[int]bool
}

// CommandState is the current state of a command.
// It contains the command and if the command is waiting for a reply.
type CommandState struct {
	Command             Command
	waitingForReplyUser userWaitMap
}

// NewCommandState creates a new CommandState with an initialized map.
func NewCommandState(cmd Command) *CommandState {
	return &CommandState{
		Command: cmd,
		waitingForReplyUser: userWaitMap{
			mutex:    &sync.Mutex{},
			userWait: map[int]bool{},
		},
	}
}

// IsWaiting checks if the current CommandState is waiting for input from
// this user.
func (state *CommandState) IsWaiting(user int) bool {
	state.waitingForReplyUser.mutex.Lock()
	defer state.waitingForReplyUser.mutex.Unlock()
	if v, ok := state.waitingForReplyUser.userWait[user]; ok {
		return v
	}

	return false
}

// SetWaiting sets that the bot should expect user input from this user.
func (state *CommandState) SetWaiting(user int) {
	state.waitingForReplyUser.mutex.Lock()
	defer state.waitingForReplyUser.mutex.Unlock()
	state.waitingForReplyUser.userWait[user] = true
}

// ReleaseWaiting sets that the bot should not expect any input from
// this user.
func (state *CommandState) ReleaseWaiting(user int) {
	state.waitingForReplyUser.mutex.Lock()
	defer state.waitingForReplyUser.mutex.Unlock()
	state.waitingForReplyUser.userWait[user] = false

}

// InlineCommand is a single command executed for an Inline Query.
type InlineCommand interface {
	Execute(*Finch, tgbotapi.InlineQuery) error
}
