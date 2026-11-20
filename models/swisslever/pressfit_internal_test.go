package swisslever

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// TestPressFitWalls requires the wall round the bore of every separate printed part pressed on a shaft to be at least
// 1.6 mm (docs/model-pipeline.md Corners and roots, Root width), at the section named for each part, and the wall round
// each pallet stone's slot too.
func TestPressFitWalls(t *testing.T) {
	_, m := sharedDefault(t)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)
	for _, e := range []struct {
		name string
		body *decad.Body
		axis Pt
		z    float64
	}{
		{"Safety Roller", m.SafetyRoller, g.B, 7}, {"Roller Table", m.RollerTable, g.B, 12},
		{"Balance Wheel hub", m.BalanceWheel, g.B, 15.9}, {"Hairspring Retainer", m.HairspringRetainer, g.B, 24},
		{"Lever hub", m.Lever, g.P, 8}, {"Knob Collar", m.KnobCollar, g.O, 35},
		{"Winding Knob floor", m.WindingKnob, g.O, 36.8}, {"Winding Knob counterbore", m.WindingKnob, g.O, 35},
	} {
		loops, off := sectionLoops(t, e.body, e.z)
		wall := boreWall(t, loops, e.axis)
		require.GreaterOrEqual(t, wall, minPressWall-0.02-off, "%s: wall round the bore at z %v is %.3f", e.name, e.z, wall)
	}

	// The safety roller's thinnest wall is the 1.62 between its crescent and its bore.
	loops, off := sectionLoops(t, m.SafetyRoller, 7)
	require.InDelta(t, CrescentOffset-CrescentRadius-BodyBoreRadius, boreWall(t, loops, g.B), 0.03+off)

	// Round each slot of the lever: the slot loops are the two loops that do not reach the bore or the outline.
	loops, off = sectionLoops(t, m.Lever, 8)
	var slots []outline
	var rest []outline
	for _, l := range loops {
		c := centroid(l)
		if c.Sub(g.P).Len() > 10 && c.Y < g.P.Y+1 && math.Abs(c.X-g.P.X) > 10 {
			slots = append(slots, l)
			continue
		}
		rest = append(rest, l)
	}
	require.Len(t, slots, 2, "the lever's two slots")
	for i, s := range slots {
		least := math.Inf(1)
		for _, r := range rest {
			least = math.Min(least, loopWall(s, r))
		}
		require.GreaterOrEqual(t, least, minPressWall-0.02-off, "wall round slot %d", i+1)
		require.InDelta(t, SlotWall, least, 0.1+off, "slot %d's wall to the nearest loop", i+1)
	}
}
