SELECT 
    Id as CustomerID,
    FirstName,
    LastName,
    Email,
    PhoneNumber
FROM Accounts
WHERE Id IS NOT NULL
ORDER BY Id 