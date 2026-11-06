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
// at X 7, Y -23.5, 0.1 below the gearbox floor so that the two parts do not touch before the screws are tightened.
const (
	gripRake      = 15 * math.Pi / 180
	gripMid       = 68.0           // height of the section centres' local origin
	gripDrop      = boxFloor - 128 // the grip hangs from the lowered floor, so it is built 128 - boxFloor higher than it sits
	gripScrewX    = 9.0            // the two screws sit this far to each side of the grip's centre
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
		xf, err := r3.Translation(r3.Vec{X: gripX, Y: gripY - (128-gripMid)*math.Tan(gripRake), Z: gripDrop})
		if err != nil {
			b.fail("create grip placement", err)
			return nil
		}
		if loft, err = decad.Transform(b.ctx, loft, xf); err != nil {
			b.fail("place grip", err)
			return nil
		}
		for _, dx := range []float64{-gripScrewX, gripScrewX} {
			pilot := b.cylinder(frame(r3.Vec{X: gripX + dx, Y: gripY, Z: gripPilotLow + gripDrop}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 0.9, 13)
			loft = b.cut(loft, pilot)
		}
		return loft
	})
	for i, dx := range []float64{-gripScrewX, gripScrewX} {
		b.part(fmt.Sprintf("handle-bolt-%d", i+1), "hardware", "fixed", func() *decad.Body {
			return b.screw(r3.Vec{X: gripX + dx, Y: gripY, Z: cavityFloor + 0.1}, r3.Vec{Z: -1}, r3.Vec{X: 1}, 16, 0, 0.8)
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

// Gearbox dimensions in mm. The gearbox is one tray: a floor, a fan-side wall and a crank-side wall, with a post on its
// left end and a back wall, open at the top, so the gears drop in from above. The floor is low enough for the crank gear
// to sit above it, so the spur pair lives in the tray, left of the bevel gear, and the grip hangs from the lowered
// floor. Each shaft turns in a square bearing block built out from a wall, which makes a long bearing without a boss or
// a sleeve. Printed floor down, every wall, the post and the blocks are vertical, so nothing needs support.
const (
	wallThick    = 9.8
	crankWallX   = drivingHeel + 0.2 // inner face of the crank-side wall, 0.2 from the driving gear's back face
	fanWallY     = -pinionHeel - 0.2 // inner face of the fan-side wall, 0.2 from the pinion's back face
	backWallY    = 29.2              // inner face of the back wall, 0.4 beyond the driving gear's reach
	boxTop       = 190.0             // the drive shaft's bearing seat is a teardrop whose point is at 185.7, so the walls keep about 4 above it
	boxBackY     = backWallY + wallThick
	boxLeftX     = -20.0
	boxFloor     = crankHeight - gearTip - 1.2 - 5 // the floor's underside: 1.2 under the crank gear, and 5 thick
	cavityFloor  = boxFloor + 5
	blockHalf    = 15.0                                // the bearing blocks and the post are 30 wide and as tall as the tray
	fanBlock     = 24.0                                // the fan block sticks out this far from the fan-side wall's outer face
	crankBlock   = 14.0                                // and the crank block this far from the crank-side wall's outer face
	fanGap       = fanBlock + 10.5                     // from the fan-side wall's outer face to the hub
	crankFace    = crankWallX + wallThick + crankBlock // the crank block's end face, X 48
	fanBlockFace = -fanWallY + wallThick + fanBlock    // the fan block's end face, 70 behind the apex
	gripY        = fanWallY + 8.2                      // the grip's centre sits this far behind the fan-side wall's inner face
	gripX        = 7.0                                 // the grip's centre, under the tray
)

// gearbox builds a rounded block minus a cavity, which leaves the floor, the fan-side wall and the crank-side wall,
// joins the two bearing blocks, then bores the shaft holes and the two screw holes through it. The cavity
// is open at the top and the back, and at both ends until the post closes the left one.
func (b *builder) gearbox() {
	b.part("gearbox", "housing", "fixed", func() *decad.Body {
		// The block is a rounded outline in (Y, Z) extruded along +X, with its two end faces rounded too.
		block := b.roundedPrism(frame(r3.Vec{X: boxLeftX}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{fanWallY - wallThick, boxFloor}, {boxBackY, boxFloor}, {boxBackY, boxTop}, {fanWallY - wallThick, boxTop}},
			[]float64{3, 3, 3, 3}, crankWallX+wallThick-boxLeftX)
		block = b.roundCaps(block, 2)
		cavity := b.roundedPrism(frame(r3.Vec{X: boxLeftX - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
			[]point2{{fanWallY, cavityFloor}, {backWallY, cavityFloor}, {backWallY, boxTop + 1}, {fanWallY, boxTop + 1}},
			[]float64{3, 3, 0, 0}, crankWallX-boxLeftX+1)
		cavity = b.roundEnd(cavity, 2)
		// The post is the part of the block the cavity leaves: a notch in the cavity, on the left end, 22 wide.
		notch := b.panel(frame(r3.Vec{X: boxLeftX - 2}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), -blockHalf, cavityFloor-1, backWallY+1, boxTop+2,
			postFace-boxLeftX+2)
		cavity = b.cut(cavity, notch)
		body := b.cut(block, cavity)
		// Each bearing block starts inside the tray so the union has no shared face.
		square := []point2{{-blockHalf, boxFloor}, {blockHalf, boxFloor}, {blockHalf, boxTop}, {-blockHalf, boxTop}}
		radii := []float64{3, 3, 3, 3}
		fanPad := b.roundedPrism(frame(r3.Vec{Y: fanWallY - wallThick + 1}, r3.Vec{X: 1}, r3.Vec{Z: 1}), square, radii, fanBlock+1)
		fanPad = b.roundEnd(fanPad, 2)
		crankPad := b.roundedPrism(frame(r3.Vec{X: crankWallX + wallThick - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), square, radii, crankBlock+1)
		crankPad = b.roundEnd(crankPad, 2)
		if b.err != nil {
			return nil
		}
		body = b.union(body, fanPad, crankPad)
		// A 608 bearing sits where each shaft crosses a wall. The shafts go in through the post from the left. The drive
		// shaft ends in the crank-side wall's bearing, and the crank shaft goes through that wall and its block. The seats
		// open at the post's outer face and the block ends and in the walls' inner faces, so each bearing drops in
		// along its shaft.
		for _, a := range []shaftAxis{crankSpline, driveSpline} {
			body = b.seat(body, a, boxLeftX-1, boxLeftX+seatDepth, boxLeftX+7, postFace+1)
			body = b.seat(body, a, crankWallX-1, crankWallX+seatDepth, crankWallX+seatDepth-0.1, crankWallX+seatDepth+reliefDepth-0.1)
		}
		body = b.hole(body, crankSpline, shaftHole, crankWallX+9.3, crankFace-8.5)
		body = b.seat(body, crankSpline, crankFace-seatDepth, crankFace+1, crankFace-seatDepth-reliefDepth+0.1, crankFace-seatDepth+0.1)
		// The fan shaft's two bearings are measured behind the apex.
		body = b.seat(body, fanSpline, -fanWallY-1, -fanWallY+seatDepth, -fanWallY+seatDepth-0.1, -fanWallY+seatDepth+reliefDepth-0.1)
		body = b.hole(body, fanSpline, shaftHole, -fanWallY+9.3, fanBlockFace-8.5)
		body = b.seat(body, fanSpline, fanBlockFace-seatDepth, fanBlockFace+1, fanBlockFace-seatDepth-reliefDepth+0.1, fanBlockFace-seatDepth+0.1)
		for _, dx := range []float64{-gripScrewX, gripScrewX} {
			body = b.cut(body, b.cylinder(frame(r3.Vec{X: gripX + dx, Y: gripY, Z: boxFloor - 1}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 1.2, 8))
		}
		return body
	})
}
