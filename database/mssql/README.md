# SQL Files for BadgerSync

This directory contains all SQL Server (MSSQL) scripts used by the BadgerSync application.

## Table Creation Scripts

### Core Tables
- `create_accounts_table.sql` - Main accounts/customers table with all custom fields
- `create_account_locations_table.sql` - Account location information
- `create_account_checkins_table.sql` - Account check-in history
- `create_user_profiles_table.sql` - User profile information
- `create_data_sets_table.sql` - Data field definitions
- `create_data_set_values.sql` - Data field values
- `create_routes_table.sql` - Route information
- `create_route_waypoints_table.sql` - Route waypoints/stops

### Performance
- `create_indexes.sql` - All database indexes for performance optimization

## Data Operations

### MERGE Statements (Upsert Operations)
- `merge_accounts_basic.sql` - Basic account information upsert
- `merge_accounts_detailed.sql` - Complete account information with all custom fields
- `merge_account_checkins.sql` - Check-in data upsert
- `merge_routes.sql` - Route data upsert
- `merge_user_profiles.sql` - User profile data upsert

### INSERT Statements
- `insert_account_locations.sql` - Insert account location records
- `insert_route_waypoints.sql` - Insert route waypoint records
- `insert_data_sets.sql` - Insert data field definitions
- `insert_data_set_values.sql` - Insert data field values

### DELETE Statements
- `delete_account_locations.sql` - Delete account locations by account ID
- `delete_route_waypoints.sql` - Delete route waypoints by route ID
- `delete_data_sets.sql` - Delete data sets by profile ID
- `delete_data_set_values.sql` - Delete data set values by profile ID

## Utility Scripts

### Existence Checks
- `check_table_exists.sql` - Check if a table exists
- `check_index_exists.sql` - Check if an index exists

## Usage

These SQL files are used internally by the BadgerSync application. They are loaded and executed programmatically based on the application's needs.

### Table Creation Order
1. `create_accounts_table.sql`
2. `create_account_locations_table.sql`
3. `create_account_checkins_table.sql`
4. `create_user_profiles_table.sql`
5. `create_data_sets_table.sql`
6. `create_data_set_values.sql`
7. `create_routes_table.sql`
8. `create_route_waypoints_table.sql`
9. `create_indexes.sql`

### Data Sync Operations
- Accounts are synced using `merge_accounts_basic.sql` or `merge_accounts_detailed.sql`
- Related data (locations, check-ins) are synced after the main account record
- Routes and waypoints are synced together
- User profiles and their data fields are synced together

## Notes

- All tables use PascalCase naming convention
- Foreign key relationships are maintained between tables
- Timestamps (`CreatedAt`, `UpdatedAt`) are automatically managed
- Custom fields (CustomText1-30, CustomNumeric1-30) are available for extensibility
- Indexes are created for optimal query performance 