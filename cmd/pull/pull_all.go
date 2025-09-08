package pull

import (
	"badgermaps/app"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func PullAllCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Pull all data from BadgerMaps",
		Long:  `Pull all accounts, checkins, routes, and the user profile from the BadgerMaps API and store them in the local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.CyanString("Pulling all data..."))

			fmt.Println(color.CyanString("Pulling accounts..."))
			PullAllAccounts(App, 0)

			fmt.Println(color.CyanString("Pulling checkins..."))
			PullAllCheckins(App)

			fmt.Println(color.CyanString("Pulling routes..."))
			PullAllRoutes(App)

			fmt.Println(color.CyanString("Pulling user profile..."))
			profile, err := App.API.GetUserProfile()
			if err != nil {
				fmt.Println(color.RedString("Error pulling user profile: %v", err))
			} else {
				if err := StoreProfile(App, profile); err != nil {
					fmt.Println(color.RedString("Error storing profile: %v", err))
				} else {
					fmt.Println(color.GreenString("Successfully pulled user profile for: %s", profile.Email))
				}
			}

			fmt.Println(color.GreenString("Finished pulling all data."))
		},
	}
	return cmd
}
