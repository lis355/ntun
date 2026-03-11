package utils

import (
	"crypto/rand"
)

func RandBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}
