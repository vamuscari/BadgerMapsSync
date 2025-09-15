var rootCmd = &cobra.Command{
	Use:   "badgermaps",
	Short: "A CLI for interacting with the BadgerMaps API",
	Run: func(cmd *cobra.Command, args []string) {
		// If no command is specified, and GUI is available, launch GUI
		if hasGUI {
			runGUI()
		} else {
			cmd.Help()
		}
	},
}

var guiCmd = &cobra.Command{
	Use:   "gui",
	Short: "Launches the graphical user interface",
	Run: func(cmd *cobra.Command, args []string) {
		if hasGUI {
			runGUI()
		} else {
			fmt.Println("GUI not enabled in this build.")
		}
	},
}
