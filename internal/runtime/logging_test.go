package runtime

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestJSONLoggerRedactsSecretsAndMessageBodies(t *testing.T) {
	var out bytes.Buffer
	logger := NewJSONLogger(&out)

	logger.Log(context.Background(), LogEvent{
		Name: "test_event",
		Fields: map[string]string{
			"dcaccount_url": "dcaccount:secret",
			"setup_code":    "123456",
			"message_text":  "raw operator message",
			"error":         "send failed setup_code=123456 token=abc body=raw message",
			"metric":        "disk.used_percent",
		},
	})

	logLine := out.String()
	for _, secret := range []string{"dcaccount:secret", "123456", "raw operator message", "token=abc"} {
		if strings.Contains(logLine, secret) {
			t.Fatalf("log line %q leaked secret %q", logLine, secret)
		}
	}
	if !strings.Contains(logLine, "[redacted]") {
		t.Fatalf("log line %q does not include redaction marker", logLine)
	}
	if !strings.Contains(logLine, "disk.used_percent") {
		t.Fatalf("log line %q does not keep safe metric field", logLine)
	}
}
