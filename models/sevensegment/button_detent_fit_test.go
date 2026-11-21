package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestButtonDetentRealFit(t *testing.T) {
	ctx := t.Context()
	doc := decad.New()
	index, err := buildButtonIndex(ctx, doc, 0)
	require.NoError(t, err)
	detent, err := buildButtonDetent(ctx, doc, 0)
	require.NoError(t, err)
	cam := detentMesh(t, detent.Cam)
	shaft := detentMesh(t, index.Shaft)
	key := detentMesh(t, detent.Key)
	gap, ok, err := cam.Exceeds(shaft, 0)
	require.NoError(t, err)
	require.True(t, ok, "cam and shaft boundary gap %.6f", gap)
	gap, ok, err = cam.Exceeds(key, 0)
	require.NoError(t, err)
	require.True(t, ok, "cam and key boundary gap %.6f", gap)
	a := math.Atan2(24, 10) + 108*math.Pi/180
	b := math.Atan2(20*math.Sin(a)-24, 20*math.Cos(a)-10)
	axis := r3.NewVec(math.Cos(b), 0, math.Sin(b))
	center := r3.NewVec(10, -107, 52)
	for _, station := range []float64{-1.5, 0, 1.5} {
		point := center.Add(axis.Scale(station))
		shaftDistance, err := shaft.PointDistance(point)
		require.NoError(t, err)
		keyDistance, err := key.PointDistance(point)
		require.NoError(t, err)
		require.InDelta(t, 1, shaftDistance, .005+shaft.Bound(), "shaft hole at %.1f", station)
		require.InDelta(t, 1, keyDistance, .005+key.Bound(), "key at %.1f", station)
		inKey, err := detent.Key.LocatePoint(ctx, point, units.Millimeters(.005))
		require.NoError(t, err)
		require.Equal(t, decad.PointInside, inKey, "key at %.1f", station)
		// Each ball reaches the next point without touching the shaft boundary.
		// The final point is outside the shaft, so the whole chain is outside.
		const step = .25
		for x := station; x < 6; x += step {
			distance, err := shaft.PointDistance(center.Add(axis.Scale(x)))
			require.NoError(t, err)
			require.Greater(t, distance, step+shaft.Bound()+1e-8, "shaft path at %.2f", x)
		}
		outside, err := index.Shaft.LocatePoint(ctx, center.Add(axis.Scale(6)), units.Millimeters(.005))
		require.NoError(t, err)
		require.Equal(t, decad.PointOutside, outside, "shaft exterior")
	}
	for _, sign := range []float64{-1, 1} {
		move, err := r3.Translation(axis.Scale(sign * .30))
		require.NoError(t, err)
		// A new key is built for each shift because Transform retires its operand.
		other, err := buildButtonDetent(ctx, decad.New(), 0)
		require.NoError(t, err)
		shifted, err := decad.Transform(ctx, other.Key, move)
		require.NoError(t, err)
		witness := center.Add(axis.Scale(sign * 10.2)).Add(r3.NewVec(0, .2, 0))
		anchor := center.Add(axis.Scale(sign * 11)).Add(r3.NewVec(0, .2, 0))
		near, err := cam.PointDistance(anchor)
		require.NoError(t, err)
		radius := math.Sqrt(anchor.Sub(witness).Dot(anchor.Sub(witness)))
		require.Greater(t, near, radius+cam.Bound()+1e-8)
		t.Logf("sign %g boundary-free witness margin %.9f mm", sign, near-radius-cam.Bound())
		inCam, err := detent.Cam.LocatePoint(ctx, anchor, units.Millimeters(.005))
		require.NoError(t, err)
		inKey, err := shifted.LocatePoint(ctx, witness, units.Millimeters(.01))
		require.NoError(t, err)
		require.Equal(t, decad.PointInside, inCam)
		require.Equal(t, decad.PointInside, inKey)
	}
	r, _ := buttonDetentPitch(0)
	frame, err := r3.NewFrame(r3.NewVec(10+r+.1, -105.75, 52), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	bearing, err := ballbearing.Build(ctx, doc, ballbearing.Params{BoreDiameter: 3, OuterDiameter: 6, Width: 2.5, Placement: &frame})
	require.NoError(t, err)
	bearingMesh := detentMesh(t, bearing.Body)
	gap, ok, err = cam.Exceeds(bearingMesh, 0)
	require.NoError(t, err)
	require.True(t, ok, "cam and bearing boundary gap %.6f", gap)
}
