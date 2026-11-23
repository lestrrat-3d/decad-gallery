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
	// faceTeethExempt lists the face teeth's corners: the tips are engagement edges and the roots retained sharp corners, a
	// documented exception for face teeth only ("Retained sharp corners" in instructions.md).
	faceTeethExempt := func(rMin, rMax float64) func(*run) []cornercheck.Exempt {
		return func(*run) []cornercheck.Exempt {
			return []cornercheck.Exempt{{Reason: "face teeth: tips are engagement edges, roots retained sharp corners", Optional: true,
				Near: cornercheck.Annulus(r3.Vec{Z: ShaftZ}, r3.Vec{Y: 1}, rMin, rMax)}}
		}
	}
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
		{name: "guide plate", build: (*run).columnGuidePlate, up: r3.Vec{Z: 1}, chamfered: true,
			exempt: func(r *run) []cornercheck.Exempt { return hexAnnulus(r, []int{4, 5, 6, 7}, r3.Vec{Z: 1}) }},
		{name: "cradle plate", build: (*run).cradlePlate, up: r3.Vec{Z: 1}, chamfered: true},
		{name: "top plate", build: (*run).flagTopPlate, up: r3.Vec{Z: -1}, chamfered: true},
		{name: "hub", build: (*run).hub, up: r3.Vec{Y: 1}, exempt: func(*run) []cornercheck.Exempt {
			return []cornercheck.Exempt{{Reason: "hub key roots: mating corners with the rotor skirt's slots", Optional: true,
				Near: cornercheck.Annulus(r3.Vec{Z: ShaftZ}, r3.Vec{Y: 1}, 7.9, 9.7)}}
		}},
		{name: "rotor", build: (*run).rotor, up: r3.Vec{Y: 1}, chamfered: true, exempt: faceTeethExempt(3.0, 14.4)},
		{name: "housing", build: (*run).housing, up: r3.Vec{Y: -1}, chamfered: true, exempt: faceTeethExempt(12.5, 15.0)},
		{name: "button", build: (*run).button, up: r3.Vec{Y: -1}, exempt: func(r *run) []cornercheck.Exempt {
			return append(faceTeethExempt(3.0, 11.2)(r), cornercheck.Exempt{Reason: "key lug corners: mating corners with the housing's slots", Optional: true,
				Near: cornercheck.Annulus(r3.Vec{Z: ShaftZ}, r3.Vec{Y: -1}, 11.2, 12.6)})
		}},
	}
	for _, seg := range Segments() {
		cases = append(cases, partCase{name: "cam " + seg.String(), up: r3.Vec{Y: -1}, chamfered: true,
			build: func(r *run) (*decad.Body, error) { return r.cam(seg) },
			// The engraved letter: its strokes are 0.9 mm wide, too thin for the 0.5 mm rounds of "Corners and roots",
			// so its corners stay sharp. Listed under "Retained sharp corners" in instructions.md.
			exempt: func(*run) []cornercheck.Exempt {
				return []cornercheck.Exempt{{Reason: "engraved cam letter: strokes too thin to round",
					Near: cornercheck.Disc(r3.Vec{Z: ShaftZ + inscriptionRadius}, r3.Vec{Y: -1}, Scale*4.2)}}
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
		"guide plate":  {section: alongZ, faces: two(guideFrom, guideTop)},
		"cradle plate": {section: alongZ, faces: two(cradleFrom, cradleFrom+cradleThick)},
		"top plate":    {section: alongZ, faces: two(topFrom, flagFaceZ)},
		"rotor": {section: alongY, round: true, faces: func(*run, *decad.Body) []chamferFace {
			return []chamferFace{{rotorPlateY + rotorPlateThick, -1}}
		}},
		"housing": {section: alongY, round: true, faces: func(*run, *decad.Body) []chamferFace {
			// The flange's two faces and the bezel: every large face of the housing.
			return []chamferFace{{housingBaseY - housingBaseThick, 1}, {housingBaseY, -1}, {housingBezelY, 1}}
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
