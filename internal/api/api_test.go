package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	pt "github.com/YouWantToPinch/pincher-api/internal/pinchertest"
	"github.com/stretchr/testify/assert"
)

// NOTE: This integration testing has two optional implementations.
// Firstly, the conventional one; a more stateful approach recording variables for use in further mock requests.
// But secondly, in an effort to make the tests more readable, a test-case slice approach was implemented.
// Far too late, it appeared that this second implementation may only make things more readable for those
// 	tests which don't demand very detailed requests, and may otherwise be an overengineered solution to
// 	running others.
// Both implementations are left here for developer use.
// When in doubt: loop through a slice of httpTestCase structs for lighter tests,
// 	but use the more traditional, stateful approach for anything else.

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

type APIClient struct {
	Mux       http.Handler
	W         *httptest.ResponseRecorder
	Resources map[string]any
}

func (c *APIClient) Request(req *http.Request) *http.Request {
	w := httptest.NewRecorder()
	c.Mux.ServeHTTP(w, req)
	c.W = w
	return req
}

func (c *APIClient) GetResource(name string) any {
	if v, ok := c.Resources[name]; ok {
		return v
	}
	return nil
}

func (c *APIClient) SaveResourceFromJSON(field string, name string) {
	jsonObject, _ := GetJSONField(c.W, field)
	c.Resources[name] = jsonObject
	slog.Debug(fmt.Sprintf("Saved resource %s at: %v (type: %T)", name, c.Resources[name], c.Resources[name]))
}

func (c *APIClient) equalsResourceAt(expected any, resourceName string) func() bool {
	return func() bool {
		return expected == c.Resources[resourceName]
	}
}

type httpTestCase struct {
	// Optional name for subtest
	Name string
	// Path saved from making the request
	Path string
	// Request to make; use pt.MakeRequest, or a premade wrapper that uses it
	RequestFunc func() *http.Request
	// JSON objects, derived from the Response body at the given JSON fields, to assign to given names
	SaveFields map[string]string
	// Status code that this subtest expects to receive in response to its Request
	Expected int
	// Further expectations beyond status code, typically surrounding resources
	Checks []func() bool
}

func (tc *httpTestCase) Handle(t *testing.T, client *APIClient) {
	t.Helper()
	tc.Path = client.Request(tc.RequestFunc()).URL.Path
	assert.Equal(t, tc.Expected, client.W.Code)
	for key, val := range tc.SaveFields {
		client.SaveResourceFromJSON(key, val)
	}
	for _, check := range tc.Checks {
		assert.True(t, check())
	}
}

func (tc *httpTestCase) getName() string {
	if tc.Name != "" {
		return tc.Name
	}
	return tc.Path
}

// ---------------
// TESTING
// ---------------

// Should properly make, count, and delete users
func Test_MakeAndResetUsers(t *testing.T) {
	// SERVER SETUP
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{Handler: SetupMux(cfg)}
	c := APIClient{Mux: pincher.Handler, Resources: map[string]any{}}

	// REQUESTS

	cases := []httpTestCase{
		// Delete all users in the database
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteAllUsers()
			},
			Expected: http.StatusOK,
		},
		// Create two new users
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser("user1", "pwd1")
			},
			Expected: http.StatusCreated,
		},
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser("user2", "pwd2")
			},
			Expected: http.StatusCreated,
		},
		// User count should now be 2
		{
			RequestFunc: func() *http.Request {
				return pt.GetUserCount()
			},
			SaveFields: map[string]string{
				"count": "count",
			},
			Expected: http.StatusOK,
			Checks: []func() bool{
				c.equalsResourceAt(int64(2), "count"),
			},
		},
		// Delete all users again
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteAllUsers()
			},
			Expected: http.StatusOK,
		},
		// User count should now be 0 again
		{
			RequestFunc: func() *http.Request {
				return pt.GetUserCount()
			},
			SaveFields: map[string]string{
				"count": "count",
			},
			Expected: http.StatusOK,
			Checks: []func() bool{
				(c.equalsResourceAt(int64(0), "count")),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.getName(), func(t *testing.T) {
			tc.Handle(t, &c)
		})
	}
}

// Should make and log in 2 users, which should be able to then delete themselves,
// but not each other
func Test_MakeLoginDeleteUsers(t *testing.T) {
	// SERVER SETUP
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{Handler: SetupMux(cfg)}
	c := APIClient{Mux: pincher.Handler, Resources: map[string]any{}}

	// REQUESTS

	cases := []httpTestCase{
		// Delete all users in the database
		{
			RequestFunc: func() *http.Request { return pt.DeleteAllUsers() },
			Expected:    http.StatusOK,
		},
		// Create two new users
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser("user1", "pwd1")
			},
			Expected: http.StatusCreated,
		},
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser("user2", "pwd2")
			},
			Expected: http.StatusCreated,
		},
		// Log in both users
		{
			Name: "User1Login",
			RequestFunc: func() *http.Request {
				return pt.LoginUser("user1", "pwd1")
			},
			SaveFields: map[string]string{
				"token": "jwt1",
			},
			Expected: http.StatusOK,
		},
		{
			Name: "User2Login",
			RequestFunc: func() *http.Request {
				return pt.LoginUser("user2", "pwd2")
			},
			SaveFields: map[string]string{
				"token": "jwt2",
			},
			Expected: http.StatusOK,
		},
		// attempt deletion of user 2 as user 1; should fail
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteUser(c.GetResource("jwt1").(string), "user2", "pwd2")
			},
			Expected: http.StatusUnauthorized,
		},
		// Attempt deletion of user 1 as user 1
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteUser(c.GetResource("jwt1").(string), "user1", "pwd1")
			},
			Expected: http.StatusOK,
		},
		// User count should now be 1
		{
			RequestFunc: func() *http.Request {
				return pt.GetUserCount()
			},
			SaveFields: map[string]string{
				"count": "count",
			},
			Expected: http.StatusOK,
			Checks: []func() bool{
				c.equalsResourceAt(int64(1), "count"),
			},
		},
		// Delete all users
		{
			RequestFunc: func() *http.Request { return pt.DeleteAllUsers() },
			Expected:    http.StatusOK,
		},
		// User count should now be 0
		{
			RequestFunc: func() *http.Request {
				return pt.GetUserCount()
			},
			SaveFields: map[string]string{
				"count": "count",
			},
			Expected: http.StatusOK,
			Checks: []func() bool{
				(c.equalsResourceAt(int64(0), "count")),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.getName(), func(t *testing.T) {
			tc.Handle(t, &c)
		})
	}
}

/*
Creates two budgets with an admin user, and adds other users to the first.
Along the way, various roles attempt various actions that
they should or should not be able to do; authorizations that
should be verified.
*/
func Test_BuildOrgDoAuthChecks(t *testing.T) {
	/// SERVER SETUP
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{Handler: SetupMux(cfg)}
	c := APIClient{Mux: pincher.Handler}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers())
	assert.Equal(t, 200, c.W.Code)

	// Create four users
	c.Request(pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	c.Request(pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	user2, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateUser("user3", "pwd3"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	user3, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateUser("user4", "pwd4"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	user4, _ := GetJSONField(c.W, "id")

	// Log in four users
	c.Request(pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(c.W, "token")
	c.Request(pt.LoginUser("user2", "pwd2"))
	jwt2, _ := GetJSONField(c.W, "token")
	c.Request(pt.LoginUser("user3", "pwd3"))
	c.Request(pt.LoginUser("user4", "pwd4"))
	jwt4, _ := GetJSONField(c.W, "token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	c.Request(pt.CreateBudget(jwt1.(string), "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."))
	budget1, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateBudget(jwt1.(string), "Personal", "user1's budget for personal finance."))

	// Try adding user2 as ADMIN using user4 (not in budget), then as MANAGER. Both should fail.
	c.Request(pt.AssignMemberToBudget(
		jwt4.(string),
		budget1.(string),
		user2.(string),
		"ADMIN"))
	c.Request(pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "MANAGER"))

	// Add user4 to Webflyx Org as user1 ADMIN, with role: VIEWER
	c.Request(pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user4.(string), "VIEWER"))

	// user4 should be assigned to only 1 budget
	c.Request(pt.GetUserBudgets(jwt4.(string)))
	var gotBudgets []Budget
	err := json.NewDecoder(c.W.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 1, len(gotBudgets))

	// Try adding user2 as MANAGER using u4. Should fail auth check.
	c.Request(pt.AssignMemberToBudget(jwt4.(string), budget1.(string), user2.(string), "MANAGER"))

	// As user1 ADMIN, add user2 and user3 as MANAGER and CONTRIBUTOR, respectively.
	c.Request(pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user2.(string), "MANAGER"))
	c.Request(pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user3.(string), "CONTRIBUTOR"))

	// Attempt deletion of Webflyx Org budget as user2. Should fail; only admin can do it.
	c.Request(pt.DeleteUserBudget(jwt2.(string), budget1.(string)))
	assert.Equal(t, http.StatusUnauthorized, c.W.Code)

	// Attempt to revoke user3's Webflyx Org membership as user4. Should fail.
	c.Request(pt.RevokeBudgetMembership(jwt4.(string), budget1.(string), user3.(string)))
	// Revoke user3's Webflyx Org membership as user1. Should succeed.
	c.Request(pt.RevokeBudgetMembership(jwt1.(string), budget1.(string), user3.(string)))

	// user1 should be assigned to 2 budgets: Webflyx Org & their personal budget
	c.Request(pt.GetUserBudgets(jwt1.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(c.W.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 2, len(gotBudgets))

	// Attempt deletion of Webflyx Org budget as user1. Should succeed.
	c.Request(pt.DeleteUserBudget(jwt1.(string), budget1.(string)))
	assert.Equal(t, http.StatusNoContent, c.W.Code)

	// user1 should be assigned to only 1 budget now: their personal budget.
	c.Request(pt.GetUserBudgets(jwt1.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(c.W.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 1, len(gotBudgets))

	// user4 should be assigned to NO budgets, now.
	c.Request(pt.GetUserBudgets(jwt4.(string)))
	gotBudgets = []Budget{}
	err = json.NewDecoder(c.W.Body).Decode(&gotBudgets)
	if err != nil {
		t.Fatalf("failed to decode response body as slice of budgets: %v", err)
	}
	assert.Equal(t, 0, len(gotBudgets))
}

// Build a small organizational budget system.
// make four users, each with a unique role,
// and let them each perform authorized actions.
func Test_BuildOrgLogTransaction(t *testing.T) {
	/// SERVER SETUP
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{Handler: SetupMux(cfg)}
	c := APIClient{Mux: pincher.Handler}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers())
	assert.Equal(t, 200, c.W.Code)

	// Create four users
	c.Request(pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	c.Request(pt.CreateUser("user2", "pwd2"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	user2, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateUser("user3", "pwd3"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	user3, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateUser("user4", "pwd4"))
	assert.Equal(t, http.StatusCreated, c.W.Code)
	user4, _ := GetJSONField(c.W, "id")

	// Log in four users
	c.Request(pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(c.W, "token")
	c.Request(pt.LoginUser("user2", "pwd2"))
	jwt2, _ := GetJSONField(c.W, "token")
	c.Request(pt.LoginUser("user3", "pwd3"))
	jwt3, _ := GetJSONField(c.W, "token")
	c.Request(pt.LoginUser("user4", "pwd4"))
	jwt4, _ := GetJSONField(c.W, "token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	c.Request(pt.CreateBudget(jwt1.(string), "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."))
	budget1, _ := GetJSONField(c.W, "id")
	c.Request(pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user2.(string), "MANAGER"))
	c.Request(pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user3.(string), "CONTRIBUTOR"))
	c.Request(pt.AssignMemberToBudget(jwt1.(string), budget1.(string), user4.(string), "VIEWER"))

	// user2 MANAGER: Adding account, groups, & categories.
	c.Request(pt.CreateBudgetAccount(jwt2.(string), budget1.(string), "savings", "Saved Org Funds", "Represents a bank account holding business capital."))
	// account1, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateBudgetAccount(jwt2.(string), budget1.(string), "credit", "Employee Business Credit Account", "Employees use cards that pull from this account to pay for business expenses."))
	account2, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateGroup(jwt2.(string), budget1.(string), "Business Capital", "Categories related to company capital"))
	group1, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateCategory(jwt2.(string), budget1.(string), group1.(string), "Surplus", "Category representing surplus funding to be spent on elective improvements to organization headquarters or employee bonuses."))
	category1, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateCategory(jwt2.(string), budget1.(string), group1.(string), "Expenses", "Category representing funds to be used for employee expenses while on the job."))
	category2, _ := GetJSONField(c.W, "id")

	// user3 CONTRIBUTOR: Adding transactions (EX: gas station).
	c.Request(pt.CreateBudgetPayee(jwt3.(string), budget1.(string), "Smash & Dash", "A gas & convenience store"))
	payee1, _ := GetJSONField(c.W, "id")

	transaction1Amounts := map[string]int64{}
	transaction1Amounts[category2.(string)] = 1800
	c.Request(pt.LogTransaction(jwt3.(string), budget1.(string), account2.(string), "NONE", "WITHDRAWAL", "2025-09-15T23:17:00Z", payee1.(string), "I filled up vehicle w/ plate no. 555-555 @ the Smash & Pass gas station.", "true", transaction1Amounts))
	//transaction1, _ := GetJSONField(c.W, "id")

	transaction2Amounts := map[string]int64{}
	transaction2Amounts[category1.(string)] = -400
	c.Request(pt.LogTransaction(jwt3.(string), budget1.(string), account2.(string), "NONE", "WITHDRAWAL", "2025-09-15T23:22:00Z", payee1.(string), "Yeah, I got a drink in the convenience store too; sue me. Take it out of my bonus or whatever.", "true", transaction2Amounts))
	// transaction2, _ := GetJSONField(c.W, "id")

	// user4 VIEWER: Works for accounting; reading transactions from employees.
	c.Request(pt.GetTransactions(jwt4.(string), budget1.(string), account2.(string), "", "", "", ""))
	assert.Equal(t, http.StatusOK, c.W.Code)
	c.Request(pt.GetTransactions(jwt4.(string), budget1.(string), "", "", payee1.(string), "", ""))
	assert.Equal(t, http.StatusOK, c.W.Code)

	// Delete all users
	c.Request(pt.DeleteAllUsers())
	assert.Equal(t, 200, c.W.Code)
}

// Build a budget and give it a predictable amount of money to operate with between 1-2 accounts.
// Log transactions of each type, and check that the endpoint for getting budget capital responds with the right amount(s).
func Test_TransactionTypesAndCapital(t *testing.T) {
	/// SERVER SETUP
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{Handler: SetupMux(cfg)}
	c := APIClient{Mux: pincher.Handler}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers())
	assert.Equal(t, 200, c.W.Code)

	// Create user
	c.Request(pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, c.W.Code)

	// Log in user
	c.Request(pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(c.W, "token")

	// user2 ADMIN: Creating personal budget, making accounts and deposit transactions.
	c.Request(pt.CreateBudget(jwt1.(string), "Personal Budget", "For personal accounting (user1)."))
	budget1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateBudgetAccount(jwt1.(string), budget1.(string), "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."))
	account1, _ := GetJSONField(c.W, "id")
	c.Request(pt.CreateBudgetAccount(jwt1.(string), budget1.(string), "credit", "Credit (Big Banking Inc)", "Reflects my credit account opened via Big Banking, Inc."))
	account2, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateGroup(jwt1.(string), budget1.(string), "Spending", "Categories related to day-to-day spending"))
	group1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateCategory(jwt1.(string), budget1.(string), group1.(string), "Dining Out", "Money for ordering takeout or dining in."))
	category1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Webflyx Org", "user1 employer"))
	payee1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."))
	payee2, _ := GetJSONField(c.W, "id")

	// deposit some money into checking account, allocated (but not explicitly assigned) to the DINING OUT category
	depositAmount := map[string]int64{}
	depositAmount[category1.(string)] = 10000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "DEPOSIT", "2025-09-15T17:00:00Z", payee1.(string), "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", "true", depositAmount))

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), account1.(string)))
	budgetCheckingCapital, _ := GetJSONField(c.W, "capital")
	assert.Equal(t, int64(10000), budgetCheckingCapital)

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), account2.(string)))
	budgetCreditCapital, _ := GetJSONField(c.W, "capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ := GetJSONField(c.W, "capital")
	assert.Equal(t, int64(10000), budgetTotalCapital)

	// spend money out of a credit account
	spendAmount := map[string]int64{}
	spendAmount[category1.(string)] = 5000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account2.(string), "NONE", "WITHDRAWAL", "2025-09-15T18:00:00Z", payee2.(string), "$50 dinner at a restaurant", "true", spendAmount))

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), account2.(string)))
	budgetCreditCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(-5000), budgetCreditCapital)

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// pay off credit account, using the checking account, using a transfer transaction
	transferAmount := map[string]int64{}
	transferAmount["TRANSFER AMOUNT"] = 5000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), account2.(string), "TRANSFER_FROM", "2025-09-15T19:00:00Z", "ACCOUNT TRANSFER", "Pay off credit account balance", "true", transferAmount))

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), account2.(string)))
	budgetCreditCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), account1.(string)))
	budgetCheckingCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(5000), budgetCheckingCapital)

	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// Delete all users
	c.Request(pt.DeleteAllUsers())
	assert.Equal(t, 200, c.W.Code)
}

// Build a budget and simulate 3 months of transactions and dollar assignment.
// Then, ensure that:
//  1. Deposit transactions with non-null categories contribute to its balance by virtue of merely being counted as activity.
//  2. Assignments are agnostic of whether or not there is an equal amount of money between the accounts they represent.
//  3. For each month, we get the assignment, activity, and balance totals we would expect from the actions recorded within the budget.
func Test_CategoryMoneyAssignment(t *testing.T) {
	// SERVER SETUP
	cfg := LoadEnvConfig("../../.env")
	pincher := &http.Server{Handler: SetupMux(cfg)}
	c := APIClient{Mux: pincher.Handler}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers())
	assert.Equal(t, 200, c.W.Code)

	// Create user
	c.Request(pt.CreateUser("user1", "pwd1"))
	assert.Equal(t, http.StatusCreated, c.W.Code)

	// Log in user
	c.Request(pt.LoginUser("user1", "pwd1"))
	jwt1, _ := GetJSONField(c.W, "token")

	// user2 ADMIN: Creating personal budget, making account and other resources.
	c.Request(pt.CreateBudget(jwt1.(string), "Personal Budget", "For personal accounting (user1)."))
	budget1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateBudgetAccount(jwt1.(string), budget1.(string), "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."))
	account1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateGroup(jwt1.(string), budget1.(string), "Ungrouped", "All categories"))
	group1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateCategory(jwt1.(string), budget1.(string), group1.(string), "Dining Out", "Money for ordering takeout or dining in."))
	category1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateCategory(jwt1.(string), budget1.(string), group1.(string), "Savings", "My savings fund."))
	category2, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Webflyx Org", "user1 employer"))
	payee1, _ := GetJSONField(c.W, "id")

	c.Request(pt.CreateBudgetPayee(jwt1.(string), budget1.(string), "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."))
	payee2, _ := GetJSONField(c.W, "id")

	// SEPTEMBER 2025 ACTIVITIES
	// deposit some money into the checking account, allocated (but not explicitly assigned) to the Dining Out category
	depositAmount := map[string]int64{}
	depositAmount[category1.(string)] = 10000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "DEPOSIT", "2025-09-15T17:00:00Z", payee1.(string), "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", "true", depositAmount))

	// spend money out of Dining Out category
	spendAmount := map[string]int64{}
	spendAmount[category1.(string)] = 5000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "WITHDRAWAL", "2025-09-15T18:00:00Z", payee2.(string), "$50 dinner at a restaurant", "true", spendAmount))

	// we expect that there's 5000 in capital remaining, and NO assignable money.
	// 5000 in Dining Out; 0 in Savings.
	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ := GetJSONField(c.W, "capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	c.Request(pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-09-01", category1.(string)))
	balanceCategory1, _ := GetJSONField(c.W, "balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	c.Request(pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-09-01", category2.(string)))
	balanceCategory2, _ := GetJSONField(c.W, "balance")
	assert.Equal(t, int64(0), balanceCategory2)

	// OCTOBER 2025 ACTIVITIES
	// deposit more money into the checking account, with NO category allocation.
	// Assign some (but not all of it, to test for underassignment) to each of two categories.
	clear(depositAmount)
	depositAmount["UNCATEGORIZED"] = 10000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "DEPOSIT", "2025-10-15T17:00:00Z", payee1.(string), "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", "true", depositAmount))

	c.Request(pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-10-01", category1.(string), 4000))
	c.Request(pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-10-01", category2.(string), 5000))

	// spend money out of Dining Out category
	clear(spendAmount)
	spendAmount[category1.(string)] = 4000
	c.Request(pt.LogTransaction(jwt1.(string), budget1.(string), account1.(string), "NONE", "WITHDRAWAL", "2025-10-15T18:00:00Z", payee2.(string), "I was very busy having fun, fun, fun!", "true", spendAmount))

	// we expect that there's 11000 in capital remaining, and 1000 in assignable money.
	// 5000 in Dining Out; 5000 in Savings.
	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-10-01", category1.(string)))
	balanceCategory1, _ = GetJSONField(c.W, "balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	c.Request(pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-10-01", category2.(string)))
	balanceCategory2, _ = GetJSONField(c.W, "balance")
	assert.Equal(t, int64(5000), balanceCategory2)

	c.Request(pt.GetMonthReport(jwt1.(string), budget1.(string), "2025-10-01"))
	budgetTotalBalance, _ := GetJSONField(c.W, "balance")
	assert.Equal(t, int64(1000), (budgetTotalCapital.(int64) - budgetTotalBalance.(int64)))

	// NOVEMBER 2025 ACTIVITIES
	// deposit NO more money into the checking account.
	// Assign the 1000 left available from OCTOBER to the SAVINGS category.
	// Assign 1000 (that we don't have) to DINING OUT to test for overassignment.

	c.Request(pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-11-01", category2.(string), 1000))
	c.Request(pt.AssignMoneyToCategory(jwt1.(string), budget1.(string), "2025-11-01", category1.(string), 1000))

	// we expect that there's still just 11000 in capital remaining, and -1000 in assignable money, which indicates overassignment.
	// 6000 in Dining Out; 6000 in Savings.
	c.Request(pt.GetBudgetCapital(jwt1.(string), budget1.(string), ""))
	budgetTotalCapital, _ = GetJSONField(c.W, "capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-11-01", category1.(string)))
	balanceCategory1, _ = GetJSONField(c.W, "balance")
	assert.Equal(t, int64(6000), balanceCategory1)

	c.Request(pt.GetMonthCategoryReport(jwt1.(string), budget1.(string), "2025-11-01", category2.(string)))
	balanceCategory2, _ = GetJSONField(c.W, "balance")
	assert.Equal(t, int64(6000), balanceCategory2)

	c.Request(pt.GetMonthReport(jwt1.(string), budget1.(string), "2025-11-01"))
	budgetTotalBalance, _ = GetJSONField(c.W, "balance")
	assert.Equal(t, int64(-1000), (budgetTotalCapital.(int64) - budgetTotalBalance.(int64)))

	// Delete all users
	//c.Request(pt.DeleteAllUsers())
	//assert.Equal(t, 200, c.W.Code)
}
