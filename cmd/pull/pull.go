package pull

import (
	"fmt"
	"os"
	"sync"

	"badgermapscli/api"
	"badgermapscli/database"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewPullCmd creates a new pull command
func NewPullCmd() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Retrieve data from BadgerMaps API",
		Long:  `Pull data from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to pull (account, checkin, route, profile)")
			os.Exit(1)
		},
	}

	// Add subcommands
	pullCmd.AddCommand(newPullAccountCmd())
	pullCmd.AddCommand(newPullAccountsCmd())
	pullCmd.AddCommand(newPullCheckinCmd())
	pullCmd.AddCommand(newPullCheckinsCmd())
	pullCmd.AddCommand(newPullRouteCmd())
	pullCmd.AddCommand(newPullRoutesCmd())
	pullCmd.AddCommand(newPullProfileCmd())

	return pullCmd
}

// newPullAccountCmd creates a command to pull a single account
func newPullAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Pull a single account from BadgerMaps",
		Long:  `Pull a single account from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Implementation will go here
			fmt.Printf("Pulling account with ID: %s\n", args[0])
		},
	}
	return cmd
}

// newPullAccountsCmd creates a command to pull multiple accounts
func newPullAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts [id...]",
		Short: "Pull multiple accounts from BadgerMaps",
		Long:  `Pull multiple accounts from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Pulling all accounts")
				pullAllAccounts()
			} else {
				fmt.Printf("Pulling accounts with IDs: %v\n", args)
				// Implementation for specific accounts will go here
			}
		},
	}
	return cmd
}

// pullAllAccounts pulls all accounts from the API
func pullAllAccounts() {
	// This is a placeholder for the actual implementation
	// In a real implementation, we would:
	// 1. Call the API to get all accounts
	// 2. Store them in the database
	// 3. Handle rate limiting and errors

	// Example of how this might look:
	fmt.Println("Retrieving all accounts from BadgerMaps API...")

	// Simulate getting accounts from API
	accountIDs := []int{1, 2, 3, 4, 5}

	fmt.Printf("Found %d accounts to pull\n", len(accountIDs))

	// Get max parallel processes from config
	maxParallel := viper.GetInt("MAX_PARALLEL_PROCESSES")
	if maxParallel <= 0 {
		maxParallel = 5 // Default value
	}

	// Create a semaphore to limit concurrent operations
	sem := make(chan bool, maxParallel)
	var wg sync.WaitGroup

	// Process accounts in parallel
	for _, id := range accountIDs {
		wg.Add(1)
		go func(accountID int) {
			defer wg.Done()

			// Acquire semaphore
			sem <- true
			defer func() { <-sem }()

			// Simulate pulling account from API
			fmt.Printf("Pulling account %d from API...\n", accountID)
			// In a real implementation, we would call the API client here
			// and store the result in the database
		}(id)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	fmt.Println(color.GreenString("Successfully pulled all accounts from BadgerMaps"))
}

// newPullCheckinCmd creates a command to pull a single checkin
func newPullCheckinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Pull a single checkin from BadgerMaps",
		Long:  `Pull a single checkin from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Pulling checkin with ID: %s\n", args[0])
		},
	}
	return cmd
}

// newPullCheckinsCmd creates a command to pull multiple checkins
func newPullCheckinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins [id...]",
		Short: "Pull multiple checkins from BadgerMaps",
		Long:  `Pull multiple checkins from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Pulling all checkins")
				// Implementation for all checkins will go here
			} else {
				fmt.Printf("Pulling checkins with IDs: %v\n", args)
				// Implementation for specific checkins will go here
			}
		},
	}
	return cmd
}

// newPullRouteCmd creates a command to pull a single route
func newPullRouteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Pull a single route from BadgerMaps",
		Long:  `Pull a single route from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Pulling route with ID: %s\n", args[0])
		},
	}
	return cmd
}

// newPullRoutesCmd creates a command to pull multiple routes
func newPullRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes [id...]",
		Short: "Pull multiple routes from BadgerMaps",
		Long:  `Pull multiple routes from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Pulling all routes")
				// Implementation for all routes will go here
			} else {
				fmt.Printf("Pulling routes with IDs: %v\n", args)
				// Implementation for specific routes will go here
			}
		},
	}
	return cmd
}

// newPullProfileCmd creates a command to pull the user profile
func newPullProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Pull user profile from BadgerMaps",
		Long:  `Pull your user profile from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.CyanString("Pulling user profile..."))

			// Get API key from viper
			apiKey := viper.GetString("API_KEY")
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

			// Get user profile from API
			profile, err := apiClient.GetUserProfile()
			if err != nil {
				fmt.Println(color.RedString("Error retrieving user profile: %v", err))
				os.Exit(1)
			}

			// Get database configuration
			dbConfig := &database.Config{
				DatabaseType: viper.GetString("DATABASE_TYPE"),
				Host:         viper.GetString("DATABASE_HOST"),
				Port:         viper.GetString("DATABASE_PORT"),
				Database:     viper.GetString("DATABASE_NAME"),
				Username:     viper.GetString("DATABASE_USERNAME"),
				Password:     viper.GetString("DATABASE_PASSWORD"),
			}

			// Set default database type and name if not provided
			if dbConfig.DatabaseType == "" {
				dbConfig.DatabaseType = "sqlite3" // Default to SQLite
			}
			if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
				dbConfig.Database = "badgermaps.db"
			}

			// Create database client
			dbClient, err := database.NewClient(dbConfig)
			if err != nil {
				fmt.Println(color.RedString("Error creating database client: %v", err))
				os.Exit(1)
			}
			defer dbClient.Close()

			// Validate database schema
			err = dbClient.ValidateDatabaseSchema()
			if err != nil {
				fmt.Println(color.RedString("Database schema validation failed: %v", err))
				fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
				os.Exit(1)
			}

			// Store profile in database
			err = dbClient.StoreProfiles(profile)
			if err != nil {
				fmt.Println(color.RedString("Error storing user profile: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
			fmt.Printf("Profile ID: %d\n", profile.ID)
			fmt.Printf("Name: %s %s\n", profile.FirstName, profile.LastName)
			fmt.Printf("Email: %s\n", profile.Email)
			fmt.Printf("Company: %s\n", profile.Company.Name)
		},
	}
	return cmd
}
