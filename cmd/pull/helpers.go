package pull

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/database"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

func PullAllAccounts(App *app.App, top int) {
	accountIDs, err := App.API.GetAccountIDs()
	if err != nil {
		fmt.Println(color.RedString("Error getting account IDs: %v", err))
		os.Exit(1)
	}

	if top > 0 && top < len(accountIDs) {
		accountIDs = accountIDs[:top]
	}

	bar := progressbar.Default(int64(len(accountIDs)))

	var wg sync.WaitGroup
	sem := make(chan struct{}, App.AdvancedConfig.MaxConcurrentRequests)

	for _, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int) {
			defer wg.Done()
			defer func() { <-sem }()

			account, err := App.API.GetAccountDetailed(accountID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n%s\n", color.RedString("Error getting detailed account info for ID %d: %v", accountID, err))
				return
			}
			if err := StoreAccountDetailed(App, account); err != nil {
				fmt.Fprintf(os.Stderr, "\n%s\n", color.RedString("Error storing account %d: %v", accountID, err))
			}
			bar.Add(1)
		}(id)
	}

	wg.Wait()
	fmt.Println(color.GreenString("Successfully pulled all accounts"))
}

func PullAllCheckins(App *app.App) {
	accountIDs, err := database.GetAllAccountIDs(App.DB)
	if err != nil {
		fmt.Println(color.RedString("Error getting account IDs: %v", err))
		os.Exit(1)
	}

	bar := progressbar.Default(int64(len(accountIDs)))

	var wg sync.WaitGroup
	sem := make(chan struct{}, App.AdvancedConfig.MaxConcurrentRequests)

	for _, id := range accountIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(accountID int) {
			defer wg.Done()
			defer func() { <-sem }()

			checkins, err := App.API.GetCheckinsForAccount(accountID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n%s\n", color.RedString("Error getting checkins for account ID %d: %v", accountID, err))
				return
			}
			for _, checkin := range checkins {
				if err := StoreCheckin(App, checkin); err != nil {
					fmt.Fprintf(os.Stderr, "\n%s\n", color.RedString("Error storing checkin %d: %v", checkin.CheckinId, err))
				}
			}
			bar.Add(1)
		}(id)
	}

	wg.Wait()
	fmt.Println(color.GreenString("Successfully pulled all checkins"))
}

func PullAllRoutes(App *app.App) {
	routes, err := App.API.GetRoutes()
	if err != nil {
		fmt.Println(color.RedString("Error getting routes: %v", err))
		os.Exit(1)
	}

	bar := progressbar.Default(int64(len(routes)))
	for _, route := range routes {
		if !route.RouteId.Valid {
			if App.State.Verbose {
				fmt.Fprintf(os.Stderr, "\n%s\n", color.YellowString("Skipping route with null ID"))
			}
			continue
		}
		if err := StoreRoute(App, route); err != nil {
			fmt.Fprintf(os.Stderr, "\n%s\n", color.RedString("Error storing route: %v", err))
		}
		bar.Add(1)
	}
	fmt.Println(color.GreenString("Successfully pulled all routes"))
}

func StoreAccountDetailed(App *app.App, acc *api.Account) error {
	if App.State.Verbose {
		fmt.Println(color.CyanString("Storing account: %s", acc.FullName))
	}
	return database.RunCommand(App.DB, "merge_accounts_detailed",
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
		acc.CustomNumeric30, acc.CustomText30,
	)
}

func StoreAccountBasic(App *app.App, acc *api.Account) error {
	if App.State.Verbose {
		fmt.Println(color.CyanString("Storing account: %s", acc.FullName))
	}
	return database.RunCommand(App.DB, "merge_accounts_basic",
		acc.AccountId, acc.FullName,
	)
}

func StoreCheckin(App *app.App, checkin api.Checkin) error {
	if App.State.Verbose {
		fmt.Println(color.CyanString("Storing checkin: %d", checkin.CheckinId))
	}
	return database.RunCommand(App.DB, "merge_account_checkins",
		checkin.CheckinId, checkin.CrmId, checkin.AccountId, checkin.LogDatetime, checkin.Type, checkin.Comments,
		checkin.ExtraFields, checkin.CreatedBy,
	)
}

func StoreRoute(App *app.App, route api.Route) error {
	if App.State.Verbose {
		fmt.Println(color.CyanString("Storing route: %s", route.Name))
	}
	return database.RunCommand(App.DB, "merge_routes",
		route.RouteId, route.Name, route.RouteDate, route.Duration, route.StartAddress, route.DestinationAddress,
		route.StartTime,
	)
}

func StoreProfile(App *app.App, profile *api.UserProfile) error {
	if App.State.Verbose {
		fmt.Println(color.CyanString("Storing profile for: %s", profile.Email))
	}

	var crmFields []string
	for _, field := range profile.CRMEditableFieldsList {
		if field.Valid {
			crmFields = append(crmFields, field.String)
		}
	}
	crmEditableFieldsListStr := strings.Join(crmFields, ",")

	err := database.RunCommand(App.DB, "merge_user_profiles",
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

	if err := database.RunCommand(App.DB, "delete_data_set_values", profile.ProfileId); err != nil {
		return err
	}
	if err := database.RunCommand(App.DB, "delete_data_sets", profile.ProfileId); err != nil {
		return err
	}

	for _, datafield := range profile.Datafields {
		err := database.RunCommand(App.DB, "insert_data_sets",
			datafield.Name, profile.ProfileId, datafield.Filterable, datafield.Label, datafield.Position, datafield.Type,
			datafield.HasData, datafield.IsUserCanAddNewTextValues, datafield.RawMin, datafield.Min, datafield.Max,
			datafield.RawMax, datafield.AccountField,
		)
		if err != nil {
			return err
		}
		for _, value := range datafield.Values {
			err := database.RunCommand(App.DB, "insert_data_set_values",
				datafield.Name, profile.ProfileId, value.Text, value.Value, datafield.Position,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
