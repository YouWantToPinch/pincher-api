package pinchertest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

// ========== MIDDLEWARE ==========

func headerJSON(req *http.Request) *http.Request {
	req.Header.Set("Content-Type", "application/json")
	return req
}

func requireToken(req *http.Request, token string) *http.Request {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	return req
}

// USER CRUD

func CreateUser(username, password string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"username":"%v","password":"%v"}`, username, password))
	req := httptest.NewRequest(http.MethodPost, "/api/users", payload)
	return headerJSON(req)
}

func GetUserCount() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/admin/users/count", nil)
	return headerJSON(req)
}

func DeleteUser(token, username, password string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"username":"%v","password":"%v"}`, username, password))
	req := httptest.NewRequest(http.MethodDelete, "/api/users", payload)
	return headerJSON(requireToken(req, token))
}

func DeleteAllUsers() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/admin/reset", nil)
	return req
}

// USER AUTH

func LoginUser(username, password string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"username":"%v","password":"%v"}`, username, password))
	req := httptest.NewRequest(http.MethodPost, "/api/login", payload)
	return headerJSON(req)
}

// USER -> BUDGET CRUD

func CreateBudget(token, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v"}`, name, notes))
	req := httptest.NewRequest(http.MethodPost, "/api/budgets", payload)
	return headerJSON(requireToken(req, token))
}

func GetUserBudgets(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
	return headerJSON(requireToken(req, token))
}

func DeleteUserBudget(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v", budgetID), nil)
	return headerJSON(requireToken(req, token))
}

// BUDGET -> ACCOUNT CRUD

func CreateBudgetAccount(token, budgetID, accountType, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"account_type":"%v","name":"%v","notes":"%v"}`, accountType, name, notes))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/accounts", budgetID), payload)
	return headerJSON(requireToken(req, token))
}

func GetBudgetAccounts(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/accounts", budgetID), nil)
	return headerJSON(requireToken(req, token))
}

func GetBudgetCapital(token, budgetID, accountID string) *http.Request {
	var path string
	if accountID != "" {
		path = fmt.Sprintf("/api/budgets/%v/accounts/%v/capital", budgetID, accountID)
	} else {
		path = fmt.Sprintf("/api/budgets/%v/capital", budgetID)
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	return headerJSON(requireToken(req, token))
}

func AssignMemberToBudget(token, budgetID, userID, memberRole string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"user_id":"%v","member_role":"%v"}`, userID, memberRole))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/members", budgetID), payload)
	return headerJSON(requireToken(req, token))
}

func RevokeBudgetMembership(token, budgetID, userID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/members/%v", budgetID, userID), nil)
	return headerJSON(requireToken(req, token))
}

func DeleteBudgetAccount(token, budgetID, accountID, name, deleteHard string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","delete_hard":"%v"}`, name, deleteHard))
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/accounts/%v", budgetID, accountID), payload)
	return headerJSON(requireToken(req, token))
}

// BUDGET -> PAYEE CRUD

func CreateBudgetPayee(token, budgetID, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v"}`, name, notes))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/payees", budgetID), payload)
	return headerJSON(requireToken(req, token))
}

func GetBudgetPayees(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/payees", budgetID), nil)
	return headerJSON(requireToken(req, token))
}

// BUDGET -> CATEGORY CRUD

func CreateCategory(token, budgetID, groupID, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v","group_id":"%v"}`, name, notes, groupID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/categories", budgetID), payload)
	return headerJSON(requireToken(req, token))
}

func GetBudgetCategories(token, budgetID, query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/categories%v", budgetID, query), nil)
	return headerJSON(requireToken(req, token))
}

func AssignCategoryToGroup(token, budgetID, categoryID, groupID string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"group_id":"%v"}`, groupID))
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/budgets/%v/categories/%v", budgetID, categoryID), payload)
	return headerJSON(requireToken(req, token))
}

func DeleteBudgetCategory(token, budgetID, categoryID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/categories/%v", budgetID, categoryID), nil)
	return headerJSON(requireToken(req, token))
}

// BUDGET -> GROUP CRUD

func CreateGroup(token, budgetID, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v"}`, name, notes))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/groups", budgetID), payload)
	return headerJSON(requireToken(req, token))
}

func GetBudgetGroups(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/groups", budgetID), nil)
	return headerJSON(requireToken(req, token))
}

func DeleteBudgetGroup(token, budgetID, groupID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/groups/%v", budgetID, groupID), nil)
	return headerJSON(requireToken(req, token))
}

// BUDGET -> TRANSACTION CRUD

func LogTransaction(token, budgetID, accountID, transferAccountID, transactionType, transactionDate, payeeID, notes, amounts, isCleared string) *http.Request {
	payloadString := fmt.Sprintf(`{"account_id":"%v","transfer_account_id":"%v","transaction_type":"%v","transaction_date":"%v","payee_id":"%v","notes":"%v","amounts":%v,"is_cleared":"%v"}`, accountID, transferAccountID, transactionType, transactionDate, payeeID, notes, amounts, isCleared)
	//slog.Debug(fmt.Sprintf("Payload string for new log transaction: %v", payloadString))
	payload := strings.NewReader(payloadString)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/transactions", budgetID), payload)
	return headerJSON(requireToken(req, token))
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v%v/transactions?start_date=%v&end_date=%v", budgetID, pathParam, startDate, endDate), nil)
	return headerJSON(requireToken(req, token))
}

func GetTransaction(token, budgetID, transactionID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/transactions/%v", budgetID, transactionID), nil)
	return headerJSON(requireToken(req, token))
}

func DeleteTransaction(token, budgetID, transactionID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/transactions/%v", budgetID, transactionID), nil)
	return headerJSON(requireToken(req, token))
}

// BUDGET -> ASSIGNMENT CRUD

func AssignMoneyToCategory(token, budgetID, monthID, categoryID string, amount int64) *http.Request {
	payloadString := fmt.Sprintf(`{"amount":%d}`, amount)
	payload := strings.NewReader(payloadString)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/months/%v/categories/%v", budgetID, monthID, categoryID), payload)
	return headerJSON(requireToken(req, token))
}

func GetMonthCategoryReport(token, budgetID, monthID, categoryID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/months/%v/categories/%v", budgetID, monthID, categoryID), nil)
	return headerJSON(requireToken(req, token))
}

func GetMonthCategoryReports(token, budgetID, monthID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/months/%v/categories", budgetID, monthID), nil)
	return headerJSON(requireToken(req, token))
}

func GetMonthReport(token, budgetID, monthID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/months/%v", budgetID, monthID), nil)
	return headerJSON(requireToken(req, token))
}
