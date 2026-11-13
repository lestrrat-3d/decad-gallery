package escapement

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// gearLoops returns the outer loops at z 3 of the pinion (about the origin, tooth 0 at -90°) and the great wheel (about
// the origin, tooth 0 on +X), and the section bound.
func gearLoops(t *testing.T, backlash float64) (pinion, great []Pt, off float64) {
	t.Helper()
	p := DefaultParams()
	p.Backlash = backlash
	g, err := Layout(p)
	require.NoError(t, err)
	b := newBuilder(t.Context(), NewDocument(), g)
	pin, err := b.buildGear("pinion", PinionTeeth, pinionLength, pinionRear, g.PinionPhase, 0)
	require.NoError(t, err)
	gw, err := b.buildGear("great wheel", GreatTeeth, greatLength, greatRear, 0, greatBore)
	require.NoError(t, err)
	po, go_ := sectionAt(t, pin, 3), sectionAt(t, gw, 3)
	return loopsBy(po)[0], loopsBy(go_)[0], po.OffSurface.Base() + go_.OffSurface.Base()
}

func TestGearMesh(t *testing.T) {
	for name, backlash := range map[string]float64{"default": DefaultParams().Backlash, "no backlash": 0} {
		t.Run(name, func(t *testing.T) {
			pinion, great, off := gearLoops(t, backlash)
			g, err := Layout(DefaultParams())
			require.NoError(t, err)
			tipP, rootP, tipG, rootG := 0.0, math.Inf(1), 0.0, math.Inf(1)
			for _, v := range pinion {
				tipP, rootP = math.Max(tipP, v.Len()), math.Min(rootP, v.Len())
			}
			for _, v := range great {
				tipG, rootG = math.Max(tipG, v.Len()), math.Min(rootG, v.Len())
			}
			// Gear train clearances: each tip clears its mate's root by at least max(0.25 m, 0.5 mm), measured from the
			// outlines (which carry their own bound).
			require.GreaterOrEqual(t, g.GearCentre-tipP-rootG, 0.5-off, "pinion tip to great wheel root")
			require.GreaterOrEqual(t, g.GearCentre-tipG-rootP, 0.5-off, "great wheel tip to pinion root")
			// Five angles over one pinion pitch, the pinion turning clockwise and the great wheel counter-clockwise.
			for k := 0; k < 5; k++ {
				phi := float64(k) / 5 * 2 * math.Pi / PinionTeeth
				p := rotPoly(pinion, -phi)
				gw := rotPoly(great, g.GreatPhase+phi*PinionTeeth/GreatTeeth)
				for i := range gw {
					gw[i] = gw[i].Add(g.G)
				}
				require.False(t, overlap(p, gw), "teeth overlap at sample %d", k)
				require.Greater(t, distPolys(p, gw), 0.0)
			}
		})
	}
}

// slotLoop returns the slot's loop in the pendulum section: the hole whose centre is far below the suspension.
func slotLoop(loops [][]Pt, s Pt) []Pt {
	for _, l := range loops {
		var c Pt
		for _, v := range l {
			c = c.Add(v)
		}
		c = Pt{c.X / float64(len(l)), c.Y / float64(len(l))}
		if c.Y < s.Y-50 && c.Y > s.Y-70 && len(l) < 400 {
			return l
		}
	}
	return nil
}

func TestCoupling(t *testing.T) {
	l := localParts(t)
	g := &l.g
	for _, phase := range []float64{0, 0.25, 0.5, 0.75} {
		s, err := g.StateAt(phase)
		require.NoError(t, err)
		table := poses(g, s)
		pin := l.posedLoops(t, CrutchName, 41.05, table[CrutchName])
		require.Len(t, pin, 1)
		slot := slotLoop(l.posedLoops(t, PendulumName, 41.05, table[PendulumName]), g.S)
		require.NotNil(t, slot, "the pendulum's slot")
		require.True(t, fitsIn(pin[0], slot), "the crutch pin stands in the slot at phase %v", phase)
		// Each side of the 3.5 mm slot clears the 3.0 mm pin by the Tight 0.25; the slot's length leaves the swing free.
		pc, sc := centroid(pin[0]), centroid(slot)
		axis := Polar(1, -s.PendulumAngle+math.Pi/2) // the slot's long axis, up the pendulum
		across := Pt{-axis.Y, axis.X}
		offset := pc.Sub(sc)
		side := math.Abs(offset.X*across.X + offset.Y*across.Y)
		require.InDelta(t, 0, side, 0.05, "the pin sits on the slot's centreline")
		gap := distPolys(pin[0], slot)
		require.GreaterOrEqual(t, gap, 0.2, "pin to slot at phase %v", phase)
		require.InDelta(t, 0.25, gap, 0.06, "pin to slot at phase %v", phase)
	}
}

func centroid(p []Pt) Pt {
	var c Pt
	for _, v := range p {
		c = c.Add(v)
	}
	return Pt{c.X / float64(len(p)), c.Y / float64(len(p))}
}
