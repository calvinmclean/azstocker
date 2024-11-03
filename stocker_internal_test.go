package stocker

import (
	"testing"
	"time"
)

func TestParseMonth(t *testing.T) {
	month := func(m time.Month) *time.Month {
		return &m
	}

	tests := []struct {
		input    string
		expected *time.Month
	}{
		{"OCTOBER", month(time.October)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			out := parseMonth(tt.input)
			if tt.expected == nil && out != nil {
				t.Errorf("expected nil but got %v", *out)
			}
			if tt.expected != nil && out == nil {
				t.Errorf("expected %v but got nil", *tt.expected)
			}
			if *tt.expected != *out {
				t.Errorf("expected %v but got %v", *tt.expected, *out)
			}
		})
	}
}
