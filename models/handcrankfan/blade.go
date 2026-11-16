package handcrankfan

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Rotor dimensions other packages build a hub and a blade joint from, in mm. They are the hand-crank fan's own values.
const (
	// HubRadius and HubLength are the rotor hub's radius and length along the fan axis.
	HubRadius = hubRadius
	HubLength = hubLength
	// BladeBoltAt is the blade bolt's distance from the fan axis. BladeBoltLength is its M2 length and BladeBoltCavity
	// the distance from its head's top face to the near face of the nut cavity.
	BladeBoltAt     = boltAt
	BladeBoltLength = 16.0
	BladeBoltCavity = 15.6
	// BladeStudRadius is the blade's root stud, which runs from radius BladeStudStart to BladeStudEnd.
	BladeStudRadius = studRadius
	BladeStudStart  = studStart
	BladeStudEnd    = studEnd
	// BladeSocketRadius is the hub socket's radius, a Regular fit round the stud. The socket's floor is at radius
	// BladeSocketStart, its outline rounded by BladeSocketFloorRound.
	BladeSocketRadius     = socketRadius
	BladeSocketStart      = socketStart
	BladeSocketFloorRound = socketFloorRound
	// DefaultBladePitch is the blade angle at 48 mm the sections are drawn at. A blade's pitch lies in
	// [MinBladePitch, MaxBladePitch].
	DefaultBladePitch = refPitch
	MinBladePitch     = 10 * math.Pi / 180
	MaxBladePitch     = 40 * math.Pi / 180
)

// BladeRootRadius is the radius of the blade's root section, where the loft starts.
func BladeRootRadius() float64 { return stations[0].r }

// BladeTipRadius is the radius of the blade's tip.
func BladeTipRadius() float64 { return stations[len(stations)-1].r }

// Beta is the blade angle at radius r, measured from the plane of rotation, for a blade angle of pitch at 48 mm.
func Beta(r, pitch float64) float64 { return beta(r, pitch) }

// Blade is one blade in its local frame: the origin on the fan axis in the hub's mid plane, +Z the blade's radial line,
// +Y downstream along the fan axis, and the blade moving toward -X. Sketches are every sketch it was built from, each
// fully constrained.
type Blade struct {
	Body     *decad.Body
	Sketches []*sketch.Sketch
}

// BuildBlade lofts the blade at pitch (MinBladePitch to MaxBladePitch), joins its drilled root stud and returns it
// unplaced. doc needs a chain tolerance of 0.1 mm (NewDocument): the stud union refuses a plain document. ctx and doc
// must not be nil.
func BuildBlade(ctx context.Context, doc *decad.Document, pitch float64) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, false)
}

// BuildBladeMirrored builds the blade of the opposite hand: BuildBlade's blade reflected in the plane x = 0 of the
// blade's local frame, so it moves toward +X and its concave face still faces the way the air goes. Its stud and bolt
// hole are on that plane and do not change. The arguments are BuildBlade's.
func BuildBladeMirrored(ctx context.Context, doc *decad.Document, pitch float64) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, true)
}

func buildBlade(ctx context.Context, doc *decad.Document, pitch float64, mirror bool) (*Blade, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("handcrankfan: context and document must not be nil")
	}
	if !finite(pitch) || pitch < MinBladePitch || pitch > MaxBladePitch {
		return nil, errors.New("handcrankfan: pitch must lie in [10°, 40°] radians")
	}
	b := &builder{ctx: ctx, doc: doc, name: "blade", mirror: mirror}
	body := b.bladeBody(pitch)
	if b.err != nil {
		return nil, b.err
	}
	if body == nil || !body.IsSolid() {
		return nil, errors.New("handcrankfan: blade is not a solid")
	}
	return &Blade{Body: body, Sketches: b.a.Sketches}, nil
}

// BladeSocketTool builds one blade socket's cutter: a cylinder of radius BladeSocketRadius from f's origin, the socket
// floor's centre, along f's normal (the radial line, outward) for HubRadius + 1 - BladeSocketStart, its start cap's
// outer loop rounded by BladeSocketFloorRound. It returns the cutter and its sketch, fully constrained.
func BladeSocketTool(ctx context.Context, doc *decad.Document, f r3.Frame) (*decad.Body, *sketch.Sketch, error) {
	b := &builder{ctx: ctx, doc: doc, name: "socket"}
	body := b.roundStart(b.cylinder(f, socketRadius, hubRadius+1-socketStart), socketFloorRound)
	if b.err != nil {
		return nil, nil, b.err
	}
	if len(b.a.Sketches) != 1 {
		return nil, nil, fmt.Errorf("handcrankfan: socket tool made %d sketches", len(b.a.Sketches))
	}
	return body, b.a.Sketches[0], nil
}
