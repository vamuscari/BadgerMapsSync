INSERT OR REPLACE INTO Accounts (
    Id, FirstName, LastName, UpdatedAt
) VALUES (?, ?, ?, CURRENT_TIMESTAMP);