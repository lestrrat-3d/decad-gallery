package swisslever

import "github.com/lestrrat-3d/decad"

// Winding knob and Knob Collar heights and dimensions in mm.
const (
	knobRadius       = 11.0
	knobLow          = 34.30
	knobHigh         = 37.30
	keyedCounterLow  = 33.30
	keyedCounterLong = 3.0
	keyedCounterR    = 3.75
	keyedCounterFlat = 3.35
	knobSocketLong   = 5.0
	collarLow        = 33.25
	collarLong       = 3.0
	collarRadius     = 3.7
	collarFlat       = 3.3
	collarBoreLow    = 32.25
	collarBoreLong   = 5.0
)

// buildKnob builds the winding knob about the origin: a disc with the keyed rear counterbore that takes the Knob Collar
// and the end-family D socket through its floor.
func (b *builder) buildKnob() (*decad.Body, error) {
	body, err := b.discAt("winding knob", 0, 0, knobLow, knobRadius, knobHigh-knobLow)
	if err != nil {
		return nil, err
	}
	counter, err := b.dTool("knob counterbore", 0, 0, keyedCounterLow, keyedCounterR, keyedCounterFlat, keyedCounterLong)
	if err != nil {
		return nil, err
	}
	if body, err = b.cut(body, counter, "knob counterbore"); err != nil {
		return nil, err
	}
	socket, err := b.dTool("knob socket", 0, 0, keyedCounterLow, EndBoreRadius, EndBoreFlat, knobSocketLong)
	if err != nil {
		return nil, err
	}
	return b.cut(body, socket, "knob socket")
}

// buildKnobCollar builds the keyed D ring that holds the knob clear of the front plate, about the origin, with the
// end-family D bore through it.
func (b *builder) buildKnobCollar() (*decad.Body, error) {
	ring, err := b.dTool("knob collar", 0, 0, collarLow, collarRadius, collarFlat, collarLong)
	if err != nil {
		return nil, err
	}
	bore, err := b.dTool("knob collar bore", 0, 0, collarBoreLow, EndBoreRadius, EndBoreFlat, collarBoreLong)
	if err != nil {
		return nil, err
	}
	return b.cut(ring, bore, "knob collar bore")
}
