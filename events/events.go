package events

import (
	"fmt"
	"time"
)

// LogLevel defines the severity of a log event.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of a LogLevel.
func (l LogLevel) String() string {
	return [...]string{"DEBUG", "INFO", "WARN", "ERROR"}[l]
}

// EventType defines the type of an event using a custom type for type safety.
type EventType int

const (
	// App Events
	LogEvent EventType = iota // A log message was generated

	// Pull Events
	PullStart
	ResourceIDsFetched
	FetchDetailStart
	FetchDetailSuccess
	StoreSuccess
	PullComplete
	PullError

	// Group Pull Events
	PullAllStart
	PullAllComplete
	PullAllError

	// Push Events
	PushScanStart
	PushScanComplete
	PushItemStart
	PushItemSuccess
	PushItemError
	PushComplete
	PushError

	// Action Config Events (for GUI refresh)
	ActionConfigCreated
	ActionConfigUpdated
	ActionConfigDeleted

	// Action Events
	ActionStart
	ActionSuccess
	ActionError

	// System Events
	Debug
)

func (e EventType) String() string {
	return [...]string{
		"LogEvent",
		"PullStart",
		"ResourceIDsFetched",
		"FetchDetailStart",
		"FetchDetailSuccess",
		"StoreSuccess",
		"PullComplete",
		"PullError",
		"PullAllStart",
		"PullAllComplete",
		"PullAllError",
		"PushScanStart",
		"PushScanComplete",
		"PushItemStart",
		"PushItemSuccess",
		"PushItemError",
		"PushComplete",
		"PushError",
		"ActionConfigCreated",
		"ActionConfigUpdated",
		"ActionConfigDeleted",
		"ActionStart",
		"ActionSuccess",
		"ActionError",
		"Debug",
	}[e]
}

// Event represents a dispatched event with a type and associated data.
type Event struct {
	Type    EventType
	Source  string      // e.g., "accounts", "checkins"
	Payload interface{} // Can be an error, a data object, an ID, etc.
}

// CompletionPayload is used for events that signify the end of a process.
type CompletionPayload struct {
	Success bool
	Error   error
	Count   int
}

// LogEventPayload is the structured data for a log event.
type LogEventPayload struct {
	Timestamp time.Time
	Level     LogLevel
	Source    string
	Message   string
	Fields    map[string]interface{}
}

// NewLogEvent creates a new log event.
func NewLogEvent(level LogLevel, source, message string, fields map[string]interface{}) Event {
	return Event{
		Type:   LogEvent,
		Source: source, // Keep source on parent for routing, but also add to payload
		Payload: LogEventPayload{
			Timestamp: time.Now(),
			Level:     level,
			Source:    source,
			Message:   message,
			Fields:    fields,
		},
	}
}

// Errorf creates a new error log event with formatted message.
func Errorf(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelError, source, fmt.Sprintf(format, a...), nil)
}

// Infof creates a new info log event with formatted message.
func Infof(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelInfo, source, fmt.Sprintf(format, a...), nil)
}

// Debugf creates a new debug log event with formatted message.
func Debugf(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelDebug, source, fmt.Sprintf(format, a...), nil)
}

// EventListener is a function that can handle an event.
type EventListener func(e Event)

// AllEventTypes returns a slice of all event type strings suitable for user configuration.
func AllEventTypes() []string {
	return []string{
		PullComplete.String(),
		PullAllStart.String(),
		PullAllComplete.String(),
		PushScanComplete.String(),
		PushComplete.String(),
		ActionSuccess.String(),
		ActionError.String(),
	}
}

// StringToEventType converts a string to an EventType.
// It returns the corresponding EventType and true if the string is a valid event type,
// otherwise it returns 0 and false.
func StringToEventType(s string) (EventType, bool) {
	for i := 0; i <= int(Debug); i++ {
		if EventType(i).String() == s {
			return EventType(i), true
		}
	}
	return 0, false
}
