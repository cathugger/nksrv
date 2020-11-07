package legacytrip

import "testing"

func TestMakeTrip(t *testing.T) {
	type tripset struct {
		src  string
		trip string
	}
	var tests = [...]tripset{
		{":^)", "qbhz/q8HqQ"},
		{"a6516a51aaaa", "Om/F889ywA"},
		{"猫に哲学", "tcVgirItgw"},
	}
	for i := range tests {
		trip, err := MakeLegacyTrip(tests[i].src)
		if err != nil {
			t.Errorf("%d got error: %v\n", i, err)
		} else if trip != tests[i].trip {
			t.Errorf("%d expected %q got %q\n", i, tests[i].trip, trip)
		}
	}
}
