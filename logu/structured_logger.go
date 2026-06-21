package logu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/jptrs93/goutil/timeu"
)

type StructuredLogHandler struct {
	Writer              io.Writer
	Level               slog.Level
	RootFieldsWhitelist []string
	attrs               []slog.Attr
	groups              []string
}

func NewStructuredLogger(w io.Writer, level slog.Level, rootFieldsWhitelist []string) *slog.Logger {
	return slog.New(NewStructuredLogHandler(w, level, rootFieldsWhitelist))
}

func NewStructuredLogHandler(w io.Writer, level slog.Level, rootFieldsWhitelist []string) slog.Handler {
	whitelist := make([]string, len(rootFieldsWhitelist))
	copy(whitelist, rootFieldsWhitelist)
	return &StructuredLogHandler{
		Writer:              w,
		Level:               level,
		RootFieldsWhitelist: whitelist,
	}
}

func (h *StructuredLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.Level
}

func (h *StructuredLogHandler) Handle(ctx context.Context, r slog.Record) error {
	rootFields := make([]structuredRootField, 0, len(h.RootFieldsWhitelist))
	rootIndexes := make(map[string]int, len(h.RootFieldsWhitelist))
	attrPairs := make([]string, 0, len(h.attrs)+r.NumAttrs())

	if lc := GetLogContext(ctx); lc != nil {
		for _, item := range lc.Items {
			value := any(true)
			if item.Value != nil {
				value = *item.Value
			}
			h.appendStructuredField(&rootFields, rootIndexes, &attrPairs, item.Name, value)
		}
	}

	prefix := strings.Join(h.groups, ".")
	for _, attr := range h.attrs {
		h.appendStructuredAttr(&rootFields, rootIndexes, &attrPairs, prefix, attr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		h.appendStructuredAttr(&rootFields, rootIndexes, &attrPairs, prefix, attr)
		return true
	})

	var b bytes.Buffer
	b.WriteByte('{')
	writeStructuredJSONField(&b, "_timestamp", r.Time.UTC().Format(timeu.RFC3339Milli))
	b.WriteByte(',')
	writeStructuredJSONField(&b, "level", structuredLevel(r.Level))
	b.WriteByte(',')
	writeStructuredJSONField(&b, "message", r.Message)
	for _, field := range rootFields {
		b.WriteByte(',')
		writeStructuredJSONField(&b, field.key, field.value)
	}
	if len(attrPairs) > 0 {
		b.WriteByte(',')
		writeStructuredJSONField(&b, "attrs", strings.Join(attrPairs, ", "))
	}
	b.WriteString("}\n")

	_, err := h.Writer.Write(b.Bytes())
	return err
}

func (h *StructuredLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)

	newGroups := make([]string, len(h.groups))
	copy(newGroups, h.groups)

	whitelist := make([]string, len(h.RootFieldsWhitelist))
	copy(whitelist, h.RootFieldsWhitelist)

	return &StructuredLogHandler{
		Writer:              h.Writer,
		Level:               h.Level,
		RootFieldsWhitelist: whitelist,
		attrs:               newAttrs,
		groups:              newGroups,
	}
}

func (h *StructuredLogHandler) WithGroup(name string) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs))
	copy(newAttrs, h.attrs)

	newGroups := make([]string, 0, len(h.groups)+1)
	newGroups = append(newGroups, h.groups...)
	newGroups = append(newGroups, name)

	whitelist := make([]string, len(h.RootFieldsWhitelist))
	copy(whitelist, h.RootFieldsWhitelist)

	return &StructuredLogHandler{
		Writer:              h.Writer,
		Level:               h.Level,
		RootFieldsWhitelist: whitelist,
		attrs:               newAttrs,
		groups:              newGroups,
	}
}

type structuredRootField struct {
	key   string
	value any
}

func (h *StructuredLogHandler) appendStructuredAttr(rootFields *[]structuredRootField, rootIndexes map[string]int, attrPairs *[]string, prefix string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		groupPrefix := joinAttrKey(prefix, attr.Key)
		for _, nested := range attr.Value.Group() {
			h.appendStructuredAttr(rootFields, rootIndexes, attrPairs, groupPrefix, nested)
		}
		return
	}

	key := joinAttrKey(prefix, attr.Key)
	if key == "" {
		return
	}
	h.appendStructuredField(rootFields, rootIndexes, attrPairs, key, attr.Value.Any())
}

func (h *StructuredLogHandler) appendStructuredField(rootFields *[]structuredRootField, rootIndexes map[string]int, attrPairs *[]string, key string, value any) {
	if key == "" {
		return
	}
	if h.isStructuredRootField(key) {
		if i, ok := rootIndexes[key]; ok {
			(*rootFields)[i].value = value
			return
		}
		rootIndexes[key] = len(*rootFields)
		*rootFields = append(*rootFields, structuredRootField{key: key, value: value})
		return
	}
	*attrPairs = append(*attrPairs, fmt.Sprintf("%s = %v", key, value))
}

func (h *StructuredLogHandler) isStructuredRootField(key string) bool {
	for _, allowed := range h.RootFieldsWhitelist {
		if key == allowed {
			return true
		}
	}
	return false
}

func writeStructuredJSONField(b *bytes.Buffer, key string, value any) {
	writeStructuredJSONValue(b, key)
	b.WriteByte(':')
	writeStructuredJSONValue(b, value)
}

func writeStructuredJSONValue(b *bytes.Buffer, value any) {
	encoded, err := json.Marshal(value)
	if err != nil {
		encoded, _ = json.Marshal(fmt.Sprint(value))
	}
	b.Write(encoded)
}

func structuredLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO"
	case slog.LevelWarn:
		return "WARN"
	case slog.LevelError:
		return "ERROR"
	default:
		return level.String()
	}
}
