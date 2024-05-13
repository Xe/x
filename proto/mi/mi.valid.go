package mi

import (
	"errors"
)

var (
	ErrNoMemberName         = errors.New("mi: no member name defined")
	ErrNoSuchMemberInSystem = errors.New("mi: no such member in system")
	ErrNoSwitchID           = errors.New("mi: no switch ID defined")
)

func (sr *SwitchReq) Valid() error {
	switch sr.GetMemberName() {
	case "":
		return ErrNoMemberName
	case "Cadey", "Nicole", "Jessie", "Sephie", "Ashe", "Mai":
		return nil
	default:
		return ErrNoSuchMemberInSystem
	}
}

func (gsr *GetSwitchReq) Valid() error {
	if gsr.GetId() == "" {
		return ErrNoSwitchID
	}

	return nil
}
