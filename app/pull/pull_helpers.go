package pull

import (
	"badgermaps/api/models"
	"badgermaps/app"
	"badgermaps/database"
	"badgermaps/events"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/guregu/null/v6"
)

func PullAccount(a *app.App, accountID int) (account *models.Account, err error) {
	a.Events.Dispatch(events.Event{Type: "pull.start", Source: "account", Payload: events.PullStartPayload{ResourceID: accountID}})
	a.Events.Dispatch(events.Infof("pull", "Pulling account with ID: %d", accountID))

	defer func() {
		success := err == nil
		payload := events.CompletionPayload{Success: success, ResourceID: accountID}
		if success {
			payload.Count = 1
		} else {
			payload.Error = err
			a.Events.Dispatch(events.Event{Type: "pull.error", Source: "account", Payload: events.ErrorPayload{Error: err, ResourceID: accountID}})
		}
		a.Events.Dispatch(events.Event{Type: "pull.complete", Source: "account", Payload: payload})
	}()

	accountResp, err := a.API.GetAccountDetailed(accountID)
	if err != nil {
		return nil, fmt.Errorf("error pulling account: %w", err)
	}
	account = &accountResp.Data

	if err = StoreAccountDetailed(a, account); err != nil {
		return nil, fmt.Errorf("error storing account: %w", err)
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled account with ID: %d", accountID))
	return account, nil
}

func PullGroupAccounts(a *app.App, top int, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: "pull.group.start", Source: "accounts"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: "pull.group.error", Source: "accounts", Payload: events.ErrorPayload{Error: err}})
		}
	}()

	accountIDsResp, err := a.API.GetAccountIDs()
	if err != nil {
		err = fmt.Errorf("error getting account IDs: %w", err)
		a.Events.Dispatch(events.Event{Type: "pull.error", Source: "accounts", Payload: events.ErrorPayload{Error: err}})
		return err
	}
	accountIDs := accountIDsResp.Data

	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}
	total := len(accountIDs)
	a.Events.Dispatch(events.Event{Type: "pull.ids_fetched", Source: "accounts", Payload: events.ResourceIDsFetchedPayload{Count: total}})

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, total)
	var successCount atomic.Int64

	for i, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int, index int) {
			defer wg.Done()
			defer func() { <-sem }()

			a.Events.Dispatch(events.Event{Type: "pull.fetch_detail.start", Source: "accounts", Payload: events.FetchDetailStartPayload{ResourceID: accountID}})
			accountResp, err := a.API.GetAccountDetailed(accountID)
			if err != nil {
				err = fmt.Errorf("error getting detailed account info for ID %d: %w", accountID, err)
				a.Events.Dispatch(events.Event{Type: "pull.error", Source: "accounts", Payload: events.ErrorPayload{Error: err, ResourceID: accountID}})
				errorChan <- err
				return
			}
			account := &accountResp.Data
			a.Events.Dispatch(events.Event{Type: "pull.fetch_detail.success", Source: "accounts", Payload: events.FetchDetailSuccessPayload{Data: account}})

			if err := StoreAccountDetailed(a, account); err != nil {
				err = fmt.Errorf("error storing account %d: %w", accountID, err)
				a.Events.Dispatch(events.Event{Type: "pull.error", Source: "accounts", Payload: events.ErrorPayload{Error: err, ResourceID: accountID}})
				errorChan <- err
			} else {
				a.Events.Dispatch(events.Event{Type: "pull.store.success", Source: "accounts", Payload: events.StoreSuccessPayload{Data: account}})
				successCount.Add(1)
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

	successTotal := int(successCount.Load())
	success := err == nil
	a.Events.Dispatch(events.Event{Type: "pull.group.complete", Source: "accounts", Payload: events.CompletionPayload{Success: success, Error: err, Count: successTotal}})
	if success {
		a.Events.Dispatch(events.Infof("pull", "Successfully pulled all accounts"))
	} else {
		a.Events.Dispatch(events.Warningf("pull", "Finished pulling accounts with %d success(es) and %d error(s)", successTotal, len(pullErrors)))
	}
	return err
}

func PullCheckin(a *app.App, checkinID int) (checkin *models.Checkin, err error) {
	a.Events.Dispatch(events.Event{Type: "pull.start", Source: "check-in", Payload: events.PullStartPayload{ResourceID: checkinID}})
	a.Events.Dispatch(events.Infof("pull", "Pulling checkin with ID: %d", checkinID))

	defer func() {
		success := err == nil
		payload := events.CompletionPayload{Success: success, ResourceID: checkinID}
		if success {
			payload.Count = 1
		} else {
			payload.Error = err
			a.Events.Dispatch(events.Event{Type: "pull.error", Source: "check-in", Payload: events.ErrorPayload{Error: err, ResourceID: checkinID}})
		}
		a.Events.Dispatch(events.Event{Type: "pull.complete", Source: "check-in", Payload: payload})
	}()

	return nil, fmt.Errorf("pulling a single checkin by ID is not supported: /appointments/{id} endpoint removed")
}

// PullCheckinsForAccount pulls all check-ins for a specific account ID.
func PullCheckinsForAccount(a *app.App, accountID int) (err error) {
	a.Events.Dispatch(events.Event{Type: "pull.start", Source: "checkins", Payload: events.PullStartPayload{ResourceID: accountID}})
	a.Events.Dispatch(events.Infof("pull", "Pulling check-ins for account ID: %d", accountID))

	count := 0
	defer func() {
		success := err == nil
		payload := events.CompletionPayload{Success: success, ResourceID: accountID}
		if success {
			payload.Count = count
		} else {
			payload.Error = err
			a.Events.Dispatch(events.Event{Type: "pull.error", Source: "checkins", Payload: events.ErrorPayload{Error: err, ResourceID: accountID}})
		}
		a.Events.Dispatch(events.Event{Type: "pull.complete", Source: "checkins", Payload: payload})
	}()

	checkinsResp, err := a.API.GetCheckinsForAccount(accountID)
	if err != nil {
		return fmt.Errorf("error getting check-ins for account %d: %w", accountID, err)
	}
	checkins := checkinsResp.Data

	count = len(checkins)
	for _, checkin := range checkins {
		if err := StoreCheckin(a, checkin); err != nil {
			return fmt.Errorf("error storing check-in %d for account %d: %w", checkin.CheckinId.Int64, accountID, err)
		}
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled %d check-ins for account %d", count, accountID))
	return nil
}

func PullGroupCheckins(a *app.App, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: "pull.group.start", Source: "checkins"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: "pull.group.error", Source: "checkins", Payload: events.ErrorPayload{Error: err}})
		}
	}()

	accountIDsResp, err := a.API.GetAccountIDs()
	if err != nil {
		err = fmt.Errorf("error getting account IDs: %w", err)
		a.Events.Dispatch(events.Event{Type: "pull.error", Source: "checkins", Payload: events.ErrorPayload{Error: err}})
		return err
	}
	accountIDs := accountIDsResp.Data
	total := len(accountIDs)
	a.Events.Dispatch(events.Event{Type: "pull.ids_fetched", Source: "checkins", Payload: events.ResourceIDsFetchedPayload{Count: total}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, total)
	var successCount atomic.Int64

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

			a.Events.Dispatch(events.Event{Type: "pull.fetch_detail.start", Source: "checkins", Payload: events.FetchDetailStartPayload{ResourceID: accountID}})
			checkinsResp, err := a.API.GetCheckinsForAccount(accountID)
			if err != nil {
				err = fmt.Errorf("error getting checkins for account ID %d: %w", accountID, err)
				a.Events.Dispatch(events.Event{Type: "pull.error", Source: "checkins", Payload: events.ErrorPayload{Error: err, ResourceID: accountID}})
				errorChan <- err
				cancel() // Cancel context on first error
				return
			}
			checkins := checkinsResp.Data
			a.Events.Dispatch(events.Event{Type: "pull.fetch_detail.success", Source: "checkins", Payload: events.FetchDetailSuccessPayload{Data: checkins}})

			for _, checkin := range checkins {
				select {
				case <-ctx.Done():
					return // Stop processing if context is cancelled
				default:
				}
				if err := StoreCheckin(a, checkin); err != nil {
					err = fmt.Errorf("error storing checkin %d: %w", checkin.CheckinId.Int64, err)
					a.Events.Dispatch(events.Event{Type: "pull.error", Source: "checkins", Payload: events.ErrorPayload{Error: err, ResourceID: checkin.CheckinId.Int64}})
					errorChan <- err
					cancel() // Cancel context on first error
				} else {
					a.Events.Dispatch(events.Event{Type: "pull.store.success", Source: "checkins", Payload: events.StoreSuccessPayload{Data: checkin}})
				}
			}
			successCount.Add(1)
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

	successTotal := int(successCount.Load())
	success := err == nil
	a.Events.Dispatch(events.Event{Type: "pull.group.complete", Source: "checkins", Payload: events.CompletionPayload{Success: success, Error: err, Count: successTotal}})
	if success {
		a.Events.Dispatch(events.Infof("pull", "Finished pulling all checkins"))
	} else {
		a.Events.Dispatch(events.Warningf("pull", "Finished pulling checkins with %d success(es) and %d error(s)", successTotal, len(pullErrors)))
	}
	return err
}

func PullRoute(a *app.App, routeID int) (route *models.Route, err error) {
	a.Events.Dispatch(events.Event{Type: "pull.start", Source: "route", Payload: events.PullStartPayload{ResourceID: routeID}})
	a.Events.Dispatch(events.Infof("pull", "Pulling route with ID: %d", routeID))

	defer func() {
		success := err == nil
		payload := events.CompletionPayload{Success: success, ResourceID: routeID}
		if success {
			payload.Count = 1
		} else {
			payload.Error = err
			a.Events.Dispatch(events.Event{Type: "pull.error", Source: "route", Payload: events.ErrorPayload{Error: err, ResourceID: routeID}})
		}
		a.Events.Dispatch(events.Event{Type: "pull.complete", Source: "route", Payload: payload})
	}()

	routeResp, err := a.API.GetRoute(routeID)
	if err != nil {
		return nil, fmt.Errorf("error pulling route: %w", err)
	}
	route = &routeResp.Data

	if err = StoreRoute(a, *route); err != nil {
		return nil, fmt.Errorf("error storing route: %w", err)
	}

	a.Events.Dispatch(events.Infof("pull", "Successfully pulled route with ID: %d", routeID))
	return route, nil
}

func PullGroupRoutes(a *app.App, progressCallback func(current, total int)) (err error) {
	a.Events.Dispatch(events.Event{Type: "pull.group.start", Source: "routes"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(events.Event{Type: "pull.group.error", Source: "routes", Payload: events.ErrorPayload{Error: err}})
		}
	}()

	routesResp, err := a.API.GetRoutes()
	if err != nil {
		err = fmt.Errorf("error getting routes: %w", err)
		a.Events.Dispatch(events.Event{Type: "pull.error", Source: "routes", Payload: events.ErrorPayload{Error: err}})
		return err
	}
	routes := routesResp.Data
	total := len(routes)
	a.Events.Dispatch(events.Event{Type: "pull.ids_fetched", Source: "routes", Payload: events.ResourceIDsFetchedPayload{Count: total}})

	successCount := 0
	var routeErrors []string
	for i, route := range routes {
		if !route.RouteId.Valid {
			if a.State.Verbose {
				a.Events.Dispatch(events.Debugf("pull", "Skipping route %d of %d with null ID", i+1, total))
			}
			continue
		}

		a.Events.Dispatch(events.Event{Type: "pull.fetch_detail.success", Source: "routes", Payload: events.FetchDetailSuccessPayload{Data: route}})
		if storeErr := StoreRoute(a, route); storeErr != nil {
			wrappedErr := fmt.Errorf("error storing route %d: %w", route.RouteId.Int64, storeErr)
			a.Events.Dispatch(events.Event{Type: "pull.error", Source: "routes", Payload: events.ErrorPayload{Error: wrappedErr, ResourceID: route.RouteId.Int64}})
			routeErrors = append(routeErrors, wrappedErr.Error())
		} else {
			a.Events.Dispatch(events.Event{Type: "pull.store.success", Source: "routes", Payload: events.StoreSuccessPayload{Data: route}})
			successCount++
		}
		if progressCallback != nil {
			progressCallback(i+1, total)
		}
	}

	if len(routeErrors) > 0 {
		err = fmt.Errorf("encountered errors during route pull:\n- %s", strings.Join(routeErrors, "\n- "))
	}

	success := len(routeErrors) == 0
	a.Events.Dispatch(events.Event{Type: "pull.group.complete", Source: "routes", Payload: events.CompletionPayload{Success: success, Error: err, Count: successCount}})
	if success {
		a.Events.Dispatch(events.Infof("pull", "Successfully pulled all routes"))
	} else {
		a.Events.Dispatch(events.Warningf("pull", "Finished pulling routes with %d success(es) and %d error(s)", successCount, len(routeErrors)))
	}
	return err
}

func PullProfile(a *app.App, progressCallback func(current, total int)) (profile *models.UserProfile, err error) {
	a.Events.Dispatch(events.Event{Type: "pull.start", Source: "user profile"})
	a.Events.Dispatch(events.Infof("pull", "Pulling user profile..."))

	defer func() {
		var profileID interface{}
		if profile != nil && profile.ProfileId.Valid {
			profileID = profile.ProfileId.Int64
		}
		success := err == nil
		payload := events.CompletionPayload{Success: success, ResourceID: profileID}
		if success {
			payload.Count = 1
		} else {
			payload.Error = err
			a.Events.Dispatch(events.Event{Type: "pull.error", Source: "user profile", Payload: events.ErrorPayload{Error: err, ResourceID: profileID}})
		}
		a.Events.Dispatch(events.Event{Type: "pull.complete", Source: "user profile", Payload: payload})
	}()

	totalSteps := 3 // 1. Get profile, 2. Store profile, 3. Update configs
	currentStep := 0

	profileResp, err := a.API.GetUserProfile()
	if err != nil {
		return nil, fmt.Errorf("error pulling user profile: %w", err)
	}
	profile = &profileResp.Data
	currentStep++
	if progressCallback != nil {
		progressCallback(currentStep, totalSteps)
	}

	if err = StoreProfile(a, profile); err != nil {
		return nil, fmt.Errorf("error storing profile: %w", err)
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
	return profile, nil
}

func StoreAccountDetailed(a *app.App, acc *models.Account) error {
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

func StoreCheckin(a *app.App, checkin models.Checkin) error {
	if a.State.Verbose {
		a.Events.Dispatch(events.Debugf("pull", "Storing checkin: %d", checkin.CheckinId.Int64))
	}

	endpointType := "standard"
	storedType := checkin.Type
	storedComments := checkin.Comments

	// Convert ExtraFields (json.RawMessage) to string for database storage
	var extraFieldsStr string
	if len(checkin.ExtraFields) > 0 {
		extraFieldsStr = string(checkin.ExtraFields)

		trimmedRaw := strings.TrimSpace(extraFieldsStr)
		if trimmedRaw == "" || strings.EqualFold(trimmedRaw, "null") || trimmedRaw == "{}" {
			endpointType = "standard"
			extraFieldsStr = ""
		} else {
			endpointType = "custom"
			var extraFields map[string]interface{}
			if err := json.Unmarshal(checkin.ExtraFields, &extraFields); err == nil {
				if value, ok := extraFields["Log Type"].(string); ok {
					trimmed := strings.TrimSpace(value)
					if trimmed != "" {
						storedType = null.StringFrom(trimmed)
					}
				}
				if value, ok := extraFields["Meeting Notes"].(string); ok {
					trimmed := strings.TrimSpace(value)
					if trimmed != "" {
						storedComments = null.StringFrom(trimmed)
					}
				}
			}
		}
	}

	return database.RunCommand(a.DB, "MergeAccountCheckins",
		checkin.CheckinId, checkin.CrmId, checkin.AccountId, checkin.LogDatetime, storedType, storedComments,
		extraFieldsStr, endpointType, checkin.CreatedBy,
	)
}

func StoreRoute(a *app.App, route models.Route) error {
	if a.State.Verbose {
		a.Events.Dispatch(events.Debugf("pull", "Storing route: %s", route.Name.String))
	}
	return database.RunCommand(a.DB, "MergeRoutes",
		route.RouteId, route.Name, route.RouteDate, route.Duration, route.StartAddress, route.DestinationAddress,
		route.StartTime,
	)
}

func StoreProfile(a *app.App, profile *models.UserProfile) error {
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
