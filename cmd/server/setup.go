package server

import (
	"bufio"
	"fmt"
	"os"

	"badgermaps/app"
	"badgermaps/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newServerSetupCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure server settings interactively",
		Long:  `An interactive setup wizard to configure server settings like host, port, TLS, and webhook secret.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := interactiveServerSetup(App); err != nil {
				fmt.Println(color.RedString("Error during server setup: %v", err))
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Server configuration saved successfully."))
		},
	}
	return cmd
}

func interactiveServerSetup(a *app.App) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("--- Server Setup ---"))

	a.State.ServerHost = utils.PromptString(reader, "Server Host", a.State.ServerHost)
	a.State.ServerPort = utils.PromptInt(reader, "Server Port", a.State.ServerPort)
	a.State.TLSEnabled = utils.PromptBool(reader, "Enable TLS/HTTPS", a.State.TLSEnabled)

	if a.State.TLSEnabled {
		a.State.TLSCert = utils.PromptString(reader, "TLS Certificate File", a.State.TLSCert)
		a.State.TLSKey = utils.PromptString(reader, "TLS Key File", a.State.TLSKey)
	}

	a.State.WebhookToken = utils.PromptString(reader, "Webhook Secret", a.State.WebhookToken)

	return a.SaveConfig()
}
