package utils

import (
	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// ColorPrinter provides color printing functions that respect the --no-color flag
type ColorPrinter struct {
	// Color functions
	Green     func(format string, a ...interface{}) string
	Yellow    func(format string, a ...interface{}) string
	Red       func(format string, a ...interface{}) string
	Blue      func(format string, a ...interface{}) string
	Cyan      func(format string, a ...interface{}) string
	Bold      func(format string, a ...interface{}) string
	Underline func(format string, a ...interface{}) string
}

// NewColorPrinter creates a new ColorPrinter that respects the --no-color flag
func NewColorPrinter() *ColorPrinter {
	// Check if colors should be disabled
	noColor := viper.GetBool("no-color")
	if noColor {
		// If colors are disabled, return a printer with no colors
		return &ColorPrinter{
			Green:     color.New().SprintfFunc(),
			Yellow:    color.New().SprintfFunc(),
			Red:       color.New().SprintfFunc(),
			Blue:      color.New().SprintfFunc(),
			Cyan:      color.New().SprintfFunc(),
			Bold:      color.New().SprintfFunc(),
			Underline: color.New().SprintfFunc(),
		}
	}

	// Otherwise, return a printer with colors
	return &ColorPrinter{
		Green:     color.New(color.FgGreen).SprintfFunc(),
		Yellow:    color.New(color.FgYellow).SprintfFunc(),
		Red:       color.New(color.FgRed).SprintfFunc(),
		Blue:      color.New(color.FgBlue).SprintfFunc(),
		Cyan:      color.New(color.FgCyan).SprintfFunc(),
		Bold:      color.New(color.Bold).SprintfFunc(),
		Underline: color.New(color.Underline).SprintfFunc(),
	}
}

// Global instance for easy access
var Colors = NewColorPrinter()

// InitColors initializes the global Colors instance
// This should be called after viper is initialized
func InitColors() {
	Colors = NewColorPrinter()
}
