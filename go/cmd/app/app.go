package app

import (
	"os"

	"github.com/joho/godotenv"
)

var IsDevelopment bool

func Initialize() {
	godotenv.Overload(".env", ".env.local")

	IsDevelopment = os.Getenv("DEVELOPMENT") == "true"
}
