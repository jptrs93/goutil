package logu

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestJSONLogHandlerWritesFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewJSONLogHandler(&buf, slog.LevelDebug))
	ctx := AddKV(context.Background(), "request_id", "req-123")
	ctx = AddKV(ctx, "tenant", "acme")
	ctx = AddTag(ctx, "canary")

	logger.With("service", "api", "debug", true).WithGroup("http").InfoContext(ctx, "served", slog.Int("status", 200), slog.String("method", "GET"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got["level"] != "INFO" || got["msg"] != "served" {
		t.Fatalf("base fields = %#v", got)
	}
	if got["request_id"] != "req-123" || got["tenant"] != "acme" {
		t.Fatalf("context fields = %#v", got)
	}
	if got["service"] != "api" || got["debug"] != true {
		t.Fatalf("logger attr fields = %#v", got)
	}
	httpGroup, ok := got["http"].(map[string]any)
	if !ok || httpGroup["status"] != float64(200) || httpGroup["method"] != "GET" {
		t.Fatalf("http group = %#v", got["http"])
	}
	tags, ok := got["_tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "canary" {
		t.Fatalf("_tags = %#v", got["_tags"])
	}
	if _, ok := got["time"].(string); !ok {
		t.Fatalf("time = %#v", got["time"])
	}
}

func TestJSONLogHandlerOmitsEmptyTags(t *testing.T) {
	var buf bytes.Buffer
	logger := NewJSONLogger(&buf, slog.LevelInfo)
	logger.Info("hello")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if _, ok := got["_tags"]; ok {
		t.Fatalf("_tags should be omitted: %#v", got)
	}
	if len(got) != 3 {
		t.Fatalf("expected only time, level, msg: %#v", got)
	}
}
