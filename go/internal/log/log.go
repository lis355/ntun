package log

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

// for prevent long computations use
// if slog.Default().Enabled(context.Background(), slog.LevelDebug) { ... }

type LogHandler struct {
	slog.Handler
	l       *log.Logger
	logFile *os.File
}

func (logHandler *LogHandler) Handle(ctx context.Context, r slog.Record) error {
	var color byte
	switch r.Level {
	case slog.LevelDebug:
		color = 34
	case slog.LevelInfo:
		color = 32
	case slog.LevelWarn:
		color = 33
	case slog.LevelError:
		color = 31
	}

	attributesString := ""
	r.Attrs(func(a slog.Attr) bool {
		attributesString += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	logHandler.l.Printf("\033[%dm[%s %s]: %s%s\033[0m",
		color,
		r.Time.Format(time.RFC3339),
		r.Level.String(),
		r.Message,
		attributesString,
	)

	logHandler.logFile.WriteString(fmt.Sprintf("[%s %s]: %s%s\n",
		r.Time.Format(time.RFC3339),
		r.Level.String(),
		r.Message,
		attributesString,
	))

	return nil
}

func Init() {
	var level slog.Level
	envLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))

	switch envLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	logFile, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}

	slog.SetDefault(
		slog.New(
			&LogHandler{
				Handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
					AddSource: true,
					Level:     level,
				}),
				l:       log.New(os.Stdout, "", 0),
				logFile: logFile,
			},
		),
	)
}

var (
	names   map[string]map[string]int
	namesMu sync.Mutex
)

func init() {
	names = make(map[string]map[string]int)
}

func ObjName(v any) string {
	parts := strings.Split(reflect.TypeOf(v).String(), ".")
	name := parts[len(parts)-1]

	namesMu.Lock()

	_, ok := names[name]
	if !ok {
		names[name] = make(map[string]int)
	}

	pname, ok := names[name][fmt.Sprintf("%p", v)]
	if !ok {
		pname = len(names[name]) + 1
		names[name][fmt.Sprintf("%p", v)] = pname
	}

	namesMu.Unlock()

	return fmt.Sprintf("[%s:%d]", name, pname)
}
