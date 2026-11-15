package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const (
	// efTol is the chord tolerance of a hub section. The chamfered hub is faceted and its mesh bound is about 0.024.
	efTol = 0.03
	// efDepth is how far inside a hub face the section is taken. A 0.5 mm 45 degree chamfer offsets the outline by
	// 0.5 - efDepth there.
	efDepth = 0.25
	// efSharp and efOffset are how far a point of the mid-height outline may lie from a face section to read as
	// unchanged, and as offset by the chamfer.
	efSharp  = 0.05
	efOffset = 0.06
	// efStep is the spacing of the points sampled along the mid-height outline.
	efStep = 0.05
)

// efSection is the hub's outline at z, flattened to z = 0 so that outlines at different heights compare in the plane.
func efSection(t *testing.T, body *decad.Body, z float64) [][]r3.Vec {
	t.Helper()
	s, err := body.SectionOutline(t.Context(), fitPlane(t, z), units.Millimeters(efTol))
	require.NoError(t, err)
	for _, loop := range s.Loops {
		for i := range loop {
			loop[i].Z = 0
		}
	}
	return s.Loops
}

// efKind names a loop of the hub's outline.
type efKind int

const (
	efOuter efKind = iota // pawls, web and core
	efBore                // the D or round shaft bore
	efHole                // a bolt hole
)

func efKindOf(loop []r3.Vec) efKind {
	lo, hi := math.Inf(1), 0.0
	for _, p := range loop {
		r := math.Hypot(p.X, p.Y)
		lo, hi = math.Min(lo, r), math.Max(hi, r)
	}
	switch {
	case hi < 6:
		return efBore
	case lo > 6.5 && hi < 10:
		return efHole
	}
	return efOuter
}

// efShift is the length of a loop, in mm, that lies unchanged, offset by the chamfer, or neither, measured from the
// mid-height outline to the face section.
type efShift struct{ unchanged, offset, other float64 }

func (s efShift) total() float64 { return s.unchanged + s.offset + s.other }

// efMeasure samples the mid-height loop and reads each point's distance to the face section's loops.
func efMeasure(mid []r3.Vec, face [][]r3.Vec) efShift {
	var out efShift
	for i, a := range mid {
		b := mid[(i+1)%len(mid)]
		n := int(math.Ceil(b.Sub(a).Len() / efStep))
		for k := 0; k < n; k++ {
			p := a.Add(b.Sub(a).Scale(float64(k) / float64(n)))
			d := fitNearest(p, face)
			l := b.Sub(a).Len() / float64(n)
			switch {
			case d < efSharp:
				out.unchanged += l
			case math.Abs(d-efDepth) < efOffset:
				out.offset += l
			default:
				out.other += l
			}
		}
	}
	return out
}

// efLengthAt is the length of the loops' segments whose end points both lie within tol of radius r about the axis.
func efLengthAt(loops [][]r3.Vec, r, tol float64) float64 {
	on := func(p r3.Vec) bool { return math.Abs(math.Hypot(p.X, p.Y)-r) < tol }
	total := 0.0
	for _, loop := range loops {
		for i, a := range loop {
			if b := loop[(i+1)%len(loop)]; on(a) && on(b) {
				total += b.Sub(a).Len()
			}
		}
	}
	return total
}

// TestOneWayClutchEdgeFinishing checks the hub's 0.5 mm chamfer on both faces in sections: the section 0.25 mm inside
// a face is the mid-height outline pulled in 0.25 mm along the arm's outer arc and each inner run, and along the whole
// D loop at the 8 mm presets, and it leaves the head run, the bolt holes and the short 605ZZ D bore unchanged. A volume
// check would not do; the faceted result reads about 0.07 mm3 high per arc run.
func TestOneWayClutchEdgeFinishing(t *testing.T) {
	for _, c := range fitCases()[:6] {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			// The 8 mm D loop is 25 mm long; the 5 mm one is 15 mm, short of the rule's 20 mm.
			longBore := m.Dimensions.Shaft.Diameter > 6
			mid := efSection(t, m.Hub, midHub)
			for _, face := range []struct {
				name string
				z    float64
			}{
				{"back face", onewayclutch.HubFrom + efDepth},
				{"front face", onewayclutch.HubFrom + onewayclutch.HubHeight - efDepth},
			} {
				loops := efSection(t, m.Hub, face.z)
				require.Len(t, loops, len(mid), "%s: loop count", face.name)
				var outer efShift
				for _, loop := range mid {
					got := efMeasure(loop, loops)
					switch efKindOf(loop) {
					case efHole:
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: a bolt hole is not chamfered", face.name)
					case efBore:
						if longBore {
							require.Greater(t, got.offset, 0.97*got.total(), "%s: the whole D loop is chamfered", face.name)
						} else {
							require.Greater(t, got.unchanged, 0.99*got.total(), "%s: the short D bore stays sharp", face.name)
						}
					case efOuter:
						outer = got
					}
				}
				// Per pawl: the head run (7.1 mm) stays sharp, with the bevel's taper out on the flank; the arm's outer
				// arc (22.9) and the inner run (47.8) are offset. Three pawls.
				perimeter := outer.total()
				t.Logf("%s %s: outer loop %.1f mm: unchanged %.1f, offset %.1f, other %.1f", c.name, face.name,
					perimeter, outer.unchanged, outer.offset, outer.other)
				require.Greater(t, outer.offset, 3*(22.9+47.8)-3*4, "%s: long runs offset", face.name)
				require.Less(t, outer.offset, 3*(22.9+47.8)+3*4, "%s: long runs offset", face.name)
				require.Greater(t, outer.unchanged, 3*7.1-3*3, "%s: head runs sharp", face.name)
				require.Less(t, outer.unchanged, 3*7.1+3*4, "%s: head runs sharp", face.name)
				require.Less(t, outer.other, 0.05*perimeter, "%s: the transitions are short", face.name)

				// The arm's outer arc reads r 13.95 at mid-height and r 13.70 in the face section.
				arm := onewayclutch.ArmOuter
				require.Greater(t, efLengthAt(mid, arm, 0.04), 3*20.0, "arm arcs at r %.2f at mid-height", arm)
				require.Greater(t, efLengthAt(loops, arm-efDepth, 0.04), 3*20.0, "%s: arm arcs at r %.2f", face.name, arm-efDepth)
				require.Less(t, efLengthAt(loops, arm, 0.04), 3*3.0, "%s: no arm arc stays at r %.2f", face.name, arm)
			}
		})
	}
}
