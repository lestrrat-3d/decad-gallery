package spurgear

import (
	"context"
	"fmt"

	"github.com/lestrrat-3d/decad"
	gearplacement "github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/param"
	"github.com/lestrrat-3d/units"
)

// Variant changes the parts of the spur pipeline that a spur specialization
// replaces. The zero value is the spur gear. Everything it leaves nil runs
// unchanged: the root disc, tooth pattern, join, bore and cap chamfer.
type Variant struct {
	// Define adds the variant's parameters to a parameter table. It runs on
	// the document's table and on the sketch world's table, after the spur
	// parameters, so its expressions may use them.
	Define func(*param.Table) error
	// Tooth builds tooth zero on the lower cap. Nil extrudes the spur tooth.
	Tooth func(ctx context.Context, in ToothInput) (*decad.Body, error)
	// LocalBuild builds the gear about the local origin and +Z and moves the finished
	// body onto Params.Placement with one Transform. A tooth built at an offset or
	// tilted frame leaves its held mesh too coarse for the tooth Unions to certify.
	LocalBuild bool
	// RootFillet replaces the spur root fillet. Nil keeps the spur fillet.
	RootFillet func(ctx context.Context, doc *decad.Document, gear *decad.Body, frame r3.Frame) (*decad.Body, error)
}

// ToothInput is what a variant's Tooth builder works from. Every length is read
// from the document's parameter table.
type ToothInput struct {
	Doc    *decad.Document
	World  *sketch.World
	Params Params
	// Frame is the gear frame; its origin is the reference point and its
	// normal is the gear axis. Lower is its plane, the lower cap.
	Frame r3.Frame
	Lower *sketch.Plane
}

// Shifted returns the gear frame moved dist along its normal.
func (in ToothInput) Shifted(dist units.Value) (r3.Frame, error) {
	return shifted(in.Frame, dist.Base())
}

// PlaneAt returns the sketch plane parallel to the lower cap, dist above it.
func (in ToothInput) PlaneAt(dist units.Value) (*sketch.Plane, error) {
	f, err := in.Shifted(dist)
	if err != nil {
		return nil, err
	}
	return in.World.CreatePlaneFromFrame(f)
}

// run is the state one build of the pipeline shares.
type run struct {
	doc    *decad.Document
	world  *sketch.World
	p      Params
	d      dimensions
	frame  r3.Frame
	lower  *sketch.Plane
	extent decad.Distance
	teeth  int
}

// begin validates p, defines the parameters and resolves the placement.
func (v Variant) begin(ctx context.Context, doc *decad.Document, p Params) (*run, error) {
	if v.LocalBuild {
		p.Placement = nil
	}
	if ctx == nil {
		return nil, fmt.Errorf("spurgear: nil context")
	}
	if doc == nil {
		return nil, fmt.Errorf("spurgear: nil document")
	}
	if err := validate(p); err != nil {
		return nil, err
	}
	world := sketch.NewWorld()
	for name, t := range map[string]*param.Table{"document": doc.Params(), "world": world.Params()} {
		if err := defineParameters(t, p); err != nil {
			return nil, fmt.Errorf("spurgear: define %s parameters: %w", name, err)
		}
		if v.Define == nil {
			continue
		}
		if err := v.Define(t); err != nil {
			return nil, fmt.Errorf("spurgear: define %s variant parameters: %w", name, err)
		}
	}
	d, err := readDimensions(doc)
	if err != nil {
		return nil, fmt.Errorf("spurgear: %w", err)
	}
	teeth, err := doc.ParamInt("teeth")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read teeth: %w", err)
	}
	thickness, err := doc.Param("thickness")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read thickness: %w", err)
	}
	frame, err := placement(p)
	if err != nil {
		return nil, fmt.Errorf("spurgear: placement: %w", err)
	}
	lower := world.XY()
	if p.Placement != nil {
		lower, err = world.CreatePlaneFromFrame(frame)
		if err != nil {
			return nil, fmt.Errorf("spurgear: placement plane: %w", err)
		}
	}
	return &run{doc: doc, world: world, p: p, d: d, frame: frame, lower: lower, teeth: teeth,
		extent: decad.Distance{D: thickness, Dir: decad.Along}}, nil
}

// BuildTooth builds tooth zero alone, without the disc, pattern or finishing.
func (v Variant) BuildTooth(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	r, err := v.begin(ctx, doc, p)
	if err != nil {
		return nil, err
	}
	return v.tooth(ctx, r)
}

// tooth builds tooth zero: the variant's, or the spur tooth extruded.
func (v Variant) tooth(ctx context.Context, r *run) (*decad.Body, error) {
	if v.Tooth != nil {
		tooth, err := v.Tooth(ctx, ToothInput{Doc: r.doc, World: r.world, Params: r.p, Frame: r.frame, Lower: r.lower})
		if err != nil {
			return nil, fmt.Errorf("spurgear: tooth: %w", err)
		}
		return tooth, nil
	}
	var ts *sketch.Sketch
	var prof *sketch.Profile
	var err error
	if r.d.base < r.d.root {
		ts, prof, err = buildEmbeddedTooth(ctx, r.world, r.lower, r.p, r.d)
	} else {
		ts, prof, err = buildToothSection(ctx, r.world, r.lower, r.p, r.d, 0)
	}
	if err != nil {
		return nil, fmt.Errorf("spurgear: tooth section: %w", err)
	}
	tooth, err := decad.Extrude(ctx, r.doc, ts, prof, r.extent)
	if err != nil {
		return nil, fmt.Errorf("spurgear: extrude tooth: %w", err)
	}
	return tooth, nil
}

// filletSpurRoots rounds the spur root corners with the derived fillet radius.
func filletSpurRoots(ctx context.Context, doc *decad.Document, body *decad.Body, r *run,
	embedded bool) (*decad.Body, error) {
	radius, err := doc.Param("fillet_radius")
	if err != nil {
		return nil, fmt.Errorf("spurgear: read fillet radius: %w", err)
	}
	if embedded {
		return filletEmbedded(ctx, body, r.frame, r.teeth, radius)
	}
	var roots []point
	left, right := sampleFlanks(r.d.pitch, r.d.base, r.d.base, r.d.tip, float64(r.teeth), r.p.Samples, 0)
	for k := 0; k < r.teeth; k++ {
		a := toothAngle(r.p, k)
		rr := rootContact(right[0], r.d)
		lr := rootContact(left[0], r.d)
		roots = append(roots, rotate(rr.x, rr.y, a), rotate(lr.x, lr.y, a))
	}
	return filletRootsValue(ctx, body, r.frame, roots, radius)
}

// buildLocal builds the gear about the local origin and +Z, then places it.
func (v Variant) buildLocal(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, fmt.Errorf("spurgear: invalid placement frame")
	}
	local := p
	local.Placement = nil
	r, err := v.begin(ctx, doc, local)
	if err != nil {
		return nil, err
	}
	gear, err := v.assemble(ctx, r)
	if err != nil {
		return nil, err
	}
	return gearplacement.Place(ctx, gear, p.Placement)
}
