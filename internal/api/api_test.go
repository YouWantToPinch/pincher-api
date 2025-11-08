package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	pt "github.com/YouWantToPinch/pincher-api/internal/pinchertest"
	"github.com/stretchr/testify/assert"
)

// ---------------
// HELPER FUNCS
// ---------------

func GetJSONField(w *httptest.ResponseRecorder, field string) (any, error) {
	res := w.Result()
	defer res.Body.Close()

	var body map[string]any
	decoder := json.NewDecoder(res.Body)
	decoder.UseNumber()
	err := decoder.Decode(&body)
	if err != nil {
		return nil, err
	}
	val, ok := body[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found in response", field)
	}

	if num, ok := val.(json.Number); ok {
		if i, err := num.Int64(); err == nil {
			return i, nil
		}
		if f, err := num.Float64(); err == nil {
			return f, nil
		}
	}

	return val, nil
}

func Call(mux http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// ---------------
// TESTING
// ---------------

// Should properly make, count, and delete users
func Test_MakeAndResetUsers(t *testing.T) {
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
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// Create two users
	w = Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, 201, w.Code)
	w = Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, 201, w.Code)

	// User count should now be 2
	w = Call(mux, pt.GetUserCount())
	count, _ := GetJSONField(w, "count")
	if !assert.Equal(t, count, int64(2)) {
		t.Fatalf("expected user count of 2, but got %d", count)
	}

	// Delete all users
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// User count should now be 0 again
	w = Call(mux, pt.GetUserCount())
	count, _ = GetJSONField(w, "count")
	if !assert.Equal(t, int64(0), count) {
		t.Fatalf("expected user count of 0, but got %d", count)
	}
}

// Should make and log in 2 users, which should be able to then delete themselves,
// but not each other
func Test_MakeLoginDeleteUsers(t *testing.T) {
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
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// Create two users
	w = Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, w.Code)
	w = Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, http.StatusCreated, w.Code)

	// Log in both users
	w = Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(w, "token")
	w = Call(mux, pt.LoginUser("user2", "pwd2"))

	// attempt deletion of user 2 as user 1
	w = Call(mux, pt.DeleteUser(jwt1.(string), "user2", "pwd2"))
	assert.Equal(t, http.StatusUnauthorized, w.Code, "Should have a valid JSON web token from login in order to perform a user deletion")

	// delete user 1 as user 1
	w = Call(mux, pt.DeleteUser(jwt1.(string), "user1", "pwd1"))

	// User count should now be 1
	w = Call(mux, pt.GetUserCount())
	count, _ := GetJSONField(w, "count")
	if !assert.Equal(t, int64(1), count) {
		t.Fatalf("expected user count of 1, but got %d", count)
	}

	// Delete all users
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// User count should now be 0 again
	w = Call(mux, pt.GetUserCount())
	count, _ = GetJSONField(w, "count")
	if !assert.Equal(t, int64(0), count) {
		t.Fatalf("expected user count of 0, but got %d", count)
	}
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
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// Create four users
	w = Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, w.Code)
	w = Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, http.StatusCreated, w.Code)
	user2, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateUser("user3", "pwd3"))
	assert.Equal(t, http.StatusCreated, w.Code)
	user3, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateUser("user4", "pwd4"))
	assert.Equal(t, http.StatusCreated, w.Code)
	user4, _ := GetJSONField(w, "id")

	// Log in four users
	w = Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(w, "token")
	w = Call(mux, pt.LoginUser("user2", "pwd2"))
	jwt2, _ := GetJSONField(w, "token")
	w = Call(mux, pt.LoginUser("user3", "pwd3"))
	w = Call(mux, pt.LoginUser("user4", "pwd4"))
	jwt4, _ := GetJSONField(w, "token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	w = Call(mux, pt.CreateBudget(jwt1.(string), "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."))
	budget1, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateBudget(jwt1.(string), "Personal", "user1's budget for personal finance."))

	// Try adding user2 as ADMIN using user4 (not in budget), then as MANAGER. Both should fail.
	w = Call(mux, pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "ADMIN"))
	w = Call(mux, pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "MANAGER"))

	// Add user4 to Webflyx Org as user1 ADMIN, with role: VIEWER
	w = Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user4.(string), "VIEWER"))

	// user4 should be assigned to only 1 budget
	w = Call(mux, pt.GetUserBudgets(jwt4.(string)))
	var gotBudgets []Budget
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 1, len(gotBudgets))

	// Try adding user2 as MANAGER using u4. Should fail auth check.
	w = Call(mux, pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "MANAGER"))

	// As user1 ADMIN, add user2 and user3 as MANAGER and CONTRIBUTOR, respectively.
	w = Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user2.(string), "MANAGER"))
	w = Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user3.(string), "CONTRIBUTOR"))

	// Attempt deletion of Webflyx Org budget as user2. Should fail; only admin can do it.
	w = Call(mux, pt.DeleteUserBudget(jwt2.(string), budget1.(string)))
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Attempt to revoke user3's Webflyx Org membership as user4. Should fail.
	w = Call(mux, pt.RevokeBudgetMembership(jwt4.(string), budget1.(string), user3.(string)))
	// Revoke user3's Webflyx Org membership as user1. Should succeed.
	w = Call(mux, pt.RevokeBudgetMembership(jwt1.(string), budget1.(string), user3.(string)))

	// user1 should be assigned to 2 budgets: Webflyx Org & their personal budget
	w = Call(mux, pt.GetUserBudgets(jwt1.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 2, len(gotBudgets))

	// Attempt deletion of Webflyx Org budget as user1. Should succeed.
	w = Call(mux, pt.DeleteUserBudget(jwt1.(string), budget1.(string)))
	assert.Equal(t, http.StatusNoContent, w.Code)

	// user1 should be assigned to only 1 budget now: their personal budget.
	w = Call(mux, pt.GetUserBudgets(jwt1.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 1, len(gotBudgets))

	// user4 should be assigned to NO budgets, now.
	w = Call(mux, pt.GetUserBudgets(jwt4.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(w.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 0, len(gotBudgets))
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
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// Create four users
	w = Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, w.Code)
	w = Call(mux, pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, http.StatusCreated, w.Code)
	user2, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateUser("user3", "pwd3"))
	assert.Equal(t, http.StatusCreated, w.Code)
	user3, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateUser("user4", "pwd4"))
	assert.Equal(t, http.StatusCreated, w.Code)
	user4, _ := GetJSONField(w, "id")

	// Log in four users
	w = Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(w, "token")
	w = Call(mux, pt.LoginUser("user2", "pwd2"))
	jwt2, _ := GetJSONField(w, "token")
	w = Call(mux, pt.LoginUser("user3", "pwd3"))
	jwt3, _ := GetJSONField(w, "token")
	w = Call(mux, pt.LoginUser("user4", "pwd4"))
	jwt4, _ := GetJSONField(w, "token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	w = Call(mux, pt.CreateBudget(jwt1.(string), "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."))
	budget1, _ := GetJSONField(w, "id")
	w = Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user2.(string), "MANAGER"))
	w = Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user3.(string), "CONTRIBUTOR"))
	w = Call(mux, pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user4.(string), "VIEWER"))

	// user2 MANAGER: Adding account, groups, & categories.
	w = Call(mux, pt.CreateBudgetAccount(jwt2.(string), budget1.(string), "savings", "Saved Org Funds", "Represents a bank account holding business capital."))
	// account1, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateBudgetAccount(jwt2.(string), budget1.(string), "credit", "Employee Business Credit Account", "Employees use cards that pull from this account to pay for business expenses."))
	account2, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateGroup(jwt2.(string), budget1.(string), "Business Capital", "Categories related to company capital"))
	group1, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateCategory(jwt2.(string), budget1.(string), group1.(string), "Surplus", "Category representing surplus funding to be spent on elective improvements to organization headquarters or employee bonuses."))
	category1, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateCategory(jwt2.(string), budget1.(string), group1.(string), "Expenses", "Category representing funds to be used for employee expenses while on the job."))
	category2, _ := GetJSONField(w, "id")

	// user3 CONTRIBUTOR: Adding transactions (EX: gas station).
	w = Call(mux, pt.CreateBudgetPayee(jwt3.(string), budget1.(string), "Smash & Dash", "A gas & convenience store"))
	payee1, _ := GetJSONField(w, "id")

	transaction1Amounts := fmt.Sprintf(`{"%s": %d}`, category2.(string), -1800)
	w = Call(mux, pt.LogTransaction(jwt3.(string), budget1.(string), account2.(string), "NONE", "WITHDRAWAL", "2025-09-15T23:17:00Z", payee1.(string), "I filled up vehicle w/ plate no. 555-555 @ the Smash & Pass gas station.", transaction1Amounts, "true"))
	//transaction1, _ := GetJSONField(w, "id")

	transaction2Amounts := fmt.Sprintf(`{"%s": %d}`, category1.(string), -400)
	w = Call(mux, pt.LogTransaction(jwt3.(string), budget1.(string), account2.(string), "NONE", "WITHDRAWAL", "2025-09-15T23:22:00Z", payee1.(string), "Yeah, I got a drink in the convenience store too; sue me. Take it out of my bonus or whatever.", transaction2Amounts, "true"))
	// transaction2, _ := GetJSONField(w, "id")

	// user4 VIEWER: Works for accounting; reading transactions from employees.
	w = Call(mux, pt.GetTransactions(jwt4.(string), budget1.(string), account2.(string), "", "", "", ""))
	assert.Equal(t, http.StatusOK, w.Code)
	w = Call(mux, pt.GetTransactions(jwt4.(string), budget1.(string), "", "", payee1.(string), "", ""))
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete all users
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)
}

// Build a budget and give it a predictable amount of money to operate with between 1-2 accounts.
// Log transactions of each type, and check that the endpoint for getting budget capital responds with the right amount(s).
func Test_TransactionTypesAndCapital(t *testing.T) {
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
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// Create user
	w = Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, w.Code)

	// Log in user
	w = Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(w, "token")

	// user2 ADMIN: Creating personal budget, making accounts and deposit transactions.
	w = Call(mux, pt.CreateBudget(jwt1.(string), "Personal Budget", "For personal accounting (user1)."))
	budget1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateBudgetAccount(jwt1.(string), budget1.(string), "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."))
	account1, _ := GetJSONField(w, "id")
	w = Call(mux, pt.CreateBudgetAccount(jwt1.(string), budget1.(string), "credit", "Credit (Big Banking Inc)", "Reflects my credit account opened via Big Banking, Inc."))
	account2, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateGroup(jwt1.(string), budget1.(string), "Spending", "Categories related to day-to-day spending"))
	group1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateCategory(jwt1.(string), budget1.(string), group1.(string), "Dining Out", "Money for ordering takeout or dining in."))
	category1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Webflyx Org", "user1 employer"))
	payee1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."))
	payee2, _ := GetJSONField(w, "id")

	// deposit some money into checking account, allocated (but not explicitly assigned) to the DINING OUT category
	depositAmount := fmt.Sprintf(`{"%s": %d}`, category1.(string), 10000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "DEPOSIT", "2025-09-15T17:00:00Z", payee1.(string), "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", depositAmount, "true"))

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), account1.(string)))
	budgetCheckingCapital, _ := GetJSONField(w, "capital")
	assert.Equal(t, int64(10000), budgetCheckingCapital)

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), account2.(string)))
	budgetCreditCapital, _ := GetJSONField(w, "capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ := GetJSONField(w, "capital")
	assert.Equal(t, int64(10000), budgetTotalCapital)

	// spend money out of a credit account
	spendAmount := fmt.Sprintf(`{"%s": %d}`, category1.(string), 5000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account2.(string), "NONE", "WITHDRAWAL", "2025-09-15T18:00:00Z", payee2.(string), "$50 dinner at a restaurant", spendAmount, "true"))

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), account2.(string)))
	budgetCreditCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(-5000), budgetCreditCapital)

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// pay off credit account, using the checking account, using a transfer transaction
	transferAmount := fmt.Sprintf(`{"TRANSFER AMOUNT": %d}`, 5000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), account2.(string), "TRANSFER_FROM", "2025-09-15T19:00:00Z", "ACCOUNT TRANSFER", "Pay off credit account balance", transferAmount, "true"))

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), account2.(string)))
	budgetCreditCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), account1.(string)))
	budgetCheckingCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(5000), budgetCheckingCapital)

	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// Delete all users
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)
}

// Build a budget and simulate 3 months of transactions and dollar assignment.
// Then, ensure that:
//  1. Deposit transactions with non-null categories contribute to its balance by virtue of merely being counted as activity.
//  2. Assignments are agnostic of whether or not there is an equal amount of money between the accounts they represent.
//  3. For each month, we get the assignment, activity, and balance totals we would expect from the actions recorded within the budget.
func Test_CategoryMoneyAssignment(t *testing.T) {
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
	w = Call(mux, pt.DeleteAllUsers())
	assert.Equal(t, 200, w.Code)

	// Create user
	w = Call(mux, pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, w.Code)

	// Log in user
	w = Call(mux, pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(w, "token")

	// user2 ADMIN: Creating personal budget, making account and other resources.
	w = Call(mux, pt.CreateBudget(jwt1.(string), "Personal Budget", "For personal accounting (user1)."))
	budget1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateBudgetAccount(jwt1.(string), budget1.(string), "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."))
	account1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateGroup(jwt1.(string), budget1.(string), "Ungrouped", "All categories"))
	group1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateCategory(jwt1.(string), budget1.(string), group1.(string), "Dining Out", "Money for ordering takeout or dining in."))
	category1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateCategory(jwt1.(string), budget1.(string), group1.(string), "Savings", "My savings fund."))
	category2, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Webflyx Org", "user1 employer"))
	payee1, _ := GetJSONField(w, "id")

	w = Call(mux, pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."))
	payee2, _ := GetJSONField(w, "id")

	// SEPTEMBER 2025 ACTIVITIES
	// deposit some money into the checking account, allocated (but not explicitly assigned) to the Dining Out category
	depositAmount := fmt.Sprintf(`{"%s": %d}`, category1.(string), 10000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "DEPOSIT", "2025-09-15T17:00:00Z", payee1.(string), "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", depositAmount, "true"))

	// spend money out of Dining Out category
	spendAmount := fmt.Sprintf(`{"%s": %d}`, category1.(string), 5000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "WITHDRAWAL", "2025-09-15T18:00:00Z", payee2.(string), "$50 dinner at a restaurant", spendAmount, "true"))

	// we expect that there's 5000 in capital remaining, and NO assignable money.
	// 5000 in Dining Out; 0 in Savings.
	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ := GetJSONField(w, "capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	w = Call(mux, pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-09-01", category1.(string)))
	balanceCategory1, _ := GetJSONField(w, "balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	w = Call(mux, pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-09-01", category2.(string)))
	balanceCategory2, _ := GetJSONField(w, "balance")
	assert.Equal(t, int64(0), balanceCategory2)

	// OCTOBER 2025 ACTIVITIES
	// deposit more money into the checking account, with NO category allocation.
	// Assign some (but not all of it, to test for underassignment) to each of two categories.
	depositAmount = fmt.Sprintf(`{"%s": %d}`, "UNCATEGORIZED", 10000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "DEPOSIT", "2025-10-15T17:00:00Z", payee1.(string), "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", depositAmount, "true"))

	w = Call(mux, pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-10-01", category1.(string), 4000))
	w = Call(mux, pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-10-01", category2.(string), 5000))

	// spend money out of Dining Out category
	spendAmount = fmt.Sprintf(`{"%s": %d}`, category1.(string), 4000)
	w = Call(mux, pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "WITHDRAWAL", "2025-10-15T18:00:00Z", payee2.(string), "I was very busy having fun, fun, fun!", spendAmount, "true"))

	// we expect that there's 11000 in capital remaining, and 1000 in assignable money.
	// 5000 in Dining Out; 5000 in Savings.
	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	w = Call(mux, pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-10-01", category1.(string)))
	balanceCategory1, _ = GetJSONField(w, "balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	w = Call(mux, pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-10-01", category2.(string)))
	balanceCategory2, _ = GetJSONField(w, "balance")
	assert.Equal(t, int64(5000), balanceCategory2)

	w = Call(mux, pt.GetMonthReport(jwt1.(string), budget1.(string), "2025-10-01"))
	budgetTotalBalance, _ := GetJSONField(w, "balance")
	assert.Equal(t, int64(1000), (budgetTotalCapital.(int64) - budgetTotalBalance.(int64)))

	// NOVEMBER 2025 ACTIVITIES
	// deposit NO more money into the checking account.
	// Assign the 1000 left available from OCTOBER to the SAVINGS category.
	// Assign 1000 (that we don't have) to DINING OUT to test for overassignment.

	w = Call(mux, pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-11-01", category2.(string), 1000))
	w = Call(mux, pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-11-01", category1.(string), 1000))

	// we expect that there's still just 11000 in capital remaining, and -1000 in assignable money, which indicates overassignment.
	// 6000 in Dining Out; 6000 in Savings.
	w = Call(mux, pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(w, "capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	w = Call(mux, pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-11-01", category1.(string)))
	balanceCategory1, _ = GetJSONField(w, "balance")
	assert.Equal(t, int64(6000), balanceCategory1)

	w = Call(mux, pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-11-01", category2.(string)))
	balanceCategory2, _ = GetJSONField(w, "balance")
	assert.Equal(t, int64(6000), balanceCategory2)

	w = Call(mux, pt.GetMonthReport(jwt1.(string), budget1.(string), "2025-11-01"))
	budgetTotalBalance, _ = GetJSONField(w, "balance")
	assert.Equal(t, int64(-1000), (budgetTotalCapital.(int64) - budgetTotalBalance.(int64)))

	// Delete all users
	//w = Call(mux, pt.DeleteAllUsers())
	//assert.Equal(t, 200, w.Code)
}
