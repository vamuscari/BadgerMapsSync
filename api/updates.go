package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// UpdateRoute updates a route
func (api *APIClient) UpdateRoute(routeID int, data map[string]string) (*Route, error) {
	url := api.endpoints.Route(routeID)

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
		return nil, fmt.Errorf("failed to update route %d: %w", routeID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update route %d failed with status %d: %s", routeID, resp.StatusCode, string(body))
	}

	var route Route
	if err := json.NewDecoder(resp.Body).Decode(&route); err != nil {
		return nil, fmt.Errorf("failed to decode route response: %w", err)
	}

	return &route, nil
}

// UpdateProfile updates the user profile
func (api *APIClient) UpdateProfile(data map[string]string) (*UserProfile, error) {
	url := api.endpoints.Profiles()

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
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update profile failed with status %d: %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode profile response: %w", err)
	}

	return &profile, nil
}
