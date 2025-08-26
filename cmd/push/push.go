package push

import (
	"badgermapscli/api"
	"badgermapscli/app"
	"badgermapscli/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/microsoft/go-mssqldb"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// PushCmd creates a new push command
func PushCmd(App *app.State) *cobra.Command {
	App.VerifySetupOrExit()

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Send data to BadgerMaps API",
		Long:  `Push data from your local database to the BadgerMaps API.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Please specify a data type to push (account, accounts)")
			os.Exit(1)
		},
	}

	// Add subcommands
	pushCmd.AddCommand(pushAccountCmd(App))
	pushCmd.AddCommand(pushAccountsCmd(App))

	return pushCmd
}

// openSQL opens a database/sql connection using the application's DB config.
func openSQL(App *app.State) (*sql.DB, error) {
	driver := App.DB.GetType()
	conn := App.DB.DatabaseConnection()
	return sql.Open(driver, conn)
}

// ensureAccountsTables ensures the Accounts and AccountsPendingChanges tables exist.
func ensureAccountsTables(App *app.State) error {
	if err := App.DB.EnforceSchema(); err != nil {
		return err
	}
	if err := App.DB.ValidateSchema(); err != nil {
		return err
	}
	// Also ensure the pending changes table exists.
	if err := database.RunCommand(App.DB, "create_accounts_pending_changes_table"); err != nil {
		// This might fail if the file doesn't exist for other DBs, so we don't fail hard.
		if App.Verbose {
			fmt.Println(color.YellowString("Could not ensure AccountsPendingChanges table exists: %v", err))
		}
	}
	return nil
}

// processAccountChange processes a single account change.
func processAccountChange(App *app.State, apiClient *api.APIClient, change map[string]any) error {
	changeID, _ := change["ChangeId"].(int64)
	accountID, _ := change["AccountId"].(int64)
	changeType, _ := change["ChangeType"].(string)
	changesJSON, _ := change["Changes"].(string)

	var err error
	switch strings.ToLower(changeType) {
	case "create":
		var data map[string]string
		if err = json.Unmarshal([]byte(changesJSON), &data); err == nil {
			_, err = apiClient.CreateAccount(data)
		}
	case "update":
		var data map[string]string
		if err = json.Unmarshal([]byte(changesJSON), &data); err == nil {
			_, err = apiClient.UpdateAccount(int(accountID), data)
		}
	case "delete":
		err = apiClient.DeleteAccount(int(accountID))
	default:
		err = fmt.Errorf("unknown change type: %s", changeType)
	}

	dbx, openErr := openSQL(App)
	if openErr != nil {
		return fmt.Errorf("failed to open database to update change status: %w", openErr)
	}
	defer dbx.Close()

	status := "completed"
	if err != nil {
		status = "failed"
	}

	_, updateErr := dbx.Exec("UPDATE AccountsPendingChanges SET Status = ?, ProcessedAt = CURRENT_TIMESTAMP WHERE ChangeId = ?", status, changeID)
	if updateErr != nil {
		return fmt.Errorf("failed to update change status for ChangeId %d: %w", changeID, updateErr)
	}

	if err != nil {
		return fmt.Errorf("error processing change %d: %w", changeID, err)
	}

	if App.Verbose {
		fmt.Println(color.GreenString("Successfully processed change %d for account %d", changeID, accountID))
	}

	return nil
}

// fetchPendingAccountChanges fetches pending account changes from the database.
func fetchPendingAccountChanges(App *app.State, changeID int) ([]map[string]any, error) {
	if err := ensureAccountsTables(App); err != nil {
		return nil, err
	}
	dbx, err := openSQL(App)
	if err != nil {
		return nil, err
	}
	defer dbx.Close()

	query := "SELECT ChangeId, AccountId, ChangeType, Changes FROM AccountsPendingChanges WHERE Status = 'pending'"
	if changeID > 0 {
		query += " AND ChangeId = ?"
	}

	var rows *sql.Rows
	if changeID > 0 {
		rows, err = dbx.Query(query, changeID)
	} else {
		rows, err = dbx.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		scans := make([]any, len(cols))
		for i := range vals {
			scans[i] = &vals[i]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, err
		}
		rowMap := make(map[string]any)
		for i, c := range cols {
			rowMap[c] = vals[i]
		}
		result = append(result, rowMap)
	}
	return result, nil
}

// pushAccountCmd pushes a single pending account change.
func pushAccountCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [changeId]",
		Short: "Push a single pending account change",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			changeID, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(color.RedString("Invalid ChangeId: %s", args[0]))
				os.Exit(1)
			}

			apiKey := App.Config.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}
			apiClient := api.NewAPIClient(apiKey)

			changes, err := fetchPendingAccountChanges(App, changeID)
			if err != nil {
				fmt.Println(color.RedString("Error fetching pending change: %v", err))
				os.Exit(1)
			}

			if len(changes) == 0 {
				fmt.Println(color.YellowString("No pending change found with ChangeId: %d", changeID))
				return
			}

			if err := processAccountChange(App, apiClient, changes[0]); err != nil {
				fmt.Println(color.RedString("Error pushing account change: %v", err))
				os.Exit(1)
			}
		},
	}
	return cmd
}

// pushAccountsCmd pushes all pending account changes.
func pushAccountsCmd(App *app.State) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Push all pending account changes",
		Run: func(cmd *cobra.Command, args []string) {
			apiKey := App.Config.APIKey
			if apiKey == "" {
				fmt.Println(color.RedString("API key not found. Please authenticate first with 'badgermaps auth'"))
				os.Exit(1)
			}
			apiClient := api.NewAPIClient(apiKey)

			changes, err := fetchPendingAccountChanges(App, 0)
			if err != nil {
				fmt.Println(color.RedString("Error fetching pending changes: %v", err))
				os.Exit(1)
			}

			if len(changes) == 0 {
				fmt.Println(color.YellowString("No pending account changes found"))
				return
			}

			maxParallel := App.Config.MaxParallelProcesses
			if maxParallel <= 0 {
				maxParallel = 5
			}
			sem := make(chan struct{}, maxParallel)
			var wg sync.WaitGroup

			for _, change := range changes {
				wg.Add(1)
				sem <- struct{}{}
				go func(c map[string]any) {
					defer func() { <-sem; wg.Done() }()
					if err := processAccountChange(App, apiClient, c); err != nil {
						fmt.Println(color.RedString("Error pushing account change: %v", err))
					}
				}(change)
			}
			wg.Wait()
			fmt.Println(color.GreenString("Finished pushing pending account changes"))
		},
	}
	return cmd
}
