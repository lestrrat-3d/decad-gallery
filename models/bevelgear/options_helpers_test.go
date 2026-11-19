package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

type memberResult struct {
	volume float64
	bound  float64
	xmin   float64
	xmax   float64
	ymax   float64
}

func buildStraightMemberFor(t *testing.T, p bevelgear.Params) memberResult {
	t.Helper()
	doc := decad.New()
	gear, err := bevelgear.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 1)
	require.True(t, gear.IsSolid())
	mesh, err := gear.Tessellate(t.Context(), units.Millimeters(0.1), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	require.Equal(t, 2, meshEuler(mesh), "the unbored member has one closed outer shell")
	volume, err := gear.Volume()
	require.NoError(t, err)
	box, err := gear.Bounds()
	require.NoError(t, err)
	return memberResult{volume: volume.Value.Base(), bound: volume.Bound.Base(),
		xmin: box.Min.X, xmax: box.Max.X, ymax: box.Max.Y}
}

func optionBase() bevelgear.Params {
	return bevelgear.Params{Module: 1, Teeth: 8, MateTeeth: 8, ShaftAngle: math.Pi / 2,
		PressureAngle: 20 * math.Pi / 180, Samples: 5}
}
