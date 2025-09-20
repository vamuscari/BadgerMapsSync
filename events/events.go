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
	return [...]string{"DEBUG", "WARN", "INFO", "ERROR"}[l]
}

// EventType is a string-based identifier for an event (e.g., "pull.start").
type EventType string

// Payload is the interface that all event payloads must implement.
type Payload interface {
	EventType() EventType
}

// Event represents a dispatched event.
type Event struct {
	Type    EventType
	Source  string  // e.g., "accounts", "checkins"
	Payload Payload // Structured data for the event
}

// --- Standard Payloads ---

// LogPayload is the structured data for a log event.
type LogPayload struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	Fields    map[string]interface{}
}

func (p LogPayload) EventType() EventType { return "log" }

// PullStartPayload is for when a pull operation begins.
type PullStartPayload struct {
	ResourceID interface{}
}

func (p PullStartPayload) EventType() EventType { return "pull.start" }

// ResourceIDsFetchedPayload is for when the initial list of IDs has been fetched.
type ResourceIDsFetchedPayload struct {
	Count int
}

func (p ResourceIDsFetchedPayload) EventType() EventType { return "pull.ids_fetched" }

// FetchDetailStartPayload is for when a detailed record fetch begins.
type FetchDetailStartPayload struct {
	ResourceID interface{}
}

func (p FetchDetailStartPayload) EventType() EventType { return "pull.fetch_detail.start" }

// FetchDetailSuccessPayload is for when a detailed record has been successfully fetched.
type FetchDetailSuccessPayload struct {
	Data interface{}
}

func (p FetchDetailSuccessPayload) EventType() EventType { return "pull.fetch_detail.success" }

// StoreSuccessPayload is for when a record has been successfully stored.
type StoreSuccessPayload struct {
	Data interface{}
}

func (p StoreSuccessPayload) EventType() EventType { return "pull.store.success" }

// CompletionPayload is used for events that signify the end of a process.
type CompletionPayload struct {
	Success bool
	Error   error
	Count   int
}

func (p CompletionPayload) EventType() EventType { return "process.complete" }

// ErrorPayload is for when an error occurs.
type ErrorPayload struct {
	Error error
}

func (p ErrorPayload) EventType() EventType { return "error" }

// --- Push Payloads ---

// PushScanStartPayload is for when a push scan begins.
type PushScanStartPayload struct{}

func (p PushScanStartPayload) EventType() EventType { return "push.scan.start" }

// PushScanCompletePayload is for when a push scan is complete.
type PushScanCompletePayload struct {
	Changes interface{}
}

func (p PushScanCompletePayload) EventType() EventType { return "push.scan.complete" }

// PushItemStartPayload is for when a single item push begins.
type PushItemStartPayload struct {
	Change interface{}
}

func (p PushItemStartPayload) EventType() EventType { return "push.item.start" }

// PushItemSuccessPayload is for when a single item push succeeds.
type PushItemSuccessPayload struct {
	Change interface{}
}

func (p PushItemSuccessPayload) EventType() EventType { return "push.item.success" }

// PushItemErrorPayload is for when a single item push fails.
type PushItemErrorPayload struct {
	Error error
}

func (p PushItemErrorPayload) EventType() EventType { return "push.item.error" }

// PushCompletePayload is for when a push operation is complete.
type PushCompletePayload struct {
	ErrorCount int
}

func (p PushCompletePayload) EventType() EventType { return "push.complete" }

// --- Action Config Payloads ---

// ActionConfigCreatedPayload is for when an action config is created.
type ActionConfigCreatedPayload struct{}

func (p ActionConfigCreatedPayload) EventType() EventType { return "action.config.created" }

// ActionConfigUpdatedPayload is for when an action config is updated.
type ActionConfigUpdatedPayload struct{}

func (p ActionConfigUpdatedPayload) EventType() EventType { return "action.config.updated" }

// ActionConfigDeletedPayload is for when an action config is deleted.
type ActionConfigDeletedPayload struct{}

func (p ActionConfigDeletedPayload) EventType() EventType { return "action.config.deleted" }

// --- Event Helper Functions ---

// NewLogEvent creates a new log event.
func NewLogEvent(level LogLevel, source, message string, fields map[string]interface{}) Event {
	return Event{
		Type:   "log",
		Source: source,
		Payload: LogPayload{
			Timestamp: time.Now(),
			Level:     level,
			Message:   message,
			Fields:    fields,
		},
	}
}

// Errorf creates a new error log event with a formatted message.
func Errorf(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelError, source, fmt.Sprintf(format, a...), nil)
}

// Infof creates a new info log event with a formatted message.
func Infof(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelInfo, source, fmt.Sprintf(format, a...), nil)
}

// Warningf creates a new warning log event with a formatted message.
func Warningf(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelWarn, source, fmt.Sprintf(format, a...), nil)
}

// Debugf creates a new debug log event with a formatted message.
func Debugf(source, format string, a ...interface{}) Event {
	return NewLogEvent(LogLevelDebug, source, fmt.Sprintf(format, a...), nil)
}

// EventListener is a function that can handle an event.
type EventListener func(e Event)

// --- For Action Configuration ---

// AllEventTypes returns a slice of all event type strings suitable for user configuration.
func AllEventTypes() []string {
	return []string{
		"pull.complete",
		"push.complete",
		"action.success",
		"action.error",
	}
}

// AllEventSources returns a slice of all event source strings suitable for user configuration.
func AllEventSources() []string {
	return []string{
		"",
		"accounts",
		"checkins",
		"routes",
		"user_profile",
	}
}
