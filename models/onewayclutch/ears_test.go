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

func earCentre(k int) r3.Vec {
	s, c := math.Sincos(onewayclutch.EarAngles[k] * math.Pi / 180)
	return r3.Vec{X: onewayclutch.EarCircle * c, Y: onewayclutch.EarCircle * s}
}

func splitRing(t *testing.T, loops [][]r3.Vec) ringLoops {
	t.Helper()
	var out ringLoops
	for _, loop := range loops {
		hi := 0.0
		for _, p := range loop {
			hi = math.Max(hi, math.Hypot(p.X, p.Y))
		}
		placed := false
		for k := range out.bores {
			c := earCentre(k)
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
		case hi > 20:
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

// TestOneWayClutchEars checks the ring's three ears in sections at the middle of each layer run: the ear tips at
// EarReach on the ear axes, the bores round, EarRadius-wide ears around them with 2.8 mm or more of wall to the outer
// loop and 3.2 mm or more to the counterbore, and the same outer loop in both directions. A section through the rear
// nut pockets checks each hexagon and its wall to every other loop.
func TestOneWayClutchEars(t *testing.T) {
	outer := map[string][]r3.Vec{}
	for _, c := range fitCases() {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			// Layer runs of the ring: the back and front counterbore runs, and the tooth band between them.
			for name, z := range map[string]float64{
				"back counterbore": (onewayclutch.RingFrom + onewayclutch.BandFrom - (onewayclutch.RingCounter - onewayclutch.ToothRoot) +
					onewayclutch.RingChamfer) / 2,
				"band": (onewayclutch.RingFrom + onewayclutch.RingTo) / 2,
				"front counterbore": (onewayclutch.RingTo + onewayclutch.BandTo + (onewayclutch.RingCounter - onewayclutch.ToothRoot) -
					onewayclutch.RingChamfer) / 2,
			} {
				rl := splitRing(t, fitSection(t, m.Ring, fitPlane(t, z)).Loops)
				for k, a := range onewayclutch.EarAngles {
					require.InDelta(t, onewayclutch.EarReach, loopRadiusAlong(rl.outer, a), 0.03, "%s: ear %d tip", name, k+1)
					c := earCentre(k)
					for _, p := range rl.bores[k] {
						require.InDelta(t, 1.3, math.Hypot(p.X-c.X, p.Y-c.Y), 0.02, "%s: ear %d bore radius", name, k+1)
					}
					wall := math.Inf(1)
					for _, p := range rl.bores[k] {
						wall = math.Min(wall, fitNearest(p, [][]r3.Vec{rl.outer}))
					}
					require.GreaterOrEqual(t, wall, 2.8-0.03, "%s: wall from bore %d to the outer loop", name, k+1)
					if name != "band" {
						inner := math.Inf(1)
						for _, p := range rl.bores[k] {
							inner = math.Min(inner, fitNearest(p, [][]r3.Vec{rl.inner}))
						}
						require.GreaterOrEqual(t, inner, 3.2-0.03, "%s: wall from bore %d to the counterbore", name, k+1)
					}
				}
				if name == "band" {
					outer[c.name] = rl.outer
				}
			}
			requireMountPockets(t, m)
		})
	}
	for _, name := range []string{"605ZZ", "608ZZ", "608ZZ-gearbox"} {
		cw, ccw := outer[name+"/"+onewayclutch.Clockwise.String()], outer[name+"/"+onewayclutch.CounterClockwise.String()]
		if cw == nil || ccw == nil {
			continue
		}
		for _, p := range cw {
			require.Less(t, fitNearest(p, [][]r3.Vec{ccw}), 0.03, "%s: the outer loop is the same in both directions", name)
		}
	}
}

// TestOneWayClutchRingEdgeFinishing checks the ring's 0.5 mm outer-loop chamfer and counterbore lead-in in sections
// 0.25 mm inside each end face: the outside and the ear tips lie 0.25 mm in, the counterbore 0.25 mm out, and the ear
// bores are not chamfered.
func TestOneWayClutchRingEdgeFinishing(t *testing.T) {
	const depth = 0.25
	for _, c := range fitCases()[:6] {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			for name, z := range map[string]float64{
				"back face":  onewayclutch.RingFrom + depth,
				"front face": onewayclutch.RingTo - depth,
			} {
				s := fitSection(t, m.Ring, fitPlane(t, z))
				rl := splitRing(t, s.Loops)
				if name == "back face" {
					// The rear nut pockets open on this face: three unchamfered hexagons and no bore loops.
					requirePocketLoops(t, s)
				}
				// Between the ears the outside is a circle 0.25 mm inside RingOuter.
				for _, deg := range []float64{30, 150, 270} {
					require.InDelta(t, onewayclutch.RingOuter-depth, loopRadiusAlong(rl.outer, deg), 0.03,
						"%s: outside at %g degrees", name, deg)
				}
				for k, a := range onewayclutch.EarAngles {
					require.InDelta(t, onewayclutch.EarReach-depth, loopRadiusAlong(rl.outer, a), 0.03,
						"%s: ear %d tip", name, k+1)
					cen := earCentre(k)
					for _, p := range rl.bores[k] { // none at the back face
						require.InDelta(t, 1.3, math.Hypot(p.X-cen.X, p.Y-cen.Y), 0.02, "%s: bore %d not chamfered", name, k+1)
					}
				}
				for _, p := range rl.inner {
					require.InDelta(t, onewayclutch.RingCounter+depth, math.Hypot(p.X, p.Y), 0.03, "%s: counterbore", name)
				}
			}
		})
	}
}

// requireMountPockets checks the ring's three rear nut pockets in the section 1 mm in from the rear face (the pockets
// run to z 0.75). The face's chamfer ends at z -0.75, so the section is above it.
func requireMountPockets(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	requirePocketLoops(t, fitSection(t, m.Ring, fitPlane(t, onewayclutch.RingFrom+1.0)))
}

// requirePocketLoops requires the section of the ring's rear end to hold, besides the outer and counterbore loops, one
// loop per ear: its nut pocket's hexagon, with its corners on the NutCavityRadius circle about the ear's axis and a flat
// toward the ring's axis. Each keeps 1.0 mm or more of wall to every other loop.
func requirePocketLoops(t *testing.T, s decad.SectionOutline) {
	t.Helper()
	require.Len(t, s.Loops, 2+onewayclutch.Ears, "outer, counterbore, and a hexagon at each ear")
	flat := m2fastener.NutCavityRadius * math.Sqrt(3) / 2
	found := 0
	for i, loop := range s.Loops {
		c := earCentre(0)
		isPocket := false
		for k := range onewayclutch.EarAngles {
			if math.Hypot(loop[0].X-earCentre(k).X, loop[0].Y-earCentre(k).Y) < 3 {
				c, isPocket = earCentre(k), true
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
		require.GreaterOrEqual(t, nearest, onewayclutch.EarCircle-flat-0.03, "a flat faces the axis")
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
