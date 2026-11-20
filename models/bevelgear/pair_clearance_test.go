package bevelgear_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestFilletedPinionRootClearsMateTips turns the fan's pair (module 2, 28 teeth driving a 14-tooth pinion) through
// one pinion tooth pitch in 8 steps and proves a positive gap, in three dimensions, between the pinion's filleted
// roots and the driving gear's tips.
//
// The pinion's root region is its triangles between the toe root and the heel pitch point whose centroid lies less
// than 1.3 fillet radii above the root cone (the radius of the fillet scales with the station). The driving gear's tip
// region is its triangles above the pitch cone over the same stations. Whole bodies touch at the flanks, which have no
// backlash, so only these two regions are compared. meshgap charges both meshes' held displacement bounds. The
// driving gear turns by -z_pinion/z_driving of the pinion's angle about +X, the pinion about +Y: at the pitch point
// the two surface speeds are equal and opposite in sign.
func TestFilletedPinionRootClearsMateTips(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Module, p.Teeth, p.MateTeeth, p.SpiralAngle = 2, 28, 14, 0
	p.EnableBore = false
	pinionP := p
	pinionP.Teeth, pinionP.MateTeeth = p.MateTeeth, p.Teeth
	pinionBand, err := pinionP.ToothBand()
	require.NoError(t, err)
	require.Positive(t, pinionBand.RootFillet)
	drivingBand, err := p.ToothBand()
	require.NoError(t, err)

	pair, err := bevelgear.BuildPair(t.Context(), decad.New(), p)
	require.NoError(t, err)
	pinionMesh, err := pair.Pinion.Tessellate(t.Context(), units.Millimeters(0.05))
	require.NoError(t, err)
	drivingMesh, err := pair.Driving.Tessellate(t.Context(), units.Millimeters(0.05))
	require.NoError(t, err)

	pinionAxis, drivingAxis := r3.Vec{Y: 1}, r3.Vec{X: 1}
	pinionRoot := func(c r3.Vec) bool {
		s := c.Y
		radius := math.Hypot(c.X, c.Z)
		fillet := pinionBand.RootFillet * s / pinionBand.HeelPitch
		return s > pinionBand.ToeRoot-0.2 && s < pinionBand.HeelPitch && radius < pinionBand.RootSlope*s+1.3*fillet
	}
	drivingTip := func(c r3.Vec) bool {
		s := c.X
		radius := math.Hypot(c.Y, c.Z)
		return s > drivingBand.ToeRoot-0.2 && s < drivingBand.HeelPitch && radius > drivingBand.PitchSlope*s
	}
	pitch := 2 * math.Pi / float64(pinionP.Teeth)
	ratio := float64(pinionP.Teeth) / float64(p.Teeth)
	worst := math.Inf(1)
	for k := range 8 {
		angle := float64(k) * pitch / 8
		root := regionMesh(t, pinionMesh, pinionAxis, angle, pinionRoot)
		tips := regionMesh(t, drivingMesh, drivingAxis, -angle*ratio, drivingTip)
		// Find the largest clearance the proof reaches, which is a lower bound on the gap.
		lower := 0.0
		for _, c := range []float64{0.02, 0.05, 0.08, 0.1, 0.12, 0.15} {
			gap, ok, err := root.Exceeds(tips, c)
			require.NoError(t, err)
			if !ok {
				break
			}
			lower = max(lower, gap)
		}
		t.Logf("pose %d: proven gap at least %.3f mm", k, lower)
		worst = min(worst, lower)
	}
	require.Greater(t, worst, 0.2, "the root fillets clear the mate's tips by a proven gap")
}
