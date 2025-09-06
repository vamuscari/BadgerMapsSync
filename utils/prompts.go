package utils

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// PromptString prompts for a string value with a default option
func PromptString(reader *bufio.Reader, prompt string, defaultValue string) string {
	if defaultValue == "" {
		fmt.Print(Colors.Cyan("%s: ", prompt))
	} else {
		fmt.Print(Colors.Cyan("%s [%s]: ", prompt, defaultValue))
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

// PromptInt prompts for an integer value with a default option
func PromptInt(reader *bufio.Reader, prompt string, defaultValue int) int {
	if defaultValue == 0 {
		fmt.Print(Colors.Cyan("%s: ", prompt))
	} else {
		fmt.Print(Colors.Cyan("%s [%d]: ", prompt, defaultValue))
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println(Colors.Yellow("Invalid input, using default value."))
		return defaultValue
	}

	return value
}

// PromptPassword prompts for a password value with a default option
func PromptPassword(reader *bufio.Reader, prompt string, defaultValue string) string {
	if defaultValue == "" {
		fmt.Print(Colors.Cyan("%s: ", prompt))
	} else {
		fmt.Print(Colors.Cyan("%s [%s]: ", prompt, defaultValue))
	}

	// Note: In a real implementation, you would use a package like
	// golang.org/x/term to read passwords without echo.
	// For simplicity, we're using regular input here.
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

// PromptBool prompts for a boolean value with a default option
func PromptBool(reader *bufio.Reader, prompt string, defaultValue bool) bool {
	var defaultStr string
	if defaultValue {
		defaultStr = "y"
	} else {
		defaultStr = "n"
	}

	fmt.Print(Colors.Cyan("%s [%s]: ", prompt, defaultStr))

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultValue
	}

	return input == "y" || input == "yes" || input == "true"
}

// PromptChoice prompts for a choice from a list of options
// Defaults to the first option
// Returns the selected option
// Be sure to use more than one option for this to work properly
func PromptChoice(reader *bufio.Reader, prompt string, options []string) string {
	fmt.Print(Colors.Cyan("%s: ", prompt))

	// Print Default Option
	fmt.Printf("%d. %s [Default]\n", 1, options[0])

	for i, option := range options {
		fmt.Printf("%d. %s\n", i+1, option)
	}

	fmt.Print(Colors.Cyan("Enter a number: "))

	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return options[0]
	}

	index, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println(Colors.Yellow("Invalid input, please choose a number."))
		return PromptChoice(reader, prompt, options)
	}

	if index < 1 || index > len(options) {
		fmt.Println(Colors.Yellow("Invalid input, please choose a number between 1 and %d.", len(options)))
		return PromptChoice(reader, prompt, options)
	}

	return options[index-1]
}
