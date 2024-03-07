package scheduling

import (
	"errors"
	"fmt"
	"time"
)

func (cm *ConversationMember) Valid() error {
	errs := []error{}
	if cm.Email == "" {
		errs = append(errs, fmt.Errorf("email is required"))
	}
	if cm.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid ConversationMember: %w", errors.Join(errs...))
	}
	return nil
}

func (pr *ParseReq) Valid() error {
	errs := []error{}
	switch pr.Month {
	case "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December":
	default:
		errs = append(errs, fmt.Errorf("month is invalid: %q", pr.Month))
	}
	for _, cm := range pr.ConversationMembers {
		if err := cm.Valid(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(pr.Message) == 0 {
		errs = append(errs, fmt.Errorf("message is required"))
	}
	if _, err := time.Parse(time.DateOnly, pr.Date); err != nil {
		errs = append(errs, fmt.Errorf("date is invalid: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid ParseReq: %w", errors.Join(errs...))
	}
	return nil
}
