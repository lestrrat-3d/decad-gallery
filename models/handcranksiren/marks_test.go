package handcranksiren_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// markGear is one gear with its expected number of mark holes.
type markGear struct {
	name   string
	front  float64
	motion string
	holes  int
}

var markGears = []markGear{
	{"crank-gear", 20.6, "crank", 1},
	{"intermediate-pinion", 20.6, "intermediate", 1},
	{"intermediate-gear", 12.6, "intermediate", 2},
	{"rotor-pinion", 12.6, "rotor", 2},
}

// lookAlikeSets are the sets "Look-alike parts" in instructions.md declares: their parts must differ in hole count.
var lookAlikeSets = [][]string{
	{"crank-gear", "intermediate-gear"},
	{"intermediate-pinion", "rotor-pinion"},
}

const (
	markCircle = 6.6  // radius of the mark circle about the gear axis
	markHole   = 0.75 // radius of one mark hole
)

// markLoops counts the loops of the gear's section across Y at y that are mark holes: closed loops that lie wholly
// within one hole radius, plus the section tolerance, of the mark circle about axis.
func markLoops(t *testing.T, body *decad.Body, axis r3.Vec, y float64) int {
	t.Helper()
	section, err := body.SectionOutline(t.Context(), atY(t, y), units.Millimeters(sectionTolerance))
	require.NoError(t, err)
	holes := 0
	for _, loop := range section.Loops {
		near, far := loopDistances(loop, axis)
		slack := 0.05
		if near >= markCircle-markHole-slack && far <= markCircle+markHole+slack {
			holes++
		}
	}
	return holes
}

// requireMarkHoles counts the mark holes of each gear in sections at mid thickness, checks that sections just inside
// each face show the same holes so they pass through, and checks that the parts of a look-alike set have different counts.
func requireMarkHoles(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	t.Helper()
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	axes := map[string]r3.Vec{"crank": state.Crank, "intermediate": state.Intermediate, "rotor": state.Rotor}
	counts := map[string]int{}
	for _, g := range markGears {
		body := parts[g.name].Body
		axis, err := handcranksiren.PosePoint(p, g.motion, axes[g.motion])
		require.NoError(t, err)
		for _, depth := range []float64{0.2, 3, 5.8} {
			require.Equal(t, g.holes, markLoops(t, body, axis, g.front+depth), "%s mark holes at depth %v", g.name, depth)
		}
		counts[g.name] = g.holes
	}
	for _, set := range lookAlikeSets {
		seen := map[int]string{}
		for _, name := range set {
			other, dup := seen[counts[name]]
			require.False(t, dup, "%s and %s share %d mark holes", name, other, counts[name])
			seen[counts[name]] = name
		}
	}
}
