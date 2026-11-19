package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// chainDocument is the document the serial spiral route needs: its chained booleans carry a coarser
// held-mesh bound than the pair's own chord tolerance.
func chainDocument() *decad.Document {
	return decad.New(decad.WithChainTolerance(units.Millimeters(0.1)))
}

type serialResult struct {
	volume, bound float64
	xmin, xmax    float64
}

func verifySerialMember(t *testing.T, doc *decad.Document, gear *decad.Body) serialResult {
	t.Helper()
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
	return serialResult{volume: volume.Value.Base(), bound: volume.Bound.Base(), xmin: box.Min.X, xmax: box.Max.X}
}

func serialSpiral() bevelgear.Params {
	p := spiralDefaults()
	p.EnableBore = false
	return p
}

// sourceOrderDefault is the default 31/31 member the source-order route builds, which the serial route's
// default member must match: its volume with the bound the volume reads, and its toe and heel stations.
var sourceOrderDefault = serialResult{volume: 3887.19, bound: 570.8, xmin: 9.9949, xmax: 19.375}
