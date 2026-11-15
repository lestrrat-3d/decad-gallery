package handcranksiren

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/r3"
)

// The crank is models/crankhandle's 608-2RS crank: an arm that an M2 x 16 clamp bolt pinches onto the crank shaft's
// D-flat, a pin whose hexagonal stub one M2 x 10 bolt and nut clamp in the arm and whose two printed halves two M2 x 6
// cross bolts clamp, and a handle that turns on the pin. It is built at CrankAngle 0, with the arm pointing up (+Z)
// from C.
const armFace = 38.6 // the arm's inner face

// pinAxis is the pin's axis at CrankAngle 0, as a point at Y 0.
var pinAxis = r3.Vec{X: crankOffset, Z: rotorHeight + crankhandle.Params6082RS().Throw}

// crankPlacement maps crankhandle's local frame onto the siren: its origin to C at the arm's inner face, its +Z (out
// of the mechanism) to +Y and its +Y (toward the pin) to +Z.
func crankPlacement() r3.Frame {
	return frame(r3.Vec{X: crankOffset, Y: armFace, Z: rotorHeight}, r3.Vec{X: -1}, r3.Vec{Z: 1})
}

// crankParams is the 608-2RS crank placed on the siren.
func crankParams() crankhandle.Params {
	p := crankhandle.Params6082RS()
	at := crankPlacement()
	p.Placement = &at
	return p
}

// crankSites returns the crank's bolts at CrankAngle 0, carried from crankhandle's local frame by crankPlacement: the pin
// bolt from the arm's back face into the pin's stub, the two cross bolts that clamp the pin's halves, and the clamp bolt
// across the arm's hub. It returns no site when crankhandle rejects the preset; crank then fails the build with
// crankhandle's own error.
func crankSites() []fastenerSite {
	dm, err := crankhandle.Params6082RS().Derive()
	if err != nil {
		return nil
	}
	xf, err := r3.FromFrame(crankPlacement())
	if err != nil {
		return nil
	}
	place := func(bolt, nut, headHost, nutHost string, s m2fastener.Site) fastenerSite {
		s.Head, s.Dir, s.Corner = xf.Apply(s.Head), xf.ApplyDir(s.Dir), xf.ApplyDir(s.Corner)
		return fastenerSite{Site: s, bolt: bolt, nut: nut, headHost: headHost, nutHost: nutHost, motion: "crank"}
	}
	sites := []fastenerSite{place("crank-pin-bolt", "crank-pin-nut", "crank-arm", "crank-pin", dm.Site)}
	for i, s := range dm.CrossSites {
		sites = append(sites, place(fmt.Sprintf("crank-cross-bolt-%d", i+1), fmt.Sprintf("crank-cross-nut-%d", i+1),
			"crank-pin", "crank-pin", s))
	}
	return append(sites, place("crank-clamp-bolt", "crank-clamp-nut", "crank-arm", "crank-arm", dm.ClampSite))
}

// crank builds the arm, the pin, the handle and the crank's bolts and nuts.
func (b *builder) crank() {
	var parts *crankhandle.Parts
	built := func() *crankhandle.Parts {
		if parts != nil || b.err != nil {
			return parts
		}
		p := crankParams()
		if p.ShaftDiameter != 2*shaftRadius || p.ShaftDiameter/2-p.FlatDepth != flatOffset {
			b.fail("build crank", fmt.Errorf("crankhandle's 608-2RS shaft does not match the crank shaft"))
			return nil
		}
		var err error
		parts, err = crankhandle.BuildParts(b.ctx, b.doc, p)
		b.fail("build crank", err)
		if parts == nil {
			return nil
		}
		b.a.Sketches = append(b.a.Sketches, parts.Sketches...)
		b.fail("build crank", sameSites(parts))
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
	for _, s := range sitesNamed("crank-") {
		b.hardware(s)
	}
}

// sameSites reports an error unless crankSites names the sites the built parts were cut for, so every bolt and nut lies
// in the pocket, hole and cavity crankhandle cut for it.
func sameSites(parts *crankhandle.Parts) error {
	want := append([]m2fastener.Site{parts.Site}, parts.CrossSites[:]...)
	want = append(want, parts.ClampSite)
	got := sitesNamed("crank-")
	if len(got) != len(want) {
		return fmt.Errorf("%d crank fastener sites, crankhandle has %d", len(got), len(want))
	}
	for i, s := range got {
		w := want[i]
		if s.Head.Sub(w.Head).Len() > 1e-9 || s.Dir.Sub(w.Dir).Len() > 1e-9 || s.Corner.Sub(w.Corner).Len() > 1e-9 ||
			s.Length != w.Length || s.Cavity != w.Cavity {
			return fmt.Errorf("site %s differs from crankhandle's", s.bolt)
		}
	}
	return nil
}
