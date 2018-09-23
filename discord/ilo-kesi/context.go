package main

import (
	"errors"
	"strings"
)

const (
	actionFront = "lawa,insa"
)

var (
	ErrUnknownAction = errors.New("ijo-kesi: unknown action")
)

type Request struct {
	Address []*part
	Action  string
	Subject *string // if null, user is asking for the info
	Punct   string
}

func parseRequest(inp Sentence) (*Request, error) {
	var result Request

	for _, part := range inp {
		switch part.Part {
		case partAddress:
			result.Address = part.Parts
		case partSubject:
			if len(part.Tokens) == 1 && part.Tokens[0] != "seme" {
				sub := strings.Join(part.Tokens, ",")
				result.Subject = &sub
			}
		case partObjectMarker:
			act := strings.Join(part.Tokens, ",")

			switch act {
			case actionFront:
			default:
				return nil, ErrUnknownAction
			}

			result.Action = act
		case partPunctuation:
			result.Punct = part.Tokens[0]
		}
	}

	return &result, nil
}
