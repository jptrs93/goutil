package logu

import (
	"context"
	"fmt"
	"strings"

	"github.com/jptrs93/goutil/ptru"
)

// LogContextKey is the context key used to store a *LogContext in context.Context.
//
// This pattern provides a simple way to inject contextual logging params that are
// scoped to the current context and its descendants. When a child context is
// created with ExtendLogContext, it gets a copied item slice plus the new item,
// so parent and sibling contexts keep their own values.
//
// Example (user ID propagation):
//
//	base := context.Background()
//	requestCtx := ExtendLogContext(base, "request_id", "req-123")
//	userCtx := ExtendLogContext(requestCtx, "user_id", 42)
//
//	// Example logs emmitted with requestCtx include: [request_id=req-123]
//	// Example logs emmitted with userCtx include: [request_id=req-123, user_id=42]
//
// This lets you add request/user/job metadata once and automatically include it
// in all logs that flow through that derived context tree. Exact formatting in
// output depends on the configured logging handler.
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
	item := LogContextItem{Name: name}
	if value != nil {
		item.Value = ptru.To(fmt.Sprintf("%v", value))
	}

	var logContext *LogContext
	existing, ok := ctx.Value(LogContextKey).(*LogContext)
	if !ok {
		logContext = &LogContext{Items: []LogContextItem{item}}
		logContext.UpdateCachedStr()
	} else {
		n := len(existing.Items)
		logContext = &LogContext{
			// ensure capacity=n to force an allocation of new underlying array
			Items: append(existing.Items[:n:n], item),
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
