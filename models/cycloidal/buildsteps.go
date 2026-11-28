package cycloidal

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/lestrrat-3d/decad"
)

// maxStepWorkers caps how many build steps run at once. A drive has four or five steps, and four workers already
// reach the build's CPU bound on a CI runner.
const maxStepWorkers = 4

// buildStep is one independent part of a drive. build runs on a run of its own, with the step's context, and
// returns file, which the runner calls after every step has finished, serially and in step order, to store what the
// step made. build must reach the document, the context, the parameters and the dimensions only through its
// arguments, and must not write the Drive itself: only file does.
type buildStep struct {
	name  string
	build func(ctx context.Context, r *run) (file func(), err error)
}

// stepWorkers is the worker option Build passes to decad.RunSteps.
func stepWorkers() decad.RunStepsOption {
	return decad.WithStepWorkers(min(runtime.GOMAXPROCS(0), maxStepWorkers))
}

// stepResult is what one step leaves for runSteps. Only the step itself writes its entry. returned stays false when
// the step panicked or ended its goroutine, so it never wrote the other fields.
type stepResult struct {
	returned bool
	file     func()
	err      error
}

// driveSteps lists Build's steps in build order: one per rotor disc, then the cam, the housing and the output plate
// with its pins. It has no context, document or parameters in scope, so a step can only use its own run. It sizes
// out.Discs; a step's file func fills out.
func driveSteps(out *Drive, discCount int) []buildStep {
	out.Discs = make([]*decad.Body, discCount)
	var steps []buildStep
	for index := range discCount {
		steps = append(steps, buildStep{fmt.Sprintf("disc %d", index+1),
			func(ctx context.Context, r *run) (func(), error) {
				disc, err := r.chamferedDisc(ctx, index)
				if err != nil {
					return nil, err
				}
				return func() { out.Discs[index] = disc }, nil
			}})
	}
	return append(steps,
		buildStep{"cam", func(ctx context.Context, r *run) (func(), error) {
			cam, err := r.buildCam(ctx)
			if err != nil {
				return nil, err
			}
			cam, err = r.chamferedBy(ctx, cam, "cam", r.camOuterCircles())
			if err != nil {
				return nil, err
			}
			return func() { out.Cam = cam }, nil
		}},
		buildStep{"housing", func(ctx context.Context, r *run) (func(), error) {
			housing, err := r.buildHousing(ctx)
			if err != nil {
				return nil, err
			}
			housing, err = r.chamferedBy(ctx, housing, "housing", r.housingOuterCircles())
			if err != nil {
				return nil, err
			}
			return func() { out.Housing = housing }, nil
		}},
		buildStep{"output", func(ctx context.Context, r *run) (func(), error) {
			plate, pin, err := r.buildOutput(ctx)
			if err != nil {
				return nil, err
			}
			if plate, err = r.chamfered(ctx, plate, "output plate"); err != nil {
				return nil, err
			}
			// One pin is chamfered and repeated, so the M pins come from one chamfer and one pattern.
			if pin, err = r.chamfered(ctx, pin, "output pin"); err != nil {
				return nil, err
			}
			pins, err := r.patternPins(ctx, pin)
			if err != nil {
				return nil, err
			}
			return func() { out.OutputPlate, out.OutputPins = plate, pins }, nil
		}})
}

// runSteps runs steps through decad.RunSteps on r's document, each on a fork of r with a sketch world of its own and
// the step's context, files their results in step order and returns the first failure in step order. On an error it
// files nothing.
func (r *run) runSteps(ctx context.Context, steps []buildStep, opts ...decad.RunStepsOption) error {
	results := make([]stepResult, len(steps))
	list := make([]decad.Step, len(steps))
	for i, st := range steps {
		list[i] = decad.Step{Name: st.name, Build: func(sctx context.Context) error {
			sr, err := r.fork()
			if err != nil {
				results[i] = stepResult{returned: true, err: err}
				return err
			}
			file, err := st.build(sctx, sr)
			results[i] = stepResult{returned: true, file: file, err: err}
			return err
		}}
	}
	runErr := decad.RunSteps(ctx, r.doc, list, opts...)
	// Steps are independent, so the first failure in step order is the error a serial build returns. A step that
	// never returned (it panicked or ended its goroutine) is a failure at its place in the order: a serial build
	// would have stopped there, so a later step's error must not hide it.
	for _, res := range results {
		if !res.returned {
			if runErr == nil {
				runErr = errors.New("a step did not return")
			}
			return fmt.Errorf("cycloidal: build steps: %w", runErr)
		}
		if res.err != nil {
			return res.err
		}
	}
	if runErr != nil {
		return fmt.Errorf("cycloidal: build steps: %w", runErr)
	}
	for _, res := range results {
		if res.file != nil {
			res.file()
		}
	}
	return nil
}
