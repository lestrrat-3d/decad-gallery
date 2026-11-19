package escapement

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPressFitWalls requires the wall round the bore of every separate printed part pressed on a shaft to be at least
// 1.6 mm (docs/model-pipeline.md Corners and roots, Root width), at the section named for each part. The Mainspring's
// outline is one long loop, so its hub wall comes from its parameters.
func TestPressFitWalls(t *testing.T) {
	_, m, err := sharedDefault(t)
	require.NoError(t, err)
	g, err := Layout(DefaultParams())
	require.NoError(t, err)

	check := func(name string, at Pt, z float64, loops [][]Pt) {
		wall := boreWall(t, loops, at)
		require.GreaterOrEqual(t, wall, minPressWall-0.01, "%s: wall round the bore at z %v is %.3f", name, z, wall)
	}
	check("Spring Retainer", g.G, 19, loopsBy(sectionAt(t, m.SpringRetainer, 19)))
	check("Pawl Carrier boss", g.G, 10, loopsBy(sectionAt(t, m.PawlCarrier, 10)))
	check("Pawl Carrier layer", g.G, 7, loopsBy(sectionAt(t, m.PawlCarrier, 7)))
	check("Escape Wheel pinion", g.O, 10, loopsBy(sectionAt(t, m.EscapeWheel, 10)))
	check("Anchor hub", g.P, 23.5, loopsBy(sectionAt(t, m.Anchor, 23.5)))
	check("Crutch Collar", g.P, 36, loopsBy(sectionAt(t, m.CrutchCollar, 36)))
	check("Knob Collar", g.G, 36, loopsBy(sectionAt(t, m.KnobCollar, 36)))
	check("Suspension Cap", g.S, 46.5, loopsBy(sectionAt(t, m.SuspensionCap, 46.5)))
	check("Crutch floor", g.P, 37.8, loopsBy(sectionAt(t, m.Crutch, 37.8)))
	check("Winding Knob floor", g.G, 37.8, loopsBy(sectionAt(t, m.WindingKnob, 37.8)))

	sp := SpringParams()
	hubWall := sp.HubDiameter/2 - (sp.ArborDiameter/2 + sp.BoreClearance)
	require.GreaterOrEqual(t, hubWall, minPressWall, "Mainspring hub wall")
}
