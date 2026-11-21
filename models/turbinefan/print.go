package turbinefan

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Print pose and print-process sizes, in mm or radians. instructions.md "Print pose and nose rib" gives each one's
// reason.
const (
	// PrintLean turns the chord at RefRadius from vertical toward the pressure face (negative values).
	PrintLean = -15 * math.Pi / 180
	// PrintLift is the z of the blade's stud flat in the print pose: the flat sits on the bed, and the stud end corner
	// the flat cut away StudFlatDepth under it.
	PrintLift = 0.0
	// StudFlatDepth is how far the flat on the stud's inner end reaches into the stud: the stud end corner lies this far
	// under the flat's plane, so the blade rests on a flat patch instead of a corner sliver.
	StudFlatDepth = 0.5
	// StudFlatRootClearance is the least height of the round root's lowest point over the bed at the rest tilt: one
	// layer, so the root prints no stray island in the first layer.
	StudFlatRootClearance = 0.2
	// RestTiltMin and RestTiltMax bracket the rest tilt RestTilt finds, and the tilt PrintPoseFor takes.
	RestTiltMin = 5 * math.Pi / 180
	RestTiltMax = 20 * math.Pi / 180
)

// HarnessContainerName is the container BuildBladePrint files the harness in.
const HarnessContainerName = "Harness"

// checkPose validates the pose arguments shared by RestTilt and PrintPoseFor.
func checkPose(sz Size, pitch float64, hand Hand) error {
	if hand != RightHand && hand != LeftHand {
		return errors.New("turbinefan: hand must be RightHand or LeftHand")
	}
	if math.IsNaN(pitch) || pitch < MinBladePitch || pitch > MaxBladePitch {
		return errors.New("turbinefan: pitch must lie in [10°, 40°] radians")
	}
	if !sz.valid() {
		return errors.New("turbinefan: sz must come from SizeFor")
	}
	return nil
}

// poseUp returns the in-section up direction of the print pose in the blade's local frame: the chord at RefRadius,
// from the leading to the trailing edge, turned by PrintLean (mirrored for LeftHand).
func poseUp(sz Size, pitch float64, hand Hand) r3.Vec {
	b := sz.Beta(sz.RefRadius, pitch)
	c := r3.Vec{X: math.Cos(b), Y: math.Sin(b)}
	l := PrintLean
	if hand == LeftHand {
		c.X, l = -c.X, -l
	}
	return r3.Vec{X: c.X*math.Cos(l) - c.Y*math.Sin(l), Y: c.X*math.Sin(l) + c.Y*math.Cos(l)}
}

// studCornerZ returns the stud end corner's height over the radial line's foot at tilt, before the lift.
func studCornerZ(sz Size, tilt float64) float64 {
	return (sz.BladeStudStart+studChamfer)*math.Sin(tilt) - sz.BladeStudRadius*math.Cos(tilt)
}

// RestTilt returns the tilt of the radial line above the bed at which the blade, posed with PrintLean, rests on both
// its stud flat and its loft's lowest point: the stud end corner the flat cut away lies StudFlatDepth under the loft's
// lowest point. blade is the body BuildBlade or BuildBladeMirrored returned (local frame); the blade's flat was cut at
// the same tilt. The tilt depends only on sz; pitch and hand are checked like PrintPoseFor's. It fails when the round
// root would stand less than StudFlatRootClearance over the bed. ctx and blade must not be nil.
func RestTilt(ctx context.Context, blade *decad.Body, sz Size, pitch float64, hand Hand) (float64, error) {
	if ctx == nil || blade == nil {
		return 0, errors.New("turbinefan: context and blade must not be nil")
	}
	if err := checkPose(sz, pitch, hand); err != nil {
		return 0, err
	}
	mesh, err := blade.Tessellate(ctx, units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	if err != nil {
		return 0, fmt.Errorf("turbinefan: rest tilt: %w", err)
	}
	tilt, _, err := restTilt(mesh.Vertices(), sz, pitch, hand, StudFlatDepth)
	return tilt, err
}

// restTilt solves the rest tilt from a blade's or its loft's mesh vertices in the local frame: the tilt th at which the
// stud end corner stands depth under the loft's lowest point (vertices with Z over RootRadius + 0.5). It returns th
// and the loft's lowest pose height before the lift, sin th Z + cos th u·p, which is where the flat's plane lies. It
// fails when the round root's lowest point (vertices with Z from RootRadius to RootRadius + 2 HubScale) would stand less
// than StudFlatRootClearance over the bed.
func restTilt(vertices []r3.Vec, sz Size, pitch float64, hand Hand, depth float64) (float64, float64, error) {
	var loft, root []r3.Vec
	for _, v := range vertices {
		if v.Z > sz.RootRadius+0.5 {
			loft = append(loft, v)
		}
		if v.Z > sz.RootRadius-0.01 && v.Z < sz.RootRadius+2*sz.HubScale {
			root = append(root, v)
		}
	}
	u := poseUp(sz, pitch, hand)
	lowest := func(vs []r3.Vec, th float64) float64 {
		low := math.Inf(1)
		for _, v := range vs {
			low = math.Min(low, math.Sin(th)*v.Z+math.Cos(th)*v.Dot(u))
		}
		return low
	}
	// over returns how far the stud end corner, lowered by depth, stands over the loft's lowest point at tilt th.
	over := func(th float64) float64 { return studCornerZ(sz, th) + depth - lowest(loft, th) }
	lo, hi := RestTiltMin, RestTiltMax
	if over(lo) <= 0 || over(hi) >= 0 {
		return 0, 0, errors.New("turbinefan: rest tilt: the stud corner and the loft do not cross between 5° and 20°")
	}
	for range 50 {
		m := (lo + hi) / 2
		if over(m) > 0 {
			lo = m
			continue
		}
		hi = m
	}
	th := (lo + hi) / 2
	plane := lowest(loft, th)
	if h := lowest(root, th) - plane; h < StudFlatRootClearance {
		return 0, 0, fmt.Errorf("turbinefan: rest tilt: the root stands %.2f mm over the bed, under %g", h,
			StudFlatRootClearance)
	}
	return th, plane, nil
}

// PrintPoseFor returns the frame that moves a blade built by BuildBlade (hand RightHand) or BuildBladeMirrored
// (LeftHand) at pitch and sz from its local frame to the print frame at tilt: bed z = 0, +Z up, the stud end corner the
// flat cut away at z = -StudFlatDepth, +X along the bed from stud to tip, leading edge down. tilt must lie in
// [RestTiltMin, RestTiltMax]; at the tilt RestTilt gives, the stud flat lies on the bed and the leading edge reaches it.
// Apply the frame with xf, _ := r3.FromFrame(f); decad.Transform(ctx, body, xf). The pose is set from the blade's own
// chord at RefRadius, so the posed loft is the same at every pitch; the left hand is the right hand reflected in y = 0.
func PrintPoseFor(sz Size, pitch float64, hand Hand, tilt float64) (r3.Frame, error) {
	if err := checkPose(sz, pitch, hand); err != nil {
		return r3.Frame{}, err
	}
	if math.IsNaN(tilt) || tilt < RestTiltMin || tilt > RestTiltMax {
		return r3.Frame{}, errors.New("turbinefan: tilt must lie in [5°, 20°] radians")
	}
	u := poseUp(sz, pitch, hand)
	v := r3.Vec{Z: 1}.Cross(u)
	bb := r3.Vec{X: -math.Sin(tilt), Z: math.Cos(tilt)}
	cc := r3.Vec{Y: -1}
	imgX := bb.Scale(u.X).Add(cc.Scale(v.X))
	imgY := bb.Scale(u.Y).Add(cc.Scale(v.Y))
	if n := imgX.Cross(imgY); n.Sub(r3.Vec{X: math.Cos(tilt), Z: math.Sin(tilt)}).Len() > 1e-9 {
		return r3.Frame{}, fmt.Errorf("turbinefan: print pose: radial line maps to %v", n)
	}
	origin := r3.Vec{X: -sz.BladeStudStart * math.Cos(tilt), Z: PrintLift - StudFlatDepth - studCornerZ(sz, tilt)}
	f, err := r3.NewFrame(origin, imgX, imgY)
	if err != nil {
		return r3.Frame{}, fmt.Errorf("turbinefan: print pose: %w", err)
	}
	return f, nil
}

// BladePrint is a blade and its harness in the print frame, filed in BladesContainerName and HarnessContainerName.
type BladePrint struct {
	Blade         *decad.Body
	BladeSketches []*sketch.Sketch
	Harness       *BladeHarness
	Pose          r3.Frame
	// Tilt is the rest tilt the blade is posed at.
	Tilt float64
}

// BuildBladePrint validates p (Derive), builds the blade of p.Hand at p.BladePitch and SizeFor(p.RotorDiameter),
// finds its RestTilt, moves it to PrintPoseFor at that tilt and builds its harness. It ignores Mount and Placement. doc must come from NewDocument.
// A rejected build adds no body; a later geometry failure may leave bodies in doc, which the caller discards.
func BuildBladePrint(ctx context.Context, doc *decad.Document, p Params) (*BladePrint, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("turbinefan: context and document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	if err := checkPose(dm.Size, p.BladePitch, p.Hand); err != nil {
		return nil, err
	}
	build := BuildBlade
	if p.Hand == LeftHand {
		build = BuildBladeMirrored
	}
	blade, err := build(ctx, doc, p.BladePitch, dm.Size)
	if err != nil {
		return nil, err
	}
	tilt, err := RestTilt(ctx, blade.Body, dm.Size, p.BladePitch, p.Hand)
	if err != nil {
		return nil, err
	}
	pose, err := PrintPoseFor(dm.Size, p.BladePitch, p.Hand, tilt)
	if err != nil {
		return nil, err
	}
	xf, err := r3.FromFrame(pose)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: print pose: %w", err)
	}
	posed, err := decad.Transform(ctx, blade.Body, xf)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: pose blade: %w", err)
	}
	harness, err := BuildBladeHarness(ctx, doc, posed, dm.Size, p.Hand, tilt)
	if err != nil {
		return nil, err
	}
	for _, c := range []struct {
		name string
		body *decad.Body
	}{{BladesContainerName, posed}, {HarnessContainerName, harness.Body}} {
		box, err := doc.Root().ChildNamed(c.name)
		if err != nil {
			return nil, fmt.Errorf("turbinefan: container %s: %w", c.name, err)
		}
		if err := box.AddBody(c.body); err != nil {
			return nil, fmt.Errorf("turbinefan: add body to %s: %w", c.name, err)
		}
	}
	return &BladePrint{Blade: posed, BladeSketches: blade.Sketches, Harness: harness, Pose: pose, Tilt: tilt}, nil
}
