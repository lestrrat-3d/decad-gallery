package handcranksiren

import "github.com/lestrrat-3d/decad"

// Stator dimensions in mm.
const (
	statorRadius = 58.0
	statorFront  = -31.65
	statorThick  = 3.0
	statorPort   = -32.65 // the stator's port tool starts here and runs 5, through the disc
)

// stator builds the stator: a chamfered disc with the same ports, and a hole for each horn bolt.
func (b *builder) stator() *decad.Body {
	disc := b.cylinder(faceOnR(statorFront), statorRadius, statorThick)
	disc = b.discChamfer(disc)
	disc = b.portsCut(disc, statorPort)
	for _, site := range sitesNamed("horn-bolt") {
		disc = b.fastenerCuts(disc, site, false, true, false)
	}
	return disc
}
