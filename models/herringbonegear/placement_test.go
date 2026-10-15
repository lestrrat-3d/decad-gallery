package herringbonegear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/herringbonegear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func smallPlacedParams() herringbonegear.Params {
	p := herringbonegear.DefaultParams()
	p.Teeth, p.Samples, p.Thickness = 8, 5, 2
	p.BoreDiameter, p.CapChamfer = 1, 0.1
	return p
}

func TestPlacedHerringboneGearMatchesWorldGear(t *testing.T) {
	p := smallPlacedParams()
	world, err := herringbonegear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	worldVolume, err := world.Volume()
	require.NoError(t, err)

	for _, tc := range []struct {
		name         string
		u, v         r3.Vec
		wantMin, max float64
	}{
		{"normal_plus_x", r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1), 10, 12},
		{"normal_minus_x", r3.NewVec(0, 0, 1), r3.NewVec(0, 1, 0), 8, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), tc.u, tc.v)
			require.NoError(t, err)
			q := p
			q.Placement = &frame
			doc := decad.New()
			gear, err := herringbonegear.Build(t.Context(), doc, q)
			require.NoError(t, err)
			require.True(t, gear.IsSolid())
			require.Len(t, doc.Bodies(), 1)
			volume, err := gear.Volume()
			require.NoError(t, err)
			require.InDelta(t, worldVolume.Value.Base(), volume.Value.Base(),
				worldVolume.Bound.Base()+volume.Bound.Base()+0.02)
			box, err := gear.Bounds()
			require.NoError(t, err)
			tol := box.Bound.Base() + 1e-8
			require.InDelta(t, tc.wantMin, box.Min.X, tol)
			require.InDelta(t, tc.max, box.Max.X, tol)
			tip := (q.Module*float64(q.Teeth) + 2*q.Module) / 2
			require.InDelta(t, -5, (box.Min.Y+box.Max.Y)/2, 0.1)
			require.InDelta(t, 7, (box.Min.Z+box.Max.Z)/2, 0.1)
			require.InDelta(t, tip, (box.Max.Y-box.Min.Y)/2, 0.1)
			require.InDelta(t, tip, (box.Max.Z-box.Min.Z)/2, 0.1)
		})
	}
}

func TestInvalidPlacementIsRejected(t *testing.T) {
	p := smallPlacedParams()
	p.Placement = &r3.Frame{}
	doc := decad.New()
	_, err := herringbonegear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "placement")
	require.Empty(t, doc.Bodies())
}
