package events

import (
	"badgermaps/app/state"
	"badgermaps/utils"
	"fmt"
	"strings"
	"time"
)

// LogListener handles log events and prints them to the console.
type LogListener struct {
	State *state.State
}

// NewLogListener creates a new LogListener.
func NewLogListener(state *state.State) *LogListener {
	return &LogListener{State: state}
}

// Handle processes the log event.
func (l *LogListener) Handle(e Event) {
	if l.State.Quiet {
		return
	}

	// Handle Debug events separately as they have a different payload
	if e.Type == Debug {
		if !l.State.Debug {
			return
		}
		msg, ok := e.Payload.(string)
		if !ok {
			return
		}
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		var sb strings.Builder
		sb.WriteString(utils.Colors.Gray("%s", timestamp))
		sb.WriteString(" ")
		sb.WriteString(utils.Colors.Gray("%-5s", "DEBUG")) // Padded level
		sb.WriteString(" ")
		sb.WriteString(utils.Colors.Cyan("[%s]", e.Source))
		sb.WriteString(" ")
		sb.WriteString(msg)
		fmt.Println(sb.String())
		return
	}

	payload, ok := e.Payload.(LogEventPayload)
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

	// Colorize level
	var levelColor func(format string, a ...interface{}) string
	switch payload.Level {
	case LogLevelDebug:
		levelColor = utils.Colors.Gray
	case LogLevelInfo:
		levelColor = utils.Colors.Blue
	case LogLevelWarn:
		levelColor = utils.Colors.Yellow
	case LogLevelError:
		levelColor = utils.Colors.Red
	default:
		levelColor = fmt.Sprintf // No color
	}

	// Build the log string
	var sb strings.Builder
	sb.WriteString(utils.Colors.Gray("%s", timestamp))
	sb.WriteString(" ")
	sb.WriteString(levelColor("%-5s", levelStr)) // Padded level
	sb.WriteString(" ")
	sb.WriteString(utils.Colors.Cyan("[%s]", sourceStr))
	sb.WriteString(" ")
	sb.WriteString(payload.Message)

	// Append fields if they exist
	if len(payload.Fields) > 0 {
		sb.WriteString(" ")
		first := true
		for k, v := range payload.Fields {
			if !first {
				sb.WriteString(" ")
			}
			sb.WriteString(utils.Colors.Green("%s=", k))
			sb.WriteString(fmt.Sprintf("%v", v))
			first = false
		}
	}

	fmt.Println(sb.String())
}
