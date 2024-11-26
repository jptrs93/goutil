package logu

import (
	"context"
	"fmt"
	"github.com/jptrs93/goutil/timeu"
	"io"
	"log/slog"
)

const LogContextKey = "_logContext"

func ExtendLogContext(ctx context.Context, additionalContext string) context.Context {
	if existingLogContext, ok := ctx.Value(LogContextKey).(string); ok {
		if len(existingLogContext) > 0 {
			return context.WithValue(ctx, LogContextKey, existingLogContext+" "+additionalContext)
		} else {
			return context.WithValue(ctx, LogContextKey, additionalContext)
		}
	}
	return context.WithValue(ctx, LogContextKey, additionalContext)
}

func GetLogContext(ctx context.Context) string {
	if existingLogContext, ok := ctx.Value(LogContextKey).(string); ok {
		return existingLogContext
	}
	return ""
}

type PlainLogHandler struct {
	Writer io.Writer
	Level  slog.Level
}

func (h *PlainLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.Level
}

func (h *PlainLogHandler) Handle(ctx context.Context, r slog.Record) error {
	logContext := ""
	if ctx != nil {
		if v, ok := ctx.Value(LogContextKey).(string); ok {
			logContext = " " + v
		}
	}
	// make the level strings all the same length
	var levelStr string
	switch r.Level {
	case slog.LevelDebug:
		levelStr = "DEBUG"
	case slog.LevelInfo:
		levelStr = "INFO"
	case slog.LevelWarn:
		levelStr = "WARN"
	case slog.LevelError:
		levelStr = "ERROR"
	}
	timestamp := r.Time.UTC().Format(timeu.RFC3339Milli)
	msg := fmt.Sprintf("%s %s%s %s\n", timestamp, levelStr, logContext, r.Message)
	_, err := fmt.Fprint(h.Writer, msg)
	return err
}

func (h *PlainLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PlainLogHandler) WithGroup(name string) slog.Handler {
	return h
}
