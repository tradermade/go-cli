package cmd

import "testing"

func TestLadderLevels(t *testing.T) {
	bids := [][]string{
		{"1.1421400", "5400000"},
		{"1.1421300", "16000000"},
	}
	want := "1.1421400 x 5400000   1.1421300 x 16000000"
	if got := ladderLevels(bids); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	// Capped at five levels.
	six := [][]string{
		{"1", "1"}, {"2", "2"}, {"3", "3"}, {"4", "4"}, {"5", "5"}, {"6", "6"},
	}
	if got := ladderLevels(six); got != "1 x 1   2 x 2   3 x 3   4 x 4   5 x 5" {
		t.Errorf("cap failed: %q", got)
	}

	if got := ladderLevels(nil); got != "-" {
		t.Errorf("empty: %q", got)
	}
}

func TestShortNum(t *testing.T) {
	if got := shortNum("1.1425399999999999"); got != "1.14254" {
		t.Errorf("artifact not cleaned: %q", got)
	}
	if got := shortNum("1.14253"); got != "1.14253" {
		t.Errorf("clean value changed: %q", got)
	}
	if got := shortNum("not-a-number"); got != "not-a-number" {
		t.Errorf("non-numeric changed: %q", got)
	}
}
