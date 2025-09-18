package action

import (
	"badgermaps/app"
	"badgermaps/events"
	"fmt"

	"github.com/spf13/cobra"
)

func ActionCmd(a *app.App) *cobra.Command {
	longDescription := `Configure actions to be taken when specific events occur.
Actions are configured in the config.yaml file.
`
	cmd := &cobra.Command{
		Use:   "action",
		Short: "Manage event-driven actions",
		Long:  longDescription,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	cmd.AddCommand(listEventsCmd(a))
	cmd.AddCommand(validateEventsCmd(a))

	return cmd
}

func listEventsCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available events",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Available events:")
			for _, eventType := range events.AllEventTypes() {
				fmt.Printf("- %s\n", eventType)
			}
		},
	}

	return cmd
}

func validateEventsCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the event actions in the configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Events.Dispatch(events.Infof("action", "Validating event actions..."))

			if a.Config.EventActions == nil {
				a.Events.Dispatch(events.Warningf("action", "No event actions found in configuration."))
				return nil
			}

			valid := true
			for _, eventAction := range a.Config.EventActions {
				a.Events.Dispatch(events.Infof("action", "Event: %s (Source: %s)", eventAction.Event, eventAction.Source))
				for i, config := range eventAction.Run {
					action, err := events.NewActionFromConfig(config)
					if err != nil {
						a.Events.Dispatch(events.Errorf("action", "  - Action %d: Error creating action: %v", i+1, err))
						valid = false
						continue
					}
					if err := action.Validate(); err != nil {
						a.Events.Dispatch(events.Errorf("action", "  - Action %d: Invalid configuration: %v", i+1, err))
						valid = false
					} else {
						a.Events.Dispatch(events.Infof("action", "  - Action %d (%s): OK", i+1, config.Type))
					}
				}
			}

			if valid {
				a.Events.Dispatch(events.Infof("action", "\nAll event actions are valid."))
			} else {
				return fmt.Errorf("\none or more event actions are invalid")
			}

			return nil
		},
	}
	return cmd
}
