package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Xe/x/web/tokiponatokens"
)

var (
	ErrNotAddressed = errors.New("ilo-kesi: ilo kesi was not addressed")
)

func (i ilo) parse(authorID, inp string) (*reply, error) {
	var result reply
	buf := bytes.NewBuffer(nil)

	parts, err := tokiponatokens.Tokenize(i.cfg.TokiPonaTokenizerAPIURL, inp)
	if err != nil {
		return nil, err
	}

	for _, sent := range parts {
		req, err := parseRequest(authorID, sent)
		if err != nil {
			return nil, err
		}

		if len(req.Address) != 2 {
			return nil, ErrNotAddressed
		}

		if req.Address[0] != "ilo" {
			return nil, ErrNotAddressed
		}

		if req.Address[1] != i.cfg.IloNimi {
			return nil, ErrNotAddressed
		}

		switch req.Action {
		case actionFront:
			if req.Subject == actionWhat {
				st, err := i.sw.Status(context.Background())
				if err != nil {
					return nil, err
				}

				qual := TimeToQualifier(st.StartedAt)
				fmt.Fprintf(buf, "%s la jan %s li lawa insa.\n", qual, withinToToki[st.Front])
				goto ok
			}

			if !i.janLawaAnuSeme(authorID) {
				return nil, ErrJanLawaAla
			}

			front := tokiToWithin[req.Subject]

			_, err := i.sw.Switch(context.Background(), front)
			if err != nil {
				return nil, err
			}

			fmt.Fprintf(buf, "tenpo ni la jan %s li lawa insa.\n", req.Subject)
			goto ok
		case actionWhat:
			switch req.Subject {
			case "tenpo ni":
				fmt.Fprintf(buf, "ni li tenpo %s\n", time.Now().Format(time.Kitchen))
				goto ok
			}
		}

		switch req.Subject {
		case "sitelen pakala":
			fmt.Fprintf(buf, "%s\n", i.chain.Generate(20))
			goto ok
		}

		return nil, ErrUnknownAction
	}

ok:
	result.msg = buf.String()
	buf.Reset()

	return &result, nil
}
