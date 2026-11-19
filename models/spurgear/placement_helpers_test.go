package spurgear_test

import (
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
