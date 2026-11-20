package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestPrintPose puts blades in the print pose: the posed loft is the same at every pitch, the left hand is the right
// hand reflected in y = 0, the stud corner is the lowest point at PrintLift, the stud axis lies in y = 0 and the pressure
// face faces -y. Bad inputs are rejected.
func TestPrintPose(t *testing.T) {
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		_, ref := posedBlade(t, d, turbinefan.DefaultBladePitch, turbinefan.RightHand)
		rb, err := ref.Bounds()
		require.NoError(t, err)
		require.InDelta(t, turbinefan.PrintLift, rb.Min.Z, rb.Bound.Base()+1e-3, "rotor %g lowest point", d)
		_, xf := poseTransform(t, sz, turbinefan.DefaultBladePitch, turbinefan.RightHand)
		for _, r := range []float64{sz.BladeStudStart, sz.BladeStudEnd} {
			require.InDelta(t, 0, xf.Apply(r3.Vec{Z: r}).Y, 1e-9, "rotor %g stud axis at %g", d, r)
		}
		if d != 192 {
			continue
		}
		for _, pitch := range []float64{turbinefan.MinBladePitch, turbinefan.MaxBladePitch} {
			_, b := posedBlade(t, d, pitch, turbinefan.RightHand)
			bb, err := b.Bounds()
			require.NoError(t, err)
			requireBoxNear(t, rb, bb, 2*math.Max(rb.Bound.Base(), bb.Bound.Base())+1e-3, false, "pitch")
		}
		_, left := posedBlade(t, d, turbinefan.DefaultBladePitch, turbinefan.LeftHand)
		lb, err := left.Bounds()
		require.NoError(t, err)
		requireBoxNear(t, rb, lb, rb.Bound.Base()+lb.Bound.Base()+1e-3, true, "left hand")
		rv, err := ref.Volume()
		require.NoError(t, err)
		lv, err := left.Volume()
		require.NoError(t, err)
		require.InDelta(t, rv.Value.Base(), lv.Value.Base(), 2e-3*rv.Value.Base())
		r7, _ := turbinefan.StationRadiusChord(sz, 7)
		pressure, suction := turbinefan.SectionSurfacePoints(sz, 7)
		p := xf.Apply(r3.Vec{X: pressure[0], Y: pressure[1], Z: r7})
		s := xf.Apply(r3.Vec{X: suction[0], Y: suction[1], Z: r7})
		require.Less(t, p.Y, s.Y, "the pressure face faces -y")
	}

	sz := mustSize(t, 192)
	_, err := turbinefan.PrintPoseFor(sz, turbinefan.DefaultBladePitch, turbinefan.Hand(2))
	require.Error(t, err)
	_, err = turbinefan.PrintPoseFor(sz, 9*math.Pi/180, turbinefan.RightHand)
	require.Error(t, err)
	_, err = turbinefan.PrintPoseFor(turbinefan.Size{}, turbinefan.DefaultBladePitch, turbinefan.RightHand)
	require.Error(t, err)
}
