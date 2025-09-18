package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type modernTheme struct{}

func newModernTheme() fyne.Theme {
	return &modernTheme{}
}

func (m *modernTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x1e, G: 0x1e, B: 0x1e, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 0xff}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x00, G: 0x62, B: 0xff, A: 0xff}
	}

	return theme.DefaultTheme().Color(name, variant)
}

func (m *modernTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *modernTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *modernTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
