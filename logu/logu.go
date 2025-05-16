package logu

import (
	"context"
	"fmt"
	"github.com/jptrs93/goutil/ptru"
	"github.com/jptrs93/goutil/sliceu"
	"github.com/jptrs93/goutil/timeu"
	"io"
	"log/slog"
	"strings"
)

const LogContextKey = "_logContext"

type LogContext struct {
	Items     []LogContextItem
	CachedStr string
}

func (lc *LogContext) UpdateCachedStr() {
	if len(lc.Items) == 0 {
		lc.CachedStr = ""
		return
	}
	vals := make([]string, 0, len(lc.Items))
	for _, i := range lc.Items {
		if i.Value != nil {
			vals = append(vals, fmt.Sprintf("%v=%v", i.Name, *i.Value))
		} else {
			vals = append(vals, i.Name)
		}
	}
	lc.CachedStr = " [" + strings.Join(vals, ", ") + "]"
}

type LogContextItem struct {
	Name  string
	Value *string
}

func ExtendLogContext(ctx context.Context, name string, value any) context.Context {
	item := LogContextItem{
		Name: name,
	}
	if value != nil {
		item.Value = ptru.To(fmt.Sprintf("%v", value))
	}
	var logContext *LogContext
	existing, ok := ctx.Value(LogContextKey).(*LogContext)
	if !ok {
		logContext = &LogContext{
			Items: []LogContextItem{item},
		}
		logContext.UpdateCachedStr()
	} else {
		// ensure we put back a copy so that it does modify the parent ctx LogContext
		logContext = &LogContext{
			Items: append(sliceu.Copy(existing.Items), item),
		}
		logContext.UpdateCachedStr()
	}
	return context.WithValue(ctx, LogContextKey, logContext)
}

func GetLogContext(ctx context.Context) *LogContext {
	if m, ok := ctx.Value(LogContextKey).(*LogContext); ok {
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
	var logContext string
	if lc := GetLogContext(ctx); lc != nil {
		logContext = lc.CachedStr
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
