package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TestPushFunctionality tests the complete push workflow
func TestPushFunctionality(t *testing.T) {
	// Setup test database
	dbPath := "test_push.db"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Create test tables
	if err := createTestTables(db); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	// Insert test data
	if err := insertTestData(db); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create mock server
	server := NewMockServer(0) // Use port 0 for testing
	ts := httptest.NewServer(http.HandlerFunc(server.handleAPIRequest))
	defer ts.Close()

	// Run tests
	t.Run("TestCustomerUpdate", func(t *testing.T) {
		testCustomerUpdate(t, ts.URL)
	})

	t.Run("TestPushFromDatabase", func(t *testing.T) {
		testPushFromDatabase(t, db, ts.URL)
	})

	t.Run("TestInvalidRequests", func(t *testing.T) {
		testInvalidRequests(t, ts.URL)
	})
}

// testCustomerUpdate tests the PATCH endpoint directly
func testCustomerUpdate(t *testing.T, baseURL string) {
	tests := []struct {
		name           string
		customerID     string
		requestData    map[string]interface{}
		expectedStatus int
		expectedFields map[string]interface{}
	}{
		{
			name:       "Update customer name and email",
			customerID: "135299181",
			requestData: map[string]interface{}{
				"first_name": "Updated",
				"last_name":  "Store Name",
				"email":      "updated@example.com",
			},
			expectedStatus: http.StatusOK,
			expectedFields: map[string]interface{}{
				"first_name": "Updated",
				"last_name":  "Store Name",
				"email":      "updated@example.com",
			},
		},
		{
			name:       "Update phone number",
			customerID: "135299181",
			requestData: map[string]interface{}{
				"phone_number": "+1-555-999-8888",
			},
			expectedStatus: http.StatusOK,
			expectedFields: map[string]interface{}{
				"phone_number": "+1-555-999-8888",
			},
		},
		{
			name:       "Update address",
			customerID: "135299181",
			requestData: map[string]interface{}{
				"original_address": "456 New Street, New City, NY 10002",
			},
			expectedStatus: http.StatusOK,
			expectedFields: map[string]interface{}{
				"original_address": "456 New Street, New City, NY 10002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			requestJSON, err := json.Marshal(tt.requestData)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			url := fmt.Sprintf("%s/api/2/customers/%s/", baseURL, tt.customerID)
			req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(requestJSON))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Parse response
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check customer ID
			expectedID, _ := strconv.Atoi(tt.customerID)
			if id, ok := response["id"]; !ok {
				t.Error("Response missing 'id' field")
			} else {
				if int(id.(float64)) != expectedID {
					t.Errorf("Expected ID %d, got %v", expectedID, id)
				}
			}

			// Check updated fields
			for field, expectedValue := range tt.expectedFields {
				if value, ok := response[field]; !ok {
					t.Errorf("Response missing field '%s'", field)
				} else if value != expectedValue {
					t.Errorf("Field '%s': expected %v, got %v", field, expectedValue, value)
				}
			}

			// Check last_modified_date was updated
			if lastModified, ok := response["last_modified_date"]; !ok {
				t.Error("Response missing 'last_modified_date' field")
			} else {
				// Should be today's date
				today := time.Now().Format("2006-01-02")
				if lastModified != today {
					t.Errorf("Expected last_modified_date %s, got %v", today, lastModified)
				}
			}

			// Check locations have unique IDs
			if locations, ok := response["locations"].([]interface{}); ok {
				for i, location := range locations {
					if locMap, ok := location.(map[string]interface{}); ok {
						if locID, ok := locMap["id"]; ok {
							expectedLocID := expectedID*1000 + i + 1
							if int(locID.(float64)) != expectedLocID {
								t.Errorf("Location %d: expected ID %d, got %v", i, expectedLocID, locID)
							}
						}
					}
				}
			}
		})
	}
}

// testPushFromDatabase tests pushing data from database to API
func testPushFromDatabase(t *testing.T, db *sql.DB, baseURL string) {
	// Query test data from database
	rows, err := db.Query(`
		SELECT 
			Id as CustomerID,
			FirstName,
			LastName,
			Email,
			PhoneNumber
		FROM Accounts 
		WHERE Id IS NOT NULL 
		ORDER BY Id 
		LIMIT 3
	`)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		t.Fatalf("Failed to get columns: %v", err)
	}

	// Process each row
	rowCount := 0
	for rows.Next() {
		rowCount++
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			t.Fatalf("Failed to scan row %d: %v", rowCount, err)
		}

		// Create record map
		record := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				record[col] = val
			}
		}

		// Test PATCH request for this record
		t.Run(fmt.Sprintf("PushRecord_%d", rowCount), func(t *testing.T) {
			customerID := record["CustomerID"]
			if customerID == nil {
				t.Fatal("CustomerID is nil")
			}

			// Create request data
			requestData := map[string]interface{}{
				"first_name":   record["FirstName"],
				"last_name":    record["LastName"],
				"email":        record["Email"],
				"phone_number": record["PhoneNumber"],
			}

			// Make PATCH request
			requestJSON, err := json.Marshal(requestData)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			url := fmt.Sprintf("%s/api/2/customers/%v/", baseURL, customerID)
			req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(requestJSON))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Verify response
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check that the customer ID matches
			if id, ok := response["id"]; !ok {
				t.Error("Response missing 'id' field")
			} else {
				// Convert both to int64 for comparison
				var responseID int64
				switch v := id.(type) {
				case float64:
					responseID = int64(v)
				case int64:
					responseID = v
				case int:
					responseID = int64(v)
				default:
					t.Errorf("Unexpected ID type: %T", id)
					return
				}

				var expectedID int64
				switch v := customerID.(type) {
				case int64:
					expectedID = v
				case int:
					expectedID = int64(v)
				case float64:
					expectedID = int64(v)
				default:
					t.Errorf("Unexpected customerID type: %T", customerID)
					return
				}

				if responseID != expectedID {
					t.Errorf("Expected ID %d, got %d", expectedID, responseID)
				}
			}

			// Check that the fields were updated
			for field, expectedValue := range requestData {
				if value, ok := response[field]; !ok {
					t.Errorf("Response missing field '%s'", field)
				} else if value != expectedValue {
					t.Errorf("Field '%s': expected %v, got %v", field, expectedValue, value)
				}
			}
		})
	}

	if rowCount == 0 {
		t.Error("No test data found in database")
	}
}

// testInvalidRequests tests error handling
func testInvalidRequests(t *testing.T, baseURL string) {
	tests := []struct {
		name           string
		customerID     string
		requestData    string
		method         string
		expectedStatus int
	}{
		{
			name:           "Invalid customer ID",
			customerID:     "invalid",
			requestData:    `{"first_name": "Test"}`,
			method:         "PATCH",
			expectedStatus: http.StatusOK, // Mock server doesn't validate customer ID format
		},
		{
			name:           "Invalid JSON",
			customerID:     "135299181",
			requestData:    `{"first_name": "Test"`,
			method:         "PATCH",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Wrong method",
			customerID:     "135299181",
			requestData:    `{"first_name": "Test"}`,
			method:         "PUT",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s/api/2/customers/%s/", baseURL, tt.customerID)
			req, err := http.NewRequest(tt.method, url, strings.NewReader(tt.requestData))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// createTestTables creates the necessary tables for testing
func createTestTables(db *sql.DB) error {
	// Create Accounts table
	accountsSQL := `
	CREATE TABLE IF NOT EXISTS Accounts (
		Id INTEGER PRIMARY KEY,
		FirstName TEXT,
		LastName TEXT,
		FullName TEXT,
		PhoneNumber TEXT,
		Email TEXT,
		AccountOwner TEXT,
		CustomerId TEXT,
		Notes TEXT,
		OriginalAddress TEXT,
		CrmId TEXT,
		DaysSinceLastCheckin INTEGER,
		FollowUpDate TEXT,
		LastCheckinDate TEXT,
		LastModifiedDate TEXT,
		CreatedAt DATETIME DEFAULT CURRENT_TIMESTAMP,
		UpdatedAt DATETIME DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(accountsSQL); err != nil {
		return fmt.Errorf("failed to create Accounts table: %v", err)
	}

	return nil
}

// insertTestData inserts test data into the database
func insertTestData(db *sql.DB) error {
	// Insert test accounts
	accounts := []struct {
		ID              int
		FirstName       string
		LastName        string
		Email           string
		PhoneNumber     string
		OriginalAddress string
	}{
		{135299181, "John", "Smith", "john.smith@example.com", "+1-555-123-4567", "123 Main Street, New York, NY 10001"},
		{135302580, "Jane", "Doe", "jane.doe@example.com", "+1-555-987-6543", "456 Oak Avenue, Los Angeles, CA 90210"},
		{135299577, "Bob", "Johnson", "bob.johnson@example.com", "+1-555-456-7890", "789 Pine Street, Chicago, IL 60601"},
	}

	for _, account := range accounts {
		_, err := db.Exec(`
			INSERT INTO Accounts (Id, FirstName, LastName, Email, PhoneNumber, OriginalAddress)
			VALUES (?, ?, ?, ?, ?, ?)
		`, account.ID, account.FirstName, account.LastName, account.Email, account.PhoneNumber, account.OriginalAddress)
		if err != nil {
			return fmt.Errorf("failed to insert account %d: %v", account.ID, err)
		}
	}

	return nil
}

// TestMockServerIntegration tests the complete integration
func TestMockServerIntegration(t *testing.T) {
	// Start mock server
	server := NewMockServer(0)
	ts := httptest.NewServer(http.HandlerFunc(server.handleAPIRequest))
	defer ts.Close()

	t.Run("TestServerStartup", func(t *testing.T) {
		// Test that server responds to basic requests
		resp, err := http.Get(ts.URL + "/api/2/profile/")
		if err != nil {
			t.Fatalf("Failed to get profile: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("TestCORSHeaders", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", ts.URL+"/api/2/customers/135299181/", nil)
		if err != nil {
			t.Fatalf("Failed to create OPTIONS request: %v", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make OPTIONS request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Check CORS headers
		corsHeaders := []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
		}

		for _, header := range corsHeaders {
			if resp.Header.Get(header) == "" {
				t.Errorf("Missing CORS header: %s", header)
			}
		}
	})
}
