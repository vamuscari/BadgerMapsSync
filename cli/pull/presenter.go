package pull

import (
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/events"
	"fmt"
	"os"

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
func (p *CliPresenter) HandlePullAccount(accountID int) error {
	listener := func(e events.Event) {
		if e.Source != "account" {
			return
		}
		switch e.Type {
		case events.PullComplete:
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled account %d.", e.Payload.(int)))
		case events.PullError:
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull account %d. The API returned an error.", accountID))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", e.Payload.(error)))
		}
	}
	p.App.Events.Subscribe(events.PullComplete, listener)
	p.App.Events.Subscribe(events.PullError, listener)

	return pull.PullAccount(p.App, accountID)
}

// HandlePullAccounts orchestrates pulling all accounts.
func (p *CliPresenter) HandlePullAccounts() error {
	var bar *progressbar.ProgressBar

	pullListener := func(e events.Event) {
		if e.Source != "accounts" {
			return
		}
		switch e.Type {
		case events.PullGroupStart:
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling accounts..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case events.ResourceIDsFetched:
			count := e.Payload.(int)
			if bar != nil {
				bar.ChangeMax(count)
				bar.Describe(fmt.Sprintf("Found %d accounts to pull.", count))
			}
		case events.StoreSuccess:
			if bar != nil {
				bar.Add(1)
			}
		case events.PullGroupError:
			err := e.Payload.(error)
			if bar != nil {
				bar.Clear()
			}
			p.App.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", err))
		case events.PullGroupComplete:
			if bar != nil {
				bar.Finish()
				p.App.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	p.App.Events.Subscribe(events.PullGroupStart, pullListener)
	p.App.Events.Subscribe(events.ResourceIDsFetched, pullListener)
	p.App.Events.Subscribe(events.StoreSuccess, pullListener)
	p.App.Events.Subscribe(events.PullGroupError, pullListener)
	p.App.Events.Subscribe(events.PullGroupComplete, pullListener)

	err := pull.PullGroupAccounts(p.App, 0, nil)
	if bar != nil && !bar.IsFinished() {
		bar.Finish()
	}
	return err
}

// HandlePullCheckin orchestrates pulling a single checkin.
func (p *CliPresenter) HandlePullCheckin(checkinID int) error {
	listener := func(e events.Event) {
		if e.Source != "check-in" {
			return
		}
		switch e.Type {
		case events.PullComplete:
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled check-in %d.", e.Payload.(int)))
		case events.PullError:
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull check-in %d. The API returned an error.", checkinID))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", e.Payload.(error)))
		}
	}
	p.App.Events.Subscribe(events.PullComplete, listener)
	p.App.Events.Subscribe(events.PullError, listener)

	return pull.PullCheckin(p.App, checkinID)
}

// HandlePullCheckins orchestrates pulling all checkins.
func (p *CliPresenter) HandlePullCheckins() error {
	var bar *progressbar.ProgressBar

	pullListener := func(e events.Event) {
		if e.Source != "checkins" {
			return
		}
		switch e.Type {
		case events.PullGroupStart:
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling checkins..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case events.ResourceIDsFetched:
			count := e.Payload.(int)
			if bar != nil {
				bar.ChangeMax(count)
				bar.Describe(fmt.Sprintf("Found %d accounts to pull checkins from.", count))
			}
		case events.StoreSuccess:
			if bar != nil {
				bar.Add(1)
			}
		case events.PullGroupError:
			err := e.Payload.(error)
			if bar != nil {
				bar.Clear()
			}
			p.App.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", err))
		case events.PullGroupComplete:
			if bar != nil {
				bar.Finish()
				p.App.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	p.App.Events.Subscribe(events.PullGroupStart, pullListener)
	p.App.Events.Subscribe(events.ResourceIDsFetched, pullListener)
	p.App.Events.Subscribe(events.StoreSuccess, pullListener)
	p.App.Events.Subscribe(events.PullGroupError, pullListener)
	p.App.Events.Subscribe(events.PullGroupComplete, pullListener)

	err := pull.PullGroupCheckins(p.App, nil)
	if bar != nil && !bar.IsFinished() {
		bar.Finish()
	}
	return err
}

// HandlePullRoute orchestrates pulling a single route.
func (p *CliPresenter) HandlePullRoute(routeID int) error {
	listener := func(e events.Event) {
		if e.Source != "route" {
			return
		}
		switch e.Type {
		case events.PullComplete:
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled route %d.", e.Payload.(int)))
		case events.PullError:
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull route %d. The API returned an error.", routeID))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", e.Payload.(error)))
		}
	}
	p.App.Events.Subscribe(events.PullComplete, listener)
	p.App.Events.Subscribe(events.PullError, listener)

	return pull.PullRoute(p.App, routeID)
}

// HandlePullRoutes orchestrates pulling all routes.
func (p *CliPresenter) HandlePullRoutes() error {
	var bar *progressbar.ProgressBar

	pullListener := func(e events.Event) {
		if e.Source != "routes" {
			return
		}
		switch e.Type {
		case events.PullGroupStart:
			bar = progressbar.NewOptions(-1,
				progressbar.OptionSetDescription("Pulling routes..."),
				progressbar.OptionSetWriter(os.Stderr),
				progressbar.OptionSpinnerType(14),
				progressbar.OptionEnableColorCodes(true),
			)
		case events.ResourceIDsFetched:
			count := e.Payload.(int)
			if bar != nil {
				bar.ChangeMax(count)
				bar.Describe(fmt.Sprintf("Found %d routes to pull.", count))
			}
		case events.StoreSuccess:
			if bar != nil {
				bar.Add(1)
			}
		case events.PullGroupError:
			err := e.Payload.(error)
			if bar != nil {
				bar.Clear()
			}
			p.App.Events.Dispatch(events.Errorf("pull", "An error occurred during pull: %v", err))
		case events.PullGroupComplete:
			if bar != nil {
				bar.Finish()
				p.App.Events.Dispatch(events.Infof("pull", "✔ Pull for %s complete.", e.Source))
			}
		}
	}

	// Subscribe the listener to all relevant events
	p.App.Events.Subscribe(events.PullGroupStart, pullListener)
	p.App.Events.Subscribe(events.ResourceIDsFetched, pullListener)
	p.App.Events.Subscribe(events.StoreSuccess, pullListener)
	p.App.Events.Subscribe(events.PullGroupError, pullListener)
	p.App.Events.Subscribe(events.PullGroupComplete, pullListener)

	err := pull.PullGroupRoutes(p.App, nil)
	if bar != nil && !bar.IsFinished() {
		bar.Finish()
	}
	return err
}

// HandlePullProfile orchestrates pulling the user profile.
func (p *CliPresenter) HandlePullProfile() error {
	listener := func(e events.Event) {
		if e.Source != "user profile" {
			return
		}
		switch e.Type {
		case events.PullComplete:
			p.App.Events.Dispatch(events.Infof("pull", "Successfully pulled user profile."))
		case events.PullError:
			p.App.Events.Dispatch(events.Errorf("pull", "Error: Failed to pull user profile. The API returned an error."))
			p.App.Events.Dispatch(events.Warningf("pull", "Details: %v", e.Payload.(error)))
		}
	}
	p.App.Events.Subscribe(events.PullComplete, listener)
	p.App.Events.Subscribe(events.PullError, listener)

	return pull.PullProfile(p.App, nil)
}
