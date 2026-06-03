[CmdletBinding()]
param(
    [string]$TaskName = "BadgerMapsSync Start Server After Boot",
    [string]$ExecutablePath,
    [string]$Arguments = "server start"
)

$ErrorActionPreference = "Stop"

Write-Warning "Startup scheduled tasks are deprecated for BadgerMapsSync server startup. Prefer: BadgerMapsSync.exe server install --config <global-config.yaml>; BadgerMapsSync.exe server start."

if ([string]::IsNullOrWhiteSpace($ExecutablePath)) {
    $ExecutablePath = Join-Path (Split-Path -Parent $PSScriptRoot) "BadgerMapsSync.exe"
}

try {
    $resolvedExecutablePath = (Resolve-Path -LiteralPath $ExecutablePath).Path
} catch {
    throw "Could not find BadgerMapsSync.exe at '$ExecutablePath'. Provide the path with -ExecutablePath."
}

$currentIdentity = [Security.Principal.WindowsIdentity]::GetCurrent()
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal($currentIdentity)
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw "Run this script from an elevated PowerShell window (Run as Administrator)."
}

$trigger = New-ScheduledTaskTrigger -AtStartup
$trigger.Delay = "PT5M"

$action = New-ScheduledTaskAction `
    -Execute $resolvedExecutablePath `
    -Argument $Arguments `
    -WorkingDirectory (Split-Path -Path $resolvedExecutablePath -Parent)

$principal = New-ScheduledTaskPrincipal `
    -UserId "SYSTEM" `
    -LogonType ServiceAccount `
    -RunLevel Highest

$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable

Register-ScheduledTask `
    -TaskName $TaskName `
    -Action $action `
    -Trigger $trigger `
    -Principal $principal `
    -Settings $settings `
    -Description "Starts BadgerMapsSync server 5 minutes after system startup." `
    -Force | Out-Null

Write-Host "Scheduled task '$TaskName' created or updated."
Write-Host "Executable : $resolvedExecutablePath"
Write-Host "Arguments  : $Arguments"
Write-Host "Trigger    : At startup + 5 minute delay"
Write-Host "Run as     : SYSTEM"
