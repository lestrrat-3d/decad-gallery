package fastenertool_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/fastenertool"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func frame(t *testing.T) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: 3}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}

func TestCylinder(t *testing.T) {
	body, sk, err := fastenertool.Cylinder(t.Context(), decad.New(), frame(t), 2, 5)
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))
	box, err := body.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 3, box.Min.Z, 1e-3)
	require.InDelta(t, 8, box.Max.Z, 1e-3)
	require.InDelta(t, 2, box.Max.X, 1e-3)
	require.InDelta(t, -2, box.Min.Y, 1e-3)
}

func TestHexPrism(t *testing.T) {
	const radius = 3.0
	body, sk, err := fastenertool.HexPrism(t.Context(), decad.New(), frame(t), radius, 4)
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))
	box, err := body.Bounds()
	require.NoError(t, err)
	// The first corner lies on U (+X); the flats face V (+Y).
	require.InDelta(t, radius, box.Max.X, 1e-3)
	require.InDelta(t, -radius, box.Min.X, 1e-3)
	require.InDelta(t, radius*math.Sqrt(3)/2, box.Max.Y, 1e-3)
	require.InDelta(t, 3, box.Min.Z, 1e-3)
	require.InDelta(t, 7, box.Max.Z, 1e-3)
	vol, err := body.Volume()
	require.NoError(t, err)
	require.InEpsilon(t, 3*math.Sqrt(3)/2*radius*radius*4, vol.Value.Base(), 1e-3)
}

func TestRoundedRevolve(t *testing.T) {
	const radius, round = 2.0, 0.5
	pts := [][2]float64{{0, 0}, {0, radius}}
	pts = append(pts, fastenertool.QuarterRound(6-round, radius-round, round, 0, 8)...)
	pts = append(pts, [2]float64{6, 0})
	body, sk, err := fastenertool.RoundedRevolve(t.Context(), decad.New(), frame(t), pts)
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), sk))
	box, err := body.Bounds()
	require.NoError(t, err)
	// The axis is U (+X) from the frame origin.
	require.InDelta(t, 0, box.Min.X, 1e-3)
	require.InDelta(t, 6, box.Max.X, 1e-3)
	require.InDelta(t, radius, box.Max.Y, 1e-3)
	require.InDelta(t, 3+radius, box.Max.Z, 1e-3)
}

func TestQuarterRound(t *testing.T) {
	far := fastenertool.QuarterRound(10, 1, 0.5, 0, 8)
	require.Len(t, far, 9)
	require.InDelta(t, 10, far[0][0], 1e-9)
	require.InDelta(t, 1.5, far[0][1], 1e-9)
	require.InDelta(t, 10.5, far[8][0], 1e-9)
	require.InDelta(t, 1, far[8][1], 1e-9)
	near := fastenertool.QuarterRound(10, 1, 0.5, math.Pi, 8)
	require.InDelta(t, 9.5, near[0][0], 1e-9)
	require.InDelta(t, 1, near[0][1], 1e-9)
	require.InDelta(t, 10, near[8][0], 1e-9)
	require.InDelta(t, 1.5, near[8][1], 1e-9)
}
