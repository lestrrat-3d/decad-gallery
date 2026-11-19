package sevensegment

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// partCase is one printed part of the display, built alone at digit 2 in the world frame, with the rule-test inputs.
type partCase struct {
	name string
	// pending, when set, is why the part cannot be built yet; its tests skip with it.
	pending string
	build   func(*run) (*decad.Body, error)
	// up is the print axis: the direction away from the bed.
	up r3.Vec
	// exempt returns the exemptions of "Corners and roots", each of which must match a finding.
	exempt func(*run) []cornercheck.Exempt
	// chamfered reports whether a face of 400 mm2 or more carries 0.5 mm chamfers on its outer loop.
	chamfered bool
}

// partCases lists every printed body: two walls, three plates, the shaft, the ratchet wheel, collar ring, knob, seven
// cams and seven sliders.
func partCases() []partCase {
	hexAnnulus := func(r *run, sites []int, axis r3.Vec) []cornercheck.Exempt {
		var out []cornercheck.Exempt
		for _, i := range sites {
			out = append(out, cornercheck.Exempt{Reason: "hex nut pocket corners match the nut's sharp corners",
				Near: cornercheck.Annulus(r.dm.Sites[i].Head, axis, 2.21, 2.81)})
		}
		return out
	}
	wallExempt := func(sites []int, front bool) func(*run) []cornercheck.Exempt {
		return func(r *run) []cornercheck.Exempt {
			var out []cornercheck.Exempt
			up := r3.Vec{Y: 1}
			if !front {
				up = r3.Vec{Y: -1}
			}
			for _, i := range sites {
				s := r.dm.Sites[i]
				out = append(out, cornercheck.Exempt{Reason: "pocket floor meets the bolt hole", Optional: true,
					Near: cornercheck.Disc(s.At(s.Cavity), up, 3.0)})
				out = append(out, cornercheck.Exempt{Reason: "head pocket floor", Optional: true,
					Near: cornercheck.Disc(s.At(2), up, 3.0)})
			}
			if front {
				out = append(out, hexAnnulus(r, []int{8, 9}, up)...)
			}
			return out
		}
	}
	cases := []partCase{
		{name: "front wall", build: func(r *run) (*decad.Body, error) { return r.wall(true) },
			up: r3.Vec{Y: 1}, exempt: wallExempt([]int{0, 1, 4, 5}, true), chamfered: true},
		{name: "rear wall", build: func(r *run) (*decad.Body, error) { return r.wall(false) }, up: r3.Vec{Y: -1},
			exempt: wallExempt([]int{2, 3, 6, 7}, false), chamfered: true},
		{name: "detent plate", build: (*run).detentPlate, up: r3.Vec{Y: 1}, chamfered: true},
		{name: "guide plate", build: (*run).columnGuidePlate, up: r3.Vec{Z: 1}, chamfered: true,
			exempt: func(r *run) []cornercheck.Exempt { return hexAnnulus(r, []int{4, 5, 6, 7}, r3.Vec{Z: 1}) }},
		{name: "cradle plate", build: (*run).cradlePlate, up: r3.Vec{Z: 1}, chamfered: true},
		{name: "top plate", build: (*run).flagTopPlate, up: r3.Vec{Z: -1}, chamfered: true},
		{name: "ratchet wheel", build: (*run).ratchetWheel, up: r3.Vec{Y: 1},
			exempt: func(r *run) []cornercheck.Exempt {
				var out []cornercheck.Exempt
				for j := range RatchetTeeth {
					tip := rotate(r3.Vec{X: RatchetTip}, r.dm.LockPhase+36*float64(j))
					out = append(out, cornercheck.Exempt{Reason: "ratchet tooth tip: engagement corner",
						Near: cornercheck.Disc(r3.Vec{X: tip.X, Z: ShaftZ + tip.Y}, r3.Vec{Y: 1}, 0.6)})
				}
				return out
			}},
		{name: "collar ring", build: (*run).collarRing, up: r3.Vec{Y: 1}},
		{name: "knob", build: (*run).knob, up: r3.Vec{Y: 1}, chamfered: true},
	}
	for _, seg := range Segments() {
		cases = append(cases, partCase{name: "cam " + seg.String(), up: r3.Vec{Y: -1}, chamfered: true,
			build: func(r *run) (*decad.Body, error) { return r.cam(seg) },
			// The engraved letter: its strokes are 0.9 mm wide, too thin for the 0.5 mm rounds of "Corners and roots",
			// so its corners stay sharp. Listed under "Retained sharp corners" in instructions.md.
			exempt: func(*run) []cornercheck.Exempt {
				return []cornercheck.Exempt{{Reason: "engraved cam letter: strokes too thin to round",
					Near: cornercheck.Disc(r3.Vec{Z: ShaftZ + inscriptionRadius}, r3.Vec{Y: -1}, 4.2)}}
			}})
	}
	for _, seg := range Segments() {
		// a, g, d print on their -X face and b-f on their -Y face.
		up := r3.Vec{Y: 1}
		if _, _, horizontal := segmentCentre(seg); horizontal {
			up = r3.Vec{X: 1}
		}
		cases = append(cases, partCase{name: "slider " + seg.String(), up: up,
			build: func(r *run) (*decad.Body, error) { return r.flagSlider(seg) }})
	}
	return cases
}

// TestSevenSegmentCorners applies "Corners and roots" to every printed part in its print orientation: only the listed
// exemptions are allowed, and the shaft is read in TestSevenSegmentSplitShaft.
func TestSevenSegmentCorners(t *testing.T) {
	for _, c := range partCases() {
		t.Run(c.name, func(t *testing.T) {
			if c.pending != "" {
				t.Skip(c.pending)
			}
			r, body := sharedBody(t, c.name, c.build)
			var exempt []cornercheck.Exempt
			if c.exempt != nil {
				exempt = c.exempt(r)
			}
			fs, err := cornercheck.Check(t.Context(), body, c.up, exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 8))
		})
	}
}

// largeFace is the net area in mm2 at which "Surface edge finishing" asks for chamfers.
const largeFace = 400.0

// largePlanes returns the number of planar faces of body of at least largeFace mm2.
func largePlanes(t *testing.T, body *decad.Body) int {
	t.Helper()
	large := 0
	for _, f := range body.Faces() {
		if _, ok := f.Surface().(decad.Plane); !ok {
			continue
		}
		area, err := f.Area()
		require.NoError(t, err)
		if area.Value.Base() >= largeFace {
			large++
		}
	}
	return large
}

// chamferFace is one face of a part whose outer loop carries the 0.5 mm chamfer: its level on the part's axis, and
// the sign of the direction from the face into the part.
type chamferFace struct {
	level  float64
	inward float64
}

// chamferSpec describes how to measure the chamfers of one part: the sections of the part normal to its axis and the
// faces whose outer loops are chamfered. A part with round rims is read by radius, any other by the four sides of its
// outline.
type chamferSpec struct {
	section func(t *testing.T, body *decad.Body, h float64) [][]r3.Vec
	faces   func(r *run, body *decad.Body) []chamferFace
	round   bool
}

// chamferSpecs lists the parts that carry chamfers on the outer loops of their faces. It is read from the geometry
// the builders use: the wall, plate and cam levels follow the constants the builders use.
func chamferSpecs() map[string]chamferSpec {
	alongY := func(t *testing.T, b *decad.Body, y float64) [][]r3.Vec { return sectionAt(t, b, y) }
	alongZ := func(t *testing.T, b *decad.Body, z float64) [][]r3.Vec { return planSection(t, b, z) }
	two := func(lo, hi float64) func(*run, *decad.Body) []chamferFace {
		return func(*run, *decad.Body) []chamferFace { return []chamferFace{{lo, 1}, {hi, -1}} }
	}
	specs := map[string]chamferSpec{
		"front wall":   {section: alongY, faces: two(frontInnerY-WallThickness, frontInnerY)},
		"rear wall":    {section: alongY, faces: two(rearInnerY, rearInnerY+WallThickness)},
		"detent plate": {section: alongY, faces: two(detentFace-DetentThickness, detentFace)},
		"guide plate":  {section: alongZ, faces: two(54, 58)},
		"cradle plate": {section: alongZ, faces: two(cradleFrom, cradleFrom+cradleThick)},
		"top plate":    {section: alongZ, faces: two(topFrom, flagFaceZ)},
		"knob": {section: alongY, round: true, faces: func(_ *run, b *decad.Body) []chamferFace {
			box, _ := b.Bounds()
			return []chamferFace{{box.Min.Y, 1}, {box.Max.Y, -1}}
		}},
	}
	for _, seg := range Segments() {
		specs["cam "+seg.String()] = chamferSpec{section: alongY, round: true,
			faces: func(r *run, _ *decad.Body) []chamferFace {
				face := r.dm.camDiscFace(seg)
				return []chamferFace{{face, 1}, {face + CamThickness, -1}}
			}}
	}
	return specs
}

// outlineReach returns the extreme coordinates of the loops of a section: the least and greatest x and y, and the
// greatest radius about the origin.
func outlineReach(loops [][]r3.Vec) (minX, maxX, minY, maxY, maxR float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, l := range loops {
		for _, p := range l {
			minX, maxX = math.Min(minX, p.X), math.Max(maxX, p.X)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
			maxR = math.Max(maxR, p.Len())
		}
	}
	return
}

// TestSevenSegmentEdgeFinishing checks "Surface edge finishing" on the built bodies. Each face loop that carries the
// 0.5 mm chamfer is read from two sections, 0.1 and 1.0 inside its face: the chamfer pulls the outline in by
// 0.5 - 0.1 = 0.4 at the first, on every side of a plate and all round a cam or the knob. Every other part has no
// planar face of 400 mm2 or more.
func TestSevenSegmentEdgeFinishing(t *testing.T) {
	specs := chamferSpecs()
	for _, c := range partCases() {
		t.Run(c.name, func(t *testing.T) {
			if c.pending != "" {
				t.Skip(c.pending)
			}
			r, body := sharedBody(t, c.name, c.build)
			spec, ok := specs[c.name]
			require.Equal(t, c.chamfered, ok, "the part's chamfer flag and its chamfer spec agree")
			if !ok {
				require.Zero(t, largePlanes(t, body), "no face of 400 mm2 or more needs a chamfer")
				return
			}
			faces := spec.faces(r, body)
			require.Len(t, faces, 2, "both faces carry a chamfer")
			for _, f := range faces {
				nMinX, nMaxX, nMinY, nMaxY, nR := outlineReach(spec.section(t, body, f.level+f.inward*0.1))
				dMinX, dMaxX, dMinY, dMaxY, dR := outlineReach(spec.section(t, body, f.level+f.inward*1.0))
				const want = 0.5 - 0.1
				if spec.round {
					require.InDelta(t, want, dR-nR, 0.05, "the rim at face %.2f is chamfered all round", f.level)
					continue
				}
				require.InDelta(t, want, nMinX-dMinX, 0.05, "the -x side at face %.2f is chamfered", f.level)
				require.InDelta(t, want, dMaxX-nMaxX, 0.05, "the +x side at face %.2f is chamfered", f.level)
				require.InDelta(t, want, nMinY-dMinY, 0.05, "the -y side at face %.2f is chamfered", f.level)
				require.InDelta(t, want, dMaxY-nMaxY, 0.05, "the +y side at face %.2f is chamfered", f.level)
			}
		})
	}
}

// markCount counts the 1.5 mm round holes on a circle of radius about the axis in the loops of a section.
func markCount(loops [][]r3.Vec, radius float64) int {
	n := 0
	for _, loop := range loops {
		c, ext := centroidExtent(loop)
		if ext < 1.6 && c.Len() > radius-0.1 && c.Len() < radius+0.1 {
			n++
		}
	}
	return n
}

// TestSevenSegmentLookAlike checks every look-alike set: the counts differ within a set, and each hole goes through.
func TestSevenSegmentLookAlike(t *testing.T) {
	type set struct {
		name   string
		parts  []string
		counts []int
	}
	sets := []set{
		{"front cams", []string{"cam d", "cam e", "cam c"}, []int{1, 2, 3}},
		{"rear cams", []string{"cam g", "cam b", "cam f", "cam a"}, []int{1, 2, 3, 4}},
		{"walls", []string{"front wall", "rear wall"}, []int{1, 2}},
	}
	cases := map[string]partCase{}
	for _, c := range partCases() {
		cases[c.name] = c
	}
	for _, s := range sets {
		seen := map[int]bool{}
		for _, n := range s.counts {
			require.False(t, seen[n], "set %s repeats a count", s.name)
			seen[n] = true
		}
		for i, name := range s.parts {
			c := cases[name]
			t.Run(name, func(t *testing.T) {
				if c.pending != "" {
					t.Skip(c.pending)
				}
				_, body := sharedBody(t, c.name, c.build)
				box, err := body.Bounds()
				require.NoError(t, err)
				radius := 10.0
				switch name {
				case "cam d", "cam e", "cam c":
					radius = 12
				case "cam g", "cam b", "cam f", "cam a":
					radius = 8
				}
				// Sections just inside each face of the plate or disc show every hole: it goes through the whole part.
				ys := []float64{box.Max.Y - 0.3, box.Max.Y - CamThickness/2, box.Max.Y - CamThickness + 0.3}
				switch name {
				case "front wall":
					ys = []float64{frontInnerY - 0.3, frontInnerY - WallThickness/2, frontInnerY - WallThickness + 0.3}
				case "rear wall":
					ys = []float64{rearInnerY + 0.3, rearInnerY + WallThickness/2, rearInnerY + WallThickness - 0.3}
				}
				for _, y := range ys {
					loops := sectionAt(t, body, y)
					require.Equal(t, s.counts[i], markCount(loops, radius), "%s at y %.2f", name, y)
				}
			})
		}
	}
}

// sectionWidth is the x extent of the horizontal section of body at height z.
func sectionWidth(t *testing.T, body *decad.Body, z float64) float64 {
	t.Helper()
	loops := planSection(t, body, z)
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, l := range loops {
		for _, p := range l {
			lo, hi = math.Min(lo, p.X), math.Max(hi, p.X)
		}
	}
	return hi - lo
}
