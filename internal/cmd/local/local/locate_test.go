package local

import "testing"

func TestLocateChartFlag(t *testing.T) {
	expect := "chartFlagValue"
	c := locateLatestAirbyteChart("airbyte", "", expect)
	if c != expect {
		t.Errorf("expected %q but got %q", expect, c)
	}
}
