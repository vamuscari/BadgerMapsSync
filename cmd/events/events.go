package events

import (
	"badgermaps/app"
	"fmt"

	"github.com/spf13/cobra"
)

func EventsCmd(a *app.App) *cobra.Command {
	longDescription := `Configure actions to be taken when specific events occur.

Available Events:
`
	for _, eventType := range app.AllEventTypes() {
		longDescription += fmt.Sprintf("  - %s\n", eventType)
	}

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Manage event-driven actions",
		Long:  longDescription,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}


	cmd.AddCommand(listEventsCmd(a))
	cmd.AddCommand(onEventCmd(a))

	return cmd
}

func listEventsCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available events",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available events:")
			for i := 0; i < int(app.PushComplete)+1; i++ {
				fmt.Printf("- %s\n", app.EventType(i).String())
			}
		},
	}

	return cmd
}

func onEventCmd(a *app.App) *cobra.Command {
	var dbFunction string

	cmd := &cobra.Command{
		Use:   "on [event]",
		Short: "Register an action for a specific event",
		Long: `Register an action to be taken when a specific event occurs.
Provide a shell command as an argument or specify a database function to execute.`, 
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			event := args[0]
			action, _ := cmd.Flags().GetString("action")

			if action != "" && dbFunction != "" {
				return fmt.Errorf("cannot use --action and --db-function at the same time")
			}
			if action == "" && dbFunction == "" {
				return fmt.Errorf("you must specify an action using --action or --db-function")
			}

			var finalAction string
			if dbFunction != "" {
				finalAction = "db:" + dbFunction
			} else {
				finalAction = action
			}

			if err := a.AddEventAction(event, finalAction); err != nil {
				return err
			}

			fmt.Printf("Action '%s' registered for event '%s'\n", finalAction, event)
			return nil
		},
	}

	cmd.Flags().String("action", "", "Shell command to execute when the event occurs.")
	cmd.Flags().StringVar(&dbFunction, "db-function", "", "Database function to execute when the event occurs.")

	return cmd
}

