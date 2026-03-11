package app

import (
	"github.com/joho/godotenv"
)

func InitEnv() {
	godotenv.Overload(".env", ".env.local")
}
