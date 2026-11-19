package onewayclutch_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestOneWayClutchEars checks the ring's three ears in sections at the middle of each layer run: the ear tips at
// EarReach on the ear axes, the bores round, EarRadius-wide ears around them with 2.8 mm or more of wall to the outer
// loop and 3.2 mm or more to the counterbore, and the same outer loop in both directions. A section through the rear
// nut pockets checks each hexagon and its wall to every other loop.
func TestOneWayClutchEars(t *testing.T) {
	outer := map[string][]r3.Vec{}
	for _, c := range fitCases() {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			dm := m.Dimensions
			// Layer runs of the ring: the back and front counterbore runs, and the tooth band between them.
			for name, z := range map[string]float64{
				"back counterbore": (onewayclutch.RingFrom + onewayclutch.BandFrom - (dm.RingCounter - dm.ToothRoot) +
					onewayclutch.RingChamfer) / 2,
				"band": (onewayclutch.RingFrom + onewayclutch.RingTo) / 2,
				"front counterbore": (onewayclutch.RingTo + onewayclutch.BandTo + (dm.RingCounter - dm.ToothRoot) -
					onewayclutch.RingChamfer) / 2,
			} {
				rl := splitRing(t, fitSection(t, m.Ring, fitPlane(t, z)).Loops, dm)
				for k, a := range onewayclutch.EarAngles {
					require.InDelta(t, dm.EarReach, loopRadiusAlong(rl.outer, a), 0.03, "%s: ear %d tip", name, k+1)
					c := earCentre(dm.EarCircle, k)
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
			dm := m.Dimensions
			for name, z := range map[string]float64{
				"back face":  onewayclutch.RingFrom + depth,
				"front face": onewayclutch.RingTo - depth,
			} {
				s := fitSection(t, m.Ring, fitPlane(t, z))
				rl := splitRing(t, s.Loops, dm)
				if name == "back face" {
					// The rear nut pockets open on this face: three unchamfered hexagons and no bore loops.
					requirePocketLoops(t, s, dm.EarCircle)
				}
				// Between the ears the outside is a circle 0.25 mm inside RingOuter.
				for _, deg := range []float64{30, 150, 270} {
					require.InDelta(t, dm.RingOuter-depth, loopRadiusAlong(rl.outer, deg), 0.03,
						"%s: outside at %g degrees", name, deg)
				}
				for k, a := range onewayclutch.EarAngles {
					require.InDelta(t, dm.EarReach-depth, loopRadiusAlong(rl.outer, a), 0.03,
						"%s: ear %d tip", name, k+1)
					cen := earCentre(dm.EarCircle, k)
					for _, p := range rl.bores[k] { // none at the back face
						require.InDelta(t, 1.3, math.Hypot(p.X-cen.X, p.Y-cen.Y), 0.02, "%s: bore %d not chamfered", name, k+1)
					}
				}
				for _, p := range rl.inner {
					require.InDelta(t, dm.RingCounter+depth, math.Hypot(p.X, p.Y), 0.03, "%s: counterbore", name)
				}
			}
		})
	}
}
