package handcranksiren

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/r3"
)

// The crank is models/crankhandle's 608-2RS crank: an arm pressed onto the crank shaft's D-flat, a pin whose hexagonal
// stub one M2 x 10 bolt and nut clamp in the arm, and a handle that turns on the pin. It is built at CrankAngle 0, with
// the arm pointing up (+Z) from C.
const armFace = 38.6 // the arm's inner face

// pinAxis is the pin's axis at CrankAngle 0, as a point at Y 0.
var pinAxis = r3.Vec{X: crankOffset, Z: rotorHeight + crankhandle.Params6082RS().Throw}

// crankPlacement maps crankhandle's local frame onto the siren: its origin to C at the arm's inner face, its +Z (out
// of the mechanism) to +Y and its +Y (toward the pin) to +Z.
func crankPlacement() r3.Frame {
	return frame(r3.Vec{X: crankOffset, Y: armFace, Z: rotorHeight}, r3.Vec{X: -1}, r3.Vec{Z: 1})
}

// crank builds the arm, the pin, the handle and the pin's bolt and nut.
func (b *builder) crank() {
	var parts *crankhandle.Parts
	built := func() *crankhandle.Parts {
		if parts != nil || b.err != nil {
			return parts
		}
		p := crankhandle.Params6082RS()
		if p.ShaftDiameter != 2*shaftRadius || p.ShaftDiameter/2-p.FlatDepth != flatOffset {
			b.fail("build crank", fmt.Errorf("crankhandle's 608-2RS shaft does not match the crank shaft"))
			return nil
		}
		at := crankPlacement()
		p.Placement = &at
		var err error
		parts, err = crankhandle.BuildParts(b.ctx, b.doc, p)
		b.fail("build crank", err)
		if parts != nil {
			b.a.Sketches = append(b.a.Sketches, parts.Sketches...)
		}
		return parts
	}
	for _, part := range []struct {
		name string
		body func(*crankhandle.Parts) *decad.Body
	}{
		{"crank-arm", func(p *crankhandle.Parts) *decad.Body { return p.Arm }},
		{"crank-pin", func(p *crankhandle.Parts) *decad.Body { return p.Pin }},
		{"crank-handle", func(p *crankhandle.Parts) *decad.Body { return p.Handle }},
	} {
		b.part(part.name, CrankContainerName, "crank", func() *decad.Body {
			if p := built(); p != nil {
				return part.body(p)
			}
			return nil
		})
	}
	b.hardware(sitesNamed("crank-pin-bolt")[0])
}
