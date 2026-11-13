// Package cornercheck reads sections of a printed part and reports sharp corners.
//
// It implements the check in "Corners and roots" of docs/model-pipeline.md. A section is the part cut by a plane
// normal to the print axis. An inside corner is a turn toward the material of more than InsideMaxTurn within
// InsideWindow of outline. A tip is a turn away from the material of more than TipMaxTurn within TipWindow.
package cornercheck

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

const (
	// InsideWindow is the outline length in mm over which an inside corner's turn is summed.
	InsideWindow = 0.4
	// InsideMaxTurn is the turn in degrees toward the material, within InsideWindow, that a section may not exceed.
	InsideMaxTurn = 50.0
	// TipWindow is the outline length in mm over which a tip's turn is summed.
	TipWindow = 0.8
	// TipMaxTurn is the turn in degrees away from the material, within TipWindow, that a section may not exceed.
	TipMaxTurn = 120.0
	// Tolerance is the first section chord tolerance in mm that Section tries.
	Tolerance = 0.01
	// MaxMeshBound is the coarsest held mesh bound in mm that a section may be read from. A coarser mesh can
	// collapse a 0.3 mm round into one chord, and the check would pass it.
	MaxMeshBound = 0.05

	// turnSlack is added to a limit before comparing, so an exact 60 degree tip (a 120 degree turn) passes.
	turnSlack = 0.01
	// minRun is the thinnest run of layers, in mm, that MidHeights cuts.
	minRun = 0.2
	// levelMerge is the distance in mm within which two face levels count as one.
	levelMerge = 1e-6
)

// Kind tells which rule a Finding breaks.
type Kind int

const (
	// Inside is a sharp inside corner.
	Inside Kind = iota
	// Tip is a sharp outward tip.
	Tip
)

// String returns the kind's name.
func (k Kind) String() string {
	if k == Tip {
		return "tip"
	}
	return "inside"
}

// Finding is one sharp corner on one section.
type Finding struct {
	Kind Kind
	// At is the world point on the outline at the corner's sharpest vertex.
	At r3.Vec
	// Turn is the largest turn in degrees, positive, found within the window around the corner.
	Turn float64
	// Height is the section's offset along the axis.
	Height float64
}

// Exempt names one listed corner. Near reports whether a finding point belongs to it.
type Exempt struct {
	Reason string
	Near   func(p r3.Vec) bool
}

// Disc returns a test for points within radius of center, measured in the plane normal to axis. The point's
// component along axis is ignored, so one center serves every section height.
func Disc(center, axis r3.Vec, radius float64) func(r3.Vec) bool {
	a, ok := axis.Normalize()
	if !ok {
		return func(r3.Vec) bool { return false }
	}
	return func(p r3.Vec) bool {
		d := p.Sub(center)
		return d.Sub(a.Scale(d.Dot(a))).Len() <= radius
	}
}

// Annulus returns a test for points whose distance from the line through center along axis lies in [rMin, rMax].
func Annulus(center, axis r3.Vec, rMin, rMax float64) func(r3.Vec) bool {
	a, ok := axis.Normalize()
	if !ok {
		return func(r3.Vec) bool { return false }
	}
	return func(p r3.Vec) bool {
		d := p.Sub(center)
		r := d.Sub(a.Scale(d.Dot(a))).Len()
		return r >= rMin && r <= rMax
	}
}

// MidHeights returns the offsets along axis halfway between consecutive levels of the body's planar faces whose
// normal is parallel to axis. It merges levels within 1e-6 mm and skips runs thinner than 0.2 mm.
func MidHeights(body *decad.Body, axis r3.Vec) ([]float64, error) {
	a, ok := axis.Normalize()
	if !ok {
		return nil, errors.New("cornercheck: zero axis")
	}
	var levels []float64
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(math.Abs(plane.Frame.N().Dot(a))-1) > 1e-9 {
			continue
		}
		levels = append(levels, plane.Frame.Origin().Dot(a))
	}
	slices.Sort(levels)
	var merged []float64
	for _, l := range levels {
		if len(merged) == 0 || l-merged[len(merged)-1] > levelMerge {
			merged = append(merged, l)
		}
	}
	var mids []float64
	for i := 1; i < len(merged); i++ {
		if merged[i]-merged[i-1] >= minRun {
			mids = append(mids, (merged[i]+merged[i-1])/2)
		}
	}
	return mids, nil
}

// Section cuts body with the plane normal to axis at the given offset and returns its findings.
func Section(ctx context.Context, body *decad.Body, axis r3.Vec, height float64) ([]Finding, error) {
	a, ok := axis.Normalize()
	if !ok {
		return nil, errors.New("cornercheck: zero axis")
	}
	helper := r3.Vec{X: 1}
	if math.Abs(a.X) > 0.9 {
		helper = r3.Vec{Y: 1}
	}
	u, _ := helper.Cross(a).Normalize()
	frame, err := r3.NewFrame(a.Scale(height), u, a.Cross(u))
	if err != nil {
		return nil, fmt.Errorf("cornercheck: section frame: %w", err)
	}
	outline, err := body.SectionOutline(ctx, frame, units.Millimeters(Tolerance))
	var bound *decad.MeshBoundError
	if errors.As(err, &bound) {
		floor := bound.Floor().Base()
		if floor > MaxMeshBound {
			return nil, fmt.Errorf("cornercheck: held mesh bound %.3f mm exceeds %.2f mm", floor, MaxMeshBound)
		}
		outline, err = body.SectionOutline(ctx, frame, bound.Floor())
	}
	if err != nil {
		return nil, fmt.Errorf("cornercheck: section at %.3f: %w", height, err)
	}
	var out []Finding
	for _, loop := range outline.Loops {
		out = append(out, scanLoop(loop, frame.N(), height)...)
	}
	return out, nil
}

// Check runs Section at every MidHeights height, drops findings an exemption claims, and returns the rest. It
// returns an error naming each exemption that claims no finding.
func Check(ctx context.Context, body *decad.Body, axis r3.Vec, exempt []Exempt) ([]Finding, error) {
	heights, err := MidHeights(body, axis)
	if err != nil {
		return nil, err
	}
	return CheckAt(ctx, body, axis, heights, exempt)
}

// CheckAt is Check at explicit section heights. Use it where the face levels are not the runs that share an
// outline, such as a herringbone gear, whose middle face level is the apex where its two helices meet.
func CheckAt(ctx context.Context, body *decad.Body, axis r3.Vec, heights []float64, exempt []Exempt) ([]Finding, error) {
	claimed := make([]bool, len(exempt))
	var rest []Finding
	for _, h := range heights {
		fs, err := Section(ctx, body, axis, h)
		if err != nil {
			return nil, err
		}
		for _, f := range fs {
			hit := false
			for i, e := range exempt {
				if e.Near(f.At) {
					claimed[i], hit = true, true
				}
			}
			if !hit {
				rest = append(rest, f)
			}
		}
	}
	var unused []string
	for i, e := range exempt {
		if !claimed[i] {
			unused = append(unused, e.Reason)
		}
	}
	if len(unused) > 0 {
		return rest, fmt.Errorf("cornercheck: exemptions claim no finding: %s", strings.Join(unused, "; "))
	}
	return rest, nil
}

// Report formats at most max findings and the total count, for a test failure message.
func Report(fs []Finding, max int) string {
	if len(fs) == 0 {
		return "no findings"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d findings:", len(fs))
	for i, f := range fs {
		if i == max {
			fmt.Fprintf(&b, " ... %d more", len(fs)-max)
			break
		}
		fmt.Fprintf(&b, " [%s turn %.1f at (%.2f, %.2f, %.2f) height %.3f]",
			f.Kind, f.Turn, f.At.X, f.At.Y, f.At.Z, f.Height)
	}
	return b.String()
}
