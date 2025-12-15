package api

import (
	"testing"
)

func TestParseBoolFromString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectCleared bool
		wantErr       bool
	}{
		{
			name:          "Truthy string is true",
			input:         "true",
			expectCleared: true,
			wantErr:       false,
		},
		{
			name:          "Falsy string is false",
			input:         "false",
			expectCleared: false,
			wantErr:       false,
		},
		{
			name:    "Empty gives error",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Int 0 does not parse",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "Int 1 does not parse",
			input:   "1",
			wantErr: true,
		},
		{
			name:    "Random string does not parse",
			input:   "msej8132cfxIUWKM",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseBoolFromString(tt.input)
			if result != tt.expectCleared {
				t.Errorf("parseBoolFromString() result = %v, want %v", result, tt.expectCleared)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBoolFromString() err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
