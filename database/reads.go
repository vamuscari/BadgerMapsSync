package database

import (
	"badgermaps/api"
	"database/sql"
	"fmt"
)

func GetAccountByID(db DB, accountID int) (*api.Account, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_account_by_id")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_account_by_id")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	var account api.Account
	err = sqlDB.QueryRow(sqlText, accountID).Scan(
		&account.AccountId, &account.FirstName, &account.LastName, &account.FullName, &account.PhoneNumber,
		&account.Email, &account.CustomerId, &account.Notes, &account.OriginalAddress, &account.CrmId,
		&account.AccountOwner, &account.DaysSinceLastCheckin, &account.LastCheckinDate, &account.LastModifiedDate,
		&account.FollowUpDate, &account.CustomNumeric, &account.CustomText, &account.CustomNumeric2,
		&account.CustomText2, &account.CustomNumeric3, &account.CustomText3, &account.CustomNumeric4,
		&account.CustomText4, &account.CustomNumeric5, &account.CustomText5, &account.CustomNumeric6,
		&account.CustomText6, &account.CustomNumeric7, &account.CustomText7, &account.CustomNumeric8,
		&account.CustomText8, &account.CustomNumeric9, &account.CustomText9, &account.CustomNumeric10,
		&account.CustomText10, &account.CustomNumeric11, &account.CustomText11, &account.CustomNumeric12,
		&account.CustomText12, &account.CustomNumeric13, &account.CustomText13, &account.CustomNumeric14,
		&account.CustomText14, &account.CustomNumeric15, &account.CustomText15, &account.CustomNumeric16,
		&account.CustomText16, &account.CustomNumeric17, &account.CustomText17, &account.CustomNumeric18,
		&account.CustomText18, &account.CustomNumeric19, &account.CustomText19, &account.CustomNumeric20,
		&account.CustomText20, &account.CustomNumeric21, &account.CustomText21, &account.CustomNumeric22,
		&account.CustomText22, &account.CustomNumeric23, &account.CustomText23, &account.CustomNumeric24,
		&account.CustomText24, &account.CustomNumeric25, &account.CustomText25, &account.CustomNumeric26,
		&account.CustomText26, &account.CustomNumeric27, &account.CustomText27, &account.CustomNumeric28,
		&account.CustomText28, &account.CustomNumeric29, &account.CustomText29, &account.CustomNumeric30,
		&account.CustomText30, &account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func GetCheckinByID(db DB, checkinID int) (*api.Checkin, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_checkin_by_id")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_checkin_by_id")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	var checkin api.Checkin
	err = sqlDB.QueryRow(sqlText, checkinID).Scan(
		&checkin.CheckinId, &checkin.CrmId, &checkin.AccountId, &checkin.LogDatetime, &checkin.Type,
		&checkin.Comments, &checkin.ExtraFields, &checkin.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &checkin, nil
}

func GetRouteByID(db DB, routeID int) (*api.Route, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_route_by_id")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_route_by_id")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	var route api.Route
	err = sqlDB.QueryRow(sqlText, routeID).Scan(
		&route.RouteId, &route.Name, &route.RouteDate, &route.Duration, &route.StartAddress,
		&route.DestinationAddress, &route.StartTime,
	)
	if err != nil {
		return nil, err
	}
	return &route, nil
}

func GetProfile(db DB) (*api.UserProfile, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_profile")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_profile")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	var profile api.UserProfile
	err = sqlDB.QueryRow(sqlText).Scan(
		&profile.ProfileId, &profile.Email, &profile.FirstName, &profile.LastName, &profile.IsManager,
		&profile.IsHideReferralIOSBanner, &profile.MarkerIcon, &profile.Manager, &profile.CRMEditableFieldsList,
		&profile.CRMBaseURL, &profile.CRMType, &profile.ReferralURL, &profile.MapStartZoom, &profile.MapStart,
		&profile.IsUserCanEdit, &profile.IsUserCanDeleteCheckins, &profile.IsUserCanAddNewTextValues,
		&profile.HasData, &profile.DefaultApptLength, &profile.Completed, &profile.TrialDaysLeft,
		&profile.Company.Id, &profile.Company.Name, &profile.Company.ShortName,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func GetAllAccountIDs(db DB) ([]int, error) {
	sqlText := sqlCommandLoader(db.GetType(), "get_all_account_ids")
	if sqlText == "" {
		return nil, fmt.Errorf("unknown or unavailable SQL command: get_all_account_ids")
	}

	sqlDB, err := sql.Open(db.GetType(), db.DatabaseConnection())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sqlDB.Close()

	rows, err := sqlDB.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
