package handcranksiren

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
)

// Grip dimensions in mm. The grip is a vertical D-handle on the left (-Y) face of gearbox-front, toward the operator: one
// extruded ring in the YZ plane from X gripFrom for gripWidth, made of a pad on the plate, two arms and a vertical bar.
// The finger window runs through it along X. The pad's back face is 0.25 off the plate (the Tight seam).
const (
	gripFrom         = 79.0
	gripWidth        = 26.0
	gripPadFace      = -16.25 // the pad's window face, where the grip bolt heads end
	gripBack         = -0.25
	gripOuterY       = -76.25
	gripBarIn        = -46.25 // the bar's face toward the window
	gripBottom       = 5.0
	gripTop          = 125.0
	gripWindowBottom = 12.0
	gripWindowTop    = 118.0
	gripCorner       = 10.0
	gripWindowCorner = 3.0
	// gripChamfer is larger than the 0.5 minimum of "Surface edge finishing": it rounds the 26 x 30 bar for the hand
	// (instructions.md, Edge finishing).
	gripChamfer = 2.0
)

// grip builds the grip and then the bolts and nuts that hold it to gearbox-front.
func (b *builder) grip() {
	b.part("grip", GripContainerName, "fixed", b.gripBody)
	for _, s := range sitesNamed("grip-bolt") {
		b.hardware(s)
	}
}

// gripBody extrudes the ring profile along +X, chamfers both caps and cuts the pocket and hole of each grip bolt. The
// sketch frame is at X gripFrom with U = +Y and V = +Z, so no edge passes through its origin.
func (b *builder) gripBody() *decad.Body {
	s := b.section(frame(r3.Vec{X: gripFrom}, r3.Vec{Y: 1}, r3.Vec{Z: 1}))
	b.fail("draw grip outline", roundedPolygon(s, []point2{
		{gripOuterY, gripBottom}, {gripBack, gripBottom}, {gripBack, gripTop}, {gripOuterY, gripTop},
	}, corners(gripCorner)))
	b.fail("draw grip window", roundedPolygon(s, []point2{
		{gripBarIn, gripWindowBottom}, {gripPadFace, gripWindowBottom}, {gripPadFace, gripWindowTop}, {gripBarIn, gripWindowTop},
	}, corners(gripWindowCorner)))
	body := b.extrude(s, 1, gripWidth)
	body = b.chamferAlong(body, r3.Vec{X: 1}, edgefinish.Long(edgefinish.All), gripChamfer, gripFrom, gripFrom+gripWidth)
	for _, site := range sitesNamed("grip-bolt") {
		body = b.fastenerCuts(body, site, true, true, false)
	}
	return body
}
