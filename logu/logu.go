package logu

import (
	"context"
	"fmt"
	"github.com/jptrs93/goutil/timeu"
	"io"
	"log/slog"
	"strings"
)

const LogContextKey = "_logContext"

func ExtendLogContext(ctx context.Context, key string, value any) context.Context {
	if existingLogContext, ok := ctx.Value(LogContextKey).(map[string]any); ok {
		existingLogContext[key] = value
		return context.WithValue(ctx, LogContextKey, existingLogContext)
	}
	logContext := map[string]any{key: value}
	return context.WithValue(ctx, LogContextKey, logContext)
}

func GetLogContext(ctx context.Context) map[string]any {
	if m, ok := ctx.Value(LogContextKey).(map[string]any); ok {
		return m
	}
	return nil
}

type PlainLogHandler struct {
	Writer io.Writer
	Level  slog.Level
}

func (h *PlainLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.Level
}

func (h *PlainLogHandler) Handle(ctx context.Context, r slog.Record) error {
	logContext := buildStrContext(GetLogContext(ctx))

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
	var stacktrace string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "stacktrace" {
			stacktrace = a.Value.String()
			return false
		}
		return true
	})
	msg := fmt.Sprintf("%s %s%s %s\n", r.Time.UTC().Format(timeu.RFC3339Milli), levelStr, logContext, r.Message)

	_, err := fmt.Fprint(h.Writer, msg)
	if err != nil {
		return err
	}
	if stacktrace != "" {
		_, err = fmt.Fprint(h.Writer, stacktrace)
	}
	return err
}

func (h *PlainLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PlainLogHandler) WithGroup(name string) slog.Handler {
	return h
}

func buildStrContext(logCtx map[string]any) string {
	if len(logCtx) == 0 {
		return ""
	}
	vals := make([]string, 0, len(logCtx))
	for k, v := range logCtx {
		vals = append(vals, fmt.Sprintf("%v=%v", k, v))
	}
	return " [" + strings.Join(vals, ", ") + "]"
}
