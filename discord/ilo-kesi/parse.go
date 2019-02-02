package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Xe/x/internal/mainsa"
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
				err = switchcounter.Validate(resp)
				if err != nil {
					return nil, err
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
			err = switchcounter.Validate(resp)
			if err != nil {
				return nil, err
			}

			fmt.Fprintf(buf, "tenpo ni la jan %s li lawa insa.\n", req.Subject)
			goto ok
		case actionWhat:
			switch req.Subject {
			case "tenpo ni":
				ni, err := mainsa.At(time.Now())
				if err != nil {
					return nil, err
				}

				fmt.Fprintf(buf, "ma insa la tenpo ni li tenpo pi %s\n", ni)
				goto ok
			case actionBotInfo:
				fmt.Fprintf(buf, "mi ilo Kesi. mi ilo e kama sona e pali pona mute. mi wile pona sina. lipu sona mi li sitelen https://github.com/Xe/x/tree/master/discord/ilo-kesi.\n")
				goto ok
			}
		case "":
			switch req.Subject {
			case "sina seme":
				fmt.Fprintf(buf, "mi ilo Kesi. mi ilo e kama sona e pali pona mute. mi wile pona sina. lipu sona mi li sitelen https://github.com/Xe/x/tree/master/discord/ilo-kesi.\n")
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
