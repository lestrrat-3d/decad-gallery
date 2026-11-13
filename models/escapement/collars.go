package escapement

import "github.com/lestrrat-3d/decad"

// Collar ring heights in mm. Each ring is a separate printed part 3.0 long pressed on its arbor's flat. Its rear face
// is 0.05 from the floor of the counterbore it enters, and the arbor's body step or shoulder stops it on the other side.
const (
	collarLength         = 3.0
	escapeRearCollarLow  = 0.25
	palletFrontCollarLow = 24.75
)

// buildCollarRing builds a collar ring about (x, y): a disc of the collar radius from z, with the body-family D bore
// through it.
func (b *builder) buildCollarRing(what string, x, y, z float64) (*decad.Body, error) {
	ring, err := b.discAt(what, x, y, z, CollarRadius, collarLength)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool(what+" bore", x, y, z-pressToolLow, bodyPressRadius, bodyPressFlat, collarLength+2*pressToolLow)
	if err != nil {
		return nil, err
	}
	return b.cut(ring, bore, what+" bore")
}

// buildEscapeRearCollar builds the ring that holds the escape wheel's pinion 0.25 mm clear of the back bearing: it is
// pressed on the escape arbor from the rear into the pinion's rear counterbore.
func (b *builder) buildEscapeRearCollar() (*decad.Body, error) {
	return b.buildCollarRing("escape rear collar", 0, 0, escapeRearCollarLow)
}

// buildPalletFrontCollar builds the ring that holds the anchor 0.25 mm clear of the front pallet bearing: it is
// pressed on the pallet arbor from the front into the anchor's front counterbore.
func (b *builder) buildPalletFrontCollar() (*decad.Body, error) {
	return b.buildCollarRing("pallet front collar", b.g.P.X, b.g.P.Y, palletFrontCollarLow)
}
