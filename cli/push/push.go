package push

import (
	"badgermaps/app"
	"os"

	"github.com/spf13/cobra"
)

// PushCmd creates a new push command
func PushCmd(App *app.App) *cobra.Command {
	presenter := NewCliPresenter(App)

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	pushCmd.AddCommand(pushAccountsCmd(presenter))
	pushCmd.AddCommand(pushCheckinsCmd(presenter))
	pushCmd.AddCommand(pushAllCmd(presenter))
	pushCmd.AddCommand(listCmd(presenter))
	return pushCmd
}

func pushAccountsCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Push pending account changes to BadgerMaps",
		Long:  `Push pending account changes from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePushAccounts()
		},
	}
	return cmd
}

func pushCheckinsCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkins",
		Short: "Push pending check-in changes to BadgerMaps",
		Long:  `Push pending check-in changes from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePushCheckins()
		},
	}
	return cmd
}

func pushAllCmd(presenter *CliPresenter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Push all pending changes to BadgerMaps",
		Long:  `Push all pending changes (accounts and check-ins) from your local database to the BadgerMaps API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandlePushAll()
		},
	}
	return cmd
}

