package sevensegment_test

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
)

// slideraise is the droop a b-f slider settles to about its groove pin: the model's built-in raise plus RaiseBelowDroop (the
// pin is modeled that much below the droop). It is 0.25 for b and c and 0.30 for e and f.
func slideraise(s sevensegment.Segment) float64 {
	if sevensegment.RaiseOf(s) == 0 {
		return 0
	}
	return sevensegment.RaiseOf(s) + sevensegment.RaiseBelowDroop
}

// meshTolerance is the depth to which a vertex of one mesh may sit inside another before the test calls it penetration: the
// tessellation bound of the two bodies (0.05 each) leaves a vertex a few hundredths inside a surface it touches by design.
const meshTolerance = 0.07

// penetration returns the deepest vertex of verts inside the solid mesh, as the distance from the vertex to the solid's
// surface (gap, posed like the solid). toSolid maps a point to the solid's own frame, where the inside test (pointIn, on
// the solid's crossing grid) runs.
func penetration(verts []r3.Vec, solid *crossingGrid, gap meshgap.Mesh, toSolid func(r3.Vec) r3.Vec) float64 {
	deepest := 0.0
	for _, v := range verts {
		if !solid.pointIn(toSolid(v)) {
			continue
		}
		d, err := gap.PointDistance(v)
		if err == nil && d > deepest {
			deepest = d
		}
	}
	return deepest
}

// pinAndTongue returns the vertices of segment s's slider that lie on its rotor pin and on its tongue: the pin is the part
// beyond the pin face ue (up to 3.6) above z 79, the tongue the part from the print face ua to ue above z 72.9, away from
// the blade.
func pinAndTongue(s sevensegment.Segment, yi float64, m *decad.Mesh) (pin, tongue []r3.Vec) {
	_, _, horizontal := segmentCentreOf(s)
	for _, v := range m.Vertices() {
		if horizontal {
			if v.Z < sevensegment.SliderHeadTop+0.1 || v.Y < yi-Scale*1.0 {
				continue
			}
			switch {
			case v.X > Scale*-1.9+0.05 && v.Z > sevensegment.PinAxisZ-Scale*1.85:
				pin = append(pin, v)
			case v.X <= Scale*-1.9+0.05 && v.X >= -Scale*3.5:
				tongue = append(tongue, v)
			}
			continue
		}
		if v.Z < sevensegment.SliderHeadTop+0.1 || math.Abs(v.X) < Scale*12 {
			continue
		}
		switch {
		case v.Y > yi-Scale*0.2+0.05 && v.Z > sevensegment.PinAxisZ-Scale*1.85:
			pin = append(pin, v)
		case v.Y <= yi-Scale*0.2+0.05 && v.Y >= yi-sevensegment.StemThickness/2:
			tongue = append(tongue, v)
		}
	}
	return pin, tongue
}

// segmentCentreOf reports whether segment s lies along X (a, g, d).
func segmentCentreOf(s sevensegment.Segment) (float64, float64, bool) {
	switch s {
	case sevensegment.SegA, sevensegment.SegG, sevensegment.SegD:
		return 0, 0, true
	}
	return 0, 0, false
}

// pinDistance is the rotor pin's distance from the stem along x for a b-f slider: 15 for b and c, 18 for e and f.
func pinDistance(s sevensegment.Segment) float64 {
	switch s {
	case sevensegment.SegB, sevensegment.SegC:
		return sevensegment.PinStationBC
	case sevensegment.SegE, sevensegment.SegF:
		return sevensegment.PinStationEF
	}
	return 0
}
