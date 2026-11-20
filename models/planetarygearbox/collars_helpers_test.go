package planetarygearbox_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// Collar ring dimensions from instructions.md, "Arbor collars": the ring is 4 mm long, its press fit starts 1 mm from
// its bearing face, and over the press fit its wall is 1.7 mm. Its bore is relieved 0.4 mm round the shaft at the
// bearing face.
const (
	collarRingLength = 4.0
	collarRingRelief = 1.0
	collarRingWall   = 1.7
	collarReliefGap  = 0.4
)

// requireCollarRings checks the collar ring on each free end of box, built from p with a nil Placement, against
// Arbor collars and Part clearances: each is at least 3.0 mm long and stands a Tight gap off its cover bearing's outer
// face; its bearing face stays inside the bearing's inner ring; it is a press fit on the shaft's flat with a wall of at
// least 1.6 mm, relieved by a Regular gap at the bearing face; and its D bore's tooth starts where the shaft's flat
// ends, so the flat's end locates the ring.
func requireCollarRings(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	spec, ok := ballbearing.Lookup(p.ShaftBearing)
	require.True(t, ok)
	rod := spec.BoreDiameter / 2
	output := box.Stages[len(box.Stages)-1].Shaft
	front, rear := bounds(t, box.FrontBearing), bounds(t, box.RearBearing)
	for _, c := range []struct {
		name         string
		collar       *decad.Body
		shaft        *decad.Body
		face, toward float64 // the ring's bearing face, and the direction from it along the ring
	}{
		{planetarygearbox.FrontCollarName, box.FrontCollar, box.InputShaft, front.Min.Z - 0.25, -1},
		{planetarygearbox.RearCollarName, box.RearCollar, output, rear.Max.Z + 0.25, 1},
	} {
		t.Run(c.name, func(t *testing.T) {
			b := bounds(t, c.collar)
			length := b.Max.Z - b.Min.Z
			require.GreaterOrEqual(t, length, 3.0, "collar length")
			require.InDelta(t, collarRingLength, length, 1e-6)
			// The bearing face stands 0.25 mm off the bearing's outer face, a Tight gap.
			gap := b.Min.Z - rear.Max.Z
			if c.toward < 0 {
				gap = front.Min.Z - b.Max.Z
			}
			require.GreaterOrEqual(t, gap, 0.2, "gap to the bearing")
			require.InDelta(t, 0.25, gap, 1e-6)
			at := func(depth float64) float64 { return c.face + c.toward*depth }

			// At its bearing face the ring is the shaft collar diameter across, inside the inner ring's face.
			face := sectionRadialSpan(t, c.collar, r3.Vec{}, at(0.05), 0)
			require.Less(t, 2*(face.max+face.bound), spec.InnerRaceDiameter, "the face bears on the inner ring only")
			require.InDelta(t, rod+1.2+0.05/1.7320508, face.max, face.bound+2e-3, "face radius")
			// The relieved bore keeps a Regular gap to the round shaft.
			requireSectionClearance(t, c.shaft, c.collar, at(0.2), 0.35)
			// Over the press fit the ring grips the shaft's flat with the press-fit clearance, 0.05 mm proven to about
			// 0.03 mm, and keeps a wall of 1.6 mm or more round its bore.
			press := at((collarRingRelief + collarRingLength) / 2)
			requireSectionClearance(t, c.shaft, c.collar, press, 0.02)
			wall := sectionRadialSpan(t, c.collar, r3.Vec{}, press, 0)
			require.InDelta(t, rod+0.05+collarRingWall, wall.max, wall.bound+2e-3, "outer radius")
			requireBoreWalls(t, c.name, c.collar, press, 1)

			if p.FlatDepth == 0 {
				return
			}
			// The tooth of the ring's D bore starts at the end of the relief, where the shaft's flat ends: the ring has
			// material across the flat's plane just inside the press fit and none just outside it, and the shaft is
			// round on the bearing's side of that plane and flat beyond it.
			tooth := r3.Vec{X: rod + 0.05 - p.FlatDepth/2}
			inside, outside := tooth, tooth
			inside.Z, outside.Z = at(collarRingRelief+0.1), at(collarRingRelief-0.1)
			require.Equal(t, decad.PointInside, sectionLocate(t, c.collar, inside), "the tooth starts at the press fit")
			require.Equal(t, decad.PointOutside, sectionLocate(t, c.collar, outside), "no tooth in the relief")
			flat := r3.Vec{X: rod - p.FlatDepth/2}
			beyond, before := flat, flat
			beyond.Z, before.Z = at(collarRingRelief+0.1), at(collarRingRelief-0.1)
			require.Equal(t, decad.PointOutside, sectionLocate(t, c.shaft, beyond), "the flat runs under the press fit")
			require.Equal(t, decad.PointInside, sectionLocate(t, c.shaft, before), "the shaft is round in the relief")
		})
	}
}
