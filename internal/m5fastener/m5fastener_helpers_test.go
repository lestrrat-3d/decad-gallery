package m5fastener_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m5fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const blockSize = 30.0

func site() m5fastener.Site {
	return m5fastener.Site{Dir: r3.Vec{Z: 1}, Corner: r3.Vec{X: 1}, Length: 16}
}

func frameAt(t *testing.T, z float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}

func bounds(t *testing.T, b *decad.Body) decad.Box {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err)
	return box
}

func volume(t *testing.T, b *decad.Body) float64 {
	t.Helper()
	v, err := b.Volume()
	require.NoError(t, err)
	return v.Value.Base()
}

// block builds a blockSize cube spanning -15..15 in X and Y and 0..blockSize in Z.
func block(t *testing.T, doc *decad.Document) *decad.Body {
	t.Helper()
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(frameAt(t, 0))
	require.NoError(t, err)
	s, err := world.CreateSketch(plane)
	require.NoError(t, err)
	h := blockSize / 2
	r := s.CreateRectangle(-h, -h, h, h)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, -h),
		sketch.NewVerticalDistance(s.Origin(), r.A, -h),
		sketch.NewHorizontalDistance(r.A, r.B, blockSize),
		sketch.NewVerticalDistance(r.B, r.C, blockSize),
	)
	_, err = s.Solve(t.Context())
	require.NoError(t, err)
	var chosen *sketch.Profile
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == 0 {
			chosen = p
		}
	}
	require.NotNil(t, chosen)
	body, err := decad.Extrude(doc, s, chosen,
		decad.Distance{D: units.Millimeters(blockSize), Dir: decad.Along}).Do(t.Context())
	require.NoError(t, err)
	return body
}
