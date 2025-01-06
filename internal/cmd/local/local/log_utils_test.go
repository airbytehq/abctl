package local

import (
	"strings"
	"testing"
)

var testLogs = strings.TrimSpace(`
nonjsonline
{"timestamp":1734723317023,"message":"Waiting for database to become available...","level":"WARN","logSource":"platform","caller":{"className":"io.airbyte.db.check.DatabaseAvailabilityCheck","methodName":"check","lineNumber":38,"threadName":"main"},"throwable":null}
`)

func TestJavaLogScanner(t *testing.T) {
	s := newLogScanner(strings.NewReader(testLogs))

	expectLogLine := func(level, msg string) {
		s.Scan()

		if s.line.Level != level {
			t.Errorf("expected level %q but got %q", level, s.line.Level)
		}
		if s.line.Message != msg {
			t.Errorf("expected msg %q but got %q", msg, s.line.Message)
		}
		if s.Err() != nil {
			t.Errorf("unexpected error %v", s.Err())
		}
	}

	expectLogLine("", "nonjsonline")
	expectLogLine("WARN", "Waiting for database to become available...")
}
