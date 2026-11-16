package onewayclutch_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

const (
	// efDepth is how far inside a hub face the section is taken. A 45 degree chamfer of HubChamfer offsets the outline
	// by HubChamfer - efDepth there: 0.15 mm.
	efDepth = 0.1
	// efSharp and efOffset are how far a point of the mid-height outline may lie from a face section to read as
	// unchanged, and as offset by the chamfer.
	efSharp  = 0.05
	efOffset = 0.05
	// efStep is the spacing of the points sampled along the mid-height outline.
	efStep = 0.05
)

// efSection is the hub's outline at z, flattened to z = 0 so that outlines at different heights compare in the plane.
func efSection(t *testing.T, body *decad.Body, z float64) [][]r3.Vec {
	t.Helper()
	s := fitSection(t, body, fitPlane(t, z))
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
			case math.Abs(d-(onewayclutch.HubChamfer-efDepth)) < efOffset:
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

// efArea is the area of the hub face in mm²: the outer loop less the D bore and the three bolt holes.
func efArea(loops [][]r3.Vec) float64 {
	total := 0.0
	for _, loop := range loops {
		a := 0.0
		for i, p := range loop {
			q := loop[(i+1)%len(loop)]
			a += p.X*q.Y - q.X*p.Y
		}
		a = math.Abs(a) / 2
		if efKindOf(loop) == efOuter {
			total += a
		} else {
			total -= a
		}
	}
	return total
}

// efFlankTaper is the length of the loops' segments that lie on a pawl's flank line inside the arm's outer radius, in
// mm: the bevel's taper, which follows the flank's plane carried back past the arm's corner. The line is the head's
// flank at distance FlankDistance from the axis; the built direction mirrors it and each pawl is turned 120 degrees.
func efFlankTaper(m *onewayclutch.Model, loops [][]r3.Vec) float64 {
	dm := m.Dimensions
	theta := dm.RampAngle * math.Pi / 180
	n := r3.Vec{X: math.Sin(theta), Y: -math.Cos(theta)}
	if n.Dot(dm.FlankEnd) < 0 {
		n = n.Scale(-1)
	}
	n.X *= dm.Sign
	total := 0.0
	for k := range onewayclutch.Pawls {
		s, c := math.Sincos(-120 * float64(k) * math.Pi / 180)
		on := func(p r3.Vec) bool {
			q := r3.Vec{X: p.X*c - p.Y*s, Y: p.X*s + p.Y*c}
			return math.Abs(q.Dot(n)-dm.FlankDistance) < 0.005
		}
		for _, loop := range loops {
			for i, a := range loop {
				if b := loop[(i+1)%len(loop)]; on(a) && on(b) && math.Hypot(a.X+b.X, a.Y+b.Y)/2 < onewayclutch.ArmOuter-0.02 {
					total += b.Sub(a).Len()
				}
			}
		}
	}
	return total
}

// TestOneWayClutchEdgeFinishing checks the hub's chamfer on both faces in sections. The 605ZZ hub's faces are over
// 400 mm², so each has a 0.25 mm chamfer: the section 0.1 mm inside a face is the mid-height outline pulled in 0.15 mm
// along the arm's outer arc and each inner run, with the bevel tapering out on the head flank's plane, 0.911 mm along
// it from the arm's corner, and it leaves the head run, the bolt holes and the short D bore unchanged. The 8 mm hubs'
// faces are under 400 mm², so they are the exact prism and every section reads the same. A volume check would not do;
// the faceted result reads about 0.07 mm3 high per arc run.
func TestOneWayClutchEdgeFinishing(t *testing.T) {
	for _, c := range fitCases()[:6] {
		t.Run(c.name, func(t *testing.T) {
			m := fitBuild(t, c)
			chamfered := m.Dimensions.Shaft.Name == "605ZZ"
			mid := efSection(t, m.Hub, midHub)
			area := efArea(mid)
			t.Logf("%s hub face %.1f mm2", c.name, area)
			if chamfered {
				require.Greater(t, area, edgefinish.LargeFaceArea, "the 605ZZ hub face is large")
			} else {
				require.Less(t, area, edgefinish.LargeFaceArea, "an 8 mm hub face is small")
			}
			for _, face := range []struct {
				name string
				z    float64
			}{
				{"back face", onewayclutch.HubFrom + efDepth},
				{"front face", onewayclutch.HubFrom + onewayclutch.HubHeight - efDepth},
			} {
				loops := efSection(t, m.Hub, face.z)
				require.Len(t, loops, len(mid), "%s: loop count", face.name)
				if !chamfered {
					for _, loop := range mid {
						got := efMeasure(loop, loops)
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: an 8 mm hub is not chamfered", face.name)
					}
					continue
				}
				var outer efShift
				for _, loop := range mid {
					got := efMeasure(loop, loops)
					switch efKindOf(loop) {
					case efHole:
						require.Greater(t, got.unchanged, 0.99*got.total(), "%s: a bolt hole is not chamfered", face.name)
					case efBore:
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

				// The arm's outer arc reads r 13.2 at mid-height and r 13.05 in the face section.
				arm := onewayclutch.ArmOuter
				chamfer := onewayclutch.HubChamfer - efDepth
				require.Greater(t, efLengthAt(mid, arm, 0.04), 3*20.0, "arm arcs at r %.2f at mid-height", arm)
				require.Greater(t, efLengthAt(loops, arm-chamfer, 0.04), 3*18.0, "%s: arm arcs at r %.2f", face.name, arm-chamfer)
				require.Less(t, efLengthAt(loops, arm, 0.04), 3*3.0, "%s: no arm arc stays at r %.2f", face.name, arm)

				// The bevel ends on the flank's plane: the section follows the flank line 0.911 mm past the arm's corner.
				taper := efFlankTaper(m, loops)
				require.Zero(t, efFlankTaper(m, mid), "%s: no taper at mid-height", face.name)
				t.Logf("%s %s: taper along the flank's plane, three pawls: %.3f mm", c.name, face.name, taper)
				// The faceted section reads 0.80 to 0.83 mm per pawl, 0.1 mm short of the exact 0.911; a missing taper or a square
				// end reads 0 or 0.5.
				require.InDelta(t, 3*0.911, taper, 3*0.15, "%s: the bevel tapers out along the flank's plane", face.name)
			}
		})
	}
}
