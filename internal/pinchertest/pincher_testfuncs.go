// Package pinchertest provides functions for stateful integration tests
// to call when requests are need to be sent
package pinchertest

import (
	"net/http"
)

// USER CRUD

func CreateUser(username, password string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/users", "", map[string]any{
		"username": username,
		"password": password,
	})
}

func GetUserCount() *http.Request {
	return MakeRequest(http.MethodGet, "/admin/users/count", "", nil)
}

func UpdateUser(token, newUsername, newPassword string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/users", token, map[string]any{
		"username": newUsername,
		"password": newPassword,
	})
}

func DeleteUser(token, username, password string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/users", token, map[string]any{
		"username": username,
		"password": password,
	})
}

func DeleteAllUsers() *http.Request {
	return MakeRequest(http.MethodPost, "/admin/reset", "", nil)
}

// USER AUTH

func LoginUser(username, password string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/login", "", map[string]any{
		"username": username,
		"password": password,
	})
}

// USER -> BUDGET CRUD

func CreateBudget(token, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets", token, map[string]any{
		"name":  name,
		"notes": notes,
	})
}

func GetUserBudgets(token string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets", token, nil)
}

func UpdateBudget(token, budgetID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func DeleteUserBudget(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID, token, nil)
}

// BUDGET -> ACCOUNT CRUD

func CreateBudgetAccount(token, budgetID, accountType, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/accounts", token, map[string]any{
		"account_type": accountType,
		"name":         name,
		"notes":        notes,
	})
}

func GetBudgetAccounts(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/accounts", token, nil)
}

func GetBudgetCapital(token, budgetID, accountID string) *http.Request {
	path := "/api/budgets/" + budgetID
	if accountID != "" {
		path += "/accounts/" + accountID
	}
	path += "/capital"

	return MakeRequest(http.MethodGet, path, token, nil)
}

func AssignMemberToBudget(token, budgetID, username, memberRole string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/members", token, map[string]any{
		"username":    username,
		"member_role": memberRole,
	})
}

func UpdateAccount(token, budgetID, accountID, accountType, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/accounts/"+accountID, token, map[string]any{
		"account_type": accountType,
		"name":         newName,
		"notes":        newNotes,
	})
}

func RevokeBudgetMembership(token, budgetID, userID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/members/"+userID, token, nil)
}

func DeleteBudgetAccount(token, budgetID, accountID, name string, deleteHard bool) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/accounts/"+accountID, token, map[string]any{
		"name":        name,
		"delete_hard": deleteHard,
	})
}

// BUDGET -> PAYEE CRUD

func CreateBudgetPayee(token, budgetID, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/payees", token, map[string]any{
		"name":  name,
		"notes": notes,
	})
}

func GetBudgetPayees(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/payees", token, nil)
}

func UpdatePayee(token, budgetID, payeeID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/payees/"+payeeID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func DeletePayee(token, budgetID, payeeID, newPayeeName string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/payees/"+payeeID, token, map[string]any{
		"new_payee_name": newPayeeName,
	})
}

// BUDGET -> CATEGORY CRUD

func CreateCategory(token, budgetID, groupName, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/categories", token, map[string]any{
		"name":       name,
		"notes":      notes,
		"group_name": groupName,
	})
}

func GetBudgetCategories(token, budgetID, query string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/categories"+query, token, nil)
}

func UpdateCategory(token, budgetID, categoryID, groupName, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/categories/"+categoryID, token, map[string]any{
		"name":       newName,
		"notes":      newNotes,
		"group_name": groupName,
	})
}

func DeleteBudgetCategory(token, budgetID, categoryID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/categories/"+categoryID, token, nil)
}

// BUDGET -> GROUP CRUD

func CreateGroup(token, budgetID, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/groups", token, map[string]any{
		"name":  name,
		"notes": notes,
	})
}

func GetBudgetGroups(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/groups", token, nil)
}

func UpdateGroup(token, budgetID, groupID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/groups/"+groupID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func DeleteBudgetGroup(token, budgetID, groupID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/groups/"+groupID, token, nil)
}

// BUDGET -> TRANSACTION CRUD

func LogTransaction(token, budgetID, accountName, transferAccountName, transactionDate, payeeName, notes string, isCleared bool, amounts map[string]int64) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/transactions", token, map[string]any{
		"account_name":          accountName,
		"transfer_account_name": transferAccountName,
		"transaction_date":      transactionDate,
		"payee_name":            payeeName,
		"notes":                 notes,
		"amounts":               amounts,
		"is_cleared":            isCleared,
	})
}

func GetTransactions(token, budgetID, accountID, categoryID, payeeID, startDate, endDate string) *http.Request {
	pathParam := ""
	if accountID != "" {
		pathParam += "/accounts/" + accountID
	} else if categoryID != "" {
		pathParam += "/categories/" + categoryID
	} else if payeeID != "" {
		pathParam += "/payees/" + payeeID
	}
	query := ""
	if startDate != "" && endDate != "" {
		query += "?start_date=" + startDate + "&end_date=" + endDate
	}
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+pathParam+"/transactions"+query, token, nil)
}

func GetTransaction(token, budgetID, transactionID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/transactions/"+transactionID, token, nil)
}

func UpdateTransaction(token, budgetID, transactionID, accountName, transferAccountName, transactionType, transactionDate, payeeName, notes, isCleared string, amounts map[string]int64) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/transactions/"+transactionID, token, map[string]any{
		"account_name":          accountName,
		"transfer_account_name": transferAccountName,
		"transaction_type":      transactionType,
		"transaction_date":      transactionDate,
		"payee_name":            payeeName,
		"notes":                 notes,
		"amounts":               amounts,
		"is_cleared":            isCleared,
	})
}

func DeleteTransaction(token, budgetID, transactionID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/transactions/"+transactionID, token, nil)
}

// BUDGET -> ASSIGNMENT CRUD

func AssignMoneyToCategory(token, budgetID, monthID, categoryID string, amount int64) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/months/"+monthID+"/categories/"+categoryID, token, map[string]int64{
		"amount": amount,
	})
}

func GetMonthCategoryReport(token, budgetID, monthID, categoryID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/months/"+monthID+"/categories/"+categoryID, token, nil)
}

func GetMonthCategoryReports(token, budgetID, monthID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/months/"+monthID+"/categories/", token, nil)
}

func GetMonthReport(token, budgetID, monthID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/months/"+monthID, token, nil)
}
