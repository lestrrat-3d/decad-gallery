package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralToothEightParallelSlices(t *testing.T) {
	p := DefaultParams()
	p.SpiralAngle = 35 * math.Pi / 180
	p.SpiralHand = "right"
	doc := decad.New()
	straight := p
	straight.SpiralAngle = 0
	tooth, err := BuildPointTooth(t.Context(), doc, straight)
	require.NoError(t, err)
	slabs, frame, err := sliceSpiralPointTooth(t.Context(), doc, p, tooth)
	require.NoError(t, err)
	require.Len(t, slabs, 8)
	require.Len(t, doc.Bodies(), 8)
	require.Positive(t, frame.span)
	require.Greater(t, frame.heel, frame.toe)
	parentU := r3.NewVec(math.Sin(frame.pitchAngle), -math.Cos(frame.pitchAngle), 0)
	parentNormal := parentU.Cross(r3.NewVec(0, 0, 1))
	drivingTrace, err := buildSpiralTrace(frame, p, false)
	require.NoError(t, err)
	pinionTrace, err := buildSpiralTrace(frame, p, true)
	require.NoError(t, err)
	require.Positive(t, drivingTrace.total)
	require.InDelta(t, drivingTrace.total, pinionTrace.total, 1e-12)
	for _, endpoints := range [][2][2]float64{
		{drivingTrace.toe, pinionTrace.toe},
		{drivingTrace.heel, pinionTrace.heel},
	} {
		require.InDelta(t, endpoints[0][0], endpoints[1][0], 1e-12)
		require.InDelta(t, endpoints[0][1], -endpoints[1][1], 1e-12)
	}
	for i, slab := range slabs {
		require.True(t, slab.IsSolid(), "slab %d", i)
		parallelCaps := 0
		for _, face := range slab.Faces() {
			plane, ok := face.Surface().(decad.Plane)
			if ok && math.Abs(plane.Frame.N().Dot(parentNormal)) > 1-1e-10 {
				parallelCaps++
			}
		}
		require.Positive(t, parallelCaps, "slab %d has no parent-plane-parallel cap", i)
		mesh, err := slab.Tessellate(t.Context(), units.Millimeters(0.1),
			decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, "slab %d", i)
		require.True(t, mesh.BoundaryVerified(), "slab %d", i)
		require.True(t, mesh.VolumeVerified(), "slab %d", i)
		if i > 0 {
			previous, err := slabs[i-1].Centroid()
			require.NoError(t, err)
			current, err := slab.Centroid()
			require.NoError(t, err)
			require.Less(t, previous.Value.Dot(frame.coneVec), current.Value.Dot(frame.coneVec))
		}
	}
}
