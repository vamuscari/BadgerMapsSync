package test

import (
	"badgermaps/app"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// TestCmd creates a new test command
func TestCmd(app *app.State) *cobra.Command {
	app.VerifySetupOrExit()

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests and diagnostics",
		Long:  `Test the BadgerMaps CLI functionality, including API connectivity and database functionality.`,
		Run: func(cmd *cobra.Command, args []string) {
			runTests(app)
		},
	}

	testCmd.AddCommand(testDatabaseCmd(app))
	return testCmd
}

func runTests(app *app.State) {
	fmt.Println(color.CyanString("Running all tests..."))
	testDatabase(app)
	fmt.Println(color.GreenString("All tests completed successfully"))
}

func testDatabaseCmd(app *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Test database functionality",
		Long:  `Test database connectivity and verify that all required tables exist with the correct schema.`,
		Run: func(cmd *cobra.Command, args []string) {
			testDatabase(app)
		},
	}
	return cmd
}

func testDatabase(app *app.State) {
	fmt.Println(color.CyanString("Testing database..."))
	db := app.DB
	if db == nil {
		fmt.Println(color.RedString("FAILED: Database not initialized in app state"))
		os.Exit(1)
	}

	fmt.Println(color.CyanString("Connecting to %s database...", db.GetType()))
	if err := db.TestConnection(); err != nil {
		fmt.Println(color.RedString("FAILED: Could not connect to database"))
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(color.GreenString("PASSED: Database connection successful"))

	fmt.Println(color.CyanString("\nValidating database schema..."))
	if err := db.ValidateSchema(); err != nil {
		fmt.Println(color.RedString("FAILED: Schema validation failed"))
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(color.GreenString("PASSED: All required tables exist"))
}
