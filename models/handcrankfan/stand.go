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
// It prints as four pieces cut at the walls: the base (the floor and the back wall between the post and the crank-side
// wall), the post, the crank-side wall with its block, and the fan-side wall with its block, which holds the fan shaft.
// Every bearing seat is a vertical hole in the piece that holds it: the post prints on its inner face, and the two
// walls print on the face that meets the base. A seat that opens onto the bed has a 45° cone for a ceiling. Ears with M2
// screws and nuts straddle the seams and hold the pieces together.
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
	blockHalf    = 15.0                                // the bearing blocks are 30 wide and as tall as the tray
	fanBlock     = 24.0                                // the fan block sticks out this far from the fan-side wall's outer face
	crankBlock   = 14.0                                // and the crank block this far from the crank-side wall's outer face
	fanGap       = fanBlock + 10.5                     // from the fan-side wall's outer face to the hub
	crankFace    = crankWallX + wallThick + crankBlock // the crank block's end face, X 45
	fanBlockFace = -fanWallY + wallThick + fanBlock    // the fan block's end face, 65.5 behind the apex
	gripY        = fanWallY + 8.2                      // the grip's centre sits this far behind the fan-side wall's inner face
	gripX        = 7.0                                 // the grip's centre, under the tray

	splitGap = 0.1 // each piece stops this far from its seam
	earHalf  = 8.0 // an ear reaches this far to each side of its seam
	earLen   = 8.0 // and sticks out this far from the tray
	earBolt  = 12.0
	earHole  = 1.3 // the ear screw's clearance hole, diameter 2.6

	wallScrewLength = 16.0 // an M2 x 16 self-tapping screw, as the grip's
	pilotRadius     = 0.9  // and its pilot, the grip screws' too
	pilotDepth      = 9.5
)

// gearboxEar is one ear: a box that straddles a seam and carries one M2 screw along dir, from its head to its nut.
// axis is the seam's coordinate along dir, lat0 and lat1 are the box's range across it (Y for an ear on a seam across
// X, X for one on a seam across Y), and the ear hangs under the floor or sits on a wall's top. head and nut name the
// pieces on the head's side and the nut's side.
type gearboxEar struct {
	dir        r3.Vec
	seam       float64
	lat0, lat1 float64
	top        bool
	head, nut  string
}

func (e gearboxEar) z0() float64 {
	if e.top {
		return boxTop - 1
	}
	return boxFloor - earLen
}

func (e gearboxEar) z1() float64 {
	if e.top {
		return boxTop + earLen
	}
	return boxFloor + 1
}

// centre is the point on the screw's line at the seam.
func (e gearboxEar) centre() r3.Vec {
	z := e.z0() + earLen/2
	if e.top {
		z = e.z1() - earLen/2
	}
	lat := (e.lat0 + e.lat1) / 2
	if e.dir.X != 0 {
		return r3.Vec{X: e.seam, Y: lat, Z: z}
	}
	return r3.Vec{X: lat, Y: e.seam, Z: z}
}

// gearboxEars are the four ears. They sit on the fan-side wall's seams with the post and the crank-side wall, at the
// corners, two under the floor and two on the wall tops, the lower two clear of the grip. The fan block leaves no room
// on the wall's outer face for a screw head, so these seams need ears.
var gearboxEars = [...]gearboxEar{
	{r3.Vec{Y: 1}, fanWallY, -19, -11, false, "gearbox-fan-wall", "gearbox-post"},
	{r3.Vec{Y: 1}, fanWallY, 24.8, 31, false, "gearbox-fan-wall", "gearbox-crank-wall"},
	{r3.Vec{Y: 1}, fanWallY, -19, -11, true, "gearbox-fan-wall", "gearbox-post"},
	{r3.Vec{Y: 1}, fanWallY, 22, 30, true, "gearbox-fan-wall", "gearbox-crank-wall"},
}

// wallScrew is one M2 x 16 self-tapping screw that joins the post or the crank-side wall to the base. Its head sits in
// a round recess in the wall's outer face, at face along dir; it passes through the wall and screws into the end of the
// base's back wall at seam. No ear is needed, because that end is 9.8 thick.
type wallScrew struct {
	dir        r3.Vec
	face, seam float64
	z          float64
	head, recv string
}

// wallScrewY is where the screws meet the back wall, in its middle.
const wallScrewY = (backWallY + boxBackY) / 2

var wallScrews = [...]wallScrew{
	{r3.Vec{X: 1}, boxLeftX, postFace, 112, "gearbox-post", "gearbox-base"},
	{r3.Vec{X: 1}, boxLeftX, postFace, 185, "gearbox-post", "gearbox-base"},
	{r3.Vec{X: -1}, crankWallX + wallThick, crankWallX, 112, "gearbox-crank-wall", "gearbox-base"},
	{r3.Vec{X: -1}, crankWallX + wallThick, crankWallX, 185, "gearbox-crank-wall", "gearbox-base"},
}

// at is the point on the screw's line where its coordinate along dir is x.
func (w wallScrew) at(x float64) r3.Vec { return r3.Vec{X: x, Y: wallScrewY, Z: w.z} }

// slab removes the box that covers the coordinate range [lo, hi] along X (axis 0) or Y (axis 1) and everything else
// in the tray's range.
func (b *builder) slab(host *decad.Body, axis int, lo, hi float64) *decad.Body {
	if axis == 0 {
		return b.cut(host, b.panel(frame(r3.Vec{X: lo}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), -100, 50, 100, 260, hi-lo))
	}
	return b.cut(host, b.panel(frame(r3.Vec{Y: lo}, r3.Vec{Z: 1}, r3.Vec{X: 1}), 50, -100, 260, 100, hi-lo))
}

// gearbox builds the housing as one body, cuts it at the walls into four pieces and adds the screws that hold them
// together.
func (b *builder) gearbox() {
	var base, post, crank *decad.Body
	b.part("gearbox-fan-wall", "housing", "fixed", func() *decad.Body {
		whole := b.housing()
		if b.err != nil {
			return nil
		}
		var copies [3]*decad.Body
		for i := range copies {
			c, err := decad.Copy(b.ctx, whole)
			b.fail("copy the housing", err)
			copies[i] = c
		}
		if b.err != nil {
			return nil
		}
		// The fan-side wall keeps Y <= its seam. The others keep Y >= the seam, and each keeps its range along X.
		base = b.slab(b.slab(b.slab(copies[0], 1, -100, fanWallY+splitGap), 0, -100, postFace+splitGap), 0, crankWallX-splitGap, 100)
		post = b.slab(b.slab(copies[1], 1, -100, fanWallY+splitGap), 0, postFace-splitGap, 100)
		crank = b.slab(b.slab(copies[2], 1, -100, fanWallY+splitGap), 0, -100, crankWallX+splitGap)
		return b.slab(whole, 1, fanWallY-splitGap, 100)
	})
	b.part("gearbox-post", "housing", "fixed", func() *decad.Body { return post })
	b.part("gearbox-crank-wall", "housing", "fixed", func() *decad.Body { return crank })
	b.part("gearbox-base", "housing", "fixed", func() *decad.Body { return base })
	for i, e := range gearboxEars {
		b.part(fmt.Sprintf("gearbox-bolt-%d", i+1), "hardware", "fixed", func() *decad.Body {
			c := e.centre().Add(e.dir.Scale(-earHalf + counterboreDepth - hardwareGap))
			return b.screw(c, e.dir, r3.Vec{Z: 1}, earBolt, 2*earHalf-trapDepth+0.2-counterboreDepth+hardwareGap, 1.0)
		})
	}
	for i, w := range wallScrews {
		b.part(fmt.Sprintf("gearbox-screw-%d", i+1), "hardware", "fixed", func() *decad.Body {
			return b.screw(w.at(w.face+w.dir.X*(counterboreDepth-hardwareGap)), w.dir, r3.Vec{Z: 1}, wallScrewLength, 0, 0.8)
		})
	}
}

// housing builds the whole tray before it is cut: a rounded block minus a cavity, the two bearing blocks, the ears, the
// seats and holes for the bearings and the shafts, and the holes for the grip screws and the ear screws.
func (b *builder) housing() *decad.Body {
	// The block is a rounded outline in (Y, Z) extruded along +X, with its two end faces rounded too.
	block := b.roundedPrism(frame(r3.Vec{X: boxLeftX}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
		[]point2{{fanWallY - wallThick, boxFloor}, {boxBackY, boxFloor}, {boxBackY, boxTop}, {fanWallY - wallThick, boxTop}},
		[]float64{3, 3, 3, 3}, crankWallX+wallThick-boxLeftX)
	block = b.roundCaps(block, 2)
	cavity := b.roundedPrism(frame(r3.Vec{X: boxLeftX - 1}, r3.Vec{Y: 1}, r3.Vec{Z: 1}),
		[]point2{{fanWallY, cavityFloor}, {backWallY, cavityFloor}, {backWallY, boxTop + 1}, {fanWallY, boxTop + 1}},
		[]float64{0, 3, 0, 0}, crankWallX-boxLeftX+1)
	// The cavity's corners that meet a seam stay sharp: a fillet there would leave a thin sliver in the piece beside the
	// seam. The post is the part of the block the cavity leaves: a notch in the cavity, on the left end, across its whole
	// width, so the post closes the end of the tray.
	notch := b.panel(frame(r3.Vec{X: boxLeftX - 2}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), fanWallY-1, cavityFloor-1, backWallY+1, boxTop+2,
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
	parts := []*decad.Body{body, fanPad, crankPad}
	for _, e := range gearboxEars {
		if e.dir.X != 0 {
			parts = append(parts, b.panel(frame(r3.Vec{X: e.seam - earHalf}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), e.lat0, e.z0(), e.lat1, e.z1(), 2*earHalf))
		} else {
			parts = append(parts, b.panel(frame(r3.Vec{Y: e.seam - earHalf}, r3.Vec{Z: 1}, r3.Vec{X: 1}), e.z0(), e.lat0, e.z1(), e.lat1, 2*earHalf))
		}
	}
	body = b.union(parts...)
	// A 608 bearing sits where each shaft crosses a wall. The shafts go in through the post from the left. The drive
	// shaft ends in the crank-side wall's bearing, and the crank shaft goes through that wall and its block. The post's
	// seats open at its outer face and print upward. The walls' seats open at the face that meets the base, which is the
	// bed, so each has a cone for a ceiling, and the blocks' seats open upward.
	for _, a := range []shaftAxis{crankSpline, driveSpline} {
		body = b.seat(body, a, boxLeftX-1, boxLeftX+seatDepth, boxLeftX+7, postFace+1)
		body = b.funnelSeat(body, a, crankWallX+seatDepth, crankWallX-1)
	}
	cone := reliefRadius - shaftHole
	body = b.hole(body, crankSpline, shaftHole, crankWallX+seatDepth+cone-0.1, crankFace-seatDepth-reliefDepth+0.2)
	body = b.seat(body, crankSpline, crankFace-seatDepth, crankFace+1, crankFace-seatDepth-reliefDepth+0.1, crankFace-seatDepth+0.1)
	// The fan shaft's two bearings are measured behind the apex.
	body = b.funnelSeat(body, fanSpline, -fanWallY+seatDepth, -fanWallY-1)
	body = b.hole(body, fanSpline, shaftHole, -fanWallY+seatDepth+cone-0.1, fanBlockFace-seatDepth-reliefDepth+0.2)
	body = b.seat(body, fanSpline, fanBlockFace-seatDepth, fanBlockFace+1, fanBlockFace-seatDepth-reliefDepth+0.1, fanBlockFace-seatDepth+0.1)
	for _, dx := range []float64{-gripScrewX, gripScrewX} {
		body = b.cut(body, b.cylinder(frame(r3.Vec{X: gripX + dx, Y: gripY, Z: boxFloor - 1}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 1.2, 8))
	}
	// Each wall screw has a recess for its head in the wall's outer face, a clearance hole through the wall and a pilot in the
	// end of the base's back wall.
	for _, w := range wallScrews {
		d, up := w.dir, r3.Vec{Z: 1}
		out := frame(w.at(w.face-d.X), up, d.Cross(up))
		body = b.cut(body, b.cylinder(out, counterboreRadius, 1+counterboreDepth))
		body = b.cut(body, b.cylinder(out, earHole, 1+wallThick))
		body = b.cut(body, b.cylinder(frame(w.at(w.seam-d.X*0.5), up, d.Cross(up)), pilotRadius, 0.5+pilotDepth))
	}
	// Each ear has a hole, a counterbore for the screw's head on one side and a hex trap for its nut on the other. The
	// head and the nut sit on either side of the seam, so the screw joins the two pieces. The corners of the trap lie on Z.
	for _, e := range gearboxEars {
		c, d, up := e.centre(), e.dir, r3.Vec{Z: 1}
		through := frame(c.Add(d.Scale(-earHalf-1)), up, d.Cross(up))
		body = b.cut(body, b.cylinder(through, earHole, 2*earHalf+2))
		body = b.cut(body, b.cylinder(through, counterboreRadius, 1+counterboreDepth))
		body = b.cut(body, b.hexagon(frame(c.Add(d.Scale(earHalf+1)), up, d.Scale(-1).Cross(up)), trapFlats, 1+trapDepth))
	}
	return body
}
