package app

import (
	"fmt"
)

// RunPullAll orchestrates the process of pulling all data from the BadgerMaps API.
// It accepts a logger function to provide real-time feedback to the caller (CLI or GUI).
func RunPullAll(a *App, log func(string)) error {
	log("Pulling all data...")

	log("Pulling accounts...")
	if err := PullAllAccounts(a, 0, log); err != nil {
		log(fmt.Sprintf("Error pulling accounts: %v", err))
		return err
	}

	log("Pulling checkins...")
	if err := PullAllCheckins(a, log); err != nil {
		log(fmt.Sprintf("Error pulling checkins: %v", err))
		return err
	}

	log("Pulling routes...")
	if err := PullAllRoutes(a, log); err != nil {
		log(fmt.Sprintf("Error pulling routes: %v", err))
		return err
	}

	log("Pulling user profile...")
	profile, err := a.API.GetUserProfile()
	if err != nil {
		log(fmt.Sprintf("Error pulling user profile: %v", err))
		return err
	}

	if err := StoreProfile(a, profile, log); err != nil {
		log(fmt.Sprintf("Error storing profile: %v", err))
		return err
	}
	log(fmt.Sprintf("Successfully pulled user profile for: %s", profile.Email.String))

	log("Finished pulling all data.")
	return nil
}

// RunPullAccount orchestrates pulling a single account from the API.
func RunPullAccount(a *App, accountID int, log func(string)) error {
	log(fmt.Sprintf("Pulling account with ID: %d", accountID))

	account, err := a.API.GetAccountDetailed(accountID)
	if err != nil {
		return fmt.Errorf("error pulling account: %w", err)
	}

	if err := StoreAccountDetailed(a, account, log); err != nil {
		return fmt.Errorf("error storing account: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled account with ID: %d", accountID))
	return nil
}

// RunPullAccounts orchestrates pulling all accounts from the API.
func RunPullAccounts(a *App, log func(string)) error {
	log("Pulling all accounts...")
	if err := PullAllAccounts(a, 0, log); err != nil {
		log(fmt.Sprintf("Error pulling accounts: %v", err))
		return err
	}
	log("Finished pulling all accounts.")
	return nil
}

// RunPullCheckin orchestrates pulling a single checkin from the API.
func RunPullCheckin(a *App, checkinID int, log func(string)) error {
	log(fmt.Sprintf("Pulling checkin with ID: %d", checkinID))

	checkin, err := a.API.GetCheckin(checkinID)
	if err != nil {
		return fmt.Errorf("error pulling checkin: %w", err)
	}

	if err := StoreCheckin(a, *checkin, log); err != nil {
		return fmt.Errorf("error storing checkin: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled checkin with ID: %d", checkinID))
	return nil
}

// RunPullCheckins orchestrates pulling all checkins from the API.
func RunPullCheckins(a *App, log func(string)) error {
	log("Pulling all checkins...")
	if err := PullAllCheckins(a, log); err != nil {
		log(fmt.Sprintf("Error pulling checkins: %v", err))
		return err
	}
	log("Finished pulling all checkins.")
	return nil
}

// RunPullRoute orchestrates pulling a single route from the API.
func RunPullRoute(a *App, routeID int, log func(string)) error {
	log(fmt.Sprintf("Pulling route with ID: %d", routeID))

	route, err := a.API.GetRoute(routeID)
	if err != nil {
		return fmt.Errorf("error pulling route: %w", err)
	}

	if err := StoreRoute(a, *route, log); err != nil {
		return fmt.Errorf("error storing route: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled route with ID: %d", routeID))
	return nil
}

// RunPullRoutes orchestrates pulling all routes from the API.
func RunPullRoutes(a *App, log func(string)) error {
	log("Pulling all routes...")
	if err := PullAllRoutes(a, log); err != nil {
		log(fmt.Sprintf("Error pulling routes: %v", err))
		return err
	}
	log("Finished pulling all routes.")
	return nil
}

// RunPullProfile orchestrates pulling the user profile from the API.
func RunPullProfile(a *App, log func(string)) error {
	log("Pulling user profile...")
	profile, err := a.API.GetUserProfile()
	if err != nil {
		return fmt.Errorf("error pulling user profile: %w", err)
	}

	if err := StoreProfile(a, profile, log); err != nil {
		return fmt.Errorf("error storing profile: %w", err)
	}

	log(fmt.Sprintf("Successfully pulled user profile for: %s", profile.Email.String))
	return nil
}
