package server

import (
	"fmt"
	"strings"
	"time"
)

const (
	TimezoneSourceOverride = "override"
	TimezoneSourceGlobal   = "global"
	TimezoneSourceLocal    = "local"
)

func NormalizeTimezone(value string) string {
	return strings.TrimSpace(value)
}

func ValidateTimezone(value string) error {
	trimmed := NormalizeTimezone(value)
	if trimmed == "" {
		return nil
	}

	if _, err := time.LoadLocation(trimmed); err != nil {
		return fmt.Errorf("invalid timezone '%s': %w", trimmed, err)
	}
	return nil
}

func ResolveTimezoneLocation(value string) (*time.Location, error) {
	trimmed := NormalizeTimezone(value)
	if trimmed == "" {
		return time.Local, nil
	}

	loc, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone '%s': %w", trimmed, err)
	}
	return loc, nil
}

func ResolveEffectiveTimezone(jobTimezone, globalTimezone string) (string, string, *time.Location, error) {
	trimmedJob := NormalizeTimezone(jobTimezone)
	if trimmedJob != "" {
		loc, err := time.LoadLocation(trimmedJob)
		if err != nil {
			return "", "", nil, fmt.Errorf("invalid job timezone '%s': %w", trimmedJob, err)
		}
		return trimmedJob, TimezoneSourceOverride, loc, nil
	}

	trimmedGlobal := NormalizeTimezone(globalTimezone)
	if trimmedGlobal != "" {
		loc, err := time.LoadLocation(trimmedGlobal)
		if err != nil {
			return "", "", nil, fmt.Errorf("invalid global timezone '%s': %w", trimmedGlobal, err)
		}
		return trimmedGlobal, TimezoneSourceGlobal, loc, nil
	}

	return time.Local.String(), TimezoneSourceLocal, time.Local, nil
}
