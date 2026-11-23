package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestPrintPose puts blades in the print pose at their rest tilt: the tilt matches the measured value for each size and
// does not change with pitch or hand, the blade's lowest point (the stud flat) sits on the bed and the stud end corner
// the flat cut away StudFlatDepth under it, the posed loft is the same at every pitch, the left hand is the right hand
// reflected in y = 0, the stud axis lies in y = 0 and the pressure face faces -y. Bad inputs are rejected.
func TestPrintPose(t *testing.T) {
	// v5-v7 rested the stud corner on the bed at 0.198526 / 0.206587 / 0.218303; the 0.5 mm stud flat tilts the blade
	// further, to where the cut-away corner lies StudFlatDepth under the leading edge's lowest point. The hexagonal
	// stud reaches less far toward the bed than the round one, so the blade rests at a slightly smaller tilt.
	rest := map[float64]float64{128: 0.236289, 192: 0.232959, 224: 0.240040}
	for _, d := range bladeSizes {
		sz := mustSize(t, d)
		_, ref, tilt, depth := posedBlade(t, d, turbinefan.DefaultBladePitch, turbinefan.RightHand)
		require.InDelta(t, rest[d], tilt, 1e-4, "rotor %g rest tilt", d)
		rb, err := ref.Bounds()
		require.NoError(t, err)
		require.InDelta(t, turbinefan.PrintLift, rb.Min.Z, rb.Bound.Base()+1e-3, "rotor %g lowest point", d)
		corner, _, _ := turbinefan.PosedEdgePoints(sz, turbinefan.RightHand, tilt, depth)
		require.InDelta(t, -depth, corner.Z, 1e-9, "rotor %g stud corner", d)
		_, xf := poseTransform(t, sz, turbinefan.DefaultBladePitch, turbinefan.RightHand, tilt, depth)
		for _, r := range []float64{sz.BladeStudStart, sz.BladeStudEnd} {
			require.InDelta(t, 0, xf.Apply(r3.Vec{Z: r}).Y, 1e-9, "rotor %g stud axis at %g", d, r)
		}
		if d != 192 {
			continue
		}
		for _, pitch := range []float64{turbinefan.MinBladePitch, turbinefan.MaxBladePitch} {
			_, b, ptilt, _ := posedBlade(t, d, pitch, turbinefan.RightHand)
			// The hexagon reaches toward the bed by an amount that turns with the pose's up direction, so the tilt moves
			// with the pitch by under a degree (0.010 rad at 40 degrees, 0.006 at 10).
			require.InDelta(t, tilt, ptilt, 0.015, "pitch %g rest tilt", pitch)
			bb, err := b.Bounds()
			require.NoError(t, err)
			// The tilt's shift with the pitch moves the posed tip by the tilt change times the 100 mm span: 1 to 1.6 mm.
			requireBoxNear(t, rb, bb, 2*math.Max(rb.Bound.Base(), bb.Bound.Base())+2.0, false, "pitch")
		}
		_, left, ltilt, ldepth := posedBlade(t, d, turbinefan.DefaultBladePitch, turbinefan.LeftHand)
		require.InDelta(t, tilt, ltilt, 1e-6, "left hand rest tilt")
		require.InDelta(t, depth, ldepth, 1e-9, "left hand flat depth")
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
	tilt := rest[192]
	_, err := turbinefan.PrintPoseFor(sz, turbinefan.DefaultBladePitch, turbinefan.Hand(2), tilt, turbinefan.StudFlatDepth)
	require.Error(t, err)
	_, err = turbinefan.PrintPoseFor(sz, 9*math.Pi/180, turbinefan.RightHand, tilt, turbinefan.StudFlatDepth)
	require.Error(t, err)
	_, err = turbinefan.PrintPoseFor(turbinefan.Size{}, turbinefan.DefaultBladePitch, turbinefan.RightHand, tilt, turbinefan.StudFlatDepth)
	require.Error(t, err)
	for _, bad := range []float64{turbinefan.RestTiltMin - 1e-3, turbinefan.RestTiltMax + 1e-3, math.NaN()} {
		_, err = turbinefan.PrintPoseFor(sz, turbinefan.DefaultBladePitch, turbinefan.RightHand, bad, turbinefan.StudFlatDepth)
		require.Error(t, err, "tilt %g", bad)
	}
}
