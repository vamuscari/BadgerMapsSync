package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const (
	StatusPositiveColorName       fyne.ThemeColorName = "statusPositive"
	StatusNegativeColorName       fyne.ThemeColorName = "statusNegative"
	StatusCardBackgroundColorName fyne.ThemeColorName = "statusCardBackground"
	StatusCardBorderColorName     fyne.ThemeColorName = "statusCardBorder"
)

type modernTheme struct {
	forceVariant *fyne.ThemeVariant
}

func newModernTheme() fyne.Theme {
	return &modernTheme{}
}

func newModernThemeForVariant(variant fyne.ThemeVariant) fyne.Theme {
	return &modernTheme{forceVariant: &variant}
}

func (m *modernTheme) effectiveVariant(v fyne.ThemeVariant) fyne.ThemeVariant {
	if m.forceVariant != nil {
		return *m.forceVariant
	}
	return v
}

func (m *modernTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	variant = m.effectiveVariant(variant)
	if variant == theme.VariantLight {
		switch name {
		case StatusPositiveColorName:
			return color.NRGBA{R: 0x0f, G: 0x7a, B: 0x2a, A: 0xff}
		case StatusNegativeColorName:
			return color.NRGBA{R: 0xb4, G: 0x1e, B: 0x1e, A: 0xff}
		case StatusCardBackgroundColorName:
			return color.NRGBA{R: 0xf9, G: 0xfb, B: 0xff, A: 0xff}
		case StatusCardBorderColorName:
			return color.NRGBA{R: 0xd0, G: 0xd5, B: 0xde, A: 0xff}
		case theme.ColorNameBackground:
			return color.NRGBA{R: 0xf5, G: 0xf5, B: 0xf5, A: 0xff}
		case theme.ColorNameButton:
			return color.NRGBA{R: 0xe0, G: 0xe0, B: 0xe0, A: 0xff}
		case theme.ColorNamePrimary:
			return color.NRGBA{R: 0x00, G: 0x62, B: 0xff, A: 0xff}
			// Add other light theme colors as needed
		}
	}

	// Dark theme colors (default)
	switch name {
	case StatusPositiveColorName:
		return color.NRGBA{R: 0x52, G: 0xd6, B: 0x6d, A: 0xff}
	case StatusNegativeColorName:
		return color.NRGBA{R: 0xff, G: 0x6a, B: 0x6a, A: 0xff}
	case StatusCardBackgroundColorName:
		return color.NRGBA{R: 0x28, G: 0x2c, B: 0x36, A: 0xff}
	case StatusCardBorderColorName:
		return color.NRGBA{R: 0x3a, G: 0x3f, B: 0x4b, A: 0xff}
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
