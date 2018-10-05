package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Xe/x/web/switchcounter"
	"github.com/Xe/x/web/tokiponatokens"
)

var (
	ErrNotTokiPona  = errors.New("toki ni li toki pona ala")
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
				req := i.sw.Status()
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return nil, err
				}
				if resp.StatusCode/100 != 2 {
					return nil, errors.New(resp.Status)
				}
				var st switchcounter.Status
				err = json.NewDecoder(resp.Body).Decode(&st)
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

			hreq := i.sw.Switch(front)
			resp, err := http.DefaultClient.Do(hreq)
			if err != nil {
				return nil, err
			}
			if resp.StatusCode/100 != 2 {
				return nil, errors.New(resp.Status)
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
