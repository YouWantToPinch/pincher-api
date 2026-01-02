package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BudgetMemberRole int

const (
	ADMIN BudgetMemberRole = iota
	MANAGER
	CONTRIBUTOR
	VIEWER
)

var bmrToString = map[BudgetMemberRole]string{
	ADMIN:       "ADMIN",
	MANAGER:     "MANAGER",
	CONTRIBUTOR: "CONTRIBUTOR",
	VIEWER:      "VIEWER",
}

var bmrFromString = map[string]BudgetMemberRole{
	"ADMIN":       ADMIN,
	"MANAGER":     MANAGER,
	"CONTRIBUTOR": CONTRIBUTOR,
	"VIEWER":      VIEWER,
}

func (bmr BudgetMemberRole) String() string {
	return bmrToString[bmr]
}

func BMRFromString(s string) (BudgetMemberRole, error) {
	s = strings.ToUpper(s)
	if val, ok := bmrFromString[s]; ok {
		return val, nil
	}
	return -1, fmt.Errorf("invalid role: %s", s)
}

type Meta struct {
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

type User struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Username       string    `json:"username"`
	HashedPassword string    `json:"-"`
}

type BudgetMembership struct {
	BudgetID   uuid.UUID        `json:"budget_id"`
	UserID     uuid.UUID        `json:"user_id"`
	MemberRole BudgetMemberRole `json:"member_role"`
}

type Budget struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	AdminID   uuid.UUID `json:"admin_id"`
	Meta
}

type Group struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	BudgetID  uuid.UUID `json:"budget_id"`
	Meta
}

type Category struct {
	ID        uuid.UUID     `json:"id"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	BudgetID  uuid.UUID     `json:"budget_id"`
	GroupID   uuid.NullUUID `json:"group_id"`
	Meta
}

type Account struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	BudgetID    uuid.UUID `json:"budget_id"`
	AccountType string    `json:"account_type"`
	IsDeleted   bool      `json:"is_deleted"`
	Meta
}

type Transaction struct {
	ID              uuid.UUID `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	BudgetID        uuid.UUID `json:"budget_id"`
	LoggerID        uuid.UUID `json:"logger_id"`
	AccountID       uuid.UUID `json:"account_id"`
	TransactionType string    `json:"transaction_type"`
	TransactionDate time.Time `json:"transaction_date"`
	PayeeID         uuid.UUID `json:"payee_id"`
	Notes           string    `json:"notes"`
	Cleared         bool      `json:"is_cleared"`
}

type TransactionSplit struct {
	ID            uuid.UUID     `json:"id"`
	TransactionID uuid.UUID     `json:"transaction_id"`
	CategoryID    uuid.NullUUID `json:"category_id"`
	Amount        int64         `json:"transaction_date"`
}

type TransactionView struct {
	ID              uuid.UUID      `json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	BudgetID        uuid.UUID      `json:"budget_id"`
	LoggerID        uuid.UUID      `json:"logger_id"`
	AccountID       uuid.UUID      `json:"account_id"`
	TransactionType string         `json:"transaction_type"`
	TransactionDate time.Time      `json:"transaction_date"`
	Payee           string         `json:"payee"`
	PayeeID         uuid.UUID      `json:"payee_id"`
	Notes           string         `json:"notes"`
	Cleared         bool           `json:"is_cleared"`
	TotalAmount     int64          `json:"total_amount"`
	Splits          map[string]int `json:"splits"`
}

type Payee struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	BudgetID  uuid.UUID `json:"budget_id"`
	Meta
}

type CategoryReport struct {
	MonthID    time.Time `json:"month_id"`
	CategoryID uuid.UUID `json:"category_id"`
	Name       string    `json:"category_name"`
	Assigned   int64     `json:"assigned"`
	Activity   int64     `json:"activity"`
	Balance    int64     `json:"balance"`
}

type MonthReport struct {
	Assigned int64 `json:"assigned"`
	Activity int64 `json:"activity"`
	Balance  int64 `json:"balance"`
}
