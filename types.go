package main

import (
	"fmt"
	"strconv"
	"strings"
	"errors"
	"time"

	"github.com/google/uuid"
)

// USD used to represent some amount in US cents.
type Cent int64

func(u Cent) Display() string {
	s := strconv.FormatInt(int64(u), 10)
	i := len(s) - 2
	return s[:i] + string('.') + s[i:]
}

type BudgetMemberRole int

const (
	ADMIN BudgetMemberRole = iota
	MANAGER
	CONTRIBUTOR
	VIEWER
)

var bmrToString = map[BudgetMemberRole]string{
	ADMIN:		 "ADMIN",
	MANAGER:	 "MANAGER",
	CONTRIBUTOR: "CONTRIBUTOR",
	VIEWER:		 "VIEWER",
}

var bmrFromString = map[string]BudgetMemberRole{
	"ADMIN":		ADMIN,
	"MANAGER":		MANAGER,
	"CONTRIBUTOR":	CONTRIBUTOR,
	"VIEWER":		VIEWER,
}

func (bmr BudgetMemberRole) String() string {
	return bmrToString[bmr]
}

func BMRFromString(s string) (BudgetMemberRole, error) {
	s = strings.ToUpper(s)
	if val, ok := bmrFromString[s]; ok {
		return val, nil
	}
	return -1, errors.New(fmt.Sprintf("Invalid role: %s", s))
}

type User struct {
	ID        		uuid.UUID	`json:"id"`
	CreatedAt 		time.Time	`json:"created_at"`
	UpdatedAt 		time.Time	`json:"updated_at"`
	Username     	string   	`json:"username"`
	HashedPassword	string		`json:"hashed_password"`
	Token			string		`json:"token"`
	RefreshToken	string		`json:"refresh_token"`
}

type BudgetMembership struct {
	BudgetID	uuid.UUID			`json:"budget_id`
	UserID		uuid.UUID			`json:"user_id`
	MemberRole	BudgetMemberRole	`json:"member_role"`
}

type Budget struct {
	ID        		uuid.UUID	`json:"id"`
	CreatedAt 		time.Time	`json:"created_at"`
	UpdatedAt 		time.Time	`json:"updated_at"`
	AdminID        	uuid.UUID	`json:"admin_id"`
	Name			string		`json:"name`
	Notes			string		`json:"notes`
}

type Group struct {
	ID				uuid.UUID	`json:"id"`
	CreatedAt 		time.Time	`json:"created_at"`
	UpdatedAt 		time.Time	`json:"updated_at"`
	BudgetID     	uuid.UUID   `json:"user_id"`
	Name			string		`json:"name"`
	Notes			string		`json:"notes"`
}

type Category struct {
	ID				uuid.UUID		`json:"id"`
	CreatedAt 		time.Time		`json:"created_at"`
	UpdatedAt 		time.Time		`json:"updated_at"`
	BudgetID     	uuid.UUID   	`json:"user_id"`
	Name			string			`json:"name"`
	GroupID			uuid.NullUUID	`json:"group_id"`
	Notes			string			`json:"notes"`
}

type Account struct {
		ID 				uuid.UUID	`json:"id"`
		CreatedAt		time.Time	`json:"created_at"`
		UpdatedAt		time.Time	`json:"updated_at"`
		BudgetID		uuid.UUID	`json:"budget_id"`
		AccountType		string		`json:"account_type"`
		Name			string		`json:"name"`
		Notes			string		`json:"notes"`
		IsDeleted		bool		`json:"is_deleted"`
	}

type Transaction struct {
		ID 				uuid.UUID	`json:"id"`
		CreatedAt		time.Time	`json:"created_at"`
		UpdatedAt		time.Time	`json:"updated_at"`
		BudgetID		uuid.UUID	`json:"budget_id"`
		LoggerID		uuid.UUID	`json:"logger_id"`
		AccountID		uuid.UUID	`json:"account_id"`
		TransactionDate time.Time	`json:"transaction_date"`
		PayeeID			uuid.UUID	`json:"payee_id"`
		Notes			string		`json:"notes"`
		Cleared			bool		`json:"is_cleared"`
	}

type Payee struct {
		ID 				uuid.UUID	`json:"id"`
		CreatedAt		time.Time	`json:"created_at"`
		UpdatedAt		time.Time	`json:"updated_at"`
		BudgetID		uuid.UUID	`json:"budget_id"`
		Name			string		`json:"name"`
	}