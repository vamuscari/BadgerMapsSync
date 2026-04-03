package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestParseCronExpressionToFieldsSixField(t *testing.T) {
	fields, mode, err := ParseCronExpressionToFields("0 15 10 * * 1-5")
	if err != nil {
		t.Fatalf("expected no parse error, got %v", err)
	}
	if mode != ParseModeFields {
		t.Fatalf("expected fields parse mode, got %q", mode)
	}

	if fields.Second != "0" || fields.Minute != "15" || fields.Hour != "10" ||
		fields.DayOfMonth != "*" || fields.Month != "*" || fields.DayOfWeek != "1-5" {
		t.Fatalf("unexpected parsed fields: %+v", fields)
	}
}

func TestParseCronExpressionToFieldsFiveFieldNormalizesSeconds(t *testing.T) {
	fields, mode, err := ParseCronExpressionToFields("0 20 * * *")
	if err != nil {
		t.Fatalf("expected no parse error, got %v", err)
	}
	if mode != ParseModeFields {
		t.Fatalf("expected fields parse mode, got %q", mode)
	}

	if fields.Second != "0" || fields.Minute != "0" || fields.Hour != "20" {
		t.Fatalf("expected normalized seconds + minute/hour, got %+v", fields)
	}
}

func TestParseCronExpressionToFieldsRejectsInvalidTokenCount(t *testing.T) {
	_, _, err := ParseCronExpressionToFields("0 0 0 * * * *")
	if err == nil {
		t.Fatalf("expected invalid field-count parse error")
	}
	if !strings.Contains(err.Error(), "expected 5 or 6 cron fields") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCronExpressionToFieldsMarksDescriptorAsRawMode(t *testing.T) {
	_, mode, err := ParseCronExpressionToFields("@daily")
	if err != nil {
		t.Fatalf("expected descriptor to switch to raw mode without parse error, got %v", err)
	}
	if mode != ParseModeRaw {
		t.Fatalf("expected raw parse mode, got %q", mode)
	}
}

func TestConsolidateCronFieldsReturnsValidatedExpression(t *testing.T) {
	expr, err := ConsolidateCronFields(CronFields{
		Second:     "0",
		Minute:     "0",
		Hour:       "20",
		DayOfMonth: "*",
		Month:      "*",
		DayOfWeek:  "*",
	})
	if err != nil {
		t.Fatalf("expected no consolidate error, got %v", err)
	}
	if expr != "0 0 20 * * *" {
		t.Fatalf("expected consolidated expression %q, got %q", "0 0 20 * * *", expr)
	}
}

func TestConsolidateCronFieldsRejectsEmptyField(t *testing.T) {
	_, err := ConsolidateCronFields(CronFields{
		Second:     "0",
		Minute:     "",
		Hour:       "20",
		DayOfMonth: "*",
		Month:      "*",
		DayOfWeek:  "*",
	})
	if err == nil {
		t.Fatalf("expected consolidate error for empty minute field")
	}
}

func TestConsolidateCronFieldsUsesCronValidator(t *testing.T) {
	_, err := ConsolidateCronFields(CronFields{
		Second:     "0",
		Minute:     "invalid",
		Hour:       "20",
		DayOfMonth: "*",
		Month:      "*",
		DayOfWeek:  "*",
	})
	if err == nil {
		t.Fatalf("expected validator error for invalid minute field")
	}
	if !strings.Contains(err.Error(), "invalid cron expression") {
		t.Fatalf("expected cron expression validation error, got %v", err)
	}
}

func TestCronWidgetNewDefaultExpression(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()

	w := NewCronWidget()
	expr, err := w.Expression()
	if err != nil {
		t.Fatalf("expected no expression error for default widget values, got %v", err)
	}
	if expr != "0 0 0 * * *" {
		t.Fatalf("expected default expression %q, got %q", "0 0 0 * * *", expr)
	}
}

func TestCronWidgetFiveFieldRoundTripNormalizesToSixField(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()

	w := NewCronWidget()
	if err := w.SetExpression("0 20 * * *"); err != nil {
		t.Fatalf("expected no set-expression error for five-field input, got %v", err)
	}

	expr, err := w.Expression()
	if err != nil {
		t.Fatalf("expected no expression error, got %v", err)
	}
	if expr != "0 0 20 * * *" {
		t.Fatalf("expected normalized six-field expression %q, got %q", "0 0 20 * * *", expr)
	}
}

func TestResolveCronScheduleForEditorUsesRawModeForDescriptor(t *testing.T) {
	mode, _, rawValue, err := ResolveCronScheduleForEditor("@daily")
	if err != nil {
		t.Fatalf("expected no resolver error for descriptor, got %v", err)
	}
	if mode != ParseModeRaw {
		t.Fatalf("expected raw mode for descriptor, got %q", mode)
	}
	if rawValue != "@daily" {
		t.Fatalf("expected raw value @daily, got %q", rawValue)
	}
}

func TestConsolidateScheduleFromEditorSupportsRawDescriptor(t *testing.T) {
	expr, err := ConsolidateScheduleFromEditor(ParseModeRaw, nil, "@daily")
	if err != nil {
		t.Fatalf("expected raw descriptor schedule to validate, got %v", err)
	}
	if expr != "@daily" {
		t.Fatalf("expected @daily expression, got %q", expr)
	}
}

func TestConsolidateScheduleFromEditorUsesRawDescriptorInFieldMode(t *testing.T) {
	expr, err := ConsolidateScheduleFromEditor(ParseModeFields, nil, "@daily")
	if err != nil {
		t.Fatalf("expected raw descriptor schedule to validate in fields mode, got %v", err)
	}
	if expr != "@daily" {
		t.Fatalf("expected @daily expression, got %q", expr)
	}
}

func TestConsolidateScheduleFromEditorFallsBackToFieldsWhenRawEmpty(t *testing.T) {
	fyneApp := test.NewApp()
	defer fyneApp.Quit()

	w := NewCronWidget()
	if err := w.SetExpression("0 30 6 * * *"); err != nil {
		t.Fatalf("failed to set cron widget expression: %v", err)
	}

	expr, err := ConsolidateScheduleFromEditor(ParseModeRaw, w, "")
	if err != nil {
		t.Fatalf("expected fallback to cron fields when raw value is empty, got %v", err)
	}
	if expr != "0 30 6 * * *" {
		t.Fatalf("expected fallback field expression %q, got %q", "0 30 6 * * *", expr)
	}
}

func TestConsolidateScheduleFromEditorValidatesRawMode(t *testing.T) {
	_, err := ConsolidateScheduleFromEditor(ParseModeRaw, nil, "@not-a-real-descriptor")
	if err == nil {
		t.Fatalf("expected raw mode validation error for unsupported descriptor")
	}
	if !strings.Contains(err.Error(), "invalid cron expression") {
		t.Fatalf("expected cron validation error, got %v", err)
	}
}
