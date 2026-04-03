package main

import "strings"

func shouldAttachConsoleForCLI(args []string) bool {
	if len(args) == 0 {
		return false
	}

	if len(args) == 1 && isPureGUILaunchArg(args[0]) {
		return false
	}

	return true
}

func isPureGUILaunchArg(arg string) bool {
	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "--gui", "--gui=true":
		return true
	default:
		return false
	}
}
