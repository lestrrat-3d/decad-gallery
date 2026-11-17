package swisslever

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// boreReach returns the least distance from axis to a segment of the section of body at z whose two ends lie within
// 4 mm of axis: the bore's nearest wall.
func boreReach(t *testing.T, body *decad.Body, z float64, axis Pt) float64 {
	t.Helper()
	loops, _ := sectionLoops(t, body, z)
	least := math.Inf(1)
	for _, l := range loops {
		for i, p := range l {
			q := l[(i+1)%len(l)]
			if p.Sub(axis).Len() > 4 || q.Sub(axis).Len() > 4 {
				continue
			}
			least = math.Min(least, segDist(axis, p, q))
		}
	}
	return least
}

// TestInstallability requires that each between-plate part with a body-family bore passes a 605ZZ seat (radius 2.5)
// on its way to its place on the arbor, and that each D bore keeps its press fit at both ends, where the arbor must
// already have its flat.
func TestInstallability(t *testing.T) {
	_, m := sharedDefault(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	for _, e := range []struct {
		name string
		body *decad.Body
		axis Pt
		z    float64
	}{
		{"escape wheel", m.EscapeWheel, g.O, 11.85}, {"lever", m.Lever, g.P, 8},
		{"safety roller", m.SafetyRoller, g.B, 7}, {"roller table", m.RollerTable, g.B, 12},
		{"balance wheel", m.BalanceWheel, g.B, 15.9}, {"hairspring retainer", m.HairspringRetainer, g.B, 24},
	} {
		reach := boreReach(t, e.body, e.z, e.axis)
		require.GreaterOrEqual(t, reach, ArborRadius+0.04, "%s: its bore must pass a bearing seat (reach %v)", e.name, reach)
	}

	// Each press D bore against its arbor at 0.1 mm inside each end: 0.05 all round, no overlap anywhere along it.
	for _, e := range []struct {
		name        string
		part, arbor *decad.Body
		z0, z1      float64
	}{
		{"escape wheel", m.EscapeWheel, m.EscapeArbor, wheelLow + 0.1, wheelHigh - 0.1},
		{"lever", m.Lever, m.PalletArbor, leverLow + 0.1, leverHigh - 0.1},
		{"safety roller", m.SafetyRoller, m.BalanceStaff, safetyLow + 0.1, safetyLow + safetyLength - 0.1},
		{"roller table", m.RollerTable, m.BalanceStaff, tableLow + 0.1, tableLow + tableThickness - 0.1},
		{"balance wheel", m.BalanceWheel, m.BalanceStaff, balanceLow + 0.1, balanceHigh - 0.1},
		{"hairspring retainer", m.HairspringRetainer, m.BalanceStaff, retainerLow + 0.1, retainerHigh - 0.1},
		{"knob collar", m.KnobCollar, m.EscapeArbor, collarLow + 0.1, collarLow + collarLong - 0.1},
	} {
		radial(t, e.part, e.arbor, e.z0, 0.05, 0, e.name+" bore at its rear end")
		radial(t, e.part, e.arbor, e.z1, 0.05, 0, e.name+" bore at its front end")
	}
	// The knob's socket runs through its floor, 1.0 thick.
	radial(t, m.WindingKnob, m.EscapeArbor, keyedCounterLow+keyedCounterLong+0.1, 0.05, 0, "knob socket at its floor's rear")
	radial(t, m.WindingKnob, m.EscapeArbor, knobHigh-0.1, 0.05, 0, "knob socket at its front face")
}
