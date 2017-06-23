package finch

import (
	"github.com/getsentry/raven-go"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"regexp"
	"strings"
)

var commands []*CommandState
var inline InlineCommand

// RegisterCommand adds a command to the bot.
func RegisterCommand(cmd Command) {
	commands = append(commands, NewCommandState(cmd))
}

// SetInline sets the Inline Query handler.
func SetInline(handler InlineCommand) {
	inline = handler
}

// SimpleCommand generates a command regex and matches it against a message.
//
// The trigger is the command without the slash,
// and the message is the text to check it against.
func SimpleCommand(trigger, message string) bool {
	// regex to match command, any arguments, and optionally bot name
	return regexp.MustCompile("^/(" + trigger + ")(@\\w+)?( .+)?$").MatchString(message)
}

// SimpleArgCommand generates a command regex and matches it against a message,
// requiring a certain number of parameters.
//
// The trigger is the command without the slash, args is number of arguments,
// and the message is the text to check it against.
func SimpleArgCommand(trigger string, args int, message string) bool {
	// regex to match command, any arguments, and optionally bot name
	matches := regexp.MustCompile("^/(" + trigger + ")(@\\w+)?( .+)?$").FindStringSubmatch(message)
	// if we don't have enough regex matches for all those items, return false
	if len(matches) < 4 {
		return false
	}
	// get the number of arguments (space seperated)
	msgArgs := len(strings.Split(strings.Trim(matches[3], " "), " "))
	// return if the number of args we got matches the number expected
	return args == msgArgs
}

// commandRouter is run for every update, and runs the correct commands.
func (f *Finch) commandRouter(update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)

			if ok && sentryEnabled {
				raven.CaptureError(err, nil)
			} else {
				log.Println(r)
			}
		}
	}()

	// if we've gotten an inline query, handle it
	if update.InlineQuery != nil {
		// if we do not have a handler, return
		if f.Inline == nil {
			log.Println("Got inline query, but no handler is set!")

			return
		}

		// execute inline handler function
		if err := f.Inline.Execute(f, *update.InlineQuery); err != nil {
			// no way to show inline error to user, so log it
			if sentryEnabled {
				raven.CaptureError(err, nil)
			}
			log.Printf("Error processing inline query:\n%+v\n", err)
		}

		return
	}

	// check if we have a callback query
	if update.CallbackQuery != nil {
		for _, command := range f.Commands {
			// check if the command is waiting for input
			if command.IsWaiting(update.CallbackQuery.From.ID) {
				if err := command.Command.ExecuteCallback(*update.CallbackQuery); err != nil {
					f.commandError(command.Command.Help().Name, *update.CallbackQuery.Message, err)
				}
			}
		}
	}

	// nothing past here can handle this!
	if update.Message == nil {
		return
	}

	// loop to check for any high priority commands
	for _, command := range f.Commands {
		// if it isn't a high priority command, ignore it
		if !command.Command.IsHighPriority(*update.Message) {
			continue
		}

		// if we shouldn't execute this command, ignore it
		if !command.Command.ShouldExecute(*update.Message) {
			continue
		}

		// execute the command
		if err := command.Command.Execute(*update.Message); err != nil {
			// some kind of error happened, send a message to sender
			f.commandError(command.Command.Help().Name, *update.Message, err)
		}
	}

	// now we can run all others
	for _, command := range f.Commands {
		// check if we're waiting for some text
		if command.IsWaiting(update.Message.From.ID) {
			// execute the waiting command
			if err := command.Command.ExecuteWaiting(*update.Message); err != nil {
				// some kind of error happened, send a message to sender
				f.commandError(command.Command.Help().Name, *update.Message, err)
			}

			// command has already dealt with this, contine to next
			continue
		}

		// we already did high priority commands, so skip now
		if command.Command.IsHighPriority(*update.Message) {
			continue
		}

		// check if we should execute this command
		if command.Command.ShouldExecute(*update.Message) {
			// execute the command
			if err := command.Command.Execute(*update.Message); err != nil {
				// some kind of error happened, send a message to sender
				f.commandError(command.Command.Help().Name, *update.Message, err)
			}
		}
	}
}

// called to init all commands
func (f *Finch) commandInit() {
	// for each command
	for _, command := range f.Commands {
		// run the command init function
		err := command.Command.Init(command, f)
		if err != nil {
			// it failed, show the error
			log.Printf("Error starting command %s: %s\n", command.Command.Help().Name, err.Error())
			if sentryEnabled {
				raven.CaptureError(err, map[string]string{"command": command.Command.Help().Name})
			}
		} else {
			// command started successfully
			log.Printf("Started command %s!", command.Command.Help().Name)
		}
	}
}

// handle some kind of error
func (f *Finch) commandError(commandName string, message tgbotapi.Message, err error) {
	var msg tgbotapi.MessageConfig

	// check if in Debug mode
	if f.API.Debug {
		// we're debugging, safe to send the actual error message
		msg = tgbotapi.NewMessage(message.Chat.ID, err.Error())
	} else {
		// production mode, just show a generic error message
		msg = tgbotapi.NewMessage(message.Chat.ID, "An error occured processing a command!")
		// log the error
		log.Println("Error processing command: " + err.Error())
	}

	msg.ReplyToMessageID = message.MessageID

	if sentryEnabled {
		raven.CaptureError(err, map[string]string{"command": commandName})
	}

	// send the error message
	_, err = f.API.Send(msg)
	if err != nil {
		log.Printf("An error happened processing an error!\n%s\n", err.Error())
		if sentryEnabled {
			raven.CaptureError(err, nil)
		}
	}
}
