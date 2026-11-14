package spiralspring_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spiralspring"
	"github.com/stretchr/testify/require"
)

func TestSpiralSpringParamsDefaults(t *testing.T) {
	dm, err := spiralspring.DefaultParams().Derive()
	require.NoError(t, err)
	require.Equal(t, 5, dm.Turns)
	for name, pair := range map[string][2]float64{
		"Pitch":            {dm.Pitch, 4.0},
		"Gap":              {dm.Gap, 2.4},
		"CenterOffset":     {dm.CenterOffset, 2.0},
		"HubRadius":        {dm.HubRadius, 6},
		"CoilRadius":       {dm.CoilRadius, 30},
		"FirstInnerRadius": {dm.FirstInnerRadius, 10.4},
		"SpokeWidth":       {dm.SpokeWidth, 3.2},
		"BoreRadius":       {dm.BoreRadius, 2.7},
		"BoreFlatDistance": {dm.BoreFlatDistance, 2.2},
		"EyeHoleDiameter":  {dm.EyeHoleDiameter, 3.7},
		"EyeRadius":        {dm.EyeRadius, 3.45},
		"EyeCenterX":       {dm.EyeCenterX, -29.2},
		"FilletRadius":     {dm.FilletRadius, 0.8},
		"TaperOffset":      {dm.TaperOffset, 0.12},
	} {
		require.InDelta(t, pair[1], pair[0], 1e-9, name)
	}
	// L = sqrt((rho_prev + g + R_e)^2 - (r_{2N-1} + t/2 + d)^2) with rho_prev = 28.0 and r_9 = 28.4.
	rootTerm := 28.0 + 2.4 + 3.45
	lateral := 28.4 + 0.8 + 2.0
	wantL := math.Sqrt(rootTerm*rootTerm - lateral*lateral)
	require.InDelta(t, wantL, dm.TailLength, 1e-9)
	require.InDelta(t, -wantL, dm.EyeCenterY, 1e-9)
	wantStrain := 0.8 * (1/6.8 - 1/9.2)
	require.InDelta(t, wantStrain, dm.WindStrain, 1e-9)
	require.InDelta(t, 634.6, dm.ActiveLength, 0.1)
}

func TestSpiralSpringParamsThicknessFloor(t *testing.T) {
	p := spiralspring.DefaultParams()
	p.Thickness = 1.5
	require.NoError(t, p.Validate())
	p.Thickness = 1.49
	require.ErrorContains(t, p.Validate(), "spiralspring: Thickness must be at least 1.5 mm")
}

func TestSpiralSpringParamsTurnBoundary(t *testing.T) {
	p := spiralspring.DefaultParams()
	p.OuterDiameter = 62.4
	dm, err := p.Derive()
	require.NoError(t, err)
	require.Equal(t, 6, dm.Turns)
	require.InDelta(t, 2.0, dm.Gap, 1e-9)
}

func TestSpiralSpringParamsEyeFloor(t *testing.T) {
	p := spiralspring.DefaultParams()
	// The root term of L grows with the coil, so a small coil with a wide eye is the case where 2*R_e wins.
	p.OuterDiameter, p.HubDiameter, p.Thickness, p.MinGap, p.Height = 22, 12, 1.5, 1, 4
	p.AnchorPinDiameter = 10
	dm, err := p.Derive()
	require.NoError(t, err)
	require.InDelta(t, 2*dm.EyeRadius, dm.TailLength, 1e-9)
}

// rootRoom returns the C4 fillet's contact x4 on the spoke's underside, the C3 fillet's contact x3(e) for the
// derived e, and r_f, from the construction's own formulas.
func rootRoom(p spiralspring.Params, dm spiralspring.Dimensions) (x4, x3 float64) {
	a := 2*p.Thickness + dm.FilletRadius
	x4 = -math.Sqrt((dm.HubRadius+dm.FilletRadius)*(dm.HubRadius+dm.FilletRadius) - a*a)
	r := dm.FirstInnerRadius - dm.TaperOffset - dm.FilletRadius
	x3 = dm.CenterOffset + dm.TaperOffset - math.Sqrt(r*r-a*a)
	return x4, x3
}

func TestSpiralSpringParamsRoot(t *testing.T) {
	want := map[string]struct{ rf, e float64 }{
		"default":         {0.8, 0.12},
		"one-turn":        {0.8, 0.12},
		"round-bore":      {0.8, 0.12},
		"turns-override":  {0.8, 0.12},
		"thickness-floor": {0.75, 0.0876},
		"gap-floor":       {0.5, 0.0648},
	}
	for _, c := range springCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			dm, err := p.Derive()
			require.NoError(t, err)
			require.GreaterOrEqual(t, dm.FilletRadius, 0.5)
			require.InDelta(t, math.Min(p.Thickness/2, dm.Gap/2), dm.FilletRadius, 1e-12)
			require.InDelta(t, math.Min(spiralspring.TaperFraction*p.Thickness, dm.TaperOffset), dm.TaperOffset, 1e-12)
			require.Greater(t, dm.TaperOffset, 0.0)
			x4, x3 := rootRoom(p, dm)
			require.GreaterOrEqual(t, x4-x3, spiralspring.TaperMargin-1e-9, "straight underside left between the fillets")
			if w, ok := want[c.name]; ok {
				require.InDelta(t, w.rf, dm.FilletRadius, 1e-9)
				require.InDelta(t, w.e, dm.TaperOffset, 1e-4)
			}
		})
	}
}

func TestSpiralSpringParamsRootRoom(t *testing.T) {
	// A hub this large for the gap leaves no room for two fillets under the spoke (e_room is -0.016).
	p := spiralspring.DefaultParams()
	p.OuterDiameter, p.HubDiameter, p.Thickness, p.MinGap, p.Height = 60, 40, 1.5, 1, 4
	require.ErrorContains(t, p.Validate(), "spiralspring: HubDiameter and Gap leave no room for the root fillets")
}
