package app

import "os"

func IsDevelopment() bool {
	return os.Getenv("DEVELOPMENT") == "true"
}
