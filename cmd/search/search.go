package search

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"badgermapscli/api"
	"badgermapscli/common"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SearchResult represents a search result
type SearchResult struct {
	ID   string
	Name string
	Type string
}

// NewSearchCmd creates a new search command
func NewSearchCmd() *cobra.Command {
	var (
		forceOnline bool
		searchType  string
	)

	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Find items by name",
		Long:  `Search for accounts, locations, and other items by name.`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			colors := common.Colors
			query := strings.Join(args, " ")

			// Get API key from environment
			apiKey := viper.GetString("API_KEY")
			if apiKey == "" && forceOnline {
				fmt.Println(colors.Red("API key not found. Please authenticate first with 'badgermaps auth'"))
				common.Errors.ExitWithAuthFailure("API key not found")
			}

			// Determine search type
			searchTypes := []string{"all"}
			if searchType != "" {
				searchTypes = strings.Split(searchType, ",")
			}

			// Search offline first unless force online
			if !forceOnline {
				results, err := searchOffline(query, searchTypes)
				if err != nil {
					fmt.Println(colors.Yellow("Offline search failed: %v", err))
					fmt.Println(colors.Yellow("Falling back to online search..."))
				} else if len(results) > 0 {
					displayResults(results)
					return
				} else {
					fmt.Println(colors.Yellow("No results found in offline cache."))
					fmt.Println(colors.Yellow("Falling back to online search..."))
				}
			}

			// If we got here, we need to search online
			if apiKey == "" {
				fmt.Println(colors.Red("API key not found. Please authenticate first with 'badgermaps auth'"))
				common.Errors.ExitWithAuthFailure("API key not found")
			}

			results, err := searchOnline(apiKey, query, searchTypes)
			if err != nil {
				fmt.Println(colors.Red("Online search failed: %v", err))
				common.Errors.ExitWithAPIError("Online search failed: %v", err)
			}

			if len(results) == 0 {
				fmt.Println(colors.Yellow("No results found."))
				return
			}

			displayResults(results)

			// Update cache in background
			go updateSearchCache(apiKey)
		},
	}

	// Add flags
	searchCmd.Flags().BoolVar(&forceOnline, "online", false, "Force online search (default is to search offline first)")
	searchCmd.Flags().StringVar(&searchType, "type", "", "Type of items to search (accounts,locations,profiles,all)")

	return searchCmd
}

// searchOffline searches the local SQLite database
func searchOffline(query string, types []string) ([]SearchResult, error) {
	var results []SearchResult

	// Open SQLite database
	dbPath := getCachePath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache database: %w", err)
	}
	defer db.Close()

	// Check if cache is too old
	var lastUpdated time.Time
	err = db.QueryRow("SELECT value FROM metadata WHERE key = 'last_updated'").Scan(&lastUpdated)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache update time: %w", err)
	}

	// If cache is older than 1 day, consider it stale
	if time.Since(lastUpdated) > 24*time.Hour {
		return nil, fmt.Errorf("cache is stale (last updated %s)", lastUpdated.Format(time.RFC3339))
	}

	// Build query
	searchQuery := "%" + query + "%"
	sqlQuery := "SELECT id, name, type FROM search_items WHERE name LIKE ? OR id LIKE ?"

	// Filter by type if specified
	if len(types) > 0 && !(len(types) == 1 && types[0] == "all") {
		sqlQuery += " AND type IN (" + strings.Repeat("?,", len(types)-1) + "?)"
	}

	// Prepare statement
	stmt, err := db.Prepare(sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare search query: %w", err)
	}
	defer stmt.Close()

	// Execute query
	var rows *sql.Rows
	if len(types) > 0 && !(len(types) == 1 && types[0] == "all") {
		args := make([]interface{}, len(types)+2)
		args[0] = searchQuery
		args[1] = searchQuery
		for i, t := range types {
			args[i+2] = t
		}
		rows, err = stmt.Query(args...)
	} else {
		rows, err = stmt.Query(searchQuery, searchQuery)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	// Process results
	for rows.Next() {
		var result SearchResult
		err := rows.Scan(&result.ID, &result.Name, &result.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// searchOnline searches using the BadgerMaps API
func searchOnline(apiKey, query string, types []string) ([]SearchResult, error) {
	var results []SearchResult

	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Search accounts
	if contains(types, "all") || contains(types, "accounts") {
		accounts, err := apiClient.SearchAccounts(query)
		if err != nil {
			return nil, fmt.Errorf("failed to search accounts: %w", err)
		}

		for _, account := range accounts {
			results = append(results, SearchResult{
				ID:   fmt.Sprintf("%d", account.ID),
				Name: account.FullName,
				Type: "account",
			})
		}
	}

	// Search locations
	if contains(types, "all") || contains(types, "locations") {
		locations, err := apiClient.SearchLocations(query)
		if err != nil {
			return nil, fmt.Errorf("failed to search locations: %w", err)
		}

		for _, location := range locations {
			// Handle nil pointer for location name
			locationName := ""
			if location.Name != nil {
				locationName = *location.Name
			}

			results = append(results, SearchResult{
				ID:   fmt.Sprintf("%d", location.ID),
				Name: locationName,
				Type: "location",
			})
		}
	}

	// Search profiles
	if contains(types, "all") || contains(types, "profiles") {
		profiles, err := apiClient.SearchProfiles(query)
		if err != nil {
			return nil, fmt.Errorf("failed to search profiles: %w", err)
		}

		for _, profile := range profiles {
			// Combine first and last name for display
			fullName := profile.FirstName + " " + profile.LastName

			results = append(results, SearchResult{
				ID:   fmt.Sprintf("%d", profile.ID),
				Name: fullName,
				Type: "profile",
			})
		}
	}

	return results, nil
}

// displayResults displays search results
func displayResults(results []SearchResult) {
	colors := common.Colors

	fmt.Printf("%s\n", colors.Blue("Search Results"))
	fmt.Printf("%s\n", colors.Blue("=============="))

	// Calculate column widths
	idWidth := 10
	nameWidth := 30
	typeWidth := 10

	for _, result := range results {
		if len(result.ID) > idWidth {
			idWidth = len(result.ID)
		}
		if len(result.Name) > nameWidth {
			nameWidth = len(result.Name)
		}
		if len(result.Type) > typeWidth {
			typeWidth = len(result.Type)
		}
	}

	// Print header
	fmt.Printf("%-*s | %-*s | %-*s\n", idWidth, "ID", nameWidth, "Name", typeWidth, "Type")
	fmt.Printf("%s+%s+%s\n",
		strings.Repeat("-", idWidth+2),
		strings.Repeat("-", nameWidth+2),
		strings.Repeat("-", typeWidth+2))

	// Print results
	for _, result := range results {
		fmt.Printf("%-*s | %-*s | %-*s\n",
			idWidth, result.ID,
			nameWidth, result.Name,
			typeWidth, result.Type)
	}
}

// updateSearchCache updates the local SQLite database with the latest data
func updateSearchCache(apiKey string) {
	// Create API client
	apiClient := api.NewAPIClient(apiKey)

	// Create cache directory if it doesn't exist
	cacheDir := filepath.Join(os.Getenv("HOME"), ".badgermaps", "cache")
	os.MkdirAll(cacheDir, 0755)

	// Open SQLite database
	dbPath := getCachePath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		fmt.Printf("Failed to open cache database: %v\n", err)
		return
	}
	defer db.Close()

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS search_items (
			id TEXT PRIMARY KEY,
			name TEXT,
			type TEXT,
			data TEXT
		);
		CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT
		);
	`)
	if err != nil {
		fmt.Printf("Failed to create cache tables: %v\n", err)
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		fmt.Printf("Failed to begin transaction: %v\n", err)
		return
	}

	// Clear existing data
	_, err = tx.Exec("DELETE FROM search_items")
	if err != nil {
		tx.Rollback()
		fmt.Printf("Failed to clear cache: %v\n", err)
		return
	}

	// Fetch and store accounts
	accounts, err := apiClient.GetAccounts()
	if err != nil {
		tx.Rollback()
		fmt.Printf("Failed to fetch accounts: %v\n", err)
		return
	}

	stmt, err := tx.Prepare("INSERT INTO search_items (id, name, type, data) VALUES (?, ?, ?, ?)")
	if err != nil {
		tx.Rollback()
		fmt.Printf("Failed to prepare statement: %v\n", err)
		return
	}

	for _, account := range accounts {
		_, err = stmt.Exec(account.ID, account.FullName, "account", "")
		if err != nil {
			tx.Rollback()
			fmt.Printf("Failed to insert account: %v\n", err)
			return
		}
	}

	// Update last updated timestamp
	_, err = tx.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)",
		"last_updated", time.Now().Format(time.RFC3339))
	if err != nil {
		tx.Rollback()
		fmt.Printf("Failed to update timestamp: %v\n", err)
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		fmt.Printf("Failed to commit transaction: %v\n", err)
		return
	}
}

// getCachePath returns the path to the SQLite cache database
func getCachePath() string {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".badgermaps", "cache")
	return filepath.Join(cacheDir, "search_cache.db")
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
