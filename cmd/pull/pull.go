package pull

import (
	"badgermaps/api"
	"badgermaps/app"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

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

	pullCmd.AddCommand(pullAccountCmd(App))
	pullCmd.AddCommand(pullAccountsCmd(App))
	pullCmd.AddCommand(pullCheckinCmd(App))
	pullCmd.AddCommand(pullCheckinsCmd(App))
	pullCmd.AddCommand(pullRouteCmd(App))
	pullCmd.AddCommand(pullRoutesCmd(App))
	pullCmd.AddCommand(pullProfileCmd(App))
	pullCmd.AddCommand(PullAllCmd(App))

	return pullCmd
}

func pullAccountCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Pull a single account from BadgerMaps",
		Long:  `Pull a single account from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			accountID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid account ID: %s", args[0]))
				os.Exit(1)
			}
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling account with ID: %d", accountID))
			}
			account, err := App.API.GetAccountDetailed(accountID)
			if err != nil {
				fmt.Println(color.RedString("Error pulling account: %v", err))
				os.Exit(1)
			}
			if err := StoreAccountDetailed(App, account); err != nil {
				fmt.Println(color.RedString("Error storing account: %v", err))
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Successfully pulled account with ID: %d", accountID))
		},
	}
	return cmd
}
func pullAccountsCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Pull all accounts from BadgerMaps",
		Long:  `Pull all accounts from the BadgerMaps API and store them in the local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling all accounts..."))
			}
			PullAllAccounts(App, 0)
		},
	}
	return cmd
}
func pullCheckinCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Pull a single checkin from BadgerMaps",
		Long:  `Pull a single checkin from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			checkinID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid checkin ID: %s", args[0]))
				os.Exit(1)
			}
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling checkin with ID: %d", checkinID))
			}
			checkin, err := App.API.GetCheckin(checkinID)
			if err != nil {
				fmt.Println(color.RedString("Error pulling checkin: %v", err))
				os.Exit(1)
			}
			if err := StoreCheckin(App, *checkin); err != nil {
				fmt.Println(color.RedString("Error storing checkin: %v", err))
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Successfully pulled checkin with ID: %d", checkinID))
		},
	}
	return cmd
}
func pullCheckinsCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Pull all checkins from BadgerMaps",
		Long:  `Pull all checkins from the BadgerMaps API and store them in the local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling all checkins..."))
			}
			checkins, err := App.API.GetCheckins()
			if err != nil {
				fmt.Println(color.RedString("Error pulling checkins: %v", err))
				os.Exit(1)
			}
			for _, checkin := range checkins {
				if err := StoreCheckin(App, checkin); err != nil {
					fmt.Println(color.RedString("Error storing checkin: %v", err))
				}
			}
			fmt.Println(color.GreenString("Successfully pulled all checkins"))
		},
	}
	return cmd
}
func pullRouteCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Pull a single route from BadgerMaps",
		Long:  `Pull a single route from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			routeID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid route ID: %s", args[0]))
				os.Exit(1)
			}
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling route with ID: %d", routeID))
			}
			route, err := App.API.GetRoute(routeID)
			if err != nil {
				fmt.Println(color.RedString("Error pulling route: %v", err))
				os.Exit(1)
			}
			// TODO: Store route
			fmt.Println(color.GreenString("Successfully pulled route with ID: %d", route.ID))
		},
	}
	return cmd
}
func pullRoutesCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Pull all routes from BadgerMaps",
		Long:  `Pull all routes from the BadgerMaps API and store them in the local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling all routes..."))
			}
			PullAllRoutes(App)
		},
	}
	return cmd
}
func pullProfileCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Pull user profile from BadgerMaps",
		Long:  `Pull the user profile from the BadgerMaps API and store it in the local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			if App.Verbose {
				fmt.Println(color.CyanString("Pulling user profile..."))
			}
			profile, err := App.API.GetUserProfile()
			if err != nil {
				fmt.Println(color.RedString("Error pulling user profile: %v", err))
				os.Exit(1)
			}
			// TODO: Store profile
			fmt.Println(color.GreenString("Successfully pulled user profile for: %s", profile.Email))
		},
	}
	return cmd
}
func PullAllCmd(App *app.State) *cobra.Command {
	// ... implementation ...
	return &cobra.Command{}
}
func PullAllAccounts(App *app.State, top int) {
	accountIDs, err := App.API.GetAccountIDs()
	if err != nil {
		fmt.Println(color.RedString("Error getting account IDs: %v", err))
		os.Exit(1)
	}

	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}

	for _, id := range accountIDs {
		account, err := App.API.GetAccountDetailed(id)
		if err != nil {
			fmt.Println(color.RedString("Error getting detailed account info for ID %d: %v", id, err))
			continue
		}
		if err := StoreAccountDetailed(App, account); err != nil {
			fmt.Println(color.RedString("Error storing account %d: %v", id, err))
		}
	}
	fmt.Println(color.GreenString("Successfully pulled all accounts"))
}
func PullAllRoutes(App *app.State) {
	routes, err := App.API.GetRoutes()
	if err != nil {
		fmt.Println(color.RedString("Error getting routes: %v", err))
		os.Exit(1)
	}

	for _, route := range routes {
		// TODO: Store route
		if App.Verbose {
			fmt.Println(color.CyanString("Pulled route: %s", route.Name))
		}
	}
	fmt.Println(color.GreenString("Successfully pulled all routes"))
}
func StoreAccountDetailed(App *app.State, acc *api.Account) error {
	// TODO: Implement database storage
	if App.Verbose {
		fmt.Println(color.CyanString("Stored account: %s", acc.FullName))
	}
	return nil
}
func StoreCheckin(App *app.State, checkin api.Checkin) error {
	// TODO: Implement database storage
	if App.Verbose {
		fmt.Println(color.CyanString("Stored checkin: %d", checkin.ID))
	}
	return nil
}
