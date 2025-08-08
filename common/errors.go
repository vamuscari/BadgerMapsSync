package common

import (
	"fmt"
	"os"
)

// Exit codes as defined in the spec
const (
	ExitSuccess            = 0 // Success
	ExitGeneralError       = 1 // General error
	ExitShellBuiltinMisuse = 2 // Misuse of shell builtins
	ExitCmdParsingError    = 3 // Command line parsing error
	ExitAuthFailure        = 4 // Authentication failure
	ExitAPIError           = 5 // API error
	ExitDatabaseError      = 6 // Database error
	ExitNetworkError       = 7 // Network error
	ExitRateLimitExceeded  = 8 // Rate limit exceeded
	ExitTimeoutError       = 9 // Timeout error
)

// ErrorUtil provides functions for handling errors with proper exit codes
type ErrorUtil struct {
	colors *ColorPrinter
}

// NewErrorUtil creates a new ErrorUtil
func NewErrorUtil() *ErrorUtil {
	return &ErrorUtil{
		colors: Colors,
	}
}

// ExitWithGeneralError exits with a general error
func (h *ErrorUtil) ExitWithGeneralError(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Error: "+format, args...))
	os.Exit(ExitGeneralError)
}

// ExitWithShellBuiltinMisuse exits with a shell builtin misuse error
func (h *ErrorUtil) ExitWithShellBuiltinMisuse(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Shell Error: "+format, args...))
	os.Exit(ExitShellBuiltinMisuse)
}

// ExitWithCmdParsingError exits with a command line parsing error
func (h *ErrorUtil) ExitWithCmdParsingError(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Command Error: "+format, args...))
	os.Exit(ExitCmdParsingError)
}

// ExitWithAuthFailure exits with an authentication failure
func (h *ErrorUtil) ExitWithAuthFailure(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Authentication Error: "+format, args...))
	os.Exit(ExitAuthFailure)
}

// ExitWithAPIError exits with an API error
func (h *ErrorUtil) ExitWithAPIError(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("API Error: "+format, args...))
	os.Exit(ExitAPIError)
}

// ExitWithDatabaseError exits with a database error
func (h *ErrorUtil) ExitWithDatabaseError(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Database Error: "+format, args...))
	os.Exit(ExitDatabaseError)
}

// ExitWithNetworkError exits with a network error
func (h *ErrorUtil) ExitWithNetworkError(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Network Error: "+format, args...))
	os.Exit(ExitNetworkError)
}

// ExitWithRateLimitExceeded exits with a rate limit exceeded error
func (h *ErrorUtil) ExitWithRateLimitExceeded(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Rate Limit Error: "+format, args...))
	os.Exit(ExitRateLimitExceeded)
}

// ExitWithTimeoutError exits with a timeout error
func (h *ErrorUtil) ExitWithTimeoutError(format string, args ...interface{}) {
	fmt.Println(h.colors.Red("Timeout Error: "+format, args...))
	os.Exit(ExitTimeoutError)
}

// Global instance for easy access
var Errors = NewErrorUtil()

// InitErrorUtil initializes the global Errors instance
func InitErrorUtil() {
	Errors = NewErrorUtil()
}
