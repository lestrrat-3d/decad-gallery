package onewayclutch_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestOneWayClutchEdgeFinishing checks the hub's chamfer on both faces in sections. Every hub's faces are over 400 mm²
// (406.1 to 413.9 on the 605ZZ clutch, 441.0 to 466.7 on the 8 mm clutches), so each has a
// HubChamfer chamfer: the section 0.1 mm inside a face is the mid-height outline pulled in HubChamfer - 0.1 along the
// arm's outer arc and each inner run, with the bevel tapering out on the head flank's plane, sqrt(ArmOuter² - F²) -
// sqrt((ArmOuter - chamfer)² - F²) along it from the arm's corner (F is FlankDistance), and it leaves the head run and
// the bolt holes unchanged. The 605ZZ's 15.8 mm D bore stays sharp; the 8 mm D bores, 25.1 to 25.3 mm, are long runs
// and take the chamfer whole. A hub under 400 mm² would be the exact prism and every section would read the same. A
// volume check would not do; the faceted result reads about 0.07 mm3 high per arc run.
func TestOneWayClutchEdgeFinishing(t *testing.T) {
	for _, c := range fitCases()[:6] {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			dm := m.Dimensions
			mid := efSection(t, m.Hub, midHub)
			area := efArea(mid, dm)
			chamfered := area >= edgefinish.LargeFaceArea
			t.Logf("%s hub face %.1f mm2", c.name, area)
			require.True(t, chamfered, "every hub face is over 400 mm²")
			for _, face := range []struct {
				name string
				z    float64
			}{
				{"back face", onewayclutch.HubFrom + efDepth},
				{"front face", onewayclutch.HubFrom + onewayclutch.HubHeight - efDepth},
			} {
				loops := efSection(t, m.Hub, face.z)
				require.Len(t, loops, len(mid), "%s: loop count", face.name)
				index := efIndex(loops)
				var outer efShift
				for _, loop := range mid {
					got := efMeasure(loop, index, dm.HubChamfer)
					switch efKindOf(loop, dm) {
					case efHole:
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: a bolt hole is not chamfered", face.name)
					case efBore:
						// ChamferRuns takes a hole loop of LongEdgeRun or more whole: the 8 mm D loops (25.1 to 25.3 mm).
						if dm.BoreLoop >= edgefinish.LongEdgeRun {
							require.Greater(t, got.offset, 0.99*got.total(), "%s: the long D bore is chamfered", face.name)
							break
						}
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: the short D bore stays sharp", face.name)
					case efOuter:
						outer = got
					}
				}
				// Per pawl: the head run stays sharp, with the bevel's taper out on the flank; the arm's outer arc and
				// the inner run are offset. Three pawls.
				perimeter := outer.total()
				t.Logf("%s %s: outer loop %.1f mm: unchanged %.1f, offset %.1f, other %.1f", c.name, face.name,
					perimeter, outer.unchanged, outer.offset, outer.other)
				require.Greater(t, outer.offset, 3*(20.1+45)-3*6, "%s: long runs offset", face.name)
				require.Greater(t, outer.unchanged, 3*7.1-3*3, "%s: head runs sharp", face.name)
				require.Less(t, outer.other, 0.05*perimeter, "%s: the transitions are short", face.name)

				// The arm's outer arc reads ArmOuter at mid-height and ArmOuter - chamfer in the face section.
				arm := dm.ArmOuter
				chamfer := dm.HubChamfer - efDepth
				require.Greater(t, efLengthAt(mid, arm, 0.04), 3*20.0, "arm arcs at r %.2f at mid-height", arm)
				require.Greater(t, efLengthAt(loops, arm-chamfer, 0.04), 3*18.0, "%s: arm arcs at r %.2f", face.name, arm-chamfer)
				require.Less(t, efLengthAt(loops, arm, 0.04), 3*3.0, "%s: no arm arc stays at r %.2f", face.name, arm)

				// The bevel ends on the flank's plane: the section follows the flank line 0.911 mm past the arm's corner.
				taper := efFlankTaper(m, loops)
				require.Zero(t, efFlankTaper(m, mid), "%s: no taper at mid-height", face.name)
				t.Logf("%s %s: taper along the flank's plane, three pawls: %.3f mm", c.name, face.name, taper)
				// The faceted section reads about 0.1 mm per pawl short of the exact value; a missing taper or a square end
				// reads 0 or 0.5.
				exact := math.Sqrt(arm*arm-dm.FlankDistance*dm.FlankDistance) -
					math.Sqrt((arm-chamfer)*(arm-chamfer)-dm.FlankDistance*dm.FlankDistance)
				require.InDelta(t, 3*exact, taper, 3*0.15, "%s: the bevel tapers out along the flank's plane", face.name)
			}
		})
	}
}

// TestEdgeFinishIndex checks real face sections against the exhaustive segment search.
func TestEdgeFinishIndex(t *testing.T) {
	m := fitBuild(t, fitCases()[0])
	mid := efSection(t, m.Hub, midHub)
	for _, z := range []float64{onewayclutch.HubFrom + efDepth, onewayclutch.HubFrom + onewayclutch.HubHeight - efDepth} {
		face := efSection(t, m.Hub, z)
		index := efIndex(face)
		count := 0
		for _, loop := range mid {
			for i, a := range loop {
				b := loop[(i+1)%len(loop)]
				n := int(math.Ceil(b.Sub(a).Len() / efStep))
				for k := 0; k < n; k++ {
					count++
					p := a.Add(b.Sub(a).Scale(float64(k) / float64(n)))
					got, want := index.nearest(p), fitNearest(p, face)
					require.Equal(t, want, got, "z %.3f sample %d point (%.6f, %.6f, %.6f)", z, count, p.X, p.Y, p.Z)
					require.Equal(t, efClass(want), efClass(got), "z %.3f sample %d", z, count)
				}
			}
		}
		require.Positive(t, count)
	}
}

func TestEdgeFinishIndexEdgeCases(t *testing.T) {
	cases := []struct {
		name   string
		loops  [][]r3.Vec
		points []r3.Vec
	}{
		{"empty", [][]r3.Vec{nil, {}}, []r3.Vec{{}, {X: 1}}},
		{"point", [][]r3.Vec{{{X: 1, Y: 2, Z: 3}}}, []r3.Vec{{}, {X: 1, Y: 2, Z: 3}}},
		{"tree", [][]r3.Vec{{{X: -5}, {X: -4}, {X: -3}, {X: -2}, {X: -1}, {},
			{X: 1}, {X: 2}, {X: 3}, {X: 4}, {X: 5}}},
			[]r3.Vec{{Y: 1}, {X: 4, Y: 1}, {X: -4, Z: 1}}},
		{"segments", [][]r3.Vec{{{X: -2, Z: 3}, {X: 2, Z: 3}}, {{Y: 2}, {Y: 2}}},
			[]r3.Vec{{}, {Z: 3}, {X: 4, Y: 2, Z: 7}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			index := efIndex(c.loops)
			for _, p := range c.points {
				require.Equal(t, fitNearest(p, c.loops), index.nearest(p), "point (%.3f, %.3f, %.3f)", p.X, p.Y, p.Z)
			}
		})
	}
}
