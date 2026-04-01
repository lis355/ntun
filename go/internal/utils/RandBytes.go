package utils

import (
	"crypto/rand"

	"github.com/google/uuid"
)

func RandBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func RandShortString() string {
	return uuid.New().String()[0:8]
}
