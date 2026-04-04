package server

import (
	"badgermaps/app"
	appserver "badgermaps/app/server"
	"badgermaps/events"
	"badgermaps/utils"
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newServerSetupCmd(App *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure server settings interactively",
		Long:  `An interactive setup wizard to configure server settings like host, port, TLS, and webhook secret.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := interactiveServerSetup(App); err != nil {
				App.Events.Dispatch(events.Errorf("server", "Error during server setup: %v", err))
				os.Exit(1)
			}
			App.Events.Dispatch(events.Infof("server", "Server configuration saved successfully."))
		},
	}
	return cmd
}

func interactiveServerSetup(a *app.App) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println(utils.Colors.Blue("--- Server Setup ---"))

	a.State.ServerHost = utils.PromptString(reader, "Server Host", a.State.ServerHost)
	a.State.ServerPort = utils.PromptInt(reader, "Server Port", a.State.ServerPort)
	a.State.ServerTimezone = utils.PromptString(reader, "Global Timezone (IANA, optional)", a.State.ServerTimezone)
	a.State.ServerWebhookSecret = utils.PromptPassword(reader, "Webhook Secret (required when webhooks are enabled)", a.State.ServerWebhookSecret)
	if err := validateServerSetupSecurity(a); err != nil {
		return err
	}
	if a.State.ServerInternalAPIToken == "" {
		generatedToken, err := appserver.GenerateInternalAPIToken()
		if err != nil {
			return fmt.Errorf("failed to generate internal API token: %w", err)
		}
		a.State.ServerInternalAPIToken = generatedToken
	}
	a.State.ServerInternalAPIToken = utils.PromptString(reader, "Internal API Token", a.State.ServerInternalAPIToken)
	a.State.TLSEnabled = utils.PromptBool(reader, "Enable TLS/HTTPS", a.State.TLSEnabled)
	a.State.ServerLogRequests = utils.PromptBool(reader, "Log all incoming requests", a.State.ServerLogRequests)

	a.State.ServerTimezone = appserver.NormalizeTimezone(a.State.ServerTimezone)
	if err := appserver.ValidateTimezone(a.State.ServerTimezone); err != nil {
		return err
	}

	if a.State.TLSEnabled {
		a.State.TLSCert = utils.PromptString(reader, "TLS Certificate File", a.State.TLSCert)
		a.State.TLSKey = utils.PromptString(reader, "TLS Key File", a.State.TLSKey)
	}

	a.Config.Server.Host = a.State.ServerHost
	a.Config.Server.Port = a.State.ServerPort
	a.Config.Server.Timezone = a.State.ServerTimezone
	a.Config.Server.TLSEnabled = a.State.TLSEnabled
	a.Config.Server.TLSCert = a.State.TLSCert
	a.Config.Server.TLSKey = a.State.TLSKey
	a.Config.Server.WebhookSecret = a.State.ServerWebhookSecret
	a.Config.Server.InternalAPIToken = a.State.ServerInternalAPIToken
	a.Config.Server.LogRequests = a.State.ServerLogRequests

	return a.SaveConfig()
}

func validateServerSetupSecurity(a *app.App) error {
	if a == nil || a.Config == nil || a.State == nil {
		return nil
	}

	enabledWebhooks := a.Config.Server.Webhooks
	webhooksEnabled := enabledWebhooks[app.WebhookAccountCreate] || enabledWebhooks[app.WebhookCheckin]
	if len(enabledWebhooks) == 0 {
		webhooksEnabled = true
	}

	return validateWebhookSecurityConfig(webhooksEnabled, a.State.ServerWebhookSecret)
}
