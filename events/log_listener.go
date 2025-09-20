package events

import (
	"badgermaps/app/state"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// LogListener handles log events and prints them to the console or a file.
type LogListener struct {
	State  *state.State
	writer io.Writer
	file   *os.File
}

// NewLogListener creates a new LogListener.
func NewLogListener(state *state.State, logFilePath string) (*LogListener, error) {
	l := &LogListener{
		State:  state,
		writer: os.Stdout, // Default to stdout
	}

	if logFilePath != "" {
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		l.file = file
		l.writer = file
	}

	return l, nil
}

// Close closes the log file if it was opened.
func (l *LogListener) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// Handle processes the log event.
func (l *LogListener) Handle(e Event) {
	if l.State.Quiet {
		return
	}

	payload, ok := e.Payload.(LogPayload)
	if !ok {
		return // Not a log event we can handle
	}

	// Respect quiet and verbosity settings
	if payload.Level == LogLevelDebug && !l.State.Debug {
		return
	}

	// Prepare output parts
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := payload.Level.String()
	sourceStr := e.Source

	// Build the log string
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s ", timestamp))
	sb.WriteString(fmt.Sprintf("%-5s ", levelStr)) // Padded level
	sb.WriteString(fmt.Sprintf("[%s] ", sourceStr))
	sb.WriteString(payload.Message)

	// Append fields if they exist
	if len(payload.Fields) > 0 {
		sb.WriteString(" ")
		first := true
		for k, v := range payload.Fields {
			if !first {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%s=", k))
			sb.WriteString(fmt.Sprintf("%v", v))
			first = false
		}
	}
	sb.WriteString("\n")

	// Write to the configured writer
	fmt.Fprint(l.writer, sb.String())
}
