package style

import "testing"

func TestDisplayWidth(t *testing.T) {
	tests := []struct {
		value string
		width int
	}{
		{"", 0},
		{"curl #1", 7},
		{"🐌", 2},
		{"🏁", 2},
		{"性能テスト", 10},
		{"한글", 4},
		{"café", 4},
		{"a‍b", 2},
	}
	for _, test := range tests {
		if got := DisplayWidth(test.value); got != test.width {
			t.Errorf("DisplayWidth(%q) = %d, want %d", test.value, got, test.width)
		}
	}
}
