package spurgear_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestPlacedGearMatchesWorldGear(t *testing.T) {
	p := spurgear.DefaultParams()
	p.BoreDiameter = 3
	p.CapChamfer = 0.1
	world, err := spurgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	worldVolume, err := world.Volume()
	require.NoError(t, err)

	p.Placement = sidePlacement(t)
	doc := decad.New()
	gear, err := spurgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	require.Len(t, doc.Bodies(), 1)
	volume, err := gear.Volume()
	require.NoError(t, err)
	require.InDelta(t, worldVolume.Value.Base(), volume.Value.Base(),
		worldVolume.Bound.Base()+volume.Bound.Base()+0.02)
	requirePlacedBounds(t, gear, p)
}

func TestPlacedEmbeddedGearWithRootFilletsAndBore(t *testing.T) {
	p := spurgear.Params{
		Module: 0.5, Teeth: 60, PressureAngle: 20 * math.Pi / 180,
		Thickness: 2, Samples: 15, BoreDiameter: 3, CapChamfer: 0.02,
	}
	world, err := spurgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	worldVolume, err := world.Volume()
	require.NoError(t, err)

	p.Placement = sidePlacement(t)
	gear, err := spurgear.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
	require.True(t, gear.IsSolid())
	volume, err := gear.Volume()
	require.NoError(t, err)
	require.InDelta(t, worldVolume.Value.Base(), volume.Value.Base(),
		worldVolume.Bound.Base()+volume.Bound.Base()+0.02)
	requirePlacedBounds(t, gear, p)
}

func TestInvalidPlacementIsRejected(t *testing.T) {
	p := spurgear.DefaultParams()
	p.Placement = &r3.Frame{}
	_, err := spurgear.Build(t.Context(), decad.New(), p)
	require.ErrorContains(t, err, "placement")
}
