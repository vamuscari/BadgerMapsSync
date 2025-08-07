package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MockServer represents the mock API server
type MockServer struct {
	port     int
	basePath string
}

// NewMockServer creates a new mock server instance
func NewMockServer(port int) *MockServer {
	return &MockServer{
		port:     port,
		basePath: "json",
	}
}

// Start initializes and starts the HTTP server
func (s *MockServer) Start() error {
	http.HandleFunc("/api/2/", s.handleAPIRequest)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock BadgerMaps API server starting on port %d", s.port)
	log.Printf("Base URL: http://localhost:%d", s.port)
	log.Printf("Available endpoints:")
	log.Printf("  GET  /api/2/profile/")
	log.Printf("  GET  /api/2/customers/")
	log.Printf("  GET  /api/2/customers/{id}/")
	log.Printf("  PATCH /api/2/customers/{id}/")
	log.Printf("  GET  /api/2/appointments/")
	log.Printf("  POST /api/2/appointments/")
	log.Printf("  GET  /api/2/routes/")
	log.Printf("  GET  /api/2/routes/{id}/")
	log.Printf("  GET  /api/2/search/users/")
	log.Printf("  GET  /api/2/datafields/")

	return http.ListenAndServe(addr, nil)
}

// handleAPIRequest routes API requests to appropriate handlers
func (s *MockServer) handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/2/")

	// Set CORS headers for testing
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")

	log.Printf("%s %s", r.Method, r.URL.Path)

	switch {
	case (path == "profile/" || path == "profiles/") && r.Method == "GET":
		s.handleProfile(w, r)
	case path == "customers/" && r.Method == "GET":
		s.handleCustomersList(w, r)
	case strings.HasPrefix(path, "customers/") && strings.HasSuffix(path, "/") && r.Method == "GET":
		s.handleCustomerDetail(w, r)
	case strings.HasPrefix(path, "customers/") && strings.HasSuffix(path, "/") && (r.Method == "PATCH" || r.Method == "POST"):
		s.handleCustomerUpdate(w, r)
	case path == "appointments/" && r.Method == "GET":
		s.handleCheckinsList(w, r)
	case path == "appointments/" && r.Method == "POST":
		s.handleCheckinCreate(w, r)
	case path == "routes/" && r.Method == "GET":
		s.handleRoutesList(w, r)
	case strings.HasPrefix(path, "routes/") && strings.HasSuffix(path, "/") && r.Method == "GET":
		s.handleRouteDetail(w, r)
	case strings.HasPrefix(path, "search/users/") && r.Method == "GET":
		s.handleUserSearch(w, r)
	case path == "datafields/" && r.Method == "GET":
		s.handleDataFields(w, r)
	default:
		s.handleNotFound(w, r)
	}
}

// handleProfile serves the profile endpoint
func (s *MockServer) handleProfile(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "profile_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleCustomersList serves the customers list endpoint
func (s *MockServer) handleCustomersList(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "accounts_list_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleCustomerDetail serves individual customer details
func (s *MockServer) handleCustomerDetail(w http.ResponseWriter, r *http.Request) {
	// Extract customer ID from URL
	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		s.handleError(w, fmt.Errorf("invalid customer ID"), http.StatusBadRequest)
		return
	}

	customerID := pathParts[len(pathParts)-1]

	// For now, always return the same customer detail
	// In a more sophisticated version, you could have multiple customer files
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "account_detail_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Update the ID in the response to match the requested ID
	var customer map[string]interface{}
	if err := json.Unmarshal(data, &customer); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	if id, err := strconv.Atoi(customerID); err == nil {
		customer["id"] = id

		// Update location IDs to be unique for each customer
		if locations, ok := customer["locations"].([]interface{}); ok {
			for i, location := range locations {
				if locMap, ok := location.(map[string]interface{}); ok {
					// Generate unique location ID based on customer ID and location index
					uniqueLocationID := id*1000 + i + 1
					locMap["id"] = uniqueLocationID
				}
			}
		}
	}

	responseData, err := json.Marshal(customer)
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Write(responseData)
}

// handleCustomerUpdate handles PATCH requests to update customers
func (s *MockServer) handleCustomerUpdate(w http.ResponseWriter, r *http.Request) {
	// Extract customer ID from URL
	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		s.handleError(w, fmt.Errorf("invalid customer ID"), http.StatusBadRequest)
		return
	}

	customerID := pathParts[len(pathParts)-1]

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.handleError(w, err, http.StatusBadRequest)
		return
	}

	// Parse the request body
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		s.handleError(w, err, http.StatusBadRequest)
		return
	}

	// Read the base customer detail response
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "account_detail_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Parse the response data
	var customer map[string]interface{}
	if err := json.Unmarshal(data, &customer); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Update the customer ID to match the requested ID
	if id, err := strconv.Atoi(customerID); err == nil {
		customer["id"] = id

		// Update location IDs to be unique for each customer
		if locations, ok := customer["locations"].([]interface{}); ok {
			for i, location := range locations {
				if locMap, ok := location.(map[string]interface{}); ok {
					// Generate unique location ID based on customer ID and location index
					uniqueLocationID := id*1000 + i + 1
					locMap["id"] = uniqueLocationID
				}
			}
		}
	}

	// Update customer fields with the request data
	for key, value := range requestData {
		if value != nil && value != "" {
			customer[key] = value
		}
	}

	// Add a timestamp to show the update was processed
	customer["last_modified_date"] = time.Now().Format("2006-01-02")

	// Return the updated customer data
	responseData, err := json.Marshal(customer)
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseData)
}

// handleCheckinsList serves check-ins for a customer
func (s *MockServer) handleCheckinsList(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer_id")
	if customerID == "" {
		s.handleError(w, fmt.Errorf("customer_id parameter required"), http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "account_checkins_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Update customer IDs in the response to match the requested customer
	var checkins []map[string]interface{}
	if err := json.Unmarshal(data, &checkins); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	if id, err := strconv.Atoi(customerID); err == nil {
		for _, checkin := range checkins {
			checkin["customer"] = id
		}
	}

	responseData, err := json.Marshal(checkins)
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Write(responseData)
}

// handleCheckinCreate handles POST requests to create check-ins
func (s *MockServer) handleCheckinCreate(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.handleError(w, err, http.StatusBadRequest)
		return
	}

	// Parse the request to extract customer ID
	var request map[string]interface{}
	if err := json.Unmarshal(body, &request); err != nil {
		s.handleError(w, err, http.StatusBadRequest)
		return
	}

	// Read the example response
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "checkin_response_example.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Update the response with the request data
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Update fields from the request
	if customer, ok := request["customer"]; ok {
		response["customer"] = customer
	}
	if logDatetime, ok := request["log_datetime"]; ok {
		response["log_datetime"] = logDatetime
	}
	if checkinType, ok := request["type"]; ok {
		response["type"] = checkinType
	}
	if comments, ok := request["comments"]; ok {
		response["comments"] = comments
	}
	if extraFields, ok := request["extra_fields"]; ok {
		response["extra_fields"] = extraFields
	}
	if createdBy, ok := request["created_by"]; ok {
		response["created_by"] = createdBy
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(responseData)
}

// handleRoutesList serves the routes list endpoint
func (s *MockServer) handleRoutesList(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "routes_list_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// handleRouteDetail serves individual route details
func (s *MockServer) handleRouteDetail(w http.ResponseWriter, r *http.Request) {
	// Extract route ID from URL
	pathParts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		s.handleError(w, fmt.Errorf("invalid route ID"), http.StatusBadRequest)
		return
	}

	routeID := pathParts[len(pathParts)-1]

	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "route_detail_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Update the ID in the response to match the requested ID
	var route map[string]interface{}
	if err := json.Unmarshal(data, &route); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	if id, err := strconv.Atoi(routeID); err == nil {
		route["id"] = id

		// Update waypoint IDs to be unique for each route
		if waypoints, ok := route["waypoints"].([]interface{}); ok {
			for i, waypoint := range waypoints {
				if wpMap, ok := waypoint.(map[string]interface{}); ok {
					// Generate unique waypoint ID based on route ID and waypoint index
					uniqueWaypointID := id*1000 + i + 1
					wpMap["id"] = uniqueWaypointID
				}
			}
		}
	}

	responseData, err := json.Marshal(route)
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Write(responseData)
}

// handleUserSearch serves user search results
func (s *MockServer) handleUserSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		s.handleError(w, fmt.Errorf("query parameter 'q' required"), http.StatusBadRequest)
		return
	}

	data, err := ioutil.ReadFile(filepath.Join(s.basePath, "search_users_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Filter results based on query (simple string matching)
	var users []map[string]interface{}
	if err := json.Unmarshal(data, &users); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	var filteredUsers []map[string]interface{}
	for _, user := range users {
		if firstName, ok := user["first_name"].(string); ok && strings.Contains(strings.ToLower(firstName), strings.ToLower(query)) {
			filteredUsers = append(filteredUsers, user)
		} else if username, ok := user["username"].(string); ok && strings.Contains(strings.ToLower(username), strings.ToLower(query)) {
			filteredUsers = append(filteredUsers, user)
		}
	}

	responseData, err := json.Marshal(filteredUsers)
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Write(responseData)
}

// handleDataFields serves data fields endpoint
func (s *MockServer) handleDataFields(w http.ResponseWriter, r *http.Request) {
	// Extract datafields from profile response
	profileData, err := ioutil.ReadFile(filepath.Join(s.basePath, "profile_response.json"))
	if err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	var profile map[string]interface{}
	if err := json.Unmarshal(profileData, &profile); err != nil {
		s.handleError(w, err, http.StatusInternalServerError)
		return
	}

	// Return only the datafields array
	if datafields, ok := profile["datafields"]; ok {
		responseData, err := json.Marshal(datafields)
		if err != nil {
			s.handleError(w, err, http.StatusInternalServerError)
			return
		}
		w.Write(responseData)
	} else {
		w.Write([]byte("[]"))
	}
}

// handleNotFound handles 404 errors
func (s *MockServer) handleNotFound(w http.ResponseWriter, r *http.Request) {
	errorResponse := map[string]interface{}{
		"error":       "Not Found",
		"message":     fmt.Sprintf("Endpoint %s not found", r.URL.Path),
		"status_code": 404,
	}

	responseData, err := json.Marshal(errorResponse)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	w.Write(responseData)
}

// handleError handles general errors
func (s *MockServer) handleError(w http.ResponseWriter, err error, statusCode int) {
	errorResponse := map[string]interface{}{
		"error":       http.StatusText(statusCode),
		"message":     err.Error(),
		"status_code": statusCode,
	}

	responseData, err := json.Marshal(errorResponse)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(statusCode)
	w.Write(responseData)
}

func main() {
	port := 8080
	if len(os.Args) > 1 {
		if p, err := strconv.Atoi(os.Args[1]); err == nil {
			port = p
		}
	}

	server := NewMockServer(port)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
