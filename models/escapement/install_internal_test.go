package escapement

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
	o := sectionAt(t, body, z)
	least := math.Inf(1)
	for _, l := range loopsBy(o) {
		for i, p := range l {
			q := l[(i+1)%len(l)]
			if p.Sub(axis).Len() > 4 || q.Sub(axis).Len() > 4 {
				continue
			}
			least = math.Min(least, distPointSeg(axis, p, q))
		}
	}
	return least
}

// TestInstallability requires that each between-plate part with a bore passes a 605ZZ seat (radius 2.5) on its way
// to its place on the arbor, and that each D bore keeps its press fit at both ends, where the arbor must already have
// its flat.
func TestInstallability(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	const seat = ArborRadius
	for _, e := range []struct {
		name string
		body *decad.Body
		axis Pt
		z    float64
	}{
		{"escape wheel", m.EscapeWheel, g.O, 10}, {"anchor", m.Anchor, g.P, 22},
		{"pawl carrier", m.PawlCarrier, g.G, 7}, {"spring retainer", m.SpringRetainer, g.G, 19},
		{"mainspring", m.Mainspring, g.G, 14.25}, {"great wheel", m.GreatWheel, g.G, 6},
	} {
		// The bore's wall is a chord between exact section vertices; the section's OffSurface bound belongs to its other
		// loops (the spring's long outline, the anchor's pallets), so it is not added here.
		reach := boreReach(t, e.body, e.z, e.axis)
		require.GreaterOrEqual(t, reach, seat+0.04, "%s: its bore must pass a bearing seat (reach %v)", e.name, reach)
	}

	// Each press D bore against its arbor at 0.1 mm inside each end: 0.05 all round, no overlap anywhere along it.
	for _, e := range []struct {
		name         string
		part, arbor  *decad.Body
		z0, z1, want float64
		min          float64
	}{
		{"escape wheel", m.EscapeWheel, m.EscapeArbor, 1.6, 23.85, 0.05, 0},
		{"anchor", m.Anchor, m.PalletArbor, 21.1, 25.9, 0.05, 0},
		{"pawl carrier", m.PawlCarrier, m.GreatWheelArbor, 5.35, 10.9, 0.05, 0},
		{"spring retainer", m.SpringRetainer, m.GreatWheelArbor, 17.6, 20.4, 0.05, 0},
		{"great wheel", m.GreatWheel, m.GreatWheelArbor, 3.6, 8.4, 0.4, 0.35},
	} {
		radial(t, e.part, e.arbor, e.z0, e.want, e.min, e.name+" bore at its rear end")
		radial(t, e.part, e.arbor, e.z1, e.want, e.min, e.name+" bore at its front end")
	}
}
