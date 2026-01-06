package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	pt "github.com/YouWantToPinch/pincher-api/internal/pinchertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: This integration testing has two optional implementations.
// Firstly, the conventional one; a more stateful approach recording variables for use in further requests,
// made with the httptest package.
// But secondly, in an effort to make the tests more readable, a table-driven approach was implemented.
// Far too late, it appeared that this second implementation may only make things more readable for those
// 	tests which don't demand very detailed requests, and may otherwise be an overengineered solution to
// 	running others that demand resources to be evaluated at runtime rather than compile time.
// Both implementations are left here for developer use.
// When in doubt: loop through a slice of httpTestCase structs for lighter tests,
// 	but use the more traditional, stateful approach for anything else.

const (
	roleAdmin       = "ADMIN"
	roleManager     = "MANAGER"
	roleContributor = "CONTRIBUTOR"
	roleViewer      = "VIEWER"

	username1 = "user1"
	username2 = "user2"
	username3 = "user3"
	username4 = "user4"

	password1 = "pwd1"
	password2 = "pwd2"
	password3 = "pwd3"
	password4 = "pwd4"

	dateSeptember = "2025-09-15"
	dateOctober   = "2025-10-15"
	dateNovember  = "2025-11-15"
)

// ---------------
// HELPER FUNCS
// ---------------

type APITestClient struct {
	Mux       http.Handler
	W         *httptest.ResponseRecorder
	Resources map[string]any
	testState *testing.T
}

func (c *APITestClient) GetJSONField(field string) (any, error) {
	res := c.W.Result()
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

func (c *APITestClient) GetJSONFieldAsString(field string) (string, error) {
	fieldRetrieved, err := c.GetJSONField(field)
	if err != nil {
		return "", err
	}
	if val, ok := fieldRetrieved.(string); ok {
		return val, nil
	}
	return "", fmt.Errorf("field retrieved from response was not of type string")
}

func (c *APITestClient) GetJSONFieldAsInt64(field string) (int64, error) {
	fieldRetrieved, err := c.GetJSONField(field)
	if err != nil {
		return 0, err
	}
	if val, ok := fieldRetrieved.(int64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("field retrieved from response was not of type int64")
}

// Request records a new request, saves the response to a new recorder for reference,
// and calls an assert check against the response status code before then returning the request.
func (c *APITestClient) Request(req *http.Request, expectedCode int) *http.Request {
	w := httptest.NewRecorder()
	c.Mux.ServeHTTP(w, req)
	c.W = w
	if expectedCode != 0 {
		assert.Equal(c.testState, expectedCode, c.W.Code)
	}
	return req
}

func (c *APITestClient) GetResource(name string) any {
	if v, ok := c.Resources[name]; ok {
		return v
	}
	return nil
}

func (c *APITestClient) SaveResourceFromJSON(field string, name string) {
	jsonObject, _ := c.GetJSONField(field)
	c.Resources[name] = jsonObject
	slog.Debug(fmt.Sprintf("Saved resource %s at: %v (type: %T)", name, c.Resources[name], c.Resources[name]))
}

func (c *APITestClient) equalsResourceAt(expected any, resourceName string) func() bool {
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

func (tc *httpTestCase) Handle(t *testing.T, client *APITestClient) {
	t.Helper()
	client.testState = t
	tc.Path = client.Request(tc.RequestFunc(), tc.Expected).URL.Path
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

// --------------------
// INTEGRATION TESTING
// --------------------

// initialize a Postgres testcontainer and
// return an APITestClient for testing
func doServerSetup(t *testing.T) *http.Server {
	pgdb := SetupPostgres(t)
	t.Cleanup(func() {
		err := pgdb.Container.Restore(pgdb.Ctx)
		require.NoError(t, err)
	})
	cfg := &APIConfig{}
	cfg.Init("../../.env", pgdb.URI)
	cfg.ConnectToDB(embed.FS{}, "")
	return &http.Server{Handler: SetupMux(cfg)}
}

// Check CRUD for each resource exclusively, in order
func Test_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := &APITestClient{Mux: pincherServer.Handler, testState: t}

	// PREP: Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)

	// REQUESTS

	t.Run("Users", func(t *testing.T) {
		// CREATE user, log in
		c.Request(pt.CreateUser(username1, password1), http.StatusCreated)
		c.Request(pt.LoginUser(username1, password1), http.StatusOK)
		jwt1, _ := c.GetJSONFieldAsString("token")
		// UPDATE user
		c.Request(pt.UpdateUser(jwt1, "newUsername", "verySecure123456"), http.StatusNoContent)
		// DELETE user
		c.Request(pt.DeleteUser(jwt1, "newUsername", "verySecure123456"), http.StatusNoContent)
	})

	// PREP: Create user, log in
	c.Request(pt.CreateUser(username1, password1), 0)
	c.Request(pt.LoginUser(username1, password1), 0)
	jwt1, _ := c.GetJSONFieldAsString("token")

	t.Run("Budgets", func(t *testing.T) {
		// CREATE budget
		c.Request(pt.CreateBudget(jwt1, "userBudget", "A new budget for the test user."), http.StatusCreated)
		budgetID1, _ := c.GetJSONFieldAsString("id")
		// READ budget(s)
		c.Request(pt.GetUserBudgets(jwt1), http.StatusOK)
		// UPDATE budgets
		c.Request(pt.UpdateBudget(jwt1, budgetID1, "User Budget", "Test user's budget, now updated."), http.StatusNoContent)
		// DELETE budget
		c.Request(pt.DeleteUserBudget(jwt1, budgetID1), http.StatusNoContent)
	})

	// PREP: Create budget
	c.Request(pt.CreateBudget(jwt1, "userBudget", "A new budget for the test user."), 0)
	budgetID1, _ := c.GetJSONFieldAsString("id")

	t.Run("Groups", func(t *testing.T) {
		// CREATE group
		c.Request(pt.CreateGroup(jwt1, budgetID1, "testGroup", "A group for testing."), http.StatusCreated)
		groupID1, _ := c.GetJSONFieldAsString("id")
		// READ group(s)
		c.Request(pt.GetBudgetGroups(jwt1, budgetID1), http.StatusOK)
		// UPDATE budgets
		c.Request(pt.UpdateGroup(jwt1, budgetID1, groupID1, "Test Group", "Test user's first group, now updated."), http.StatusNoContent)
		// DELETE budget
		c.Request(pt.DeleteBudgetGroup(jwt1, budgetID1, groupID1), http.StatusNoContent)
	})

	// PREP: Create group
	c.Request(pt.CreateGroup(jwt1, budgetID1, "testGroup", "A group for testing."), http.StatusCreated)
	groupID1, _ := c.GetJSONFieldAsString("id")

	t.Run("Categories", func(t *testing.T) {
		// PREP: Create another group
		c.Request(pt.CreateGroup(jwt1, budgetID1, "tempTestGroup", "A group for testing updates."), 0)

		// CREATE category
		c.Request(pt.CreateCategory(jwt1, budgetID1, "testGroup", "testCategory", "A category for testing."), http.StatusCreated)
		categoryID1, _ := c.GetJSONFieldAsString("id")
		// READ categories(s)
		c.Request(pt.GetBudgetCategories(jwt1, budgetID1, "?group_id="+groupID1), http.StatusOK)
		// UPDATE category
		c.Request(pt.UpdateCategory(jwt1, budgetID1, categoryID1, "tempTestGroup", "Test Group", "Test user's first category, now updated to a new group."), http.StatusNoContent)
		// DELETE category
		c.Request(pt.DeleteBudgetCategory(jwt1, budgetID1, categoryID1), http.StatusNoContent)
	})

	// PREP: Create category
	c.Request(pt.CreateCategory(jwt1, budgetID1, "testGroup", "testCategory", "A category for testing."), http.StatusCreated)
	categoryID1, _ := c.GetJSONFieldAsString("id")

	t.Run("Accounts", func(t *testing.T) {
		// CREATE account
		c.Request(pt.CreateBudgetAccount(jwt1, budgetID1, "CHECKING", "testAccount", "An account for testing."), http.StatusCreated)
		accountID1, _ := c.GetJSONFieldAsString("id")
		// READ account(s)
		c.Request(pt.GetBudgetAccounts(jwt1, budgetID1), http.StatusOK)
		// UPDATE account
		// NOTE: As of this commit, Account Types mean nothing, so updates to an account
		// 	may include changing them. This may need to be prohibited when Account Types
		// 	logic is implemented.
		c.Request(pt.UpdateAccount(jwt1, budgetID1, accountID1, "CREDIT", "Test Account", "Test user's first account, now updated."), http.StatusNoContent)
		// DELETE account SOFT
		c.Request(pt.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "Test Account", false), http.StatusOK)
		// DELETE account HARD
		c.Request(pt.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "Test Account", true), http.StatusNoContent)
	})

	// CREATE account
	c.Request(pt.CreateBudgetAccount(jwt1, budgetID1, "CREDIT", "testAccount", "An account for testing."), http.StatusCreated)

	t.Run("Payees", func(t *testing.T) {
		// CREATE payee
		c.Request(pt.CreateBudgetPayee(jwt1, budgetID1, "testPayee", "A payee for testing."), http.StatusCreated)
		payeeID1, _ := c.GetJSONFieldAsString("id")
		// READ payee(s)
		c.Request(pt.GetBudgetPayees(jwt1, budgetID1), http.StatusOK)
		// UPDATE payee
		c.Request(pt.UpdatePayee(jwt1, budgetID1, payeeID1, "Test Payee", "Test user's first account, now updated."), http.StatusNoContent)
		// DELETE payee
		c.Request(pt.DeletePayee(jwt1, budgetID1, payeeID1, ""), http.StatusNoContent)
	})

	// PREP: Create payee
	c.Request(pt.CreateBudgetPayee(jwt1, budgetID1, "testPayee", "A payee for testing."), http.StatusCreated)

	t.Run("Transactions", func(t *testing.T) {
		// PREP: Create payee
		c.Request(pt.CreateBudgetPayee(jwt1, budgetID1, "testPayee2", "A payee for testing payee reassignment & deletion."), http.StatusCreated)
		payeeID2, _ := c.GetJSONFieldAsString("id")

		// CREATE transaction
		transactionAmounts := map[string]int64{}
		transactionAmounts[categoryID1] = -500

		c.Request(pt.LogTransaction(jwt1, budgetID1, "testAccount", "", dateSeptember, "testPayee2", "A transaction for testing.", true, map[string]int64{categoryID1: -500}), http.StatusCreated)
		transactionID1, _ := c.GetJSONFieldAsString("id")
		// READ transaction
		c.Request(pt.GetTransaction(jwt1, budgetID1, transactionID1), http.StatusOK)
		// UPDATE transaction
		c.Request(pt.LogTransaction(jwt1, budgetID1, "testAccount", "", dateSeptember, "testPayee2", "A transaction whose notes are now updated.", true, map[string]int64{categoryID1: -500}), http.StatusCreated)
		// DELETE payee
		c.Request(pt.DeletePayee(jwt1, budgetID1, payeeID2, "testPayee"), http.StatusNoContent)
		// DELETE transaction
		c.Request(pt.DeleteTransaction(jwt1, budgetID1, transactionID1), http.StatusNoContent)
	})
}

// Should properly make, count, and delete users
func Test_MakeAndResetUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, Resources: map[string]any{}}

	// REQUESTS

	cases := []httpTestCase{
		// Delete all users in the database
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteAllUsers()
			},
			Expected: http.StatusNoContent,
		},
		// Create two new users
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser(username1, password1)
			},
			Expected: http.StatusCreated,
		},
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser(username2, password2)
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
			Expected: http.StatusNoContent,
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, Resources: map[string]any{}}

	// REQUESTS

	cases := []httpTestCase{
		// Delete all users in the database
		{
			RequestFunc: func() *http.Request { return pt.DeleteAllUsers() },
			Expected:    http.StatusNoContent,
		},
		// Create two new users
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser(username1, password1)
			},
			Expected: http.StatusCreated,
		},
		{
			RequestFunc: func() *http.Request {
				return pt.CreateUser(username2, password2)
			},
			Expected: http.StatusCreated,
		},
		// Log in both users
		{
			Name: "User1Login",
			RequestFunc: func() *http.Request {
				return pt.LoginUser(username1, password1)
			},
			SaveFields: map[string]string{
				"token": "jwt1",
			},
			Expected: http.StatusOK,
		},
		{
			Name: "User2Login",
			RequestFunc: func() *http.Request {
				return pt.LoginUser(username2, password2)
			},
			SaveFields: map[string]string{
				"token": "jwt2",
			},
			Expected: http.StatusOK,
		},
		// attempt deletion of user 2 as user 1; should fail
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteUser(c.GetResource("jwt1").(string), username2, password2)
			},
			Expected: http.StatusForbidden,
		},
		// Attempt deletion of user 1 as user 1
		{
			RequestFunc: func() *http.Request {
				return pt.DeleteUser(c.GetResource("jwt1").(string), username1, password1)
			},
			Expected: http.StatusNoContent,
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
			Expected:    http.StatusNoContent,
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
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)

	// Create four users
	c.Request(pt.CreateUser(username1, password1), http.StatusCreated)
	c.Request(pt.CreateUser(username2, password2), http.StatusCreated)
	c.Request(pt.CreateUser(username3, password3), http.StatusCreated)
	user3ID, _ := c.GetJSONFieldAsString("id")
	c.Request(pt.CreateUser(username4, password4), http.StatusCreated)

	// Log in four users
	c.Request(pt.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")
	c.Request(pt.LoginUser(username2, password2), http.StatusOK)
	jwt2, _ := c.GetJSONFieldAsString("token")
	c.Request(pt.LoginUser(username3, password3), http.StatusOK)
	c.Request(pt.LoginUser(username4, password4), http.StatusOK)
	jwt4, _ := c.GetJSONFieldAsString("token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	c.Request(pt.CreateBudget(jwt1, "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")
	c.Request(pt.CreateBudget(jwt1, "Personal", "user1's budget for personal finance."), http.StatusCreated)

	// Try adding user2 as ADMIN using user4 (not in budget), then as MANAGER. Both should fail.
	c.Request(pt.AssignMemberToBudget(jwt4, budget1, username2, roleAdmin), http.StatusForbidden)
	c.Request(pt.AssignMemberToBudget(jwt4, budget1, username2, roleManager), http.StatusForbidden)

	// Add user4 to Webflyx Org as user1 ADMIN, with role: VIEWER
	c.Request(pt.AssignMemberToBudget(jwt1, budget1, username4, roleViewer), http.StatusCreated)

	// user4 should be assigned to only 1 budget
	c.Request(pt.GetUserBudgets(jwt4), http.StatusOK)
	gotBudgets, _ := c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 1)

	// As user4, try adding user2 as a MANAGER. Should fail auth check.
	c.Request(pt.AssignMemberToBudget(jwt4, budget1, username2, roleManager), http.StatusForbidden)

	// As user1 ADMIN, add user2 and user3 as MANAGER and CONTRIBUTOR, respectively.
	c.Request(pt.AssignMemberToBudget(jwt1, budget1, username2, roleManager), http.StatusCreated)
	c.Request(pt.AssignMemberToBudget(jwt1, budget1, username3, roleContributor), http.StatusCreated)

	// Attempt deletion of Webflyx Org budget as user2. Should fail; only admin can do it.
	c.Request(pt.DeleteUserBudget(jwt2, budget1), http.StatusForbidden)

	// Attempt to revoke user3's Webflyx Org membership as user4. Should fail.
	c.Request(pt.RevokeBudgetMembership(jwt4, budget1, user3ID), http.StatusForbidden)
	// Revoke user3's Webflyx Org membership as user1. Should succeed.
	c.Request(pt.RevokeBudgetMembership(jwt1, budget1, user3ID), http.StatusNoContent)

	// user1 should be assigned to 2 budgets: Webflyx Org & their personal budget
	c.Request(pt.GetUserBudgets(jwt1), http.StatusOK)
	gotBudgets, _ = c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 2)

	// Attempt deletion of Webflyx Org budget as user1. Should succeed.
	c.Request(pt.DeleteUserBudget(jwt1, budget1), http.StatusNoContent)

	// user1 should be assigned to only 1 budget now: their personal budget.
	c.Request(pt.GetUserBudgets(jwt1), http.StatusOK)
	gotBudgets, _ = c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 1)

	// user4 should be assigned to NO budgets, now.
	c.Request(pt.GetUserBudgets(jwt4), http.StatusOK)
	gotBudgets, _ = c.GetJSONFieldAsString("budgets")
	assert.Empty(t, gotBudgets)
}

// Build a small organizational budget system.
// make four users, each with a unique role,
// and let them each perform authorized actions.
func Test_BuildOrgLogTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)

	// Create four users
	c.Request(pt.CreateUser(username1, password1), http.StatusCreated)
	c.Request(pt.CreateUser(username2, password2), http.StatusCreated)
	c.Request(pt.CreateUser(username3, password3), http.StatusCreated)
	c.Request(pt.CreateUser(username4, password4), http.StatusCreated)

	// Log in four users
	c.Request(pt.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")
	c.Request(pt.LoginUser(username2, password2), http.StatusOK)
	jwt2, _ := c.GetJSONFieldAsString("token")
	c.Request(pt.LoginUser(username3, password3), http.StatusOK)
	jwt3, _ := c.GetJSONFieldAsString("token")
	c.Request(pt.LoginUser(username4, password4), http.StatusOK)
	jwt4, _ := c.GetJSONFieldAsString("token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	c.Request(pt.CreateBudget(jwt1, "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."), 0)
	budget1, _ := c.GetJSONFieldAsString("id")
	c.Request(pt.AssignMemberToBudget(jwt1, budget1, username2, roleManager), http.StatusCreated)
	c.Request(pt.AssignMemberToBudget(jwt1, budget1, username3, roleContributor), http.StatusCreated)
	c.Request(pt.AssignMemberToBudget(jwt1, budget1, username4, roleViewer), http.StatusCreated)

	// user2 MANAGER: Adding account, groups, & categories.
	c.Request(pt.CreateBudgetAccount(jwt2, budget1, "savings", "Saved Org Funds", "Represents a bank account holding business capital."), http.StatusCreated)
	c.Request(pt.CreateBudgetAccount(jwt2, budget1, "credit", "Employee Business Credit Account", "Employees use cards that pull from this account to pay for business expenses."), http.StatusCreated)
	account2, _ := c.GetJSONFieldAsString("id")
	c.Request(pt.CreateGroup(jwt2, budget1, "Business Capital", "Categories related to company capital"), http.StatusCreated)
	c.Request(pt.CreateCategory(jwt2, budget1, "Business Capital", "Surplus", "Category representing surplus funding to be spent on elective improvements to organization headquarters or employee bonuses."), http.StatusCreated)
	category1, _ := c.GetJSONFieldAsString("id")
	c.Request(pt.CreateCategory(jwt2, budget1, "Business Capital", "Expenses", "Category representing funds to be used for employee expenses while on the job."), http.StatusCreated)
	category2, _ := c.GetJSONFieldAsString("id")

	// user3 CONTRIBUTOR: Adding transactions (EX: gas station).
	c.Request(pt.CreateBudgetPayee(jwt3, budget1, "Smash & Dash", "A gas & convenience store"), http.StatusCreated)
	payee1ID, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.LogTransaction(jwt3, budget1, "Employee Business Credit Account", "", dateSeptember, "Smash & Dash", "I filled up vehicle w/ plate no. 555-555 @ the Smash & Pass gas station.", true, map[string]int64{category2: -1800}), http.StatusCreated)
	// transaction1, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.LogTransaction(jwt3, budget1, "Employee Business Credit Account", "", dateSeptember, "Smash & Dash", "Yeah, I got a drink in the convenience store too; sue me. Take it out of my bonus or whatever.", true, map[string]int64{category1: -400}), http.StatusCreated)
	// transaction2, _ := c.GetJSONFieldAsString("id")

	// user4 VIEWER: Works for accounting; reading transactions from employees.
	c.Request(pt.GetTransactions(jwt4, budget1, account2, "", "", "", ""), http.StatusOK)
	c.Request(pt.GetTransactions(jwt4, budget1, "", "", payee1ID, "", ""), http.StatusOK)

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)
}

// Build a budget and give it a predictable amount of money to operate with between 1-2 accounts.
// Log transactions of each type, and check that the endpoint for getting budget capital responds with the right amount(s).
func Test_TransactionTypesAndCapital(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)

	// Create user
	c.Request(pt.CreateUser(username1, password1), http.StatusCreated)

	// Log in user
	c.Request(pt.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")

	// user2 ADMIN: Creating personal budget, making accounts and deposit transactions.
	c.Request(pt.CreateBudget(jwt1, "Personal Budget", "For personal accounting (user1)."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.CreateBudgetAccount(jwt1, budget1, "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."), http.StatusCreated)
	account1, _ := c.GetJSONFieldAsString("id")
	c.Request(pt.CreateBudgetAccount(jwt1, budget1, "credit", "Credit (Big Banking Inc)", "Reflects my credit account opened via Big Banking, Inc."), http.StatusCreated)
	account2, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.CreateGroup(jwt1, budget1, "Spending", "Categories related to day-to-day spending"), http.StatusCreated)

	c.Request(pt.CreateCategory(jwt1, budget1, "Spending", "Dining Out", "Money for ordering takeout or dining in."), http.StatusCreated)
	category1, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.CreateBudgetPayee(jwt1, budget1, "Webflyx Org", "user1 employer"), http.StatusCreated)

	c.Request(pt.CreateBudgetPayee(jwt1, budget1, "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."), http.StatusCreated)

	// deposit some money into checking account, allocated (but not explicitly assigned) to the DINING OUT category
	c.Request(pt.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateSeptember, "Webflyx Org", "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", true, map[string]int64{category1: 10000}), http.StatusCreated)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, account1), http.StatusOK)
	budgetCheckingCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(10000), budgetCheckingCapital)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, account2), http.StatusOK)
	budgetCreditCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(10000), budgetTotalCapital)

	// spend money out of a credit account
	c.Request(pt.LogTransaction(jwt1, budget1, "Credit (Big Banking Inc)", "", dateSeptember, "Messy Joe's", "$50 dinner at a restaurant", true, map[string]int64{category1: -5000}), http.StatusCreated)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, account2), http.StatusOK)
	budgetCreditCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(-5000), budgetCreditCapital)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// pay off credit account, using the checking account, using a transfer transaction
	c.Request(pt.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "Credit (Big Banking Inc)", dateSeptember, "ACCOUNT TRANSFER", "Pay off credit account balance", true, map[string]int64{"TRANSFER AMOUNT": -5000}), http.StatusCreated)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, account2), http.StatusOK)
	budgetCreditCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, account1), http.StatusOK)
	budgetCheckingCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetCheckingCapital)

	c.Request(pt.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)
}

// Build a budget and simulate 3 months of transactions and dollar assignment.
// Then, ensure that:
//  1. Deposit transactions with non-null categories contribute to its balance by virtue of merely being counted as activity.
//  2. Assignments are agnostic of whether or not there is an equal amount of money between the accounts they represent.
//  3. For each month, we get the assignment, activity, and balance totals we would expect from the actions recorded within the budget.
func Test_CategoryMoneyAssignment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)

	// Create user
	c.Request(pt.CreateUser(username1, password1), http.StatusCreated)

	// Log in user
	c.Request(pt.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")

	// user1 ADMIN: Creating personal budget, making account and other resources.
	c.Request(pt.CreateBudget(jwt1, "Personal Budget", "For personal accounting (user1)."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.CreateBudgetAccount(jwt1, budget1, "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."), http.StatusCreated)

	// Note: lack of group assignment is purposeful.
	c.Request(pt.CreateCategory(jwt1, budget1, "", "Dining Out", "Money for ordering takeout or dining in."), http.StatusCreated)
	category1, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.CreateCategory(jwt1, budget1, "", "Savings", "My savings fund."), http.StatusCreated)
	category2, _ := c.GetJSONFieldAsString("id")

	c.Request(pt.CreateBudgetPayee(jwt1, budget1, "Webflyx Org", "user1 employer"), http.StatusCreated)
	c.Request(pt.CreateBudgetPayee(jwt1, budget1, "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."), http.StatusCreated)

	// SEPTEMBER 2025 ACTIVITIES
	// deposit some money into the checking account, allocated (but not explicitly assigned) to the Dining Out category
	c.Request(pt.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateSeptember, "Webflyx Org", "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", true, map[string]int64{category1: 10000}), http.StatusCreated)

	// spend money out of Dining Out category
	c.Request(pt.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateSeptember, "Messy Joe's", "$50 dinner at a restaurant", true, map[string]int64{category1: -5000}), http.StatusCreated)

	// we expect that there's 5000 in capital remaining, and NO assignable money.
	// 5000 in Dining Out; 0 in Savings.
	c.Request(pt.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	c.Request(pt.GetMonthCategoryReport(jwt1, budget1, dateSeptember, category1), http.StatusOK)
	balanceCategory1, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	c.Request(pt.GetMonthCategoryReport(jwt1, budget1, dateSeptember, category2), http.StatusOK)
	balanceCategory2, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(0), balanceCategory2)

	// OCTOBER 2025 ACTIVITIES
	// deposit more money into the checking account, with NO category allocation.
	// Assign some (but not all of it, to test for underassignment) to each of two categories.
	c.Request(pt.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateOctober, "Webflyx Org", "$100 deposit into account; no category allocation, we'll assign it manually.", true, map[string]int64{"UNCATEGORIZED": 10000}), http.StatusCreated)

	c.Request(pt.AssignMoneyToCategory(jwt1, budget1, dateOctober, category1, 4000), http.StatusOK)
	c.Request(pt.AssignMoneyToCategory(jwt1, budget1, dateOctober, category2, 5000), http.StatusOK)

	// spend money out of Dining Out category
	c.Request(pt.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateOctober, "Messy Joe's", "I was very busy having fun, fun, fun!", true, map[string]int64{category1: -4000}), http.StatusCreated)

	// we expect that there's 11000 in capital remaining, and 1000 in assignable money.
	// 5000 in Dining Out; 5000 in Savings.
	c.Request(pt.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(pt.GetMonthCategoryReport(jwt1, budget1, dateOctober, category1), http.StatusOK)
	balanceCategory1, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	c.Request(pt.GetMonthCategoryReport(jwt1, budget1, dateOctober, category2), http.StatusOK)
	balanceCategory2, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), balanceCategory2)

	c.Request(pt.GetMonthReport(jwt1, budget1, dateOctober), http.StatusOK)
	budgetTotalBalance, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(1000), (budgetTotalCapital - budgetTotalBalance))

	// NOVEMBER 2025 ACTIVITIES
	// deposit NO more money into the checking account.
	// Assign the 1000 left available from OCTOBER to the SAVINGS category.
	// Assign 1000 (that we don't have) to DINING OUT to test for overassignment.

	c.Request(pt.AssignMoneyToCategory(jwt1, budget1, dateNovember, category2, 1000), http.StatusOK)
	c.Request(pt.AssignMoneyToCategory(jwt1, budget1, dateNovember, category1, 1000), http.StatusOK)

	// we expect that there's still just 11000 in capital remaining, and -1000 in assignable money, which indicates overassignment.
	// 6000 in Dining Out; 6000 in Savings.
	c.Request(pt.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(pt.GetMonthCategoryReport(jwt1, budget1, dateNovember, category1), http.StatusOK)
	balanceCategory1, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(6000), balanceCategory1)

	c.Request(pt.GetMonthCategoryReport(jwt1, budget1, dateNovember, category2), http.StatusOK)
	balanceCategory2, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(6000), balanceCategory2)

	c.Request(pt.GetMonthReport(jwt1, budget1, dateNovember), http.StatusOK)
	budgetTotalBalance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(-1000), (budgetTotalCapital - budgetTotalBalance))

	// Delete all users
	c.Request(pt.DeleteAllUsers(), http.StatusNoContent)
}
