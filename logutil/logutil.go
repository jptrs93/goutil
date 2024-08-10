package logutil

import (
	"context"
	"fmt"
	"github.com/jptrs93/goutil/timeutil"
	"io"
	"log/slog"
)

const LogContextKey = "_logContext"

type PlainLogHandler struct {
	Writer io.Writer
	Level  slog.Level
}

func (h *PlainLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.Level
}

func (h *PlainLogHandler) Handle(ctx context.Context, r slog.Record) error {
	logContext := ""
	if v, ok := ctx.Value(LogContextKey).(string); ok {
		logContext = " " + v
	}
	// make the level strings all the same length
	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO "
	case slog.LevelWarn:
		levelStr = "WARN "
	case slog.LevelError:
		levelStr = "ERROR"
	}
	msg := fmt.Sprintf("%s %s%s %s\n", r.Time.UTC().Format(timeutil.RFC3339Milli), levelStr, logContext, r.Message)
	_, err := fmt.Fprint(h.Writer, msg)
	return err
}

func (h *PlainLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PlainLogHandler) WithGroup(name string) slog.Handler {
	return h
}
