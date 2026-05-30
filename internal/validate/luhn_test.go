package validate

import "testing"

func TestOrderNumber(t *testing.T) {
	tests := []struct {
		number string
		valid  bool
	}{
		{number: "12345678903", valid: true},
		{number: "79927398713", valid: true},
		{number: "12345678900", valid: false},
		{number: "abc", valid: false},
		{number: "", valid: false},
		{number: "12a34", valid: false},
	}

	for _, tc := range tests {
		t.Run(tc.number, func(t *testing.T) {
			if got := OrderNumber(tc.number); got != tc.valid {
				t.Fatalf("OrderNumber(%q) = %v, want %v", tc.number, got, tc.valid)
			}
		})
	}
}
