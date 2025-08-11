SELECT Id, FullName, FirstName, LastName, Email, PhoneNumber
FROM Accounts
WHERE FullName LIKE $1 OR FirstName LIKE $2 OR LastName LIKE $3 OR Email LIKE $4 OR PhoneNumber LIKE $5