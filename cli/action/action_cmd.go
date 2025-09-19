package action

import (
	"github.com/spf13/cobra"
)

// ActionCmd represents the action command
var ActionCmd = &cobra.Command{
	Use:   "action",
	Short: "Manage event actions",
	Long:  `Manage event actions.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
