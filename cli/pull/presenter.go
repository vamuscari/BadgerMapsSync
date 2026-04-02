package pull

import (
	"badgermaps/app"
	"badgermaps/app/pull"
	appserver "badgermaps/app/server"
	"badgermaps/app/syncproxy"
	"badgermaps/events"
	"fmt"
	"os"
	"strconv"

	"github.com/schollz/progressbar/v3"
)

// CliPresenter handles the presentation logic for the pull command.
type CliPresenter struct {
	App *app.App
}

// NewCliPresenter creates a new presenter for the pull command.
func NewCliPresenter(a *app.App) *CliPresenter {
	return &CliPresenter{App: a}
}

// HandlePullAccount orchestrates pulling a single account.
func (p *CliPresenter) HandlePullAccount(accountID int, opts ResponseSaveOptions) error {
	listener := func(e events.Event) {
		if e.Source != "account" {
			return
		}
		switch e.Type {
		case "pull.complete":
			payload := e.Payload.(events.CompletionPayload)
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled account %d.", payload.Count))
		case "pull.error":
			payload := e.Payload.(events.ErrorPayload)
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull account %d. The API returned an error.", accountID))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", payload.Error))
		}
	}
	p.App.Events.Subscribe("pull.complete", listener)
	p.App.Events.Subscribe("pull.error", listener)

	localRun := func() error {
		account, err := pull.PullAccount(p.App, accountID)
		if err != nil {
			return err
		}
		return p.saveResponse("account", strconv.Itoa(accountID), account, opts)
	}

	if opts.enabled() {
		if _, running := p.App.Server.GetServerStatus(); running {
			return fmt.Errorf("response saving is not supported while server is active; stop the server or run without --save-response/--response-file")
		}
		return localRun()
	}

	return syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullAccount,
		"cli.pull.account",
		accountID,
		localRun,
	)
}

// HandlePullAccounts orchestrates pulling all accounts.
func (p *CliPresenter) HandlePullAccounts() error {
	var bar *progressbar.ProgressBar

	pullListener := func(e events.Event) {
		if e.Source != "accounts" {
			return
		}
		switch e.Type {
		case "pull.group.start":
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling accounts..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case "pull.ids_fetched":
			payload := e.Payload.(events.ResourceIDsFetchedPayload)
			if bar != nil {
				bar.ChangeMax(payload.Count)
				bar.Describe(fmt.Sprintf("Found %d accounts to pull.", payload.Count))
			}
		case "pull.store.success":
			if bar != nil {
				bar.Add(1)
			}
		case "pull.group.error":
			payload := e.Payload.(events.ErrorPayload)
			if bar != nil {
				bar.Clear()
			}
			p.App.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", payload.Error))
		case "pull.group.complete":
			if bar != nil {
				bar.Finish()
				p.App.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	p.App.Events.Subscribe("pull.*", pullListener)

	err := syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullAccounts,
		"cli.pull.accounts",
		0,
		func() error {
			return pull.PullGroupAccounts(p.App, 0, nil)
		},
	)
	if bar != nil && !bar.IsFinished() {
		bar.Finish()
	}
	return err
}

// HandlePullCheckin orchestrates pulling a single checkin.
func (p *CliPresenter) HandlePullCheckin(checkinID int, opts ResponseSaveOptions) error {
	listener := func(e events.Event) {
		if e.Source != "check-in" {
			return
		}
		switch e.Type {
		case "pull.complete":
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled check-in %d.", checkinID))
		case "pull.error":
			payload := e.Payload.(events.ErrorPayload)
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull check-in %d. The API returned an error.", checkinID))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", payload.Error))
		}
	}
	p.App.Events.Subscribe("pull.complete", listener)
	p.App.Events.Subscribe("pull.error", listener)

	localRun := func() error {
		checkin, err := pull.PullCheckin(p.App, checkinID)
		if err != nil {
			return err
		}
		return p.saveResponse("checkin", strconv.Itoa(checkinID), checkin, opts)
	}

	if opts.enabled() {
		if _, running := p.App.Server.GetServerStatus(); running {
			return fmt.Errorf("response saving is not supported while server is active; stop the server or run without --save-response/--response-file")
		}
		return localRun()
	}

	return syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullCheckin,
		"cli.pull.checkin",
		checkinID,
		localRun,
	)
}

// HandlePullCheckins orchestrates pulling all checkins.
func (p *CliPresenter) HandlePullCheckins() error {
	var bar *progressbar.ProgressBar

	pullListener := func(e events.Event) {
		if e.Source != "checkins" {
			return
		}
		switch e.Type {
		case "pull.group.start":
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling checkins..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case "pull.ids_fetched":
			payload := e.Payload.(events.ResourceIDsFetchedPayload)
			if bar != nil {
				bar.ChangeMax(payload.Count)
				bar.Describe(fmt.Sprintf("Found %d accounts to pull checkins from.", payload.Count))
			}
		case "pull.store.success":
			if bar != nil {
				bar.Add(1)
			}
		case "pull.group.error":
			payload := e.Payload.(events.ErrorPayload)
			if bar != nil {
				bar.Clear()
			}
			p.App.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", payload.Error))
		case "pull.group.complete":
			if bar != nil {
				bar.Finish()
				p.App.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	p.App.Events.Subscribe("pull.*", pullListener)

	err := syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullCheckins,
		"cli.pull.checkins",
		0,
		func() error {
			return pull.PullGroupCheckins(p.App, nil)
		},
	)
	if bar != nil && !bar.IsFinished() {
		bar.Finish()
	}
	return err
}

// HandlePullRoute orchestrates pulling a single route.
func (p *CliPresenter) HandlePullRoute(routeID int, opts ResponseSaveOptions) error {
	listener := func(e events.Event) {
		if e.Source != "route" {
			return
		}
		switch e.Type {
		case "pull.complete":
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled route %d.", routeID))
		case "pull.error":
			payload := e.Payload.(events.ErrorPayload)
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull route %d. The API returned an error.", routeID))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", payload.Error))
		}
	}
	p.App.Events.Subscribe("pull.complete", listener)
	p.App.Events.Subscribe("pull.error", listener)

	localRun := func() error {
		route, err := pull.PullRoute(p.App, routeID)
		if err != nil {
			return err
		}
		return p.saveResponse("route", strconv.Itoa(routeID), route, opts)
	}

	if opts.enabled() {
		if _, running := p.App.Server.GetServerStatus(); running {
			return fmt.Errorf("response saving is not supported while server is active; stop the server or run without --save-response/--response-file")
		}
		return localRun()
	}

	return syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullRoute,
		"cli.pull.route",
		routeID,
		localRun,
	)
}

// HandlePullRoutes orchestrates pulling all routes.
func (p *CliPresenter) HandlePullRoutes() error {
	var bar *progressbar.ProgressBar

	pullListener := func(e events.Event) {
		if e.Source != "routes" {
			return
		}
		switch e.Type {
		case "pull.group.start":
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling routes..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case "pull.ids_fetched":
			payload := e.Payload.(events.ResourceIDsFetchedPayload)
			if bar != nil {
				bar.ChangeMax(payload.Count)
				bar.Describe(fmt.Sprintf("Found %d routes to pull.", payload.Count))
			}
		case "pull.store.success":
			if bar != nil {
				bar.Add(1)
			}
		case "pull.group.error":
			payload := e.Payload.(events.ErrorPayload)
			if bar != nil {
				bar.Clear()
			}
			p.App.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", payload.Error))
		case "pull.group.complete":
			if bar != nil {
				bar.Finish()
				p.App.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	p.App.Events.Subscribe("pull.*", pullListener)

	err := syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullRoutes,
		"cli.pull.routes",
		0,
		func() error {
			return pull.PullGroupRoutes(p.App, nil)
		},
	)
	if bar != nil && !bar.IsFinished() {
		bar.Finish()
	}
	return err
}

// HandlePullProfile orchestrates pulling the user profile.
func (p *CliPresenter) HandlePullProfile(opts ResponseSaveOptions) error {
	listener := func(e events.Event) {
		if e.Source != "user profile" {
			return
		}
		switch e.Type {
		case "pull.complete":
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled user profile."))
		case "pull.error":
			payload := e.Payload.(events.ErrorPayload)
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull user profile. The API returned an error."))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", payload.Error))
		}
	}
	p.App.Events.Subscribe("pull.complete", listener)
	p.App.Events.Subscribe("pull.error", listener)

	localRun := func() error {
		profile, err := pull.PullProfile(p.App, nil)
		if err != nil {
			return err
		}

		identifier := "profile"
		if profile != nil && profile.ProfileId.Valid {
			identifier = fmt.Sprintf("%d", profile.ProfileId.Int64)
		}
		return p.saveResponse("profile", identifier, profile, opts)
	}

	if opts.enabled() {
		if _, running := p.App.Server.GetServerStatus(); running {
			return fmt.Errorf("response saving is not supported while server is active; stop the server or run without --save-response/--response-file")
		}
		return localRun()
	}

	return syncproxy.RunWithServerRouting(
		p.App,
		appserver.SyncModePullProfile,
		"cli.pull.profile",
		0,
		localRun,
	)
}
