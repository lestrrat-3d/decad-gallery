package turbinefan

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// Print pose and print-process sizes, in mm or radians. instructions.md "Print pose and harness" gives each one's reason.
const (
	// PrintTilt is the blade's radial line's angle above the bed in the print pose. It exceeds the rest tilt at every
	// size, so the stud corner is the blade's lowest point.
	PrintTilt = 14 * math.Pi / 180
	// PrintLean turns the chord at RefRadius from vertical toward the pressure face (negative values), so the trailing
	// edge leans away from the wall's struts.
	PrintLean = -15 * math.Pi / 180
	// HarnessPlateThickness is the harness plate's thickness on the bed.
	HarnessPlateThickness = 1.2
	// CradleGapZ is the vertical support gap from the cradle and the plate to the blade: one 0.2 mm layer.
	CradleGapZ = 0.2
	// PrintLift is the z of the blade's lowest point, the stud corner, in the print pose: one gap over the plate.
	PrintLift = HarnessPlateThickness + CradleGapZ
)

// HarnessContainerName is the container BuildBladePrint files the harness in.
const HarnessContainerName = "Harness"

// PrintPoseFor returns the frame that moves a blade built by BuildBlade (hand RightHand) or BuildBladeMirrored
// (LeftHand) at pitch and sz from its local frame to the print frame: bed z = 0, +Z up, +X along the bed from stud to
// tip, leading edge down. Apply it with xf, _ := r3.FromFrame(f); decad.Transform(ctx, body, xf). The pose is set from
// the blade's own chord at RefRadius, so the posed loft is the same at every pitch; the left hand is the right hand
// reflected in y = 0.
func PrintPoseFor(sz Size, pitch float64, hand Hand) (r3.Frame, error) {
	if hand != RightHand && hand != LeftHand {
		return r3.Frame{}, errors.New("turbinefan: hand must be RightHand or LeftHand")
	}
	if math.IsNaN(pitch) || pitch < MinBladePitch || pitch > MaxBladePitch {
		return r3.Frame{}, errors.New("turbinefan: pitch must lie in [10°, 40°] radians")
	}
	if !sz.valid() {
		return r3.Frame{}, errors.New("turbinefan: sz must come from SizeFor")
	}
	b := sz.Beta(sz.RefRadius, pitch)
	c := r3.Vec{X: math.Cos(b), Y: math.Sin(b)} // chord from the leading to the trailing edge at RefRadius
	l := PrintLean
	if hand == LeftHand {
		c.X, l = -c.X, -l
	}
	u := r3.Vec{X: c.X*math.Cos(l) - c.Y*math.Sin(l), Y: c.X*math.Sin(l) + c.Y*math.Cos(l)} // in-section up
	v := r3.Vec{Z: 1}.Cross(u)
	t := PrintTilt
	bb := r3.Vec{X: -math.Sin(t), Z: math.Cos(t)}
	cc := r3.Vec{Y: -1}
	imgX := bb.Scale(u.X).Add(cc.Scale(v.X))
	imgY := bb.Scale(u.Y).Add(cc.Scale(v.Y))
	if n := imgX.Cross(imgY); n.Sub(r3.Vec{X: math.Cos(t), Z: math.Sin(t)}).Len() > 1e-9 {
		return r3.Frame{}, fmt.Errorf("turbinefan: print pose: radial line maps to %v", n)
	}
	zc := (sz.BladeStudStart+studChamfer)*math.Sin(t) - sz.BladeStudRadius*math.Cos(t) // stud corner after the turn
	f, err := r3.NewFrame(r3.Vec{X: -sz.BladeStudStart * math.Cos(t), Z: PrintLift - zc}, imgX, imgY)
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
}

// BuildBladePrint validates p (Derive), builds the blade of p.Hand at p.BladePitch and SizeFor(p.RotorDiameter),
// moves it to PrintPoseFor and builds its harness. It ignores Mount and Placement. doc must come from NewDocument.
// A rejected build adds no body; a later geometry failure may leave bodies in doc, which the caller discards.
func BuildBladePrint(ctx context.Context, doc *decad.Document, p Params) (*BladePrint, error) {
	if ctx == nil || doc == nil {
		return nil, errors.New("turbinefan: context and document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return nil, err
	}
	pose, err := PrintPoseFor(dm.Size, p.BladePitch, p.Hand)
	if err != nil {
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
	xf, err := r3.FromFrame(pose)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: print pose: %w", err)
	}
	posed, err := decad.Transform(ctx, blade.Body, xf)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: pose blade: %w", err)
	}
	harness, err := BuildBladeHarness(ctx, doc, posed, dm.Size, p.Hand)
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
	return &BladePrint{Blade: posed, BladeSketches: blade.Sketches, Harness: harness, Pose: pose}, nil
}
