package pull

import (
	"badgermapscli/api"
	"badgermapscli/app"
	"badgermapscli/database"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

// PullCmd creates a new pull command
func PullCmd(App *app.Application) *cobra.Command {
	App.VerifySetupOrExit()

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Retrieve data from BadgerMaps API",
		Long:  `Pull data from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to pull (account, checkin, route, profile, all)")
			os.Exit(1)
		},
	}

	// Add subcommands
	pullCmd.AddCommand(pullAccountCmd(App))
	pullCmd.AddCommand(pullAccountsCmd(App))
	pullCmd.AddCommand(pullCheckinCmd(App))
	pullCmd.AddCommand(pullCheckinsCmd(App))
	pullCmd.AddCommand(pullRouteCmd(App))
	pullCmd.AddCommand(pullRoutesCmd(App))
	pullCmd.AddCommand(pullProfileCmd(App))
	pullCmd.AddCommand(pullAllCmd(App))

	return pullCmd
}

// pullAccountCmd creates a command to pull a single account
func pullAccountCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Pull a single account from BadgerMaps",
		Long:  `Pull a single account from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling account with ID: %s", args[0]))
			}

			// Get API key from App
			apiKey := App.APIKey
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

			// Get database Appuration
			dbConfig := &database.Config{
				DatabaseType: App.DBType,
				Host:         App.DBHost,
				Port:         App.DBPort,
				Database:     App.DBPath,
				Username:     App.DBUser,
				Password:     App.DBPassword,
			}

			// Set default database type and name if not provided
			if dbConfig.DatabaseType == "" {
				dbConfig.DatabaseType = "sqlite3" // Default to SQLite
			}
			if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
				dbConfig.Database = "badgermaps.db"
			}

			// Create database client
			dbClient, err := database.NewClient(dbConfig, App.Verbose)
			if err != nil {
				fmt.Println(color.RedString("Error creating database client: %v", err))
				os.Exit(1)
			}
			defer dbClient.Close()

			// Validate database schema
			err = dbClient.ValidateDatabaseSchema()
			if err != nil {
				fmt.Println(color.RedString("Error validating database schema: %v", err))
				os.Exit(1)
			}

			// Store account in database
			accounts := []api.Account{*account}
			err = dbClient.StoreAccounts(accounts)
			if err != nil {
				fmt.Println(color.RedString("Error storing account: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("Successfully pulled account: %s", account.FullName))
		},
	}
	return cmd
}

// pullAccountsCmd creates a command to pull multiple accounts
func pullAccountsCmd(App *app.Application) *cobra.Command {
	var top int

	cmd := &cobra.Command{
		Use:   "accounts [id...]",
		Short: "Pull multiple accounts from BadgerMaps",
		Long:  `Pull multiple accounts from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Pull all accounts
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling all accounts"))
				}
				PullAllAccounts(App, top)
			} else {
				// Pull specific accounts
				if App.Verbose {
					fmt.Printf(color.CyanString("Pulling accounts with IDs: %v\n"), args)
				}

				// Get API key from App
				apiKey := App.APIKey
				if apiKey == "" {
					fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
					os.Exit(1)
				}

				// Create API client
				apiClient := api.NewAPIClient(apiKey)

				// Get database Appuration
				dbConfig := &database.Config{
					DatabaseType: App.DBType,
					Host:         App.DBHost,
					Port:         App.DBPort,
					Database:     App.DBPath,
					Username:     App.DBUser,
					Password:     App.DBPassword,
				}

				// Set default database type and name if not provided
				if dbConfig.DatabaseType == "" {
					dbConfig.DatabaseType = "sqlite3" // Default to SQLite
				}
				if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
					dbConfig.Database = "badgermaps.db"
				}

				// Create database client
				dbClient, err := database.NewClient(dbConfig, App.Verbose)
				if err != nil {
					fmt.Println(color.RedString("Error creating database client: %v", err))
					os.Exit(1)
				}
				defer dbClient.Close()

				// Validate database schema
				err = dbClient.ValidateDatabaseSchema()
				if err != nil {
					fmt.Println(color.RedString("Error validating database schema: %v", err))
					os.Exit(1)
				}

				// Pull each account
				for _, arg := range args {
					accountID, err := strconv.Atoi(arg)
					if err != nil {
						fmt.Println(color.RedString("Invalid account ID: %s", arg))
						continue
					}

					// Get account from API
					account, err := apiClient.GetAccount(accountID)
					if err != nil {
						fmt.Println(color.RedString("Error retrieving account %d: %v", accountID, err))
						continue
					}

					// Store account in database
					accounts := []api.Account{*account}
					err = dbClient.StoreAccounts(accounts)
					if err != nil {
						fmt.Println(color.RedString("Error storing account %d: %v", accountID, err))
						continue
					}

					fmt.Println(color.GreenString("Successfully pulled account: %s", account.FullName))
				}
			}
		},
	}

	// Add flags
	cmd.Flags().IntVar(&top, "top", 0, "Limit the number of accounts to pull (0 = all)")

	return cmd
}

// PullAllAccounts pulls all accounts from the API
func PullAllAccounts(App *app.Application, top int) {
	// Get API key from App
	apiKey := App.APIKey
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get database Appuration
	dbConfig := &database.Config{
		DatabaseType: App.DBType,
		Host:         App.DBHost,
		Port:         App.DBPort,
		Database:     App.DBPath,
		Username:     App.DBUser,
		Password:     App.DBPassword,
	}

	// Set default database type and name if not provided
	if dbConfig.DatabaseType == "" {
		dbConfig.DatabaseType = "sqlite3" // Default to SQLite
	}
	if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
		dbConfig.Database = "badgermaps.db"
	}

	// Create database client
	dbClient, err := database.NewClient(dbConfig, App.Verbose)
	if err != nil {
		fmt.Println(color.RedString("Error creating database client: %v", err))
		os.Exit(1)
	}
	defer dbClient.Close()

	// Validate database schema
	err = dbClient.ValidateDatabaseSchema()
	if err != nil {
		fmt.Println(color.RedString("Error validating database schema: %v", err))
		os.Exit(1)
	}

	// Get all accounts from API
	fmt.Println(color.CyanString("Retrieving accounts from BadgerMaps API..."))
	accounts, err := apiClient.GetAccounts()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving accounts: %v", err))
		os.Exit(1)
	}

	// Apply top limit if specified
	if top > 0 && top < len(accounts) {
		accounts = accounts[:top]
	}

	fmt.Printf(color.CyanString("Found %d accounts\n"), len(accounts))

	// Create a progress bar
	bar := progressbar.Default(int64(len(accounts)))

	// Get max parallel processes from App
	maxParallel := App.MaxParallelProcesses
	if maxParallel <= 0 {
		maxParallel = 5 // Default value
	}

	// Create a semaphore to limit concurrent operations
	sem := make(chan bool, maxParallel)
	var wg sync.WaitGroup

	// Create a slice to collect errors
	var errors []string
	var errorsMutex sync.Mutex

	// Process accounts in parallel
	for _, account := range accounts {
		wg.Add(1)
		go func(acc api.Account) {
			defer wg.Done()

			// Acquire semaphore
			sem <- true
			defer func() { <-sem }()

			// Store account in database
			accounts := []api.Account{acc}
			err := dbClient.StoreAccounts(accounts)
			if err != nil {
				errorMsg := fmt.Sprintf("Error storing account %d (%s): %v", acc.ID, acc.FullName, err)
				errorsMutex.Lock()
				errors = append(errors, errorMsg)
				errorsMutex.Unlock()
			}

			// Update progress bar
			bar.Add(1)
		}(account)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Print all collected errors
	if len(errors) > 0 {
		fmt.Println(color.RedString("\nErrors encountered during account pull:"))
		for _, err := range errors {
			fmt.Println(color.RedString("- %s", err))
		}
	}

	fmt.Println(color.GreenString("\nSuccessfully pulled %d accounts", len(accounts)-len(errors)))
}

// pullCheckinCmd creates a command to pull a single checkin
func pullCheckinCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Pull a single checkin from BadgerMaps",
		Long:  `Pull a single checkin from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling checkin with ID: %s", args[0]))
			}

			// Get API key from App
			apiKey := App.APIKey
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

			// Get database Appuration
			dbConfig := &database.Config{
				DatabaseType: App.DBType,
				Host:         App.DBHost,
				Port:         App.DBPort,
				Database:     App.DBPath,
				Username:     App.DBUser,
				Password:     App.DBPassword,
			}

			// Set default database type and name if not provided
			if dbConfig.DatabaseType == "" {
				dbConfig.DatabaseType = "sqlite3" // Default to SQLite
			}
			if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
				dbConfig.Database = "badgermaps.db"
			}

			// Create database client
			dbClient, err := database.NewClient(dbConfig, App.Verbose)
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

			if App.Verbose {
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
func pullCheckinsCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins [id...]",
		Short: "Pull multiple checkins from BadgerMaps",
		Long:  `Pull multiple checkins from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling all checkins"))
				}
				pullAllCheckins(App)
			} else {
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling checkins with IDs: %v", args))
				}

				// Get API key from App
				apiKey := App.APIKey
				if apiKey == "" {
					fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
					os.Exit(1)
				}

				// Create API client
				apiClient := api.NewAPIClient(apiKey)

				// Get database Appuration
				dbConfig := &database.Config{
					DatabaseType: App.DBType,
					Host:         App.DBHost,
					Port:         App.DBPort,
					Database:     App.DBPath,
					Username:     App.DBUser,
					Password:     App.DBPassword,
				}

				// Set default database type and name if not provided
				if dbConfig.DatabaseType == "" {
					dbConfig.DatabaseType = "sqlite3" // Default to SQLite
				}
				if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
					dbConfig.Database = "badgermaps.db"
				}

				// Create database client
				dbClient, err := database.NewClient(dbConfig, App.Verbose)
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

				// Pull each checkin
				for _, arg := range args {
					checkinID, err := strconv.Atoi(arg)
					if err != nil {
						fmt.Println(color.RedString("Invalid checkin ID: %s", arg))
						continue
					}

					// Get checkin from API
					checkin, err := apiClient.GetCheckin(checkinID)
					if err != nil {
						fmt.Println(color.RedString("Error retrieving checkin %d: %v", checkinID, err))
						continue
					}

					// Store checkin in database
					checkins := []api.Checkin{*checkin}
					err = dbClient.StoreCheckins(checkins)
					if err != nil {
						fmt.Println(color.RedString("Error storing checkin %d: %v", checkinID, err))
						continue
					}

					if App.Verbose {
						fmt.Println(color.GreenString("Successfully pulled and stored checkin %d", checkinID))
					}
				}
			}
		},
	}
	return cmd
}

// pullAllCheckins pulls all checkins from the API
func pullAllCheckins(App *app.Application) {
	// Get API key from App
	apiKey := App.APIKey
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get database Appuration
	dbConfig := &database.Config{
		DatabaseType: App.DBType,
		Host:         App.DBHost,
		Port:         App.DBPort,
		Database:     App.DBPath,
		Username:     App.DBUser,
		Password:     App.DBPassword,
	}

	// Set default database type and name if not provided
	if dbConfig.DatabaseType == "" {
		dbConfig.DatabaseType = "sqlite3" // Default to SQLite
	}
	if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
		dbConfig.Database = "badgermaps.db"
	}

	// Create database client
	dbClient, err := database.NewClient(dbConfig, App.Verbose)
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

	// Get all checkins from API
	fmt.Println(color.CyanString("Retrieving checkins from BadgerMaps API..."))
	checkins, err := apiClient.GetCheckins()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving checkins: %v", err))
		os.Exit(1)
	}

	fmt.Printf(color.CyanString("Found %d checkins\n"), len(checkins))

	// Store checkins in database
	err = dbClient.StoreCheckins(checkins)
	if err != nil {
		fmt.Println(color.RedString("Error storing checkins: %v", err))
		os.Exit(1)
	}

	fmt.Println(color.GreenString("Successfully pulled and stored %d checkins", len(checkins)))
}

// pullRouteCmd creates a command to pull a single route
func pullRouteCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Pull a single route from BadgerMaps",
		Long:  `Pull a single route from the BadgerMaps API to your local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Parse route ID
			routeID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid route ID: %s", args[0]))
				os.Exit(1)
			}

			// Call the PullRoute function
			err = PullRoute(routeID, App)
			if err != nil {
				fmt.Println(color.RedString("Error: %v", err))
				os.Exit(1)
			}

			if App.Verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored route ID %d", routeID))
			}
		},
	}
	return cmd
}

// pullRoutesCmd creates a command to pull multiple routes
func pullRoutesCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes [id...]",
		Short: "Pull multiple routes from BadgerMaps",
		Long:  `Pull multiple routes from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling all routes"))
				}
				PullAllRoutes(App)
			} else {
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling routes with IDs: %v", args))
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

				// Get max parallel processes from App
				maxParallel := App.MaxParallelProcesses
				if maxParallel <= 0 {
					maxParallel = 10 // Default to 10 concurrent connections as specified
				}

				if App.Verbose {
					fmt.Printf("Using maximum of %d concurrent operations\n", maxParallel)
				}

				// Create a semaphore to limit concurrent operations
				sem := make(chan bool, maxParallel)
				var wg sync.WaitGroup

				// Create progress bar
				bar := progressbar.NewOptions(len(routeIDs),
					progressbar.OptionEnableColorCodes(true),
					progressbar.OptionShowCount(),
					progressbar.OptionSetDescription("[cyan]Retrieving and storing routes...[reset]"),
					progressbar.OptionSetTheme(progressbar.Theme{
						Saucer:        "[green]=[reset]",
						SaucerHead:    "[green]>[reset]",
						SaucerPadding: " ",
						BarStart:      "[",
						BarEnd:        "]",
					}))

				// Process routes in parallel
				var successCount int32
				var errorCount int32
				var routesMutex sync.Mutex
				var errors []string

				for _, id := range routeIDs {
					wg.Add(1)
					sem <- true // Acquire semaphore

					go func(routeID int) {
						defer func() {
							<-sem // Release semaphore
							wg.Done()
						}()

						err := PullRoute(routeID, App)

						routesMutex.Lock()
						defer routesMutex.Unlock()

						if err != nil {
							errorCount++
							errors = append(errors, fmt.Sprintf("Error pulling route %d: %v", routeID, err))
						} else {
							successCount++
						}

						bar.Add(1)
					}(id)
				}

				// Wait for all goroutines to finish
				wg.Wait()

				// Print errors
				if len(errors) > 0 {
					fmt.Println(color.RedString("\nErrors encountered:"))
					for _, err := range errors {
						fmt.Println(color.RedString("- %s", err))
					}
				}

				fmt.Printf("\nSuccessfully pulled %d/%d routes\n", successCount, len(routeIDs))
			}
		},
	}
	return cmd
}

// PullRoute pulls a single route from the API
func PullRoute(routeID int, App *app.Application) error {
	// Get API key from App
	apiKey := App.APIKey
	if apiKey == "" {
		return fmt.Errorf("API key not found. Please authenticate first with 'badgermaps auth'")
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get database Appuration
	dbConfig := &database.Config{
		DatabaseType: App.DBType,
		Host:         App.DBHost,
		Port:         App.DBPort,
		Database:     App.DBPath,
		Username:     App.DBUser,
		Password:     App.DBPassword,
	}

	// Set default database type and name if not provided
	if dbConfig.DatabaseType == "" {
		dbConfig.DatabaseType = "sqlite3" // Default to SQLite
	}
	if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
		dbConfig.Database = "badgermaps.db"
	}

	// Create database client
	dbClient, err := database.NewClient(dbConfig, App.Verbose)
	if err != nil {
		return fmt.Errorf("error creating database client: %w", err)
	}
	defer dbClient.Close()

	// Validate database schema
	err = dbClient.ValidateDatabaseSchema()
	if err != nil {
		return fmt.Errorf("database schema validation failed: %w", err)
	}

	// Get route from API
	route, err := apiClient.GetRoute(routeID)
	if err != nil {
		return fmt.Errorf("error retrieving route: %w", err)
	}

	// Store route in database
	routes := []api.Route{*route}
	err = dbClient.StoreRoutes(routes)
	if err != nil {
		return fmt.Errorf("error storing route: %w", err)
	}

	// Note: The API doesn't have a GetRouteWaypoints method and waypoints are stored
	// as part of the route in StoreRoutes, so we don't need to handle waypoints separately

	return nil
}

// PullAllRoutes pulls all routes from the API
func PullAllRoutes(App *app.Application) {
	// Get API key from App
	apiKey := App.APIKey
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get database Appuration
	dbConfig := &database.Config{
		DatabaseType: App.DBType,
		Host:         App.DBHost,
		Port:         App.DBPort,
		Database:     App.DBPath,
		Username:     App.DBUser,
		Password:     App.DBPassword,
	}

	// Set default database type and name if not provided
	if dbConfig.DatabaseType == "" {
		dbConfig.DatabaseType = "sqlite3" // Default to SQLite
	}
	if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
		dbConfig.Database = "badgermaps.db"
	}

	// Create database client
	dbClient, err := database.NewClient(dbConfig, App.Verbose)
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

	// Get all routes from API
	fmt.Println(color.CyanString("Retrieving routes from BadgerMaps API..."))
	routes, err := apiClient.GetRoutes()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving routes: %v", err))
		os.Exit(1)
	}

	fmt.Printf(color.CyanString("Found %d routes\n"), len(routes))

	// Store routes in database
	err = dbClient.StoreRoutes(routes)
	if err != nil {
		fmt.Println(color.RedString("Error storing routes: %v", err))
		os.Exit(1)
	}

	if App.Verbose {
		fmt.Printf("Pulled and stored %d routes\n", len(routes))
	}

	fmt.Println(color.GreenString("Successfully pulled and stored all routes"))
}

// pullProfileCmd creates a command to pull the user profile
func pullProfileCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Pull user profile from BadgerMaps",
		Long:  `Pull your user profile from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling user profile..."))
			}

			// Create a slice to collect errors
			var errors []string

			// Get API key from App
			apiKey := App.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

			// Get user profile from API
			profile, err := apiClient.GetUserProfile()
			if err != nil {
				errorMsg := fmt.Sprintf("Error retrieving user profile: %v", err)
				errors = append(errors, errorMsg)
				// Return early as we can't proceed without a profile
				fmt.Println(color.RedString(errorMsg))
				return
			}

			// Get database Appuration
			dbConfig := &database.Config{
				DatabaseType: App.DBType,
				Host:         App.DBHost,
				Port:         App.DBPort,
				Database:     App.DBPath,
				Username:     App.DBUser,
				Password:     App.DBPassword,
			}

			// Set default database type and name if not provided
			if dbConfig.DatabaseType == "" {
				dbConfig.DatabaseType = "sqlite3" // Default to SQLite
			}
			if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
				dbConfig.Database = "badgermaps.db"
			}

			// Create database client
			dbClient, err := database.NewClient(dbConfig, App.Verbose)
			if err != nil {
				errorMsg := fmt.Sprintf("Error creating database client: %v", err)
				errors = append(errors, errorMsg)
				fmt.Println(color.RedString(errorMsg))
				return
			}
			defer dbClient.Close()

			// Validate database schema
			err = dbClient.ValidateDatabaseSchema()
			if err != nil {
				errorMsg := fmt.Sprintf("Database schema validation failed: %v", err)
				errors = append(errors, errorMsg)
				fmt.Println(color.RedString(errorMsg))
				fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
				return
			}

			// Store profile in database
			err = dbClient.StoreProfiles(profile)
			if err != nil {
				errorMsg := fmt.Sprintf("Error storing user profile: %v", err)
				errors = append(errors, errorMsg)
				fmt.Println(color.RedString(errorMsg))
				return
			}

			// Print all collected errors
			if len(errors) > 0 {
				fmt.Println(color.RedString("\nErrors encountered during profile retrieval:"))
				for _, err := range errors {
					fmt.Println(color.RedString("- %s", err))
				}
			}

			if App.Verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				fmt.Printf("Profile ID: %d\n", profile.ID)
				fmt.Printf("Name: %s %s\n", profile.FirstName, profile.LastName)
				fmt.Printf("Email: %s\n", profile.Email)
				fmt.Printf("Company: %s\n", profile.Company.Name)
			} else {
				fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				fmt.Printf("Profile ID: %d\n", profile.ID)
			}
		},
	}
	return cmd
}

// pullAllCmd creates a command to pull all data types in order
func pullAllCmd(App *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Pull all data from BadgerMaps",
		Long:  `Pull all data types from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling all data from BadgerMaps..."))
			}

			// Pull user profile
			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling User Profile ==="))
			}

			// Create a slice to collect errors
			var errors []string

			// Get API key from App
			apiKey := App.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

			// Get database Appuration
			dbConfig := &database.Config{
				DatabaseType: App.DBType,
				Host:         App.DBHost,
				Port:         App.DBPort,
				Database:     App.DBPath,
				Username:     App.DBUser,
				Password:     App.DBPassword,
			}

			// Set default database type and name if not provided
			if dbConfig.DatabaseType == "" {
				dbConfig.DatabaseType = "sqlite3" // Default to SQLite
			}
			if dbConfig.DatabaseType == "sqlite3" && dbConfig.Database == "" {
				dbConfig.Database = "badgermaps.db"
			}

			// Create database client
			dbClient, err := database.NewClient(dbConfig, App.Verbose)
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

			// Pull user profile
			profile, err := apiClient.GetUserProfile()
			if err != nil {
				errorMsg := fmt.Sprintf("Error retrieving user profile: %v", err)
				errors = append(errors, errorMsg)
				fmt.Println(color.RedString(errorMsg))
			} else {
				// Store profile in database
				err = dbClient.StoreProfiles(profile)
				if err != nil {
					errorMsg := fmt.Sprintf("Error storing user profile: %v", err)
					errors = append(errors, errorMsg)
					fmt.Println(color.RedString(errorMsg))
				} else if App.Verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				}
			}

			// Pull all accounts
			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Accounts ==="))
			}
			PullAllAccounts(App, 0)

			// Pull all checkins
			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Checkins ==="))
			}
			pullAllCheckins(App)

			// Pull all routes
			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Routes ==="))
			}
			PullAllRoutes(App)

			// Print summary
			if len(errors) > 0 {
				fmt.Println(color.RedString("\nErrors encountered during data pull:"))
				for _, err := range errors {
					fmt.Println(color.RedString("- %s", err))
				}
			}

			fmt.Println(color.GreenString("\nSuccessfully pulled all data from BadgerMaps"))
		},
	}
	return cmd
}
