package utils

import (
	"badgermaps/app/state"
	"runtime"

	"github.com/fatih/color"
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
	Gray      func(format string, a ...interface{}) string
}

// NewColorPrinter creates a new ColorPrinter that respects the --no-color flag
func NewColorPrinter(noColor bool) *ColorPrinter {
	// Globally disable or enable colors in the fatih/color library
	color.NoColor = noColor

	// Return a printer with the appropriate color functions
	return &ColorPrinter{
		Green:     color.New(color.FgGreen).SprintfFunc(),
		Yellow:    color.New(color.FgYellow).SprintfFunc(),
		Red:       color.New(color.FgRed).SprintfFunc(),
		Blue:      color.New(color.FgBlue).SprintfFunc(),
		Cyan:      color.New(color.FgCyan).SprintfFunc(),
		Bold:      color.New(color.Bold).SprintfFunc(),
		Underline: color.New(color.Underline).SprintfFunc(),
		Gray:      color.New(color.FgHiBlack).SprintfFunc(),
	}
}

// Global instance for easy access
var Colors = NewColorPrinter(false)

// InitColors initializes the global Colors instance
// This should be called after viper is initialized
func InitColors(s *state.State) {
	if runtime.GOOS == "windows" {
		Colors = NewColorPrinter(true)
	} else {
		Colors = NewColorPrinter(s.NoColor)
	}
}
