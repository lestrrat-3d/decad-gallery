package turbinefan_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestStudFlat builds blades at every size and pitch, and the left hand at 192, and poses each at its rest tilt. The
// stud's flat lies on the bed: a planar face facing straight down at z = 0 of 2 mm² or more, and a stud island in
// the first layer (z = 0.1) of 2.9 mm² or more: a hexagon flat faces the bed at the low pitches, and a corner nears it at
// 40 degrees, which leaves the least flat. At 12 and 15 degrees the flat is shallower than StudFlatDepth where that depth
// would tilt the blade too far to leave the root clear. The stud's section through the bolt hole keeps the hexagonal stud's area, and the round root stays StudFlatRootClearance or more over the bed.
func TestStudFlat(t *testing.T) {
	type flatCase struct {
		d     float64
		pitch float64
		hand  turbinefan.Hand
	}
	var cases []flatCase
	for _, d := range bladeSizes {
		for _, pitch := range bladePitches {
			cases = append(cases, flatCase{d, pitch, turbinefan.RightHand})
		}
		// A flat StudFlatDepth deep leaves the round root of the 128 mm blade under StudFlatRootClearance over the bed at
		// every pitch from 11 to 19 degrees (0.14 mm at 15), so the flat is shallower there.
		for _, deg := range []float64{12, 15} {
			cases = append(cases, flatCase{d, deg * math.Pi / 180, turbinefan.RightHand})
		}
	}
	cases = append(cases, flatCase{192, turbinefan.DefaultBladePitch, turbinefan.LeftHand})
	for _, c := range cases {
		sz := mustSize(t, c.d)
		build := turbinefan.BuildBlade
		if c.hand == turbinefan.LeftHand {
			build = turbinefan.BuildBladeMirrored
		}
		blade, err := build(t.Context(), turbinefan.NewDocument(), c.pitch, sz)
		require.NoError(t, err)
		tilt, depth, err := turbinefan.RestTilt(t.Context(), blade.Body, sz, c.pitch, c.hand)
		require.NoError(t, err)
		_, xf := poseTransform(t, sz, c.pitch, c.hand, tilt, depth)
		name := func(what string) string { return what + " at rotor " + mustName(c.d, c.pitch, c.hand) }

		// The round root's lowest point, read from the unposed blade's mesh by its local radius.
		mesh, err := blade.Body.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
		require.NoError(t, err)
		root := math.Inf(1)
		for _, v := range mesh.Vertices() {
			if v.Z > sz.RootRadius-0.01 && v.Z < sz.RootRadius+2*sz.HubScale {
				root = math.Min(root, xf.Apply(v).Z)
			}
		}

		// The stud's section across its axis through the bolt hole's axis, before posing.
		bolt, err := r3.NewFrame(r3.Vec{Z: sz.BladeBoltAt}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		require.NoError(t, err)
		studArea, _ := loopsArea(section(t, blade.Body, bolt))

		corner, _, _ := turbinefan.PosedEdgePointsAt(sz, c.pitch, c.hand, tilt, depth)
		copied, err := decad.Copy(t.Context(), blade.Body)
		require.NoError(t, err)
		posed, err := decad.Transform(t.Context(), copied, xf)
		require.NoError(t, err)
		total, stud := loopsArea(sectionAt(t, posed, 0.1))
		flat := 0.0
		for _, f := range posed.Faces() {
			if f.Surface().Kind() != decad.KindPlane {
				continue
			}
			// Most faces report no centroid, so each planar face is read at its point nearest the stud end corner the
			// flat cut away, which lies depth under the flat's end.
			cp, err := f.ClosestPoint(t.Context(), corner, units.Millimeters(0.02))
			require.NoError(t, err)
			n, err := f.NormalAt(cp.Point)
			require.NoError(t, err)
			if n.Value.Sub(r3.Vec{Z: -1}).Len() > 1e-6 || math.Abs(cp.Point.Z) > 0.02 {
				continue
			}
			a, err := f.Area()
			require.NoError(t, err)
			flat = math.Max(flat, a.Value.Base())
		}
		t.Logf("%s: rest tilt %.6f rad (%.3f°), flat depth %.3f, flat face %.2f mm², stud island %.2f mm², first layer %.2f mm², root "+
			"%.3f mm over the bed, stud section at the bolt line %.3f mm² (round %.3f)", mustName(c.d, c.pitch, c.hand),
			tilt, tilt*180/math.Pi, depth, flat, stud, total, root, studArea, drilledStudArea(sz, m2fastener.HoleRadius))
		require.LessOrEqual(t, depth, turbinefan.StudFlatDepth, name("flat depth"))
		require.GreaterOrEqual(t, flat, 2.0, name("flat face"))
		require.GreaterOrEqual(t, stud, 2.9, name("stud island"))
		require.GreaterOrEqual(t, root, turbinefan.StudFlatRootClearance, name("root clearance"))
		require.InDelta(t, drilledStudArea(sz, m2fastener.HoleRadius), studArea, 0.05, name("stud section"))
	}
}

// TestStudFlatRootClearance requires the rest tilt to fail when the round root would stand under
// StudFlatRootClearance over the bed: a 1.0 mm flat at 128 leaves the root 0.06 mm up.
func TestStudFlatRootClearance(t *testing.T) {
	sz := mustSize(t, 128)
	blade, err := turbinefan.BuildBlade(t.Context(), turbinefan.NewDocument(), turbinefan.DefaultBladePitch, sz)
	require.NoError(t, err)
	mesh, err := blade.Body.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	require.NoError(t, err)
	_, err = turbinefan.RestTiltAtDepth(mesh.Vertices(), sz, turbinefan.DefaultBladePitch, turbinefan.RightHand,
		turbinefan.StudFlatDepth)
	require.NoError(t, err, "the default depth")
	_, err = turbinefan.RestTiltAtDepth(mesh.Vertices(), sz, turbinefan.DefaultBladePitch, turbinefan.RightHand, 1.0)
	require.ErrorContains(t, err, "root stands")
}
