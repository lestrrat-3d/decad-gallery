package escapement

import "github.com/lestrrat-3d/decad"

// Collar ring heights in mm. The only separate collar rings are the crutch and knob rings, each 3.0 long and pressed on
// an arbor's end; the other collars are steps of the arbors themselves.
const (
	collarLength = 3.0

	// The crutch and knob collars stand outside the front plate (outer face z 34), start at z 34.25 where the arbor's
	// outside flat starts, and are keyed D rings: outer arc r 3.7 and chord x 3.3, in a D counterbore of the part (arc
	// r 3.75, chord x 3.35, a 0.05 press), so the ring carries the drive as well as the part's own socket.
	outsideCollarLow   = 34.25
	keyedRingFlat      = 3.3
	keyedCounterRadius = 3.75
	keyedCounterFlat   = 3.35
	keyedCounterLow    = 34.30 // the counterbore tool, from 0.05 above the collar's rear face
	keyedCounterLength = 3.0   // to z 37.30, 0.05 above the collar's front face
)

// keyedCounterbore builds the cutter for a crutch or knob rear counterbore about (x, y): the D outline of the collar
// ring, 0.05 larger, on +X like the part's bore.
func (b *builder) keyedCounterbore(what string, x, y float64) (*decad.Body, error) {
	return b.dTool(what, x, y, keyedCounterLow, keyedCounterRadius, keyedCounterFlat, keyedCounterLength)
}

// buildKeyedRing builds an outside collar ring about (x, y): a D outline (arc r 3.7, chord x 3.3) 3.0 long from
// z 34.25, with the end-family D bore through it.
func (b *builder) buildKeyedRing(what string, x, y float64) (*decad.Body, error) {
	ring, err := b.dTool(what, x, y, outsideCollarLow, CollarRadius, keyedRingFlat, collarLength)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool(what+" bore", x, y, outsideCollarLow-pressToolLow, endPressRadius, endPressFlat,
		collarLength+2*pressToolLow)
	if err != nil {
		return nil, err
	}
	return b.cut(ring, bore, what+" bore")
}

// buildCrutchCollar builds the ring that holds the crutch 1.30 mm clear of the front plate on the pallet arbor's end.
func (b *builder) buildCrutchCollar() (*decad.Body, error) {
	return b.buildKeyedRing("crutch collar", b.g.P.X, b.g.P.Y)
}

// buildKnobCollar builds the ring that holds the winding knob clear of the front plate, about the origin like the knob.
func (b *builder) buildKnobCollar() (*decad.Body, error) {
	return b.buildKeyedRing("knob collar", 0, 0)
}
