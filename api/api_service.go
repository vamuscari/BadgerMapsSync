package api

import (
	"encoding/json"
	"fmt"
	"github.com/guregu/null/v6"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultApiBaseURL = "https://badgerapis.badgermapping.com/api/2"
)

type APIConfig struct {
	BaseURL string `yaml:"api_url"`
	APIKey  string `yaml:"api_key"`
}

// APIClient handles BadgerMaps API interactions
type APIClient struct {
	BaseURL   string `mapstructure:"API_URL"`
	APIKey    string `mapstructure:"API_KEY"`
	UserID    int
	client    *http.Client
	endpoints *Endpoints
	connected bool
}

// NewAPIClient creates a new BadgerMaps API client
func NewAPIClient(config *APIConfig) *APIClient {
	client := &APIClient{
		BaseURL:   config.BaseURL,
		APIKey:    config.APIKey,
		client:    &http.Client{Timeout: 30 * time.Second},
		endpoints: NewEndpoints(config.BaseURL),
	}

	if err := client.TestAPIConnection(); err == nil {
		client.connected = true
	}

	return client
}

// IsConnected returns true if the client has successfully connected to the API
func (api *APIClient) IsConnected() bool {
	return api.connected
}

// SetConnected sets the connected status of the API client
func (api *APIClient) SetConnected(connected bool) {
	api.connected = connected
}


// All other API method receivers need to be updated from `api.apiKey` to `api.APIKey`
// Example for GetAccounts:
func (api *APIClient) GetAccounts() ([]Account, error) {
	url := api.endpoints.Customers()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to customers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint customers test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accounts []Account
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("failed to decode customers response: %w", err)
	}

	return accounts, nil
}

// Account represents a BadgerMaps account (customer)
type Account struct {
	AccountId            null.Int     `json:"id"`
	FirstName            *null.String `json:"first_name"`
	LastName             null.String  `json:"last_name"`
	FullName             null.String  `json:"full_name"`
	PhoneNumber          null.String  `json:"phone_number"`
	Email                null.String  `json:"email"`
	CustomerId           *null.String `json:"customer_id"`
	Notes                *null.String `json:"notes"`
	OriginalAddress      null.String  `json:"original_address"`
	CrmId                *null.String `json:"crm_id"`
	AccountOwner         *null.String `json:"account_owner"`
	DaysSinceLastCheckin null.Int     `json:"days_since_last_checkin"`
	LastCheckinDate      *null.String `json:"last_checkin_date"`
	LastModifiedDate     *null.String `json:"last_modified_date"`
	FollowUpDate         *null.String `json:"follow_up_date"`
	Locations            []Location   `json:"locations"`
	CustomNumeric        *null.Float  `json:"custom_numeric"`
	CustomText           *null.String `json:"custom_text"`
	CustomNumeric2       *null.Float  `json:"custom_numeric2"`
	CustomText2          *null.String `json:"custom_text2"`
	CustomNumeric3       *null.Float  `json:"custom_numeric3"`
	CustomText3          *null.String `json:"custom_text3"`
	CustomNumeric4       *null.Float  `json:"custom_numeric4"`
	CustomText4          *null.String `json:"custom_text4"`
	CustomNumeric5       *null.Float  `json:"custom_numeric5"`
	CustomText5          *null.String `json:"custom_text5"`
	CustomNumeric6       *null.Float  `json:"custom_numeric6"`
	CustomText6          *null.String `json:"custom_text6"`
	CustomNumeric7       *null.Float  `json:"custom_numeric7"`
	CustomText7          *null.String `json:"custom_text7"`
	CustomNumeric8       *null.Float  `json:"custom_numeric8"`
	CustomText8          *null.String `json:"custom_text8"`
	CustomNumeric9       *null.Float  `json:"custom_numeric9"`
	CustomText9          *null.String `json:"custom_text9"`
	CustomNumeric10      *null.Float  `json:"custom_numeric10"`
	CustomText10         *null.String `json:"custom_text10"`
	CustomNumeric11      *null.Float  `json:"custom_numeric11"`
	CustomText11         *null.String `json:"custom_text11"`
	CustomNumeric12      *null.Float  `json:"custom_numeric12"`
	CustomText12         *null.String `json:"custom_text12"`
	CustomNumeric13      *null.Float  `json:"custom_numeric13"`
	CustomText13         *null.String `json:"custom_text13"`
	CustomNumeric14      *null.Float  `json:"custom_numeric14"`
	CustomText14         *null.String `json:"custom_text14"`
	CustomNumeric15      *null.Float  `json:"custom_numeric15"`
	CustomText15         *null.String `json:"custom_text15"`
	CustomNumeric16      *null.Float  `json:"custom_numeric16"`
	CustomText16         *null.String `json:"custom_text16"`
	CustomNumeric17      *null.Float  `json:"custom_numeric17"`
	CustomText17         *null.String `json:"custom_text17"`
	CustomNumeric18      *null.Float  `json:"custom_numeric18"`
	CustomText18         *null.String `json:"custom_text18"`
	CustomNumeric19      *null.Float  `json:"custom_numeric19"`
	CustomText19         *null.String `json:"custom_text19"`
	CustomNumeric20      *null.Float  `json:"custom_numeric20"`
	CustomText20         *null.String `json:"custom_text20"`
	CustomNumeric21      *null.Float  `json:"custom_numeric21"`
	CustomText21         *null.String `json:"custom_text21"`
	CustomNumeric22      *null.Float  `json:"custom_numeric22"`
	CustomText22         *null.String `json:"custom_text22"`
	CustomNumeric23      *null.Float  `json:"custom_numeric23"`
	CustomText23         *null.String `json:"custom_text23"`
	CustomNumeric24      *null.Float  `json:"custom_numeric24"`
	CustomText24         *null.String `json:"custom_text24"`
	CustomNumeric25      *null.Float  `json:"custom_numeric25"`
	CustomText25         *null.String `json:"custom_text25"`
	CustomNumeric26      *null.Float  `json:"custom_numeric26"`
	CustomText26         *null.String `json:"custom_text26"`
	CustomNumeric27      *null.Float  `json:"custom_numeric27"`
	CustomText27         *null.String `json:"custom_text27"`
	CustomNumeric28      *null.Float  `json:"custom_numeric28"`
	CustomText28         *null.String `json:"custom_text28"`
	CustomNumeric29      *null.Float  `json:"custom_numeric29"`
	CustomText29         *null.String `json:"custom_text29"`
	CustomNumeric30      *null.Float  `json:"custom_numeric30"`
	CustomText30         *null.String `json:"custom_text30"`
	CreatedAt            null.String  `json:"created_at"`
	UpdatedAt            null.String  `json:"updated_at"`
}

// Location represents a BadgerMaps location
type Location struct {
	LocationId    null.Int     `json:"id"`
	City          null.String  `json:"city"`
	Name          *null.String `json:"name"`
	Zipcode       null.String  `json:"zipcode"`
	Long          null.Float   `json:"long"`
	State         null.String  `json:"state"`
	Lat           null.Float   `json:"lat"`
	AddressLine1  null.String  `json:"address_line_1"`
	Location      null.String  `json:"location"`
	IsApproximate null.Bool    `json:"is_approximate"`
}

// Route represents a BadgerMaps route
type Route struct {
	RouteId            null.Int    `json:"id"`
	Name               null.String `json:"name"`
	RouteDate          null.String `json:"route_date"`
	Duration           *null.Int   `json:"duration"`
	Waypoints          []Waypoint  `json:"waypoints"`
	StartAddress       null.String `json:"start_address"`
	DestinationAddress null.String `json:"destination_address"`
	StartTime          null.String `json:"start_time"`
}

// Wayponull.Int represents a route waypoint
type Waypoint struct {
	WaypointID      null.Int     `json:"id"`
	Name            null.String  `json:"name"`
	Address         null.String  `json:"address"`
	Suite           *null.String `json:"suite"`
	City            *null.String `json:"city"`
	State           *null.String `json:"state"`
	Zipcode         *null.String `json:"zipcode"`
	Location        null.String  `json:"location"`
	Lat             null.Float   `json:"lat"`
	Long            null.Float   `json:"long"`
	LayoverMinutes  null.Int     `json:"layover_minutes"`
	Position        null.Int     `json:"position"`
	CompleteAddress *null.String `json:"complete_address"`
	LocationID      null.Int     `json:"location_id"`
	CustomerID      null.Int     `json:"customer_id"`
	ApptTime        *null.String `json:"appt_time"`
	Type            null.Int     `json:"type"`
	PlaceID         *null.String `json:"place_id"`
}

// Checkin represents a BadgerMaps checkin (apponull.Intment)
type Checkin struct {
	CheckinId   null.Int     `json:"id"`
	CrmId       *null.String `json:"crm_id"`
	AccountId   null.Int     `json:"customer"` // Rename for clarity
	LogDatetime null.String  `json:"log_datetime"`
	Type        null.String  `json:"type"`
	Comments    null.String  `json:"comments"`
	ExtraFields *null.String `json:"extra_fields"`
	CreatedBy   null.String  `json:"created_by"`
}

// UserProfile represents a BadgerMaps user profile
type UserProfile struct {
	ProfileId                 null.Int      `json:"id"`
	Email                     null.String   `json:"email"`
	FirstName                 null.String   `json:"first_name"`
	LastName                  null.String   `json:"last_name"`
	IsManager                 null.Bool     `json:"is_manager"`
	IsHideReferralIOSBanner   null.Bool     `json:"is_hide_referral_ios_banner"`
	MarkerIcon                null.String   `json:"marker_icon"`
	Manager                   *null.String  `json:"manager"`
	CRMEditableFieldsList     []null.String `json:"crm_editable_fields_list"`
	CRMBaseURL                null.String   `json:"crm_base_url"`
	CRMType                   null.String   `json:"crm_type"`
	ReferralURL               null.String   `json:"referral_url"`
	MapStartZoom              null.Int      `json:"map_start_zoom"`
	MapStart                  null.String   `json:"map_start"`
	IsUserCanEdit             null.Bool     `json:"is_user_can_edit"`
	IsUserCanDeleteCheckins   null.Bool     `json:"is_user_can_delete_checkins"`
	IsUserCanAddNewTextValues null.Bool     `json:"is_user_can_add_new_text_values"`
	HasData                   null.Bool     `json:"has_data"`
	DefaultApptLength         null.Int      `json:"default_appt_length"`
	Completed                 null.Bool     `json:"completed"`
	TrialDaysLeft             null.Int      `json:"trial_days_left"`
	ApptlogFields             []DataField   `json:"apptlog_fields"`
	AcctlogFields             []DataField   `json:"acctlog_fields"`
	Datafields                []DataField   `json:"datafields"`
	Company                   Company       `json:"company"`
}

// Company represents a BadgerMaps company
type Company struct {
	Id        null.Int    `json:"id"`
	ShortName null.String `json:"short_name"`
	Name      null.String `json:"name"`
}

// DataField represents a custom data field
type DataField struct {
	Name                      null.String  `json:"name"`
	Filterable                null.Bool    `json:"filterable"`
	Label                     null.String  `json:"label"`
	Values                    []FieldValue `json:"values,omitempty"`
	Position                  null.Int     `json:"position"`
	Type                      null.String  `json:"type"`
	HasData                   null.Bool    `json:"has_data"`
	IsUserCanAddNewTextValues null.Bool    `json:"is_user_can_add_new_text_values"`
	RawMin                    *null.Float  `json:"rawmin,omitempty"`
	Min                       *null.Float  `json:"min,omitempty"`
	Max                       *null.Float  `json:"max,omitempty"`
	RawMax                    *null.Float  `json:"rawmax,omitempty"`
	AccountField              null.String  `json:"account_field"`
}

// FieldValue represents a field value option
type FieldValue struct {
	Text  null.String `json:"text"`
	Value interface{} `json:"value"`
}

// GetAccountIDs retrieves all account IDs from the BadgerMaps API
func (api *APIClient) GetAccountIDs() ([]int, error) {
	url := api.endpoints.Customers()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to customers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint customers test failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Use a temporary struct to decode only the ID
	var accounts []struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("failed to decode customers response: %w", err)
	}

	// Extract IDs into a slice of ints
	ids := make([]int, len(accounts))
	for i, acc := range accounts {
		ids[i] = acc.ID
	}

	return ids, nil
}

// GetCheckinIDs retrieves all checkin IDs from the BadgerMaps API
func (api *APIClient) GetCheckinIDs() ([]int, error) {
	url := api.endpoints.Appointments()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint appointments test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var checkins []struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&checkins); err != nil {
		return nil, fmt.Errorf("failed to decode appointments response: %w", err)
	}

	ids := make([]int, len(checkins))
	for i, checkin := range checkins {
		ids[i] = checkin.ID
	}

	return ids, nil
}

// GetRoutes retrieves all routes from the BadgerMaps API
func (api *APIClient) GetRoutes() ([]Route, error) {
	url := api.endpoints.Routes()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to routes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint routes test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var routes []Route
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("failed to decode routes response: %w", err)
	}

	return routes, nil
}

// GetCheckins retrieves all checkins from the BadgerMaps API
func (api *APIClient) GetCheckins() ([]Checkin, error) {
	url := api.endpoints.Appointments()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint appointments test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var checkins []Checkin
	if err := json.NewDecoder(resp.Body).Decode(&checkins); err != nil {
		return nil, fmt.Errorf("failed to decode appointments response: %w", err)
	}

	return checkins, nil
}

// GetUserProfile retrieves the current user's profile from the BadgerMaps API
func (api *APIClient) GetUserProfile() (*UserProfile, error) {
	url := api.endpoints.Profiles()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint profiles test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profiles response: %w", err)
	}

	for i, datafield := range profile.Datafields {
		if accountField, ok := DataSetAccountFieldMappings[datafield.Name.String]; ok {
			profile.Datafields[i].AccountField = null.StringFrom(accountField)
		}
	}

	return &profile, nil
}

// TestAPIConnection tests the API connectivity
func (api *APIClient) TestAPIConnection() error {
	// Skip health endpoint test since it doesn't exist
	// Instead, test with a simple API call to verify connectivity
	url := api.endpoints.Profiles()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create API test request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx status code as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var profile struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
			return fmt.Errorf("failed to decode profile response: %w", err)
		}
		api.UserID = profile.ID
		api.connected = true
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API test failed with status %d: %s", resp.StatusCode, string(body))
}

// GetRaw performs a GET request to the specified endpoint and returns the raw response body as a string.
func (api *APIClient) GetRaw(endpoint string) (string, error) {
	var url string
	if strings.Contains(endpoint, "?") {
		parts := strings.Split(endpoint, "?")
		endpointName := parts[0]
		queryParams := parts[1]
		url = fmt.Sprintf("%s?%s", api.endpoints.GetEndpoint(endpointName), queryParams)
	} else if strings.Contains(endpoint, "/") {
		url = fmt.Sprintf("%s/%s/", api.BaseURL, endpoint)
	} else {
		url = api.endpoints.GetEndpoint(endpoint)
		if url == "" {
			return "", fmt.Errorf("endpoint not found: %s", endpoint)
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for endpoint %s: %w", endpoint, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from endpoint %s: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("endpoint %s test failed with status %d", endpoint, resp.StatusCode)
	}

	return string(body), nil
}

// PostRaw performs a POST request to the specified endpoint and returns the raw response body as a string.
func (api *APIClient) PostRaw(endpoint string, data map[string]string) (string, error) {
	url := api.endpoints.GetEndpoint(endpoint)
	if url == "" {
		return "", fmt.Errorf("endpoint not found: %s", endpoint)
	}

	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request for endpoint %s: %w", endpoint, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from endpoint %s: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusCreated {
		return string(bodyBytes), fmt.Errorf("endpoint %s test failed with status %d", endpoint, resp.StatusCode)
	}

	return string(bodyBytes), nil
}

// PatchRaw performs a PATCH request to the specified endpoint and returns the raw response body as a string.
func (api *APIClient) PatchRaw(endpoint string, data map[string]string) (string, error) {
	var url string
	if strings.Contains(endpoint, "/") {
		url = fmt.Sprintf("%s/%s/", api.BaseURL, endpoint)
	} else {
		url = api.endpoints.GetEndpoint(endpoint)
		if url == "" {
			return "", fmt.Errorf("endpoint not found: %s", endpoint)
		}
	}

	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request for endpoint %s: %w", endpoint, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from endpoint %s: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		return string(bodyBytes), fmt.Errorf("endpoint %s test failed with status %d", endpoint, resp.StatusCode)
	}

	return string(bodyBytes), nil
}

// DeleteRaw performs a DELETE request to the specified endpoint and returns the raw response body as a string.
func (api *APIClient) DeleteRaw(endpoint string) (string, error) {
	var url string
	if strings.Contains(endpoint, "/") {
		url = fmt.Sprintf("%s/%s/", api.BaseURL, endpoint)
	} else {
		url = api.endpoints.GetEndpoint(endpoint)
		if url == "" {
			return "", fmt.Errorf("endpoint not found: %s", endpoint)
		}
	}

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for endpoint %s: %w", endpoint, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))

	resp, err := api.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from endpoint %s: %w", endpoint, err)
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("endpoint %s test failed with status %d", endpoint, resp.StatusCode)
	}

	return string(body), nil
}

// GetAccountDetailed retrieves a specific account by ID
func (api *APIClient) GetAccountDetailed(accountID int) (*Account, error) {
	url := api.endpoints.Customer(accountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to customer %d: %w", accountID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint customer %d test failed with status %d: %s", accountID, resp.StatusCode, string(body))
	}

	var account Account
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode customer response: %w", err)
	}

	return &account, nil
}

// UpdateAccount updates an account
func (api *APIClient) UpdateAccount(accountID int, data map[string]string) (*Account, error) {
	url := api.endpoints.Customer(accountID)

	// Convert data to form-encoded string
	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update customer %d: %w", accountID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update customer %d failed with status %d: %s", accountID, resp.StatusCode, string(body))
	}

	var account Account
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode customer response: %w", err)
	}

	return &account, nil
}

// DeleteAccount deletes an account
func (api *APIClient) DeleteAccount(accountID int) error {
	url := api.endpoints.Customer(accountID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete customer %d: %w", accountID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete customer %d failed with status %d: %s", accountID, resp.StatusCode, string(body))
	}

	return nil
}

// CreateAccount creates a new account
func (api *APIClient) CreateAccount(data map[string]string) (*Account, error) {
	url := api.endpoints.Customers()

	// Convert data to form-encoded string
	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create customer failed with status %d: %s", resp.StatusCode, string(body))
	}

	var account Account
	if err := json.NewDecoder(resp.Body).Decode(&account); err != nil {
		return nil, fmt.Errorf("failed to decode customer response: %w", err)
	}

	return &account, nil
}

// GetRoute retrieves a specific route by ID
func (api *APIClient) GetRoute(routeID int) (*Route, error) {
	url := api.endpoints.Route(routeID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to route %d: %w", routeID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint route %d test failed with status %d: %s", routeID, resp.StatusCode, string(body))
	}

	var route Route
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, fmt.Errorf("failed to decode route response: %w", err)
	}

	return &route, nil
}

// GetCheckinsForAccount retrieves checkins for a specific account
func (api *APIClient) GetCheckinsForAccount(customerID int) ([]Checkin, error) {
	url := api.endpoints.AppointmentsForCustomer(customerID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointments for customer %d: %w", customerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint appointments for customer %d test failed with status %d: %s", customerID, resp.StatusCode, string(body))
	}

	var checkins []Checkin
	if err := json.NewDecoder(resp.Body).Decode(&checkins); err != nil {
		return nil, fmt.Errorf("failed to decode appointments response: %w", err)
	}

	return checkins, nil
}

// CreateCheckin creates a new checkin for an account
func (api *APIClient) CreateCheckin(data map[string]string) (*Checkin, error) {
	url := api.endpoints.Appointments()

	// Convert data to form-encoded string
	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create appointment failed with status %d: %s", resp.StatusCode, string(body))
	}

	var checkin Checkin
	if err := json.NewDecoder(resp.Body).Decode(&checkin); err != nil {
		return nil, fmt.Errorf("failed to decode appointment response: %w", err)
	}

	return &checkin, nil
}

// UpdateLocation updates a location
func (api *APIClient) UpdateLocation(locationID int, data map[string]string) (*Location, error) {
	url := api.endpoints.Location(locationID)

	// Convert data to form-encoded string
	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update location %d: %w", locationID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update location %d failed with status %d: %s", locationID, resp.StatusCode, string(body))
	}

	var location Location
	if err := json.NewDecoder(resp.Body).Decode(&location); err != nil {
		return nil, fmt.Errorf("failed to decode location response: %w", err)
	}

	return &location, nil
}

// SearchUsers searches for users by email or ID
func (api *APIClient) SearchUsers(query string) (*UserProfile, error) {
	url := api.endpoints.SearchUsers(query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search users failed with status %d: %s", resp.StatusCode, string(body))
	}

	var user UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user search response: %w", err)
	}

	return &user, nil
}

// SearchAccounts searches for accounts by name or ID
func (api *APIClient) SearchAccounts(query string) ([]Account, error) {
	url := api.endpoints.SearchAccounts(query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search accounts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search accounts failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accounts []Account
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("failed to decode accounts search response: %w", err)
	}

	return accounts, nil
}

// SearchLocations searches for locations by name or ID
func (api *APIClient) SearchLocations(query string) ([]Location, error) {
	url := api.endpoints.SearchLocations(query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search locations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search locations failed with status %d: %s", resp.StatusCode, string(body))
	}

	var locations []Location
	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, fmt.Errorf("failed to decode locations search response: %w", err)
	}

	return locations, nil
}

// SearchProfiles searches for user profiles by name or ID
func (api *APIClient) SearchProfiles(query string) ([]UserProfile, error) {
	url := api.endpoints.SearchProfiles(query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search profiles failed with status %d: %s", resp.StatusCode, string(body))
	}

	var profiles []UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
		return nil, fmt.Errorf("failed to decode profiles search response: %w", err)
	}

	return profiles, nil
}

// GetCheckin retrieves a specific checkin by ID
func (api *APIClient) GetCheckin(checkinID int) (*Checkin, error) {
	url := api.endpoints.Appointment(checkinID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointment %d: %w", checkinID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint appointment %d test failed with status %d: %s", checkinID, resp.StatusCode, string(body))
	}

	var checkin Checkin
	if err := json.NewDecoder(resp.Body).Decode(&checkin); err != nil {
		return nil, fmt.Errorf("failed to decode appointment response: %w", err)
	}

	return &checkin, nil
}

// UpdateCheckin updates a checkin
func (api *APIClient) UpdateCheckin(checkinID int, data map[string]string) (*Checkin, error) {
	url := api.endpoints.Appointment(checkinID)

	// Convert data to form-encoded string
	formData := make([]string, 0, len(data))
	for key, value := range data {
		formData = append(formData, fmt.Sprintf("%s=%s", key, value))
	}
	body := strings.Join(formData, "&")

	req, err := http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update appointment %d: %w", checkinID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update appointment %d failed with status %d: %s", checkinID, resp.StatusCode, string(body))
	}

	var checkin Checkin
	if err := json.NewDecoder(resp.Body).Decode(&checkin); err != nil {
		return nil, fmt.Errorf("failed to decode appointment response: %w", err)
	}

	return &checkin, nil
}

// DeleteCheckin deletes a checkin
func (api *APIClient) DeleteCheckin(checkinID int) error {
	url := api.endpoints.Appointment(checkinID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete appointment %d: %w", checkinID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete appointment %d failed with status %d: %s", checkinID, resp.StatusCode, string(body))
	}

	return nil
}

func (api *APIClient) RawRequest(method, endpoint string, data map[string]string) ([]byte, error) {
	var body string
	var err error
	switch strings.ToUpper(method) {
	case "GET":
		body, err = api.GetRaw(endpoint)
	case "POST":
		body, err = api.PostRaw(endpoint, data)
	case "PATCH":
		body, err = api.PatchRaw(endpoint, data)
	case "DELETE":
		body, err = api.DeleteRaw(endpoint)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	if err != nil {
		return nil, err
	}
	return []byte(body), nil
}
