package spurgear

import "testing"

func TestParamsPositionalFieldOrder(t *testing.T) {
	p := Params{1, 2, 3, 4, 5, 6, 7, 8, nil}
	if p.Module != 1 || p.PressureAngle != 2 || p.Thickness != 3 || p.Teeth != 4 || p.Samples != 5 ||
		p.BoreDiameter != 6 || p.CapChamfer != 7 || p.Backlash != 8 || p.Placement != nil {
		t.Fatal("Params positional field order changed")
	}
}
