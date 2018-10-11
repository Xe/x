package main

import (
	"errors"
	"strings"
	"time"

	"github.com/Xe/x/web/tokiponatokens"
)

const (
	actionFront   = "lawa,insa"
	actionMarkov  = "sitelen"
	actionWhat    = "seme"
	actionBotInfo = "sina"
)

var (
	ErrUnknownAction = errors.New("ilo-kesi: mi sona ala")
)

type Request struct {
	Address []string
	Action  string
	Subject string
	Punct   string
	Author  string

	Input tokiponatokens.Sentence
}

func parseRequest(authorID string, inp tokiponatokens.Sentence) (*Request, error) {
	var result Request
	result.Author = authorID
	result.Input = inp

	for _, part := range inp {
		switch part.Type {
		case tokiponatokens.PartAddress:
			for i, pt := range part.Parts {
				if i == 0 {
					result.Address = append(result.Address, pt.Tokens[0])
					continue
				}

				result.Address = append(result.Address, strings.Title(strings.Join(pt.Tokens, "")))
			}
		case tokiponatokens.PartSubject:
			if len(part.Tokens) == 0 {
				sub := strings.Title(strings.Join(part.Parts[1].Tokens, ""))
				result.Subject = sub
			} else {
				sub := strings.Join(part.Tokens, " ")
				result.Subject = sub
			}
		case tokiponatokens.PartObjectMarker:
			act := strings.Join(part.Tokens, ",")

			switch act {
			case actionFront, actionWhat:
			case actionMarkov:
			default:
				return nil, ErrUnknownAction
			}

			result.Action = act
		case tokiponatokens.PartPunctuation:
			result.Punct = part.Tokens[0]
		}
	}

	return &result, nil
}

func TimeToQualifier(t time.Time) string {
	const (
		nowRange = 15 * time.Minute
	)

	s := time.Since(t)
	if s > 0 {
		return "tenpo kama"
	}

	if s < nowRange {
		return "tenpo ni"
	}

	return "tenpo pini"
}
