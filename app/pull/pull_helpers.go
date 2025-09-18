package pull

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/database"
	"badgermaps/events"
	"context"
	"fmt"
	"strings"
	"sync"
)

func PullAccount(a *app.App, accountID int) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullStart, Source: "account", Payload: accountID})
	a.Events.Dispatch(events.Infof("pull", "Pulling account with ID: %d", accountID))

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullError, Source: "account", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullComplete, Source: "account", Payload: accountID})
		}
	}()

	account, err := a.API.GetAccountDetailed(accountID)
	if err != nil {
		return fmt.Errorf("error pulling account: %w", err)
	}

	if err = StoreAccountDetailed(a, account); err != nil {
		return fmt.Errorf("error storing account: %w", err)
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled account with ID: %d", accountID))
	return nil
}

func PullGroupAccounts(a *app.App, top int, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullGroupStart, Source: "accounts"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullGroupError, Source: "accounts", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullGroupComplete, Source: "accounts"})
		}
	}()

	accountIDs, err := a.API.GetAccountIDs()
	if err != nil {
		err = fmt.Errorf("error getting account IDs: %w", err)
		a.Events.Dispatch(events.Event{Type: events.PullError, Source: "accounts", Payload: err})
		return err
	}

	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}
	total := len(accountIDs)
	a.Events.Dispatch(events.Event{Type: events.ResourceIDsFetched, Source: "accounts", Payload: total})

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, total)

	for i, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int, index int) {
			defer wg.Done()
			defer func() { <-sem }()

			a.Events.Dispatch(events.Event{Type: events.FetchDetailStart, Source: "accounts", Payload: accountID})
			account, err := a.API.GetAccountDetailed(accountID)
			if err != nil {
				err = fmt.Errorf("error getting detailed account info for ID %d: %w", accountID, err)
				a.Events.Dispatch(events.Event{Type: events.PullError, Source: "accounts", Payload: err})
				errorChan <- err
				return
			}
			a.Events.Dispatch(events.Event{Type: events.FetchDetailSuccess, Source: "accounts", Payload: account})

			if err := StoreAccountDetailed(a, account); err != nil {
				err = fmt.Errorf("error storing account %d: %w", accountID, err)
				a.Events.Dispatch(events.Event{Type: events.PullError, Source: "accounts", Payload: err})
				errorChan <- err
			} else {
				a.Events.Dispatch(events.Event{Type: events.StoreSuccess, Source: "accounts", Payload: account})
			}
			if progressCallback != nil {
				progressCallback(index+1, total)
			}
		}(id, i)
	}

	wg.Wait()
	close(errorChan)

	var pullErrors []string
	for err := range errorChan {
		pullErrors = append(pullErrors, err.Error())
	}

	if len(pullErrors) > 0 {
		err = fmt.Errorf("encountered errors during account pull:\n- %s", strings.Join(pullErrors, "\n- "))
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled all accounts"))
	return err
}

func PullCheckin(a *app.App, checkinID int) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullStart, Source: "check-in", Payload: checkinID})
	a.Events.Dispatch(events.Infof("pull", "Pulling checkin with ID: %d", checkinID))

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullError, Source: "check-in", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullComplete, Source: "check-in", Payload: checkinID})
		}
	}()

	checkin, err := a.API.GetCheckin(checkinID)
	if err != nil {
		return fmt.Errorf("error pulling checkin: %w", err)
	}

	if err = StoreCheckin(a, *checkin); err != nil {
		return fmt.Errorf("error storing checkin: %w", err)
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled checkin with ID: %d", checkinID))
	return nil
}

func PullGroupCheckins(a *app.App, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullGroupStart, Source: "checkins"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullGroupError, Source: "checkins", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullGroupComplete, Source: "checkins"})
		}
	}()

	accountIDs, err := a.API.GetAccountIDs()
	if err != nil {
		err = fmt.Errorf("error getting account IDs: %w", err)
		a.Events.Dispatch(events.Event{Type: events.PullError, Source: "checkins", Payload: err})
		return err
	}
	total := len(accountIDs)
	a.Events.Dispatch(events.Event{Type: events.ResourceIDsFetched, Source: "checkins", Payload: total})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, total)

	for i, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int, index int) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				return // Stop processing if context is cancelled
			default:
			}

			a.Events.Dispatch(events.Event{Type: events.FetchDetailStart, Source: "checkins", Payload: accountID})
			checkins, err := a.API.GetCheckinsForAccount(accountID)
			if err != nil {
				err = fmt.Errorf("error getting checkins for account ID %d: %w", accountID, err)
				a.Events.Dispatch(events.Event{Type: events.PullError, Source: "checkins", Payload: err})
				errorChan <- err
				cancel() // Cancel context on first error
				return
			}
			a.Events.Dispatch(events.Event{Type: events.FetchDetailSuccess, Source: "checkins", Payload: checkins})

			for _, checkin := range checkins {
				select {
				case <-ctx.Done():
					return // Stop processing if context is cancelled
				default:
				}
				if err := StoreCheckin(a, checkin); err != nil {
					err = fmt.Errorf("error storing checkin %d: %w", checkin.CheckinId.Int64, err)
					a.Events.Dispatch(events.Event{Type: events.PullError, Source: "checkins", Payload: err})
					errorChan <- err
					cancel() // Cancel context on first error
				} else {
					a.Events.Dispatch(events.Event{Type: events.StoreSuccess, Source: "checkins", Payload: checkin})
				}
			}
			if progressCallback != nil {
				progressCallback(index+1, total)
			}
		}(id, i)
	}

	wg.Wait()
	close(errorChan)

	var pullErrors []string
	for err := range errorChan {
		pullErrors = append(pullErrors, err.Error())
	}

	if len(pullErrors) > 0 {
		err = fmt.Errorf("encountered errors during check-in pull:\n- %s", strings.Join(pullErrors, "\n- "))
	}

	a.Events.Dispatch(events.Infof("pull", "Finished pulling all checkins"))
	return err
}

func PullRoute(a *app.App, routeID int) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullStart, Source: "route", Payload: routeID})
	a.Events.Dispatch(events.Infof("pull", "Pulling route with ID: %d", routeID))

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullError, Source: "route", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullComplete, Source: "route", Payload: routeID})
		}
	}()

	route, err := a.API.GetRoute(routeID)
	if err != nil {
		return fmt.Errorf("error pulling route: %w", err)
	}

	if err = StoreRoute(a, *route); err != nil {
		return fmt.Errorf("error storing route: %w", err)
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled route with ID: %d", routeID))
	return nil
}

func PullGroupRoutes(a *app.App, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullGroupStart, Source: "routes"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullGroupError, Source: "routes", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullGroupComplete, Source: "routes"})
		}
	}()

	routes, err := a.API.GetRoutes()
	if err != nil {
		err = fmt.Errorf("error getting routes: %w", err)
		a.Events.Dispatch(events.Event{Type: events.PullError, Source: "routes", Payload: err})
		return err
	}
	total := len(routes)
	a.Events.Dispatch(events.Event{Type: events.ResourceIDsFetched, Source: "routes", Payload: total})

	errorCount := 0
	for i, route := range routes {
		if !route.RouteId.Valid {
			if a.State.Verbose {
				a.Events.Dispatch(events.Debugf("pull", "Skipping route %d of %d with null ID", i+1, total))
			}
			continue
		}

		a.Events.Dispatch(events.Event{Type: events.FetchDetailSuccess, Source: "routes", Payload: route})
		if err := StoreRoute(a, route); err != nil {
			err = fmt.Errorf("error storing route: %w", err)
			a.Events.Dispatch(events.Event{Type: events.PullError, Source: "routes", Payload: err})
			errorCount++
		} else {
			a.Events.Dispatch(events.Event{Type: events.StoreSuccess, Source: "routes", Payload: route})
		}
		if progressCallback != nil {
			progressCallback(i+1, total)
		}
	}

	a.Events.Dispatch(events.Event{Type: events.PullComplete, Source: "routes", Payload: errorCount})
	a.Events.Dispatch(events.Infof("pull", "Successfully pulled all routes"))
	return nil
}

func PullProfile(a *app.App, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: events.PullStart, Source: "user profile"})
	a.Events.Dispatch(events.Infof("pull", "Pulling user profile..."))

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: events.PullError, Source: "user profile", Payload: err})
		} else {
			a.Events.Dispatch(events.Event{Type: events.PullComplete, Source: "user profile"})
		}
	}()

	totalSteps := 3 // 1. Get profile, 2. Store profile, 3. Update configs
	currentStep := 0

	profile, err := a.API.GetUserProfile()
	if err != nil {
		return fmt.Errorf("error pulling user profile: %w", err)
	}
	currentStep++
	if progressCallback != nil {
		progressCallback(currentStep, totalSteps)
	}

	if err = StoreProfile(a, profile); err != nil {
		return fmt.Errorf("error storing profile: %w", err)
	}
	currentStep++
	if progressCallback != nil {
		progressCallback(currentStep, totalSteps)
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled user profile for: %s", profile.Email.String))
	currentStep++
	if progressCallback != nil {
		progressCallback(currentStep, totalSteps)
	}
	return nil
}

func StoreAccountDetailed(a *app.App, acc *api.Account) error {
	if a.State.Verbose {
		a.Events.Dispatch(events.Debugf("pull", "Storing account: %s", acc.FullName.String))
	}
	return database.RunCommand(a.DB, "MergeAccountsDetailed",
		acc.AccountId, acc.FirstName, acc.LastName, acc.FullName, acc.PhoneNumber, acc.Email, acc.CustomerId, acc.Notes,
		acc.OriginalAddress, acc.CrmId, acc.AccountOwner, acc.DaysSinceLastCheckin, acc.LastCheckinDate,
		acc.LastModifiedDate, acc.FollowUpDate, acc.CustomNumeric, acc.CustomText, acc.CustomNumeric2,
		acc.CustomText2, acc.CustomNumeric3, acc.CustomText3, acc.CustomNumeric4, acc.CustomText4,
		acc.CustomNumeric5, acc.CustomText5, acc.CustomNumeric6, acc.CustomText6, acc.CustomNumeric7,
		acc.CustomText7, acc.CustomNumeric8, acc.CustomText8, acc.CustomNumeric9, acc.CustomText9,
		acc.CustomNumeric10, acc.CustomText10, acc.CustomNumeric11, acc.CustomText11, acc.CustomNumeric12,
		acc.CustomText12, acc.CustomNumeric13, acc.CustomText13, acc.CustomNumeric14, acc.CustomText14,
		acc.CustomNumeric15, acc.CustomText15, acc.CustomNumeric16, acc.CustomText16, acc.CustomNumeric17,
		acc.CustomText17, acc.CustomNumeric18, acc.CustomText18, acc.CustomNumeric19, acc.CustomText19,
		acc.CustomNumeric20, acc.CustomText20, acc.CustomNumeric21, acc.CustomText21, acc.CustomNumeric22,
		acc.CustomText22, acc.CustomNumeric23, acc.CustomText23, acc.CustomNumeric24, acc.CustomText24,
		acc.CustomNumeric25, acc.CustomText25, acc.CustomNumeric26, acc.CustomText26, acc.CustomNumeric27,
		acc.CustomText27, acc.CustomNumeric28, acc.CustomText28, acc.CustomNumeric29, acc.CustomText29,
		acc.CustomNumeric30, acc.CustomText30, acc.CreatedAt, acc.UpdatedAt,
	)
}

func StoreCheckin(a *app.App, checkin api.Checkin) error {
	if a.State.Verbose {
		a.Events.Dispatch(events.Debugf("pull", "Storing checkin: %d", checkin.CheckinId.Int64))
	}
	return database.RunCommand(a.DB, "MergeAccountCheckins",
		checkin.CheckinId, checkin.CrmId, checkin.AccountId, checkin.LogDatetime, checkin.Type, checkin.Comments,
		checkin.ExtraFields, checkin.CreatedBy,
	)
}

func StoreRoute(a *app.App, route api.Route) error {
	if a.State.Verbose {
		a.Events.Dispatch(events.Debugf("pull", "Storing route: %s", route.Name.String))
	}
	return database.RunCommand(a.DB, "MergeRoutes",
		route.RouteId, route.Name, route.RouteDate, route.Duration, route.StartAddress, route.DestinationAddress,
		route.StartTime,
	)
}

func StoreProfile(a *app.App, profile *api.UserProfile) error {
	if a.State.Verbose {
		a.Events.Dispatch(events.Debugf("pull", "Storing profile for: %s", profile.Email.String))
	}

	var crmFields []string
	for _, field := range profile.CRMEditableFieldsList {
		if field.Valid {
			crmFields = append(crmFields, field.String)
		}
	}
	crmEditableFieldsListStr := strings.Join(crmFields, ",")

	err := database.RunCommand(a.DB, "MergeUserProfiles",
		profile.ProfileId, profile.Email, profile.FirstName, profile.LastName, profile.IsManager,
		profile.IsHideReferralIOSBanner, profile.MarkerIcon, profile.Manager, crmEditableFieldsListStr,
		profile.CRMBaseURL, profile.CRMType, profile.ReferralURL, profile.MapStartZoom, profile.MapStart,
		profile.IsUserCanEdit, profile.IsUserCanDeleteCheckins, profile.IsUserCanAddNewTextValues,
		profile.HasData, profile.DefaultApptLength, profile.Completed, profile.TrialDaysLeft,
		profile.Company.Id, profile.Company.Name, profile.Company.ShortName,
	)
	if err != nil {
		return err
	}

	// Update Configurations table
	if err := database.UpdateConfiguration(a.DB, "ApiProfileId", fmt.Sprintf("%d", profile.ProfileId.Int64)); err != nil {
		return err
	}
	if err := database.UpdateConfiguration(a.DB, "ApiProfileName", fmt.Sprintf("%s %s", profile.FirstName.String, profile.LastName.String)); err != nil {
		return err
	}
	if err := database.UpdateConfiguration(a.DB, "CompanyId", fmt.Sprintf("%d", profile.Company.Id.Int64)); err != nil {
		return err
	}
	if err := database.UpdateConfiguration(a.DB, "CompanyName", profile.Company.Name.String); err != nil {
		return err
	}
	if err := database.UpdateConfiguration(a.DB, "SqlDbUserName", a.DB.GetUsername()); err != nil {
		return err
	}

	if err := database.RunCommand(a.DB, "DeleteDataSetValues", profile.ProfileId); err != nil {
		return err
	}
	if err := database.RunCommand(a.DB, "DeleteDataSets", profile.ProfileId); err != nil {
		return err
	}

	for _, datafield := range profile.Datafields {
		err := database.RunCommand(a.DB, "InsertDataSets",
			datafield.Name, profile.ProfileId, datafield.Filterable, datafield.Label, datafield.Position, datafield.Type,
			datafield.HasData, datafield.IsUserCanAddNewTextValues, datafield.RawMin, datafield.Min, datafield.Max,
			datafield.RawMax, datafield.AccountField,
		)
		if err != nil {
			return err
		}
		for _, value := range datafield.Values {
			err := database.RunCommand(a.DB, "InsertDataSetValues",
				datafield.Name, profile.ProfileId, value.Text, value.Value, datafield.Position,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
