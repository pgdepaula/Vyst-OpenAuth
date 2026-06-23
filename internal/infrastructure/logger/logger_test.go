package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{"Default", Config{}},
		{"DebugJSON", Config{Level: "debug", Format: "json"}},
		{"InfoText", Config{Level: "info", Format: "text"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(tt.config)
			if l == nil {
				t.Error("New() returned nil")
			}
		})
	}
}

func TestGlobal(t *testing.T) {
	l := Global()
	if l == nil {
		t.Error("Global() returned nil")
	}

	newLogger := New(Config{Level: "debug"})
	SetGlobal(newLogger)

	if Global() != newLogger {
		t.Error("SetGlobal() failed to update global logger")
	}
}

func TestLoggerOutput(t *testing.T) {
	// Capture output
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	l := &Wrapper{Logger: slog.New(handler)}

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		l.Info("test message", "key", "value")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["msg"] != "test message" {
			t.Errorf("Expected msg 'test message', got '%v'", logEntry["msg"])
		}
		if logEntry["level"] != "INFO" {
			t.Errorf("Expected level 'INFO', got '%v'", logEntry["level"])
		}
		if logEntry["key"] != "value" {
			t.Errorf("Expected key 'value', got '%v'", logEntry["key"])
		}
	})

	t.Run("Error with Context", func(t *testing.T) {
		buf.Reset()
		ctx := context.Background()
		l.ErrorContext(ctx, "error message", "error", "something went wrong")

		var logEntry map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
			t.Fatalf("Failed to parse log output: %v", err)
		}

		if logEntry["msg"] != "error message" {
			t.Errorf("Expected msg 'error message', got '%v'", logEntry["msg"])
		}
		if logEntry["level"] != "ERROR" {
			t.Errorf("Expected level 'ERROR', got '%v'", logEntry["level"])
		}
	})
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// parseLevel is unexported, so we test it indirectly via New or we export it.
			// Since we can't easily access unexported functions from a separate test package
			// (unless we are in the same package), and we are in `package logger`, we can access it.
			got := parseLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
