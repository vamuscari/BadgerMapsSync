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
func storeAccountBasic(App *app.State, acc api.Account) error {
	// merge basic account fields
	first := ""
	if acc.FirstName != nil {
		first = *acc.FirstName
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
			name = *loc.Name
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
func storeAccountDetailed(App *app.State, acc *api.Account) error {
	valStr := func(p *string) any {
		if p == nil {
			return nil
		}
		return *p
	}
	valF := func(p *float64) any {
		if p == nil {
			return nil
		}
		return *p
	}

	args := []any{
		acc.ID,
		valStr(acc.FirstName),
		acc.LastName,
		acc.FullName,
		acc.PhoneNumber,
		acc.Email,
		valStr(acc.AccountOwner),
		valStr(acc.CustomerID),
		valStr(acc.Notes),
		acc.OriginalAddress,
		valStr(acc.CRMID),
		acc.DaysSinceLastCheckin,
		valStr(acc.FollowUpDate),
		valStr(acc.LastCheckinDate),
		valStr(acc.LastModifiedDate),
		valF(acc.CustomNumeric),
		valStr(acc.CustomText),
		valF(acc.CustomNumeric2),
		valStr(acc.CustomText2),
		valF(acc.CustomNumeric3),
		valStr(acc.CustomText3),
		valF(acc.CustomNumeric4),
		valStr(acc.CustomText4),
		valF(acc.CustomNumeric5),
		valStr(acc.CustomText5),
		valF(acc.CustomNumeric6),
		valStr(acc.CustomText6),
		valF(acc.CustomNumeric7),
		valStr(acc.CustomText7),
		valF(acc.CustomNumeric8),
		valStr(acc.CustomText8),
		valF(acc.CustomNumeric9),
		valStr(acc.CustomText9),
		valF(acc.CustomNumeric10),
		valStr(acc.CustomText10),
		valF(acc.CustomNumeric11),
		valStr(acc.CustomText11),
		valF(acc.CustomNumeric12),
		valStr(acc.CustomText12),
		valF(acc.CustomNumeric13),
		valStr(acc.CustomText13),
		valF(acc.CustomNumeric14),
		valStr(acc.CustomText14),
		valF(acc.CustomNumeric15),
		valStr(acc.CustomText15),
		valF(acc.CustomNumeric16),
		valStr(acc.CustomText16),
		valF(acc.CustomNumeric17),
		valStr(acc.CustomText17),
		valF(acc.CustomNumeric18),
		valStr(acc.CustomText18),
		valF(acc.CustomNumeric19),
		valStr(acc.CustomText19),
		valF(acc.CustomNumeric20),
		valStr(acc.CustomText20),
		valF(acc.CustomNumeric21),
		valStr(acc.CustomText21),
		valF(acc.CustomNumeric22),
		valStr(acc.CustomText22),
		valF(acc.CustomNumeric23),
		valStr(acc.CustomText23),
		valF(acc.CustomNumeric24),
		valStr(acc.CustomText24),
		valF(acc.CustomNumeric25),
		valStr(acc.CustomText25),
		valF(acc.CustomNumeric26),
		valStr(acc.CustomText26),
		valF(acc.CustomNumeric27),
		valStr(acc.CustomText27),
		valF(acc.CustomNumeric28),
		valStr(acc.CustomText28),
		valF(acc.CustomNumeric29),
		valStr(acc.CustomText29),
		valF(acc.CustomNumeric30),
		valStr(acc.CustomText30),
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
			name = *loc.Name
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

func storeCheckin(App *app.State, c api.Checkin) error {
	// SQLite expects Type TEXT; other DBs have corresponding SQL. Pass as-is.
	crm := ""
	if c.CRMID != nil {
		crm = *c.CRMID
	}
	extra := ""
	if c.ExtraFields != nil {
		extra = *c.ExtraFields
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

func storeRoute(App *app.State, r api.Route) error {
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
			suite = *w.Suite
		}
		city := ""
		if w.City != nil {
			city = *w.City
		}
		state := ""
		if w.State != nil {
			state = *w.State
		}
		zip := ""
		if w.Zipcode != nil {
			zip = *w.Zipcode
		}
		complete := ""
		if w.CompleteAddress != nil {
			complete = *w.CompleteAddress
		}
		appt := ""
		if w.ApptTime != nil {
			appt = *w.ApptTime
		}
		place := ""
		if w.PlaceID != nil {
			place = *w.PlaceID
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

func storeProfile(App *app.State, p *api.UserProfile) error {
	manager := ""
	if p.Manager != nil {
		manager = *p.Manager
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
func pullAccountCmd(App *app.State) *cobra.Command {
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

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

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

			if err := storeAccountDetailed(App, account); err != nil {
				fmt.Println(color.RedString("Error storing account: %v", err))
				os.Exit(1)
			}

			fmt.Println(color.GreenString("Successfully pulled account: %s", account.FullName))
		},
	}
	return cmd
}

// pullAccountsCmd creates a command to pull multiple accounts
func pullAccountsCmd(App *app.State) *cobra.Command {
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
				return
			}
		},
	}

	// Add flags
	cmd.Flags().IntVar(&top, "top", 0, "Limit the number of accounts to pull (0 = all)")

	return cmd
}

// PullAllAccounts pulls all accounts from the API
func PullAllAccounts(App *app.State, top int) {
	// Create API client
	apiClient := api.NewAPIClient(App.Config.APIKey)

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
	for _, account := range accounts {
		wg.Add(1)
		go func(acc api.Account) {
			defer wg.Done()

			// Acquire semaphore
			sem <- true
			defer func() { <-sem }()

			detailedAcc, err := apiClient.GetAccountDetailed(acc.ID)
			if err != nil {
				errorMsg := fmt.Sprintf("Error retrieving account %d (%s): %v", acc.ID, acc.FullName, err)
				errorsMutex.Lock()
				errors = append(errors, errorMsg)
				errorsMutex.Unlock()
				return
			}

			if err := storeAccountDetailed(App, detailedAcc); err != nil {
				fmt.Println(color.RedString("Error storing account %d: %v", acc, err))
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
func pullCheckinCmd(App *app.State) *cobra.Command {
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

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

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
			if err := storeCheckin(App, *checkin); err != nil {
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
func pullCheckinsCmd(App *app.State) *cobra.Command {
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

			// Create API client
			apiClient := api.NewAPIClient(apiKey)

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

				if err := storeCheckin(App, *checkin); err != nil {
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
func pullAllCheckins(App *app.State) {
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

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get all accounts first
	fmt.Println(color.CyanString("Retrieving accounts from BadgerMaps API..."))
	accounts, err := apiClient.GetAccounts()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving accounts: %v", err))
		os.Exit(1)
	}
	fmt.Printf(color.CyanString("Found %d accounts\n"), len(accounts))

	// Set up concurrency control
	maxParallel := App.Config.MaxParallelProcesses
	if maxParallel <= 0 {
		maxParallel = 5
	}
	sem := make(chan bool, maxParallel)
	var wg sync.WaitGroup

	// Progress bar per account processed
	bar := progressbar.Default(int64(len(accounts)))

	// Track totals and errors
	var totalStored int
	var totalStoredMutex sync.Mutex
	var errors []string
	var errorsMutex sync.Mutex

	for _, acc := range accounts {
		wg.Add(1)
		sem <- true
		go func(a api.Account) {
			defer func() {
				<-sem
				wg.Done()
			}()

			// Fetch checkins for this account
			checkins, err := apiClient.GetCheckinsForAccount(a.ID)
			if err != nil {
				errorsMutex.Lock()
				errors = append(errors, fmt.Sprintf("Error retrieving checkins for account %d (%s): %v", a.ID, a.FullName, err))
				errorsMutex.Unlock()
				bar.Add(1)
				return
			}

			stored := 0
			for _, c := range checkins {
				if err := storeCheckin(App, c); err != nil {
					errorsMutex.Lock()
					errors = append(errors, fmt.Sprintf("Error storing checkin %d for account %d: %v", c.ID, a.ID, err))
					errorsMutex.Unlock()
					continue
				}
				stored++
			}
			totalStoredMutex.Lock()
			totalStored += stored
			totalStoredMutex.Unlock()

			bar.Add(1)
		}(acc)
	}

	wg.Wait()

	if len(errors) > 0 {
		fmt.Println(color.RedString("\nErrors encountered during checkin pull:"))
		for _, e := range errors {
			fmt.Println(color.RedString("- %s", e))
		}
	}

	fmt.Println(color.GreenString("\nSuccessfully pulled and stored %d checkins across %d accounts", totalStored, len(accounts)))
}

// pullRouteCmd creates a command to pull a single route
func pullRouteCmd(App *app.State) *cobra.Command {
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
func pullRoutesCmd(App *app.State) *cobra.Command {
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
func PullRoute(routeID int, App *app.State) error {
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

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get route from API
	route, err := apiClient.GetRoute(routeID)
	if err != nil {
		return fmt.Errorf("error retrieving route: %w", err)
	}

	// Store route and its waypoints
	if err := storeRoute(App, *route); err != nil {
		return fmt.Errorf("error storing route: %w", err)
	}

	return nil
}

// PullAllRoutes pulls all routes from the API
func PullAllRoutes(App *app.State) {
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

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Get all routes from API
	fmt.Println(color.CyanString("Retrieving routes from BadgerMaps API..."))
	routes, err := apiClient.GetRoutes()
	if err != nil {
		fmt.Println(color.RedString("Error retrieving routes: %v", err))
		os.Exit(1)
	}

	fmt.Printf(color.CyanString("Found %d routes\n"), len(routes))

	// Store routes one by one (with waypoints)
	for _, r := range routes {
		if err := storeRoute(App, r); err != nil {
			fmt.Println(color.RedString("Error storing route %d: %v", r.ID, err))
		}
	}

	if App.Verbose {
		fmt.Printf("Pulled and stored %d routes\n", len(routes))
	}

	fmt.Println(color.GreenString("Successfully pulled and stored all routes"))
}

// pullProfileCmd creates a command to pull the user profile
func pullProfileCmd(App *app.State) *cobra.Command {
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
			if err := storeProfile(App, profile); err != nil {
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
func pullAllCmd(App *app.State) *cobra.Command {
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
			apiClient := api.NewAPIClient(apiKey)

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
				if err := storeProfile(App, profile); err != nil {
					errors = append(errors, fmt.Sprintf("Error storing user profile: %v", err))
				} else if App.Verbose {
					fmt.Println(color.GreenString("Successfully pulled and stored user profile"))
				}
			}

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Accounts ==="))
			}
			PullAllAccounts(App, 0)

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Checkins ==="))
			}
			pullAllCheckins(App)

			if App.Verbose {
				fmt.Println(color.CyanString("\n=== Pulling Routes ==="))
			}
			PullAllRoutes(App)

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
