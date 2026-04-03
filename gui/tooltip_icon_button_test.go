package gui

import (
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"testing"
)

func TestTooltipIconButtonHoverCallbacksAndTap(t *testing.T) {
	_ = test.NewApp()

	var hints []string
	tapped := 0

	button := newTooltipIconButton(theme.MediaPlayIcon(), "Play now", func(text string) {
		hints = append(hints, text)
	}, func() {
		tapped++
	})

	button.MouseIn(&desktop.MouseEvent{})
	if len(hints) != 1 || hints[0] != "Play now" {
		t.Fatalf("expected hover hint callback with %q, got %#v", "Play now", hints)
	}

	button.MouseMoved(&desktop.MouseEvent{})
	if len(hints) != 1 {
		t.Fatalf("expected mouse move not to change hints, got %#v", hints)
	}

	test.Tap(button)
	if tapped != 1 {
		t.Fatalf("expected tap callback to fire once, got %d", tapped)
	}
	if len(hints) != 1 {
		t.Fatalf("expected tap not to alter hover hints, got %#v", hints)
	}

	button.MouseOut()
	if len(hints) != 2 || hints[1] != "" {
		t.Fatalf("expected hover clear callback on mouse out, got %#v", hints)
	}
}
