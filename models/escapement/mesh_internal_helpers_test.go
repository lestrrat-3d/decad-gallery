package escapement

import (
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

func centroid(p []Pt) Pt {
	var c Pt
	for _, v := range p {
		c = c.Add(v)
	}
	return Pt{c.X / float64(len(p)), c.Y / float64(len(p))}
}
