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
		"FilletRadius":     {dm.FilletRadius, 0.6},
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
