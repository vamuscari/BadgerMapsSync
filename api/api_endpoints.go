package api

import (
	"fmt"
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
	return &Endpoints{
		baseURL: DefaultApiBaseURL,
	}
}

// Login returns the URL for the login endpoint.
// This endpoint is used for user authentication.
func (e *Endpoints) Login() string {
	return fmt.Sprintf("%s/login/", e.baseURL)
}

// Customers returns the URL for the customers endpoint.
// This endpoint is used for listing or creating customer accounts.
func (e *Endpoints) Customers() string {
	return fmt.Sprintf("%s/customers/", e.baseURL)
}

// Customer returns the URL for a specific customer by ID.
// This endpoint is used for retrieving, updating, or deleting a single customer account.
func (e *Endpoints) Customer(id int) string {
	return fmt.Sprintf("%s/customers/%d/", e.baseURL, id)
}

// Routes returns the URL for the routes endpoint.
// This endpoint returns a list of all routes.
func (e *Endpoints) Routes() string {
	return fmt.Sprintf("%s/routes/", e.baseURL)
}

// Route returns the URL for a specific route by ID.
// This endpoint returns details for a single route.
func (e *Endpoints) Route(id int) string {
	return fmt.Sprintf("%s/routes/%d/", e.baseURL, id)
}

// Appointments returns the URL for the appointments endpoint.
// This endpoint is used for creating appointments (check-ins).
func (e *Endpoints) Appointments() string {
	return fmt.Sprintf("%s/appointments/", e.baseURL)
}

// AppointmentsForCustomer returns the URL for appointments filtered by customer ID.
// This endpoint returns a list of appointments for a specific customer.
func (e *Endpoints) AppointmentsForCustomer(customerID int) string {
	return fmt.Sprintf("%s/appointments/?customer_id=%d", e.baseURL, customerID)
}

// Profiles returns the URL for the profiles endpoint.
// This endpoint returns the profile for the currently logged-in user.
func (e *Endpoints) Profiles() string {
	return fmt.Sprintf("%s/profiles/", e.baseURL)
}

// Location returns the URL for a specific location by ID.
// This endpoint is used for updating a single location.
func (e *Endpoints) Location(id int) string {
	return fmt.Sprintf("%s/locations/%d/", e.baseURL, id)
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
		"login":        e.Login,
		"customers":    e.Customers,
		"routes":       e.Routes,
		"appointments": e.Appointments,
		"profiles":     e.Profiles,
	}

	// Check if the endpoint exists in the map
	if fn, ok := endpointMap[name]; ok {
		return fn()
	}

	// Return empty string if endpoint not found
	return ""
}
