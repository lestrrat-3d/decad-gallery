package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestVirtualToothProfiles(t *testing.T) {
	for _, tc := range []struct {
		name      string
		teeth     int
		mateTeeth int
		wantCount float64
		wantGamma float64
	}{
		{name: "small equal pair", teeth: 8, mateTeeth: 8,
			wantCount: 8 * math.Sqrt2, wantGamma: math.Pi / 4},
		{name: "unequal pair", teeth: 16, mateTeeth: 12,
			wantCount: 16 / 0.6, wantGamma: math.Atan2(4, 3)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := bevelgear.DefaultParams()
			p.Teeth, p.MateTeeth, p.Samples = tc.teeth, tc.mateTeeth, 5
			world := sketch.NewWorld()
			_, profile, err := bevelgear.BuildVirtualToothProfile(t.Context(), world, world.XY(), p)
			require.NoError(t, err)
			require.True(t, profile.Valid)
			require.Empty(t, profile.Holes)
			require.Len(t, profile.Outer, 6)

			pitch := float64(tc.teeth) / (2 * math.Cos(tc.wantGamma))
			tip := pitch + 1
			root := pitch - 1.3625
			arcRadii := make([]float64, 0, 2)
			fitted := 0
			pressure := p.PressureAngle
			wantBaseAngle := math.Pi/(2*tc.wantCount) + math.Tan(pressure) - pressure
			for _, edge := range profile.Outer {
				switch entity := edge.Entity.(type) {
				case *sketch.Arc:
					arcRadii = append(arcRadii, entity.R())
				case *sketch.FitSpline:
					fitted++
					require.Len(t, entity.Fit, p.Samples)
					basePoint := entity.Fit[0]
					require.InDelta(t, pitch*math.Cos(pressure),
						math.Hypot(basePoint.X(), basePoint.Y()), 1e-8)
					actualFromToothCenter := math.Abs(math.Atan2(math.Sin(math.Atan2(basePoint.Y(), basePoint.X())-math.Pi),
						math.Cos(math.Atan2(basePoint.Y(), basePoint.X())-math.Pi)))
					require.InDelta(t, wantBaseAngle, actualFromToothCenter, 1e-8,
						"the flank must use the real virtual tooth count")
				}
			}
			require.Equal(t, 2, fitted)
			require.Len(t, arcRadii, 2)
			require.InDelta(t, root, math.Min(arcRadii[0], arcRadii[1]), 1e-8)
			require.InDelta(t, tip, math.Max(arcRadii[0], arcRadii[1]), 1e-8)
		})
	}
}

func TestEmbeddedVirtualToothProfile(t *testing.T) {
	p := bevelgear.DefaultParams()
	p.Teeth, p.MateTeeth = 60, 60
	world := sketch.NewWorld()
	_, profile, err := bevelgear.BuildVirtualToothProfile(t.Context(), world, world.XY(), p)
	require.NoError(t, err)
	require.True(t, profile.Valid)
	require.Empty(t, profile.Holes)
	require.Len(t, profile.Outer, 4)

	pitch := float64(p.Teeth) / (2 * math.Cos(math.Pi/4))
	root := pitch - 1.3625
	trimmed := 0
	for _, edge := range profile.Outer {
		flank, ok := edge.Entity.(*sketch.FitSpline)
		if !ok {
			continue
		}
		require.True(t, edge.Partial)
		require.True(t, edge.TExact)
		x, y := flank.Eval(edge.TStart)
		require.InDelta(t, root, math.Hypot(x, y), 1e-8)
		trimmed++
	}
	require.Equal(t, 2, trimmed)
}
