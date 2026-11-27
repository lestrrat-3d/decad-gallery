package sevensegment

// CI: isolated-tests

import (
	"context"
	"errors"
	goreflect "reflect"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestSevenSegmentRunFieldsAreStepLocal(t *testing.T) {
	t.Parallel()
	// A step builds on its own run. Adding a field here must be reviewed against that rule, and the step lists must
	// not gain a context, document or parameters argument that a closure could capture.
	want := []struct {
		name string
		typ  goreflect.Type
	}{
		{"ctx", goreflect.TypeFor[context.Context]()},
		{"doc", goreflect.TypeFor[*decad.Document]()},
		{"p", goreflect.TypeFor[Params]()},
		{"dm", goreflect.TypeFor[Dimensions]()},
		{"sketches", goreflect.TypeFor[[]*sketch.Sketch]()},
	}
	typ := goreflect.TypeFor[run]()
	require.Equal(t, len(want), typ.NumField())
	for i, w := range want {
		require.Equal(t, w.name, typ.Field(i).Name)
		require.Equal(t, w.typ, typ.Field(i).Type, w.name)
	}
	require.Equal(t, goreflect.TypeOf(func(*Parts) []buildStep { return nil }), goreflect.TypeOf(partSteps))
	require.Equal(t, goreflect.TypeOf(func(*Model) []buildStep { return nil }), goreflect.TypeOf(hardwareSteps))
}

func TestSevenSegmentStepsDeterministic(t *testing.T) {
	t.Parallel()
	p := Params{Digit: 2}
	var prints [2][]stepPrint
	var shapes [2][]string
	for i, workers := range []int{1, 4} {
		steps, m := lightSteps()
		require.Len(t, steps, len(lightStepNames))
		doc := NewDocument()
		_, sketches, err := runSteps(t.Context(), doc, p, steps, decad.WithStepWorkers(workers))
		require.NoError(t, err, "%d workers", workers)
		require.NotEmpty(t, sketches)
		shapes[i] = sketchShapes(sketches)
		prints[i] = stepPrints(t.Context(), t, doc)
		require.NotEmpty(t, prints[i])
		// Every filed body sits in the document in step order.
		last := -1
		for _, b := range []*decad.Body{m.PivotBlocks[0][0], m.GuidePlate, m.Shaft, m.Hub, m.Spring, m.Cams[SegA],
			m.Sliders[SegA], m.GroovePins[SegA], m.Cams[SegD], m.Bearings[0].Body, m.Bolts[0].Body, m.Nuts[0].Body} {
			index := -1
			for j, d := range doc.Bodies() {
				if d == b {
					index = j
				}
			}
			require.Greater(t, index, last, "%d workers: body order", workers)
			last = index
		}
	}
	require.Empty(t, firstPrintDiff(prints[0], prints[1]))
	require.Equal(t, shapes[0], shapes[1], "sketch counts and order")
}

func TestSevenSegmentStepsFirstError(t *testing.T) {
	t.Parallel()
	errOne, errTwo := errors.New("step one failed"), errors.New("step two failed")
	filed := false
	ok := buildStep{"ok", func(r *run) (func(), error) {
		_, err := r.box(wallFrame(true), 0, 1, 0, 1, 1)
		return func() { filed = true }, err
	}}
	// Step one fails only after step two has failed, so the failures finish in the reverse of step order. Three
	// workers run all three steps at once.
	twoFailed := make(chan struct{})
	one := buildStep{"one", func(*run) (func(), error) {
		select {
		case <-twoFailed:
		case <-time.After(time.Minute):
			return nil, errors.New("step two never failed")
		}
		return nil, errOne
	}}
	two := buildStep{"two", func(*run) (func(), error) {
		defer close(twoFailed)
		return nil, errTwo
	}}
	_, _, err := runSteps(t.Context(), NewDocument(), Params{}, []buildStep{ok, one, two}, decad.WithStepWorkers(3))
	require.Same(t, errOne, err)
	require.False(t, filed, "a failed run files nothing")

	outer := t.Context()
	scope := buildStep{"scope", func(r *run) (func(), error) {
		stray := *r
		stray.ctx = outer
		_, err := stray.box(wallFrame(true), 0, 1, 0, 1, 1)
		return nil, err
	}}
	_, _, err = runSteps(t.Context(), NewDocument(), Params{}, []buildStep{ok, scope}, decad.WithStepWorkers(2))
	require.ErrorIs(t, err, decad.ErrStepScope)

	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err = runSteps(canceled, NewDocument(), Params{}, []buildStep{ok})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSevenSegmentStepsPanicBeforeError(t *testing.T) {
	t.Parallel()
	errLater := errors.New("later step failed")
	boom := buildStep{"boom", func(*run) (func(), error) { panic("boom in step") }}
	// The failing step finishes while the panicking one is still running, so its error is known first.
	laterFailed := make(chan struct{})
	slowBoom := buildStep{"slow boom", func(*run) (func(), error) {
		<-laterFailed
		panic("boom in step")
	}}
	later := buildStep{"later", func(*run) (func(), error) {
		defer close(laterFailed)
		return nil, errLater
	}}
	for name, first := range map[string]buildStep{"immediate": boom, "after later": slowBoom} {
		steps := []buildStep{first, later}
		if name == "immediate" {
			steps = []buildStep{first, {"later", func(*run) (func(), error) { return nil, errLater }}}
		}
		_, _, err := runSteps(t.Context(), NewDocument(), Params{}, steps, decad.WithStepWorkers(2))
		require.Error(t, err, name)
		require.NotSame(t, errLater, err, name) // the panic must not be hidden by the later error
		require.ErrorContains(t, err, "sevensegment: build steps:", name)
		require.ErrorContains(t, err, "boom in step", name)
	}
}

func TestSevenSegmentStepsRunFault(t *testing.T) {
	t.Parallel()
	outer := t.Context()
	// The step swallows the refusal, so it returns no error but the run is marked failed.
	swallow := buildStep{"swallow", func(r *run) (func(), error) {
		stray := *r
		stray.ctx = outer
		_, _ = stray.box(wallFrame(true), 0, 1, 0, 1, 1)
		return nil, nil
	}}
	_, _, err := runSteps(t.Context(), NewDocument(), Params{}, []buildStep{swallow}, decad.WithStepWorkers(1))
	require.ErrorContains(t, err, "sevensegment: build steps:")
	require.ErrorIs(t, err, decad.ErrStepScope)
}

func TestSevenSegmentStepsRace(t *testing.T) {
	t.Parallel()
	// CI runs this test with -race.
	steps, m := lightSteps()
	doc := NewDocument()
	_, _, err := runSteps(t.Context(), doc, Params{Digit: 2}, steps, stepWorkers())
	require.NoError(t, err)
	for _, b := range []*decad.Body{m.PivotBlocks[6][1], m.GuidePlate, m.Shaft, m.Hub, m.Spring, m.Cams[SegA], m.Cams[SegD],
		m.Sliders[SegA], m.GroovePins[SegA], m.Bearings[0].Body, m.Bolts[0].Body, m.Nuts[0].Body} {
		require.NotNil(t, b)
		require.True(t, b.IsSolid())
	}
	require.Len(t, doc.Bodies(), 14+len(lightStepNames)-1)
}
