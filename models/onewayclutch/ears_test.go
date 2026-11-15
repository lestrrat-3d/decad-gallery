package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// ringLoops is a ring section split into the outer loop, the three ear bore loops (in EarAngles order) and the inner
// loop, which is the teeth in a layer run at the band and the counterbore in an end run.
type ringLoops struct {
	outer, inner []r3.Vec
	bores        [onewayclutch.Ears][]r3.Vec
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
			if math.Hypot(loop[0].X-c.X, loop[0].Y-c.Y) < 2 {
				require.Nil(t, out.bores[k], "one bore loop at ear %d", k+1)
				out.bores[k] = loop
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
// EarReach on the ear axes, the bores round, EarRadius-wide ears around them with 2.2 mm or more of wall to the outer
// loop and 1.0 mm or more to the counterbore, and the same outer loop in both directions.
func TestOneWayClutchEars(t *testing.T) {
	outer := map[string][]r3.Vec{}
	for _, c := range fitCases() {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			// Layer runs of the ring: the back and front counterbore runs, and the tooth band between them.
			for name, z := range map[string]float64{
				"back counterbore":  (onewayclutch.RingFrom + onewayclutch.BandFrom - 1.5 + 0.5) / 2,
				"band":              (onewayclutch.RingFrom + onewayclutch.RingTo) / 2,
				"front counterbore": (onewayclutch.RingTo + onewayclutch.BandTo + 1.5 - 0.5) / 2,
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
					require.GreaterOrEqual(t, wall, 2.2, "%s: wall from bore %d to the outer loop", name, k+1)
					if name != "band" {
						inner := math.Inf(1)
						for _, p := range rl.bores[k] {
							inner = math.Min(inner, fitNearest(p, [][]r3.Vec{rl.inner}))
						}
						require.GreaterOrEqual(t, inner, 1.0, "%s: wall from bore %d to the counterbore", name, k+1)
					}
				}
				if name == "band" {
					outer[c.name] = rl.outer
				}
			}
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
				rl := splitRing(t, fitSection(t, m.Ring, fitPlane(t, z)).Loops)
				// Between the ears the outside is a circle 0.25 mm inside RingOuter.
				for _, deg := range []float64{30, 150, 270} {
					require.InDelta(t, onewayclutch.RingOuter-depth, loopRadiusAlong(rl.outer, deg), 0.03,
						"%s: outside at %g degrees", name, deg)
				}
				for k, a := range onewayclutch.EarAngles {
					require.InDelta(t, onewayclutch.EarReach-depth, loopRadiusAlong(rl.outer, a), 0.03,
						"%s: ear %d tip", name, k+1)
					cen := earCentre(k)
					for _, p := range rl.bores[k] {
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
