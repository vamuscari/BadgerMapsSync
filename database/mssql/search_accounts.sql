SELECT Id, FullName, FirstName, LastName, Email, PhoneNumber
FROM Accounts
WHERE FullName LIKE @p1 OR FirstName LIKE @p2 OR LastName LIKE @p3 OR Email LIKE @p4 OR PhoneNumber LIKE @p5