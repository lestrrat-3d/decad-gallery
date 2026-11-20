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
// pitch, and the left hand at 192: a solid on the bed with 23 fully constrained sketches, two cradle walls from the stud
// end past the tip gate, struts at GateRise in planes x = const that clear the pressure wall, tabs that widen clear of
// their walls, a keel above the plate, pins on the keel, and left-hand gates that mirror the right.
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
		// Struts are analytic. A tab's or a pin's contact is read from a section outline chorded at 0.02 mm, so on a
		// nearly flat face it moves between the hands by up to the chord's sag.
		contact, dir, root := [3]float64{1e-6, 1e-6, 1e-6}, 1e-6, [3]float64{1e-6, 1e-6, 1e-6}
		switch {
		case i >= 12:
			contact, dir, root = [3]float64{0.1, 0.1, 0.1}, 0.1, [3]float64{0.1, 0.1, 0.1}
		case i >= 6:
			contact, dir, root = [3]float64{0.01, 0.01, 0.15}, 1e-9, [3]float64{0.01, 0.05, 0.15}
		}
		for j, c := range []struct {
			want, got, tol float64
		}{
			{r.Contact.X, l.Contact.X, contact[0]}, {-r.Contact.Y, l.Contact.Y, contact[1]},
			{r.Contact.Z, l.Contact.Z, contact[2]},
			{r.Dir.X, l.Dir.X, dir}, {-r.Dir.Y, l.Dir.Y, dir}, {r.Dir.Z, l.Dir.Z, dir},
			{r.Root.X, l.Root.X, root[0]}, {-r.Root.Y, l.Root.Y, root[1]}, {r.Root.Z, l.Root.Z, root[2]},
		} {
			require.InDelta(t, c.want, c.got, c.tol, "gate %d field %d mirrored", i, j)
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
