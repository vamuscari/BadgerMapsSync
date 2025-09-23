package events

import (
	"fmt"
	"reflect"
	"strings"
)

// EventTokenOption describes a token that can be inserted into an action configuration field.
type EventTokenOption struct {
	Label        string
	Token        string
	Format       string
	RequiresPath bool
	Placeholder  string
}

var baseTokenOptions = []EventTokenOption{
	{Label: "Event Type ($EVENT_TYPE)", Token: "$EVENT_TYPE"},
	{Label: "Event Source ($EVENT_SOURCE)", Token: "$EVENT_SOURCE"},
	{Label: "Event JSON ($EVENT_JSON)", Token: "$EVENT_JSON"},
	{Label: "Event Payload JSON ($EVENT_PAYLOAD_JSON)", Token: "$EVENT_PAYLOAD_JSON"},
	{Label: "Event Payload Text ($EVENT_PAYLOAD)", Token: "$EVENT_PAYLOAD"},
}

var customPayloadOption = EventTokenOption{
	Label:        "Custom payload field… ($EVENT_PAYLOAD[path])",
	Format:       "$EVENT_PAYLOAD[%s]",
	RequiresPath: true,
	Placeholder:  "e.g. Data.id",
}

type payloadDescriptor struct {
	payloadType reflect.Type
	payloadKind string
}

type descriptorLookup map[string]*payloadDescriptor

type eventPayloadRegistry struct {
	defaults *payloadDescriptor
	sources  descriptorLookup
}

var payloadRegistry = map[EventType]eventPayloadRegistry{
	"log": {
		defaults: newDescriptor(LogPayload{}),
	},
	"pull.start": {
		defaults: newDescriptor(PullStartPayload{}),
	},
	"pull.ids_fetched": {
		defaults: newDescriptor(ResourceIDsFetchedPayload{}),
	},
	"pull.fetch_detail.start": {
		defaults: newDescriptor(FetchDetailStartPayload{}),
	},
	"pull.fetch_detail.success": {
		defaults: newDescriptor(FetchDetailSuccessPayload{}),
		sources: descriptorLookup{
			"accounts": newDescriptorWithKind(FetchDetailSuccessPayload{}, "account"),
			"checkins": newDescriptorWithKind(FetchDetailSuccessPayload{}, "checkin"),
			"routes":   newDescriptorWithKind(FetchDetailSuccessPayload{}, "route"),
		},
	},
	"pull.store.success": {
		defaults: newDescriptor(StoreSuccessPayload{}),
		sources: descriptorLookup{
			"accounts": newDescriptorWithKind(StoreSuccessPayload{}, "account"),
			"checkins": newDescriptorWithKind(StoreSuccessPayload{}, "checkin"),
			"routes":   newDescriptorWithKind(StoreSuccessPayload{}, "route"),
		},
	},
	"pull.complete": {
		defaults: newDescriptor(CompletionPayload{}),
		sources: descriptorLookup{
			"account":      newDescriptorWithKind(CompletionPayload{}, "account"),
			"check-in":     newDescriptorWithKind(CompletionPayload{}, "checkin"),
			"route":        newDescriptorWithKind(CompletionPayload{}, "route"),
			"user profile": newDescriptorWithKind(CompletionPayload{}, "user"),
		},
	},
	"pull.group.complete": {
		defaults: newDescriptor(CompletionPayload{}),
	},
	"pull.group.error": {
		defaults: newDescriptor(ErrorPayload{}),
	},
	"pull.error": {
		defaults: newDescriptor(ErrorPayload{}),
		sources: descriptorLookup{
			"account":      newDescriptorWithKind(ErrorPayload{}, "account"),
			"check-in":     newDescriptorWithKind(ErrorPayload{}, "checkin"),
			"checkins":     newDescriptorWithKind(ErrorPayload{}, "checkins"),
			"route":        newDescriptorWithKind(ErrorPayload{}, "route"),
			"routes":       newDescriptorWithKind(ErrorPayload{}, "routes"),
			"user profile": newDescriptorWithKind(ErrorPayload{}, "user"),
		},
	},
	"push.scan.start": {
		defaults: newDescriptor(PushScanStartPayload{}),
	},
	"push.scan.complete": {
		defaults: newDescriptor(PushScanCompletePayload{}),
	},
	"push.item.start": {
		defaults: newDescriptor(PushItemStartPayload{}),
		sources: descriptorLookup{
			"accounts": newDescriptorWithKind(PushItemStartPayload{}, "account_change"),
			"checkins": newDescriptorWithKind(PushItemStartPayload{}, "checkin_change"),
		},
	},
	"push.item.success": {
		defaults: newDescriptor(PushItemSuccessPayload{}),
		sources: descriptorLookup{
			"accounts": newDescriptorWithKind(PushItemSuccessPayload{}, "account_change"),
			"checkins": newDescriptorWithKind(PushItemSuccessPayload{}, "checkin_change"),
		},
	},
	"push.item.error": {
		defaults: newDescriptor(PushItemErrorPayload{}),
	},
	"push.complete": {
		defaults: newDescriptor(PushCompletePayload{}),
	},
	"push.error": {
		defaults: newDescriptor(ErrorPayload{}),
	},
	"action.config.created": {
		defaults: newDescriptor(ActionConfigCreatedPayload{}),
	},
	"action.config.updated": {
		defaults: newDescriptor(ActionConfigUpdatedPayload{}),
	},
	"action.config.deleted": {
		defaults: newDescriptor(ActionConfigDeletedPayload{}),
	},
}

func newDescriptor(payload interface{}) *payloadDescriptor {
	return &payloadDescriptor{payloadType: reflect.TypeOf(payload)}
}

func newDescriptorWithKind(payload interface{}, kind string) *payloadDescriptor {
	return &payloadDescriptor{payloadType: reflect.TypeOf(payload), payloadKind: kind}
}

// EventTokenOptions returns the list of dynamic token options for the provided event type and source.
func EventTokenOptions(eventType, source string) []EventTokenOption {
	options := make([]EventTokenOption, 0, len(baseTokenOptions)+8)
	options = append(options, baseTokenOptions...)

	if fields := payloadFieldOptions(EventType(eventType), source); len(fields) > 0 {
		options = append(options, fields...)
	}

	options = append(options, customPayloadOption)
	return options
}

func payloadFieldOptions(eventType EventType, source string) []EventTokenOption {
	descriptor := resolvePayloadDescriptor(eventType, source)
	if descriptor == nil || descriptor.payloadType == nil {
		return nil
	}

	fields := collectPayloadFields(descriptor.payloadType, nil, 0)
	if len(fields) == 0 {
		// Provide at least the root payload path when we know the type but it has no exported fields.
		if descriptor.payloadType.Kind() == reflect.Struct && descriptor.payloadType.NumField() == 0 {
			return []EventTokenOption{makePayloadOption(descriptor, []string{}, "")}
		}
		return nil
	}

	options := make([]EventTokenOption, 0, len(fields))
	for _, field := range fields {
		options = append(options, makePayloadOption(descriptor, field.path, field.displayLabel))
	}
	return options
}

func resolvePayloadDescriptor(eventType EventType, source string) *payloadDescriptor {
	reg, ok := payloadRegistry[eventType]
	if !ok {
		return nil
	}

	normalizedSource := strings.ToLower(strings.TrimSpace(source))
	if normalizedSource != "" && reg.sources != nil {
		if desc := reg.sources[normalizedSource]; desc != nil {
			return desc
		}
	}
	return reg.defaults
}

type payloadField struct {
	path         []string
	displayLabel string
}

func collectPayloadFields(t reflect.Type, prefix []string, depth int) []payloadField {
	if depth > 4 {
		return nil
	}

	if t == nil {
		return nil
	}

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		if len(prefix) == 0 {
			return nil
		}
		return []payloadField{{
			path:         prefix,
			displayLabel: strings.Join(prefix, " › "),
		}}
	}

	// Only expand structs defined in the events package; treat others as leaves.
	if t.PkgPath() != "badgermaps/events" && len(prefix) > 0 {
		return []payloadField{{
			path:         prefix,
			displayLabel: strings.Join(prefix, " › "),
		}}
	}

	// When prefix is non-empty, include the struct itself as selectable.
	fields := make([]payloadField, 0, t.NumField())
	if len(prefix) > 0 {
		fields = append(fields, payloadField{
			path:         prefix,
			displayLabel: strings.Join(prefix, " › "),
		})
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		jsonKey := jsonKeyForField(field)
		if jsonKey == "" {
			continue
		}
		nextPrefix := appendPrefix(prefix, jsonKey)
		fields = append(fields, collectPayloadFields(field.Type, nextPrefix, depth+1)...)
	}

	return fields
}

func jsonKeyForField(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	if tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			return parts[0]
		}
	}
	return field.Name
}

func appendPrefix(prefix []string, key string) []string {
	out := make([]string, len(prefix)+1)
	copy(out, prefix)
	out[len(prefix)] = key
	return out
}

func makePayloadOption(descriptor *payloadDescriptor, path []string, label string) EventTokenOption {
	pathString := strings.Join(path, ".")
	token := "$EVENT_PAYLOAD"
	if pathString != "" {
		token = fmt.Sprintf("$EVENT_PAYLOAD[%s]", pathString)
	}

	displayPieces := []string{"Payload"}
	if descriptor.payloadKind != "" {
		displayPieces = append(displayPieces, descriptor.payloadKind)
	}
	if label != "" {
		displayPieces = append(displayPieces, label)
	} else if pathString != "" {
		displayPieces = append(displayPieces, strings.ReplaceAll(pathString, ".", " › "))
	} else {
		displayPieces = append(displayPieces, "value")
	}

	display := strings.Join(displayPieces, " › ")

	return EventTokenOption{
		Label: display + fmt.Sprintf(" (%s)", token),
		Token: token,
	}
}
