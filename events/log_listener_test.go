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
