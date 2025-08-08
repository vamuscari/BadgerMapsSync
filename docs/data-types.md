# Data Types

This document describes the data types used in the BadgerMaps CLI and how they map to the BadgerMaps API.

## Supported Data Types

The BadgerMaps CLI supports the following primary data types:

- `account`: Customer account information
- `checkin`: Sales visit tracking
- `route`: Route planning data
- `profile`: User profile data

## Data Models

### Account

Represents a customer account in BadgerMaps.

**API Model Fields:**
- `id`: Unique identifier
- `first_name`: Customer's first name
- `last_name`: Customer's last name
- `full_name`: Customer's full name
- `phone_number`: Contact phone number
- `email`: Contact email address
- `customer_id`: External customer ID
- `notes`: Additional notes
- `original_address`: Original address string
- `crm_id`: CRM system identifier
- `account_owner`: Owner of the account
- `days_since_last_checkin`: Days since last check-in
- `last_checkin_date`: Date of last check-in
- `last_modified_date`: Date of last modification
- `follow_up_date`: Scheduled follow-up date
- `locations`: Array of associated locations
- `custom_numeric1-30`: Custom numeric fields
- `custom_text1-30`: Custom text fields

**Database Tables:**
- `Accounts`: Stores basic account information
- `AccountLocations`: Stores locations associated with accounts

### Location

Represents a physical location associated with an account.

**API Model Fields:**
- `id`: Unique identifier
- `city`: City name
- `name`: Location name
- `zipcode`: Postal code
- `long`: Longitude coordinate
- `state`: State or province
- `lat`: Latitude coordinate
- `address_line_1`: Street address
- `location`: Formatted location string
- `is_approximate`: Whether the coordinates are approximate

**Database Table:**
- `AccountLocations`: Stores locations with references to accounts

### Route

Represents a planned route with waypoints.

**API Model Fields:**
- `id`: Unique identifier
- `name`: Route name
- `route_date`: Date of the route
- `duration`: Estimated duration in minutes
- `waypoints`: Array of waypoints
- `start_address`: Starting address
- `destination_address`: Final destination address
- `start_time`: Starting time

**Database Tables:**
- `Routes`: Stores basic route information
- `RouteWaypoints`: Stores waypoints associated with routes

### Waypoint

Represents a stop on a route.

**API Model Fields:**
- `id`: Unique identifier
- `name`: Waypoint name
- `address`: Street address
- `suite`: Suite or unit number
- `city`: City name
- `state`: State or province
- `zipcode`: Postal code
- `location`: Formatted location string
- `lat`: Latitude coordinate
- `long`: Longitude coordinate
- `layover_minutes`: Planned duration at this stop
- `position`: Order in the route
- `complete_address`: Full address
- `location_id`: Associated location ID
- `customer_id`: Associated customer ID
- `appt_time`: Appointment time
- `type`: Waypoint type
- `place_id`: Google Places ID

**Database Table:**
- `RouteWaypoints`: Stores waypoints with references to routes

### Checkin

Represents a sales visit or check-in at a customer location.

**API Model Fields:**
- `id`: Unique identifier
- `crm_id`: CRM system identifier
- `customer`: Associated customer ID
- `log_datetime`: Date and time of the check-in
- `type`: Type of check-in
- `comments`: Notes from the visit
- `extra_fields`: Additional custom fields
- `created_by`: User who created the check-in

**Database Table:**
- `AccountCheckins`: Stores check-in data with references to accounts

### UserProfile

Represents a BadgerMaps user profile.

**API Model Fields:**
- `id`: Unique identifier
- `email`: User's email address
- `first_name`: User's first name
- `last_name`: User's last name
- `is_manager`: Whether the user is a manager
- `marker_icon`: Icon used for the user on maps
- `manager`: Manager's email address
- `crm_editable_fields_list`: Fields editable in CRM
- `crm_base_url`: Base URL for CRM integration
- `crm_type`: Type of CRM integration
- `map_start_zoom`: Default map zoom level
- `map_start`: Default map starting location
- `company`: Associated company information

**Database Table:**
- `UserProfiles`: Stores user profile information

## Data Sets

Data sets are collections of custom field values.

**API Model Fields:**
- `id`: Unique identifier
- `name`: Data set name
- `filterable`: Whether the data set can be filtered
- `label`: Display label
- `values`: Array of possible values
- `position`: Display position
- `type`: Data type (text, numeric, etc.)

**Database Tables:**
- `DataSets`: Stores data set definitions
- `DataSetValues`: Stores values for data sets

## Database Schema

The BadgerMaps CLI supports multiple database types:

- SQLite3 (default)
- PostgreSQL
- Microsoft SQL Server

Each database type has its own implementation of the schema in the corresponding subdirectory:

- `database/sqlite3/`
- `database/postgres/`
- `database/mssql/`

## JSON to Database Mapping

The CLI converts JSON responses from the API to database records using a structured approach:

1. Parse the JSON response
2. Validate against the expected schema
3. Extract relevant fields
4. Map fields to the appropriate database tables
5. Generate SQL statements
6. Execute the SQL statements
7. Verify the results

For example, an account with locations from the API is split into records in both the `Accounts` and `AccountLocations` tables, with appropriate foreign key relationships.