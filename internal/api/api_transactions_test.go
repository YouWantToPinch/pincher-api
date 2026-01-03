package api

import (
	"testing"
	"time"
)

func TestValidateTXN(t *testing.T) {
	tests := []struct {
		name             string
		mockPayload      *LogTransactionrqSchema
		expectCleared    bool
		expectAmounts    int
		expectType       string
		expectDate       time.Time
		expectIsTransfer bool
		wantErr          bool
	}{
		{
			name: "Infer TRANSFER_FROM",
			mockPayload: &LogTransactionrqSchema{
				Cleared:           "true",
				TransactionDate:   "2025-09-15",
				TransferAccountID: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
				Amounts: map[string]int64{
					"UNCATEGORIZED": -1000,
				},
			},
			expectCleared:    true,
			expectAmounts:    1,
			expectType:       "TRANSFER_FROM",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: true,
			wantErr:          false,
		},
		{
			name: "Infer TRANSFER_TO",
			mockPayload: &LogTransactionrqSchema{
				Cleared:           "true",
				TransactionDate:   "2025-09-15",
				TransferAccountID: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
				Amounts: map[string]int64{
					"UNCATEGORIZED": 1000,
				},
			},
			expectCleared:    true,
			expectAmounts:    1,
			expectType:       "TRANSFER_TO",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: true,
			wantErr:          false,
		},
		{
			name: "Infer WITHDRAWAL",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out": -1000,
				},
			},
			expectCleared:    true,
			expectAmounts:    1,
			expectType:       "WITHDRAWAL",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          false,
		},
		{
			name: "Infer DEPOSIT",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Income Buffer": 1000,
				},
			},
			expectCleared:    true,
			expectAmounts:    1,
			expectType:       "DEPOSIT",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          false,
		},
		{
			name: "Discard zeroes",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out":    -1000,
					"UNCATEGORIZED": 0,
				},
			},
			expectCleared:    true,
			expectAmounts:    1,
			expectType:       "WITHDRAWAL",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          false,
		},
		{
			name: "Bad time format",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15T17:00:00Z",
				Amounts: map[string]int64{
					"Dining Out": -1000,
				},
			},
			expectCleared:    false,
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
		{
			name: "No amounts",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15",
			},
			expectCleared:    true,
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
		{
			name: "No amounts after discard",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out": 0,
				},
			},
			expectCleared:    true,
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
		{
			name: "Bad txn splits",
			mockPayload: &LogTransactionrqSchema{
				Cleared:         "true",
				TransactionDate: "2025-09-15",
				Amounts: map[string]int64{
					"Dining Out":       -1000,
					"General Spending": 500,
				},
			},
			expectCleared:    true,
			expectAmounts:    0,
			expectType:       "NONE",
			expectDate:       time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			expectIsTransfer: false,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleared, amounts, txnType, txnDate, isTransfer, err := validateTxn(tt.mockPayload)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTxn() error = %v, wantErr %v", err, tt.wantErr)
			}
			if cleared != tt.expectCleared {
				t.Errorf("validateTxn() cleared = %v, want %v", cleared, tt.expectCleared)
			}
			if len(amounts) != tt.expectAmounts {
				t.Errorf("validateTxn() amounts = %v, want %v", len(amounts), tt.expectAmounts)
			}
			if txnType != tt.expectType {
				t.Errorf("validateTxn() txnType = %v, want %v", txnType, tt.expectType)
			}
			if txnDate != tt.expectDate {
				t.Errorf("validateTxn() txnDate = %v, want %v", txnDate, tt.expectDate)
			}
			if isTransfer != tt.expectIsTransfer {
				t.Errorf("validateTxn() isTransfer = %v, want %v", isTransfer, tt.expectIsTransfer)
			}
		})
	}
}
