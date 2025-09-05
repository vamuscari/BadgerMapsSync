package server

import (
	"bufio"
	"fmt"
	"os"

	"badgermaps/app"
	"badgermaps/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newServerSetupCmd(App *app.State) *cobra.Command {
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

func interactiveServerSetup(App *app.State) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("--- Server Setup ---"))

	// Server Host
	App.Config.ServerHost = utils.PromptString(reader, "Server Host", App.Config.ServerHost)
	viper.Set("SERVER_HOST", App.Config.ServerHost)

	// Server Port
	App.Config.ServerPort = utils.PromptInt(reader, "Server Port", App.Config.ServerPort)
	viper.Set("SERVER_PORT", App.Config.ServerPort)

	// Enable TLS
	App.Config.ServerTLSEnable = utils.PromptBool(reader, "Enable TLS/HTTPS", App.Config.ServerTLSEnable)
	viper.Set("SERVER_TLS_ENABLED", App.Config.ServerTLSEnable)

	if App.Config.ServerTLSEnable {
		// TLS Cert and Key
		App.Config.ServerTLSCert = utils.PromptString(reader, "TLS Certificate File", App.Config.ServerTLSCert)
		viper.Set("SERVER_TLS_CERT", App.Config.ServerTLSCert)
		App.Config.ServerTLSKey = utils.PromptString(reader, "TLS Key File", App.Config.ServerTLSKey)
		viper.Set("SERVER_TLS_KEY", App.Config.ServerTLSKey)
	}

	// Webhook Secret
	App.Config.WebhookSecret = utils.PromptString(reader, "Webhook Secret", App.Config.WebhookSecret)
	viper.Set("WEBHOOK_SECRET", App.Config.WebhookSecret)

	// Save the configuration
	if err := viper.WriteConfig(); err != nil {
		// If the config file doesn't exist, try to save it to the default path.
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			if err = viper.SafeWriteConfig(); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
