# Fixes


## Actions Refactor
I have the content of events/actions.go and events/app.go. I'll now update cli/action/actions.go with the logic from events/actions.go.

  My plan is to:

   1. Update `EventAction` struct: I'll replace the current EventAction struct in cli/action/actions.go with the more detailed version from events/actions.go.
   2. Update `Execute` function: I'll replace the placeholder Execute function with the implementation from events/actions.go, including the executeCommand, executeDB, and executeAPI functions.
   3. Add `Validate` function: I'll add the Validate function from events/actions.go.
   4. Update `executeCommand`: I'll use the more robust executeCommand implementation from events/actions.go.
   5. Implement `executeDB` and `executeAPI`: I'll move the logic for executeDB and executeAPI to cli/action/actions.go and adapt it to use the Executor struct.


## Events 
- Events Actions are not running properly
- Actions needs better testing.
- The Action list is not complete

## omniboxes for searching
- omibox for Accounts, Checkins, and routes.
- Accounts should search by AccountID and full_name.
- Checkins should search by AccountID and full_name. Checkin pulls by accountID
-  Routes should search by routeID and route name.
- Omiboxes should have a 3 dot config button next to them. the config has checkboxes allowing the user to disable or enable specific fields in the omibox. All should be enabled by default. 

## Explorer
- the Table should have some sort of pagination with the ability to sort by a single field if the user clicks on the field. 


