package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	pt "github.com/YouWantToPinch/pincher-api/internal/pinchertest"
	"github.com/stretchr/testify/assert"
)

// ---------------
// TESTING
// ---------------

// Should properly make, count, and delete users
func Test_MakeAndResetUsers(t *testing.T) {
	// TEST SETUP
	var err error
	var w *httptest.ResponseRecorder
	// SERVER SETUP
	const port = "8080"
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: SetupMux(cfg),
	}
	mux := pincher.Handler
	// REQUESTS

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)

	// Create two users
	w = pt.Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, w.Code, 201)
	w = pt.Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, w.Code, 201)

	// User count should now be 2
	w = pt.Call(mux, pt.GetUserCount())
	var count int64
	err = json.NewDecoder(w.Body).Decode(&count)
	if err != nil {
		t.Fatalf("failed to decode response body as int64: %v", err)
	}
	if !assert.Equal(t, count, int64(2)) {
		t.Fatalf("expected user count of 2, but got %d", count)
	}

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)

	// User count should now be 0 again
	w = pt.Call(mux, pt.GetUserCount())
	err = json.NewDecoder(w.Body).Decode(&count)
	if err != nil {
		t.Fatalf("failed to decode response body as int64: %v", err)
	}
	if !assert.Equal(t, count, int64(0)) {
		t.Fatalf("expected user count of 0, but got %d", count)
	}
}

// Should make and log in 2 users, which should be able to then delete themselves,
// but not each other
func Test_MakeLoginDeleteUsers(t *testing.T) {
	// TEST SETUP
	var err error
	var w *httptest.ResponseRecorder
	// SERVER SETUP
	const port = "8080"
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: SetupMux(cfg),
	}
	mux := pincher.Handler
	// REQUESTS

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)

	// Create two users
	w = pt.Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, w.Code, http.StatusCreated)
	w = pt.Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, w.Code, http.StatusCreated)

	// Log in both users
	w = pt.Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := pt.GetJSONField(w, "token")
	w = pt.Call(mux, pt.LoginUser("user2", "pwd2"))

	// attempt deletion of user 2 as user 1
	w = pt.Call(mux, pt.DeleteUser(jwt1.(string), "user2", "pwd2"))
	assert.Equal(t, w.Code, http.StatusUnauthorized, "Should have a valid JSON web token from login in order to perform a user deletion")

	// delete user 1 as user 1
	w = pt.Call(mux, pt.DeleteUser(jwt1.(string), "user1", "pwd1"))

	// User count should now be 1
	w = pt.Call(mux, pt.GetUserCount())
	var count int64
	err = json.NewDecoder(w.Body).Decode(&count)
	if err != nil {
		t.Fatalf("failed to decode response body as int64: %v", err)
	}
	if !assert.Equal(t, count, int64(1)) {
		t.Fatalf("expected user count of 1, but got %d", count)
	}

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)

	// User count should now be 0 again
	w = pt.Call(mux, pt.GetUserCount())
	err = json.NewDecoder(w.Body).Decode(&count)
	if err != nil {
		t.Fatalf("failed to decode response body as int64: %v", err)
	}
	if !assert.Equal(t, count, int64(0)) {
		t.Fatalf("expected user count of 0, but got %d", count)
	}
}

// Build a small organizational budget system.
// make four users, each with a unique role,
// and let them each perform authorized actions.
func Test_BuildOrgLogTransaction(t *testing.T) {
	// TEST SETUP
	var w *httptest.ResponseRecorder
	// SERVER SETUP
	const port = "8080"
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: SetupMux(cfg),
	}
	mux := pincher.Handler
	// REQUESTS

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)

	// Create four users
	w = pt.Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, w.Code, http.StatusCreated)
	w = pt.Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, w.Code, http.StatusCreated)
	user2, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateUser("user3", "pwd3"))
	assert.Equal(t, w.Code, http.StatusCreated)
	user3, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateUser("user4", "pwd4"))
	assert.Equal(t, w.Code, http.StatusCreated)
	user4, _ := pt.GetJSONField(w, "id")

	// Log in four users
	w = pt.Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := pt.GetJSONField(w, "token")
	w = pt.Call(mux, pt.LoginUser("user2", "pwd2"))
	jwt2, _ := pt.GetJSONField(w, "token")
	w = pt.Call(mux, pt.LoginUser("user3", "pwd3"))
	jwt3, _ := pt.GetJSONField(w, "token")
	w = pt.Call(mux, pt.LoginUser("user4", "pwd4"))
	jwt4, _ := pt.GetJSONField(w, "token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	w = pt.Call(mux, pt.CreateBudget(jwt1.(string), "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."))
	budget1, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user2.(string), "MANAGER"))
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user3.(string), "CONTRIBUTOR"))
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user4.(string), "VIEWER"))

	// user2 MANAGER: Adding account, groups, & categories.
	w = pt.Call(mux, pt.CreateBudgetAccount(jwt2.(string), budget1.(string), "savings", "Saved Org Funds", "Represents a bank account holding business capital."))
	account1, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateBudgetAccount(jwt2.(string), budget1.(string), "credit", "Employee Business Credit Account", "Employees use cards that pull from this account to pay for business expenses."))
	account2, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateGroup(jwt2.(string), budget1.(string), "Business Capital", "Categories related to company capital"))
	group1, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateCategory(jwt2.(string), budget1.(string), group1.(string), "Surplus", "Category representing surplus funding to be spent on elective improvements to organization headquarters."))
	w = pt.Call(mux, pt.CreateCategory(jwt2.(string), budget1.(string), group1.(string), "Surplus", "Category representing surplus funding to be spent on elective improvements to organization headquarters."))

	// user3 CONTRIBUTOR: Driving a company vehicle; needs to fuel up.
	w = pt.Call(mux, pt.CreateBudgetPayee(jwt3.(string), budget1.(string), "Smash & Dash", "A gas & convenience store"))
	payee1, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.LogTransaction(jwt3.(string), budget1.(string), account2.(string), "2025-09-15T23:17:00Z", payee1.(string), "I filled up vehicle w/ plate no. 555-555 @ the Smash & Pass gas station.", "true"))

	// user4 VIEWER: Works for accounting; reading transactions from employees.
	w = pt.Call(mux, pt.GetTransactions(jwt4.(string), budget1.(string), account1.(string), "", ""))
	assert.Equal(t, w.Code, http.StatusOK)
	w = pt.Call(mux, pt.GetTransactions(jwt4.(string), budget1.(string), account2.(string), "", ""))
	assert.Equal(t, w.Code, http.StatusOK)

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)
}

/*
Creates two budgets with an admin user, and adds other users to the first.
Along the way, various roles attempt various actions that
they should or should not be able to do; authorizations that
should be verified.
*/
func Test_BuildOrgDoAuthChecks(t *testing.T) {
	// TEST SETUP
	var err error
	var w *httptest.ResponseRecorder
	// SERVER SETUP
	const port = "8080"
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{
		Addr:    ":" + port,
		Handler: SetupMux(cfg),
	}
	mux := pincher.Handler
	// REQUESTS

	// Delete all users
	w = pt.Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, w.Code, 200)

	// Create four users
	w = pt.Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, w.Code, http.StatusCreated)
	w = pt.Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, w.Code, http.StatusCreated)
	user2, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateUser("user3", "pwd3"))
	assert.Equal(t, w.Code, http.StatusCreated)
	user3, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateUser("user4", "pwd4"))
	assert.Equal(t, w.Code, http.StatusCreated)
	user4, _ := pt.GetJSONField(w, "id")

	// Log in four users
	w = pt.Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := pt.GetJSONField(w, "token")
	w = pt.Call(mux, pt.LoginUser("user2", "pwd2"))
	jwt2, _ := pt.GetJSONField(w, "token")
	w = pt.Call(mux, pt.LoginUser("user3", "pwd3"))
	w = pt.Call(mux, pt.LoginUser("user4", "pwd4"))
	jwt4, _ := pt.GetJSONField(w, "token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	w = pt.Call(mux, pt.CreateBudget(jwt1.(string), "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."))
	budget1, _ := pt.GetJSONField(w, "id")
	w = pt.Call(mux, pt.CreateBudget(jwt1.(string), "Personal", "user1's budget for personal finance."))

	// Try adding user2 as ADMIN using user4 (not in budget), then as MANAGER. Both should fail.
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "ADMIN"))
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "MANAGER"))

	// Add user4 to Webflyx Org as user1 ADMIN, with role: VIEWER
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user4.(string), "VIEWER"))

	// user4 should be assigned to only 1 budget
	w = pt.Call(mux, pt.GetUserBudgets(jwt4.(string)))
	var gotBudgets []Budget
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, len(gotBudgets), 1)

	// Try adding user2 as MANAGER using u4. Should fail auth check.
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "MANAGER"))

	// As user1 ADMIN, add user2 and user3 as MANAGER and CONTRIBUTOR, respectively.
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user2.(string), "MANAGER"))
	w = pt.Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user3.(string), "CONTRIBUTOR"))

	// Attempt deletion of Webflyx Org budget as user2. Should fail; only admin can do it.
	w = pt.Call(mux, pt.DeleteUserBudget(jwt2.(string), budget1.(string)))
	assert.NotEqual(t, w.Code, http.StatusNoContent)

	// Attempt to revoke user3's Webflyx Org membership as user4. Should fail.
	w = pt.Call(mux, pt.RevokeBudgetMembership(jwt4.(string), budget1.(string), user3.(string)))
	// Revoke user3's Webflyx Org membership as user1. Should succeed.
	w = pt.Call(mux, pt.RevokeBudgetMembership(jwt1.(string), budget1.(string), user3.(string)))

	// user1 should be assigned to 2 budgets: Webflyx Org & their personal budget
	w = pt.Call(mux, pt.GetUserBudgets(jwt1.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, len(gotBudgets), 2)

	// Attempt deletion of Webflyx Org budget as user1. Should succeed.
	w = pt.Call(mux, pt.DeleteUserBudget(jwt1.(string), budget1.(string)))
	assert.Equal(t, w.Code, http.StatusNoContent)

	// user1 should be assigned to only 1 budget now: their personal budget.
	w = pt.Call(mux, pt.GetUserBudgets(jwt1.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, len(gotBudgets), 1)

	// user4 should be assigned to NO budgets, now.
	w = pt.Call(mux, pt.GetUserBudgets(jwt4.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, len(gotBudgets), 0)
}
