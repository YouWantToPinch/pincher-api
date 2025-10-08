package server

import (
	"testing"
)

func Test_ToString(t *testing.T) {
	cases := []struct {
		input    Cent
		expected string
	}{
		{
			input:    20000,
			expected: "200.00",
		},
		{
			input:    50000 + 139,
			expected: "501.39",
		},
	}

	for _, c := range cases {
		displayStr := c.input.Display()
		if displayStr != c.expected {
			t.Errorf("ERROR: expected string %s, but got string: %s", c.expected, displayStr)
			t.Fail()
		}
	}
}
