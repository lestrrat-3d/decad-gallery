package handcrankfan

import (
	"fmt"
	"math"

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
)

// The blade bolt is an M2 x 16 socket head cap screw with an M2 nut. It runs parallel to the fan axis at boltAt
// from the axis: the head sits in a pocket in the front face and the nut in a cavity the hub encloses.
const boltAt = turbinefan.BladeBoltAt

// fan builds the fan shaft, the hub, the blades and the blade bolts.
func (b *builder) fan() {
	b.part("fan-shaft", "shaft", "fan", func() *decad.Body {
		shaft := b.cylinder(alongMinusY(-fanTip), shaftRadius, hubBack+hubLength+4-fanTip)
		// One set of grooves passes through the pinion and one through the hub, with the round shaft between them for the
		// bearings. The collar sits between the front bearing and the hub.
		shaft = b.grooves(shaft, fanSpline, fanTip-1, -fanWallY+0.3)
		shaft = b.grooves(shaft, fanSpline, hubBack-1, hubBack+hubLength+5)
		return b.union(shaft, b.cylinder(alongMinusY(-(fanBlockFace+0.1)), 6, hubBack-tightFit-(fanBlockFace+0.1)))
	})
	b.part("fan-hub", "rotor", "fan", b.hub)
	var blades []*decad.Body
	b.part("fan-blade-1", "blade", "fan", func() *decad.Body {
		blades = b.blades()
		if len(blades) == 0 {
			return nil
		}
		return blades[0]
	})
	for k := 1; k < b.p.Blades; k++ {
		b.part(fmt.Sprintf("fan-blade-%d", k+1), "blade", "fan", func() *decad.Body { return blades[k] })
	}
	// Each nut follows its bolt in the part list. The blade-1 bodies are built once and copied about the fan axis.
	sites, err := bladeSites(b.p.Blades)
	b.fail("place blade fasteners", err)
	var bolts, nuts []*decad.Body
	for k := 0; k < b.p.Blades && b.err == nil; k++ {
		b.part(sites[k].bolt, "hardware", "fan", func() *decad.Body {
			if k == 0 {
				bolts, nuts = b.ring(b.placeBolt(sites[0])), b.ring(b.placeNut(sites[0]))
			}
			if len(bolts) == 0 {
				return nil
			}
			return bolts[k]
		})
		b.part(sites[k].nut, "hardware", "fan", func() *decad.Body {
			if len(nuts) == 0 {
				return nil
			}
			return nuts[k]
		})
	}
}

// ring returns body and its copies about the fan axis, one for each blade.
func (b *builder) ring(body *decad.Body) []*decad.Body {
	if b.err != nil || body == nil {
		return nil
	}
	out := []*decad.Body{body}
	if b.p.Blades > 1 {
		// Transform retires its input, so the other copies are made from the finished one.
		copies, err := decad.PatternCopies(b.ctx, body, decad.CircularPattern{
			Center: b.a.State.Apex, Axis: fanAxis, Count: b.p.Blades})
		if err != nil {
			b.fail("copy about the fan axis", err)
			return nil
		}
		out = append(out, copies...)
	}
	return out
}

// hub builds the disc with the shaft bore and one bolt hole for each blade, then cuts a socket, a head pocket and a
// nut cavity for each blade. The tools run past the faces they open, so none shares a face with the hub.
func (b *builder) hub() *decad.Body {
	n := b.p.Blades
	s := b.section(frame(r3.Vec{Y: -hubBack, Z: axisHeight}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
	circle(s, 0, 0, hubRadius)
	for k := 0; k < n; k++ {
		theta := 2 * math.Pi * float64(k) / float64(n)
		circle(s, boltAt*math.Cos(theta), boltAt*math.Sin(theta), m2fastener.HoleRadius)
	}
	hub := b.roundCaps(b.extrude(s, n, hubLength), 1)
	hub = b.splineCut(hub, fanSpline, hubBack-1, hubLength+2, pressFit)
	// The socket is a plain round hole. It is 6.3 across, small enough to print on its side without support.
	// A round at the floor takes the inside corner off the print.
	var socket *decad.Body
	if b.err == nil {
		tool, sk, err := turbinefan.BladeSocketTool(b.ctx, b.doc, frame(r3.Vec{X: turbinefan.BladeSocketStart, Y: -hubMid, Z: axisHeight},
			r3.Vec{Y: 1}, r3.Vec{Z: 1}))
		b.fail("build blade socket", err)
		if err == nil {
			b.a.Sketches = append(b.a.Sketches, sk)
			socket = tool
		}
	}
	for _, copy := range b.ring(socket) {
		hub = b.cut(hub, copy)
	}
	// The head pocket and the nut cavity of each blade's bolt; the bolt holes are in the sketch.
	sites, err := bladeSites(n)
	b.fail("place blade fasteners", err)
	for _, site := range sites {
		hub = b.fastenerCuts(hub, site, true, false, true)
	}
	return hub
}

// blades builds blade 1 (bladeBody), places it along +X in the hub's mid plane and copies it about the fan axis.
func (b *builder) blades() []*decad.Body {
	if b.err != nil {
		return nil
	}
	built, err := turbinefan.BuildBlade(b.ctx, b.doc, b.p.BladePitch)
	if err != nil {
		b.fail("build blade", err)
		return nil
	}
	b.a.Sketches = append(b.a.Sketches, built.Sketches...)
	blade := built.Body
	// Local Z becomes +X, local X the direction of motion (+Z) and local Y downstream (-Y).
	xf, err := r3.FromFrame(frame(r3.Vec{Y: -hubMid, Z: axisHeight}, r3.Vec{Z: 1}, r3.Vec{Y: -1}))
	if err != nil {
		b.fail("create blade placement", err)
		return nil
	}
	if blade, err = decad.Transform(b.ctx, blade, xf); err != nil {
		b.fail("place blade", err)
		return nil
	}
	return b.ring(blade)
}
