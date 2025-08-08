# Server Mode

This document describes the server mode functionality of the BadgerMaps CLI.

## Overview

The BadgerMaps CLI can run in server mode, which allows it to listen for incoming webhooks from the BadgerMaps API. This enables real-time updates to your local database when changes occur in the BadgerMaps system.

## Starting the Server

To start the server, use the `server` command:

```bash
badgermaps server
```

By default, the server listens on `localhost:8080`. You can customize the host and port using flags or configuration options:

```bash
badgermaps server --host 0.0.0.0 --port 9000
```

## Webhook Endpoints

The server supports the following webhook endpoints:

- `/webhook/account`: Process account updates
- `/webhook/checkin`: Process check-in updates
- `/webhook/route`: Process route updates
- `/webhook/profile`: Process profile updates

When a webhook is received, the server processes the payload and updates the local database accordingly.

## Webhook Authentication

Webhooks are authenticated using shared secrets. The server validates the webhook payload before processing it to ensure it comes from a trusted source.

The authentication method uses an HMAC signature in the `X-BadgerMaps-Signature` header. The signature is verified against the request body using the shared secret.

## TLS/HTTPS Support

For secure webhook endpoints, you can enable TLS/HTTPS:

```bash
badgermaps server --tls --cert /path/to/cert.pem --key /path/to/key.pem
```

Alternatively, you can configure TLS in the configuration file or environment variables:

```
BADGERMAPS_SERVER_TLS_ENABLED=true
BADGERMAPS_SERVER_TLS_CERT=/path/to/cert.pem
BADGERMAPS_SERVER_TLS_KEY=/path/to/key.pem
```

## Scheduling

The server supports built-in scheduling with cron-like syntax. This allows you to run periodic tasks, such as pulling data from the API at regular intervals:

```bash
badgermaps server --schedule "0 */6 * * *"
```

This example runs the server and executes a data pull every 6 hours.

The schedule syntax follows the standard cron format with five fields:

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of the month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of the week (0 - 6) (Sunday to Saturday)
│ │ │ │ │
│ │ │ │ │
* * * * *
```

Special time strings are also supported:

- `@hourly`: Run once an hour at the beginning of the hour
- `@daily`: Run once a day at midnight
- `@weekly`: Run once a week at midnight on Sunday
- `@monthly`: Run once a month at midnight on the first day of the month
- `@yearly`: Run once a year at midnight on January 1

## Logging

The server logs all webhook requests and responses. You can control the verbosity of logging using the global flags:

- `--verbose` or `-v`: Enable verbose output with additional details
- `--quiet` or `-q`: Suppress all non-essential output
- `--debug`: Enable debug mode with maximum verbosity

Example:

```bash
badgermaps server --verbose
```

## Rate Limiting

The server implements rate limiting for webhook endpoints to prevent abuse. By default, it allows 100 requests per minute. You can customize the rate limits in the configuration:

```
BADGERMAPS_WEBHOOK_RATE_LIMIT_REQUESTS=200
BADGERMAPS_WEBHOOK_RATE_LIMIT_PERIOD=60
```

## Webhook Retry Mechanism

If processing a webhook fails, the server implements a retry mechanism. It will attempt to process the webhook again with exponential backoff:

1. First retry: 1 second delay
2. Second retry: 2 seconds delay
3. Third retry: 4 seconds delay
4. And so on, up to a maximum delay

The maximum number of retries and the maximum delay can be configured:

```
BADGERMAPS_WEBHOOK_MAX_RETRIES=5
BADGERMAPS_WEBHOOK_MAX_RETRY_DELAY=60
```

## Running as a Service

To run the server as a background service, you can use your operating system's service manager:

### Systemd (Linux)

Create a systemd service file at `/etc/systemd/system/badgermaps.service`:

```ini
[Unit]
Description=BadgerMaps CLI Server
After=network.target

[Service]
ExecStart=/usr/local/bin/badgermaps server
Restart=on-failure
User=badgermaps
Environment=BADGERMAPS_API_TOKEN=your_api_token

[Install]
WantedBy=multi-user.target
```

Then enable and start the service:

```bash
sudo systemctl enable badgermaps
sudo systemctl start badgermaps
```

### Launchd (macOS)

Create a launchd plist file at `~/Library/LaunchAgents/com.badgermaps.cli.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.badgermaps.cli</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/badgermaps</string>
        <string>server</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>BADGERMAPS_API_TOKEN</key>
        <string>your_api_token</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Then load the service:

```bash
launchctl load ~/Library/LaunchAgents/com.badgermaps.cli.plist
```

## Monitoring

You can monitor the server's status using the `status` subcommand:

```bash
badgermaps server status
```

This will show information about the server, including:

- Whether the server is running
- The server's uptime
- The number of webhook requests received
- The number of successful and failed webhook processing attempts
- Memory usage