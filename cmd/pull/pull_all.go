package pull

import (
	"badgermaps/app"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func PullAllCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Pull all data from BadgerMaps",
		Long:  `Pull all accounts, checkins, routes, and the user profile from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			bar := progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling all data..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
			defer bar.Close()

			log := func(message string) {
				if strings.Contains(message, "Finished") {
					bar.Finish()
					fmt.Println(color.GreenString(message))
				} else if strings.Contains(message, "Error") {
					bar.Clear()
					fmt.Println(color.RedString(message))
				} else {
					bar.Describe(message)
				}
			}

			err := app.RunPullAll(App, log)
			if err != nil {
				return fmt.Errorf("pull all command failed")
			}
			return nil
		},
	}
	return cmd
}
