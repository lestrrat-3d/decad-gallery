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

// stepResult is what one step leaves for runSteps. Only the step itself writes its entry. returned stays false when
// the step panicked or ended its goroutine, so it never wrote the other fields.
type stepResult struct {
	returned bool
	file     func()
	sketches []*sketch.Sketch
	err      error
}

// runSteps checks ctx, doc and p, derives the dimensions, runs steps through decad.RunSteps on doc (each on a fresh
// run whose ctx is the step's ctx), files their results in step order and returns the dimensions and the sketches in
// step order. On an error it files nothing.
func runSteps(ctx context.Context, doc *decad.Document, p Params, steps []buildStep,
	opts ...decad.RunStepsOption) (Dimensions, []*sketch.Sketch, error) {
	dm, _, sketches, err := runPosedSteps(ctx, doc, p, nil, steps, opts...)
	return dm, sketches, err
}

// runPosedSteps is runSteps with extra poses. Each step also gets one run per extra pose, in r.poses, that carries
// the pose's parameters and dimensions and the step's context. It returns the dimensions of p and of every extra
// pose, in order. Every extra pose is checked as BuildParts checks p.
func runPosedSteps(ctx context.Context, doc *decad.Document, p Params, extra []Pose, steps []buildStep,
	opts ...decad.RunStepsOption) (Dimensions, []Dimensions, []*sketch.Sketch, error) {
	if ctx == nil {
		return Dimensions{}, nil, nil, errors.New("sevensegment: context must not be nil")
	}
	if doc == nil {
		return Dimensions{}, nil, nil, errors.New("sevensegment: document must not be nil")
	}
	dm, err := p.Derive()
	if err != nil {
		return Dimensions{}, nil, nil, err
	}
	poseParams := make([]Params, len(extra))
	poseDims := make([]Dimensions, len(extra))
	for i, e := range extra {
		poseParams[i] = p
		poseParams[i].Digit, poseParams[i].Placement = e.Digit, e.Placement
		if poseDims[i], err = poseParams[i].Derive(); err != nil {
			return Dimensions{}, nil, nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Dimensions{}, nil, nil, fmt.Errorf("sevensegment: before creating geometry: %w", err)
	}
	results := make([]stepResult, len(steps))
	list := make([]decad.Step, len(steps))
	for i, st := range steps {
		list[i] = decad.Step{Name: st.name, Build: func(sctx context.Context) error {
			r := &run{ctx: sctx, doc: doc, p: p, dm: dm}
			for j := range extra {
				r.poses = append(r.poses, &run{ctx: sctx, doc: doc, p: poseParams[j], dm: poseDims[j]})
			}
			file, err := st.build(r)
			results[i] = stepResult{returned: true, file: file, sketches: r.sketches, err: err}
			return err
		}}
	}
	runErr := decad.RunSteps(ctx, doc, list, opts...)
	// Steps are independent, so the first failure in step order is the error a serial build returns. A step that
	// never returned (it panicked or ended its goroutine) is a failure at its place in the order: a serial build
	// would have stopped there, so a later step's error must not hide it.
	for _, res := range results {
		if !res.returned {
			if runErr == nil {
				runErr = errors.New("a step did not return")
			}
			return Dimensions{}, nil, nil, fmt.Errorf("sevensegment: build steps: %w", runErr)
		}
		if res.err != nil {
			return Dimensions{}, nil, nil, res.err
		}
	}
	if runErr != nil {
		return Dimensions{}, nil, nil, fmt.Errorf("sevensegment: build steps: %w", runErr)
	}
	var sketches []*sketch.Sketch
	for _, res := range results {
		if res.file != nil {
			res.file()
		}
		sketches = append(sketches, res.sketches...)
	}
	return dm, poseDims, sketches, nil
}

// partSteps lists BuildParts's steps in build order. It has no context, document or parameters in scope, so a step
// can only use its own run. out is the main parts and extras the parts of the extra poses, in order: a step files its
// body for pose i into the i-th of out followed by extras.
func partSteps(out *Parts, extras []*Parts) []buildStep {
	all := append([]*Parts{out}, extras...)
	one := func(name string, build func(*run) (*decad.Body, error), pose poseFunc, dst func(*Parts) **decad.Body) buildStep {
		return single(name, build, pose, all, dst)
	}
	steps := []buildStep{
		{"flags", func(r *run) (func(), error) {
			// Every pose builds its own prototypes, as a standalone build does, so its flags have the same history as
			// those of BuildParts for that pose.
			flags, err := r.flags()
			if err != nil {
				return nil, err
			}
			posed := make([][7]Flag, len(r.poses))
			for i, pr := range r.poses {
				if posed[i], err = pr.flags(); err != nil {
					return nil, err
				}
			}
			return func() {
				out.Flags = flags
				for i, f := range posed {
					extras[i].Flags = f
				}
			}, nil
		}},
		{"pivot blocks", func(r *run) (func(), error) {
			// The digit does not move a block, so a pose with the same placement files the main blocks. A pose with
			// another placement builds its own blocks, as a standalone build does.
			blocks, err := r.pivotBlocks()
			if err != nil {
				return nil, fmt.Errorf("sevensegment: pivot blocks: %w", err)
			}
			posed := make([][7][2]*decad.Body, len(r.poses))
			for i, pr := range r.poses {
				if sameFrame(r.p.Placement, pr.p.Placement) {
					posed[i] = blocks
				} else if posed[i], err = pr.pivotBlocks(); err != nil {
					return nil, fmt.Errorf("sevensegment: pivot blocks: %w", err)
				}
			}
			return func() {
				out.PivotBlocks = blocks
				for i, b := range posed {
					extras[i].PivotBlocks = b
				}
			}, nil
		}},
		one("front wall", func(r *run) (*decad.Body, error) { return r.wall(true) }, nil,
			func(p *Parts) **decad.Body { return &p.FrontWall }),
		one("rear wall", func(r *run) (*decad.Body, error) { return r.wall(false) }, nil,
			func(p *Parts) **decad.Body { return &p.RearWall }),
		one("guide plate", (*run).columnGuidePlate, nil, func(p *Parts) **decad.Body { return &p.GuidePlate }),
		one("cradle plate", (*run).cradlePlate, nil, func(p *Parts) **decad.Body { return &p.CradlePlate }),
		one("top plate", (*run).flagTopPlate, nil, func(p *Parts) **decad.Body { return &p.TopPlate }),
		one("shaft", (*run).shaft, spinTurn, func(p *Parts) **decad.Body { return &p.Shaft }),
		one("hub", (*run).hub, spinTurn, func(p *Parts) **decad.Body { return &p.Hub }),
		one("rotor", (*run).rotor, spinTurn, func(p *Parts) **decad.Body { return &p.Rotor }),
		one("housing", (*run).housing, nil, func(p *Parts) **decad.Body { return &p.Housing }),
		one("button", (*run).button, nil, func(p *Parts) **decad.Body { return &p.Button }),
		one("spring", func(r *run) (*decad.Body, error) { return r.spring(SpringRestLength) }, spinTurn,
			func(p *Parts) **decad.Body { return &p.Spring }),
	}
	for _, seg := range Segments() {
		steps = append(steps,
			one("cam "+seg.String(), func(r *run) (*decad.Body, error) { return r.cam(seg) }, spinTurn,
				func(p *Parts) **decad.Body { return &p.Cams[seg] }),
			one("slider "+seg.String(), func(r *run) (*decad.Body, error) { return r.flagSlider(seg) }, liftIfLit(seg),
				func(p *Parts) **decad.Body { return &p.Sliders[seg] }))
		if _, _, horizontal := segmentCentre(seg); horizontal {
			steps = append(steps, one("groove pin "+seg.String(),
				func(r *run) (*decad.Body, error) { return r.groovePinFor(seg) }, liftIfLit(seg),
				func(p *Parts) **decad.Body { return &p.GroovePins[seg] }))
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

// poseFunc says how the digit moves a body that has been built unposed: the motion for the run's digit and whether there
// is one. A nil poseFunc means the digit does not move the body.
type poseFunc func(r *run) (r3.Transform, bool, error)

// moves returns the motion pose gives a body for r's digit.
func (pose poseFunc) moves(r *run) (r3.Transform, bool, error) {
	if pose == nil {
		return r3.Transform{}, false, nil
	}
	return pose(r)
}

// sameFrame reports whether two placements are the same: both nil, or equal frames.
func sameFrame(a, b *r3.Frame) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// single makes a one-body step: it builds the body once, unposed, and checks it is a solid. It then makes the body of
// every extra pose in r.poses as a copy of it, moved by the pose and placed, and last poses and places the body itself
// for r's own pose. dst selects where a Parts files the body; all holds the main parts and then one Parts per extra
// pose. An extra pose whose body would equal the main body (the digit does not move it and the placement is the same)
// files the main body itself and makes no copy.
func single(name string, build func(*run) (*decad.Body, error), pose poseFunc, all []*Parts,
	dst func(*Parts) **decad.Body) buildStep {
	return buildStep{name, func(r *run) (func(), error) {
		body, err := build(r)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: %s: %w", name, err)
		}
		if body == nil || !body.IsSolid() {
			return nil, fmt.Errorf("sevensegment: %s is not a solid", name)
		}
		move, moved, err := pose.moves(r)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: pose %s: %w", name, err)
		}
		// Copies come first: the main pose consumes body.
		extra := make([]*decad.Body, len(r.poses)) // nil: the main body
		for i, pr := range r.poses {
			pmove, pmoved, err := pose.moves(pr)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: pose %s: %w", name, err)
			}
			if !moved && !pmoved && sameFrame(r.p.Placement, pr.p.Placement) {
				continue
			}
			// A body the digit moves is copied under the digit's motion and then placed, the two operations a
			// standalone build runs. A body it does not move is copied under the placement alone, and one that
			// stays where it was built is built again. A plain copy, or a copy that is then placed, would carry the
			// copy in its history, and the volume bound of the body would differ from a standalone build's in the
			// last bits.
			c := decad.Copy(body)
			placeAfter := true
			switch {
			case pmoved:
				c = c.Place(decad.CopyTransform{Transform: pmove})
			case pr.p.Placement != nil:
				lift, err := r3.FromFrame(*pr.p.Placement)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: place %s: %w", name, err)
				}
				c, placeAfter = c.Place(decad.CopyTransform{Transform: lift}), false
			default:
				again, err := build(pr)
				if err != nil {
					return nil, fmt.Errorf("sevensegment: %s: %w", name, err)
				}
				if again == nil || !again.IsSolid() {
					return nil, fmt.Errorf("sevensegment: %s is not a solid", name)
				}
				extra[i] = again
				continue
			}
			copied, err := c.Do(r.ctx)
			if err != nil {
				return nil, fmt.Errorf("sevensegment: copy %s: %w", name, err)
			}
			if !placeAfter {
				extra[i] = copied
				continue
			}
			if extra[i], err = placement.Place(r.ctx, copied, pr.p.Placement); err != nil {
				return nil, fmt.Errorf("sevensegment: place %s: %w", name, err)
			}
		}
		if moved {
			if body, err = decad.Transform(body, move).Do(r.ctx); err != nil {
				return nil, fmt.Errorf("sevensegment: pose %s: %w", name, err)
			}
		}
		placed, err := placement.Place(r.ctx, body, r.p.Placement)
		if err != nil {
			return nil, fmt.Errorf("sevensegment: place %s: %w", name, err)
		}
		return func() {
			*dst(all[0]) = placed
			for i, b := range extra {
				if b == nil {
					b = placed
				}
				*dst(all[i+1]) = b
			}
		}, nil
	}}
}

// spinTurn is the camshaft group's turn for r's digit.
func spinTurn(r *run) (r3.Transform, bool, error) { return r.camshaftTurn(), true, nil }

// liftIfLit raises a body by the slider travel when segment seg is lit for r's digit.
func liftIfLit(seg Segment) poseFunc {
	return func(r *run) (r3.Transform, bool, error) {
		if lit, _ := Lit(r.p.Digit, seg); !lit {
			return r3.Transform{}, false, nil
		}
		lift, err := r3.Translation(r3.Vec{Z: Travel})
		return lift, err == nil, err
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
