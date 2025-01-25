package config

import (
	"errors"
	"fmt"
)

type Rule string

const (
	RuleUnknown   = ""
	RuleAllow     = "ALLOW"
	RuleDeny      = "DENY"
	RuleChallenge = "CHALLENGE"
)

type Bot struct {
	Name           string  `json:"name"`
	UserAgentRegex *string `json:"user_agent_regex"`
	PathRegex      *string `json:"path_regex"`
	Action         Rule    `json:"action"`
}

var (
	ErrBotMustHaveName                = errors.New("config.Bot: must set name")
	ErrBotMustHaveUserAgentPathOrBoth = errors.New("config.Bot: must set either user_agent_regex, path_regex, or both")
	ErrUnknownAction                  = errors.New("config.Bot: unknown action")
)

func (b Bot) Valid() error {
	var err error

	if b.Name == "" {
		err = errors.Join(err, ErrBotMustHaveName)
	}

	if b.UserAgentRegex == nil && b.PathRegex == nil {
		err = errors.Join(err, ErrBotMustHaveUserAgentPathOrBoth)
	}

	switch b.Action {
	case RuleAllow, RuleChallenge, RuleDeny:
		// okay
	default:
		err = errors.Join(err, fmt.Errorf("%w: %q", ErrUnknownAction, b.Action))
	}

	if err != nil {
		return fmt.Errorf("config: bot entry for %q is not valid: %w", b.Name, err)
	}

	return nil
}

type Config struct {
	Bots []Bot `json:"bots"`
}
