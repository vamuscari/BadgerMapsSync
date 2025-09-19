package pull

import (
	"badgermaps/app"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

// PullCmd creates a new pull command
func PullCmd(App *app.App) *cobra.Command {
	presenter := NewCliPresenter(App)

	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Retrieve data from BadgerMaps API",
		Long:  `Pull data from the BadgerMaps API to your local database.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	pullCmd.AddCommand(pullAccountCmd(presenter))
	pullCmd.AddCommand(pullAccountsCmd(presenter))
	pullCmd.AddCommand(pullCheckinCmd(presenter))
	pullCmd.AddCommand(pullCheckinsCmd(presenter))
	pullCmd.AddCommand(pullRouteCmd(presenter))
	pullCmd.AddCommand(pullRoutesCmd(presenter))
	pullCmd.AddCommand(pullProfileCmd(presenter))
	pullCmd.AddCommand(PullAllCmd(App)) // Assuming PullAllCmd will be refactored similarly

	return pullCmd
}

func pullAccountCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [id]",
		Short: "Pull a single account from BadgerMaps",
		Long:  `Pull a single account from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid account ID: %s", args[0])
			}
			return presenter.HandlePullAccount(accountID)
		},
	}
	return cmd
}

func pullAccountsCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Pull all accounts from BadgerMaps",
		Long:  `Pull all accounts from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePullAccounts()
		},
	}
	return cmd
}

func pullCheckinCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkin [id]",
		Short: "Pull a single checkin from BadgerMaps",
		Long:  `Pull a single checkin from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			checkinID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid checkin ID: %s", args[0])
			}
			return presenter.HandlePullCheckin(checkinID)
		},
	}
	return cmd
}

func pullCheckinsCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Pull all checkins from BadgerMaps",
		Long:  `Pull all checkins from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePullCheckins()
		},
	}
	return cmd
}

func pullRouteCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route [id]",
		Short: "Pull a single route from BadgerMaps",
		Long:  `Pull a single route from the BadgerMaps API and store it in the local database.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			routeID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid route ID: %s", args[0])
			}
			return presenter.HandlePullRoute(routeID)
		},
	}
	return cmd
}

func pullRoutesCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Pull all routes from BadgerMaps",
		Long:  `Pull all routes from the BadgerMaps API and store them in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePullRoutes()
		},
	}
	return cmd
}

func pullProfileCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Pull user profile from BadgerMaps",
		Long:  `Pull the user profile from the BadgerMaps API and store it in the local database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePullProfile()
		},
	}
	return cmd
}
