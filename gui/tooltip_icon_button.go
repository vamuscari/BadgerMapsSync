package gui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

var _ desktop.Hoverable = (*tooltipIconButton)(nil)

type tooltipIconButton struct {
	widget.Button
	tooltipText string
	onHoverHint func(string)
}

func newTooltipIconButton(icon fyne.Resource, tooltip string, onHoverHint func(string), tapped func()) *tooltipIconButton {
	button := &tooltipIconButton{
		tooltipText: strings.TrimSpace(tooltip),
		onHoverHint: onHoverHint,
	}
	button.Text = ""
	button.Icon = icon
	button.OnTapped = tapped
	button.Importance = widget.LowImportance
	button.ExtendBaseWidget(button)
	return button
}

func (b *tooltipIconButton) MouseIn(event *desktop.MouseEvent) {
	b.Button.MouseIn(event)
	if b.Disabled() || b.onHoverHint == nil {
		return
	}
	b.onHoverHint(b.tooltipText)
}

func (b *tooltipIconButton) MouseMoved(*desktop.MouseEvent) {}

func (b *tooltipIconButton) MouseOut() {
	b.Button.MouseOut()
	if b.onHoverHint != nil {
		b.onHoverHint("")
	}
}

func (b *tooltipIconButton) Tapped(event *fyne.PointEvent) {
	b.Button.Tapped(event)
}
