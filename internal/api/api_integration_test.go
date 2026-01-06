package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: The methodology of this integration testing suite is as follows:
// - The APITestClient provides a means of making requests and pulling recorded values from responses.
// - Each test is run within a separate testcontainer.
// - Each test follows a stateful approach.

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
	testState *testing.T
}

// Request records a new request, saves the response to a new recorder for reference,
// and calls an assert check against the response status code.
func (c *APITestClient) Request(req *http.Request, expectedCode int) {
	w := httptest.NewRecorder()
	c.Mux.ServeHTTP(w, req)
	c.W = w
	if expectedCode != 0 {
		assert.Equal(c.testState, expectedCode, c.W.Code)
	}
}

// GetJSONField returns a value from the last response recorded,
// assuming that the response content-type was JSON.
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
	if num, ok := fieldRetrieved.(json.Number); ok {
		if i, err := num.Int64(); err == nil {
			return i, nil
		}
		return -987654321, err
	}

	if val, ok := fieldRetrieved.(int64); ok {
		return val, nil
	}
	return 0, fmt.Errorf("field retrieved from response was not of type int64")
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

// Check CRUD for each resource in as much isolation as possible.
// NOTE:
// Before testing much interaction between resources, the CRUD operations of EACH is tested.
// The ORDER of which resources are tested, of course, conforms to resource dependency.
// For example, budget CRUD ops depend on working user CRUD ops, so users are tested first.
func Test_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := &APITestClient{Mux: pincherServer.Handler, testState: t}

	// PREP: Delete all users
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)

	// REQUESTS

	t.Run("Users", func(t *testing.T) {
		// CREATE user, log in
		c.Request(c.CreateUser(username1, password1), http.StatusCreated)
		c.Request(c.LoginUser(username1, password1), http.StatusOK)
		jwt1, _ := c.GetJSONFieldAsString("token")
		// UPDATE user
		c.Request(c.UpdateUser(jwt1, "newUsername", "verySecure123456"), http.StatusNoContent)
		// DELETE user
		c.Request(c.DeleteUser(jwt1, "newUsername", "verySecure123456"), http.StatusNoContent)
	})

	// PREP: Create user, log in
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)
	c.Request(c.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")

	t.Run("Budgets", func(t *testing.T) {
		// CREATE budget
		c.Request(c.CreateBudget(jwt1, "userBudget", "A new budget for the test user."), http.StatusCreated)
		budgetID1, _ := c.GetJSONFieldAsString("id")
		// READ budget(s)
		c.Request(c.GetUserBudgets(jwt1), http.StatusOK)
		// UPDATE budgets
		c.Request(c.UpdateBudget(jwt1, budgetID1, "User Budget", "Test user's budget, now updated."), http.StatusNoContent)
		// DELETE budget
		c.Request(c.DeleteUserBudget(jwt1, budgetID1), http.StatusNoContent)
	})

	// PREP: Create budget
	c.Request(c.CreateBudget(jwt1, "userBudget", "A new budget for the test user."), http.StatusCreated)
	budgetID1, _ := c.GetJSONFieldAsString("id")

	t.Run("Groups", func(t *testing.T) {
		// CREATE group
		c.Request(c.CreateGroup(jwt1, budgetID1, "testGroup", "A group for testing."), http.StatusCreated)
		groupID1, _ := c.GetJSONFieldAsString("id")
		// READ group(s)
		c.Request(c.GetBudgetGroups(jwt1, budgetID1), http.StatusOK)
		// UPDATE budgets
		c.Request(c.UpdateGroup(jwt1, budgetID1, groupID1, "Test Group", "Test user's first group, now updated."), http.StatusNoContent)
		// DELETE budget
		c.Request(c.DeleteBudgetGroup(jwt1, budgetID1, groupID1), http.StatusNoContent)
	})

	// PREP: Create group
	c.Request(c.CreateGroup(jwt1, budgetID1, "testGroup", "A group for testing."), http.StatusCreated)
	groupID1, _ := c.GetJSONFieldAsString("id")

	t.Run("Categories", func(t *testing.T) {
		// PREP: Create another group
		c.Request(c.CreateGroup(jwt1, budgetID1, "temc.stGroup", "A group for testing updates."), http.StatusCreated)

		// CREATE category
		c.Request(c.CreateCategory(jwt1, budgetID1, "testGroup", "testCategory", "A category for testing."), http.StatusCreated)
		categoryID1, _ := c.GetJSONFieldAsString("id")
		// READ categories(s)
		c.Request(c.GetBudgetCategories(jwt1, budgetID1, "?group_id="+groupID1), http.StatusOK)
		// UPDATE category
		c.Request(c.UpdateCategory(jwt1, budgetID1, categoryID1, "temc.stGroup", "Test Group", "Test user's first category, now updated to a new group."), http.StatusNoContent)
		// DELETE category
		c.Request(c.DeleteBudgetCategory(jwt1, budgetID1, categoryID1), http.StatusNoContent)
	})

	// PREP: Create category
	c.Request(c.CreateCategory(jwt1, budgetID1, "testGroup", "testCategory", "A category for testing."), http.StatusCreated)
	categoryID1, _ := c.GetJSONFieldAsString("id")

	t.Run("Accounts", func(t *testing.T) {
		// CREATE account
		c.Request(c.CreateBudgetAccount(jwt1, budgetID1, "CHECKING", "testAccount", "An account for testing."), http.StatusCreated)
		accountID1, _ := c.GetJSONFieldAsString("id")
		// READ account(s)
		c.Request(c.GetBudgetAccounts(jwt1, budgetID1), http.StatusOK)
		// UPDATE account
		// NOTE: As of this commit, Account Types mean nothing, so updates to an account
		// 	may include changing them. This may need to be prohibited when Account Types
		// 	logic is implemented.
		c.Request(c.UpdateAccount(jwt1, budgetID1, accountID1, "CREDIT", "Test Account", "Test user's first account, now updated."), http.StatusNoContent)
		// DELETE account SOFT
		c.Request(c.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "Test Account", false), http.StatusOK)
		// DELETE account HARD
		c.Request(c.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "Test Account", true), http.StatusNoContent)
	})

	// CREATE account
	c.Request(c.CreateBudgetAccount(jwt1, budgetID1, "CREDIT", "testAccount", "An account for testing."), http.StatusCreated)

	t.Run("Payees", func(t *testing.T) {
		// CREATE payee
		c.Request(c.CreateBudgetPayee(jwt1, budgetID1, "testPayee", "A payee for testing."), http.StatusCreated)
		payeeID1, _ := c.GetJSONFieldAsString("id")
		// READ payee(s)
		c.Request(c.GetBudgetPayees(jwt1, budgetID1), http.StatusOK)
		// UPDATE payee
		c.Request(c.UpdatePayee(jwt1, budgetID1, payeeID1, "Test Payee", "Test user's first account, now updated."), http.StatusNoContent)
		// DELETE payee
		c.Request(c.DeletePayee(jwt1, budgetID1, payeeID1, ""), http.StatusNoContent)
	})

	// PREP: Create payee
	c.Request(c.CreateBudgetPayee(jwt1, budgetID1, "testPayee", "A payee for testing."), http.StatusCreated)

	t.Run("Transactions", func(t *testing.T) {
		// PREP: Create payee
		c.Request(c.CreateBudgetPayee(jwt1, budgetID1, "testPayee2", "A payee for testing payee reassignment & deletion."), http.StatusCreated)
		payeeID2, _ := c.GetJSONFieldAsString("id")

		// CREATE transaction
		c.Request(c.LogTransaction(jwt1, budgetID1, "testAccount", "", dateSeptember, "testPayee2", "A transaction for testing.", true, map[string]int64{categoryID1: -500}), http.StatusCreated)
		transactionID1, _ := c.GetJSONFieldAsString("id")
		// READ transaction
		c.Request(c.GetTransaction(jwt1, budgetID1, transactionID1), http.StatusOK)
		// UPDATE transaction
		c.Request(c.LogTransaction(jwt1, budgetID1, "testAccount", "", dateSeptember, "testPayee2", "A transaction whose notes are now updated.", true, map[string]int64{categoryID1: -500}), http.StatusCreated)
		// DELETE payee
		c.Request(c.DeletePayee(jwt1, budgetID1, payeeID2, "testPayee"), http.StatusNoContent)
		// DELETE transaction
		c.Request(c.DeleteTransaction(jwt1, budgetID1, transactionID1), http.StatusNoContent)
	})
}

// Should properly make, count, and delete users
func Test_MakeAndResetUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)
	c.Request(c.CreateUser(username2, password2), http.StatusCreated)
	c.Request(c.GetUserCount(), http.StatusOK)
	userCount, _ := c.GetJSONFieldAsInt64("count")
	assert.Equal(t, int64(2), userCount)
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
}

// Should make and log in 2 users,
// which should be able to then delete themselves, but not each other
func Test_MakeLoginDeleteUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)
	c.Request(c.CreateUser(username2, password2), http.StatusCreated)
	c.Request(c.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")
	c.Request(c.LoginUser(username2, password2), http.StatusOK)
	jwt2, _ := c.GetJSONFieldAsString("token")
	c.Request(c.DeleteUser(jwt1, username2, password2), http.StatusForbidden)
	c.Request(c.DeleteUser(jwt2, username1, password1), http.StatusForbidden)
	c.Request(c.DeleteUser(jwt1, username1, password1), http.StatusNoContent)
	c.Request(c.GetUserCount(), http.StatusOK)
	userCount, _ := c.GetJSONFieldAsInt64("count")
	assert.Equal(t, int64(1), userCount)
	c.Request(c.DeleteUser(jwt2, username2, password2), http.StatusNoContent)
	c.Request(c.GetUserCount(), http.StatusOK)
	userCount, _ = c.GetJSONFieldAsInt64("count")
	assert.Equal(t, int64(0), userCount)
}

func Test_BuildOrgDoAuthChecks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pincherServer := doServerSetup(t)
	c := APITestClient{Mux: pincherServer.Handler, testState: t}

	// REQUESTS

	// Delete all users
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)

	// Create four users
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)
	c.Request(c.CreateUser(username2, password2), http.StatusCreated)
	c.Request(c.CreateUser(username3, password3), http.StatusCreated)
	user3ID, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateUser(username4, password4), http.StatusCreated)

	// Log in four users
	c.Request(c.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")
	c.Request(c.LoginUser(username2, password2), http.StatusOK)
	jwt2, _ := c.GetJSONFieldAsString("token")
	c.Request(c.LoginUser(username3, password3), http.StatusOK)
	c.Request(c.LoginUser(username4, password4), http.StatusOK)
	jwt4, _ := c.GetJSONFieldAsString("token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	c.Request(c.CreateBudget(jwt1, "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateBudget(jwt1, "Personal", "user1's budget for personal finance."), http.StatusCreated)

	// Try adding user2 as ADMIN using user4 (not in budget), then as MANAGER. Both should fail.
	c.Request(c.AssignMemberToBudget(jwt4, budget1, username2, roleAdmin), http.StatusForbidden)
	c.Request(c.AssignMemberToBudget(jwt4, budget1, username2, roleManager), http.StatusForbidden)

	// Add user4 to Webflyx Org as user1 ADMIN, with role: VIEWER
	c.Request(c.AssignMemberToBudget(jwt1, budget1, username4, roleViewer), http.StatusCreated)

	// user4 should be assigned to only 1 budget
	c.Request(c.GetUserBudgets(jwt4), http.StatusOK)
	gotBudgets, _ := c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 1)

	// As user4, try adding user2 as a MANAGER. Should fail auth check.
	c.Request(c.AssignMemberToBudget(jwt4, budget1, username2, roleManager), http.StatusForbidden)

	// As user1 ADMIN, add user2 and user3 as MANAGER and CONTRIBUTOR, respectively.
	c.Request(c.AssignMemberToBudget(jwt1, budget1, username2, roleManager), http.StatusCreated)
	c.Request(c.AssignMemberToBudget(jwt1, budget1, username3, roleContributor), http.StatusCreated)

	// Attemc.deletion of Webflyx Org budget as user2. Should fail; only admin can do it.
	c.Request(c.DeleteUserBudget(jwt2, budget1), http.StatusForbidden)

	// Attemc.to revoke user3's Webflyx Org membership as user4. Should fail.
	c.Request(c.RevokeBudgetMembership(jwt4, budget1, user3ID), http.StatusForbidden)
	// Revoke user3's Webflyx Org membership as user1. Should succeed.
	c.Request(c.RevokeBudgetMembership(jwt1, budget1, user3ID), http.StatusNoContent)

	// user1 should be assigned to 2 budgets: Webflyx Org & their personal budget
	c.Request(c.GetUserBudgets(jwt1), http.StatusOK)
	gotBudgets, _ = c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 2)

	// Attemc.deletion of Webflyx Org budget as user1. Should succeed.
	c.Request(c.DeleteUserBudget(jwt1, budget1), http.StatusNoContent)

	// user1 should be assigned to only 1 budget now: their personal budget.
	c.Request(c.GetUserBudgets(jwt1), http.StatusOK)
	gotBudgets, _ = c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 1)

	// user4 should be assigned to NO budgets, now.
	c.Request(c.GetUserBudgets(jwt4), http.StatusOK)
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
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)

	// Create four users
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)
	c.Request(c.CreateUser(username2, password2), http.StatusCreated)
	c.Request(c.CreateUser(username3, password3), http.StatusCreated)
	c.Request(c.CreateUser(username4, password4), http.StatusCreated)

	// Log in four users
	c.Request(c.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")
	c.Request(c.LoginUser(username2, password2), http.StatusOK)
	jwt2, _ := c.GetJSONFieldAsString("token")
	c.Request(c.LoginUser(username3, password3), http.StatusOK)
	jwt3, _ := c.GetJSONFieldAsString("token")
	c.Request(c.LoginUser(username4, password4), http.StatusOK)
	jwt4, _ := c.GetJSONFieldAsString("token")

	// user1 ADMIN: Creating Webflyx Org budget & assigning u2, u3, u4 as MANAGER, CONTRIBUTOR, VIEWER.
	c.Request(c.CreateBudget(jwt1, "Webflyx Org", "For budgeting Webflyx Org financial resources and tracking expenses."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")
	c.Request(c.AssignMemberToBudget(jwt1, budget1, username2, roleManager), http.StatusCreated)
	c.Request(c.AssignMemberToBudget(jwt1, budget1, username3, roleContributor), http.StatusCreated)
	c.Request(c.AssignMemberToBudget(jwt1, budget1, username4, roleViewer), http.StatusCreated)

	// user2 MANAGER: Adding account, groups, & categories.
	c.Request(c.CreateBudgetAccount(jwt2, budget1, "savings", "Saved Org Funds", "Represents a bank account holding business capital."), http.StatusCreated)
	c.Request(c.CreateBudgetAccount(jwt2, budget1, "credit", "Employee Business Credit Account", "Employees use cards that pull from this account to pay for business expenses."), http.StatusCreated)
	account2, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateGroup(jwt2, budget1, "Business Capital", "Categories related to company capital"), http.StatusCreated)
	c.Request(c.CreateCategory(jwt2, budget1, "Business Capital", "Surplus", "Category representing surplus funding to be spent on elective improvements to organization headquarters or employee bonuses."), http.StatusCreated)
	category1, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateCategory(jwt2, budget1, "Business Capital", "Expenses", "Category representing funds to be used for employee expenses while on the job."), http.StatusCreated)
	category2, _ := c.GetJSONFieldAsString("id")

	// user3 CONTRIBUTOR: Adding transactions (EX: gas station).
	c.Request(c.CreateBudgetPayee(jwt3, budget1, "Smash & Dash", "A gas & convenience store"), http.StatusCreated)
	payee1ID, _ := c.GetJSONFieldAsString("id")

	c.Request(c.LogTransaction(jwt3, budget1, "Employee Business Credit Account", "", dateSeptember, "Smash & Dash", "I filled up vehicle w/ plate no. 555-555 @ the Smash & Pass gas station.", true, map[string]int64{category2: -1800}), http.StatusCreated)
	// transaction1, _ := c.GetJSONFieldAsString("id")

	c.Request(c.LogTransaction(jwt3, budget1, "Employee Business Credit Account", "", dateSeptember, "Smash & Dash", "Yeah, I got a drink in the convenience store too; sue me. Take it out of my bonus or whatever.", true, map[string]int64{category1: -400}), http.StatusCreated)
	// transaction2, _ := c.GetJSONFieldAsString("id")

	// user4 VIEWER: Works for accounting; reading transactions from employees.
	c.Request(c.GetTransactions(jwt4, budget1, account2, "", "", "", ""), http.StatusOK)
	c.Request(c.GetTransactions(jwt4, budget1, "", "", payee1ID, "", ""), http.StatusOK)

	// Delete all users
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
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
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)

	// Create user
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)

	// Log in user
	c.Request(c.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")

	// user2 ADMIN: Creating personal budget, making accounts and deposit transactions.
	c.Request(c.CreateBudget(jwt1, "Personal Budget", "For personal accounting (user1)."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateBudgetAccount(jwt1, budget1, "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."), http.StatusCreated)
	account1, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateBudgetAccount(jwt1, budget1, "credit", "Credit (Big Banking Inc)", "Reflects my credit account opened via Big Banking, Inc."), http.StatusCreated)
	account2, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateGroup(jwt1, budget1, "Spending", "Categories related to day-to-day spending"), http.StatusCreated)

	c.Request(c.CreateCategory(jwt1, budget1, "Spending", "Dining Out", "Money for ordering takeout or dining in."), http.StatusCreated)
	category1, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateBudgetPayee(jwt1, budget1, "Webflyx Org", "user1 employer"), http.StatusCreated)

	c.Request(c.CreateBudgetPayee(jwt1, budget1, "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."), http.StatusCreated)

	// deposit some money into checking account, allocated (but not explicitly assigned) to the DINING OUT category
	c.Request(c.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateSeptember, "Webflyx Org", "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", true, map[string]int64{category1: 10000}), http.StatusCreated)

	c.Request(c.GetBudgetCapital(jwt1, budget1, account1), http.StatusOK)
	budgetCheckingCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(10000), budgetCheckingCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1, account2), http.StatusOK)
	budgetCreditCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(10000), budgetTotalCapital)

	// spend money out of a credit account
	c.Request(c.LogTransaction(jwt1, budget1, "Credit (Big Banking Inc)", "", dateSeptember, "Messy Joe's", "$50 dinner at a restaurant", true, map[string]int64{category1: -5000}), http.StatusCreated)

	c.Request(c.GetBudgetCapital(jwt1, budget1, account2), http.StatusOK)
	budgetCreditCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(-5000), budgetCreditCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// pay off credit account, using the checking account, using a transfer transaction
	c.Request(c.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "Credit (Big Banking Inc)", dateSeptember, "ACCOUNT TRANSFER", "Pay off credit account balance", true, map[string]int64{"TRANSFER AMOUNT": -5000}), http.StatusCreated)

	c.Request(c.GetBudgetCapital(jwt1, budget1, account2), http.StatusOK)
	budgetCreditCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1, account1), http.StatusOK)
	budgetCheckingCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetCheckingCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// Delete all users
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
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
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)

	// Create user
	c.Request(c.CreateUser(username1, password1), http.StatusCreated)

	// Log in user
	c.Request(c.LoginUser(username1, password1), http.StatusOK)
	jwt1, _ := c.GetJSONFieldAsString("token")

	// user1 ADMIN: Creating personal budget, making account and other resources.
	c.Request(c.CreateBudget(jwt1, "Personal Budget", "For personal accounting (user1)."), http.StatusCreated)
	budget1, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateBudgetAccount(jwt1, budget1, "checking", "Checking (Big Banking Inc)", "Reflects my checking account opened via Big Banking, Inc."), http.StatusCreated)

	// Note: lack of group assignment is purposeful.
	c.Request(c.CreateCategory(jwt1, budget1, "", "Dining Out", "Money for ordering takeout or dining in."), http.StatusCreated)
	category1, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateCategory(jwt1, budget1, "", "Savings", "My savings fund."), http.StatusCreated)
	category2, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateBudgetPayee(jwt1, budget1, "Webflyx Org", "user1 employer"), http.StatusCreated)
	c.Request(c.CreateBudgetPayee(jwt1, budget1, "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."), http.StatusCreated)

	// SEc.MBER 2025 ACTIVITIES
	// deposit some money into the checking account, allocated (but not explicitly assigned) to the Dining Out category
	c.Request(c.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateSeptember, "Webflyx Org", "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", true, map[string]int64{category1: 10000}), http.StatusCreated)

	// spend money out of Dining Out category
	c.Request(c.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateSeptember, "Messy Joe's", "$50 dinner at a restaurant", true, map[string]int64{category1: -5000}), http.StatusCreated)

	// we expect that there's 5000 in capital remaining, and NO assignable money.
	// 5000 in Dining Out; 0 in Savings.
	c.Request(c.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1, dateSeptember, category1), http.StatusOK)
	balanceCategory1, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1, dateSeptember, category2), http.StatusOK)
	balanceCategory2, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(0), balanceCategory2)

	// OCTOBER 2025 ACTIVITIES
	// deposit more money into the checking account, with NO category allocation.
	// Assign some (but not all of it, to test for underassignment) to each of two categories.
	c.Request(c.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateOctober, "Webflyx Org", "$100 deposit into account; no category allocation, we'll assign it manually.", true, map[string]int64{"UNCATEGORIZED": 10000}), http.StatusCreated)

	c.Request(c.AssignMoneyToCategory(jwt1, budget1, dateOctober, category1, 4000), http.StatusOK)
	c.Request(c.AssignMoneyToCategory(jwt1, budget1, dateOctober, category2, 5000), http.StatusOK)

	// spend money out of Dining Out category
	c.Request(c.LogTransaction(jwt1, budget1, "Checking (Big Banking Inc)", "", dateOctober, "Messy Joe's", "I was very busy having fun, fun, fun!", true, map[string]int64{category1: -4000}), http.StatusCreated)

	// we expect that there's 11000 in capital remaining, and 1000 in assignable money.
	// 5000 in Dining Out; 5000 in Savings.
	c.Request(c.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1, dateOctober, category1), http.StatusOK)
	balanceCategory1, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), balanceCategory1)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1, dateOctober, category2), http.StatusOK)
	balanceCategory2, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), balanceCategory2)

	c.Request(c.GetMonthReport(jwt1, budget1, dateOctober), http.StatusOK)
	budgetTotalBalance, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(1000), (budgetTotalCapital - budgetTotalBalance))

	// NOVEMBER 2025 ACTIVITIES
	// deposit NO more money into the checking account.
	// Assign the 1000 left available from OCTOBER to the SAVINGS category.
	// Assign 1000 (that we don't have) to DINING OUT to test for overassignment.

	c.Request(c.AssignMoneyToCategory(jwt1, budget1, dateNovember, category2, 1000), http.StatusOK)
	c.Request(c.AssignMoneyToCategory(jwt1, budget1, dateNovember, category1, 1000), http.StatusOK)

	// we expect that there's still just 11000 in capital remaining, and -1000 in assignable money, which indicates overassignment.
	// 6000 in Dining Out; 6000 in Savings.
	c.Request(c.GetBudgetCapital(jwt1, budget1, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1, dateNovember, category1), http.StatusOK)
	balanceCategory1, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(6000), balanceCategory1)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1, dateNovember, category2), http.StatusOK)
	balanceCategory2, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(6000), balanceCategory2)

	c.Request(c.GetMonthReport(jwt1, budget1, dateNovember), http.StatusOK)
	budgetTotalBalance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(-1000), (budgetTotalCapital - budgetTotalBalance))

	// Delete all users
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
}
