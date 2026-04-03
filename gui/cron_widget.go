package gui

import (
	appserver "badgermaps/app/server"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParseMode string

const (
	ParseModeFields ParseMode = "fields"
	ParseModeRaw    ParseMode = "raw"
)

type CronFields struct {
	Second     string
	Minute     string
	Hour       string
	DayOfMonth string
	Month      string
	DayOfWeek  string
}

func DefaultCronFields() CronFields {
	return CronFields{
		Second:     "0",
		Minute:     "0",
		Hour:       "0",
		DayOfMonth: "*",
		Month:      "*",
		DayOfWeek:  "*",
	}
}

func ParseCronExpressionToFields(expr string) (CronFields, ParseMode, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return CronFields{}, ParseModeFields, fmt.Errorf("cron expression is required")
	}

	if strings.HasPrefix(trimmed, "@") {
		return CronFields{}, ParseModeRaw, nil
	}

	parts := strings.Fields(trimmed)
	switch len(parts) {
	case 5:
		parts = append([]string{"0"}, parts...)
	case 6:
		// Already in expected field shape.
	default:
		return CronFields{}, ParseModeFields, fmt.Errorf("expected 5 or 6 cron fields, got %d", len(parts))
	}

	return CronFields{
		Second:     parts[0],
		Minute:     parts[1],
		Hour:       parts[2],
		DayOfMonth: parts[3],
		Month:      parts[4],
		DayOfWeek:  parts[5],
	}, ParseModeFields, nil
}

func ConsolidateCronFields(fields CronFields) (string, error) {
	values := []struct {
		name  string
		value string
	}{
		{name: "second", value: strings.TrimSpace(fields.Second)},
		{name: "minute", value: strings.TrimSpace(fields.Minute)},
		{name: "hour", value: strings.TrimSpace(fields.Hour)},
		{name: "day", value: strings.TrimSpace(fields.DayOfMonth)},
		{name: "month", value: strings.TrimSpace(fields.Month)},
		{name: "weekday", value: strings.TrimSpace(fields.DayOfWeek)},
	}

	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value.value == "" {
			return "", fmt.Errorf("%s field is required", value.name)
		}
		parts = append(parts, value.value)
	}

	expression := strings.Join(parts, " ")
	if err := appserver.TestCronExpression(expression); err != nil {
		return "", fmt.Errorf("invalid cron expression: %w", err)
	}

	return expression, nil
}

func ResolveCronScheduleForEditor(expression string) (ParseMode, CronFields, string, error) {
	trimmed := strings.TrimSpace(expression)
	if trimmed == "" {
		return ParseModeFields, DefaultCronFields(), "", nil
	}

	fields, mode, err := ParseCronExpressionToFields(trimmed)
	if err != nil {
		return ParseModeRaw, CronFields{}, trimmed, err
	}
	if mode == ParseModeRaw {
		return ParseModeRaw, CronFields{}, trimmed, nil
	}

	return ParseModeFields, fields, "", nil
}

func ConsolidateScheduleFromEditor(mode ParseMode, cronWidget *CronWidget, rawSchedule string) (string, error) {
	trimmedRaw := strings.TrimSpace(rawSchedule)
	if trimmedRaw != "" {
		if err := appserver.TestCronExpression(trimmedRaw); err != nil {
			return "", fmt.Errorf("invalid cron expression: %w", err)
		}
		return trimmedRaw, nil
	}

	if cronWidget != nil {
		return cronWidget.Expression()
	}

	if mode == ParseModeRaw {
		return "", fmt.Errorf("cron schedule is required")
	}
	return "", fmt.Errorf("cron field widget is required")
}

type CronWidget struct {
	secondEntry     *widget.Entry
	minuteEntry     *widget.Entry
	hourEntry       *widget.Entry
	dayOfMonthEntry *widget.Entry
	monthEntry      *widget.Entry
	dayOfWeekEntry  *widget.Entry
	root            fyne.CanvasObject
}

func NewCronWidget() *CronWidget {
	w := &CronWidget{
		secondEntry:     widget.NewEntry(),
		minuteEntry:     widget.NewEntry(),
		hourEntry:       widget.NewEntry(),
		dayOfMonthEntry: widget.NewEntry(),
		monthEntry:      widget.NewEntry(),
		dayOfWeekEntry:  widget.NewEntry(),
	}

	for _, entry := range []*widget.Entry{
		w.secondEntry,
		w.minuteEntry,
		w.hourEntry,
		w.dayOfMonthEntry,
		w.monthEntry,
		w.dayOfWeekEntry,
	} {
		entry.SetPlaceHolder("*")
	}

	w.root = container.NewHBox(
		w.fieldColumn(w.secondEntry, "Second"),
		w.fieldColumn(w.minuteEntry, "Minute"),
		w.fieldColumn(w.hourEntry, "Hour"),
		w.fieldColumn(w.dayOfMonthEntry, "Day"),
		w.fieldColumn(w.monthEntry, "Month"),
		w.fieldColumn(w.dayOfWeekEntry, "Weekday"),
	)
	w.SetFields(DefaultCronFields())
	return w
}

func (w *CronWidget) fieldColumn(entry *widget.Entry, title string) fyne.CanvasObject {
	label := widget.NewLabel(title)
	label.Alignment = fyne.TextAlignCenter
	return container.NewVBox(entry, label)
}

func (w *CronWidget) Object() fyne.CanvasObject {
	return w.root
}

func (w *CronWidget) SetFields(fields CronFields) {
	w.secondEntry.SetText(strings.TrimSpace(fields.Second))
	w.minuteEntry.SetText(strings.TrimSpace(fields.Minute))
	w.hourEntry.SetText(strings.TrimSpace(fields.Hour))
	w.dayOfMonthEntry.SetText(strings.TrimSpace(fields.DayOfMonth))
	w.monthEntry.SetText(strings.TrimSpace(fields.Month))
	w.dayOfWeekEntry.SetText(strings.TrimSpace(fields.DayOfWeek))
}

func (w *CronWidget) Fields() CronFields {
	return CronFields{
		Second:     strings.TrimSpace(w.secondEntry.Text),
		Minute:     strings.TrimSpace(w.minuteEntry.Text),
		Hour:       strings.TrimSpace(w.hourEntry.Text),
		DayOfMonth: strings.TrimSpace(w.dayOfMonthEntry.Text),
		Month:      strings.TrimSpace(w.monthEntry.Text),
		DayOfWeek:  strings.TrimSpace(w.dayOfWeekEntry.Text),
	}
}

func (w *CronWidget) SetExpression(expr string) error {
	fields, mode, err := ParseCronExpressionToFields(expr)
	if err != nil {
		return err
	}
	if mode == ParseModeRaw {
		return fmt.Errorf("expression requires advanced raw mode")
	}
	w.SetFields(fields)
	return nil
}

func (w *CronWidget) Expression() (string, error) {
	return ConsolidateCronFields(w.Fields())
}
