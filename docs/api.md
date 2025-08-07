# API Documentation

## Overview

The BadgerMaps CLI integrates with the BadgerMaps API to sync data between the API and local databases. This document covers the API integration, authentication, endpoints, and usage patterns.

## Authentication

The BadgerMaps API uses API key authentication. Configure your API key in the `.env` file:

```bash
BADGERMAPS_API_KEY=your_api_key_here
```

## API Base URL

The default API base URL is `https://api.badgermapping.com/v1`. You can override this by setting:

```bash
BADGERMAPS_API_URL=https://custom-api.example.com/v2
```

## Supported Endpoints

### Accounts

**Endpoint:** `/customers/`
**Method:** GET
**Description:** Retrieve all accounts/customers from the API

**Response Format:**
```json
[
  {
    "id": 12345,
    "first_name": "John",
    "last_name": "Doe",
    "updated_at": "2025-08-06T10:00:00Z"
  }
]
```

**Usage:**
```bash
./badgersync pull accounts
```

### Account Details

**Endpoint:** `/customers/{id}`
**Method:** GET
**Description:** Retrieve detailed information for a specific account

**Response Format:**
```json
{
  "id": 12345,
  "first_name": "John",
  "last_name": "Doe",
  "email": "john.doe@example.com",
  "phone": "+1-555-123-4567",
  "address": {
    "street": "123 Main St",
    "city": "Anytown",
    "state": "CA",
    "zip": "12345"
  },
  "updated_at": "2025-08-06T10:00:00Z"
}
```

### Routes

**Endpoint:** `/routes/`
**Method:** GET
**Description:** Retrieve all routes from the API

**Response Format:**
```json
[
  {
    "id": 67890,
    "name": "Morning Route",
    "account_id": 12345,
    "created_at": "2025-08-06T10:00:00Z",
    "updated_at": "2025-08-06T10:00:00Z"
  }
]
```

**Usage:**
```bash
./badgersync pull routes
```

### Check-ins

**Endpoint:** `/checkins/`
**Method:** GET
**Description:** Retrieve all check-ins from the API

**Response Format:**
```json
[
  {
    "id": 11111,
    "account_id": 12345,
    "route_id": 67890,
    "checkin_time": "2025-08-06T10:00:00Z",
    "location": {
      "latitude": 37.7749,
      "longitude": -122.4194
    },
    "notes": "Customer visit completed"
  }
]
```

**Usage:**
```bash
./badgersync pull checkins
```

### User Profiles

**Endpoint:** `/profiles/`
**Method:** GET
**Description:** Retrieve all user profiles from the API

**Response Format:**
```json
[
  {
    "id": 22222,
    "user_id": 33333,
    "first_name": "Jane",
    "last_name": "Smith",
    "email": "jane.smith@example.com",
    "role": "driver",
    "updated_at": "2025-08-06T10:00:00Z"
  }
]
```

**Usage:**
```bash
./badgersync pull profiles
```

## Error Handling

The API client handles various error scenarios:

### HTTP Status Codes

- **200 OK:** Successful request
- **401 Unauthorized:** Invalid or missing API key
- **403 Forbidden:** Insufficient permissions
- **404 Not Found:** Resource not found
- **429 Too Many Requests:** Rate limit exceeded
- **500 Internal Server Error:** Server error

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_API_KEY",
    "message": "The provided API key is invalid",
    "details": "Please check your API key configuration"
  }
}
```

## Rate Limiting

The BadgerMaps API implements rate limiting. The CLI automatically handles rate limiting by:

- Respecting `Retry-After` headers
- Implementing exponential backoff
- Logging rate limit warnings

## Testing API Connectivity

Test your API configuration:

```bash
./badgersync test
```

This command will:
1. Verify API key validity
2. Test connectivity to all endpoints
3. Display API response times
4. Show any configuration issues

## API Client Implementation

The API client is implemented in `api/client.go` and provides:

- Automatic authentication
- Request/response logging
- Error handling and retries
- Rate limiting support
- JSON serialization/deserialization

### Key Methods

```go
// GetAccounts retrieves all accounts from the API
func (c *Client) GetAccounts() ([]Account, error)

// GetAccount retrieves a specific account by ID
func (c *Client) GetAccount(id int) (*Account, error)

// GetRoutes retrieves all routes from the API
func (c *Client) GetRoutes() ([]Route, error)

// GetCheckins retrieves all check-ins from the API
func (c *Client) GetCheckins() ([]Checkin, error)

// GetProfiles retrieves all user profiles from the API
func (c *Client) GetProfiles() ([]Profile, error)
```

## Configuration Examples

### Basic Configuration

```bash
# .env file
BADGERMAPS_API_KEY=your_api_key_here
BADGERMAPS_API_URL=https://api.badgermapping.com/v1
```

### Custom API Endpoint

```bash
# .env file
BADGERMAPS_API_KEY=your_api_key_here
BADGERMAPS_API_URL=https://custom-api.example.com/v2
```

### With Logging

```bash
# .env file
BADGERMAPS_API_KEY=your_api_key_here
LOG_LEVEL=debug
LOG_FILE=./api_debug.log
```

## Troubleshooting

### Common Issues

1. **Invalid API Key**
   - Verify your API key is correct
   - Check for extra spaces or characters
   - Ensure the key has proper permissions

2. **Connection Timeouts**
   - Check your internet connection
   - Verify the API URL is correct
   - Check firewall settings

3. **Rate Limiting**
   - Reduce request frequency
   - Implement proper delays between requests
   - Check API usage limits

4. **Authentication Errors**
   - Verify API key format
   - Check API key permissions
   - Ensure proper environment variable loading

### Debug Mode

Enable debug logging to troubleshoot API issues:

```bash
# .env file
LOG_LEVEL=debug
```

This will log all API requests and responses for debugging purposes. 