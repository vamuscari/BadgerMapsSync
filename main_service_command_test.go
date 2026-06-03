package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestIsServerServiceRegistrationCommand(t *testing.T) {
	root := &cobra.Command{Use: "badgermaps"}
	serverCmd := &cobra.Command{Use: "server"}
	installCmd := &cobra.Command{Use: "install"}
	uninstallCmd := &cobra.Command{Use: "uninstall"}
	startCmd := &cobra.Command{Use: "start"}
	root.AddCommand(serverCmd)
	serverCmd.AddCommand(installCmd, uninstallCmd, startCmd)

	if !isServerServiceRegistrationCommand(installCmd) {
		t.Fatalf("expected server install to be a service registration command")
	}
	if !isServerServiceRegistrationCommand(uninstallCmd) {
		t.Fatalf("expected server uninstall to be a service registration command")
	}
	if isServerServiceRegistrationCommand(startCmd) {
		t.Fatalf("did not expect server start to be a service registration command")
	}
}
