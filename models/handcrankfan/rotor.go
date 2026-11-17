package handcrankfan

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
)

// Rotor dimensions in mm. The hub's back face is at Y = -hubBack and its front face at Y = -(hubBack + hubLength).
const (
	hubRadius = turbinefan.HubRadius
	hubLength = turbinefan.HubLength
	hubBack   = -(fanWallY - wallThick) + fanGap
	hubMid    = hubBack + hubLength/2

	fanTip = 26.0 // the fan shaft's rear tip, inside the pinion and clear of the drive gear's teeth

	// The shaft ends 1 mm inside the hub's front face, past the D bore's 0.5 front chamfer, and the D flat starts 1.5
	// behind the collar's end, which the collar refills.
	fanShaftEnd = hubBack + hubLength - 1
	flatStart   = hubBack - 2
	// collarEnd is the collar's end face: a press-fit gap from the hub's back face, as Arbor collars asks of a collar
	// that stops a pressed part.
	collarEnd = hubBack - pressFit

	// setScrewBackOff is how far the set screw is drawn short of tight, along its own line: its tip is level with the
	// bore's flat, 0.05 off the shaft's flat at its centred position (turbinefan places the tip against a shaft pushed
	// that far sideways, which the bearing-centred shaft is not).
	setScrewBackOff = 2 * pressFit
)

// The blade bolt is an M2 x 16 socket head cap screw with an M2 nut. It runs parallel to the fan axis at boltAt
// from the axis: the head sits in a pocket in the front face and the nut in a slot open to the round face.
const boltAt = turbinefan.BladeBoltAt

// fanLocalFrame maps turbinefan's local frame onto the fan at crank angle 0: its origin to the fan axis in the hub's
// back face, its +Z (forward, toward the front face) to -Y, its +Y (blade 1) to +X and its +X to -Z.
func fanLocalFrame() r3.Frame {
	return frame(r3.Vec{Y: -hubBack, Z: axisHeight}, r3.Vec{Z: -1}, r3.Vec{X: 1})
}

// fanPlacement is the Placement that puts turbinefan's hub where fanLocalFrame does. turbinefan's placement point is the
// hub's front face, so the origin lies hubLength ahead of the back face, and the axes are the same.
func fanPlacement() r3.Frame {
	return frame(r3.Vec{Y: -hubBack - hubLength, Z: axisHeight}, r3.Vec{Z: -1}, r3.Vec{X: 1})
}

// fanParams is turbinefan's 608ZZ fan, right-handed, at p's pitch, placed on the fan axis. The fan turns clockwise as
// the operator sees it, which is clockwise seen from turbinefan's front.
func fanParams(p Params) turbinefan.Params {
	tp := turbinefan.Params608ZZ()
	tp.BladePitch, tp.Hand = p.BladePitch, turbinefan.RightHand
	f := fanPlacement()
	tp.Placement = &f
	return tp
}

// fanShaft builds the fan shaft: a round shaft with the pinion's spline grooves, a D flat where the hub grips it, and a
// collar between the front bearing and the hub. t is the distance behind the apex.
func (b *builder) fanShaft() *decad.Body {
	shaft := b.cylinder(alongMinusY(-fanTip), shaftRadius, fanShaftEnd-fanTip)
	shaft = b.grooves(shaft, fanSpline, fanTip-1, -fanWallY+0.3)
	// The flat is cut on the bare shaft, before the collar refills its first 1.95 mm.
	flat := fanParams(b.p).FlatDepth
	shaft = b.cut(shaft, b.panel(alongMinusY(-flatStart), -(shaftRadius+1.5), -(shaftRadius+1.5), -(shaftRadius-flat),
		shaftRadius+1.5, hubBack+hubLength+2-flatStart))
	return b.union(shaft, b.cylinder(alongMinusY(-(fanBlockFace+0.1)), 6, collarEnd-(fanBlockFace+0.1)))
}

// fan builds the fan shaft, then the hub and the three blades from turbinefan, then the blade bolts and the set screw
// with their nuts at turbinefan's sites.
func (b *builder) fan() {
	b.part("fan-shaft", "shaft", "fan", b.fanShaft)
	var rotor *turbinefan.Parts
	built := func() *turbinefan.Parts {
		if rotor != nil || b.err != nil {
			return rotor
		}
		p := fanParams(b.p)
		if p.ShaftDiameter != 2*shaftRadius {
			b.fail("build rotor", fmt.Errorf("turbinefan's 608ZZ shaft does not match the fan shaft"))
			return nil
		}
		var err error
		rotor, err = turbinefan.BuildParts(b.ctx, b.doc, p)
		b.fail("build rotor", err)
		if rotor == nil {
			return nil
		}
		b.a.Sketches = append(b.a.Sketches, rotor.Sketches...)
		b.fail("build rotor", sameFanSites(rotor))
		return rotor
	}
	b.part("fan-hub", "rotor", "fan", func() *decad.Body {
		if r := built(); r != nil {
			return r.Hub
		}
		return nil
	})
	for k := range turbinefan.BladeCount {
		b.part(fmt.Sprintf("fan-blade-%d", k+1), "blade", "fan", func() *decad.Body {
			if r := built(); r != nil {
				return r.Blades[k]
			}
			return nil
		})
	}
	for _, s := range sitesNamed("fan-") {
		b.hardware(s)
	}
}

// sameFanSites reports an error unless fanSites names the sites the built hub was cut for, so every bolt and nut lies in
// the pocket, hole and slot turbinefan cut for it. The set screw's site differs by its back-off alone.
func sameFanSites(rotor *turbinefan.Parts) error {
	want := append(rotor.BladeSites[:], rotor.SetScrewSite)
	got := sitesNamed("fan-")
	if len(got) != len(want) {
		return fmt.Errorf("%d fan fastener sites, turbinefan has %d", len(got), len(want))
	}
	for i, s := range got {
		w := want[i]
		if i == len(want)-1 {
			w.Head, w.Cavity = w.Head.Sub(w.Dir.Scale(setScrewBackOff)), w.Cavity+setScrewBackOff
		}
		if s.Head.Sub(w.Head).Len() > 1e-9 || s.Dir.Sub(w.Dir).Len() > 1e-9 || s.Corner.Sub(w.Corner).Len() > 1e-9 ||
			s.Length != w.Length || s.Cavity != w.Cavity {
			return fmt.Errorf("site %s differs from turbinefan's", s.bolt)
		}
	}
	return nil
}

// fanSites returns the blade bolts and the set screw at crank angle 0, carried from turbinefan's local frame by
// fanLocalFrame. The pitch moves no site. It returns no site when turbinefan rejects the preset; fan then fails the build
// with turbinefan's own error.
func fanSites() []fastenerSite {
	dm, err := turbinefan.Params608ZZ().Derive()
	if err != nil {
		return nil
	}
	xf, err := r3.FromFrame(fanLocalFrame())
	if err != nil {
		return nil
	}
	place := func(bolt, nut string, s m2fastener.Site) fastenerSite {
		s.Head, s.Dir, s.Corner = xf.Apply(s.Head), xf.ApplyDir(s.Dir), xf.ApplyDir(s.Corner)
		return fastenerSite{Site: s, bolt: bolt, nut: nut, headHost: "fan-hub", nutHost: "fan-hub", motion: "fan"}
	}
	var sites []fastenerSite
	for k, s := range dm.BladeSites {
		sites = append(sites, place(fmt.Sprintf("fan-bolt-%d", k+1), fmt.Sprintf("fan-nut-%d", k+1), s))
	}
	// The set screw is drawn backed off along its own line, with its nut unmoved.
	set := dm.SetScrewSite
	set.Head, set.Cavity = set.Head.Sub(set.Dir.Scale(setScrewBackOff)), set.Cavity+setScrewBackOff
	return append(sites, place("fan-set-screw", "fan-set-screw-nut", set))
}
