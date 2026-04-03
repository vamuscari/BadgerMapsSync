package action

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ActionCmd represents the action command
var ActionCmd = &cobra.Command{
	Use:   "action",
	Short: "Deprecated action command",
	Long:  `Deprecated. Global event action configuration has been removed; use workflow steps in Jobs or workflow profiles.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("action command is deprecated: configure actions as workflow steps under jobs/workflow_profiles")
	},
}
