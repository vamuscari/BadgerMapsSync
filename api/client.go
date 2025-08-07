package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	// BadgerMaps API base URL
	ApiBaseURL = "https://badgerapis.badgermapping.com/api/2"
)

// APIClient handles BadgerMaps API interactions
type APIClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewAPIClient creates a new BadgerMaps API client
func NewAPIClient(apiKey string) *APIClient {
	return NewAPIClientWithURL(apiKey, ApiBaseURL)
}

// NewAPIClientWithURL creates a new BadgerMaps API client with custom URL
func NewAPIClientWithURL(apiKey, baseURL string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Account represents a BadgerMaps account (customer)
type Account struct {
	ID              int        `json:"id"`
	FirstName       *string    `json:"first_name"`
	LastName        string     `json:"last_name"`
	FullName        string     `json:"full_name"`
	PhoneNumber     string     `json:"phone_number"`
	Email           string     `json:"email"`
	CustomerID      *string    `json:"customer_id"`
	Notes           *string    `json:"notes"`
	OriginalAddress string     `json:"original_address"`
	CRMID           *string    `json:"crm_id"`
	Locations       []Location `json:"locations"`
	CustomNumeric   *float64   `json:"custom_numeric"`
	CustomText      *string    `json:"custom_text"`
	CustomNumeric2  *float64   `json:"custom_numeric2"`
	CustomText2     *string    `json:"custom_text2"`
	CustomNumeric3  *float64   `json:"custom_numeric3"`
	CustomText3     *string    `json:"custom_text3"`
	CustomNumeric4  *float64   `json:"custom_numeric4"`
	CustomText4     *string    `json:"custom_text4"`
	CustomNumeric5  *float64   `json:"custom_numeric5"`
	CustomText5     *string    `json:"custom_text5"`
	CustomNumeric6  *float64   `json:"custom_numeric6"`
	CustomText6     *string    `json:"custom_text6"`
	CustomNumeric7  *float64   `json:"custom_numeric7"`
	CustomText7     *string    `json:"custom_text7"`
	CustomNumeric8  *float64   `json:"custom_numeric8"`
	CustomText8     *string    `json:"custom_text8"`
	CustomNumeric9  *float64   `json:"custom_numeric9"`
	CustomText9     *string    `json:"custom_text9"`
	CustomNumeric10 *float64   `json:"custom_numeric10"`
	CustomText10    *string    `json:"custom_text10"`
	CustomNumeric11 *float64   `json:"custom_numeric11"`
	CustomText11    *string    `json:"custom_text11"`
	CustomNumeric12 *float64   `json:"custom_numeric12"`
	CustomText12    *string    `json:"custom_text12"`
	CustomNumeric13 *float64   `json:"custom_numeric13"`
	CustomText13    *string    `json:"custom_text13"`
	CustomNumeric14 *float64   `json:"custom_numeric14"`
	CustomText14    *string    `json:"custom_text14"`
	CustomNumeric15 *float64   `json:"custom_numeric15"`
	CustomText15    *string    `json:"custom_text15"`
	CustomNumeric16 *float64   `json:"custom_numeric16"`
	CustomText16    *string    `json:"custom_text16"`
	CustomNumeric17 *float64   `json:"custom_numeric17"`
	CustomText17    *string    `json:"custom_text17"`
	CustomNumeric18 *float64   `json:"custom_numeric18"`
	CustomText18    *string    `json:"custom_text18"`
	CustomNumeric19 *float64   `json:"custom_numeric19"`
	CustomText19    *string    `json:"custom_text19"`
	CustomNumeric20 *float64   `json:"custom_numeric20"`
	CustomText20    *string    `json:"custom_text20"`
	CustomNumeric21 *float64   `json:"custom_numeric21"`
	CustomText21    *string    `json:"custom_text21"`
	CustomNumeric22 *float64   `json:"custom_numeric22"`
	CustomText22    *string    `json:"custom_text22"`
	CustomNumeric23 *float64   `json:"custom_numeric23"`
	CustomText23    *string    `json:"custom_text23"`
	CustomNumeric24 *float64   `json:"custom_numeric24"`
	CustomText24    *string    `json:"custom_text24"`
	CustomNumeric25 *float64   `json:"custom_numeric25"`
	CustomText25    *string    `json:"custom_text25"`
	CustomNumeric26 *float64   `json:"custom_numeric26"`
	CustomText26    *string    `json:"custom_text26"`
	CustomNumeric27 *float64   `json:"custom_numeric27"`
	CustomText27    *string    `json:"custom_text27"`
	CustomNumeric28 *float64   `json:"custom_numeric28"`
	CustomText28    *string    `json:"custom_text28"`
	CustomNumeric29 *float64   `json:"custom_numeric29"`
	CustomText29    *string    `json:"custom_text29"`
	CustomNumeric30 *float64   `json:"custom_numeric30"`
	CustomText30    *string    `json:"custom_text30"`
}

// Location represents a BadgerMaps location
type Location struct {
	ID           int     `json:"id"`
	City         string  `json:"city"`
	Name         *string `json:"name"`
	Zipcode      string  `json:"zipcode"`
	Long         float64 `json:"long"`
	State        string  `json:"state"`
	Lat          float64 `json:"lat"`
	AddressLine1 string  `json:"address_line_1"`
	Location     string  `json:"location"`
}

// Route represents a BadgerMaps route
type Route struct {
	ID                 int        `json:"id"`
	Name               string     `json:"name"`
	RouteDate          string     `json:"route_date"`
	Duration           *int       `json:"duration"`
	Waypoints          []Waypoint `json:"waypoints"`
	StartAddress       string     `json:"start_address"`
	DestinationAddress string     `json:"destination_address"`
	StartTime          string     `json:"start_time"`
}

// Waypoint represents a route waypoint
type Waypoint struct {
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	Suite           *string `json:"suite"`
	City            *string `json:"city"`
	State           *string `json:"state"`
	Zipcode         *string `json:"zipcode"`
	Location        string  `json:"location"`
	Lat             float64 `json:"lat"`
	Long            float64 `json:"long"`
	LayoverMinutes  int     `json:"layover_minutes"`
	Position        int     `json:"position"`
	CompleteAddress *string `json:"complete_address"`
	LocationID      int     `json:"location_id"`
	CustomerID      int     `json:"customer_id"`
	ApptTime        *string `json:"appt_time"`
	Type            int     `json:"type"`
	PlaceID         *string `json:"place_id"`
}

// Checkin represents a BadgerMaps checkin (appointment)
type Checkin struct {
	ID          int     `json:"id"`
	CRMID       *string `json:"crm_id"`
	Customer    int     `json:"customer"`
	LogDatetime string  `json:"log_datetime"`
	Type        string  `json:"type"`
	Comments    string  `json:"comments"`
	ExtraFields *string `json:"extra_fields"`
	CreatedBy   string  `json:"created_by"`
}

// UserProfile represents a BadgerMaps user profile
type UserProfile struct {
	ID                        int         `json:"id"`
	Email                     string      `json:"email"`
	FirstName                 string      `json:"first_name"`
	LastName                  string      `json:"last_name"`
	IsManager                 bool        `json:"is_manager"`
	IsHideReferralIOSBanner   bool        `json:"is_hide_referral_ios_banner"`
	MarkerIcon                string      `json:"marker_icon"`
	Manager                   *string     `json:"manager"`
	CRMEditableFieldsList     []string    `json:"crm_editable_fields_list"`
	CRMBaseURL                string      `json:"crm_base_url"`
	CRMType                   string      `json:"crm_type"`
	ReferralURL               string      `json:"referral_url"`
	MapStartZoom              int         `json:"map_start_zoom"`
	MapStart                  string      `json:"map_start"`
	IsUserCanEdit             bool        `json:"is_user_can_edit"`
	IsUserCanDeleteCheckins   bool        `json:"is_user_can_delete_checkins"`
	IsUserCanAddNewTextValues bool        `json:"is_user_can_add_new_text_values"`
	HasData                   bool        `json:"has_data"`
	DefaultApptLength         int         `json:"default_appt_length"`
	Completed                 bool        `json:"completed"`
	TrialDaysLeft             int         `json:"trial_days_left"`
	ApptlogFields             []DataField `json:"apptlog_fields"`
	AcctlogFields             []DataField `json:"acctlog_fields"`
	Datafields                []DataField `json:"datafields"`
	Company                   Company     `json:"company"`
}

// Company represents a BadgerMaps company
type Company struct {
	ID        int    `json:"id"`
	ShortName string `json:"short_name"`
	Name      string `json:"name"`
}

// DataField represents a custom data field
type DataField struct {
	Name                      string       `json:"name"`
	Filterable                bool         `json:"filterable"`
	Label                     string       `json:"label"`
	Values                    []FieldValue `json:"values,omitempty"`
	Position                  int          `json:"position"`
	Type                      string       `json:"type"`
	HasData                   bool         `json:"has_data"`
	IsUserCanAddNewTextValues bool         `json:"is_user_can_add_new_text_values"`
	RawMin                    *float64     `json:"rawmin,omitempty"`
	Min                       *float64     `json:"min,omitempty"`
	Max                       *float64     `json:"max,omitempty"`
	RawMax                    *float64     `json:"rawmax,omitempty"`
}

// FieldValue represents a field value option
type FieldValue struct {
	Text  string      `json:"text"`
	Value interface{} `json:"value"`
}

// GetAccounts retrieves all accounts from the BadgerMaps API
func (api *APIClient) GetAccounts() ([]Account, error) {
	url := fmt.Sprintf("%s/customers/", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to customers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint customers test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var accounts []Account
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return nil, fmt.Errorf("failed to decode customers response: %w", err)
	}

	return accounts, nil
}

// GetRoutes retrieves all routes from the BadgerMaps API
func (api *APIClient) GetRoutes() ([]Route, error) {
	url := fmt.Sprintf("%s/routes/", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to routes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/appointments/", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/profiles/", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to profiles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("endpoint profiles test failed with status %d: %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profiles response: %w", err)
	}

	return &profile, nil
}

// TestAPIConnection tests the API connectivity
func (api *APIClient) TestAPIConnection() error {
	// Skip health endpoint test since it doesn't exist
	// Instead, test with a simple API call to verify connectivity
	url := fmt.Sprintf("%s/profiles/", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create API test request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to API: %w", err)
	}
	defer resp.Body.Close()

	// Accept any 2xx status code as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := ioutil.ReadAll(resp.Body)
	return fmt.Errorf("API test failed with status %d: %s", resp.StatusCode, string(body))
}

// TestEndpoint tests a specific API endpoint
func (api *APIClient) TestEndpoint(endpoint string) error {
	url := fmt.Sprintf("%s/%s", api.baseURL, strings.TrimPrefix(endpoint, "/"))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create %s test request: %w", endpoint, err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("endpoint %s test failed with status %d: %s", endpoint, resp.StatusCode, string(body))
	}

	return nil
}

// TestAllEndpoints tests all API endpoints
func (api *APIClient) TestAllEndpoints() map[string]error {
	endpoints := []string{"customers", "routes", "appointments", "profiles"}
	results := make(map[string]error)

	for _, endpoint := range endpoints {
		results[endpoint] = api.TestEndpoint(endpoint)
	}

	return results
}

// GetAccount retrieves a specific account by ID
func (api *APIClient) GetAccount(accountID int) (*Account, error) {
	url := fmt.Sprintf("%s/customers/%d", api.baseURL, accountID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to customer %d: %w", accountID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/customers/%d", api.baseURL, accountID)

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

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update customer %d: %w", accountID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/customers/%d", api.baseURL, accountID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))

	resp, err := api.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete customer %d: %w", accountID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("delete customer %d failed with status %d: %s", accountID, resp.StatusCode, string(body))
	}

	return nil
}

// CreateAccount creates a new account
func (api *APIClient) CreateAccount(data map[string]string) (*Account, error) {
	url := fmt.Sprintf("%s/customers", api.baseURL)

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

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/routes/%d", api.baseURL, routeID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to route %d: %w", routeID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/appointments?customer_id=%d", api.baseURL, customerID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to appointments for customer %d: %w", customerID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/appointments", api.baseURL)

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

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/locations/%d", api.baseURL, locationID)

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

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update location %d: %w", locationID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
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
	url := fmt.Sprintf("%s/search/users/?q=%s", api.baseURL, query)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("search users failed with status %d: %s", resp.StatusCode, string(body))
	}

	var user UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user search response: %w", err)
	}

	return &user, nil
}
