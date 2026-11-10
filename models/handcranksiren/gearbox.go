package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
)

// Gearbox dimensions in mm. The gearbox is a sandwich: a front plate, a frame and a rear plate, each printed flat, so every
// bearing seat is a vertical hole. The seams between the pieces are tightFit wide.
const (
	boxLeft, boxRight = -21.0, 107.0
	boxBottom, boxTop = 0.0, 131.0
	boxCorner         = 13.0
	cavityLeft        = -14.0
	cavityRight       = 100.0
	cavityBottom      = 7.0
	cavityTop         = 124.0
	cavityCorner      = 6.0
	plateThick        = 10.0
	frontY0, frontY1  = 0.0, 10.0
	frameY0, frameLen = 10.25, 17.1
	rearY0            = 27.6
	seatFloorFront    = 2.6  // the front plate's seat floors
	reliefFloorFront  = 1.6  // the front plate's relief floors
	seatFloorRear     = 35.0 // the rear plate's seat floors
	reliefFloorRear   = 36.0
	reliefOverlap     = 0.5 // a relief or hole tool reaches this far into the cut beside it
	rearThroughEnd    = 38.6
	toolStock         = 1.0
)

// outline returns the rounded-rectangle corners in the sketch frame of the axis R: u = x, v = 84 − z.
func outline(left, right, bottom, top float64) []point2 {
	// Counter-clockwise in (u, v), which is the winding the sketch fillet needs to stay fully constrained.
	return []point2{{left, rotorHeight - top}, {right, rotorHeight - top}, {right, rotorHeight - bottom}, {left, rotorHeight - bottom}}
}

// plate extrudes the outline from Y y0 for length. The sketch frame is the rotor axis, so no edge passes through the
// sketch origin.
func (b *builder) plate(y0, length float64) *decad.Body {
	rotor, _, _ := b.axes()
	r := boxCorner
	return b.roundedPrism(rotor.across(y0), outline(boxLeft, boxRight, boxBottom, boxTop), []float64{r, r, r, r}, length)
}

// seat cuts a bearing seat into host: the round pocket between pocket0 and pocket1 and a relief bore between relief0 and
// relief1 that frees the inner ring where the pocket's floor is.
func (b *builder) seat(host *decad.Body, a shaftAxis, pocket0, pocket1, relief0, relief1 float64) *decad.Body {
	host = b.cut(host, b.cylinder(a.across(pocket0), seatRadius, pocket1-pocket0))
	return b.cut(host, b.cylinder(a.across(relief0), reliefRadius, relief1-relief0))
}

// gearbox builds the three pieces.
func (b *builder) gearbox() {
	b.part("gearbox-front", HousingContainerName, "fixed", b.gearboxFront)
	b.part("gearbox-frame", HousingContainerName, "fixed", b.gearboxFrame)
	b.part("gearbox-rear", HousingContainerName, "fixed", b.gearboxRear)
}

// finish chamfers the large faces of a plate or frame at the Y offsets.
func (b *builder) finish(body *decad.Body, offsets ...float64) *decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	out, _, err := edgefinish.Chamfer(b.ctx, body, r3.Vec{Y: 1}, offsets, edgefinish.Long(edgefinish.All), edgefinish.MinChamfer)
	b.fail("finish edges", err)
	return out
}

// gearboxFront builds the front plate: three seats opening toward the frame, the rotor shaft's hole through the front,
// its edges finished, and the pocket and hole of each front bolt.
func (b *builder) gearboxFront() *decad.Body {
	rotor, inter, crank := b.axes()
	body := b.plate(frontY0, plateThick)
	for _, a := range []shaftAxis{rotor, inter, crank} {
		body = b.seat(body, a, seatFloorFront, frontY1+toolStock, reliefFloorFront, seatFloorFront+reliefOverlap)
	}
	body = b.cut(body, b.cylinder(rotor.across(frontY0-toolStock), shaftHoleRadius,
		reliefFloorFront+reliefOverlap-(frontY0-toolStock)))
	body = b.finish(body, frontY0, frontY1)
	for _, s := range sitesNamed("gearbox-front-bolt") {
		body = b.fastenerCuts(body, s, true, true, false)
	}
	// The drum bolts end in nut cavities near the front plate's inner face.
	for _, s := range sitesNamed("drum-bolt") {
		body = b.fastenerCuts(body, s, false, true, true)
	}
	return body
}

// gearboxRear builds the rear plate: three seats opening toward the frame, the crank shaft's hole through the rear, its
// edges finished, and the pocket and hole of each rear bolt.
func (b *builder) gearboxRear() *decad.Body {
	rotor, inter, crank := b.axes()
	rearY1 := rearY0 + plateThick
	body := b.plate(rearY0, plateThick)
	for _, a := range []shaftAxis{rotor, inter, crank} {
		body = b.seat(body, a, rearY0-toolStock, seatFloorRear, seatFloorRear-reliefOverlap, reliefFloorRear)
	}
	body = b.cut(body, b.cylinder(crank.across(reliefFloorRear-reliefOverlap), shaftHoleRadius,
		rearThroughEnd-(reliefFloorRear-reliefOverlap)))
	body = b.finish(body, rearY0, rearY1)
	for _, s := range sitesNamed("gearbox-rear-bolt") {
		body = b.fastenerCuts(body, s, true, true, false)
	}
	return body
}

// gearboxFrame builds the frame between the plates: the outline with the cavity as a hole, edges finished, then the hole
// and the nut cavity of every gearbox bolt.
func (b *builder) gearboxFrame() *decad.Body {
	rotor, _, _ := b.axes()
	s := b.section(rotor.across(frameY0))
	b.fail("draw frame outline", roundedPolygon(s, outline(boxLeft, boxRight, boxBottom, boxTop),
		[]float64{boxCorner, boxCorner, boxCorner, boxCorner}))
	b.fail("draw frame cavity", roundedPolygon(s, outline(cavityLeft, cavityRight, cavityBottom, cavityTop),
		[]float64{cavityCorner, cavityCorner, cavityCorner, cavityCorner}))
	body := b.extrude(s, 1, frameLen)
	body = b.finish(body, frameY0, frameY0+frameLen)
	for _, site := range sitesNamed("gearbox-") {
		if site.nutHost == "gearbox-frame" {
			body = b.fastenerCuts(body, site, false, true, true)
		}
	}
	return body
}

// head builds the siren head and its hardware: the drum, the rotor, the stator, the collector and the horn, then the
// bolts and nuts that hold them.
func (b *builder) head() {
	b.part("drum", HousingContainerName, "fixed", b.drum)
	b.part("rotor", RotorContainerName, "rotor", b.rotor)
	b.part("stator", StatorContainerName, "fixed", b.stator)
	b.part("collector", HousingContainerName, "fixed", b.collector)
	b.part("horn", HornContainerName, "fixed", b.horn)
	for _, prefix := range []string{"drum-bolt", "collector-bolt", "throat-bolt"} {
		for _, s := range sitesNamed(prefix) {
			b.hardware(s)
		}
	}
}
