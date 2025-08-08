# API Integration

This document describes how BadgerMaps CLI integrates with the BadgerMaps API.

## API Endpoints

All API endpoint URLs are created using the `api_endpoints.go` file, which provides methods for each endpoint. This centralizes all API URL construction in one place for easier maintenance.

The base API URL is: `https://badgerapis.badgermapping.com/api/2`

## Available Endpoints

The BadgerMaps CLI interacts with the following API endpoints:

- **Customers/Accounts**: Manage customer accounts
  - `GET /customers/`: List all customer accounts
  - `GET /customers/{id}/`: Get a specific customer account
  - `POST /customers/`: Create a new customer account
  - `PATCH /customers/{id}/`: Update a customer account
  - `DELETE /customers/{id}/`: Delete a customer account

- **Routes**: Manage routes
  - `GET /routes/`: List all routes
  - `GET /routes/{id}/`: Get a specific route

- **Appointments/Check-ins**: Manage check-ins
  - `GET /appointments/`: List all check-ins
  - `GET /appointments/{id}`: Get a specific check-in
  - `GET /appointments/?customer_id={id}`: Get check-ins for a specific customer
  - `POST /appointments/`: Create a new check-in

- **Profiles**: Manage user profiles
  - `GET /profiles/`: Get the current user's profile
  - `GET /profiles/{id}`: Get a specific user profile

- **Locations**: Manage locations
  - `PATCH /locations/{id}`: Update a location


## Authentication

All API requests require authentication using an API token. The token is included in the `Authorization` header of each request:

```
Authorization: Token YOUR_API_TOKEN
```

You can authenticate with the API using the `auth` command:

```bash
badgermaps auth
```

## Rate Limit Handling

The CLI implements exponential backoff with jitter for API requests to handle rate limiting:

- Starts with a base delay (e.g., 1 second)
- After each failure, multiplies delay by a factor (e.g., 2)
- Adds random jitter (±20%) to prevent thundering herd problem
- Caps at a maximum delay (e.g., 60 seconds)

The CLI also respects the `Retry-After` header when provided by the API.

## JSON to Database Conversion

JSON responses from the API are converted to database tables using a structured approach:

1. Parse JSON response
2. Validate against expected schema
3. Extract fields
4. Map to database schema
5. Generate SQL
6. Execute SQL
7. Verify results

## Error Handling

The CLI provides clear, actionable error messages for API-related issues, including:

- Connection errors
- Authentication failures
- Rate limiting
- Invalid responses
- Server errors

Each error includes an error code and suggestions for resolution when possible.