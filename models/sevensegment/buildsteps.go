package sevensegment

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// maxStepWorkers caps how many build steps run at once. Each extra worker holds about 0.35 GB of geometry, and four
// workers already reach the build's CPU bound on a CI runner.
const maxStepWorkers = 4

// buildStep is one step of a build. build runs on the step's own run and returns file; the runner calls file after
// every step has finished, serially and in step order, to store what the step made. build must reach the document,
// the context, the parameters and the dimensions only through r.
type buildStep struct {
	name  string
	build func(r *run) (file func(), err error)
}

// stepWorkers is the worker option BuildParts and Build pass to decad.RunSteps.
func stepWorkers() decad.RunStepsOption {
	return decad.WithStepWorkers(min(runtime.GOMAXPROCS(0), maxStepWorkers))
}

// stepResult is what one step leaves for runSteps. Only the step itself writes its entry.
type stepResult struct {
	file     func()
	sketches []*sketch.Sketch
	err      error
}

// runSteps checks ctx, doc and p, derives the dimensions, runs steps through decad.RunSteps on doc (each on a fresh
// run whose ctx is the step's ctx), files their results in step order and returns the dimensions and the sketches in
// step order. On an error it files nothing.
func runSteps(ctx context.Context, doc *decad.Document, p Params, steps []buildStep,
	opts ...decad.RunStepsOption) (Dimensions, []*sketch.Sketch, error) {
	if ctx == nil {
		return Dimensions{}, nil, errors.New("sevensegment: context must not be nil")
	}
	if doc == nil {
		return Dimensions{}, nil, errors.New("sevensegment: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return Dimensions{}, nil, err
	}
	if err := ctx.Err(); err != nil {
		return Dimensions{}, nil, fmt.Errorf("sevensegment: before creating geometry: %w", err)
	}
	results := make([]stepResult, len(steps))
	list := make([]decad.Step, len(steps))
	for i, st := range steps {
		list[i] = decad.Step{Name: st.name, Build: func(sctx context.Context) error {
			r := &run{ctx: sctx, doc: doc, p: p, dm: dm}
			file, err := st.build(r)
			results[i] = stepResult{file: file, sketches: r.sketches, err: err}
			return err
		}}
	}
	runErr := decad.RunSteps(ctx, doc, list, opts...)
	// Steps are independent, so the first failure in step order is the error a serial build returns.
	for _, res := range results {
		if res.err != nil {
			return Dimensions{}, nil, res.err
		}
	}
	if runErr != nil {
		return Dimensions{}, nil, fmt.Errorf("sevensegment: build steps: %w", runErr)
	}
	var sketches []*sketch.Sketch
	for _, res := range results {
		if res.file != nil {
			res.file()
		}
		sketches = append(sketches, res.sketches...)
	}
	return dm, sketches, nil
}

// partSteps lists BuildParts's steps in build order. It has no context, document or parameters in scope, so a step
// can only use its own run.
func partSteps(out *Parts) []buildStep {
	steps := []buildStep{
		{"flags", func(r *run) (func(), error) {
			flags, err := r.flags()
			if err != nil {
				return nil, err
			}
			return func() { out.Flags = flags }, nil
		}},
		{"pivot blocks", func(r *run) (func(), error) {
			blocks, err := r.pivotBlocks()
			if err != nil {
				return nil, fmt.Errorf("sevensegment: pivot blocks: %w", err)
			}
			return func() { out.PivotBlocks = blocks }, nil
		}},
		single("front wall", func(r *run) (*decad.Body, error) { return r.wall(true) }, nil, &out.FrontWall),
		single("rear wall", func(r *run) (*decad.Body, error) { return r.wall(false) }, nil, &out.RearWall),
		single("guide plate", (*run).columnGuidePlate, nil, &out.GuidePlate),
		single("cradle plate", (*run).cradlePlate, nil, &out.CradlePlate),
		single("top plate", (*run).flagTopPlate, nil, &out.TopPlate),
		single("shaft", (*run).shaft, (*run).spin, &out.Shaft),
		single("hub", (*run).hub, (*run).spin, &out.Hub),
		single("rotor", (*run).rotor, (*run).spin, &out.Rotor),
		single("housing", (*run).housing, nil, &out.Housing),
		single("button", (*run).button, nil, &out.Button),
		single("spring", func(r *run) (*decad.Body, error) { return r.spring(SpringRestLength) }, (*run).spin, &out.Spring),
	}
	for _, seg := range Segments() {
		steps = append(steps,
			single("cam "+seg.String(), func(r *run) (*decad.Body, error) { return r.cam(seg) }, (*run).spin, &out.Cams[seg]),
			single("slider "+seg.String(), func(r *run) (*decad.Body, error) { return r.flagSlider(seg) }, liftIfLit(seg),
				&out.Sliders[seg]))
		if _, _, horizontal := segmentCentre(seg); horizontal {
			steps = append(steps, single("groove pin "+seg.String(),
				func(r *run) (*decad.Body, error) { return r.groovePinFor(seg) }, liftIfLit(seg), &out.GroovePins[seg]))
		}
	}
	return steps
}

// hardwareSteps lists Build's bearing, bolt and nut steps, after the part steps.
func hardwareSteps(m *Model) []buildStep {
	var steps []buildStep
	for i, f := range []struct {
		origin, u, v r3.Vec
	}{
		{r3.Vec{Y: frontInnerY, Z: ShaftZ}, r3.Vec{X: 1}, r3.Vec{Z: 1}},
		{r3.Vec{Y: rearInnerY, Z: ShaftZ}, r3.Vec{Z: 1}, r3.Vec{X: 1}},
	} {
		steps = append(steps, buildStep{fmt.Sprintf("bearing %d", i+1), func(r *run) (func(), error) {
			frame, err := r3.NewFrame(f.origin, f.u, f.v)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: bearing %d frame: %w", i+1, err)
			}
			bp := ballbearing.Params608ZZ()
			bp.Placement = &frame
			if r.p.Placement != nil {
				lift, err := r3.FromFrame(*r.p.Placement)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: bearing %d placement: %w", i+1, err)
				}
				bp.Lift = &lift
			}
			bearing, err := ballbearing.Build(r.ctx, r.doc, bp)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: bearing %d: %w", i+1, err)
			}
			return func() { m.Bearings[i] = bearing }, nil
		}})
	}
	for i := range m.Bolts {
		steps = append(steps,
			buildStep{fmt.Sprintf("bolt %d", i+1), func(r *run) (func(), error) {
				site, err := placeSite(r.dm.Sites[i], r.p.Placement)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: bolt %d: %w", i+1, err)
				}
				bolt, err := m2fastener.PlaceBolt(r.ctx, r.doc, site)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: bolt %d: %w", i+1, err)
				}
				return func() { m.Bolts[i] = bolt }, nil
			}},
			buildStep{fmt.Sprintf("nut %d", i+1), func(r *run) (func(), error) {
				site, err := placeSite(r.dm.Sites[i], r.p.Placement)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: nut %d: %w", i+1, err)
				}
				nut, err := m2fastener.PlaceNut(r.ctx, r.doc, site)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: nut %d: %w", i+1, err)
				}
				return func() { m.Nuts[i] = nut }, nil
			}})
	}
	return steps
}

// single makes a one-body step: it builds the body, checks it is a solid, poses it (a nil pose keeps it), places it
// and files it in dst.
func single(name string, build func(*run) (*decad.Body, error), pose func(*run, *decad.Body) (*decad.Body, error),
	dst **decad.Body) buildStep {
	return buildStep{name, func(r *run) (func(), error) {
		body, err := build(r)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: %s: %w", name, err)
		}
		if body == nil || !body.IsSolid() {
			return nil, fmt.Errorf("sevensegment: %s is not a solid", name)
		}
		if pose != nil {
			if body, err = pose(r, body); err != nil {
				return nil, fmt.Errorf("sevensegment: pose %s: %w", name, err)
			}
		}
		placed, err := placement.Place(r.ctx, body, r.p.Placement)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: place %s: %w", name, err)
		}
		return func() { *dst = placed }, nil
	}}
}

// spin turns b with the camshaft group.
func (r *run) spin(b *decad.Body) (*decad.Body, error) {
	return decad.Transform(b, r.camshaftTurn()).Do(r.ctx)
}

// liftIfLit raises b by the slider travel when segment seg is lit for the digit.
func liftIfLit(seg Segment) func(r *run, b *decad.Body) (*decad.Body, error) {
	return func(r *run, b *decad.Body) (*decad.Body, error) {
		lit, _ := Lit(r.p.Digit, seg)
		if !lit {
			return b, nil
		}
		lift, err := r3.Translation(r3.Vec{Z: Travel})
		if err != nil {
			return nil, err
		}
		return decad.Transform(b, lift).Do(r.ctx)
	}
}

// flags builds the 14 flag bodies, turned about their segment axes and placed.
func (r *run) flags() ([7]Flag, error) {
	var out [7]Flag
	var prototypes [2]Flag
	for _, seg := range Segments() {
		f, err := r.prototypeFlag(seg, &prototypes)
		if err != nil {
			return out, fmt.Errorf("sevensegment: flag %s: %w", seg, err)
		}
		cx, cy, horizontal := segmentCentre(seg)
		axis := r3.Vec{Y: 1}
		if horizontal {
			axis = r3.Vec{X: 1}
		}
		turn, err := r3.RotationAround(r3.Vec{X: cx, Y: cy, Z: RotorAxisZ}, axis,
			units.Degrees(r.dm.FlagAngle(seg, r.dm.Theta)))
		if err != nil {
			return out, fmt.Errorf("sevensegment: flag %s: turn: %w", seg, err)
		}
		for _, b := range []struct {
			name string
			in   *decad.Body
			dst  **decad.Body
		}{{"yellow face", f.Yellow, &out[seg].Yellow}, {"black body", f.Black, &out[seg].Black}} {
			posed, err := decad.Transform(b.in, turn).Do(r.ctx)
			if err != nil {
				return out, fmt.Errorf("sevensegment: flag %s %s: turn: %w", seg, b.name, err)
			}
			if *b.dst, err = placement.Place(r.ctx, posed, r.p.Placement); err != nil {
				return out, fmt.Errorf("sevensegment: place flag %s %s: %w", seg, b.name, err)
			}
		}
	}
	return out, nil
}
