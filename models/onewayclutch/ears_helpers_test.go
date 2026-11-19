package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// ringLoops is a ring section split into the outer loop, the three ear bore loops (in EarAngles order) and the inner
// loop, which is the teeth in a layer run at the band and the counterbore in an end run. Where a section crosses a
// rear nut pocket, the ear's loop is the pocket's hexagon, in pockets, and the ear has no bore loop.
type ringLoops struct {
	outer, inner []r3.Vec
	bores        [onewayclutch.Ears][]r3.Vec
	pockets      [onewayclutch.Ears][]r3.Vec
}

func earCentre(earCircle float64, k int) r3.Vec {
	s, c := math.Sincos(onewayclutch.EarAngles[k] * math.Pi / 180)
	return r3.Vec{X: earCircle * c, Y: earCircle * s}
}

func splitRing(t *testing.T, loops [][]r3.Vec, dm onewayclutch.Dimensions) ringLoops {
	t.Helper()
	var out ringLoops
	for _, loop := range loops {
		hi := 0.0
		for _, p := range loop {
			hi = math.Max(hi, math.Hypot(p.X, p.Y))
		}
		placed := false
		for k := range out.bores {
			c := earCentre(dm.EarCircle, k)
			switch d := math.Hypot(loop[0].X-c.X, loop[0].Y-c.Y); {
			case d < 2:
				require.Nil(t, out.bores[k], "one bore loop at ear %d", k+1)
				out.bores[k] = loop
				placed = true
			case d < 3:
				require.Nil(t, out.pockets[k], "one pocket loop at ear %d", k+1)
				out.pockets[k] = loop
				placed = true
			}
		}
		switch {
		case placed:
		case hi > dm.RingCounter+1:
			require.Nil(t, out.outer, "one outer loop")
			out.outer = loop
		default:
			require.Nil(t, out.inner, "one inner loop")
			out.inner = loop
		}
	}
	require.NotNil(t, out.outer)
	require.NotNil(t, out.inner)
	for k, b := range out.bores {
		if out.pockets[k] != nil {
			require.Nil(t, b, "an ear has a pocket loop or a bore loop, not both (ear %d)", k+1)
			continue
		}
		require.NotNil(t, b, "bore loop at ear %d", k+1)
	}
	return out
}

// loopRadiusAlong returns the largest distance from the axis, along the ray at deg, of the loop's points within 1.5 mm
// of that ray.
func loopRadiusAlong(loop []r3.Vec, deg float64) float64 {
	s, c := math.Sincos(deg * math.Pi / 180)
	best := 0.0
	for _, p := range loop {
		along, across := p.X*c+p.Y*s, -p.X*s+p.Y*c
		if math.Abs(across) < 1.5 {
			best = math.Max(best, along)
		}
	}
	return best
}

// requireMountPockets checks the ring's three rear nut pockets in the section 1 mm in from the rear face (the pockets
// run to z 0.75). The face's chamfer ends at z -0.75, so the section is above it.
func requireMountPockets(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	requirePocketLoops(t, fitSection(t, m.Ring, fitPlane(t, onewayclutch.RingFrom+1.0)), m.Dimensions.EarCircle)
}

// requirePocketLoops requires the section of the ring's rear end to hold, besides the outer and counterbore loops, one
// loop per ear: its nut pocket's hexagon, with its corners on the NutCavityRadius circle about the ear's axis and a flat
// toward the ring's axis. Each keeps 1.0 mm or more of wall to every other loop.
func requirePocketLoops(t *testing.T, s decad.SectionOutline, earCircle float64) {
	t.Helper()
	require.Len(t, s.Loops, 2+onewayclutch.Ears, "outer, counterbore, and a hexagon at each ear")
	flat := m2fastener.NutCavityRadius * math.Sqrt(3) / 2
	found := 0
	for i, loop := range s.Loops {
		c := earCentre(earCircle, 0)
		isPocket := false
		for k := range onewayclutch.EarAngles {
			if math.Hypot(loop[0].X-earCentre(earCircle, k).X, loop[0].Y-earCentre(earCircle, k).Y) < 3 {
				c, isPocket = earCentre(earCircle, k), true
			}
		}
		if !isPocket {
			continue
		}
		found++
		// Every vertex lies between the flat and the corner circle of the hexagon.
		inward := 0
		for _, p := range loop {
			d := math.Hypot(p.X-c.X, p.Y-c.Y)
			require.GreaterOrEqual(t, d, flat-0.03, "pocket vertex inside the flat")
			require.LessOrEqual(t, d, m2fastener.NutCavityRadius+0.03, "pocket vertex outside the corners")
			if math.Abs(d-m2fastener.NutCavityRadius) < 0.03 {
				inward++
			}
		}
		require.GreaterOrEqual(t, inward, 6, "six corners on the NutCavityRadius circle")
		// A flat faces the ring's axis: no vertex is nearer than that flat. A corner there would reach 19.09.
		nearest := math.Inf(1)
		for _, p := range loop {
			nearest = math.Min(nearest, math.Hypot(p.X, p.Y))
		}
		require.GreaterOrEqual(t, nearest, earCircle-flat-0.03, "a flat faces the axis")
		wall := math.Inf(1)
		for j, other := range s.Loops {
			if j == i {
				continue
			}
			for _, p := range loop {
				wall = math.Min(wall, fitNearest(p, [][]r3.Vec{other}))
			}
		}
		require.GreaterOrEqual(t, wall, 1.0, "wall round a mount nut pocket")
		t.Logf("mount nut pocket at (%.2f, %.2f): wall %.3f", c.X, c.Y, wall)
	}
	require.Equal(t, onewayclutch.Ears, found, "three pockets")
}
