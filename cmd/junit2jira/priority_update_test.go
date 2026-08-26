package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePriorityThresholds(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    []int
		expectError bool
	}{
		{
			name:     "valid default thresholds",
			input:    "4,16,64,128,256",
			expected: []int{4, 16, 64, 128, 256},
		},
		{
			name:     "valid custom thresholds",
			input:    "10,50,100,200,400",
			expected: []int{10, 50, 100, 200, 400},
		},
		{
			name:     "valid with spaces",
			input:    " 4 , 16 , 64 , 128 , 256 ",
			expected: []int{4, 16, 64, 128, 256},
		},
		{
			name:        "invalid - too few values",
			input:       "4,16,64",
			expected:    defaultPriorityThresholds,
			expectError: true,
		},
		{
			name:        "invalid - too many values",
			input:       "4,16,64,128,256,512",
			expected:    defaultPriorityThresholds,
			expectError: true,
		},
		{
			name:        "invalid - non-numeric value",
			input:       "4,16,abc,128,256",
			expected:    defaultPriorityThresholds,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    defaultPriorityThresholds,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePriorityThresholds(tt.input)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
