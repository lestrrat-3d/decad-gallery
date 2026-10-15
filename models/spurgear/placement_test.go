package spurgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// sidePlacement puts the reference point at (10, -5, 7) on a plane whose normal is +X.
func sidePlacement(t *testing.T) *r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	return &f
}

func requirePlacedBounds(t *testing.T, gear *decad.Body, p spurgear.Params) {
	t.Helper()
	box, err := gear.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	origin := p.Placement.Origin()
	axis := p.Placement.N()
	require.InDelta(t, origin.X, box.Min.X, tol)
	require.InDelta(t, origin.X+p.Thickness*axis.X, box.Max.X, tol)
	tip := (p.Module*float64(p.Teeth) + 2*p.Module) / 2
	for _, v := range []struct {
		name           string
		lo, hi, center float64
	}{
		{"Y", box.Min.Y, box.Max.Y, origin.Y},
		{"Z", box.Min.Z, box.Max.Z, origin.Z},
	} {
		require.InDelta(t, v.center, (v.lo+v.hi)/2, 0.1, v.name)
		require.InDelta(t, tip, (v.hi-v.lo)/2, 0.1, v.name)
	}
}

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
