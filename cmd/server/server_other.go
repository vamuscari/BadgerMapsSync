//go:build !windows
// +build !windows

package server

import (
	"badgermaps/app"

	"github.com/spf13/cobra"
)

func init() {
	// On non-Windows platforms, these functions are no-ops.
	ServerCmdFunc = func(a *app.App, serverCmd *cobra.Command) {
		// No extra commands to add on non-windows systems
	}
	IsWindowsService = func() bool {
		return false
	}
	RunWindowsService = func() {
		// This should never be called on non-windows systems
	}
}
