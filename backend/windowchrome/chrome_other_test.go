//go:build !darwin || !cgo

package windowchrome

import "testing"

func TestMeasureReportsNothingWithoutAppKit(t *testing.T) {
	if c, ok := Measure(); ok || c.TitlebarHeight != 0 || c.ControlsEnd != 0 {
		t.Fatalf("Measure() = %+v, %v; want nothing", c, ok)
	}
}
