package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

func Init() {
	PrintLogo()
	InitEnv()
}

func InitEnv() {
	godotenv.Overload(".env", ".env.local")
}

func PrintLogo() {
	// https://patorjk.com/software/taag
	// BlurVision ASCII
	fmt.Print("\033[32m" + strings.TrimSpace(`
░▒▓███████▓▒░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓███████▓▒░  
░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░ ░▒▓█▓▒░    ░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░
`+"\033[0m"),
	)
}

func CacheDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	cacheDir := filepath.Join(cwd, ".cache")

	return cacheDir, nil
}

func WriteCacheFile(relativeFilePath string, buf []byte) error {
	cacheDir, err := CacheDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(cacheDir, relativeFilePath)

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filePath, buf, 0644); err != nil {
		return err
	}

	return nil
}

func ReadCacheFile(relativeFilePath string) ([]byte, error) {
	cacheDir, err := CacheDir()
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(cacheDir, relativeFilePath)

	buf, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return buf, err
}
