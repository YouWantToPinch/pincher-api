package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

// GetJSONField returns a field value from the last response recorded,
// at the given JSON path, assuming that the response content-type was JSON.
func (c *APITestClient) GetJSONField(JSONPath string) (any, error) {
	res := c.W.Result()
	defer res.Body.Close()

	var body map[string]any
	decoder := json.NewDecoder(res.Body)
	decoder.UseNumber()
	err := decoder.Decode(&body)
	if err != nil {
		return nil, err
	}

	var currentVal any = body
	for part := range strings.SplitSeq(JSONPath, ".") {
		m, ok := currentVal.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("path %q is not an object", part)
		}

		currentVal, ok = m[part]
		if !ok {
			return nil, fmt.Errorf("field %q not found", part)
		}
	}

	return currentVal, nil
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
	groupName := "testGroup"
	c.Request(c.CreateGroup(jwt1, budgetID1, "testGroup", "A group for testing."), http.StatusCreated)

	t.Run("Categories", func(t *testing.T) {
		groupID1, _ := c.GetJSONFieldAsString("id")

		// PREP: Create another group
		c.Request(c.CreateGroup(jwt1, budgetID1, "tempTestGroup", "A group for testing updates."), http.StatusCreated)

		// CREATE category
		c.Request(c.CreateCategory(jwt1, budgetID1, groupName, "testCategory", "A category for testing."), http.StatusCreated)
		categoryID1, _ := c.GetJSONFieldAsString("id")
		// READ categories(s)
		c.Request(c.GetBudgetCategories(jwt1, budgetID1, "?group_id="+groupID1), http.StatusOK)
		// UPDATE category
		c.Request(c.UpdateCategory(jwt1, budgetID1, categoryID1, "tempTestGroup", "Test Category", "Test user's first category, now assigned to a new group with new info."), http.StatusNoContent)
		// DELETE category
		c.Request(c.DeleteBudgetCategory(jwt1, budgetID1, categoryID1), http.StatusNoContent)
	})

	// PREP: Create category
	categoryName := "testCategory"
	c.Request(c.CreateCategory(jwt1, budgetID1, "testGroup", "testCategory", "A category for testing."), http.StatusCreated)

	t.Run("Accounts", func(t *testing.T) {
		// CREATE account
		c.Request(c.CreateBudgetAccount(jwt1, budgetID1, "ON_BUDGET", "testAccount", "An account for testing."), http.StatusCreated)
		accountID1, _ := c.GetJSONFieldAsString("id")
		// READ account(s)
		c.Request(c.GetBudgetAccounts(jwt1, budgetID1), http.StatusOK)
		// UPDATE account
		c.Request(c.UpdateAccount(jwt1, budgetID1, accountID1, "Test Account", "Test user's first account, now updated."), http.StatusNoContent)
		// DELETE account SOFT
		c.Request(c.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "Test Account", false), http.StatusOK)
		// DELETE account HARD
		c.Request(c.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "Test Account", true), http.StatusNoContent)
	})

	// CREATE account
	accountName := "testAccount"
	c.Request(c.CreateBudgetAccount(jwt1, budgetID1, "ON_BUDGET", accountName, "An account for testing."), http.StatusCreated)

	t.Run("Payees", func(t *testing.T) {
		// CREATE payee
		c.Request(c.CreateBudgetPayee(jwt1, budgetID1, "testPayee", "A payee for testing."), http.StatusCreated)
		payeeID1, _ := c.GetJSONFieldAsString("id")
		// READ payee(s)
		c.Request(c.GetBudgetPayees(jwt1, budgetID1), http.StatusOK)
		// UPDATE payee
		c.Request(c.UpdatePayee(jwt1, budgetID1, payeeID1, "Test Payee", "Test user's first payee, now updated."), http.StatusNoContent)
		// DELETE payee
		c.Request(c.DeletePayee(jwt1, budgetID1, payeeID1, ""), http.StatusNoContent)
	})

	// PREP: Create payee
	payeeName := "testPayee"
	c.Request(c.CreateBudgetPayee(jwt1, budgetID1, "testPayee", "A payee for testing."), http.StatusCreated)

	t.Run("Transactions", func(t *testing.T) {
		// PREP: Create payee, account
		c.Request(c.CreateBudgetPayee(jwt1, budgetID1, "testPayee2", "A payee for testing payee reassignment & deletion."), http.StatusCreated)
		payeeID2, _ := c.GetJSONFieldAsString("id")

		c.Request(c.CreateBudgetAccount(jwt1, budgetID1, "ON_BUDGET", "tempAccount", "An account for testing transfer interactions."), http.StatusCreated)
		accountID1, _ := c.GetJSONFieldAsString("id")

		// CREATE transaction (deposit)
		c.Request(c.LogTransaction(jwt1, budgetID1, accountName, "", dateSeptember, payeeName, "A $1000 DEPOSIT test transaction.", true, map[string]int64{categoryName: 100000}), http.StatusCreated)
		transactionID1, _ := c.GetJSONFieldAsString("id")
		// READ transaction
		c.Request(c.GetTransaction(jwt1, budgetID1, transactionID1), http.StatusOK)
		// UPDATE transaction (deposit)
		c.Request(c.UpdateTransaction(jwt1, budgetID1, transactionID1, accountName, "", dateSeptember, "testPayee2", "An updated deposit transaction (it was actually just $100)", true, map[string]int64{categoryName: 10000}), http.StatusNoContent)
		// DELETE payee
		c.Request(c.DeletePayee(jwt1, budgetID1, payeeID2, payeeName), http.StatusNoContent)

		// CREATE transaction (withdrawal)
		c.Request(c.LogTransaction(jwt1, budgetID1, accountName, "", dateSeptember, payeeName, "A $5 WITHDRAWAL test transaction.", true, map[string]int64{categoryName: -500}), http.StatusCreated)
		transactionID2, _ := c.GetJSONFieldAsString("id")

		// CREATE transaction (transfer)
		c.Request(c.LogTransaction(jwt1, budgetID1, accountName, "tempAccount", dateSeptember, "", "A transfer of $50 from testAccount to tempAccount.", true, map[string]int64{"TRANSFER": -5000}), http.StatusCreated)
		transactionID3, _ := c.GetJSONFieldAsString("from_transaction.id")

		// UPDATE transaction (transfer)
		c.Request(c.UpdateTransaction(jwt1, budgetID1, transactionID3, accountName, "tempAccount", dateSeptember, "testPayee2", "An updated transfer transaction (it was actually just $10). It should update the corresponding one.", true, map[string]int64{categoryName: 1000}), http.StatusNoContent)

		// DELETE transactions
		c.Request(c.DeleteTransaction(jwt1, budgetID1, transactionID1), http.StatusNoContent)
		c.Request(c.DeleteTransaction(jwt1, budgetID1, transactionID2), http.StatusNoContent)
		// we expect this to delete the corresponding transfer txn as part of its own process
		c.Request(c.DeleteTransaction(jwt1, budgetID1, transactionID3), http.StatusNoContent)

		// DELETE tempAccount (should work, as we expect all of its one txn to be deleted)
		c.Request(c.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "tempAccount", false), http.StatusOK)
		c.Request(c.DeleteBudgetAccount(jwt1, budgetID1, accountID1, "tempAccount", true), http.StatusNoContent)
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
	budget1ID, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateBudget(jwt1, "Personal", "user1's budget for personal finance."), http.StatusCreated)

	// Try adding user2 as ADMIN using user4 (not in budget), then as MANAGER. Both should fail.
	c.Request(c.AssignMemberToBudget(jwt4, budget1ID, username2, roleAdmin), http.StatusForbidden)
	c.Request(c.AssignMemberToBudget(jwt4, budget1ID, username2, roleManager), http.StatusForbidden)

	// Add user4 to Webflyx Org as user1 ADMIN, with role: VIEWER
	c.Request(c.AssignMemberToBudget(jwt1, budget1ID, username4, roleViewer), http.StatusCreated)

	// user4 should be assigned to only 1 budget
	c.Request(c.GetUserBudgets(jwt4), http.StatusOK)
	gotBudgets, _ := c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 1)

	// As user4, try adding user2 as a MANAGER. Should fail auth check.
	c.Request(c.AssignMemberToBudget(jwt4, budget1ID, username2, roleManager), http.StatusForbidden)

	// As user1 ADMIN, add user2 and user3 as MANAGER and CONTRIBUTOR, respectively.
	c.Request(c.AssignMemberToBudget(jwt1, budget1ID, username2, roleManager), http.StatusCreated)
	c.Request(c.AssignMemberToBudget(jwt1, budget1ID, username3, roleContributor), http.StatusCreated)

	// Attemc.deletion of Webflyx Org budget as user2. Should fail; only admin can do it.
	c.Request(c.DeleteUserBudget(jwt2, budget1ID), http.StatusForbidden)

	// Attemc.to revoke user3's Webflyx Org membership as user4. Should fail.
	c.Request(c.RevokeBudgetMembership(jwt4, budget1ID, user3ID), http.StatusForbidden)
	// Revoke user3's Webflyx Org membership as user1. Should succeed.
	c.Request(c.RevokeBudgetMembership(jwt1, budget1ID, user3ID), http.StatusNoContent)

	// user1 should be assigned to 2 budgets: Webflyx Org & their personal budget
	c.Request(c.GetUserBudgets(jwt1), http.StatusOK)
	gotBudgets, _ = c.GetJSONField("budgets")
	assert.Len(t, gotBudgets.([]any), 2)

	// Attemc.deletion of Webflyx Org budget as user1. Should succeed.
	c.Request(c.DeleteUserBudget(jwt1, budget1ID), http.StatusNoContent)

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
	budgetID, _ := c.GetJSONFieldAsString("id")
	c.Request(c.AssignMemberToBudget(jwt1, budgetID, username2, roleManager), http.StatusCreated)
	c.Request(c.AssignMemberToBudget(jwt1, budgetID, username3, roleContributor), http.StatusCreated)
	c.Request(c.AssignMemberToBudget(jwt1, budgetID, username4, roleViewer), http.StatusCreated)

	// user2 MANAGER: Adding account, groups, & categories.
	c.Request(c.CreateBudgetAccount(jwt2, budgetID, "ON_BUDGET", "Saved Org Funds", "Represents a bank account holding business capital."), http.StatusCreated)
	c.Request(c.CreateBudgetAccount(jwt2, budgetID, "ON_BUDGET", "Employee Business Credit Account", "Employees use cards that pull from this account to pay for business expenses."), http.StatusCreated)
	account2, _ := c.GetJSONFieldAsString("id")
	c.Request(c.CreateGroup(jwt2, budgetID, "Business Capital", "Categories related to company capital"), http.StatusCreated)
	category1Name := "Surplus"
	c.Request(c.CreateCategory(jwt2, budgetID, "Business Capital", category1Name, "Category representing surplus funding to be spent on elective improvements to organization headquarters or employee bonuses."), http.StatusCreated)
	category2Name := "Expenses"
	c.Request(c.CreateCategory(jwt2, budgetID, "Business Capital", category2Name, "Category representing funds to be used for employee expenses while on the job."), http.StatusCreated)

	// user3 CONTRIBUTOR: Adding transactions (EX: gas station).
	c.Request(c.CreateBudgetPayee(jwt3, budgetID, "Smash & Dash", "A gas & convenience store"), http.StatusCreated)
	payee1ID, _ := c.GetJSONFieldAsString("id")

	c.Request(c.LogTransaction(jwt3, budgetID, "Employee Business Credit Account", "", dateSeptember, "Smash & Dash", "I filled up vehicle w/ plate no. 555-555 @ the Smash & Pass gas station.", true, map[string]int64{category2Name: -1800}), http.StatusCreated)

	c.Request(c.LogTransaction(jwt3, budgetID, "Employee Business Credit Account", "", dateSeptember, "Smash & Dash", "Yeah, I got a drink in the convenience store too; sue me. Take it out of my bonus or whatever.", true, map[string]int64{category1Name: -400}), http.StatusCreated)

	// user4 VIEWER: Works for accounting; reading transactions from employees.
	c.Request(c.GetTransactions(jwt4, budgetID, account2, "", "", "", ""), http.StatusOK)
	c.Request(c.GetTransactions(jwt4, budgetID, "", "", payee1ID, "", ""), http.StatusOK)
	txnQuery, _ := c.GetJSONField("transactions")
	assert.Len(t, txnQuery.([]any), 2)

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
	budget1ID, _ := c.GetJSONFieldAsString("id")

	account1Name := "Checking (Big Banking Inc)"
	c.Request(c.CreateBudgetAccount(jwt1, budget1ID, "ON_BUDGET", account1Name, "Reflects my checking account opened via Big Banking, Inc."), http.StatusCreated)
	account1ID, _ := c.GetJSONFieldAsString("id")
	account2Name := "Credit (Big Banking Inc)"
	c.Request(c.CreateBudgetAccount(jwt1, budget1ID, "ON_BUDGET", account2Name, "Reflects my credit account opened via Big Banking, Inc."), http.StatusCreated)
	account2ID, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateGroup(jwt1, budget1ID, "Spending", "Categories related to day-to-day spending"), http.StatusCreated)

	categoryName := "Dining Out"
	c.Request(c.CreateCategory(jwt1, budget1ID, "Spending", "Dining Out", "Money for ordering takeout or dining in."), http.StatusCreated)

	c.Request(c.CreateBudgetPayee(jwt1, budget1ID, "Webflyx Org", "user1 employer"), http.StatusCreated)

	c.Request(c.CreateBudgetPayee(jwt1, budget1ID, "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."), http.StatusCreated)

	// deposit some money into checking account, allocated (but not explicitly assigned) to the DINING OUT category
	c.Request(c.LogTransaction(jwt1, budget1ID, account1Name, "", dateSeptember, "Webflyx Org", "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", true, map[string]int64{categoryName: 10000}), http.StatusCreated)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, account1ID), http.StatusOK)
	budgetCheckingCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(10000), budgetCheckingCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, account2ID), http.StatusOK)
	budgetCreditCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, ""), http.StatusOK)
	budgetTotalCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(10000), budgetTotalCapital)

	// spend money out of a credit account
	c.Request(c.LogTransaction(jwt1, budget1ID, "Credit (Big Banking Inc)", "", dateSeptember, "Messy Joe's", "$50 dinner at a restaurant", true, map[string]int64{categoryName: -5000}), http.StatusCreated)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, account2ID), http.StatusOK)
	budgetCreditCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(-5000), budgetCreditCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	// pay off credit account, using the checking account, using a transfer transaction
	c.Request(c.LogTransaction(jwt1, budget1ID, account1Name, account2Name, dateSeptember, "TRANSFER", "Pay off credit account balance", true, map[string]int64{"TRANSFER": -5000}), http.StatusCreated)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, account2ID), http.StatusOK)
	budgetCreditCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(0), budgetCreditCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, account1ID), http.StatusOK)
	budgetCheckingCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetCheckingCapital)

	c.Request(c.GetBudgetCapital(jwt1, budget1ID, ""), http.StatusOK)
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
	budget1ID, _ := c.GetJSONFieldAsString("id")

	accountName := "Checking (Big Banking Inc)"
	c.Request(c.CreateBudgetAccount(jwt1, budget1ID, "ON_BUDGET", accountName, "Reflects my checking account opened via Big Banking, Inc."), http.StatusCreated)

	// Note: lack of group assignment is purposeful.
	categoryName := "Dining Out"
	c.Request(c.CreateCategory(jwt1, budget1ID, "", "Dining Out", "Money for ordering takeout or dining in."), http.StatusCreated)
	category1ID, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateCategory(jwt1, budget1ID, "", "Savings", "My savings fund."), http.StatusCreated)
	category2ID, _ := c.GetJSONFieldAsString("id")

	c.Request(c.CreateBudgetPayee(jwt1, budget1ID, "Webflyx Org", "user1 employer"), http.StatusCreated)
	c.Request(c.CreateBudgetPayee(jwt1, budget1ID, "Messy Joe's", "Nice atmosphere. Food's great. It's got a bit of an edge."), http.StatusCreated)

	// SEc.MBER 2025 ACTIVITIES
	// deposit some money into the checking account, allocated (but not explicitly assigned) to the Dining Out category
	c.Request(c.LogTransaction(jwt1, budget1ID, accountName, "", dateSeptember, "Webflyx Org", "$100 deposit into account; set category to Dining Out to automatically assign it to that category.", true, map[string]int64{categoryName: 10000}), http.StatusCreated)

	// spend money out of Dining Out category
	c.Request(c.LogTransaction(jwt1, budget1ID, accountName, "", dateSeptember, "Messy Joe's", "$50 dinner at a restaurant", true, map[string]int64{categoryName: -5000}), http.StatusCreated)

	// we expect that there's 5000 in capital remaining, and NO assignable money.
	// 5000 in Dining Out; 0 in Savings.
	c.Request(c.GetBudgetCapital(jwt1, budget1ID, ""), http.StatusOK)
	budgetTotalCapital, _ := c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(5000), budgetTotalCapital)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1ID, dateSeptember, category1ID), http.StatusOK)
	category1Balance, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), category1Balance)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1ID, dateSeptember, category2ID), http.StatusOK)
	category2Balance, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(0), category2Balance)

	// OCTOBER 2025 ACTIVITIES
	// deposit more money into the checking account, with NO category allocation.
	// Assign some (but not all of it, to test for underassignment) to each of two categories.
	c.Request(c.LogTransaction(jwt1, budget1ID, accountName, "", dateOctober, "Webflyx Org", "$100 deposit into account; no category allocation, we'll assign it manually.", true, map[string]int64{"UNCATEGORIZED": 10000}), http.StatusCreated)

	c.Request(c.AssignMoneyToCategory(jwt1, budget1ID, dateOctober, category1ID, 4000), http.StatusOK)
	c.Request(c.AssignMoneyToCategory(jwt1, budget1ID, dateOctober, category2ID, 5000), http.StatusOK)

	// spend money out of Dining Out category
	c.Request(c.LogTransaction(jwt1, budget1ID, accountName, "", dateOctober, "Messy Joe's", "I was very busy having fun, fun, fun!", true, map[string]int64{categoryName: -4000}), http.StatusCreated)

	// we expect that there's 11000 in capital remaining, and 1000 in assignable money.
	// 5000 in Dining Out; 5000 in Savings.
	c.Request(c.GetBudgetCapital(jwt1, budget1ID, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1ID, dateOctober, category1ID), http.StatusOK)
	category1Balance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), category1Balance)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1ID, dateOctober, category2ID), http.StatusOK)
	category2Balance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(5000), category2Balance)

	c.Request(c.GetMonthReport(jwt1, budget1ID, dateOctober), http.StatusOK)
	budgetTotalBalance, _ := c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(1000), (budgetTotalCapital - budgetTotalBalance))

	// NOVEMBER 2025 ACTIVITIES
	// deposit NO more money into the checking account.
	// Assign the 1000 left available from OCTOBER to the SAVINGS category.
	// Assign 1000 (that we don't have) to DINING OUT to test for overassignment.

	c.Request(c.AssignMoneyToCategory(jwt1, budget1ID, dateNovember, category2ID, 1000), http.StatusOK)
	c.Request(c.AssignMoneyToCategory(jwt1, budget1ID, dateNovember, category1ID, 1000), http.StatusOK)

	// we expect that there's still just 11000 in capital remaining, and -1000 in assignable money, which indicates overassignment.
	// 6000 in Dining Out; 6000 in Savings.
	c.Request(c.GetBudgetCapital(jwt1, budget1ID, ""), http.StatusOK)
	budgetTotalCapital, _ = c.GetJSONFieldAsInt64("capital")
	assert.Equal(t, int64(11000), budgetTotalCapital)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1ID, dateNovember, category1ID), http.StatusOK)
	category1Balance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(6000), category1Balance)

	c.Request(c.GetMonthCategoryReport(jwt1, budget1ID, dateNovember, category2ID), http.StatusOK)
	category2Balance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(6000), category2Balance)

	c.Request(c.GetMonthReport(jwt1, budget1ID, dateNovember), http.StatusOK)
	budgetTotalBalance, _ = c.GetJSONFieldAsInt64("balance")
	assert.Equal(t, int64(-1000), (budgetTotalCapital - budgetTotalBalance))

	// Delete all users
	c.Request(c.DeleteAllUsers(), http.StatusNoContent)
}
