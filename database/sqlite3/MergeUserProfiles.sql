INSERT OR REPLACE INTO UserProfiles (
    ProfileId, Email, FirstName, LastName, IsManager, IsHideReferralIOSBanner, MarkerIcon, Manager,
    CRMEditableFieldsList, CRMBaseUrl, CRMType, ReferralURL, MapStartZoom, MapStart, IsUserCanEdit,
    IsUserCanDeleteCheckins, IsUserCanAddNewTextValues, HasData, DefaultApptLength, Completed, TrialDaysLeft,
    CompanyId, CompanyName, CompanyShortName, UpdatedAt
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);