package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type modernTheme struct{}

func (m modernTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DarkTheme().Color(name, variant)
}

func (m modernTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (m modernTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (m modernTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}

func newModernTheme() fyne.Theme {
	return &modernTheme{}
}
