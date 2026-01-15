package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresContainer struct {
	Ctx       context.Context
	Container postgres.PostgresContainer
	URI       string
}

type StdoutLogConsumer struct{}

func (lc *StdoutLogConsumer) Accept(l tc.Log) {
	if l.LogType == "STDERR" {
		_, err := fmt.Fprintln(os.Stdout, string(l.Content))
		if err != nil {
			fmt.Println("Error writing to stdout:", err)
			return
		}
	}
}

func SetupPostgres(t testing.TB) *postgresContainer {
	t.Helper()
	ctx := context.Background()

	// Ensure migration files exist
	_, err := filepath.Glob("../../sql/schema/*.sql")
	require.NoError(t, err)

	g := StdoutLogConsumer{}

	pgc, err := postgres.Run(
		ctx,
		"postgres:18.1-alpine",
		postgres.WithDatabase("pincher"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithLogConsumerConfig(&tc.LogConsumerConfig{
			Consumers: []tc.LogConsumer{&g},
		}),
		postgres.BasicWaitStrategies(),
		tc.WithReuseByName("pincherdb-integration-tests"),
	)
	defer tc.CleanupContainer(t, pgc)
	require.NoError(t, err)

	err = pgc.Snapshot(ctx)
	require.NoError(t, err)

	dbURL, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	return &postgresContainer{Ctx: ctx, Container: *pgc, URI: dbURL}
}

func MakeRequest(method, path, token string, body any) *http.Request {
	var buffer io.Reader

	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		buffer = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, buffer)
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}

// ------------------------
//  APITestClient REQUESTS
// ------------------------

// USER CRUD

func (c *APITestClient) CreateUser(username, password string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/users", "", map[string]any{
		"username": username,
		"password": password,
	})
}

func (c *APITestClient) GetUserCount() *http.Request {
	return MakeRequest(http.MethodGet, "/admin/users/count", "", nil)
}

func (c *APITestClient) UpdateUser(token, newUsername, newPassword string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/users", token, map[string]any{
		"username": newUsername,
		"password": newPassword,
	})
}

func (c *APITestClient) DeleteUser(token, username, password string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/users", token, map[string]any{
		"username": username,
		"password": password,
	})
}

func (c *APITestClient) DeleteAllUsers() *http.Request {
	return MakeRequest(http.MethodPost, "/admin/reset", "", nil)
}

// USER AUTH

func (c *APITestClient) LoginUser(username, password string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/login", "", map[string]any{
		"username": username,
		"password": password,
	})
}

// USER -> BUDGET CRUD

func (c *APITestClient) CreateBudget(token, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets", token, map[string]any{
		"name":  name,
		"notes": notes,
	})
}

func (c *APITestClient) GetUserBudgets(token string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets", token, nil)
}

func (c *APITestClient) UpdateBudget(token, budgetID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func (c *APITestClient) DeleteUserBudget(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID, token, nil)
}

// BUDGET -> ACCOUNT CRUD

func (c *APITestClient) CreateBudgetAccount(token, budgetID, accountType, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/accounts", token, map[string]any{
		"account_type": accountType,
		"name":         name,
		"notes":        notes,
	})
}

func (c *APITestClient) GetBudgetAccounts(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/accounts", token, nil)
}

func (c *APITestClient) GetBudgetCapital(token, budgetID, accountID string) *http.Request {
	path := "/api/budgets/" + budgetID
	if accountID != "" {
		path += "/accounts/" + accountID
	}
	path += "/capital"

	return MakeRequest(http.MethodGet, path, token, nil)
}

func (c *APITestClient) AssignMemberToBudget(token, budgetID, username, memberRole string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/members", token, map[string]any{
		"username":    username,
		"member_role": memberRole,
	})
}

func (c *APITestClient) UpdateAccount(token, budgetID, accountID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/accounts/"+accountID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func (c *APITestClient) RevokeBudgetMembership(token, budgetID, userID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/members/"+userID, token, nil)
}

func (c *APITestClient) DeleteBudgetAccount(token, budgetID, accountID, name string, deleteHard bool) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/accounts/"+accountID, token, map[string]any{
		"name":        name,
		"delete_hard": deleteHard,
	})
}

// BUDGET -> PAYEE CRUD

func (c *APITestClient) CreateBudgetPayee(token, budgetID, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/payees", token, map[string]any{
		"name":  name,
		"notes": notes,
	})
}

func (c *APITestClient) GetBudgetPayees(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/payees", token, nil)
}

func (c *APITestClient) UpdatePayee(token, budgetID, payeeID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/payees/"+payeeID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func (c *APITestClient) DeletePayee(token, budgetID, payeeID, newPayeeName string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/payees/"+payeeID, token, map[string]any{
		"new_payee_name": newPayeeName,
	})
}

// BUDGET -> CATEGORY CRUD

func (c *APITestClient) CreateCategory(token, budgetID, groupName, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/categories", token, map[string]any{
		"name":       name,
		"notes":      notes,
		"group_name": groupName,
	})
}

func (c *APITestClient) GetBudgetCategories(token, budgetID, query string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/categories"+query, token, nil)
}

func (c *APITestClient) UpdateCategory(token, budgetID, categoryID, groupName, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/categories/"+categoryID, token, map[string]any{
		"name":       newName,
		"notes":      newNotes,
		"group_name": groupName,
	})
}

func (c *APITestClient) DeleteBudgetCategory(token, budgetID, categoryID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/categories/"+categoryID, token, nil)
}

// BUDGET -> GROUP CRUD

func (c *APITestClient) CreateGroup(token, budgetID, name, notes string) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/groups", token, map[string]any{
		"name":  name,
		"notes": notes,
	})
}

func (c *APITestClient) GetBudgetGroups(token, budgetID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/groups", token, nil)
}

func (c *APITestClient) UpdateGroup(token, budgetID, groupID, newName, newNotes string) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/groups/"+groupID, token, map[string]any{
		"name":  newName,
		"notes": newNotes,
	})
}

func (c *APITestClient) DeleteBudgetGroup(token, budgetID, groupID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/groups/"+groupID, token, nil)
}

// BUDGET -> TRANSACTION CRUD

func (c *APITestClient) LogTransaction(token, budgetID, accountName, transferAccountName, transactionDate, payeeName, notes string, isCleared bool, amounts map[string]int64) *http.Request {
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

func (c *APITestClient) GetTransactions(token, budgetID, accountID, categoryID, payeeID, startDate, endDate string) *http.Request {
	query := url.Values{}
	if startDate != "" && endDate != "" {
		query.Set("start_date", startDate)
		query.Set("end_date", endDate)
	}
	if accountID != "" {
		query.Set("account_id", accountID)
	}
	if categoryID != "" {
		query.Set("category_id", categoryID)
	}
	if payeeID != "" {
		query.Set("payee_id", payeeID)
	}
	path := "/api/budgets/" + budgetID + "/transactions"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return MakeRequest(http.MethodGet, path, token, nil)
}

func (c *APITestClient) GetTransaction(token, budgetID, transactionID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/transactions/"+transactionID, token, nil)
}

func (c *APITestClient) UpdateTransaction(token, budgetID, transactionID, accountName, transferAccountName, transactionDate, payeeName, notes string, isCleared bool, amounts map[string]int64) *http.Request {
	return MakeRequest(http.MethodPut, "/api/budgets/"+budgetID+"/transactions/"+transactionID, token, map[string]any{
		"account_name":          accountName,
		"transfer_account_name": transferAccountName,
		"transaction_date":      transactionDate,
		"payee_name":            payeeName,
		"notes":                 notes,
		"amounts":               amounts,
		"is_cleared":            isCleared,
	})
}

func (c *APITestClient) DeleteTransaction(token, budgetID, transactionID string) *http.Request {
	return MakeRequest(http.MethodDelete, "/api/budgets/"+budgetID+"/transactions/"+transactionID, token, nil)
}

// BUDGET -> ASSIGNMENT CRUD

func (c *APITestClient) AssignMoneyToCategory(token, budgetID, monthID, categoryName string, amount int64) *http.Request {
	return MakeRequest(http.MethodPost, "/api/budgets/"+budgetID+"/months/"+monthID+"/categories", token, map[string]any{
		"amount":        amount,
		"to_category":   categoryName,
		"from_category": "",
	})
}

func (c *APITestClient) GetMonthCategoryReport(token, budgetID, monthID, categoryID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/months/"+monthID+"/categories/"+categoryID, token, nil)
}

func (c *APITestClient) GetMonthCategoryReports(token, budgetID, monthID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/months/"+monthID+"/categories", token, nil)
}

func (c *APITestClient) GetMonthReport(token, budgetID, monthID string) *http.Request {
	return MakeRequest(http.MethodGet, "/api/budgets/"+budgetID+"/months/"+monthID, token, nil)
}
