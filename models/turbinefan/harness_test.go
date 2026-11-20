package turbinefan_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestBuildBladeHarness builds the harness for posed blades at 128, 192 and 224 mm and the least, default and greatest
// pitch, and the left hand at 192: a solid on the bed with 14 fully constrained sketches, struts at GateRise in planes
// x = const that clear the wall, a keel above the plate, pins on the keel, and left-hand gates that mirror the right.
func TestBuildBladeHarness(t *testing.T) {
	var right *turbinefan.BladeHarness
	for _, d := range bladeSizes {
		for _, pitch := range bladePitches {
			h := requireHarnessBuild(t, d, pitch, turbinefan.RightHand)
			if d == 192 && pitch == turbinefan.DefaultBladePitch {
				right = h
			}
		}
	}
	left := requireHarnessBuild(t, 192, turbinefan.DefaultBladePitch, turbinefan.LeftHand)
	for i := range right.Gates {
		r, l := right.Gates[i], left.Gates[i]
		// A pin's contact is the lowest point of a section outline, chorded at 0.02 mm, so on a nearly flat bottom (the
		// lying stud) its y and z move by up to the chord's sag between the hands. The trailing-edge gates are analytic.
		tol := 1e-6
		if i >= 6 {
			tol = 0.1
		}
		for _, pair := range [][2]float64{
			{r.Contact.X, l.Contact.X}, {-r.Contact.Y, l.Contact.Y}, {r.Contact.Z, l.Contact.Z},
			{r.Dir.X, l.Dir.X}, {-r.Dir.Y, l.Dir.Y}, {r.Dir.Z, l.Dir.Z},
			{r.Root.X, l.Root.X}, {-r.Root.Y, l.Root.Y}, {r.Root.Z, l.Root.Z},
		} {
			require.InDelta(t, pair[0], pair[1], tol, "gate %d mirrored", i)
		}
	}

	sz := mustSize(t, 192)
	doc, blade := posedBlade(t, 192, turbinefan.DefaultBladePitch, turbinefan.RightHand)
	_, err := turbinefan.BuildBladeHarness(t.Context(), doc, nil, sz, turbinefan.RightHand)
	require.Error(t, err)
	_, err = turbinefan.BuildBladeHarness(t.Context(), doc, blade, turbinefan.Size{}, turbinefan.RightHand)
	require.Error(t, err)
	require.Equal(t, 1, len(doc.Bodies()), "only the posed blade")
}

// TestBladeHarnessVerifyAll tessellates the default-pitch harness at every size with every mesh verification on.
func TestBladeHarnessVerifyAll(t *testing.T) {
	for _, d := range bladeSizes {
		doc, blade := posedBlade(t, d, turbinefan.DefaultBladePitch, turbinefan.RightHand)
		h, err := turbinefan.BuildBladeHarness(t.Context(), doc, blade, mustSize(t, d), turbinefan.RightHand)
		require.NoError(t, err, "rotor %g", d)
		_, err = h.Body.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, "rotor %g", d)
	}
}
