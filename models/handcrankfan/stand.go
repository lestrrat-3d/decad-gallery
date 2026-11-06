package handcrankfan

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Grip dimensions in mm. The grip is an oval loft raked 15° with its base toward the operator. Its top centre is
// at X 7, Y -28, 0.1 below the gearbox floor so that the two parts do not touch before the screws are tightened.
const (
	gripRake      = 15 * math.Pi / 180
	gripMid       = 68.0 // height of the section centres' local origin
	gripFloor     = 128.0
	gripScrewX    = 9.0 // the two screws sit this far to each side of the grip's centre
	gripPilotLow  = 116.0
	gripFitPoints = 25
	gripFitGap    = 2 * math.Pi / 180 // the back of each section is closed by two short lines
)

// gripStation is one oval section: height, width along X and length along Y.
type gripStation struct{ z, width, length float64 }

var gripStations = [...]gripStation{
	{8, 28, 38}, {9, 33, 43}, {11, 37, 47}, {14, 36, 46}, {18, 32, 41}, {26, 31, 39}, {45, 32, 40.5},
	{65, 33.5, 42}, {85, 34, 42.5}, {100, 33, 41}, {114, 31, 38}, {122, 31.5, 36.5}, {126, 33, 35.5}, {127.9, 35, 35.5},
}

// handle builds the grip held in the left hand: an oval loft with a flared base, a palm swell and a flared top, and
// two M2 x 16 screws that hold it to the gearbox floor. The screws cut their own thread in two pilot holes.
func (b *builder) handle() {
	b.part("handle", "handle", "fixed", func() *decad.Body {
		w := sketch.NewWorld()
		sections := make([]decad.LoftSection, len(gripStations))
		for i, st := range gripStations {
			sections[i] = b.gripSection(w, st)
		}
		if b.err != nil {
			return nil
		}
		loft, err := decad.LoftSections(b.ctx, b.doc, sections...)
		if err != nil {
			b.fail("loft grip", err)
			return nil
		}
		// The sections are drawn about the centre of the grip's mid height, so each sees the origin along rays.
		xf, err := r3.Translation(r3.Vec{X: 7, Y: -28 - (gripFloor-gripMid)*math.Tan(gripRake)})
		if err != nil {
			b.fail("create grip placement", err)
			return nil
		}
		if loft, err = decad.Transform(b.ctx, loft, xf); err != nil {
			b.fail("place grip", err)
			return nil
		}
		for _, dx := range []float64{-gripScrewX, gripScrewX} {
			pilot := b.cylinder(frame(r3.Vec{X: 7 + dx, Y: -28, Z: gripPilotLow}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 0.9, 13)
			loft = b.cut(loft, pilot)
		}
		return loft
	})
	for i, dx := range []float64{-gripScrewX, gripScrewX} {
		b.part(fmt.Sprintf("handle-bolt-%d", i+1), "hardware", "fixed", func() *decad.Body {
			return b.screw(r3.Vec{X: 7 + dx, Y: -28, Z: gripFloor + 6.1}, r3.Vec{Z: -1}, r3.Vec{X: 1}, 16, 0, 0.8)
		})
	}
}

// gripSection sketches one oval on the world XY plane at height st.z: a fit spline through 25 points from just past
// the back to just before it, closed by two lines that meet at the midpoint of the gap.
func (b *builder) gripSection(w *sketch.World, st gripStation) decad.LoftSection {
	if b.err != nil {
		return decad.LoftSection{}
	}
	plane, err := w.CreateOffsetPlane(w.XY(), st.z)
	if err != nil {
		b.fail("create grip plane", err)
		return decad.LoftSection{}
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		b.fail("create grip section", err)
		return decad.LoftSection{}
	}
	b.a.Sketches = append(b.a.Sketches, sk)
	yc := (st.z - gripMid) * math.Tan(gripRake)
	fit := make([]*sketch.Point, gripFitPoints)
	for k := range fit {
		phi := -math.Pi/2 + gripFitGap + float64(k)*(2*math.Pi-2*gripFitGap)/float64(gripFitPoints-1)
		fit[k] = sketchcheck.DimensionPoint(sk, st.width/2*math.Cos(phi), yc+st.length/2*math.Sin(phi))
	}
	first, last := fit[0], fit[len(fit)-1]
	tail := sk.CreatePoint(0, yc-st.length/2*math.Cos(gripFitGap))
	sk.AddConstraint(sketch.NewMidpointOf(tail, first, last))
	if _, err := sk.CreateFitSpline(fit...); err != nil {
		b.fail("create grip spline", err)
		return decad.LoftSection{}
	}
	sk.CreateLine(last, tail)
	sk.CreateLine(tail, first)
	if _, err := sk.Solve(b.ctx); err != nil {
		b.fail("solve grip section", err)
		return decad.LoftSection{}
	}
	if err := sketchcheck.VerifyFullyConstrained(b.ctx, sk); err != nil {
		b.fail("verify grip section", fmt.Errorf("height %g: %w", st.z, err))
		return decad.LoftSection{}
	}
	for _, profile := range sk.Profiles() {
		if profile.Valid && len(profile.Holes) == 0 && len(profile.Outer) == 3 {
			return decad.LoftSection{Sketch: sk, Profile: profile}
		}
	}
	b.fail("select grip profile", fmt.Errorf("height %g: no valid profile of three edges", st.z))
	return decad.LoftSection{}
}

// Gearbox dimensions in mm. The gearbox is a tray with a floor, a fan-side wall and a crank-side wall, open at the top,
// the back and the left, so the gears drop in from above. Each shaft turns in a hole through its wall and a square
// bearing block built out from the wall, which makes a bearing 24 to 34 long without a boss or a sleeve. Printed floor
// down, every wall and both blocks are vertical, so nothing needs support.
const (
	wallThick   = 9.8
	crankWallX  = 24.2  // inner face of the crank-side wall, 0.2 from the driving gear's back face at X 24
	fanWallY    = -36.2 // inner face of the fan-side wall, 0.2 from the pinion's back face at Y -36
	boxFloor    = 128.0
	boxTop      = 181.0 // the bores' tops are at 174.2, so the walls keep about 7 above them
	boxBackY    = 11.0  // nothing but the floor and the crank-side wall lies behind the crank axis
	boxLeftX    = -20.0
	cavityFloor = 134.0
	blockHalf   = 11.0                                // both bearing blocks are 22 wide and as tall as the box
	fanBlock    = 24.0                                // the fan block sticks out this far from the fan-side wall's outer face
	crankBlock  = 14.0                                // and the crank block this far from the crank-side wall's outer face
	fanGap      = fanBlock + 6                        // from the fan-side wall's outer face to the hub
	crankFace   = crankWallX + wallThick + crankBlock // the crank block's end face, X 48
)

// gearbox builds a rounded block minus a cavity, which leaves the floor, the fan-side wall and the crank-side wall,
// joins the two bearing blocks, then bores the two shaft holes and the two screw holes through it. The cavity is open at
// the top, the back and the left, so the gears drop in from above and the shafts slide in through the walls.
func (b *builder) gearbox() {
	b.part("gearbox", "housing", "fixed", func() *decad.Body {
		// The block is a rounded outline in (Y, Z) extruded along +X, with its two end faces rounded too.
		block := b.roundedPrism(frame(r3.Vec{X: boxLeftX}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{fanWallY - wallThick, boxFloor}, {boxBackY, boxFloor}, {boxBackY, boxTop}, {fanWallY - wallThick, boxTop}},
			[]float64{3, 3, 3, 3}, crankWallX+wallThick-boxLeftX)
		block = b.roundCaps(block, 2)
		cavity := b.roundedPrism(frame(r3.Vec{X: boxLeftX - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{fanWallY, cavityFloor}, {boxBackY + 1, cavityFloor}, {boxBackY + 1, boxTop + 1}, {fanWallY, boxTop + 1}},
			[]float64{3, 0, 0, 0}, crankWallX-boxLeftX+1)
		cavity = b.roundEnd(cavity, 2)
		body := b.cut(block, cavity)
		// Each bearing block starts 1 inside its wall so the union has no shared face.
		fanPad := b.roundedPrism(frame(r3.Vec{Y: fanWallY - wallThick + 1}, r3.Vec{X: 1}, r3.Vec{Z: 1}),
			[]point2{{-blockHalf, boxFloor}, {blockHalf, boxFloor}, {blockHalf, boxTop}, {-blockHalf, boxTop}},
			[]float64{3, 3, 3, 3}, fanBlock+1)
		fanPad = b.roundEnd(fanPad, 2)
		crankPad := b.roundedPrism(frame(r3.Vec{X: crankWallX + wallThick - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{-blockHalf, boxFloor}, {blockHalf, boxFloor}, {blockHalf, boxTop}, {-blockHalf, boxTop}},
			[]float64{3, 3, 3, 3}, crankBlock+1)
		crankPad = b.roundEnd(crankPad, 2)
		if b.err != nil {
			return nil
		}
		body = b.union(body, fanPad, crankPad)
		body = b.cut(body, b.cylinder(alongX(20), 4.2, 38))
		body = b.cut(body, b.cylinder(alongMinusY(-30), 3.2, 74))
		for _, dx := range []float64{-gripScrewX, gripScrewX} {
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: 7 + dx, Y: -28, Z: 127}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 1.2, 8))
		}
		return body
	})
}
