package cycloidal

// CI: isolated-tests

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

func TestCycloidalRunFieldsAreStepLocal(t *testing.T) {
	t.Parallel()
	// Steps run together on forks of one run. A new field of run is shared by every step unless fork replaces it, so
	// adding one needs this list, and the decision whether fork must copy it, reviewed.
	var got []string
	for f := range reflect.TypeFor[run]().Fields() {
		got = append(got, f.Name+" "+f.Type.String())
	}
	require.Equal(t, []string{
		"doc *decad.Document", "world *sketch.World", "p cycloidal.Params", "d cycloidal.dimensions",
		"frame r3.Frame", "vals cycloidal.paramValues",
	}, got)
	// driveSteps has no context, document or parameters in scope: its only inputs are the Drive it files into and
	// the disc count.
	require.Equal(t, reflect.TypeFor[func(*Drive, int) []buildStep](), reflect.TypeOf(driveSteps))

	// Every step gets a fork with a sketch world of its own and the same values.
	r := stepRun(t)
	seen := map[*run]struct{}{}
	worlds := map[any]struct{}{}
	var mu sync.Mutex
	step := func(name string) buildStep {
		return buildStep{name, func(_ context.Context, sr *run) (func(), error) {
			mu.Lock()
			defer mu.Unlock()
			seen[sr], worlds[sr.world] = struct{}{}, struct{}{}
			require.Equal(t, r.p, sr.p)
			require.Equal(t, r.vals, sr.vals)
			return nil, nil
		}}
	}
	require.NoError(t, r.runSteps(t.Context(), []buildStep{step("a"), step("b"), step("c")}, decad.WithStepWorkers(3)))
	require.Len(t, seen, 3)
	require.Len(t, worlds, 3)
	require.NotContains(t, worlds, r.world)
}

func TestCycloidalStepsFilesInStepOrder(t *testing.T) {
	t.Parallel()
	// The first step ends after the second, yet files first. A failed run files nothing.
	r := stepRun(t)
	second := make(chan struct{})
	var filed []string
	steps := []buildStep{
		{"first", func(context.Context, *run) (func(), error) {
			<-second
			return func() { filed = append(filed, "first") }, nil
		}},
		{"second", func(context.Context, *run) (func(), error) {
			defer close(second)
			return func() { filed = append(filed, "second") }, nil
		}},
	}
	require.NoError(t, r.runSteps(t.Context(), steps, decad.WithStepWorkers(2)))
	require.Equal(t, []string{"first", "second"}, filed)

	filed = nil
	boom := errors.New("boom")
	steps = append(steps, buildStep{"third", func(context.Context, *run) (func(), error) { return nil, boom }})
	second = make(chan struct{})
	require.ErrorIs(t, r.runSteps(t.Context(), steps, decad.WithStepWorkers(3)), boom)
	require.Empty(t, filed)
}

func TestCycloidalStepsFirstError(t *testing.T) {
	t.Parallel()
	// Steps 1 and 2 fail. Step 2 fails first in time, but a serial build stops at step 1, so its error is the one
	// returned, with the text a serial build gives (no step prefix).
	r := stepRun(t)
	first, second := errors.New("cycloidal: housing: first failure"), errors.New("cycloidal: output: second failure")
	// With one worker the steps cannot wait for each other, so only three workers force the order in time.
	for _, workers := range []int{1, 3} {
		late := make(chan struct{})
		steps := []buildStep{
			noStep("ok"),
			{"failing early in order", func(context.Context, *run) (func(), error) {
				if workers > 1 {
					<-late
				}
				return nil, first
			}},
			{"failing early in time", func(context.Context, *run) (func(), error) {
				if workers > 1 {
					defer close(late)
				}
				return nil, second
			}},
		}
		err := r.runSteps(t.Context(), steps, decad.WithStepWorkers(workers))
		require.Same(t, first, err, "workers %d", workers)
	}
}

func TestCycloidalStepsPanicBeforeError(t *testing.T) {
	t.Parallel()
	// Step 0 panics and step 1 returns an ordinary error. A serial build would have stopped at step 0, so the
	// error is the run's, which names the panic, and not step 1's own error.
	r := stepRun(t)
	later := errors.New("cycloidal: later step failed")
	steps := []buildStep{
		{"panics", func(context.Context, *run) (func(), error) { panic("step blew up") }},
		{"fails", func(context.Context, *run) (func(), error) { return nil, later }},
	}
	for _, workers := range []int{1, 2} {
		err := r.runSteps(t.Context(), steps, decad.WithStepWorkers(workers))
		require.Error(t, err, "workers %d", workers)
		require.NotSame(t, later, err, "workers %d", workers)
		require.ErrorContains(t, err, "cycloidal: build steps", "workers %d", workers)
		require.ErrorContains(t, err, "step blew up", "workers %d", workers)
	}
}

func TestCycloidalParamReads(t *testing.T) {
	t.Parallel()
	// begin reads the four document parameters the steps use, because decad refuses doc.Param while steps run.
	r := stepRun(t)
	for name, got := range map[string]any{
		"disc_thickness":         r.vals.discThickness,
		"chamfer_size":           r.vals.chamferSize,
		"output_plate_thickness": r.vals.plateThickness,
		"output_plate_height":    r.vals.plateHeight,
	} {
		want, err := r.doc.Param(name)
		require.NoError(t, err, name)
		require.Equal(t, want, got, name)
	}
	// A step that reads a parameter itself fails with the scope error, and runSteps reports it as the step's own.
	steps := []buildStep{{"reads", func(_ context.Context, sr *run) (func(), error) {
		_, err := sr.doc.Param("disc_thickness")
		return nil, err
	}}}
	require.ErrorIs(t, r.runSteps(t.Context(), steps), decad.ErrStepScope)
}

func TestCycloidalStepsDeterministic(t *testing.T) {
	t.Parallel()
	// The solid-cam drive built with one worker and with four must match body for body: mesh, volume, place in
	// doc.Bodies() and container.
	p := DefaultParams()
	p.InputShaftDiameter = 0
	var prints [2][]builtPrint
	for i, workers := range []int{1, 4} {
		doc := stepDocument()
		drive, err := build(t.Context(), doc, p, decad.WithStepWorkers(workers))
		require.NoError(t, err, "workers %d", workers)
		prints[i] = builtPrints(t, doc, drive)
	}
	require.NotEmpty(t, prints[0])
	require.Empty(t, firstBuiltDiff(prints[0], prints[1]))
}
