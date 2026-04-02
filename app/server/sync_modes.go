package server

import (
	"fmt"
	"strings"
)

type SyncMode string

const (
	SyncModeNone         SyncMode = "none"
	SyncModePull         SyncMode = "pull"
	SyncModePush         SyncMode = "push"
	SyncModePullPush     SyncMode = "pull_push"
	SyncModePullAccounts SyncMode = "pull_accounts"
	SyncModePullCheckins SyncMode = "pull_checkins"
	SyncModePullRoutes   SyncMode = "pull_routes"
	SyncModePullProfile  SyncMode = "pull_profile"
	SyncModePushAccounts SyncMode = "push_accounts"
	SyncModePushCheckins SyncMode = "push_checkins"
	SyncModePullAccount  SyncMode = "pull_account"
	SyncModePullCheckin  SyncMode = "pull_checkin"
	SyncModePullRoute    SyncMode = "pull_route"
)

func ParseSyncMode(raw string) (SyncMode, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case string(SyncModeNone):
		return SyncModeNone, nil
	case string(SyncModePull), string(SyncTypeFull):
		return SyncModePull, nil
	case string(SyncModePush):
		return SyncModePush, nil
	case string(SyncModePullPush):
		return SyncModePullPush, nil
	case string(SyncModePullAccounts), string(SyncTypeAccounts):
		return SyncModePullAccounts, nil
	case string(SyncModePullCheckins), string(SyncTypeCheckins):
		return SyncModePullCheckins, nil
	case string(SyncModePullRoutes), string(SyncTypeRoutes):
		return SyncModePullRoutes, nil
	case string(SyncModePullProfile):
		return SyncModePullProfile, nil
	case string(SyncModePushAccounts):
		return SyncModePushAccounts, nil
	case string(SyncModePushCheckins):
		return SyncModePushCheckins, nil
	case string(SyncModePullAccount):
		return SyncModePullAccount, nil
	case string(SyncModePullCheckin):
		return SyncModePullCheckin, nil
	case string(SyncModePullRoute):
		return SyncModePullRoute, nil
	default:
		return "", fmt.Errorf("unsupported sync mode: %s", raw)
	}
}

type SyncJobStatus string

const (
	SyncJobQueued    SyncJobStatus = "queued"
	SyncJobRunning   SyncJobStatus = "running"
	SyncJobCompleted SyncJobStatus = "completed"
	SyncJobFailed    SyncJobStatus = "failed"
)

func (s SyncJobStatus) IsTerminal() bool {
	return s == SyncJobCompleted || s == SyncJobFailed
}
