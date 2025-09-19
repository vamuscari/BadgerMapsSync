package push

import (
	"badgermaps/app"
	"badgermaps/app/push"
	"badgermaps/database"
	"badgermaps/events"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/schollz/progressbar/v3"
)

// CliPresenter handles the presentation logic for the push command.
type CliPresenter struct {
	App *app.App
}

// NewCliPresenter creates a new presenter for the push command.
func NewCliPresenter(a *app.App) *CliPresenter {
	return &CliPresenter{App: a}
}

// HandleList orchestrates listing pending push changes.
func (p *CliPresenter) HandleList(entityType, status, date string, accountID int, orderBy string) error {
	options := push.PushFilterOptions{
		Status:    status,
		AccountID: accountID,
		OrderBy:   orderBy,
	}

	// Only include date if it's provided
	if date != "" {
		_, err := time.Parse("2006-01-02", date)
		if err != nil {
			return fmt.Errorf("invalid date format, please use YYYY-MM-DD")
		}
		options.Date = date
	}

	results, err := push.GetFilteredPendingChanges(p.App, entityType, options)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	switch entityType {
	case "accounts":
		changes, ok := results.([]database.AccountPendingChange)
		if !ok {
			return fmt.Errorf("unexpected type for account changes")
		}
		if len(changes) == 0 {
			p.App.Events.Dispatch(events.Infof("push", "No pending account changes found."))
			return nil
		}
		fmt.Fprintln(w, "ID\tAccount ID\tType\tStatus\tCreated At\tChanges")
		for _, c := range changes {
			fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\n", c.ChangeId, c.AccountId, c.ChangeType, c.Status, c.CreatedAt.Format(time.RFC3339), c.Changes)
		}
	case "checkins":
		changes, ok := results.([]database.CheckinPendingChange)
		if !ok {
			return fmt.Errorf("unexpected type for checkin changes")
		}
		if len(changes) == 0 {
			p.App.Events.Dispatch(events.Infof("push", "No pending check-in changes found."))
			return nil
		}
		fmt.Fprintln(w, "ID\tCheckin ID\tAccount ID\tType\tStatus\tCreated At\tChanges")
		for _, c := range changes {
			fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t%s\t%s\n", c.ChangeId, c.CheckinId, c.AccountId, c.ChangeType, c.Status, c.CreatedAt.Format(time.RFC3339), c.Changes)
		}
	}

	return nil
}

// HandlePushAccounts orchestrates pushing pending account changes.
func (p *CliPresenter) HandlePushAccounts() error {
	var bar *progressbar.ProgressBar

	pushListener := func(e events.Event) {
		if e.Source != "accounts" {
			return
		}
		switch e.Type {
		case events.PushScanStart:
			p.App.Events.Dispatch(events.Infof("push", "Scanning for pending %s changes...", e.Source))
		case events.PushItemError:
			err := e.Payload.(error)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push: %v", err))
		case events.PushError:
			err := e.Payload.(error)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push scan: %v", err))
		case events.PushComplete:
			if bar != nil {
				bar.Finish()
			}
			errorCount := e.Payload.(int)
			p.App.Events.Dispatch(events.Infof("push", "✔ Push for %s complete. Encountered %d errors.", e.Source, errorCount))
		}
	}

	p.App.Events.Subscribe(events.PushScanStart, pushListener)
	p.App.Events.Subscribe(events.PushScanComplete, pushListener)
	p.App.Events.Subscribe(events.PushItemSuccess, pushListener)
	p.App.Events.Subscribe(events.PushItemError, pushListener)
	p.App.Events.Subscribe(events.PushComplete, pushListener)

	return push.RunPushAccounts(p.App)
}

// HandlePushCheckins orchestrates pushing pending check-in changes.
func (p *CliPresenter) HandlePushCheckins() error {
	var bar *progressbar.ProgressBar

	pushListener := func(e events.Event) {
		// Only listen for checkin events
		if e.Source != "checkins" {
			return
		}

		switch e.Type {
		case events.PushScanStart:
			p.App.Events.Dispatch(events.Infof("push", "Scanning for pending %s changes...", e.Source))
		case events.PushScanComplete:
			changes := e.Payload.([]database.CheckinPendingChange)
			if len(changes) > 0 {
				bar = progressbar.NewOptions(len(changes),
					progressbar.OptionSetDescription(fmt.Sprintf("Pushing %d %s changes", len(changes), e.Source)),
					progressbar.OptionSetWriter(os.Stderr),
					progressbar.OptionEnableColorCodes(true),
				)
			}
		case events.PushItemSuccess:
			if bar != nil {
				bar.Add(1)
			}
		case events.PushItemError:
			err := e.Payload.(error)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push: %v", err))
		case events.PushError:
			err := e.Payload.(error)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push scan: %v", err))
		case events.PushComplete:
			if bar != nil {
				bar.Finish()
			}
			errorCount := e.Payload.(int)
			p.App.Events.Dispatch(events.Infof("push", "✔ Push for %s complete. Encountered %d errors.", e.Source, errorCount))
		}
	}

	p.App.Events.Subscribe(events.PushScanStart, pushListener)
	p.App.Events.Subscribe(events.PushScanComplete, pushListener)
	p.App.Events.Subscribe(events.PushItemSuccess, pushListener)
	p.App.Events.Subscribe(events.PushItemError, pushListener)
	p.App.Events.Subscribe(events.PushComplete, pushListener)

	return push.RunPushCheckins(p.App)
}

// HandlePushAll orchestrates pushing all pending changes.
func (p *CliPresenter) HandlePushAll() error {
	if err := p.HandlePushAccounts(); err != nil {
		return err
	}
	return p.HandlePushCheckins()
}
