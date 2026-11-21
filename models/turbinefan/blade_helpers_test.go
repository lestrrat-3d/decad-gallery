package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// bladeSizes are the rotor diameters every blade test covers: the compact, default and large rotors.
var bladeSizes = []float64{128, 192, 224}

// bladePitches are the pitches every blade test covers: the least, the default and the greatest.
var bladePitches = []float64{turbinefan.MinBladePitch, turbinefan.DefaultBladePitch, turbinefan.MaxBladePitch}

func mustSize(t *testing.T, d float64) turbinefan.Size {
	t.Helper()
	sz, err := turbinefan.SizeFor(d)
	require.NoError(t, err)
	return sz
}

// requireThickness compares the ring's extent across the chord direction with its extent along it. The chord runs at
// angle from +X toward -Z.
func requireThickness(t *testing.T, want, angle float64, ring []r3.Vec, tol, y float64) {
	t.Helper()
	along := r3.Vec{X: math.Cos(angle), Z: -math.Sin(angle)}
	across := r3.Vec{X: math.Sin(angle), Z: math.Cos(angle)}
	lo, hi := [2]float64{math.Inf(1), math.Inf(1)}, [2]float64{math.Inf(-1), math.Inf(-1)}
	for _, v := range ring {
		for i, axis := range []r3.Vec{along, across} {
			d := v.X*axis.X + v.Z*axis.Z
			lo[i], hi[i] = math.Min(lo[i], d), math.Max(hi[i], d)
		}
	}
	require.InDelta(t, want, (hi[1]-lo[1])/(hi[0]-lo[0]), tol, "thickness ratio at radius %g", y)
}

// drilledStudArea returns the exact area of the hexagonal stud's section across its axis through the bolt hole's axis:
// the hexagon's area less the band 2 holeRadius wide that the hole cuts across it along the flats' normal. The band lies
// within the hexagon's straight middle, which needs holeRadius <= BladeStudRadius / 2.
func drilledStudArea(sz turbinefan.Size, holeRadius float64) float64 {
	r := sz.BladeStudRadius
	return 3*math.Sqrt(3)/2*r*r - 2*holeRadius*math.Sqrt(3)*r
}

// loopsArea returns the summed area of an outline's loops and the area of the loop whose centroid has the least x.
func loopsArea(o decad.SectionOutline) (float64, float64) {
	total, first, firstX := 0.0, 0.0, math.Inf(1)
	for _, l := range o.Loops {
		a, cen := loopArea(l)
		total += a
		if cen.X < firstX {
			first, firstX = a, cen.X
		}
	}
	return total, first
}

// mustName names a blade case: rotor diameter, pitch in degrees and hand.
func mustName(d, pitch float64, hand turbinefan.Hand) string {
	h := "R"
	if hand == turbinefan.LeftHand {
		h = "L"
	}
	return fmt.Sprintf("%g/%.0f/%s", d, pitch*180/math.Pi, h)
}
