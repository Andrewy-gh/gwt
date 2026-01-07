package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	// Save original output
	oldOut := Out
	defer func() { Out = oldOut }()

	// Create a buffer to capture output
	var buf bytes.Buffer
	Out = &buf

	// Test success message
	Success("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, symbolSuccess) {
		t.Errorf("Expected output to contain success symbol, got: %s", output)
	}
}

func TestError(t *testing.T) {
	// Save original output
	oldErr := Err
	defer func() { Err = oldErr }()

	// Create a buffer to capture output
	var buf bytes.Buffer
	Err = &buf

	// Test error message
	Error("error message")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected output to contain 'error message', got: %s", output)
	}
	if !strings.Contains(output, symbolError) {
		t.Errorf("Expected output to contain error symbol, got: %s", output)
	}
}

func TestQuietMode(t *testing.T) {
	// Save original state
	oldOut := Out
	oldQuiet := quietMode
	defer func() {
		Out = oldOut
		quietMode = oldQuiet
	}()

	// Enable quiet mode
	SetQuiet(true)

	// Create a buffer to capture output
	var buf bytes.Buffer
	Out = &buf

	// Test that Info is suppressed in quiet mode
	Info("info message")

	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output in quiet mode, got: %s", output)
	}

	// Error should still work in quiet mode
	buf.Reset()
	Err = &buf
	Error("error message")

	output = buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected error message even in quiet mode, got: %s", output)
	}
}

func TestVerboseMode(t *testing.T) {
	// Save original state
	oldOut := Out
	oldVerbose := verboseMode
	defer func() {
		Out = oldOut
		verboseMode = oldVerbose
	}()

	// Test verbose disabled
	SetVerbose(false)
	var buf bytes.Buffer
	Out = &buf

	Verbose("verbose message")

	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output when verbose disabled, got: %s", output)
	}

	// Test verbose enabled
	SetVerbose(true)
	buf.Reset()

	Verbose("verbose message")

	output = buf.String()
	if !strings.Contains(output, "verbose message") {
		t.Errorf("Expected verbose message when verbose enabled, got: %s", output)
	}
}
