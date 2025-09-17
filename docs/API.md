# API Module

This document provides a detailed overview of the API module in the BadgerMaps CLI.

- [README](../README.md)

## Overview

The `api` package is responsible for all interactions with the BadgerMaps API. It provides a client for making HTTP requests to the API and defines the data structures for the API resources.

## `api.APIClient`

The `api.APIClient` is the primary component of the API module. It handles the details of making authenticated requests to the BadgerMaps API, such as setting the authorization header and handling different HTTP methods.

The `APIClient` is configured with the API key and base URL, which are loaded from the application's configuration.

## API Endpoints

The API endpoints are defined in `api/api_endpoints.go`. The `Endpoints` struct provides methods for building the URLs for the various API endpoints. This centralizes all URL construction, making it easier to manage and update the API endpoints.

The supported endpoints are documented in the main [README](../README.md) file.

## Data Structures

The `api` package defines Go structs that correspond to the JSON data structures used by the BadgerMaps API. These structs are used for both serializing data to be sent to the API and deserializing the API's responses.

The main data structures include:

- `Account`
- `Location`
- `Route`
- `Waypoint`
- `Checkin`
- `UserProfile`
- `Company`
- `DataField`
- `FieldValue`

These structs make use of the `guregu/null` library to handle nullable JSON fields gracefully.

## Mappings

The `api/mappings.go` file contains a map (`DataSetAccountFieldMappings`) that maps the short names of custom data fields from the API to the corresponding fields in the `Account` struct. This is used when processing the user profile to associate custom fields with the correct account fields.

## Adding a New API Endpoint

To add a new API endpoint, you need to:

1.  Add a new method to the `Endpoints` struct in `api/api_endpoints.go` to construct the URL for the new endpoint.
2.  Add a new method to the `APIClient` in `api/api_service.go` to make requests to the new endpoint.
3.  If the new endpoint returns a new type of data, define a corresponding Go struct in `api/api_service.go`.
