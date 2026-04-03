package gui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestSyncCenterCardBackgroundLightModeKeepsJobCardContrast(t *testing.T) {
	fyneApp := test.NewApp()
	fyneApp.Settings().SetTheme(newModernThemeForVariant(theme.VariantLight))

	ui := &Gui{fyneApp: fyneApp}
	section := color.NRGBAModel.Convert(ui.syncCenterCardBackground(0)).(color.NRGBA)
	jobCard := color.NRGBAModel.Convert(ui.syncCenterCardBackground(2)).(color.NRGBA)

	if section == jobCard {
		t.Fatal("expected job card background to differ from section background in light mode")
	}

	sectionLuma := int(section.R)*299 + int(section.G)*587 + int(section.B)*114
	jobCardLuma := int(jobCard.R)*299 + int(jobCard.G)*587 + int(jobCard.B)*114
	if jobCardLuma >= sectionLuma {
		t.Fatalf("expected job card background to be darker than section in light mode (section=%v, job=%v)", section, jobCard)
	}
}
