package config

import "testing"

func TestMaskKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"abcd1234efgh5678", "abcd********5678"},
		{"short", "*****"},
		{"12345678", "********"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := MaskKey(tc.in); got != tc.want {
			t.Errorf("MaskKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
