package events

import "fmt"

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
	PullError
	PullComplete

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
	PushError
	PushComplete

	// Event CRUD
	EventCreate
	EventRead
	EventUpdate
	EventDelete

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
		"PullError",
		"PullComplete",
		"PullAllStart",
		"PullAllComplete",
		"PullAllError",
		"PushScanStart",
		"PushScanComplete",
		"PushItemStart",
		"PushItemSuccess",
		"PushItemError",
		"PushComplete",
		"EventCreate",
		"EventRead",
		"EventUpdate",
		"EventDelete",
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

// LogEventPayload is the structured data for a log event.
type LogEventPayload struct {
	Level   LogLevel
	Message string
	Fields  map[string]interface{}
}

// NewLogEvent creates a new log event.
func NewLogEvent(level LogLevel, source, message string, fields map[string]interface{}) Event {
	return Event{
		Type:   LogEvent,
		Source: source,
		Payload: LogEventPayload{
			Level:   level,
			Message: message,
			Fields:  fields,
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

// AllEventTypes returns a slice of all event type strings.
func AllEventTypes() []string {
	var types []string
	for i := 0; i <= int(EventDelete); i++ {
		types = append(types, EventType(i).String())
	}
	return types
}

// StringToEventType converts a string to an EventType.
// It returns the corresponding EventType and true if the string is a valid event type,
// otherwise it returns 0 and false.
func StringToEventType(s string) (EventType, bool) {
	for i := 0; i <= int(EventDelete); i++ {
		if EventType(i).String() == s {
			return EventType(i), true
		}
	}
	return 0, false
}
