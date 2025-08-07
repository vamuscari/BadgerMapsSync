# Push Functionality Test Suite

## Overview

This test suite validates the PATCH endpoint functionality for customer updates in the BadgerMaps CLI mock server. The tests ensure that the push functionality works correctly for both direct API calls and database-to-API synchronization.

## Test Structure

### 1. TestPushFunctionality
Main test suite that covers the complete push workflow:

#### TestCustomerUpdate
Tests direct PATCH requests to the customer update endpoint:
- **Update customer name and email**: Verifies basic field updates
- **Update phone number**: Tests phone number field updates
- **Update address**: Tests address field updates

**Validates:**
- ✅ HTTP status codes
- ✅ Response structure
- ✅ Field updates
- ✅ Customer ID matching
- ✅ Timestamp updates
- ✅ Location ID uniqueness

#### TestPushFromDatabase
Tests pushing data from SQLite database to the API:
- **PushRecord_1**: Tests first database record
- **PushRecord_2**: Tests second database record  
- **PushRecord_3**: Tests third database record

**Validates:**
- ✅ Database query execution
- ✅ Data type conversions
- ✅ PATCH request formation
- ✅ Response validation
- ✅ Field mapping

#### TestInvalidRequests
Tests error handling and edge cases:
- **Invalid customer ID**: Tests with non-numeric customer ID
- **Invalid JSON**: Tests with malformed JSON payload
- **Wrong method**: Tests with incorrect HTTP method

**Validates:**
- ✅ Error status codes
- ✅ Request validation
- ✅ Method validation

### 2. TestMockServerIntegration
Integration tests for the mock server:

#### TestServerStartup
- **Basic connectivity**: Verifies server responds to requests
- **Profile endpoint**: Tests GET /api/2/profile/

#### TestCORSHeaders
- **CORS support**: Validates CORS headers for cross-origin requests
- **Preflight requests**: Tests OPTIONS method handling

## Test Coverage

**Current Coverage: 21.7%**

### Covered Functions:
- `NewMockServer`: 100.0%
- `handleAPIRequest`: 61.9%
- `handleProfile`: 60.0%
- `handleCustomerUpdate`: 73.7%
- `handleNotFound`: 71.4%
- `handleError`: 71.4%

### Key Test Scenarios:

#### ✅ Customer Data Updates
```bash
PATCH /api/2/customers/135299181/
{
  "first_name": "Updated",
  "last_name": "Store Name", 
  "email": "updated@example.com"
}
```

#### ✅ Database Integration
- SQLite database setup
- Test data insertion
- Query execution
- Data type handling

#### ✅ Error Handling
- Invalid customer IDs
- Malformed JSON
- Wrong HTTP methods
- Missing required fields

#### ✅ Response Validation
- Customer ID matching
- Field updates
- Timestamp updates
- Location ID uniqueness

## Running Tests

### Basic Test Execution
```bash
cd testing
go test -v
```

### With Coverage
```bash
./run_tests.sh
```

### Individual Test Suites
```bash
# Run only push functionality tests
go test -v -run TestPushFunctionality

# Run only integration tests
go test -v -run TestMockServerIntegration

# Run specific test case
go test -v -run "TestPushFunctionality/TestCustomerUpdate/Update_customer_name_and_email"
```

## Test Data

### Database Schema
```sql
CREATE TABLE Accounts (
    Id INTEGER PRIMARY KEY,
    FirstName TEXT,
    LastName TEXT,
    Email TEXT,
    PhoneNumber TEXT,
    OriginalAddress TEXT,
    -- ... other fields
);
```

### Test Records
```go
accounts := []struct {
    ID             int
    FirstName      string
    LastName       string
    Email          string
    PhoneNumber    string
    OriginalAddress string
}{
    {135299181, "John", "Smith", "john.smith@example.com", "+1-555-123-4567", "123 Main Street, New York, NY 10001"},
    {135302580, "Jane", "Doe", "jane.doe@example.com", "+1-555-987-6543", "456 Oak Avenue, Los Angeles, CA 90210"},
    {135299577, "Bob", "Johnson", "bob.johnson@example.com", "+1-555-456-7890", "789 Pine Street, Chicago, IL 60601"},
}
```

## Mock Server Features Tested

### PATCH Endpoint: `/api/2/customers/{id}/`
- **Methods**: PATCH, POST (for compatibility)
- **Request**: JSON with customer fields to update
- **Response**: Complete updated customer object

### Features Validated:
1. **Dynamic ID Handling**: Updates customer ID to match request
2. **Field Merging**: Combines request data with existing customer data
3. **Timestamp Updates**: Sets `last_modified_date` to current date
4. **Location ID Uniqueness**: Generates unique location IDs per customer
5. **CORS Support**: Handles cross-origin requests
6. **Error Handling**: Validates request format and method

### Response Structure:
```json
{
  "id": 135299181,
  "first_name": "Updated",
  "last_name": "Store Name",
  "email": "updated@example.com",
  "phone_number": "+1-555-123-4567",
  "last_modified_date": "2025-08-05",
  "locations": [
    {
      "id": 135299181001,
      "name": "Main Office",
      "address_line_1": "123 Main Street"
    }
  ]
}
```

## Test Results Summary

### ✅ All Tests Passing
- **Customer Update Tests**: 3/3 PASS
- **Database Push Tests**: 3/3 PASS
- **Invalid Request Tests**: 3/3 PASS
- **Server Integration Tests**: 2/2 PASS

### 🎯 Functionality Verified
- ✅ Customer data updates via PATCH
- ✅ Database-to-API synchronization
- ✅ Field mapping and validation
- ✅ Error handling and edge cases
- ✅ CORS and HTTP method support
- ✅ Response structure validation
- ✅ ID uniqueness and type handling

## Future Test Enhancements

### Potential Additions:
1. **Performance Tests**: Load testing with multiple concurrent requests
2. **Security Tests**: Authentication and authorization validation
3. **Data Validation Tests**: Field format and constraint validation
4. **Integration Tests**: Full badgersync push command testing
5. **Edge Case Tests**: Large payloads, special characters, etc.

### Coverage Improvements:
- Add tests for other endpoints (GET, POST)
- Test database connection error handling
- Validate SQL query optimization
- Test concurrent database operations 