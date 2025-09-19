package test

import (
	"badgermaps/app"
	"badgermaps/app/test"

	"github.com/spf13/cobra"
)

// TestCmd creates a new test command
func TestCmd(App *app.App) *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests and diagnostics",
		Long:  `Test the BadgerMaps CLI functionality, including API connectivity and database functionality.`,
		Run: func(cmd *cobra.Command, args []string) {
			test.RunTests(App)
		},
	}

	testCmd.AddCommand(testDatabaseCmd(App))
	testCmd.AddCommand(testApiCmd(App))
	return testCmd
}

func testDatabaseCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Test database functionality",
		Long:  `Test database connectivity and verify that all required tables exist with the correct schema.`,
		Run: func(cmd *cobra.Command, args []string) {
			test.TestDatabase(App)
		},
	}
	return cmd
}

func testApiCmd(App *app.App) *cobra.Command {
	var save bool
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Test API functionality",
		Long:  `Test API connectivity and verify that all endpoints are responding correctly.`,
		Run: func(cmd *cobra.Command, args []string) {
			test.TestApi(App, save)
		},
	}
	cmd.Flags().BoolVarP(&save, "save", "s", false, "Save test output to a log file and separate files for each endpoint response")
	return cmd
}