package push

import (
	"github.com/spf13/cobra"
)

func listCmd(presenter *CliPresenter) *cobra.Command {
	var status, entityType, date, orderBy string
	var accountID int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pending push changes with filtering and sorting",
		Long:  `Displays a list of pending data changes (pushes) that have not yet been sent to the BadgerMaps API. You can filter by status, entity type, date, and account, as well as control the sort order.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return presenter.HandleList(entityType, status, date, accountID, orderBy)
		},
	}

	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status (e.g., pending, processing, completed, failed)")
	cmd.Flags().StringVarP(&entityType, "type", "t", "accounts", "Type of entity to list (accounts or checkins)")
	cmd.Flags().StringVar(&date, "date", "", "Filter by creation date (YYYY-MM-DD)")
	cmd.Flags().IntVarP(&accountID, "account", "a", 0, "Filter by Account ID")
	cmd.Flags().StringVar(&orderBy, "order-by", "date_desc", "Sort order (e.g., date, date_desc, status, type, account)")

	return cmd
}
