SELECT AccountId, FullName, FirstName, LastName, Email, PhoneNumber
FROM Accounts
WHERE FullName LIKE ? OR FirstName LIKE ? OR LastName LIKE ? OR Email LIKE ? OR PhoneNumber LIKE ?