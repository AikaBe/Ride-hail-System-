package uuid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
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

func NewUUID() (string, error) {
	u := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, u)
	if err != nil {
		return "", err
	}

	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u[0:4],
		u[4:6],
		u[6:8],
		u[8:10],
		u[10:16],
	), nil
}
