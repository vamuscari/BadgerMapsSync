package app

import (
	"badgermaps/api"
	"badgermaps/database"
	"context"
	"fmt"
	"strings"
	"sync"
)

func PullAllAccounts(a *App, top int, log func(string)) error {
	accountIDs, err := a.API.GetAccountIDs()
	if err != nil {
		return fmt.Errorf("error getting account IDs: %w", err)
	}

	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.MaxConcurrentRequests)
	errorChan := make(chan error, len(accountIDs))

	for i, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int, index int) {
			defer wg.Done()
			defer func() { <-sem }()

			log(fmt.Sprintf("Pulling account %d of %d (ID: %d)", index+1, len(accountIDs), accountID))
			account, err := a.API.GetAccountDetailed(accountID)
			if err != nil {
				errorChan <- fmt.Errorf("error getting detailed account info for ID %d: %w", accountID, err)
				return
			}
			if err := StoreAccountDetailed(a, account, log); err != nil {
				errorChan <- fmt.Errorf("error storing account %d: %w", accountID, err)
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
		return fmt.Errorf("encountered errors during account pull:\n- %s", strings.Join(pullErrors, "\n- "))
	}

	log("Successfully pulled all accounts")
	return nil
}

func PullAllCheckins(a *App, log func(string)) error {
	accountIDs, err := database.GetAllAccountIDs(a.DB)
	if err != nil {
		return fmt.Errorf("error getting account IDs from DB: %w", err)
	}

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

			log(fmt.Sprintf("Pulling checkins for account %d of %d (ID: %d)", index+1, len(accountIDs), accountID))
			checkins, err := a.API.GetCheckinsForAccount(accountID)
			if err != nil {
				errorChan <- fmt.Errorf("error getting checkins for account ID %d: %w", accountID, err)
				cancel() // Cancel context on first error
				return
			}
			for _, checkin := range checkins {
				select {
				case <-ctx.Done():
					return // Stop processing if context is cancelled
				default:
				}
				if err := StoreCheckin(a, checkin, log); err != nil {
					errorChan <- fmt.Errorf("error storing checkin %d: %w", checkin.CheckinId, err)
					cancel() // Cancel context on first error
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
		return fmt.Errorf("encountered errors during check-in pull:\n- %s", strings.Join(pullErrors, "\n- "))
	}

	log("Successfully pulled all checkins")
	return nil
}

func PullAllRoutes(a *App, log func(string)) error {
	routes, err := a.API.GetRoutes()
	if err != nil {
		return fmt.Errorf("error getting routes: %w", err)
	}

	for i, route := range routes {
		if !route.RouteId.Valid {
			if a.State.Verbose {
				log(fmt.Sprintf("Skipping route %d of %d with null ID", i+1, len(routes)))
			}
			continue
		}
		log(fmt.Sprintf("Storing route %d of %d: %s", i+1, len(routes), route.Name))
		if err := StoreRoute(a, route, log); err != nil {
			// Log the error but continue trying to store other routes
			log(fmt.Sprintf("Error storing route: %v", err))
		}
	}
	log("Successfully pulled all routes")
	return nil
}

func StoreAccountDetailed(a *App, acc *api.Account, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing account: %s", acc.FullName))
	}
	return database.RunCommand(a.DB, "merge_accounts_detailed",
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
		log(fmt.Sprintf("Storing checkin: %d", checkin.CheckinId))
	}
	return database.RunCommand(a.DB, "merge_account_checkins",
		checkin.CheckinId, checkin.CrmId, checkin.AccountId, checkin.LogDatetime, checkin.Type, checkin.Comments,
		checkin.ExtraFields, checkin.CreatedBy,
	)
}

func StoreRoute(a *App, route api.Route, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing route: %s", route.Name))
	}
	return database.RunCommand(a.DB, "merge_routes",
		route.RouteId, route.Name, route.RouteDate, route.Duration, route.StartAddress, route.DestinationAddress,
		route.StartTime,
	)
}

func StoreProfile(a *App, profile *api.UserProfile, log func(string)) error {
	if a.State.Verbose {
		log(fmt.Sprintf("Storing profile for: %s", profile.Email))
	}

	var crmFields []string
	for _, field := range profile.CRMEditableFieldsList {
		if field.Valid {
			crmFields = append(crmFields, field.String)
		}
	}
	crmEditableFieldsListStr := strings.Join(crmFields, ",")

	err := database.RunCommand(a.DB, "merge_user_profiles",
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

	if err := database.RunCommand(a.DB, "delete_data_set_values", profile.ProfileId); err != nil {
		return err
	}
	if err := database.RunCommand(a.DB, "delete_data_sets", profile.ProfileId); err != nil {
		return err
	}

	for _, datafield := range profile.Datafields {
		err := database.RunCommand(a.DB, "insert_data_sets",
			datafield.Name, profile.ProfileId, datafield.Filterable, datafield.Label, datafield.Position, datafield.Type,
			datafield.HasData, datafield.IsUserCanAddNewTextValues, datafield.RawMin, datafield.Min, datafield.Max,
			datafield.RawMax, datafield.AccountField,
		)
		if err != nil {
			return err
		}
		for _, value := range datafield.Values {
			err := database.RunCommand(a.DB, "insert_data_set_values",
				datafield.Name, profile.ProfileId, value.Text, value.Value, datafield.Position,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
