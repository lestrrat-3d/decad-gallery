package handcranksiren_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireCrankGrip pins the crank clamp's grip on the siren's crank shaft (models/crankhandle instructions.md, "Clamp
// capacity"): the shaft reaches at least 15.5 mm into the arm's hub, carries both flats where the arm's bore clamps it,
// and leaves its second flat out of the rear bearing.
func requireCrankGrip(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	t.Helper()
	shaft, arm := parts["crank-shaft"].Body, parts["crank-arm"].Body
	sb, err := shaft.Bounds()
	require.NoError(t, err)
	ab, err := arm.Bounds()
	require.NoError(t, err)
	require.GreaterOrEqual(t, sb.Max.Y-ab.Min.Y, 15.5, "the shaft's engagement in the arm")
	toLocal, err := handcranksiren.CrankLocal(p)
	require.NoError(t, err)
	flat := crankhandle.Params6082RS().ShaftDiameter/2 - crankhandle.Params6082RS().FlatDepth
	// The section at the arm's outer face, local y of its points: both flats appear.
	o, err := shaft.SectionOutline(t.Context(), atY(t, 46.6), units.Millimeters(roundFitTolerance))
	require.NoError(t, err)
	tol := o.OffSurface.Base() + 1e-3
	up, down := false, false
	for _, l := range o.Loops {
		for _, v := range l {
			y := toLocal.Apply(v).Y
			up = up || math.Abs(y-flat) <= tol
			down = down || math.Abs(y+flat) <= tol
		}
	}
	require.True(t, up, "the shaft's flat toward the pin at Y 46.6")
	require.True(t, down, "the shaft's second flat at Y 46.6")
	// The rear bearing's mid-plane: the second flat ends before it, so the round reaches the bearing's bore.
	o, err = shaft.SectionOutline(t.Context(), atY(t, 31.25), units.Millimeters(roundFitTolerance))
	require.NoError(t, err)
	tol = o.OffSurface.Base() + 1e-3
	lowest := math.Inf(1)
	for _, l := range o.Loops {
		for _, v := range l {
			lowest = math.Min(lowest, toLocal.Apply(v).Y)
		}
	}
	require.InDelta(t, -crankhandle.Params6082RS().ShaftDiameter/2, lowest, tol, "the second flat stays out of the bearing")
}
