# Windows Service Guide

BadgerMapsSync can run the webhook server as a machine-wide Windows service. This is the preferred production mode on shared Windows hosts because the server runs once for all users instead of starting from an individual desktop session.

## Install

Create or choose a global config file that the service account can read. A common layout is:

```powershell
C:\ProgramData\BadgerMapsSync\config.yaml
```

From an elevated PowerShell window:

```powershell
.\BadgerMapsSync.exe server install --config C:\ProgramData\BadgerMapsSync\config.yaml
.\BadgerMapsSync.exe server start
```

The installer registers:

- Service name: `BadgerMapsSync`
- Display name: `BadgerMaps Sync Service`
- Account: `LocalSystem`
- Startup: automatic

If `--config` is omitted, installation looks for `config.yaml` beside `BadgerMapsSync.exe`. User-local config files under `%LOCALAPPDATA%` are not used for service installation.

## Manage

Use the CLI or Windows Services console:

```powershell
.\BadgerMapsSync.exe server status
.\BadgerMapsSync.exe server stop
.\BadgerMapsSync.exe server restart
.\BadgerMapsSync.exe server uninstall
```

Starting, stopping, restarting, installing, and uninstalling may require an elevated shell depending on local Windows policy. The GUI shows service status and uses the same service-control path when permissions allow it.

## Notes

- Restart the service after changing server settings, scheduled jobs, or timezone configuration.
- Keep database paths, TLS certificate paths, and log/config paths readable by `LocalSystem`.
- The older startup scheduled-task script is no longer the recommended Windows startup mechanism; use the service instead.
