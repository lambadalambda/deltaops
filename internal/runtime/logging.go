package runtime

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
)

type Logger interface {
	Log(context.Context, LogEvent)
}

type LogEvent struct {
	Name   string            `json:"event"`
	Fields map[string]string `json:"fields,omitempty"`
}

type JSONLogger struct {
	mu sync.Mutex
	w  io.Writer
}

type discardLogger struct{}

func NewJSONLogger(w io.Writer) *JSONLogger {
	return &JSONLogger{w: w}
}

func (l *JSONLogger) Log(_ context.Context, event LogEvent) {
	if l == nil || l.w == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = json.NewEncoder(l.w).Encode(redactEvent(event))
}

func (discardLogger) Log(context.Context, LogEvent) {}

func redactEvent(event LogEvent) LogEvent {
	redacted := LogEvent{Name: event.Name}
	if len(event.Fields) == 0 {
		return redacted
	}
	redacted.Fields = make(map[string]string, len(event.Fields))
	for key, value := range event.Fields {
		if sensitiveField(key) {
			redacted.Fields[key] = "[redacted]"
			continue
		}
		redacted.Fields[key] = redactValue(value)
	}
	return redacted
}

func sensitiveField(key string) bool {
	key = strings.ToLower(key)
	for _, marker := range []string{"secret", "password", "token", "setup_code", "dcaccount", "message", "body", "text", "error", "cause"} {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func redactValue(value string) string {
	lower := strings.ToLower(value)
	for _, marker := range []string{"dcaccount:", "setup_code", "token=", "password", "secret", "message", "body="} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	return value
}

func safeErrorText(err error) string {
	if err == nil {
		return ""
	}
	if redacted := redactValue(err.Error()); redacted != err.Error() {
		return "[redacted]"
	}
	return err.Error()
}
