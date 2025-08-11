package api

import (
	"testing"
)

func TestEndpoints(t *testing.T) {
	// Create a new Endpoints instance with a base URL
	baseURL := "https://api.example.com/api/2"
	endpoints := NewEndpoints(baseURL)

	// Test DefaultEndpoints
	defaultEndpoints := DefaultEndpoints()
	if defaultEndpoints.baseURL != ApiBaseURL {
		t.Errorf("DefaultEndpoints() baseURL = %v, want %v", defaultEndpoints.baseURL, ApiBaseURL)
	}

	// Test Customers
	expected := "https://api.example.com/api/2/customers/"
	result := endpoints.Customers()
	if result != expected {
		t.Errorf("Customers() = %v, want %v", result, expected)
	}

	// Test Customer
	expected = "https://api.example.com/api/2/customers/123/"
	result = endpoints.Customer(123)
	if result != expected {
		t.Errorf("Customer(123) = %v, want %v", result, expected)
	}

	// Test Routes
	expected = "https://api.example.com/api/2/routes/"
	result = endpoints.Routes()
	if result != expected {
		t.Errorf("Routes() = %v, want %v", result, expected)
	}

	// Test Route
	expected = "https://api.example.com/api/2/routes/456/"
	result = endpoints.Route(456)
	if result != expected {
		t.Errorf("Route(456) = %v, want %v", result, expected)
	}

	// Test Appointments
	expected = "https://api.example.com/api/2/appointments/"
	result = endpoints.Appointments()
	if result != expected {
		t.Errorf("Appointments() = %v, want %v", result, expected)
	}

	// Test AppointmentsForCustomer
	expected = "https://api.example.com/api/2/appointments/?customer_id=789"
	result = endpoints.AppointmentsForCustomer(789)
	if result != expected {
		t.Errorf("AppointmentsForCustomer(789) = %v, want %v", result, expected)
	}

	// Test Appointment
	expected = "https://api.example.com/api/2/appointments/101"
	result = endpoints.Appointment(101)
	if result != expected {
		t.Errorf("Appointment(101) = %v, want %v", result, expected)
	}

	// Test Profiles
	expected = "https://api.example.com/api/2/profiles/"
	result = endpoints.Profiles()
	if result != expected {
		t.Errorf("Profiles() = %v, want %v", result, expected)
	}

	// Test Profile
	expected = "https://api.example.com/api/2/profiles/202"
	result = endpoints.Profile(202)
	if result != expected {
		t.Errorf("Profile(202) = %v, want %v", result, expected)
	}

	// Test Locations
	expected = "https://api.example.com/api/2/locations/"
	result = endpoints.Locations()
	if result != expected {
		t.Errorf("Locations() = %v, want %v", result, expected)
	}

	// Test Location
	expected = "https://api.example.com/api/2/locations/303"
	result = endpoints.Location(303)
	if result != expected {
		t.Errorf("Location(303) = %v, want %v", result, expected)
	}

	// Test SearchUsers
	expected = "https://api.example.com/api/2/search/users/?q=john"
	result = endpoints.SearchUsers("john")
	if result != expected {
		t.Errorf("SearchUsers(\"john\") = %v, want %v", result, expected)
	}

	// Test SearchAccounts
	expected = "https://api.example.com/api/2/search/accounts/?q=acme"
	result = endpoints.SearchAccounts("acme")
	if result != expected {
		t.Errorf("SearchAccounts(\"acme\") = %v, want %v", result, expected)
	}

	// Test SearchLocations
	expected = "https://api.example.com/api/2/search/locations/?q=san+francisco"
	result = endpoints.SearchLocations("san francisco")
	if result != expected {
		t.Errorf("SearchLocations(\"san francisco\") = %v, want %v", result, expected)
	}

	// Test SearchProfiles
	expected = "https://api.example.com/api/2/search/profiles/?q=smith"
	result = endpoints.SearchProfiles("smith")
	if result != expected {
		t.Errorf("SearchProfiles(\"smith\") = %v, want %v", result, expected)
	}

	// Test DataSets
	expected = "https://api.example.com/api/2/data_sets/"
	result = endpoints.DataSets()
	if result != expected {
		t.Errorf("DataSets() = %v, want %v", result, expected)
	}

	// Test DataSet
	expected = "https://api.example.com/api/2/data_sets/404"
	result = endpoints.DataSet(404)
	if result != expected {
		t.Errorf("DataSet(404) = %v, want %v", result, expected)
	}

	// Test DataSetValues
	expected = "https://api.example.com/api/2/data_set_values/"
	result = endpoints.DataSetValues()
	if result != expected {
		t.Errorf("DataSetValues() = %v, want %v", result, expected)
	}

	// Test DataSetValue
	expected = "https://api.example.com/api/2/data_set_values/505"
	result = endpoints.DataSetValue(505)
	if result != expected {
		t.Errorf("DataSetValue(505) = %v, want %v", result, expected)
	}

	// Test Waypoints
	expected = "https://api.example.com/api/2/waypoints/"
	result = endpoints.Waypoints()
	if result != expected {
		t.Errorf("Waypoints() = %v, want %v", result, expected)
	}

	// Test Waypoint
	expected = "https://api.example.com/api/2/waypoints/606"
	result = endpoints.Waypoint(606)
	if result != expected {
		t.Errorf("Waypoint(606) = %v, want %v", result, expected)
	}

	// Test WaypointsForRoute
	expected = "https://api.example.com/api/2/waypoints/?route_id=707"
	result = endpoints.WaypointsForRoute(707)
	if result != expected {
		t.Errorf("WaypointsForRoute(707) = %v, want %v", result, expected)
	}

	// Test GetEndpoint with valid endpoint names
	testCases := map[string]string{
		"customers":       "https://api.example.com/api/2/customers/",
		"routes":          "https://api.example.com/api/2/routes/",
		"appointments":    "https://api.example.com/api/2/appointments/",
		"profiles":        "https://api.example.com/api/2/profiles/",
		"locations":       "https://api.example.com/api/2/locations/",
		"waypoints":       "https://api.example.com/api/2/waypoints/",
		"data_sets":       "https://api.example.com/api/2/data_sets/",
		"data_set_values": "https://api.example.com/api/2/data_set_values/",
	}

	for name, expected := range testCases {
		result = endpoints.GetEndpoint(name)
		if result != expected {
			t.Errorf("GetEndpoint(%q) = %v, want %v", name, result, expected)
		}
	}

	// Test GetEndpoint with invalid endpoint name
	result = endpoints.GetEndpoint("invalid_endpoint")
	if result != "" {
		t.Errorf("GetEndpoint(\"invalid_endpoint\") = %v, want %v", result, "")
	}
}
