package log

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"
)

// for prevent long computations use
// if slog.Default().Enabled(context.Background(), slog.LevelDebug) { ... }

type LogHandler struct {
	slog.Handler
	l *log.Logger
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

	return nil
}

func Initialize() {
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

	slog.SetDefault(
		slog.New(
			&LogHandler{
				Handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
					AddSource: true,
					Level:     level,
				}),
				l: log.New(os.Stdout, "", 0),
			}))
}
