package app

// EventType defines the type of an event using a custom type for type safety.
type EventType int

const (
	// Pull Events
	PullStart EventType = iota
	ResourceIDsFetched
	FetchDetailStart
	FetchDetailSuccess
	StoreSuccess
	PullError
	PullComplete
	AccountCheckinsPulled

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
)

func (e EventType) String() string {
	return [...]string{
		"PullStart",
		"ResourceIDsFetched",
		"FetchDetailStart",
		"FetchDetailSuccess",
		"StoreSuccess",
		"PullError",
		"PullComplete",
		"AccountCheckinsPulled",
		"PushScanStart",
		"PushScanComplete",
		"PushItemStart",
		"PushItemSuccess",
		"PushItemError",
		"PushError",
		"PushComplete",
		"EventCreate",
		"EventRead",
		"EventUpdate",
		"EventDelete",
	}[e]
}

// Event represents a dispatched event with a type and associated data.
type Event struct {
	Type    EventType
	Source  string      // e.g., "accounts", "checkins"
	Payload interface{} // Can be an error, a data object, an ID, etc.
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
