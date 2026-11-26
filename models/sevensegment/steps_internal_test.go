package sevensegment

// CI: isolated-tests

import (
	"context"
	"errors"
	goreflect "reflect"
	"testing"

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
	for i, workers := range []int{1, 4} {
		steps, m := lightSteps()
		require.Len(t, steps, len(lightStepNames))
		doc := NewDocument()
		_, sketches, err := runSteps(t.Context(), doc, p, steps, decad.WithStepWorkers(workers))
		require.NoError(t, err, "%d workers", workers)
		require.NotEmpty(t, sketches)
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
}

func TestSevenSegmentStepsFirstError(t *testing.T) {
	t.Parallel()
	errOne, errTwo := errors.New("step one failed"), errors.New("step two failed")
	filed := false
	ok := buildStep{"ok", func(r *run) (func(), error) {
		_, err := r.box(wallFrame(true), 0, 1, 0, 1, 1)
		return func() { filed = true }, err
	}}
	fail := func(name string, err error) buildStep {
		return buildStep{name, func(*run) (func(), error) { return nil, err }}
	}
	_, _, err := runSteps(t.Context(), NewDocument(), Params{}, []buildStep{ok, fail("one", errOne), fail("two", errTwo)},
		decad.WithStepWorkers(2))
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
