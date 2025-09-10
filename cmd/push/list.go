package push

import (
	"badgermaps/app"
	"badgermaps/database"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func listCmd(App *app.App) *cobra.Command {
	var status, entityType, date, orderBy string
	var accountID int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pending push changes with filtering and sorting",
		Long:  `Displays a list of pending data changes (pushes) that have not yet been sent to the BadgerMaps API. You can filter by status, entity type, date, and account, as well as control the sort order.`, 
		RunE: func(cmd *cobra.Command, args []string) error {
			options := app.PushFilterOptions{
				Status:    status,
				AccountID: accountID,
				OrderBy:   orderBy,
			}

			// Only include date if it's provided
			if date != "" {
				_, err := time.Parse("2006-01-02", date)
				if err != nil {
					return fmt.Errorf("invalid date format, please use YYYY-MM-DD")
				}
				options.Date = date
			}

			results, err := app.GetFilteredPendingChanges(App, entityType, options)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer w.Flush()

			switch entityType {
			case "accounts":
				changes, ok := results.([]database.AccountPendingChange)
				if !ok {
					return fmt.Errorf("unexpected type for account changes")
				}
				if len(changes) == 0 {
					fmt.Println("No pending account changes found.")
					return nil
				}
				fmt.Fprintln(w, "ID\tAccount ID\tType\tStatus\tCreated At\tChanges")
				for _, c := range changes {
					fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\n", c.ChangeId, c.AccountId, c.ChangeType, c.Status, c.CreatedAt.Format(time.RFC3339), c.Changes)
				}
			case "checkins":
				changes, ok := results.([]database.CheckinPendingChange)
				if !ok {
					return fmt.Errorf("unexpected type for checkin changes")
				}
				if len(changes) == 0 {
					fmt.Println("No pending check-in changes found.")
					return nil
				}
				fmt.Fprintln(w, "ID\tCheckin ID\tAccount ID\tType\tStatus\tCreated At\tChanges")
				for _, c := range changes {
					fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t%s\t%s\n", c.ChangeId, c.CheckinId, c.AccountId, c.ChangeType, c.Status, c.CreatedAt.Format(time.RFC3339), c.Changes)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&status, "status", "pending", "Filter by status (e.g., pending, processing, completed, failed)")
	cmd.Flags().StringVarP(&entityType, "type", "t", "accounts", "Type of entity to list (accounts or checkins)")
	cmd.Flags().StringVar(&date, "date", "", "Filter by creation date (YYYY-MM-DD)")
	cmd.Flags().IntVarP(&accountID, "account", "a", 0, "Filter by Account ID")
	cmd.Flags().StringVar(&orderBy, "order-by", "date_desc", "Sort order (e.g., date, date_desc, status, type, account)")

	return cmd
}
