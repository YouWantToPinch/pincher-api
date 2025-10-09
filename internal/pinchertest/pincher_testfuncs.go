package pinchertest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

func Call(mux http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// USER CRUD

func CreateUser(username, password string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"username":"%v","password":"%v"}`, username, password))
	req := httptest.NewRequest(http.MethodPost, "/api/users", payload)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetUserCount() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/admin/users/count", nil)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteUser(token, username, password string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"username":"%v","password":"%v"}`, username, password))
	req := httptest.NewRequest(http.MethodDelete, "/api/users", payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteAllUsers() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/admin/reset", nil)
	return req
}

// USER AUTH

func LoginUser(username, password string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"username":"%v","password":"%v"}`, username, password))
	req := httptest.NewRequest(http.MethodPost, "/api/login", payload)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// USER -> BUDGET CRUD

func CreateBudget(token, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v"}`, name, notes))
	req := httptest.NewRequest(http.MethodPost, "/api/budgets", payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetUserBudgets(token string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/budgets", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteUserBudget(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v", budgetID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// BUDGET -> ACCOUNT CRUD

func CreateBudgetAccount(token, budgetID, accountType, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"account_type":"%v","name":"%v","notes":"%v"}`, accountType, name, notes))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/accounts", budgetID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetBudgetAccounts(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/accounts", budgetID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func AssignMemberToBudget(token, budgetID, userID, memberRole string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"user_id":"%v","member_role":"%v"}`, userID, memberRole))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/members", budgetID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func RevokeBudgetMembership(token, budgetID, userID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/members/%v", budgetID, userID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteBudgetAccount(token, budgetID, accountID, name, deleteHard string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","delete_hard":"%v"}`, name, deleteHard))
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/accounts/%v", budgetID, accountID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// BUDGET -> PAYEE CRUD

func CreateBudgetPayee(token, budgetID, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v"}`, name, notes))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/payees", budgetID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetBudgetPayees(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/payees", budgetID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// BUDGET -> CATEGORY CRUD

func CreateCategory(token, budgetID, groupID, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v","group_id":"%v"}`, name, notes, groupID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/categories", budgetID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetBudgetCategories(token, budgetID, query string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/categories%v", budgetID, query), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func AssignCategoryToGroup(token, budgetID, categoryID, groupID string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"group_id":"%v"}`, groupID))
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/budgets/%v/categories/%v", budgetID, categoryID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteBudgetCategory(token, budgetID, categoryID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/categories/%v", budgetID, categoryID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// BUDGET -> GROUP CRUD

func CreateGroup(token, budgetID, name, notes string) *http.Request {
	payload := strings.NewReader(fmt.Sprintf(`{"name":"%v","notes":"%v"}`, name, notes))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/groups", budgetID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetBudgetGroups(token, budgetID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/groups", budgetID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteBudgetGroup(token, budgetID, groupID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/groups/%v", budgetID, groupID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// BUDGET -> TRANSACTION CRUD

func LogTransaction(token, budgetID, accountID, transactionDate, payeeID, notes, amounts, isCleared string) *http.Request {
	payloadString := fmt.Sprintf(`{"account_id":"%v","transaction_date":"%v","payee_id":"%v","notes":"%v","amounts":%v,"is_cleared":"%v"}`, accountID, transactionDate, payeeID, notes, amounts, isCleared)
	//slog.Info(fmt.Sprintf("Payload string for new log transaction: %v", payloadString))
	payload := strings.NewReader(payloadString)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/budgets/%v/transactions", budgetID), payload)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
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
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func GetTransaction(token, budgetID, transactionID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/budgets/%v/transactions/%v", budgetID, transactionID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func DeleteTransaction(token, budgetID, transactionID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/budgets/%v/transactions/%v", budgetID, transactionID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")
	return req
}
