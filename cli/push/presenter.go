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
		fmt.Fprintln(w, "ID\tCheckin ID\tAccount ID\tChange Type\tEndpoint\tCheckin Type\tStatus\tCreated At\tComments")
		for _, c := range changes {
			fmt.Fprintf(
				w,
				"%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
				c.ChangeId,
				c.CheckinId,
				c.AccountId,
				c.ChangeType,
				c.EndpointType.String,
				c.Type.String,
				c.Status,
				c.CreatedAt.Format(time.RFC3339),
				c.Comments.String,
			)
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
		case "push.scan.start":
			p.App.Events.Dispatch(events.Infof("push", "Scanning for pending %s changes...", e.Source))
		case "push.item.error":
			payload := e.Payload.(events.PushItemErrorPayload)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push: %v", payload.Error))
		case "push.error":
			payload := e.Payload.(events.ErrorPayload)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push scan: %v", payload.Error))
		case "push.complete":
			if bar != nil {
				bar.Finish()
			}
			payload := e.Payload.(events.PushCompletePayload)
			p.App.Events.Dispatch(events.Infof("push", "✔ Push for %s complete. Encountered %d errors.", e.Source, payload.ErrorCount))
		}
	}

	p.App.Events.Subscribe("push.*", pushListener)

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
		case "push.scan.start":
			p.App.Events.Dispatch(events.Infof("push", "Scanning for pending %s changes...", e.Source))
		case "push.scan.complete":
			payload := e.Payload.(events.PushScanCompletePayload)
			changes := payload.Changes.([]database.CheckinPendingChange)
			if len(changes) > 0 {
				bar = progressbar.NewOptions(len(changes),
					progressbar.OptionSetDescription(fmt.Sprintf("Pushing %d %s changes", len(changes), e.Source)),
					progressbar.OptionSetWriter(os.Stderr),
					progressbar.OptionEnableColorCodes(true),
				)
			}
		case "push.item.success":
			if bar != nil {
				bar.Add(1)
			}
		case "push.item.error":
			payload := e.Payload.(events.PushItemErrorPayload)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push: %v", payload.Error))
		case "push.error":
			payload := e.Payload.(events.ErrorPayload)
			p.App.Events.Dispatch(events.Errorf("push", "An error occurred during push scan: %v", payload.Error))
		case "push.complete":
			if bar != nil {
				bar.Finish()
			}
			payload := e.Payload.(events.PushCompletePayload)
			p.App.Events.Dispatch(events.Infof("push", "✔ Push for %s complete. Encountered %d errors.", e.Source, payload.ErrorCount))
		}
	}

	p.App.Events.Subscribe("push.*", pushListener)

	return push.RunPushCheckins(p.App)
}

// HandlePushAll orchestrates pushing all pending changes.
func (p *CliPresenter) HandlePushAll() error {
	if err := p.HandlePushAccounts(); err != nil {
		return err
	}
	return p.HandlePushCheckins()
}
