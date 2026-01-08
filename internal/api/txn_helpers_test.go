package api

import (
	"testing"
	"time"

	"github.com/YouWantToPinch/pincher-api/internal/database"
)

func TestCheckIsTransfer(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{
			name:   "Is Transfer: TRANSFER_TO",
			input:  "TRANSFER_TO",
			expect: true,
		},
		{
			name:   "Is Transfer: TRANSFER_FROM",
			input:  "TRANSFER_FROM",
			expect: true,
		},
		{
			name:   "Not Transfer: TRANSFER",
			input:  "TRANSFER",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := checkIsTransfer(tt.input)
			if actual != tt.expect {
				t.Errorf("want: %v | actual: %v", tt.expect, actual)
			}
		})
	}
}

func TestInvertTransferType(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "TRANSFER_TO -> TRANSFER_FROM",
			input:  "TRANSFER_TO",
			expect: "TRANSFER_FROM",
		},
		{
			name:   "TRANSFER_FROM -> TRANSFER_TO",
			input:  "TRANSFER_FROM",
			expect: "TRANSFER_TO",
		},
		{
			name:   "Deposit; return input unchanged",
			input:  "DEPOSIT",
			expect: "DEPOSIT",
		},
		{
			name:   "Withdrawal; return input unchanged",
			input:  "WITHDRAWAL",
			expect: "WITHDRAWAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := invertTransferType(tt.input)
			if actual != tt.expect {
				t.Errorf("want: %v | actual: %v", tt.expect, actual)
			}
		})
	}
}

func TestInvertAmountsMap(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]int64
		expect map[string]int64
	}{
		{
			name:   "Zero len map returns identical",
			input:  map[string]int64{},
			expect: map[string]int64{},
		},
		{
			name:   "Positives -> Negatives",
			input:  map[string]int64{"Category1": 1, "Category2": 3, "Category3": 5, "Category4": 42},
			expect: map[string]int64{"Category1": -1, "Category2": -3, "Category3": -5, "Category4": -42},
		},
		{
			name:   "Negatives -> Positives",
			input:  map[string]int64{"Category1": -1, "Category2": -3, "Category3": -5, "Category4": -42},
			expect: map[string]int64{"Category1": 1, "Category2": 3, "Category3": 5, "Category4": 42},
		},
		{
			// NOTE: validateTxn function should return an error on mixed signs when validating amounts.
			// Still, this is a good sanity-check test for this unit.
			name:   "Mixed signs invert",
			input:  map[string]int64{"Category1": -1, "Category2": 3, "Category3": -5, "Category4": 42},
			expect: map[string]int64{"Category1": 1, "Category2": -3, "Category3": 5, "Category4": -42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.expect) != len(tt.input) {
				t.Errorf("unexpected map length; want: %v | actual: %v", len(tt.input), len(tt.expect))
			}
			for k, v := range tt.input {
				if tt.expect[k] != -v {
					t.Errorf("for key %v in expected map; want: %v | actual: %v", k, -v, tt.expect[k])
				}
			}
		})
	}
}

func TestGetOrderedTransferIDs(t *testing.T) {
	// PREP: pointers for testing
	fromTxnPtr := &database.Transaction{TransactionType: "TRANSFER_FROM"}
	toTxnPtr := &database.Transaction{TransactionType: "TRANSFER_TO"}

	t.Run("input 'to, from' returns identical", func(t *testing.T) {
		toPtr, fromPtr, _ := getOrderedTransferIDs(toTxnPtr, fromTxnPtr)
		if toPtr != toTxnPtr || fromPtr != fromTxnPtr {
			t.Errorf("unexpected ptr order; want: 'to, from' | actual: 'from, to'")
		}
	})
	t.Run("input 'from, to' returns swapped", func(t *testing.T) {
		toPtr, fromPtr, _ := getOrderedTransferIDs(fromTxnPtr, toTxnPtr)
		if toPtr != toTxnPtr || fromPtr != fromTxnPtr {
			t.Errorf("unexpected ptr order; want: 'to, from' | actual: 'from, to'")
		}
	})
	t.Run("input 'to, non-transfer' returns error", func(t *testing.T) {
		nonTransferTxnPtr := &database.Transaction{TransactionType: "DEPOSIT"}
		_, _, err := getOrderedTransferIDs(toTxnPtr, nonTransferTxnPtr)
		if err == nil {
			t.Errorf("expected error, but got none")
		}
	})
	t.Run("input 'to, to-transfer' returns error", func(t *testing.T) {
		otherToTxnPtr := &database.Transaction{TransactionType: "TRANSFER_TO"}
		_, _, err := getOrderedTransferIDs(toTxnPtr, otherToTxnPtr)
		if err == nil {
			t.Errorf("expected error, but got none")
		}
	})
}

func TestTotalAmountsFromMap(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]int64
		expect int64
	}{
		{
			name:   "1 + 1 = 2",
			input:  map[string]int64{"Category1": 1, "Category2": 1},
			expect: 2,
		},
		{
			name:   "1 + 3 + 5 + 42 = 51",
			input:  map[string]int64{"Category1": 1, "Category2": 3, "Category3": 5, "Category4": 42},
			expect: 51,
		},
		{
			name:   "-1 + -3 + -5 + -42 = -51",
			input:  map[string]int64{"Category1": -1, "Category2": -3, "Category3": -5, "Category4": -42},
			expect: -51,
		},
		{
			name:   "-1 + 3 + -5 + 42 = 39",
			input:  map[string]int64{"Category1": -1, "Category2": 3, "Category3": -5, "Category4": 42},
			expect: 39,
		},
		{
			name:   "empty map zero",
			input:  map[string]int64{},
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := totalFromAmountsMap(tt.input)
			if actual != tt.expect {
				t.Errorf("want: %v | acatual: %v", tt.expect, actual)
			}
		})
	}
}

func TestValidateTXN(t *testing.T) {
	tests := []struct {
		name             string
		mockPayload      *UpsertTransactionRqSchema
		expectAmounts    int
		expectDate       time.Time
		expectIsTransfer bool
		expectType       string
		expectedVal      bool
		wantErr          bool
	}{
		{
			name: "Infer TRANSFER_FROM",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate:     "2025-09-15",
				TransferAccountName: "OtherAccount",
				Amounts: map[string]int64{
					"UNCATEGORIZED": -1000,
				},
			},
			expectAmounts:    1,
			expectType:       "TRANSFER_FROM",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: true,
			wantErr:          false,
		},
		{
			name: "Infer TRANSFER_TO",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate:     "2025-09-15",
				TransferAccountName: "OtherAccount",
				Amounts: map[string]int64{
					"UNCATEGORIZED": 1000,
				},
			},
			expectAmounts:    1,
			expectType:       "TRANSFER_TO",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: true,
			wantErr:          false,
		},
		{
			name: "Infer WITHDRAWAL",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out": -1000,
				},
			},
			expectAmounts:    1,
			expectType:       "WITHDRAWAL",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          false,
		},
		{
			name: "Infer DEPOSIT",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Income Buffer": 1000,
				},
			},
			expectAmounts:    1,
			expectType:       "DEPOSIT",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          false,
		},
		{
			name: "Discard zeroes",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out":    -1000,
					"UNCATEGORIZED": 0,
				},
			},
			expectAmounts:    1,
			expectType:       "WITHDRAWAL",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          false,
		},
		{
			name: "Bad time format",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15T17:00:00Z",
				Amounts: map[string]int64{
					"Dining Out": -1000,
				},
			},
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
		{
			name: "No amounts",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15",
			},
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
		{
			name: "No amounts after discard",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out": 0,
				},
			},
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
		{
			name: "Bad txn splits",
			mockPayload: &UpsertTransactionRqSchema{
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out":       -1000,
					"General Spending": 500,
				},
			},
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedTxn, err := validateTxnInput(tt.mockPayload)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTxn() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				// end test here if we got an error; validatedTxn will be invalid
				return
			}
			if len(validatedTxn.amounts) != tt.expectAmounts {
				t.Errorf("validateTxn() amounts = %v, want %v", len(validatedTxn.amounts), tt.expectAmounts)
			}
			if validatedTxn.txnType != tt.expectType {
				t.Errorf("validateTxn() txnType = %v, want %v", validatedTxn.txnType, tt.expectType)
			}
			if validatedTxn.txnDate != tt.expectDate {
				t.Errorf("validateTxn() txnDate = %v, want %v", validatedTxn.txnDate, tt.expectDate)
			}
			if validatedTxn.isTransfer != tt.expectIsTransfer {
				t.Errorf("validateTxn() isTransfer = %v, want %v", validatedTxn.isTransfer, tt.expectIsTransfer)
			}
		})
	}
}
