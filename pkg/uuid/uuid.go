package uuid

import (
	"crypto/rand"
	"fmt"
	"io"
)

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
