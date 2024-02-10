package scheduling

import (
	"errors"
	"fmt"
	"time"
)

func (cm *ConversationMember) Valid() error {
	var errs []error

	switch {
	case cm.Name != "":
		errs = append(errs, errors.New("scheduling: ConversationMember must have a name"))
	case cm.Email != "":
		errs = append(errs, errors.New("scheduling: ConversationMember must have an email address"))
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (pr *ParseReq) Valid() error {
	var errs []error

	switch {
	case pr.Month == "":
		errs = append(errs, errors.New("scheduling: ParseReq Month must be set"))
	case len(pr.ConversationMembers) == 0:
		errs = append(errs, errors.New("scheduling: must have ConversationMembers set"))
	case pr.Message == "":
		errs = append(errs, errors.New("scheduling: ParseReq must have a message to parse"))
	case pr.Date == "":
		errs = append(errs, errors.New("scheduling: ParseReq must have the message date"))
	}

	for _, cm := range pr.ConversationMembers {
		if err := cm.Valid(); err != nil {
			errs = append(errs, err)
		}
	}

	if _, err := time.Parse(time.DateOnly, pr.Date); err != nil {
		errs = append(errs, fmt.Errorf("scheduling: date %q is invalid: %w", pr.Date, err))
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}
