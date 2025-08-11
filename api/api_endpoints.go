package api

import (
	"fmt"
	"net/url"
)

// Endpoints provides methods for building API endpoint URLs.
// This struct centralizes all API URL construction in one place for easier maintenance.
// Each method follows RESTful API design principles for endpoint naming.
type Endpoints struct {
	baseURL string // The base URL for the API
}

// NewEndpoints creates a new Endpoints instance with the given base URL.
// This allows for customizing the API base URL if needed.
//
// Parameters:
//   - baseURL: The base URL for the API (e.g., "https://badgerapis.badgermapping.com/api/2")
//
// Returns:
//   - *Endpoints: A new Endpoints instance
func NewEndpoints(baseURL string) *Endpoints {
	return &Endpoints{
		baseURL: baseURL,
	}
}

// DefaultEndpoints creates a new Endpoints instance with the default API base URL.
// This is a convenience method for creating an Endpoints instance with the default API base URL.
//
// Returns:
//   - *Endpoints: A new Endpoints instance with the default API base URL
func DefaultEndpoints() *Endpoints {
	return NewEndpoints(ApiBaseURL)
}

// Customers returns the URL for the customers endpoint.
// This endpoint returns a list of all customer accounts.
//
// Returns:
//   - string: The URL for the customers endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/customers/")
func (e *Endpoints) Customers() string {
	return fmt.Sprintf("%s/customers/", e.baseURL)
}

// Customer returns the URL for a specific customer by ID.
// This endpoint returns details for a single customer account.
//
// Parameters:
//   - id: The ID of the customer
//
// Returns:
//   - string: The URL for the specific customer (e.g., "https://badgerapis.badgermapping.com/api/2/customers/123/")
func (e *Endpoints) Customer(id int) string {
	return fmt.Sprintf("%s/customers/%d/", e.baseURL, id)
}

// Routes returns the URL for the routes endpoint.
// This endpoint returns a list of all routes.
//
// Returns:
//   - string: The URL for the routes endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/routes/")
func (e *Endpoints) Routes() string {
	return fmt.Sprintf("%s/routes/", e.baseURL)
}

// Route returns the URL for a specific route by ID.
// This endpoint returns details for a single route.
//
// Parameters:
//   - id: The ID of the route
//
// Returns:
//   - string: The URL for the specific route (e.g., "https://badgerapis.badgermapping.com/api/2/routes/123/")
func (e *Endpoints) Route(id int) string {
	return fmt.Sprintf("%s/routes/%d/", e.baseURL, id)
}

// Appointments returns the URL for the appointments endpoint.
// This endpoint returns a list of all appointments (check-ins).
//
// Returns:
//   - string: The URL for the appointments endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/appointments/")
func (e *Endpoints) Appointments() string {
	return fmt.Sprintf("%s/appointments/", e.baseURL)
}

// AppointmentsForCustomer returns the URL for appointments filtered by customer ID.
// This endpoint returns a list of appointments for a specific customer.
//
// Parameters:
//   - customerID: The ID of the customer
//
// Returns:
//   - string: The URL for appointments filtered by customer ID (e.g., "https://badgerapis.badgermapping.com/api/2/appointments/?customer_id=123")
func (e *Endpoints) AppointmentsForCustomer(customerID int) string {
	return fmt.Sprintf("%s/appointments/?customer_id=%d", e.baseURL, customerID)
}

// Appointment returns the URL for a specific appointment by ID.
// This endpoint returns details for a single appointment.
//
// Parameters:
//   - id: The ID of the appointment
//
// Returns:
//   - string: The URL for the specific appointment (e.g., "https://badgerapis.badgermapping.com/api/2/appointments/123")
func (e *Endpoints) Appointment(id int) string {
	return fmt.Sprintf("%s/appointments/%d", e.baseURL, id)
}

// Profiles returns the URL for the profiles endpoint.
// This endpoint returns a list of all user profiles.
//
// Returns:
//   - string: The URL for the profiles endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/profiles/")
func (e *Endpoints) Profiles() string {
	return fmt.Sprintf("%s/profiles/", e.baseURL)
}

// Profile returns the URL for a specific profile by ID.
// This endpoint returns details for a single user profile.
//
// Parameters:
//   - id: The ID of the profile
//
// Returns:
//   - string: The URL for the specific profile (e.g., "https://badgerapis.badgermapping.com/api/2/profiles/123")
func (e *Endpoints) Profile(id int) string {
	return fmt.Sprintf("%s/profiles/%d", e.baseURL, id)
}

// Locations returns the URL for the locations endpoint.
// This endpoint returns a list of all locations.
//
// Returns:
//   - string: The URL for the locations endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/locations/")
func (e *Endpoints) Locations() string {
	return fmt.Sprintf("%s/locations/", e.baseURL)
}

// Location returns the URL for a specific location by ID.
// This endpoint returns details for a single location.
//
// Parameters:
//   - id: The ID of the location
//
// Returns:
//   - string: The URL for the specific location (e.g., "https://badgerapis.badgermapping.com/api/2/locations/123")
func (e *Endpoints) Location(id int) string {
	return fmt.Sprintf("%s/locations/%d", e.baseURL, id)
}

// SearchUsers returns the URL for searching users.
// This endpoint searches for users by email or ID.
//
// Parameters:
//   - query: The search query string
//
// Returns:
//   - string: The URL for searching users (e.g., "https://badgerapis.badgermapping.com/api/2/search/users/?q=john")
func (e *Endpoints) SearchUsers(query string) string {
	encodedQuery := url.QueryEscape(query)
	return fmt.Sprintf("%s/search/users/?q=%s", e.baseURL, encodedQuery)
}

// SearchAccounts returns the URL for searching accounts.
// This endpoint searches for customer accounts by name or ID.
//
// Parameters:
//   - query: The search query string
//
// Returns:
//   - string: The URL for searching accounts (e.g., "https://badgerapis.badgermapping.com/api/2/search/accounts/?q=acme")
func (e *Endpoints) SearchAccounts(query string) string {
	encodedQuery := url.QueryEscape(query)
	return fmt.Sprintf("%s/search/accounts/?q=%s", e.baseURL, encodedQuery)
}

// SearchLocations returns the URL for searching locations.
// This endpoint searches for locations by name or ID.
//
// Parameters:
//   - query: The search query string
//
// Returns:
//   - string: The URL for searching locations (e.g., "https://badgerapis.badgermapping.com/api/2/search/locations/?q=san+francisco")
func (e *Endpoints) SearchLocations(query string) string {
	encodedQuery := url.QueryEscape(query)
	return fmt.Sprintf("%s/search/locations/?q=%s", e.baseURL, encodedQuery)
}

// SearchProfiles returns the URL for searching profiles.
// This endpoint searches for user profiles by name or ID.
//
// Parameters:
//   - query: The search query string
//
// Returns:
//   - string: The URL for searching profiles (e.g., "https://badgerapis.badgermapping.com/api/2/search/profiles/?q=smith")
func (e *Endpoints) SearchProfiles(query string) string {
	encodedQuery := url.QueryEscape(query)
	return fmt.Sprintf("%s/search/profiles/?q=%s", e.baseURL, encodedQuery)
}

// DataSets returns the URL for the data sets endpoint.
// This endpoint returns a list of all data sets.
//
// Returns:
//   - string: The URL for the data sets endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/data_sets/")
func (e *Endpoints) DataSets() string {
	return fmt.Sprintf("%s/data_sets/", e.baseURL)
}

// DataSet returns the URL for a specific data set by ID.
// This endpoint returns details for a single data set.
//
// Parameters:
//   - id: The ID of the data set
//
// Returns:
//   - string: The URL for the specific data set (e.g., "https://badgerapis.badgermapping.com/api/2/data_sets/123")
func (e *Endpoints) DataSet(id int) string {
	return fmt.Sprintf("%s/data_sets/%d", e.baseURL, id)
}

// DataSetValues returns the URL for the data set values endpoint.
// This endpoint returns a list of all data set values.
//
// Returns:
//   - string: The URL for the data set values endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/data_set_values/")
func (e *Endpoints) DataSetValues() string {
	return fmt.Sprintf("%s/data_set_values/", e.baseURL)
}

// DataSetValue returns the URL for a specific data set value by ID.
// This endpoint returns details for a single data set value.
//
// Parameters:
//   - id: The ID of the data set value
//
// Returns:
//   - string: The URL for the specific data set value (e.g., "https://badgerapis.badgermapping.com/api/2/data_set_values/123")
func (e *Endpoints) DataSetValue(id int) string {
	return fmt.Sprintf("%s/data_set_values/%d", e.baseURL, id)
}

// Waypoints returns the URL for the waypoints endpoint.
// This endpoint returns a list of all waypoints.
//
// Returns:
//   - string: The URL for the waypoints endpoint (e.g., "https://badgerapis.badgermapping.com/api/2/waypoints/")
func (e *Endpoints) Waypoints() string {
	return fmt.Sprintf("%s/waypoints/", e.baseURL)
}

// Waypoint returns the URL for a specific waypoint by ID.
// This endpoint returns details for a single waypoint.
//
// Parameters:
//   - id: The ID of the waypoint
//
// Returns:
//   - string: The URL for the specific waypoint (e.g., "https://badgerapis.badgermapping.com/api/2/waypoints/123")
func (e *Endpoints) Waypoint(id int) string {
	return fmt.Sprintf("%s/waypoints/%d", e.baseURL, id)
}

// WaypointsForRoute returns the URL for waypoints filtered by route ID.
// This endpoint returns a list of waypoints for a specific route.
//
// Parameters:
//   - routeID: The ID of the route
//
// Returns:
//   - string: The URL for waypoints filtered by route ID (e.g., "https://badgerapis.badgermapping.com/api/2/waypoints/?route_id=123")
func (e *Endpoints) WaypointsForRoute(routeID int) string {
	return fmt.Sprintf("%s/waypoints/?route_id=%d", e.baseURL, routeID)
}

// GetEndpoint returns the URL for a specific endpoint by name.
// This function uses a map to search for the endpoint by name.
//
// Parameters:
//   - name: The name of the endpoint (e.g., "customers", "routes", "appointments")
//
// Returns:
//   - string: The URL for the specified endpoint, or an empty string if the endpoint is not found
func (e *Endpoints) GetEndpoint(name string) string {
	// Map of endpoint names to their URL-generating functions
	endpointMap := map[string]func() string{
		"customers":       e.Customers,
		"routes":          e.Routes,
		"appointments":    e.Appointments,
		"profiles":        e.Profiles,
		"locations":       e.Locations,
		"waypoints":       e.Waypoints,
		"data_sets":       e.DataSets,
		"data_set_values": e.DataSetValues,
	}

	// Check if the endpoint exists in the map
	if fn, ok := endpointMap[name]; ok {
		return fn()
	}

	// Return empty string if endpoint not found
	return ""
}
