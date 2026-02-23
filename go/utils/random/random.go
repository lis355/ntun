package random

import (
	"crypto/rand"
	"strconv"
)

func RandomHash() (s string) {
	bytes := make([]byte, 16)
	rand.Read(bytes)

	for _, c := range bytes {
		s += strconv.FormatInt(int64(c), 16)
	}

	return
}
