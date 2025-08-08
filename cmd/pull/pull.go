package pull

import (
	"badgermapscli/api"
	"badgermapscli/database"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// PullCmd creates a new pull command
func PullCmd() *cobra.Command {
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Retrieve data from BadgerMaps API",
		Long:  `Pull data from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to pull (account, checkin, route, profile, all)")
			os.Exit(1)
		},
	}

	// Global flags are defined in main.go

	// Add subcommands
	pullCmd.AddCommand(pullAccountCmd())
	pullCmd.AddCommand(pullAccountsCmd())
	pullCmd.AddCommand(pullCheckinCmd())
	pullCmd.AddCommand(pullCheckinsCmd())
	pullCmd.AddCommand(pullRouteCmd())
	pullCmd.AddCommand(pullRoutesCmd())
	pullCmd.AddCommand(pullProfileCmd())
	pullCmd.AddCommand(pullAllCmd())

	return pullCmd
}

// pullAccountCmd creates a command to pull a single account
func pullAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Pull a single account from BadgerMaps",
		Long:  `Pull a single account from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")

			if verbose {
				fmt.Println(color.CyanString("Pulling account with ID: %s", args[0]))
			}

			// Get API key from viper
			apiKey := viper.GetString("API_KEY")
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Parse account ID
			accountID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid account ID: %s", args[0]))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

			// Get account from API
			account, err := apiClient.GetAccount(accountID)
			if err != nil {
				fmt.Println(color.RedString("Error retrieving account: %v", err))
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
			dbClient, err := database.NewClient(dbConfig, false)
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

			// Store account in database
			accounts := []api.Account{*account}
			err = dbClient.StoreAccounts(accounts)
			if err != nil {
				fmt.Println(color.RedString("Error storing account: %v", err))
				os.Exit(1)
			}

			if verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored account"))
				fmt.Printf("Account ID: %d\n", account.ID)
				fmt.Printf("Name: %s\n", account.FullName)
				fmt.Printf("Email: %s\n", account.Email)
				fmt.Printf("Locations: %d\n", len(account.Locations))
			}
		},
	}
	return cmd
}

// pullAccountsCmd creates a command to pull multiple accounts
func pullAccountsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts [id...]",
		Short: "Pull multiple accounts from BadgerMaps",
		Long:  `Pull multiple accounts from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")
			// Get top flag
			top, _ := cmd.Flags().GetInt("top")
			if len(args) == 0 {
				if verbose {
					fmt.Println(color.CyanString("Pulling all accounts"))
				}
				pullAllAccounts(verbose, top)
			} else {
				if verbose {
					fmt.Println(color.CyanString("Pulling accounts with IDs: %v", args))
				}

				// Get API key from viper
				apiKey := viper.GetString("API_KEY")
				if apiKey == "" {
					fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
					os.Exit(1)
				}

				// Create API client
				apiClient := api.NewAPIClient(apiKey)

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
				dbClient, err := database.NewClient(dbConfig, verbose)
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

				// Parse account IDs
				accountIDs := make([]int, 0, len(args))
				for _, arg := range args {
					id, err := strconv.Atoi(arg)
					if err != nil {
						fmt.Println(color.RedString("Invalid account ID: %s", arg))
						os.Exit(1)
					}
					accountIDs = append(accountIDs, id)
				}

				// Get max parallel processes from config
				maxParallel := viper.GetInt("MAX_PARALLEL_PROCESSES")
				if maxParallel <= 0 {
					maxParallel = 5 // Default value
				}

				// Create a semaphore to limit concurrent operations
				sem := make(chan bool, maxParallel)
				var wg sync.WaitGroup

				// Process accounts in parallel
				accounts := make([]api.Account, 0, len(accountIDs))
				var accountsMutex sync.Mutex

				for _, id := range accountIDs {
					wg.Add(1)
					go func(accountID int) {
						defer wg.Done()

						// Acquire semaphore
						sem <- true
						defer func() { <-sem }()

						fmt.Printf("Pulling account %d from API...\n", accountID)

						// Get account from API
						account, err := apiClient.GetAccount(accountID)
						if err != nil {
							fmt.Println(color.RedString("Error retrieving account %d: %v", accountID, err))
							return
						}

						// Add account to the list
						accountsMutex.Lock()
						accounts = append(accounts, *account)
						accountsMutex.Unlock()
					}(id)
				}

				// Wait for all goroutines to finish
				wg.Wait()

				if len(accounts) == 0 {
					fmt.Println(color.YellowString("No accounts were retrieved successfully"))
					os.Exit(1)
				}

				// Store accounts in database
				err = dbClient.StoreAccounts(accounts)
				if err != nil {
					fmt.Println(color.RedString("Error storing accounts: %v", err))
					os.Exit(1)
				}

				if verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored %d accounts", len(accounts)))
				}
			}
		},
	}

	// Add top flag
	cmd.Flags().Int("top", 10, "Limit the number of accounts to retrieve (default 10)")

	return cmd
}

// pullAllAccounts pulls all accounts from the API
func pullAllAccounts(verbose bool, top int) {
	if verbose {
		fmt.Println(color.CyanString("Retrieving all accounts from BadgerMaps API..."))
	}

	// Get API key from viper
	apiKey := viper.GetString("API_KEY")
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get all accounts from API (basic list)
	accountsList, err := apiClient.GetAccounts()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving accounts list: %v", err))
		os.Exit(1)
	}

	// Limit the number of accounts if top is specified
	if top > 0 && top < len(accountsList) {
		if verbose {
			fmt.Printf("Found %d accounts, limiting to top %d\n", len(accountsList), top)
		}
		accountsList = accountsList[:top]
	} else if verbose {
		fmt.Printf("Found %d accounts to pull\n", len(accountsList))
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
	dbClient, err := database.NewClient(dbConfig, verbose)
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

	// Get max parallel processes from config
	maxParallel := viper.GetInt("MAX_PARALLEL_PROCESSES")
	if maxParallel <= 0 {
		maxParallel = 5 // Default value
	}

	if verbose {
		fmt.Printf("Using maximum of %d concurrent operations\n", maxParallel)
	}

	// Create a semaphore to limit concurrent operations
	sem := make(chan bool, maxParallel)
	var wg sync.WaitGroup

	// Process accounts in parallel
	var successCount int32
	var accountsMutex sync.Mutex

	if verbose {
		fmt.Println(color.CyanString("Retrieving and storing detailed account information..."))
	}

	// Create progress bar if not in verbose mode
	var bar *progressbar.ProgressBar
	if !verbose {
		bar = progressbar.NewOptions(len(accountsList),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription("[cyan]Retrieving and storing accounts...[reset]"),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
	}

	for _, basicAccount := range accountsList {
		wg.Add(1)
		go func(accountID int) {
			defer wg.Done()

			// Acquire semaphore
			sem <- true
			defer func() { <-sem }()

			// Get detailed account from API
			detailedAccount, err := apiClient.GetAccount(accountID)
			if err != nil {
				fmt.Println(color.RedString("Error retrieving account %d: %v", accountID, err))
				return
			}

			// Store account directly in the database
			accounts := []api.Account{*detailedAccount}
			err = dbClient.StoreAccounts(accounts)
			if err != nil {
				fmt.Println(color.RedString("Error storing account %d: %v", accountID, err))
				return
			}

			// Increment success counter
			accountsMutex.Lock()
			successCount++
			if verbose {
				fmt.Printf("Retrieved and stored account %d: %s\n", accountID, detailedAccount.FullName)
			} else if bar != nil {
				bar.Add(1)
			}
			accountsMutex.Unlock()
		}(basicAccount.ID)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Add a newline after the progress bar
	if !verbose && bar != nil {
		fmt.Println()
	}

	if successCount == 0 {
		fmt.Println(color.YellowString("No accounts were retrieved and stored successfully"))
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Successfully retrieved and stored %d/%d accounts\n", successCount, len(accountsList))
		fmt.Println(color.GreenString("Successfully pulled and stored all accounts from BadgerMaps"))
	}
}

// pullCheckinCmd creates a command to pull a single checkin
func pullCheckinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Pull a single checkin from BadgerMaps",
		Long:  `Pull a single checkin from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")

			if verbose {
				fmt.Println(color.CyanString("Pulling checkin with ID: %s", args[0]))
			}

			// Get API key from viper
			apiKey := viper.GetString("API_KEY")
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Parse checkin ID
			checkinID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid checkin ID: %s", args[0]))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

			// Get checkin from API
			checkin, err := apiClient.GetCheckin(checkinID)
			if err != nil {
				fmt.Println(color.RedString("Error retrieving checkin: %v", err))
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
			dbClient, err := database.NewClient(dbConfig, false)
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

			// Store checkin in database
			checkins := []api.Checkin{*checkin}
			err = dbClient.StoreCheckins(checkins)
			if err != nil {
				fmt.Println(color.RedString("Error storing checkin: %v", err))
				os.Exit(1)
			}

			if verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored checkin"))
				fmt.Printf("Checkin ID: %d\n", checkin.ID)
				fmt.Printf("Customer: %d\n", checkin.Customer)
				fmt.Printf("Type: %s\n", checkin.Type)
				fmt.Printf("Date: %s\n", checkin.LogDatetime)
			}
		},
	}
	return cmd
}

// pullCheckinsCmd creates a command to pull multiple checkins
func pullCheckinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins [id...]",
		Short: "Pull multiple checkins from BadgerMaps",
		Long:  `Pull multiple checkins from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")
			if len(args) == 0 {
				if verbose {
					fmt.Println(color.CyanString("Pulling all checkins"))
				}
				pullAllCheckins(verbose)
			} else {
				if verbose {
					fmt.Println(color.CyanString("Pulling checkins with IDs: %v", args))
				}

				// Get API key from viper
				apiKey := viper.GetString("API_KEY")
				if apiKey == "" {
					fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
					os.Exit(1)
				}

				// Create API client
				apiClient := api.NewAPIClient(apiKey)

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
				dbClient, err := database.NewClient(dbConfig, verbose)
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

				// Parse checkin IDs
				checkinIDs := make([]int, 0, len(args))
				for _, arg := range args {
					id, err := strconv.Atoi(arg)
					if err != nil {
						fmt.Println(color.RedString("Invalid checkin ID: %s", arg))
						os.Exit(1)
					}
					checkinIDs = append(checkinIDs, id)
				}

				// Get max parallel processes from config
				maxParallel := viper.GetInt("MAX_PARALLEL_PROCESSES")
				if maxParallel <= 0 {
					maxParallel = 5 // Default value
				}

				// Create a semaphore to limit concurrent operations
				sem := make(chan bool, maxParallel)
				var wg sync.WaitGroup

				// Process checkins in parallel
				checkins := make([]api.Checkin, 0, len(checkinIDs))
				var checkinsMutex sync.Mutex

				for _, id := range checkinIDs {
					wg.Add(1)
					go func(checkinID int) {
						defer wg.Done()

						// Acquire semaphore
						sem <- true
						defer func() { <-sem }()

						if verbose {
							fmt.Printf("Pulling checkin %d from API...\n", checkinID)
						}

						// Get checkin from API
						checkin, err := apiClient.GetCheckin(checkinID)
						if err != nil {
							fmt.Println(color.RedString("Error retrieving checkin %d: %v", checkinID, err))
							return
						}

						// Add checkin to the list
						checkinsMutex.Lock()
						checkins = append(checkins, *checkin)
						checkinsMutex.Unlock()
					}(id)
				}

				// Wait for all goroutines to finish
				wg.Wait()

				if len(checkins) == 0 {
					fmt.Println(color.YellowString("No checkins were retrieved successfully"))
					os.Exit(1)
				}

				// Store checkins in database
				err = dbClient.StoreCheckins(checkins)
				if err != nil {
					fmt.Println(color.RedString("Error storing checkins: %v", err))
					os.Exit(1)
				}

				if verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored %d checkins", len(checkins)))
				}
			}
		},
	}
	return cmd
}

// pullAllCheckins pulls all checkins from the API
func pullAllCheckins(verbose bool) {
	if verbose {
		fmt.Println(color.CyanString("Retrieving all checkins from BadgerMaps API..."))
	}

	// Get API key from viper
	apiKey := viper.GetString("API_KEY")
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get all checkins from API
	checkins, err := apiClient.GetCheckins()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving checkins: %v", err))
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Found %d checkins to pull\n", len(checkins))
	}

	// Create progress bar if not in verbose mode
	var bar *progressbar.ProgressBar
	if !verbose {
		bar = progressbar.NewOptions(len(checkins),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription("[cyan]Storing checkins...[reset]"),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
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
	dbClient, err := database.NewClient(dbConfig, verbose)
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

	// Store checkins in database
	if verbose {
		fmt.Println("Storing checkins in database...")
	}

	err = dbClient.StoreCheckins(checkins)
	if err != nil {
		fmt.Println(color.RedString("Error storing checkins: %v", err))
		os.Exit(1)
	}

	// Update progress bar if not in verbose mode
	if !verbose && bar != nil {
		bar.Finish()
		fmt.Println()
	}

	if verbose {
		fmt.Println(color.GreenString("Successfully pulled and stored all checkins from BadgerMaps"))
	}
}

// pullRouteCmd creates a command to pull a single route
func pullRouteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Pull a single route from BadgerMaps",
		Long:  `Pull a single route from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")

			if verbose {
				fmt.Println(color.CyanString("Pulling route with ID: %s", args[0]))
			}

			// Get API key from viper
			apiKey := viper.GetString("API_KEY")
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Parse route ID
			routeID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid route ID: %s", args[0]))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

			// Get route from API
			route, err := apiClient.GetRoute(routeID)
			if err != nil {
				fmt.Println(color.RedString("Error retrieving route: %v", err))
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
			dbClient, err := database.NewClient(dbConfig, false)
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

			// Store route in database
			routes := []api.Route{*route}
			err = dbClient.StoreRoutes(routes)
			if err != nil {
				fmt.Println(color.RedString("Error storing route: %v", err))
				os.Exit(1)
			}

			if verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored route"))
				fmt.Printf("Route ID: %d\n", route.ID)
				fmt.Printf("Name: %s\n", route.Name)
				fmt.Printf("Date: %s\n", route.RouteDate)
				fmt.Printf("Waypoints: %d\n", len(route.Waypoints))
			}
		},
	}
	return cmd
}

// pullRoutesCmd creates a command to pull multiple routes
func pullRoutesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes [id...]",
		Short: "Pull multiple routes from BadgerMaps",
		Long:  `Pull multiple routes from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")
			if len(args) == 0 {
				if verbose {
					fmt.Println(color.CyanString("Pulling all routes"))
				}
				pullAllRoutes(verbose)
			} else {
				if verbose {
					fmt.Println(color.CyanString("Pulling routes with IDs: %v", args))
				}

				// Get API key from viper
				apiKey := viper.GetString("API_KEY")
				if apiKey == "" {
					fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
					os.Exit(1)
				}

				// Create API client
				apiClient := api.NewAPIClient(apiKey)

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
				dbClient, err := database.NewClient(dbConfig, verbose)
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

				// Parse route IDs
				routeIDs := make([]int, 0, len(args))
				for _, arg := range args {
					id, err := strconv.Atoi(arg)
					if err != nil {
						fmt.Println(color.RedString("Invalid route ID: %s", arg))
						os.Exit(1)
					}
					routeIDs = append(routeIDs, id)
				}

				// Get max parallel processes from config
				maxParallel := viper.GetInt("MAX_PARALLEL_PROCESSES")
				if maxParallel <= 0 {
					maxParallel = 5 // Default value
				}

				// Create a semaphore to limit concurrent operations
				sem := make(chan bool, maxParallel)
				var wg sync.WaitGroup

				// Process routes in parallel
				routes := make([]api.Route, 0, len(routeIDs))
				var routesMutex sync.Mutex

				for _, id := range routeIDs {
					wg.Add(1)
					go func(routeID int) {
						defer wg.Done()

						// Acquire semaphore
						sem <- true
						defer func() { <-sem }()

						if verbose {
							fmt.Printf("Pulling route %d from API...\n", routeID)
						}

						// Get route from API
						route, err := apiClient.GetRoute(routeID)
						if err != nil {
							fmt.Println(color.RedString("Error retrieving route %d: %v", routeID, err))
							return
						}

						// Add route to the list
						routesMutex.Lock()
						routes = append(routes, *route)
						routesMutex.Unlock()
					}(id)
				}

				// Wait for all goroutines to finish
				wg.Wait()

				if len(routes) == 0 {
					fmt.Println(color.YellowString("No routes were retrieved successfully"))
					os.Exit(1)
				}

				// Store routes in database
				err = dbClient.StoreRoutes(routes)
				if err != nil {
					fmt.Println(color.RedString("Error storing routes: %v", err))
					os.Exit(1)
				}

				if verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored %d routes", len(routes)))
				}
			}
		},
	}
	return cmd
}

// pullAllRoutes pulls all routes from the API
func pullAllRoutes(verbose bool) {
	if verbose {
		fmt.Println(color.CyanString("Retrieving all routes from BadgerMaps API..."))
	}

	// Get API key from viper
	apiKey := viper.GetString("API_KEY")
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get all routes from API
	routes, err := apiClient.GetRoutes()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving routes: %v", err))
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Found %d routes to pull\n", len(routes))
	}

	// Create progress bar if not in verbose mode
	var bar *progressbar.ProgressBar
	if !verbose {
		bar = progressbar.NewOptions(len(routes),
			progressbar.OptionEnableColorCodes(true),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription("[cyan]Storing routes...[reset]"),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "[green]=[reset]",
				SaucerHead:    "[green]>[reset]",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}))
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
	dbClient, err := database.NewClient(dbConfig, verbose)
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

	// Store routes in database
	if verbose {
		fmt.Println("Storing routes in database...")
	}

	err = dbClient.StoreRoutes(routes)
	if err != nil {
		fmt.Println(color.RedString("Error storing routes: %v", err))
		os.Exit(1)
	}

	// Update progress bar if not in verbose mode
	if !verbose && bar != nil {
		bar.Finish()
		fmt.Println()
	}

	if verbose {
		fmt.Println(color.GreenString("Successfully pulled and stored all routes from BadgerMaps"))
	}
}

// pullProfileCmd creates a command to pull the user profile
func pullProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Pull user profile from BadgerMaps",
		Long:  `Pull your user profile from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")

			if verbose {
				fmt.Println(color.CyanString("Pulling user profile..."))
			}

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
			dbClient, err := database.NewClient(dbConfig, false)
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

			if verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				fmt.Printf("Profile ID: %d\n", profile.ID)
				fmt.Printf("Name: %s %s\n", profile.FirstName, profile.LastName)
				fmt.Printf("Email: %s\n", profile.Email)
				fmt.Printf("Company: %s\n", profile.Company.Name)
			}
		},
	}
	return cmd
}

// pullAllCmd creates a command to pull all data types in order
func pullAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Pull all data types from BadgerMaps",
		Long:  `Pull all data types from the BadgerMaps API to your local database in the order: profile, accounts, checkins, routes.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get verbose flag from global config
			verbose := viper.GetBool("verbose")

			if verbose {
				fmt.Println(color.CyanString("Pulling all data types in order: profile, accounts, checkins, routes"))
			}

			// Step 1: Pull profile
			if verbose {
				fmt.Println(color.CyanString("\n=== Pulling user profile ==="))
			} else {
				fmt.Println("Pulling user profile...")
			}

			// Get API key from viper
			apiKey := viper.GetString("API_KEY")
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

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
			dbClient, err := database.NewClient(dbConfig, verbose)
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

			// Get user profile from API
			profile, err := apiClient.GetUserProfile()
			if err != nil {
				fmt.Println(color.RedString("Error retrieving user profile: %v", err))
				fmt.Println(color.YellowString("Continuing with other data types..."))
			} else {
				// Store profile in database
				err = dbClient.StoreProfiles(profile)
				if err != nil {
					fmt.Println(color.RedString("Error storing user profile: %v", err))
					fmt.Println(color.YellowString("Continuing with other data types..."))
				} else if verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				}
			}

			// Step 2: Pull accounts
			if verbose {
				fmt.Println(color.CyanString("\n=== Pulling accounts ==="))
			} else {
				fmt.Println("Pulling accounts...")
			}

			// Get top value for accounts (default to 0 which means all)
			top := 0

			// Get all accounts from API (basic list)
			accountsList, err := apiClient.GetAccounts()
			if err != nil {
				fmt.Println(color.RedString("Error retrieving accounts list: %v", err))
				fmt.Println(color.YellowString("Continuing with other data types..."))
			} else {
				// Limit the number of accounts if top is specified
				if top > 0 && top < len(accountsList) {
					if verbose {
						fmt.Printf("Found %d accounts, limiting to top %d\n", len(accountsList), top)
					}
					accountsList = accountsList[:top]
				} else if verbose {
					fmt.Printf("Found %d accounts to pull\n", len(accountsList))
				}

				// Get max parallel processes from config
				maxParallel := viper.GetInt("MAX_PARALLEL_PROCESSES")
				if maxParallel <= 0 {
					maxParallel = 5 // Default value
				}

				if verbose {
					fmt.Printf("Using maximum of %d concurrent operations\n", maxParallel)
				}

				// Create a semaphore to limit concurrent operations
				sem := make(chan bool, maxParallel)
				var wg sync.WaitGroup

				// Process accounts in parallel
				var successCount int32
				var accountsMutex sync.Mutex

				if verbose {
					fmt.Println(color.CyanString("Retrieving and storing detailed account information..."))
				}

				// Create progress bar if not in verbose mode
				var bar *progressbar.ProgressBar
				if !verbose {
					bar = progressbar.NewOptions(len(accountsList),
						progressbar.OptionEnableColorCodes(true),
						progressbar.OptionShowCount(),
						progressbar.OptionSetDescription("[cyan]Retrieving and storing accounts...[reset]"),
						progressbar.OptionSetTheme(progressbar.Theme{
							Saucer:        "[green]=[reset]",
							SaucerHead:    "[green]>[reset]",
							SaucerPadding: " ",
							BarStart:      "[",
							BarEnd:        "]",
						}))
				}

				for _, basicAccount := range accountsList {
					wg.Add(1)
					go func(accountID int) {
						defer wg.Done()

						// Acquire semaphore
						sem <- true
						defer func() { <-sem }()

						// Get detailed account from API
						detailedAccount, err := apiClient.GetAccount(accountID)
						if err != nil {
							fmt.Println(color.RedString("Error retrieving account %d: %v", accountID, err))
							return
						}

						// Store account directly in the database
						accounts := []api.Account{*detailedAccount}
						err = dbClient.StoreAccounts(accounts)
						if err != nil {
							fmt.Println(color.RedString("Error storing account %d: %v", accountID, err))
							return
						}

						// Increment success counter
						accountsMutex.Lock()
						successCount++
						if verbose {
							fmt.Printf("Retrieved and stored account %d: %s\n", accountID, detailedAccount.FullName)
						} else if bar != nil {
							bar.Add(1)
						}
						accountsMutex.Unlock()
					}(basicAccount.ID)
				}

				// Wait for all goroutines to finish
				wg.Wait()

				// Add a newline after the progress bar
				if !verbose && bar != nil {
					fmt.Println()
				}

				if successCount == 0 && len(accountsList) > 0 {
					fmt.Println(color.YellowString("No accounts were retrieved and stored successfully"))
				} else if verbose {
					fmt.Printf("Successfully retrieved and stored %d/%d accounts\n", successCount, len(accountsList))
					fmt.Println(color.GreenString("Successfully pulled and stored accounts from BadgerMaps"))
				}
			}

			// Step 3: Pull checkins
			if verbose {
				fmt.Println(color.CyanString("\n=== Pulling checkins ==="))
			} else {
				fmt.Println("Pulling checkins...")
			}

			// Get all checkins from API
			checkins, err := apiClient.GetCheckins()
			if err != nil {
				fmt.Println(color.RedString("Error retrieving checkins: %v", err))
				fmt.Println(color.YellowString("Continuing with other data types..."))
			} else {
				if verbose {
					fmt.Printf("Found %d checkins to pull\n", len(checkins))
				}

				// Create progress bar if not in verbose mode
				var bar *progressbar.ProgressBar
				if !verbose {
					bar = progressbar.NewOptions(len(checkins),
						progressbar.OptionEnableColorCodes(true),
						progressbar.OptionShowCount(),
						progressbar.OptionSetDescription("[cyan]Storing checkins...[reset]"),
						progressbar.OptionSetTheme(progressbar.Theme{
							Saucer:        "[green]=[reset]",
							SaucerHead:    "[green]>[reset]",
							SaucerPadding: " ",
							BarStart:      "[",
							BarEnd:        "]",
						}))
				}

				// Store checkins in database
				if verbose {
					fmt.Println("Storing checkins in database...")
				}

				err = dbClient.StoreCheckins(checkins)
				if err != nil {
					fmt.Println(color.RedString("Error storing checkins: %v", err))
					fmt.Println(color.YellowString("Continuing with other data types..."))
				} else {
					// Update progress bar if not in verbose mode
					if !verbose && bar != nil {
						bar.Finish()
						fmt.Println()
					}

					if verbose {
						fmt.Println(color.GreenString("Successfully pulled and stored all checkins from BadgerMaps"))
					}
				}
			}

			// Step 4: Pull routes
			if verbose {
				fmt.Println(color.CyanString("\n=== Pulling routes ==="))
			} else {
				fmt.Println("Pulling routes...")
			}

			// Get all routes from API
			routes, err := apiClient.GetRoutes()
			if err != nil {
				fmt.Println(color.RedString("Error retrieving routes: %v", err))
				fmt.Println(color.YellowString("Finished pulling all available data types"))
			} else {
				if verbose {
					fmt.Printf("Found %d routes to pull\n", len(routes))
				}

				// Create progress bar if not in verbose mode
				var bar *progressbar.ProgressBar
				if !verbose {
					bar = progressbar.NewOptions(len(routes),
						progressbar.OptionEnableColorCodes(true),
						progressbar.OptionShowCount(),
						progressbar.OptionSetDescription("[cyan]Storing routes...[reset]"),
						progressbar.OptionSetTheme(progressbar.Theme{
							Saucer:        "[green]=[reset]",
							SaucerHead:    "[green]>[reset]",
							SaucerPadding: " ",
							BarStart:      "[",
							BarEnd:        "]",
						}))
				}

				// Store routes in database
				if verbose {
					fmt.Println("Storing routes in database...")
				}

				err = dbClient.StoreRoutes(routes)
				if err != nil {
					fmt.Println(color.RedString("Error storing routes: %v", err))
				} else {
					// Update progress bar if not in verbose mode
					if !verbose && bar != nil {
						bar.Finish()
						fmt.Println()
					}

					if verbose {
						fmt.Println(color.GreenString("Successfully pulled and stored all routes from BadgerMaps"))
					}
				}
			}

			fmt.Println(color.GreenString("\nFinished pulling all data types"))
		},
	}
	return cmd
}
