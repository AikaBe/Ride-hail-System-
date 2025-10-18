package model

import (
	"errors"
	"regexp"
)

type UUID string

var uuidRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-5][a-fA-F0-9]{3}-[89abAB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

func (u UUID) Validate() error {
	if !uuidRegex.MatchString(string(u)) {
		return errors.New("invalid UUID format")
	}
	return nil
}

func (u UUID) IsZero() bool {
	return u == "" || u == "00000000-0000-0000-0000-000000000000"
}
