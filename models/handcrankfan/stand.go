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

// Gearbox dimensions in mm. The gearbox is a tray: a floor, a fan-side wall, a crank-side wall, a back wall and a post
// on its left end, open at the top, so the gears drop in from above. The floor is low enough for the crank gear to sit
// above it, so the spur pair lives in the tray, left of the bevel gear, and the grip hangs from the lowered floor. Each
// shaft turns in a square bearing block built out from a wall, which makes a long bearing without a boss or a sleeve.
//
// It prints in two halves, split along the plane Y = 0 that holds the crank shaft and the drive shaft. Each of their
// bearing seats is cut across the split, so each half holds half of it as an open trough, and no bore lies on its side.
// A half prints with the split face up: the front half on its fan block, the back half on the back wall. The fan shaft
// runs along Y, so its seats are vertical holes in the front half. Four ears with M2 screws and nuts hold the halves
// together.
const (
	wallThick    = 9.8
	crankWallX   = drivingHeel + 0.2 // inner face of the crank-side wall, 0.2 from the driving gear's back face
	fanWallY     = -pinionHeel - 0.2 // inner face of the fan-side wall, 0.2 from the pinion's back face
	backWallY    = 29.2              // inner face of the back wall, 0.3 beyond the driving gear's reach
	boxTop       = 190.0             // the top of every wall
	boxBackY     = backWallY + wallThick
	boxLeftX     = -20.0
	boxFloor     = crankHeight - gearTip - 1.2 - 5 // the floor's underside: 1.2 under the crank gear, and 5 thick
	cavityFloor  = boxFloor + 5
	blockHalf    = 15.0                                // the bearing blocks are 30 wide at their ends and as tall as the tray
	fanBlock     = 24.0                                // the fan block sticks out this far from the fan-side wall's outer face
	crankBlock   = 14.0                                // and the crank block this far from the crank-side wall's outer face
	fanGap       = fanBlock + 10.5                     // from the fan-side wall's outer face to the hub
	crankFace    = crankWallX + wallThick + crankBlock // the crank block's end face, X 45
	fanBlockFace = -fanWallY + wallThick + fanBlock    // the fan block's end face, 65.5 behind the apex
	gripY        = fanWallY + 8.2                      // the grip's centre sits this far behind the fan-side wall's inner face
	gripX        = 7.0                                 // the grip's centre, under the tray

	splitGap = 0.1 // each half stops this far from the split plane
	earHalf  = 8.0 // an ear reaches this far to each side of the split plane
	earLen   = 8.0 // and sticks out this far from the tray
	earBolt  = 12.0
	earHole  = 1.3 // the ear screw's clearance hole, diameter 2.6
)

// gearboxEars are the four ears: two under the floor and two on the tops of the post and the crank-side wall, which
// leaves the grip, the crank's collar and the arm clear.
var gearboxEars = [...]struct{ x0, x1, z0, z1 float64 }{
	{-19, -11, boxFloor - earLen, boxFloor + 1},
	{24.8, 31, boxFloor - earLen, boxFloor + 1},
	{-19, -11, boxTop - 1, boxTop + earLen},
	{22, 30, boxTop - 1, boxTop + earLen},
}

// earAxis is the line of ear i's screw: its hole runs along Y through the ear's middle.
func earAxis(i int) (x, z float64) {
	e := gearboxEars[i]
	if i < 2 {
		return (e.x0 + e.x1) / 2, e.z0 + earLen/2
	}
	return (e.x0 + e.x1) / 2, e.z1 - earLen/2
}

// gearbox builds the housing as one body, cuts it across the split plane into two halves and adds the screws that
// hold them together.
func (b *builder) gearbox() {
	var back *decad.Body
	b.part("gearbox-front", "housing", "fixed", func() *decad.Body {
		body := b.housing()
		if b.err != nil {
			return nil
		}
		other, err := decad.Copy(b.ctx, body)
		b.fail("copy the housing", err)
		if b.err != nil {
			return nil
		}
		// The front half keeps Y <= -0.1 and the back half Y >= 0.1.
		front := b.cut(body, b.panel(frame(r3.Vec{Y: 100}, r3.Vec{X: 1}, r3.Vec{Z: 1}), -60, 60, 80, 230, 100+splitGap))
		back = b.cut(other, b.panel(frame(r3.Vec{Y: -100}, r3.Vec{Z: 1}, r3.Vec{X: 1}), 60, -60, 230, 80, 100+splitGap))
		return front
	})
	b.part("gearbox-back", "housing", "fixed", func() *decad.Body { return back })
	for i := range gearboxEars {
		b.part(fmt.Sprintf("gearbox-bolt-%d", i+1), "hardware", "fixed", func() *decad.Body {
			x, z := earAxis(i)
			return b.screw(r3.Vec{X: x, Y: -earHalf + counterboreDepth - hardwareGap, Z: z}, r3.Vec{Y: 1}, r3.Vec{Z: 1},
				earBolt, 2*earHalf-trapDepth+0.2-counterboreDepth+hardwareGap, 1.0)
		})
	}
}

// housing builds the whole tray before it is split: a rounded block minus a cavity, the two bearing blocks, the ears,
// the seats and holes for the bearings and the shafts, and the holes for the grip screws and the ear screws.
func (b *builder) housing() *decad.Body {
	// The block is a rounded outline in (Y, Z) extruded along +X, with its two end faces rounded too.
	block := b.roundedPrism(frame(r3.Vec{X: boxLeftX}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
		[]point2{{fanWallY - wallThick, boxFloor}, {boxBackY, boxFloor}, {boxBackY, boxTop}, {fanWallY - wallThick, boxTop}},
		[]float64{3, 3, 3, 3}, crankWallX+wallThick-boxLeftX)
	block = b.roundCaps(block, 2)
	cavity := b.roundedPrism(frame(r3.Vec{X: boxLeftX - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
		[]point2{{fanWallY, cavityFloor}, {backWallY, cavityFloor}, {backWallY, boxTop + 1}, {fanWallY, boxTop + 1}},
		[]float64{3, 3, 0, 0}, crankWallX-boxLeftX+1)
	cavity = b.roundEnd(cavity, 2)
	// The post is the part of the block the cavity leaves: a notch in the cavity, on the left end, across its whole
	// width, so the post closes the end of the tray.
	notch := b.panel(frame(r3.Vec{X: boxLeftX - 2}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), fanWallY-1, cavityFloor-1, backWallY+1, boxTop+2,
		postFace-boxLeftX+2)
	cavity = b.cut(cavity, notch)
	body := b.cut(block, cavity)
	// Each bearing block starts inside the tray so the union has no shared face. Both are trapezoids seen from above: they
	// widen toward their wall at 45° or less, so the half that prints on them never overhangs.
	wallOuterX, wallOuterY := crankWallX+wallThick, fanWallY-wallThick
	tall := boxTop - boxFloor
	crankPad := b.polygonPrism(frame(r3.Vec{Z: boxFloor}, r3.Vec{X: 1}, r3.Vec{Y: 1}), []point2{
		{wallOuterX - 1, -(blockHalf + crankBlock + 1)}, {crankFace, -blockHalf}, {crankFace, blockHalf}, {wallOuterX - 1, blockHalf + crankBlock + 1},
	}, tall)
	fanPad := b.polygonPrism(frame(r3.Vec{Z: boxFloor}, r3.Vec{X: 1}, r3.Vec{Y: 1}), []point2{
		{boxLeftX + 1, wallOuterY + 1}, {wallOuterX - 1, wallOuterY + 1}, {blockHalf, -fanBlockFace}, {-blockHalf, -fanBlockFace},
	}, tall)
	if b.err != nil {
		return nil
	}
	parts := []*decad.Body{body, fanPad, crankPad}
	for _, e := range gearboxEars {
		parts = append(parts, b.panel(frame(r3.Vec{X: e.x0}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), -earHalf, e.z0, earHalf, e.z1, e.x1-e.x0))
	}
	body = b.union(parts...)
	// A 608 bearing sits where each shaft crosses a wall. The shafts go in through the post from the left. The drive
	// shaft ends in the crank-side wall's bearing, and the crank shaft goes through that wall and its block. The seats
	// open at the post's outer face and the block ends and in the walls' inner faces, so each bearing drops in along its
	// shaft. The crank and drive shafts' seats are cut across the split plane.
	for _, a := range []shaftAxis{crankSpline, driveSpline} {
		body = b.seat(body, a, boxLeftX-1, boxLeftX+seatDepth, boxLeftX+7, postFace+1)
		body = b.seat(body, a, crankWallX-1, crankWallX+seatDepth, crankWallX+seatDepth-0.1, crankWallX+seatDepth+reliefDepth-0.1)
	}
	body = b.hole(body, crankSpline, shaftHole, crankWallX+9.3, crankFace-8.5)
	body = b.seat(body, crankSpline, crankFace-seatDepth, crankFace+1, crankFace-seatDepth-reliefDepth+0.1, crankFace-seatDepth+0.1)
	// The fan shaft's two bearings are measured behind the apex. They are vertical holes in the front half: the wall's
	// seat opens upward and the block's opens downward on the bed, with a 45° cone for a ceiling.
	body = b.seat(body, fanSpline, -fanWallY-1, -fanWallY+seatDepth, -fanWallY+seatDepth-0.1, -fanWallY+seatDepth+reliefDepth-0.1)
	body = b.hole(body, fanSpline, shaftHole, -fanWallY+9.3, fanBlockFace-seatDepth-(reliefRadius-shaftHole)+1)
	body = b.funnelSeat(body, fanSpline, fanBlockFace-seatDepth, fanBlockFace+1)
	for _, dx := range []float64{-gripScrewX, gripScrewX} {
		body = b.cut(body, b.cylinder(frame(r3.Vec{X: gripX + dx, Y: gripY, Z: boxFloor - 1}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 1.2, 8))
	}
	// Each ear has a hole, a counterbore for the screw's head on the front side and a hex trap for its nut on the back side.
	// The Y axis is vertical when a half prints, so all three are vertical holes.
	for i := range gearboxEars {
		x, z := earAxis(i)
		body = b.cut(body, b.cylinder(frame(r3.Vec{X: x, Y: -earHalf - 1, Z: z}, r3.Vec{Z: 1}, r3.Vec{X: 1}), earHole, 2*earHalf+2))
		body = b.cut(body, b.cylinder(frame(r3.Vec{X: x, Y: -earHalf - 1, Z: z}, r3.Vec{Z: 1}, r3.Vec{X: 1}), counterboreRadius, 1+counterboreDepth))
		body = b.cut(body, b.hexagon(frame(r3.Vec{X: x, Y: earHalf + 1, Z: z}, r3.Vec{Z: -1}, r3.Vec{X: 1}), trapFlats, 1+trapDepth))
	}
	return body
}
