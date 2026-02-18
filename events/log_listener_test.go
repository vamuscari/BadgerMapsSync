package events

import (
	"badgermaps/app/state"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestLogListener_FileLogging(t *testing.T) {
	// Create a temporary file for logging
	tmpfile, err := ioutil.TempFile("", "test.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	// Create a new log listener with the temp file path
	s := state.NewState()
	s.Debug = true
	listener, err := NewLogListener(s, tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create log listener: %v", err)
	}
	defer listener.Close()

	// Dispatch a log event
	event := Infof("test_source", "This is a test message")
	listener.Handle(event)

	// Read the content of the log file
	content, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Verify the content
	if !strings.Contains(string(content), "This is a test message") {
		t.Errorf("Log file does not contain the expected message. Got: %s", string(content))
	}
	if !strings.Contains(string(content), "[test_source]") {
		t.Errorf("Log file does not contain the expected source. Got: %s", string(content))
	}
}

func TestLogListener_FileLogging_NonDebugWritesOnlyErrors(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "test-errors-only.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	s := state.NewState()
	listener, err := NewLogListener(s, tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to create log listener: %v", err)
	}
	defer listener.Close()

	listener.Handle(Infof("test_source", "info message"))
	listener.Handle(Warningf("test_source", "warn message"))
	listener.Handle(Errorf("test_source", "error message"))
	listener.Handle(Debugf("test_source", "debug message"))

	content, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	logText := string(content)

	if !strings.Contains(logText, "error message") {
		t.Errorf("Expected file log to contain error message. Got: %s", logText)
	}
	if strings.Contains(logText, "info message") {
		t.Errorf("Expected file log to exclude info messages. Got: %s", logText)
	}
	if strings.Contains(logText, "warn message") {
		t.Errorf("Expected file log to exclude warning messages. Got: %s", logText)
	}
	if strings.Contains(logText, "debug message") {
		t.Errorf("Expected file log to exclude debug messages. Got: %s", logText)
	}
}
