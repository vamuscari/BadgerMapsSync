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

func newServerSetupCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure server settings interactively",
		Long:  `An interactive setup wizard to configure server settings like host, port, TLS, and webhook secret.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := interactiveServerSetup(); err != nil {
				fmt.Println(color.RedString("Error during server setup: %v", err))
				os.Exit(1)
			}
			fmt.Println(color.GreenString("Server configuration saved successfully."))
		},
	}
	return cmd
}

func interactiveServerSetup() error {
	reader := bufio.NewReader(os.Stdin)
	var serverConfig ServerConfig
	viper.Unmarshal(&serverConfig)

	fmt.Println(utils.Colors.Blue("--- Server Setup ---"))

	serverConfig.Host = utils.PromptString(reader, "Server Host", serverConfig.Host)
	viper.Set("SERVER_HOST", serverConfig.Host)

	serverConfig.Port = utils.PromptInt(reader, "Server Port", serverConfig.Port)
	viper.Set("SERVER_PORT", serverConfig.Port)

	serverConfig.TLSEnabled = utils.PromptBool(reader, "Enable TLS/HTTPS", serverConfig.TLSEnabled)
	viper.Set("SERVER_TLS_ENABLED", serverConfig.TLSEnabled)

	if serverConfig.TLSEnabled {
		serverConfig.TLSCert = utils.PromptString(reader, "TLS Certificate File", serverConfig.TLSCert)
		viper.Set("SERVER_TLS_CERT", serverConfig.TLSCert)
		serverConfig.TLSKey = utils.PromptString(reader, "TLS Key File", serverConfig.TLSKey)
		viper.Set("SERVER_TLS_KEY", serverConfig.TLSKey)
	}

	serverConfig.WebhookSecret = utils.PromptString(reader, "Webhook Secret", serverConfig.WebhookSecret)
	viper.Set("WEBHOOK_SECRET", serverConfig.WebhookSecret)

	if err := viper.WriteConfig(); err != nil {
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
