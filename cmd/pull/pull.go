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

// helper functions to store records using App.DB and database.RunCommand
func StoreAccountBasic(App *app.State, acc api.Account) error {
	// merge basic account fields
	first := ""
	if acc.FirstName != nil {
		first = acc.FirstName.String
	}
	if err := database.RunCommand(App.DB, "merge_accounts_basic", acc.ID, first, acc.LastName); err != nil {
		return err
	}
	// refresh locations: delete then insert
	if err := database.RunCommand(App.DB, "delete_account_locations", acc.ID); err != nil {
		return err
	}
	for _, loc := range acc.Locations {
		name := ""
		if loc.Name != nil {
			name = loc.Name.String
		}
		if err := database.RunCommand(App.DB, "insert_account_locations",
			loc.ID,
			acc.ID,
			loc.City,
			name,
			loc.Zipcode,
			loc.Long,
			loc.State,
			loc.Lat,
			loc.AddressLine1,
			loc.Location,
		); err != nil {
			return err
		}
	}
	return nil
}

// storeAccountDetailed stores the full account information, including all standard and custom fields, and refreshes locations.
func StoreAccountDetailed(App *app.State, acc *api.Account) error {

	args := []any{
		acc.ID,
		acc.FirstName,
		acc.LastName,
		acc.FullName,
		acc.PhoneNumber,
		acc.Email,
		acc.AccountOwner,
		acc.CustomerID,
		acc.Notes,
		acc.OriginalAddress,
		acc.CRMID,
		acc.DaysSinceLastCheckin,
		acc.FollowUpDate,
		acc.LastCheckinDate,
		acc.LastModifiedDate,
		acc.CustomNumeric,
		acc.CustomText,
		acc.CustomNumeric2,
		acc.CustomText2,
		acc.CustomNumeric3,
		acc.CustomText3,
		acc.CustomNumeric4,
		acc.CustomText4,
		acc.CustomNumeric5,
		acc.CustomText5,
		acc.CustomNumeric6,
		acc.CustomText6,
		acc.CustomNumeric7,
		acc.CustomText7,
		acc.CustomNumeric8,
		acc.CustomText8,
		acc.CustomNumeric9,
		acc.CustomText9,
		acc.CustomNumeric10,
		acc.CustomText10,
		acc.CustomNumeric11,
		acc.CustomText11,
		acc.CustomNumeric12,
		acc.CustomText12,
		acc.CustomNumeric13,
		acc.CustomText13,
		acc.CustomNumeric14,
		acc.CustomText14,
		acc.CustomNumeric15,
		acc.CustomText15,
		acc.CustomNumeric16,
		acc.CustomText16,
		acc.CustomNumeric17,
		acc.CustomText17,
		acc.CustomNumeric18,
		acc.CustomText18,
		acc.CustomNumeric19,
		acc.CustomText19,
		acc.CustomNumeric20,
		acc.CustomText20,
		acc.CustomNumeric21,
		acc.CustomText21,
		acc.CustomNumeric22,
		acc.CustomText22,
		acc.CustomNumeric23,
		acc.CustomText23,
		acc.CustomNumeric24,
		acc.CustomText24,
		acc.CustomNumeric25,
		acc.CustomText25,
		acc.CustomNumeric26,
		acc.CustomText26,
		acc.CustomNumeric27,
		acc.CustomText27,
		acc.CustomNumeric28,
		acc.CustomText28,
		acc.CustomNumeric29,
		acc.CustomText29,
		acc.CustomNumeric30,
		acc.CustomText30,
	}

	if err := database.RunCommand(App.DB, "merge_accounts_detailed", args...); err != nil {
		return err
	}

	// refresh locations
	if err := database.RunCommand(App.DB, "delete_account_locations", acc.ID); err != nil {
		return err
	}
	for _, loc := range acc.Locations {
		name := ""
		if loc.Name != nil {
			name = loc.Name.String
		}
		if err := database.RunCommand(App.DB, "insert_account_locations",
			loc.ID,
			acc.ID,
			loc.City,
			name,
			loc.Zipcode,
			loc.Long,
			loc.State,
			loc.Lat,
			loc.AddressLine1,
			loc.Location,
		); err != nil {
			return err
		}
	}
	return nil
}

func StoreCheckin(App *app.State, c api.Checkin) error {
	// SQLite expects Type TEXT; other DBs have corresponding SQL. Pass as-is.
	crm := ""
	if c.CRMID != nil {
		crm = c.CRMID.String
	}
	extra := ""
	if c.ExtraFields != nil {
		extra = c.ExtraFields.String
	}
	return database.RunCommand(App.DB, "merge_account_checkins",
		c.ID,
		crm,
		c.Customer,
		c.LogDatetime,
		c.Type,
		c.Comments,
		extra,
		c.CreatedBy,
	)
}

func StoreRoute(App *app.State, r api.Route) error {
	// upsert route
	if err := database.RunCommand(App.DB, "merge_routes",
		r.ID,
		r.Name,
		r.RouteDate,
		r.StartTime,
		r.Duration,
		r.StartAddress,
		r.DestinationAddress,
	); err != nil {
		return err
	}
	// refresh waypoints
	if err := database.RunCommand(App.DB, "delete_route_waypoints", r.ID); err != nil {
		return err
	}
	for _, w := range r.Waypoints {
		suite := ""
		if w.Suite != nil {
			suite = w.Suite.String
		}
		city := ""
		if w.City != nil {
			city = w.City.String
		}
		state := ""
		if w.State != nil {
			state = w.State.String
		}
		zip := ""
		if w.Zipcode != nil {
			zip = w.Zipcode.String
		}
		complete := ""
		if w.CompleteAddress != nil {
			complete = w.CompleteAddress.String
		}
		appt := ""
		if w.ApptTime != nil {
			appt = w.ApptTime.String
		}
		place := ""
		if w.PlaceID != nil {
			place = w.PlaceID.String
		}
		if err := database.RunCommand(App.DB, "insert_route_waypoints",
			w.ID,
			r.ID,
			w.Name,
			w.Address,
			suite,
			city,
			state,
			zip,
			w.Location,
			w.Lat,
			w.Long,
			w.LayoverMinutes,
			w.Position,
			complete,
			w.LocationID,
			w.CustomerID,
			appt,
			w.Type,
			place,
		); err != nil {
			return err
		}
	}
	return nil
}

func StoreProfile(App *app.State, p *api.UserProfile) error {
	manager := ""
	if p.Manager != nil {
		manager = p.Manager.String
	}
	return database.RunCommand(App.DB, "merge_user_profiles",
		p.ID,
		p.FirstName,
		p.LastName,
		p.Email,
		p.IsManager,
		manager,
		p.Company.ID,
		p.Company.Name,
		p.Company.ShortName,
		p.Completed,
		p.TrialDaysLeft,
		p.HasData,
		p.DefaultApptLength,
	)
}

// PullCmd creates a new pull command
func PullCmd(App *app.State) *cobra.Command {
	App.VerifySetupOrExit()

	apiClient := api.NewAPIClient(App.Config.APIKey)

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
	pullCmd.AddCommand(pullAccountCmd(App, apiClient))
	pullCmd.AddCommand(pullAccountsCmd(App, apiClient))
	pullCmd.AddCommand(pullCheckinCmd(App, apiClient))
	pullCmd.AddCommand(pullCheckinsCmd(App, apiClient))
	pullCmd.AddCommand(pullRouteCmd(App, apiClient))
	pullCmd.AddCommand(pullRoutesCmd(App, apiClient))
	pullCmd.AddCommand(pullProfileCmd(App, apiClient))
	pullCmd.AddCommand(PullAllCmd(App, apiClient))

	return pullCmd
}

// pullAccountCmd creates a command to pull a single account
func pullAccountCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
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
			apiKey := App.Config.APIKey
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

			// Get account from API
			account, err := apiClient.GetAccountDetailed(accountID)
			if err != nil {
				fmt.Println(color.RedString("Error retrieving account: %v", err))
				os.Exit(1)
			}

			// Ensure schema and store using App.DB
			if err := App.DB.EnforceSchema(); err != nil {
				fmt.Println(color.RedString("Error ensuring database schema: %v", err))
				os.Exit(1)
			}
			if err := App.DB.ValidateSchema(); err != nil {
				fmt.Println(color.RedString("Error validating database schema: %v", err))
				os.Exit(1)
			}

			if err := StoreAccountDetailed(App, account); err != nil {
				fmt.Println(color.RedString("Error storing account: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("Successfully pulled account: %s", account.FullName))
		},
	}
	return cmd
}

// pullAccountsCmd creates a command to pull multiple accounts
func pullAccountsCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
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
				PullAllAccounts(App, apiClient, top)
				return
			}

			// Pull specific accounts
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling accounts with IDs: %v", args))
			}

			// Get API key from App
			apiKey := App.Config.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Ensure schema
			if err := App.DB.EnforceSchema(); err != nil {
				fmt.Println(color.RedString("Error ensuring database schema: %v", err))
				os.Exit(1)
			}
			if err := App.DB.ValidateSchema(); err != nil {
				fmt.Println(color.RedString("Database schema validation failed: %v", err))
				fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
				os.Exit(1)
			}

			// Get max parallel processes from config
			maxParallel := App.Config.MaxParallelProcesses
			if maxParallel <= 0 {
				maxParallel = 5 // Default value
			}

			// Create a semaphore to limit concurrent operations
			sem := make(chan bool, maxParallel)
			var wg sync.WaitGroup

			// Create a slice to collect errors
			var errors []string
			var successCount int
			var accountsMutex sync.Mutex

			bar := progressbar.Default(int64(len(args)))

			// Process accounts in parallel
			for _, arg := range args {
				wg.Add(1)
				go func(idStr string) {
					defer wg.Done()

					// Acquire semaphore
					sem <- true
					defer func() { <-sem }()

					accountID, err := strconv.Atoi(idStr)
					if err != nil {
						accountsMutex.Lock()
						errors = append(errors, fmt.Sprintf("Invalid account ID: %s", idStr))
						accountsMutex.Unlock()
						bar.Add(1)
						return
					}

					detailedAcc, err := apiClient.GetAccountDetailed(accountID)
					if err != nil {
						accountsMutex.Lock()
						errors = append(errors, fmt.Sprintf("Error retrieving account %d: %v", accountID, err))
						accountsMutex.Unlock()
						bar.Add(1)
						return
					}

					if err := StoreAccountDetailed(App, detailedAcc); err != nil {
						accountsMutex.Lock()
						errors = append(errors, fmt.Sprintf("Error storing account %d: %v", accountID, err))
						accountsMutex.Unlock()
					} else {
						accountsMutex.Lock()
						successCount++
						accountsMutex.Unlock()
					}

					// Update progress bar
					bar.Add(1)
				}(arg)
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

			fmt.Printf(color.GreenString("\nSuccessfully pulled %d/%d accounts\n"), successCount, len(args))
		},
	}

	// Add flags
	cmd.Flags().IntVar(&top, "top", 0, "Limit the number of accounts to pull (0 = all)")

	return cmd
}

// PullAllAccounts pulls all accounts from the API
func PullAllAccounts(App *app.State, apiClient *api.APIClient, top int) {
	// Get all account IDs from API
	fmt.Println(color.CyanString("Retrieving account IDs from BadgerMaps API..."))
	accountIDs, err := apiClient.GetAccountIDs()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving account IDs: %v", err))
		os.Exit(1)
	}

	// Apply top limit if specified
	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}

	fmt.Printf(color.CyanString("Found %d accounts\n"), len(accountIDs))

	// Create a progress bar
	bar := progressbar.Default(int64(len(accountIDs)))

	// Get max parallel processes from config
	maxParallel := App.Config.MaxParallelProcesses
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
	for _, accountID := range accountIDs {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Acquire semaphore
			sem <- true
			defer func() { <-sem }()

			detailedAcc, err := apiClient.GetAccountDetailed(id)
			if err != nil {
				errorMsg := fmt.Sprintf("Error retrieving account %d: %v", id, err)
				errorsMutex.Lock()
				errors = append(errors, errorMsg)
				errorsMutex.Unlock()
				bar.Add(1) // Still advance the bar on error
				return
			}

			if err := StoreAccountDetailed(App, detailedAcc); err != nil {
				errorMsg := fmt.Sprintf("Error storing account %d: %v", id, err)
				errorsMutex.Lock()
				errors = append(errors, errorMsg)
				errorsMutex.Unlock()
			}

			// Update progress bar
			bar.Add(1)
		}(accountID)
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

	fmt.Println(color.GreenString("\nSuccessfully pulled %d accounts", len(accountIDs)-len(errors)))
}

// pullCheckinCmd creates a command to pull a single checkin
func pullCheckinCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
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
			apiKey := App.Config.APIKey
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

			// Get checkin from API
			checkin, err := apiClient.GetCheckin(checkinID)
			if err != nil {
				fmt.Println(color.RedString("Error retrieving checkin: %v", err))
				os.Exit(1)
			}

			// Ensure schema
			if err := App.DB.EnforceSchema(); err != nil {
				fmt.Println(color.RedString("Error ensuring database schema: %v", err))
				os.Exit(1)
			}
			if err := App.DB.ValidateSchema(); err != nil {
				fmt.Println(color.RedString("Database schema validation failed: %v", err))
				fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
				os.Exit(1)
			}

			// Store checkin in database
			if err := StoreCheckin(App, *checkin); err != nil {
				fmt.Println(color.RedString("Error storing checkin: %v", err))
				os.Exit(1)
			}

			if App.Verbose {
				fmt.Println(color.GreenString("Successfully pulled and stored checkin"))
				fmt.Printf("Checkin ID: %d\n", checkin.ID.Int64)
				fmt.Printf("Customer: %d\n", checkin.Customer.Int64)
				fmt.Printf("Type: %s\n", checkin.Type.String)
				fmt.Printf("Date: %s\n", checkin.LogDatetime.String)
			}
		},
	}
	return cmd
}

// pullCheckinsCmd creates a command to pull multiple checkins
func pullCheckinsCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins [id...]",
		Short: "Pull multiple checkins from BadgerMaps",
		Long:  `Pull multiple checkins from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling all checkins"))
				}
				pullAllCheckins(App, apiClient)
				return
			}

			if App.Verbose {
				fmt.Println(color.CyanString("Pulling checkins with IDs: %v", args))
			}

			// Get API key from App
			apiKey := App.Config.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Ensure schema
			if err := App.DB.EnforceSchema(); err != nil {
				fmt.Println(color.RedString("Error ensuring database schema: %v", err))
				os.Exit(1)
			}
			if err := App.DB.ValidateSchema(); err != nil {
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

				checkin, err := apiClient.GetCheckin(checkinID)
				if err != nil {
					fmt.Println(color.RedString("Error retrieving checkin %d: %v", checkinID, err))
					continue
				}

				if err := StoreCheckin(App, *checkin); err != nil {
					fmt.Println(color.RedString("Error storing checkin %d: %v", checkinID, err))
					continue
				}

				if App.Verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored checkin %d", checkinID))
				}
			}
		},
	}
	return cmd
}

// pullAllCheckins pulls all checkins by first retrieving accounts, then per-account checkins
func pullAllCheckins(App *app.State, apiClient *api.APIClient) {
	// Get API key from App
	apiKey := App.Config.APIKey
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Ensure schema
	if err := App.DB.EnforceSchema(); err != nil {
		fmt.Println(color.RedString("Error ensuring database schema: %v", err))
		os.Exit(1)
	}
	if err := App.DB.ValidateSchema(); err != nil {
		fmt.Println(color.RedString("Database schema validation failed: %v", err))
		fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
		os.Exit(1)
	}

	// Get all account IDs first
	fmt.Println(color.CyanString("Retrieving account IDs from BadgerMaps API..."))
	accountIDs, err := apiClient.GetAccountIDs()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving account IDs: %v", err))
		os.Exit(1)
	}
	fmt.Printf(color.CyanString("Found %d accounts\n"), len(accountIDs))

	// Set up concurrency control
	maxParallel := App.Config.MaxParallelProcesses
	if maxParallel <= 0 {
		maxParallel = 5
	}
	sem := make(chan bool, maxParallel)
	var wg sync.WaitGroup

	// Progress bar per account processed
	bar := progressbar.Default(int64(len(accountIDs)))

	// Track totals and errors
	var totalStored int
	var totalStoredMutex sync.Mutex
	var errors []string
	var errorsMutex sync.Mutex

	for _, accID := range accountIDs {
		wg.Add(1)
		sem <- true
		go func(accountID int) {
			defer func() {
				<-sem
				wg.Done()
			}()

			// Fetch checkins for this account
			checkins, err := apiClient.GetCheckinsForAccount(accountID)
			if err != nil {
				errorsMutex.Lock()
				errors = append(errors, fmt.Sprintf("Error retrieving checkins for account %d: %v", accountID, err))
				errorsMutex.Unlock()
				bar.Add(1)
				return
			}

			stored := 0
			for _, c := range checkins {
				if err := StoreCheckin(App, c); err != nil {
					errorsMutex.Lock()
					errors = append(errors, fmt.Sprintf("Error storing checkin %d for account %d: %v", c.ID.Int64, accountID, err))
					errorsMutex.Unlock()
					continue
				}
				stored++
			}
			totalStoredMutex.Lock()
			totalStored += stored
			totalStoredMutex.Unlock()

			bar.Add(1)
		}(accID)
	}

	wg.Wait()

	if len(errors) > 0 {
		fmt.Println(color.RedString("\nErrors encountered during checkin pull:"))
		for _, e := range errors {
			fmt.Println(color.RedString("- %s", e))
		}
	}

	fmt.Println(color.GreenString("\nSuccessfully pulled and stored %d checkins across %d accounts", totalStored, len(accountIDs)))
}

// pullRouteCmd creates a command to pull a single route
func pullRouteCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
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
			err = PullRoute(routeID, App, apiClient)
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
func pullRoutesCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes [id...]",
		Short: "Pull multiple routes from BadgerMaps",
		Long:  `Pull multiple routes from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if App.Verbose {
					fmt.Println(color.CyanString("Pulling all routes"))
				}
				PullAllRoutes(App, apiClient)
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
				maxParallel := App.Config.MaxParallelProcesses
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

						err := PullRoute(routeID, App, apiClient)

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
func PullRoute(routeID int, App *app.State, apiClient *api.APIClient) error {
	// Get API key from App
	apiKey := App.Config.APIKey
	if apiKey == "" {
		return fmt.Errorf("API key not found. Please authenticate first with 'badgermaps auth'")
	}

	// Ensure schema
	if err := App.DB.EnforceSchema(); err != nil {
		return fmt.Errorf("error ensuring database schema: %w", err)
	}
	if err := App.DB.ValidateSchema(); err != nil {
		return fmt.Errorf("database schema validation failed: %w", err)
	}

	// Get route from API
	route, err := apiClient.GetRoute(routeID)
	if err != nil {
		return fmt.Errorf("error retrieving route: %w", err)
	}

	// Store route and its waypoints
	if err := StoreRoute(App, *route); err != nil {
		return fmt.Errorf("error storing route: %w", err)
	}

	return nil
}

// PullAllRoutes pulls all routes from the API
func PullAllRoutes(App *app.State, apiClient *api.APIClient) {
	// Get API key from App
	apiKey := App.Config.APIKey
	if apiKey == "" {
		fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
		os.Exit(1)
	}

	// Ensure schema
	if err := App.DB.EnforceSchema(); err != nil {
		fmt.Println(color.RedString("Error ensuring database schema: %v", err))
		os.Exit(1)
	}
	if err := App.DB.ValidateSchema(); err != nil {
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

	// Get max parallel processes from config
	maxParallel := App.Config.MaxParallelProcesses
	if maxParallel <= 0 {
		maxParallel = 10 // Default value
	}

	// Create a semaphore to limit concurrent operations
	sem := make(chan bool, maxParallel)
	var wg sync.WaitGroup

	// Create a progress bar
	bar := progressbar.Default(int64(len(routes)))

	// Create a slice to collect errors
	var errors []string
	var errorsMutex sync.Mutex

	// Process routes in parallel
	for _, route := range routes {
		wg.Add(1)
		go func(r api.Route) {
			defer wg.Done()

			// Acquire semaphore
			sem <- true
			defer func() { <-sem }()

			// Store route and its waypoints
			if err := StoreRoute(App, r); err != nil {
				errorMsg := fmt.Sprintf("Error storing route %d: %v", r.ID.Int64, err)
				errorsMutex.Lock()
				errors = append(errors, errorMsg)
				errorsMutex.Unlock()
			}

			// Update progress bar
			bar.Add(1)
		}(route)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Print all collected errors
	if len(errors) > 0 {
		fmt.Println(color.RedString("\nErrors encountered during route pull:"))
		for _, err := range errors {
			fmt.Println(color.RedString("- %s", err))
		}
	}

	fmt.Println(color.GreenString("\nSuccessfully pulled %d routes", len(routes)-len(errors)))
}

// pullProfileCmd creates a command to pull the user profile
func pullProfileCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
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
			apiKey := App.Config.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Get user profile from API
			profile, err := apiClient.GetUserProfile()
			if err != nil {
				errorMsg := fmt.Sprintf("Error retrieving user profile: %v", err)
				errors = append(errors, errorMsg)
				// Return early as we can't proceed without a profile
				fmt.Println(color.RedString(errorMsg))
				return
			}

			// Ensure schema
			if err := App.DB.EnforceSchema(); err != nil {
				errorMsg := fmt.Sprintf("Error ensuring database schema: %v", err)
				errors = append(errors, errorMsg)
				fmt.Println(color.RedString(errorMsg))
				return
			}
			if err := App.DB.ValidateSchema(); err != nil {
				errorMsg := fmt.Sprintf("Database schema validation failed: %v", err)
				errors = append(errors, errorMsg)
				fmt.Println(color.RedString(errorMsg))
				fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
				return
			}

			// Store profile in database
			if err := StoreProfile(App, profile); err != nil {
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
				fmt.Printf("Profile ID: %d\n", profile.ID.Int64)
				fmt.Printf("Name: %s %s\n", profile.FirstName.String, profile.LastName.String)
				fmt.Printf("Email: %s\n", profile.Email.String)
				fmt.Printf("Company: %s\n", profile.Company.Name.String)
			} else {
				fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				fmt.Printf("Profile ID: %d\n", profile.ID.Int64)
			}
		},
	}
	return cmd
}

// PullAllCmd creates a command to pull all data types in order
func PullAllCmd(App *app.State, apiClient *api.APIClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Pull all data from BadgerMaps",
		Long:  `Pull all data types from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling all data from BadgerMaps..."))
			}

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling User Profile ==="))
			}

			var errors []string

			apiKey := App.Config.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}

			// Ensure schema
			if err := App.DB.EnforceSchema(); err != nil {
				errors = append(errors, fmt.Sprintf("Error ensuring database schema: %v", err))
			}
			if err := App.DB.ValidateSchema(); err != nil {
				errors = append(errors, fmt.Sprintf("Database schema validation failed: %v", err))
				fmt.Println(color.YellowString("Try running 'badgermaps utils init-db' to initialize the database"))
			}

			profile, err := apiClient.GetUserProfile()
			if err != nil {
				errors = append(errors, fmt.Sprintf("Error retrieving user profile: %v", err))
			} else {
				if err := StoreProfile(App, profile); err != nil {
					errors = append(errors, fmt.Sprintf("Error storing user profile: %v", err))
				}
			}

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Accounts ==="))
			}
			PullAllAccounts(App, apiClient, 0)

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Checkins ==="))
			}
			pullAllCheckins(App, apiClient)

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Routes ==="))
			}
			PullAllRoutes(App, apiClient)

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
