package main

import (
	"fmt"
	"ntun/internal/app"
	"os"
	"strings"
)

func main() {
	var s strings.Builder
	fmt.Fprintf(&s, "PROGRAM_NAME=%s\n", app.Name)
	fmt.Fprintf(&s, "PROGRAM_VERSION=%s\n", app.Version)
	os.WriteFile("./builds/INFO.env", []byte(s.String()), os.ModePerm)
}
