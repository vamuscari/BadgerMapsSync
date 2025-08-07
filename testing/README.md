# Testing Folder - Mock API Responses

This folder contains a complete mock HTTP server and JSON files that mimic the BadgerMaps API responses for testing purposes. These files follow the exact structure defined in the Go structs in `badger/badger.go`.

## File Structure

### Server Files
- `mock_server.go` - Complete HTTP server that serves the JSON responses
- `run_mock_server.sh` - Shell script to build and run the server (Linux/Mac)
- `run_mock_server.bat` - Batch script to build and run the server (Windows)
- `test_api.py` - Python test script to verify all endpoints
- `config_mock.json` - Configuration file for mock server settings

### JSON Response Files (`json/` folder)

#### Profile Endpoints
- `json/profile_response.json` - Mock response for `/api/2/profile/` endpoint
  - Contains user profile information, company details, data fields, and permissions

#### Account Endpoints
- `json/accounts_list_response.json` - Mock response for `/api/2/customers/` (list)
  - Array of basic account information (id, first_name, last_name, original_address)
- `json/account_detail_response.json` - Mock response for `/api/2/customers/{id}/` (detail)
  - Complete account information including all custom fields, locations, and metadata
- `json/account_update_request_example.json` - Example request body for PATCH `/api/2/customers/{id}/`
  - Shows how to update account information

#### Check-in Endpoints
- `json/account_checkins_response.json` - Mock response for `/api/2/appointments/?customer_id={id}`
  - Array of check-in records for a specific customer
- `json/checkin_request_example.json` - Example request body for POST `/api/2/appointments/`
  - Shows how to create a new check-in
- `json/checkin_response_example.json` - Example response for newly created check-in
  - Shows the response structure after creating a check-in

#### Route Endpoints
- `json/routes_list_response.json` - Mock response for `/api/2/routes/` (list)
  - Array of basic route information (id, name, route_date)
- `json/route_detail_response.json` - Mock response for `/api/2/routes/{id}/` (detail)
  - Complete route information including waypoints and scheduling details

#### Search Endpoints
- `json/search_users_response.json` - Mock response for `/api/2/search/users/?q=query`
  - Array of user search results

#### Error Handling
- `json/error_response_example.json` - Example error response structure

## Usage

The testing folder provides two main ways to use the mock data:

### 1. Direct JSON Files
The JSON files in the `json/` folder can be used for:

- **Unit Testing** - Mock API responses for testing Go code without making actual API calls
- **Development** - Reference the structure when implementing new features
- **Documentation** - Examples of expected API responses

### 2. HTTP Mock Server
The mock server provides a complete HTTP API that can be used for:

- **Integration Testing** - Make actual HTTP requests to test endpoints
- **Frontend Development** - Test web applications against the mock API
- **API Development** - Verify request/response handling
- **Automated Testing** - Use in CI/CD pipelines

## Data Consistency

The mock data is designed to be consistent across files:
- Customer ID 1001 (John Smith) appears in multiple files
- Route ID 4001 (Downtown Healthcare Route) has waypoints that reference customer IDs
- Check-in records reference existing customer IDs
- All dates and times are realistic and consistent

## API Endpoints Covered

- `GET /api/2/profile/` - User profile
- `GET /api/2/customers/` - Account list
- `GET /api/2/customers/{id}/` - Account detail
- `PATCH /api/2/customers/{id}/` - Update account
- `GET /api/2/appointments/?customer_id={id}` - Account check-ins
- `POST /api/2/appointments/` - Create check-in
- `GET /api/2/routes/` - Route list
- `GET /api/2/routes/{id}/` - Route detail
- `GET /api/2/search/users/?q=query` - Search users

## Mock HTTP Server

The testing folder includes a complete mock HTTP server that serves the JSON files as API responses. This allows you to make actual HTTP requests to test endpoints.

### Running the Mock Server

**Linux/Mac:**
```bash
cd testing
./run_mock_server.sh
```

**Windows:**
```cmd
cd testing
run_mock_server.bat
```

**Manual:**
```bash
cd testing
go build -o mock_server mock_server.go
./mock_server 8080
```

### Testing the Mock Server

Use the provided Python test script to verify all endpoints:

```bash
cd testing
python3 test_api.py
```

Or test individual endpoints with curl:

```bash
# Get profile
curl http://localhost:8080/api/2/profile/

# Get customers list
curl http://localhost:8080/api/2/customers/

# Get specific customer
curl http://localhost:8080/api/2/customers/1001/

# Get customer check-ins
curl "http://localhost:8080/api/2/appointments/?customer_id=1001"

# Create a check-in
curl -X POST http://localhost:8080/api/2/appointments/ \
  -H "Content-Type: application/json" \
  -d '{"customer": 1001, "type": "visit", "comments": "Test"}'

# Get routes
curl http://localhost:8080/api/2/routes/

# Search users
curl "http://localhost:8080/api/2/search/users/?q=john"
```

### Server Features

- **CORS Support** - Allows cross-origin requests for testing
- **Dynamic ID Handling** - Updates response IDs to match request IDs
- **Parameter Validation** - Validates required parameters
- **Error Handling** - Returns proper HTTP error responses
- **Request Logging** - Logs all incoming requests
- **Flexible Port** - Can run on any port (default: 8080)

## Notes

- All JSON files follow the exact structure defined in the Go structs
- Null values are represented as `null` (not omitted)
- Dates are in ISO 8601 format
- IDs are consistent across related files for realistic testing scenarios
- The mock server serves the same data structure as the real API 
- Mock Beverage Distribution