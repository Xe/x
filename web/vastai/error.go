package vastai

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Error struct {
	Success    bool   `json:"success"`
	ErrorKind  string `json:"error"`
	Msg        string `json:"msg"`
	StatusCode int    `json:"status_code"`
}

func (e Error) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.ErrorKind, e.StatusCode, e.Msg)
}

func NewError(resp *http.Response) error {
	var e Error

	dec := json.NewDecoder(resp.Body)
	defer resp.Body.Close()
	if err := dec.Decode(&e); err != nil {
		return fmt.Errorf("vastai: can't decode json while handling error: %w", err)
	}

	e.StatusCode = resp.StatusCode

	return e
}
