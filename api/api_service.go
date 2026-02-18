package api

import (
	"badgermaps/api/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/guregu/null/v6"
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

// encodeFormData converts a map into a URL-encoded form body.
// Using url.Values ensures keys like "extra_fields[Meeting Notes]" and
// values that contain spaces or special characters are encoded correctly.
func encodeFormData(data map[string]string) string {
	values := url.Values{}
	for key, value := range data {
		values.Set(key, value)
	}
	return values.Encode()
}

func hasNonNullValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	switch strings.ToLower(trimmed) {
	case "null", "nil", "<nil>":
		return false
	default:
		return true
	}
}

func canonicalCustomCheckinExtraFieldKey(key string) string {
	trimmed := strings.TrimSpace(key)
	switch strings.ToLower(trimmed) {
	case "log type":
		return "Log Type"
	case "meeting notes":
		return "Meeting Notes"
	default:
		return trimmed
	}
}

func parseRawExtraFields(raw string) (map[string]interface{}, error) {
	parsed := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}

	normalized := map[string]interface{}{}
	for key, value := range parsed {
		canonicalKey := canonicalCustomCheckinExtraFieldKey(key)
		if canonicalKey == "" {
			continue
		}
		if stringValue, ok := value.(string); ok && !hasNonNullValue(stringValue) {
			continue
		}
		normalized[canonicalKey] = value
	}

	return normalized, nil
}

func (api *APIClient) applyAuthHeaders(req *http.Request, contentType string) {
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", api.APIKey))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
}

func responsePreview(body []byte, max int) string {
	if len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "..."
}

func doJSON[T any](api *APIClient, req *http.Request, expectedStatus int, decodeErrPrefix string) (*APIResponse[T], error) {
	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, responsePreview(body, 500))
	}

	var data T
	if len(body) > 0 {
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("%s: %w\nResponse preview: %s", decodeErrPrefix, err, responsePreview(body, 500))
		}
	}

	return &APIResponse[T]{
		Data:       data,
		Raw:        body,
		StatusCode: resp.StatusCode,
		Headers:    resp.Header.Clone(),
	}, nil
}

// GetAccounts retrieves all accounts from the BadgerMaps API.
func (api *APIClient) GetAccounts() (*APIResponse[[]models.Account], error) {
	endpoint := api.endpoints.Customers()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[[]models.Account](api, req, http.StatusOK, "failed to decode customers response")
	if err != nil {
		return nil, fmt.Errorf("customers request failed: %w", err)
	}
	return result, nil
}

// GetAccountIDs retrieves all account IDs from the BadgerMaps API
func (api *APIClient) GetAccountIDs() (*APIResponse[[]int], error) {
	endpoint := api.endpoints.Customers()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	rawResult, err := doJSON[[]struct {
		ID int `json:"id"`
	}](api, req, http.StatusOK, "failed to decode customers response")
	if err != nil {
		return nil, fmt.Errorf("customers request failed: %w", err)
	}

	// Extract IDs into a slice of ints
	ids := make([]int, len(rawResult.Data))
	for i, acc := range rawResult.Data {
		ids[i] = acc.ID
	}

	return &APIResponse[[]int]{
		Data:       ids,
		Raw:        rawResult.Raw,
		StatusCode: rawResult.StatusCode,
		Headers:    rawResult.Headers,
	}, nil
}

// GetCheckinIDs retrieves all checkin IDs from the BadgerMaps API
func (api *APIClient) GetCheckinIDs() (*APIResponse[[]int], error) {
	endpoint := api.endpoints.Appointments()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	rawResult, err := doJSON[[]struct {
		ID int `json:"id"`
	}](api, req, http.StatusOK, "failed to decode appointments response")
	if err != nil {
		return nil, fmt.Errorf("appointments request failed: %w", err)
	}

	ids := make([]int, len(rawResult.Data))
	for i, checkin := range rawResult.Data {
		ids[i] = checkin.ID
	}

	return &APIResponse[[]int]{
		Data:       ids,
		Raw:        rawResult.Raw,
		StatusCode: rawResult.StatusCode,
		Headers:    rawResult.Headers,
	}, nil
}

// GetRoutes retrieves all routes from the BadgerMaps API
func (api *APIClient) GetRoutes() (*APIResponse[[]models.Route], error) {
	endpoint := api.endpoints.Routes()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[[]models.Route](api, req, http.StatusOK, "failed to decode routes response")
	if err != nil {
		return nil, fmt.Errorf("routes request failed: %w", err)
	}
	return result, nil
}

// GetCheckins retrieves all checkins from the BadgerMaps API
func (api *APIClient) GetCheckins() (*APIResponse[[]models.Checkin], error) {
	endpoint := api.endpoints.Appointments()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[[]models.Checkin](api, req, http.StatusOK, "failed to decode appointments response")
	if err != nil {
		return nil, fmt.Errorf("appointments request failed: %w", err)
	}
	return result, nil
}

// GetUserProfile retrieves the current user's profile from the BadgerMaps API
func (api *APIClient) GetUserProfile() (*APIResponse[models.UserProfile], error) {
	endpoint := api.endpoints.Profiles()
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[models.UserProfile](api, req, http.StatusOK, "failed to decode profiles response")
	if err != nil {
		return nil, fmt.Errorf("profiles request failed: %w", err)
	}

	for i, datafield := range result.Data.Datafields {
		if accountField, ok := DataSetAccountFieldMappings[datafield.Name.String]; ok {
			result.Data.Datafields[i].AccountField = null.StringFrom(accountField)
		}
	}

	return result, nil
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

// GetAccountDetailed retrieves a specific account by ID
func (api *APIClient) GetAccountDetailed(accountID int) (*APIResponse[models.Account], error) {
	endpoint := api.endpoints.Customer(accountID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[models.Account](api, req, http.StatusOK, "failed to decode customer response")
	if err != nil {
		return nil, fmt.Errorf("customer %d request failed: %w", accountID, err)
	}
	return result, nil
}

// UpdateAccount updates an account
func (api *APIClient) UpdateAccount(accountID int, input models.AccountUpload) (*APIResponse[models.Account], error) {
	endpoint := api.endpoints.Customer(accountID)
	if input.Fields == nil {
		input.Fields = map[string]string{}
	}

	// Convert data to form-encoded string
	body := encodeFormData(input.Fields)

	req, err := http.NewRequest("PATCH", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/x-www-form-urlencoded")

	result, err := doJSON[models.Account](api, req, http.StatusOK, "failed to decode customer response")
	if err != nil {
		return nil, fmt.Errorf("failed to update customer %d: %w", accountID, err)
	}

	return result, nil
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

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete customer %d failed with status %d: %s", accountID, resp.StatusCode, string(body))
	}

	return nil
}

// CreateAccount creates a new account
func (api *APIClient) CreateAccount(input models.AccountUpload) (*APIResponse[models.Account], error) {
	endpoint := api.endpoints.Customers()
	if input.Fields == nil {
		input.Fields = map[string]string{}
	}

	// Convert data to form-encoded string
	body := encodeFormData(input.Fields)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/x-www-form-urlencoded")

	result, err := doJSON[models.Account](api, req, http.StatusCreated, "failed to decode customer response")
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}
	return result, nil
}

// GetRoute retrieves a specific route by ID
func (api *APIClient) GetRoute(routeID int) (*APIResponse[models.Route], error) {
	endpoint := api.endpoints.Route(routeID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[models.Route](api, req, http.StatusOK, "failed to decode route response")
	if err != nil {
		return nil, fmt.Errorf("route %d request failed: %w", routeID, err)
	}
	return result, nil
}

// GetCheckinsForAccount retrieves checkins for a specific account
func (api *APIClient) GetCheckinsForAccount(customerID int) (*APIResponse[[]models.Checkin], error) {
	endpoint := api.endpoints.AppointmentsForCustomer(customerID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/json")

	result, err := doJSON[[]models.Checkin](api, req, http.StatusOK, "failed to decode appointments response")
	if err != nil {
		return nil, fmt.Errorf("appointments for customer %d request failed: %w", customerID, err)
	}
	return result, nil
}

// CreateCheckin creates a new checkin for an account.
// checkinType is required; fields with empty/null-like values are skipped.
func (api *APIClient) CreateCheckin(input models.CheckinUpload) (*APIResponse[models.Checkin], error) {
	customer := input.Customer
	checkinType := input.Type
	fields := input.Fields

	if customer <= 0 {
		return nil, fmt.Errorf("customer is required and must be > 0")
	}
	if !hasNonNullValue(checkinType) {
		return nil, fmt.Errorf("checkin type is required")
	}

	endpoint := api.endpoints.Appointments()
	form := map[string]string{
		"customer": strconv.Itoa(customer),
		"type":     strings.TrimSpace(checkinType),
	}
	for key, value := range fields {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" || normalized == "customer" || normalized == "type" {
			continue
		}
		if !hasNonNullValue(value) {
			continue
		}
		form[normalized] = value
	}

	// Convert data to form-encoded string
	body := encodeFormData(form)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/x-www-form-urlencoded")

	result, err := doJSON[models.Checkin](api, req, http.StatusCreated, "failed to decode appointment response")
	if err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}
	return result, nil
}

// CreateCustomCheckin creates a new custom checkin for an account.
// Do not use unless enables for Badger Account.
// Format for extra_fields -> {"Log Type":"Phone Call","Meeting Notes":"Notes"}
// Encoding for extra_fields -> customer=someNumericID&extra_fields=%7B%22Log%20Type%22%3A%22Phone%20Call%22%2C%22Meeting%20Notes%22%3A%22Notes%22%7D
// checkinType is required; fields with empty/null-like values are skipped.
func (api *APIClient) CreateCustomCheckin(input models.CustomCheckinUpload) (*APIResponse[models.Checkin], error) {
	customer := input.Customer
	checkinType := input.Type
	fields := input.Fields
	typedExtraFields := input.ExtraFields

	if customer <= 0 {
		return nil, fmt.Errorf("customer is required and must be > 0")
	}
	if !hasNonNullValue(checkinType) {
		return nil, fmt.Errorf("checkin type is required")
	}

	endpoint := api.endpoints.Appointments()

	form := map[string]string{
		"customer": strconv.Itoa(customer),
		"type":     strings.TrimSpace(checkinType),
	}
	var rawExtraFields string
	derivedExtraFields := map[string]string{}
	for key, value := range fields {
		normalized := strings.ToLower(strings.TrimSpace(key))
		switch normalized {
		case "", "customer", "type":
			continue
		case "extra_fields":
			if !hasNonNullValue(value) {
				continue
			}
			rawExtraFields = value
		case "comments", "log_datetime", "crm_id", "lat", "long", "created_by":
			if !hasNonNullValue(value) {
				continue
			}
			form[normalized] = value
		default:
			if !hasNonNullValue(value) {
				continue
			}
			canonicalKey := canonicalCustomCheckinExtraFieldKey(key)
			if canonicalKey == "" {
				continue
			}
			derivedExtraFields[canonicalKey] = value
		}
	}

	combinedExtraFields := map[string]interface{}{}
	if rawExtraFields != "" {
		parsedRawExtraFields, err := parseRawExtraFields(rawExtraFields)
		if err != nil {
			return nil, fmt.Errorf("failed to decode existing extra_fields JSON: %w", err)
		}
		for key, value := range parsedRawExtraFields {
			combinedExtraFields[key] = value
		}
	}

	combinedExtraFields["Log Type"] = strings.TrimSpace(checkinType)
	if typedExtraFields != nil {
		if hasNonNullValue(typedExtraFields.LogType) {
			combinedExtraFields["Log Type"] = strings.TrimSpace(typedExtraFields.LogType)
		}
		if hasNonNullValue(typedExtraFields.MeetingNotes) {
			combinedExtraFields["Meeting Notes"] = typedExtraFields.MeetingNotes
		}
	}
	for key, value := range derivedExtraFields {
		combinedExtraFields[key] = value
	}

	if len(combinedExtraFields) > 0 {
		extraFieldsJSON, err := json.Marshal(combinedExtraFields)
		if err != nil {
			return nil, fmt.Errorf("failed to encode merged extra_fields: %w", err)
		}
		form["extra_fields"] = string(extraFieldsJSON)
	}

	body := encodeFormData(form)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/x-www-form-urlencoded")

	result, err := doJSON[models.Checkin](api, req, http.StatusCreated, "failed to decode appointment response")
	if err != nil {
		return nil, fmt.Errorf("failed to create appointment: %w", err)
	}
	return result, nil
}

// UpdateLocation updates a location
func (api *APIClient) UpdateLocation(locationID int, input models.LocationUpload) (*APIResponse[models.Location], error) {
	endpoint := api.endpoints.Location(locationID)
	if input.Fields == nil {
		input.Fields = map[string]string{}
	}

	// Convert data to form-encoded string
	body := encodeFormData(input.Fields)

	req, err := http.NewRequest("PATCH", endpoint, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	api.applyAuthHeaders(req, "application/x-www-form-urlencoded")

	result, err := doJSON[models.Location](api, req, http.StatusOK, "failed to decode location response")
	if err != nil {
		return nil, fmt.Errorf("failed to update location %d: %w", locationID, err)
	}

	return result, nil
}
