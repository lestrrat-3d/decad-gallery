package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// perimeter returns the length of the closed polygon.
func perimeter(p []Pt) float64 {
	sum := 0.0
	for i := range p {
		sum += p[i].Sub(p[(i+1)%len(p)]).Len()
	}
	return sum
}

// bevel is the volume a 0.5 mm 45° chamfer removes per millimetre of straight edge.
const bevel = 0.5 * 0.5 / 2

func TestEdgeFinishing(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	plain := newBuilder(t.Context(), NewDocument(), g)
	plain.plain = true

	// rimBevel is the volume a 0.5 mm chamfer removes round the rim of a hole of radius r: pi (r + c/3) c².
	rimBevel := func(r float64) float64 { return math.Pi * (r + 0.5/3) * 0.25 }

	check := func(name string, finished *decad.Body, unfinished func() (*decad.Body, error), edges, extra float64) {
		t.Run(name, func(t *testing.T) {
			raw, err := unfinished()
			require.NoError(t, err)
			fv, err := finished.Volume()
			require.NoError(t, err)
			rv, err := raw.Volume()
			require.NoError(t, err)
			drop := rv.Value.Base() - fv.Value.Base()
			want := edges*bevel + extra
			// A chamfer along a curved edge removes a little more or less than the straight-edge figure.
			require.InDelta(t, want, drop, 0.05*want, "volume drop %v against %v mm of chamfered edge", drop, edges)
		})
	}

	// Plates: the outer loop and both windows on both faces, the three seat rims on the inner face and the three lip-bore
	// rims on the outer face.
	outer := func(z float64, body *decad.Body) float64 { return perimeter(loopsBy(sectionAt(t, body, z))[0]) }
	backEdges := 2*plateRims(t, m.BackPlate, -3) + 3*2*math.Pi*seatRadius + 3*2*math.Pi*lipRadius
	frontEdges := 2*plateRims(t, m.FrontPlate, 31) + 3*2*math.Pi*seatRadius + 3*2*math.Pi*lipRadius
	check("back plate", m.BackPlate, plain.buildBackPlate, backEdges, 0)
	check("front plate", m.FrontPlate, plain.buildFrontPlate, frontEdges, 0)
	// Anchor: the outer loop of both faces, pallets included, and the front counterbore's rim.
	check("anchor", m.Anchor, plain.buildAnchor, 2*perimeter(outerLoop2(sectionAt(t, m.Anchor, 23.5).Loops, g.P)),
		rimBevel(anchorCounterRadius))
	// Pendulum: the outer loop of both faces, the pocket rim and the lip-bore rim.
	check("pendulum", m.Pendulum, plain.buildPendulum, 2*outer(40.5, m.Pendulum)+2*math.Pi*pocketRadius+2*math.Pi*lipRadius, 0)
	// Great wheel: the rear counterbore's rim only; the tooth flanks and the ratchet stay sharp.
	check("great wheel", m.GreatWheel, plain.buildGreatWheel, 0, rimBevel(greatCounterRadius))
	// Escape wheel: the six window loops on both faces and the front counterbore's rim. The pinion's rear face and its
	// rear counterbore stay sharp: the face is 255 mm², under 400.
	var windows float64
	for _, l := range loopsBy(sectionAt(t, m.EscapeWheel, 23.5))[1:] {
		if len(l) > 20 && centroidLen(l) > 10 {
			windows += perimeter(l)
		}
	}
	check("escape wheel", m.EscapeWheel, plain.buildEscapeWheel, 2*windows, rimBevel(wheelCounterRadius))

	// Kept sharp: the mounting holes and pillar holes of the back plate, the bolt holes and head pockets of the front
	// plate (the bolt hole itself ends on the pocket floor), the slot of the pendulum, and the teeth, ratchet and flange of the other parts keep their rim edges.
	for _, h := range mountHoles {
		require.True(t, inside(t, m.BackPlate, h.X+2.3, h.Y, -0.1), "mounting hole rim is sharp on the inner face")
		require.True(t, inside(t, m.BackPlate, h.X+2.3, h.Y, -5.9), "mounting hole rim is sharp on the rear face")
	}
	for _, p := range pillars {
		require.True(t, inside(t, m.FrontPlate, p.X+2.3, p.Y, 33.9), "head pocket rim is sharp")
	}
	require.True(t, inside(t, m.Pendulum, g.S.X+2.0, g.S.Y-58, 43.65), "the slot's rim is sharp")
}

func centroidLen(l []Pt) float64 { return centroid(l).Len() }

// plateRims is the length of the outer loop and the two window loops of a plate at height z.
func plateRims(t *testing.T, body *decad.Body, z float64) float64 {
	t.Helper()
	sum := 0.0
	for _, l := range loopsBy(sectionAt(t, body, z)) {
		// The outer loop and the windows are the loops longer than 100 mm; holes, pockets and sockets are shorter.
		if perimeter(l) > 100 {
			sum += perimeter(l)
		}
	}
	return sum
}
