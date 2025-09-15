package app

import (
	"badgermaps/api"
	"badgermaps/database"
	"context"
	"fmt"
	"strings"
	"sync"
)

func PullAll(a *App, top int, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullAllStart, Source: "all"})
	log("Pulling all data...")

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullAllError, Source: "all", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullAllComplete, Source: "all"})
		}
	}()

	log("Pulling accounts...")
	if err = PullAllAccounts(a, top, log); err != nil {
		log(fmt.Sprintf("Error pulling accounts: %v", err))
		return err
	}

	log("Pulling checkins...")
	if err = PullAllCheckins(a, log); err != nil {
		log(fmt.Sprintf("Error pulling checkins: %v", err))
		return err
	}

	log("Pulling routes...")
	if err = PullAllRoutes(a, log); err != nil {
		log(fmt.Sprintf("Error pulling routes: %v", err))
		return err
	}

	log("Pulling user profile...")
	profile, err := a.API.GetUserProfile()
	if err != nil {
		log(fmt.Sprintf("Error pulling user profile: %v", err))
		return err
	}

	if err = StoreProfile(a, profile, log); err != nil {
		log(fmt.Sprintf("Error storing profile: %v", err))
		return err
	}
	log(fmt.Sprintf("Successfully pulled user profile for: %s", profile.Email.String))

	log("Finished pulling all data.")
	return nil
}

func PullAccount(a *App, accountID int, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullStart, Source: "account", Payload: accountID})
	log(fmt.Sprintf("Pulling account with ID: %d", accountID))

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullError, Source: "account", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullComplete, Source: "account", Payload: accountID})
		}
	}()

	account, err := a.API.GetAccountDetailed(accountID)
	if err != nil {
		return fmt.Errorf("error pulling account: %w", err)
	}

	if err = StoreAccountDetailed(a, account, log); err != nil {
		return fmt.Errorf("error storing account: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled account with ID: %d", accountID))
	return nil
}

func PullAllAccounts(a *App, top int, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullAllStart, Source: "accounts"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullAllError, Source: "accounts", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullAllComplete, Source: "accounts"})
		}
	}()

	accountIDs, err := a.API.GetAccountIDs()
	if err != nil {
		err = fmt.Errorf("error getting account IDs: %w", err)
		a.Events.Dispatch(Event{Type: PullError, Source: "accounts", Payload: err})
		return err
	}

	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}
	a.Events.Dispatch(Event{Type: ResourceIDsFetched, Source: "accounts", Payload: len(accountIDs)})

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, len(accountIDs))

	for i, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int, index int) {
			defer wg.Done()
			defer func() { <-sem }()

			a.Events.Dispatch(Event{Type: FetchDetailStart, Source: "accounts", Payload: accountID})
			account, err := a.API.GetAccountDetailed(accountID)
			if err != nil {
				err = fmt.Errorf("error getting detailed account info for ID %d: %w", accountID, err)
				a.Events.Dispatch(Event{Type: PullError, Source: "accounts", Payload: err})
				errorChan <- err
				return
			}
			a.Events.Dispatch(Event{Type: FetchDetailSuccess, Source: "accounts", Payload: account})

			if err := StoreAccountDetailed(a, account, log); err != nil {
				err = fmt.Errorf("error storing account %d: %w", accountID, err)
				a.Events.Dispatch(Event{Type: PullError, Source: "accounts", Payload: err})
				errorChan <- err
			} else {
				a.Events.Dispatch(Event{Type: StoreSuccess, Source: "accounts", Payload: account})
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

		log("Successfully pulled all accounts")
		return err
}

func PullCheckin(a *App, checkinID int, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullStart, Source: "check-in", Payload: checkinID})
	log(fmt.Sprintf("Pulling checkin with ID: %d", checkinID))

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullError, Source: "check-in", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullComplete, Source: "check-in", Payload: checkinID})
		}
	}()

	checkin, err := a.API.GetCheckin(checkinID)
	if err != nil {
		return fmt.Errorf("error pulling checkin: %w", err)
	}

	if err = StoreCheckin(a, *checkin, log); err != nil {
		return fmt.Errorf("error storing checkin: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled checkin with ID: %d", checkinID))
	return nil
}

func PullAllCheckins(a *App, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullAllStart, Source: "checkins"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullAllError, Source: "checkins", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullAllComplete, Source: "checkins"})
		}
	}()

	accountIDs, err := a.API.GetAccountIDs()
	if err != nil {
		err = fmt.Errorf("error getting account IDs: %w", err)
		a.Events.Dispatch(Event{Type: PullError, Source: "checkins", Payload: err})
		return err
	}
	a.Events.Dispatch(Event{Type: ResourceIDsFetched, Source: "checkins", Payload: len(accountIDs)})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, len(accountIDs))

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

			a.Events.Dispatch(Event{Type: FetchDetailStart, Source: "checkins", Payload: accountID})
			checkins, err := a.API.GetCheckinsForAccount(accountID)
			if err != nil {
				err = fmt.Errorf("error getting checkins for account ID %d: %w", accountID, err)
				a.Events.Dispatch(Event{Type: PullError, Source: "checkins", Payload: err})
				errorChan <- err
				cancel() // Cancel context on first error
				return
			}
			a.Events.Dispatch(Event{Type: FetchDetailSuccess, Source: "checkins", Payload: checkins})

			for _, checkin := range checkins {
				select {
				case <-ctx.Done():
					return // Stop processing if context is cancelled
				default:
				}
				if err := StoreCheckin(a, checkin, log); err != nil {
					err = fmt.Errorf("error storing checkin %d: %w", checkin.CheckinId.Int64, err)
					a.Events.Dispatch(Event{Type: PullError, Source: "checkins", Payload: err})
					errorChan <- err
					cancel() // Cancel context on first error
				} else {
					a.Events.Dispatch(Event{Type: StoreSuccess, Source: "checkins", Payload: checkin})
				}
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

	log("Finished pulling all checkins")
	return err
}

func PullRoute(a *App, routeID int, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullStart, Source: "route", Payload: routeID})
	log(fmt.Sprintf("Pulling route with ID: %d", routeID))

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullError, Source: "route", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullComplete, Source: "route", Payload: routeID})
		}
	}()

	route, err := a.API.GetRoute(routeID)
	if err != nil {
		return fmt.Errorf("error pulling route: %w", err)
	}

	if err = StoreRoute(a, *route, log); err != nil {
		return fmt.Errorf("error storing route: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled route with ID: %d", routeID))
	return nil
}

func PullAllRoutes(a *App, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullAllStart, Source: "routes"})

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullAllError, Source: "routes", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullAllComplete, Source: "routes"})
		}
	}()

	routes, err := a.API.GetRoutes()
	if err != nil {
		err = fmt.Errorf("error getting routes: %w", err)
		a.Events.Dispatch(Event{Type: PullError, Source: "routes", Payload: err})
		return err
	}
	a.Events.Dispatch(Event{Type: ResourceIDsFetched, Source: "routes", Payload: len(routes)})

	errorCount := 0
	for i, route := range routes {
		if !route.RouteId.Valid {
			if a.State.Verbose {
				log(fmt.Sprintf("Skipping route %d of %d with null ID", i+1, len(routes)))
			}
			continue
		}

		a.Events.Dispatch(Event{Type: FetchDetailSuccess, Source: "routes", Payload: route})
		if err := StoreRoute(a, route, log); err != nil {
			err = fmt.Errorf("error storing route: %w", err)
			a.Events.Dispatch(Event{Type: PullError, Source: "routes", Payload: err})
			errorCount++
		} else {
			a.Events.Dispatch(Event{Type: StoreSuccess, Source: "routes", Payload: route})
		}
	}

	a.Events.Dispatch(Event{Type: PullComplete, Source: "routes", Payload: errorCount})
	log("Successfully pulled all routes")
	return nil
}

func PullProfile(a *App, log func(string)) (err error) {
	a.Events.Dispatch(Event{Type: PullStart, Source: "user profile"})
	log("Pulling user profile...")

	defer func() {
		if err != nil {
			a.Events.Dispatch(Event{Type: PullError, Source: "user profile", Payload: err})
		} else {
			a.Events.Dispatch(Event{Type: PullComplete, Source: "user profile"})
		}
	}()

	profile, err := a.API.GetUserProfile()
	if err != nil {
		return fmt.Errorf("error pulling user profile: %w", err)
	}

	if err = StoreProfile(a, profile, log); err != nil {
		return fmt.Errorf("error storing profile: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled user profile for: %s", profile.Email.String))
	return nil
}

func StoreAccountDetailed(a *App, acc *api.Account, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing account: %s", acc.FullName.String))
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

func StoreCheckin(a *App, checkin api.Checkin, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing checkin: %d", checkin.CheckinId.Int64))
	}
	return database.RunCommand(a.DB, "MergeAccountCheckins",
		checkin.CheckinId, checkin.CrmId, checkin.AccountId, checkin.LogDatetime, checkin.Type, checkin.Comments,
		checkin.ExtraFields, checkin.CreatedBy,
	)
}

func StoreRoute(a *App, route api.Route, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing route: %s", route.Name.String))
	}
	return database.RunCommand(a.DB, "MergeRoutes",
		route.RouteId, route.Name, route.RouteDate, route.Duration, route.StartAddress, route.DestinationAddress,
		route.StartTime,
	)
}

func StoreProfile(a *App, profile *api.UserProfile, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing profile for: %s", profile.Email.String))
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
