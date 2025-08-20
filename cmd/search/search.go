package search

import (
	"fmt"
	"strings"
	"unicode"

	"badgermapscli/api"
	"badgermapscli/app"
	"badgermapscli/database"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/spf13/cobra"
)

// SearchCmd creates a new search command
func SearchCmd(config *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for accounts and routes",
		Long: `Search for accounts and routes in the local database.
The search uses fuzzy matching to find items that match the query.
Results include the item type and three additional fields to help with filtering.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			query := strings.Join(args, " ")
			online, _ := cmd.Flags().GetBool("online")

			if config.Verbose {
				fmt.Printf("Searching for: %s\n", query)
				if online {
					fmt.Println("Using online mode")
				} else {
					fmt.Println("Using offline mode (local cache)")
				}
			}

			// Perform the search
			results, err := performSearch(query, online, config)
			if err != nil {
				fmt.Println(app.Colors.Red("Error: Search failed: %v", err))
				return
			}

			// Display results
			displayResults(results)
		},
	}

	// Add flags
	cmd.Flags().BoolP("online", "o", false, "Use online mode instead of local cache")

	return cmd
}

// initCacheDB initializes the cache database
func initCacheDB(config *app.Config) (*database.Client, error) {
	// Get database configuration
	dbConfig := &database.Config{
		DatabaseType: "sqlite3",
		Database:     config.CacheDBPath,
		Host:         config.CacheDBHost,
	}

	db, err := database.NewClient(dbConfig, config.Verbose)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// SearchResult represents a search result item
type SearchResult struct {
	ID          int
	Name        string
	Type        string
	Field1Name  string
	Field1Value string
	Field2Name  string
	Field2Value string
	Field3Name  string
	Field3Value string
}

// performSearch searches for accounts and routes that match the query
func performSearch(query string, online bool, config *app.Config) ([]SearchResult, error) {
	var results []SearchResult

	if online {
		// Use the API for online search
		return searchOnline(query, config)
	}

	// Use the local database for offline search
	// Get database client for the main database
	dbConfig := &database.Config{
		DatabaseType: config.DBType,
		Database:     config.DBPath,
		Host:         config.DBHost,
		Port:         config.DBPort,
		Username:     config.DBUser,
		Password:     config.DBPassword,
	}

	client, err := database.NewClient(dbConfig, config.Verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer client.Close()

	// Search accounts directly using the client
	accountResults, err := searchAccounts(client, query, config)
	if err != nil {
		return nil, fmt.Errorf("failed to search accounts: %w", err)
	}
	results = append(results, accountResults...)

	// Search routes directly using the client
	routeResults, err := searchRoutes(client, query, config)
	if err != nil {
		return nil, fmt.Errorf("failed to search routes: %w", err)
	}
	results = append(results, routeResults...)

	return results, nil
}

// searchOnline searches for accounts and routes using the API
func searchOnline(query string, config *app.Config) ([]SearchResult, error) {
	var results []SearchResult

	// Get API client
	apiKey := config.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("API key not found in configuration")
	}

	apiClient := api.NewAPIClient(apiKey)

	// Search accounts
	if config.Verbose {
		fmt.Println("Searching accounts online...")
	}

	// Use the SearchAccounts method to search for accounts
	accounts, err := apiClient.SearchAccounts(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search accounts online: %w", err)
	}

	// Convert account results to SearchResult
	for _, account := range accounts {
		// Create a SearchResult for each account
		result := SearchResult{
			ID:          account.ID,
			Name:        account.FullName,
			Type:        "Account",
			Field1Name:  "Email",
			Field1Value: account.Email,
			Field2Name:  "Phone",
			Field2Value: account.PhoneNumber,
			Field3Name:  "Owner",
			Field3Value: getStringValue(account.AccountOwner),
		}
		results = append(results, result)
	}

	// For routes, we would need to get all routes and filter them
	// since there's no direct search endpoint for routes
	if config.Verbose {
		fmt.Println("Fetching routes online...")
	}

	// Use the GetRoutes method to get all routes
	routes, err := apiClient.GetRoutes()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch routes online: %w", err)
	}

	// Filter routes by name
	for _, route := range routes {
		// Calculate fuzzy match score
		score := calculateFuzzyScore(query, route.Name)

		// Only include results with a minimum score
		if score > 0.3 {
			// Create a SearchResult for each matching route
			result := SearchResult{
				ID:          route.ID,
				Name:        route.Name,
				Type:        "Route",
				Field1Name:  "Date",
				Field1Value: route.RouteDate,
				Field2Name:  "Start",
				Field2Value: route.StartAddress,
				Field3Name:  "Destination",
				Field3Value: route.DestinationAddress,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// getStringValue safely gets the value of a string pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// searchAccounts searches for accounts that match the query
func searchAccounts(client *database.Client, query string, config *app.Config) ([]SearchResult, error) {
	var results []SearchResult

	// Get database connection
	db := client.GetDB()

	// Load the SQL query from the appropriate file based on database type
	sqlLoader := database.NewSQLLoader(config.DBType, config.Verbose)
	sqlQuery, err := sqlLoader.LoadSearchAccountsSQL()
	if err != nil {
		return nil, fmt.Errorf("failed to load search accounts SQL: %w", err)
	}

	// Prepare the query with wildcards for fuzzy matching
	searchPattern := "%" + query + "%"
	rows, err := db.Query(sqlQuery, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}
	defer rows.Close()

	// Process results
	for rows.Next() {
		var id int
		var fullName, firstName, lastName, email, phoneNumber string

		if err := rows.Scan(&id, &fullName, &firstName, &lastName, &email, &phoneNumber); err != nil {
			return nil, fmt.Errorf("failed to scan account row: %w", err)
		}

		// Calculate fuzzy match score
		score := calculateFuzzyScore(query, fullName)

		// Only include results with a minimum score
		if score > 0.3 {
			result := SearchResult{
				ID:          id,
				Name:        fullName,
				Type:        "Account",
				Field1Name:  "Email",
				Field1Value: email,
				Field2Name:  "Phone",
				Field2Value: phoneNumber,
				Field3Name:  "Name",
				Field3Value: fmt.Sprintf("%s %s", firstName, lastName),
			}
			results = append(results, result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating account rows: %w", err)
	}

	return results, nil
}

// searchRoutes searches for routes that watch the query
func searchRoutes(client *database.Client, query string, config *app.Config) ([]SearchResult, error) {
	var results []SearchResult

	// Get database connection
	db := client.GetDB()

	// Load the SQL query from the appropriate file based on database type
	sqlLoader := database.NewSQLLoader(config.DBType, config.Verbose)
	sqlQuery, err := sqlLoader.LoadSearchRoutesSQL()
	if err != nil {
		return nil, fmt.Errorf("failed to load search routes SQL: %w", err)
	}

	// Prepare the query with wildcards for fuzzy matching
	searchPattern := "%" + query + "%"
	rows, err := db.Query(sqlQuery, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query routes: %w", err)
	}
	defer rows.Close()

	// Process results
	for rows.Next() {
		var id int
		var name, routeDate, startAddress, destinationAddress string

		if err := rows.Scan(&id, &name, &routeDate, &startAddress, &destinationAddress); err != nil {
			return nil, fmt.Errorf("failed to scan route row: %w", err)
		}

		// Calculate fuzzy match score
		score := calculateFuzzyScore(query, name)

		// Only include results with a minimum score
		if score > 0.3 {
			result := SearchResult{
				ID:          id,
				Name:        name,
				Type:        "Route",
				Field1Name:  "Date",
				Field1Value: routeDate,
				Field2Name:  "Start",
				Field2Value: startAddress,
				Field3Name:  "Destination",
				Field3Value: destinationAddress,
			}
			results = append(results, result)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating route rows: %w", err)
	}

	return results, nil
}

// calculateFuzzyScore calculates a similarity score between the query and the text
// Returns a value between 0 and 1, where 1 is a perfect match
func calculateFuzzyScore(query, text string) float64 {
	// Convert to lowercase for case-insensitive matching
	query = strings.ToLower(query)
	text = strings.ToLower(text)

	// If the query is found as a substring, that's a strong match
	if strings.Contains(text, query) {
		return 0.9
	}

	// Split into words for word-level matching
	queryWords := splitIntoWords(query)
	textWords := splitIntoWords(text)

	// Count how many query words are found in the text
	matchedWords := 0
	for _, qWord := range queryWords {
		for _, tWord := range textWords {
			if strings.Contains(tWord, qWord) || levenshteinDistance(qWord, tWord) <= 2 {
				matchedWords++
				break
			}
		}
	}

	// Calculate score based on matched words
	if len(queryWords) == 0 {
		return 0
	}

	return float64(matchedWords) / float64(len(queryWords))
}

// splitIntoWords splits a string into words
func splitIntoWords(s string) []string {
	var words []string
	var currentWord strings.Builder

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			currentWord.WriteRune(r)
		} else {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}
	}

	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create a matrix to store the distances
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize the first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill in the rest of the matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// displayResults displays the search results in a formatted table
func displayResults(results []SearchResult) {
	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	// Find the maximum width for each column
	maxIDWidth := 5          // "ID" header width
	maxNameWidth := 4        // "Name" header width
	maxTypeWidth := 4        // "Type" header width
	maxField1NameWidth := 10 // Minimum width for field name
	maxField2NameWidth := 10
	maxField3NameWidth := 10
	maxField1ValueWidth := 10 // Minimum width for field value
	maxField2ValueWidth := 10
	maxField3ValueWidth := 10

	for _, result := range results {
		idWidth := len(fmt.Sprintf("%d", result.ID))
		if idWidth > maxIDWidth {
			maxIDWidth = idWidth
		}

		if len(result.Name) > maxNameWidth {
			maxNameWidth = len(result.Name)
		}

		if len(result.Type) > maxTypeWidth {
			maxTypeWidth = len(result.Type)
		}

		if len(result.Field1Name) > maxField1NameWidth {
			maxField1NameWidth = len(result.Field1Name)
		}
		if len(result.Field2Name) > maxField2NameWidth {
			maxField2NameWidth = len(result.Field2Name)
		}
		if len(result.Field3Name) > maxField3NameWidth {
			maxField3NameWidth = len(result.Field3Name)
		}

		if len(result.Field1Value) > maxField1ValueWidth {
			maxField1ValueWidth = len(result.Field1Value)
		}
		if len(result.Field2Value) > maxField2ValueWidth {
			maxField2ValueWidth = len(result.Field2Value)
		}
		if len(result.Field3Value) > maxField3ValueWidth {
			maxField3ValueWidth = len(result.Field3Value)
		}
	}

	// Print header
	fmt.Printf("%-*s | %-*s | %-*s | %-*s: %-*s | %-*s: %-*s | %-*s: %-*s\n",
		maxIDWidth, "ID",
		maxNameWidth, "Name",
		maxTypeWidth, "Type",
		maxField1NameWidth, "Field1",
		maxField1ValueWidth, "Value",
		maxField2NameWidth, "Field2",
		maxField2ValueWidth, "Value",
		maxField3NameWidth, "Field3",
		maxField3ValueWidth, "Value",
	)

	// Print separator
	separator := strings.Repeat("-", maxIDWidth) + "-+-" +
		strings.Repeat("-", maxNameWidth) + "-+-" +
		strings.Repeat("-", maxTypeWidth) + "-+-" +
		strings.Repeat("-", maxField1NameWidth) + "-+-" +
		strings.Repeat("-", maxField1ValueWidth) + "-+-" +
		strings.Repeat("-", maxField2NameWidth) + "-+-" +
		strings.Repeat("-", maxField2ValueWidth) + "-+-" +
		strings.Repeat("-", maxField3NameWidth) + "-+-" +
		strings.Repeat("-", maxField3ValueWidth)
	fmt.Println(separator)

	// Print results
	for _, result := range results {
		fmt.Printf("%-*d | %-*s | %-*s | %-*s: %-*s | %-*s: %-*s | %-*s: %-*s\n",
			maxIDWidth, result.ID,
			maxNameWidth, result.Name,
			maxTypeWidth, result.Type,
			maxField1NameWidth, result.Field1Name,
			maxField1ValueWidth, result.Field1Value,
			maxField2NameWidth, result.Field2Name,
			maxField2ValueWidth, result.Field2Value,
			maxField3NameWidth, result.Field3Name,
			maxField3ValueWidth, result.Field3Value,
		)
	}

	// Print summary
	fmt.Printf("\nFound %d results.\n", len(results))
}
