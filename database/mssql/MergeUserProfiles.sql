MERGE UserProfiles AS target
USING (SELECT ? as ProfileId, ? as Email, ? as FirstName, ? as LastName, ? as IsManager, ? as IsHideReferralIOSBanner,
       ? as MarkerIcon, ? as Manager, ? as CRMEditableFieldsList, ? as CRMBaseUrl, ? as CRMType, ? as ReferralURL, ? as MapStartZoom,
       ? as MapStart, ? as IsUserCanEdit, ? as IsUserCanDeleteCheckins, ? as IsUserCanAddNewTextValues,
       ? as HasData, ? as DefaultApptLength, ? as Completed, ? as TrialDaysLeft, ? as CompanyId,
       ? as CompanyName, ? as CompanyShortName) AS source
ON target.ProfileId = source.ProfileId
WHEN MATCHED THEN
    UPDATE SET
        Email = source.Email,
        FirstName = source.FirstName,
        LastName = source.LastName,
        IsManager = source.IsManager,
        IsHideReferralIOSBanner = source.IsHideReferralIOSBanner,
        MarkerIcon = source.MarkerIcon,
        Manager = source.Manager,
        CRMEditableFieldsList = source.CRMEditableFieldsList,
        CRMBaseUrl = source.CRMBaseUrl,
        CRMType = source.CRMType,
        ReferralURL = source.ReferralURL,
        MapStartZoom = source.MapStartZoom,
        MapStart = source.MapStart,
        IsUserCanEdit = source.IsUserCanEdit,
        IsUserCanDeleteCheckins = source.IsUserCanDeleteCheckins,
        IsUserCanAddNewTextValues = source.IsUserCanAddNewTextValues,
        HasData = source.HasData,
        DefaultApptLength = source.DefaultApptLength,
        Completed = source.Completed,
        TrialDaysLeft = source.TrialDaysLeft,
        CompanyId = source.CompanyId,
        CompanyName = source.CompanyName,
        CompanyShortName = source.CompanyShortName,
        UpdatedAt = GETDATE()
WHEN NOT MATCHED THEN
    INSERT (ProfileId, Email, FirstName, LastName, IsManager, IsHideReferralIOSBanner, MarkerIcon, Manager,
            CRMEditableFieldsList, CRMBaseUrl, CRMType, ReferralURL, MapStartZoom, MapStart, IsUserCanEdit,
            IsUserCanDeleteCheckins, IsUserCanAddNewTextValues, HasData, DefaultApptLength, Completed, TrialDaysLeft,
            CompanyId, CompanyName, CompanyShortName)
    VALUES (source.ProfileId, source.Email, source.FirstName, source.LastName, source.IsManager,
            source.IsHideReferralIOSBanner, source.MarkerIcon, source.Manager, source.CRMEditableFieldsList,
            source.CRMBaseUrl, source.CRMType, source.ReferralURL, source.MapStartZoom, source.MapStart,
            source.IsUserCanEdit, source.IsUserCanDeleteCheckins, source.IsUserCanAddNewTextValues,
            source.HasData, source.DefaultApptLength, source.Completed, source.TrialDaysLeft,
            source.CompanyId, source.CompanyName, source.CompanyShortName); 