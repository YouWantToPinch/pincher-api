package api

import (
	"testing"
	"time"
)

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
				TransferAccountName: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
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
				TransferAccountName: "f81d4fae-7dec-11d0-a765-00a0c91e6bf6",
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
