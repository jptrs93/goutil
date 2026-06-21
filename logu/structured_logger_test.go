package logu

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestStructuredLogHandlerWritesRootFieldsAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewStructuredLogHandler(&buf, slog.LevelDebug, []string{"request_id", "http.service", "http.status"}))
	ctx := ExtendLogContext(context.Background(), "request_id", "req-123")
	ctx = ExtendLogContext(ctx, "tenant", "acme")

	logger.With("service", "api", "debug", true).WithGroup("http").InfoContext(ctx, "served", slog.Int("status", 200), slog.String("method", "GET"))

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if got["level"] != "INFO" || got["message"] != "served" || got["request_id"] != "req-123" || got["http.service"] != "api" || got["http.status"] != float64(200) {
		t.Fatalf("root fields = %#v", got)
	}
	attrs, ok := got["attrs"].(string)
	if !ok {
		t.Fatalf("attrs = %#v", got["attrs"])
	}
	for _, want := range []string{"tenant = acme", "http.debug = true", "http.method = GET"} {
		if !strings.Contains(attrs, want) {
			t.Fatalf("attrs %q does not contain %q", attrs, want)
		}
	}
	if _, ok := got["_timestamp"].(string); !ok {
		t.Fatalf("_timestamp = %#v", got["_timestamp"])
	}
}

func TestStructuredLogHandlerOmitsEmptyAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewStructuredLogHandler(&buf, slog.LevelInfo, nil))
	logger.Info("hello")

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log line: %v", err)
	}
	if _, ok := got["attrs"]; ok {
		t.Fatalf("attrs should be omitted: %#v", got)
	}
}
