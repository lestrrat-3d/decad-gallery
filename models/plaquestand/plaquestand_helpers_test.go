package plaquestand_test

import (
	"context"
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}

// builtStand is one parameter set's build, made once and shared by every test that asks for it.
type builtStand struct {
	once  sync.Once
	doc   *decad.Document
	model *plaquestand.Model
	err   error
}

var sharedStands sync.Map // plaquestand.Params -> *builtStand

// sharedBuild returns p's build, made on first use in a document of its own. Tests must treat the document and
// its bodies as read-only: none may add, cut, move or place anything in it. The build does not use the first
// caller's context, which ends with that caller's test.
func sharedBuild(p plaquestand.Params) (*decad.Document, *plaquestand.Model, error) {
	v, _ := sharedStands.LoadOrStore(p, &builtStand{})
	b := v.(*builtStand)
	b.once.Do(func() {
		b.doc = decad.New()
		b.model, b.err = plaquestand.Build(context.Background(), b.doc, p)
	})
	return b.doc, b.model, b.err
}

// requireBuilt returns p's shared build and checks what every case shares: eight bodies in their containers, and
// every sketch fully constrained. The returned document and model are read-only.
func requireBuilt(t *testing.T, p plaquestand.Params) (*decad.Document, *plaquestand.Model) {
	t.Helper()
	doc, m, err := sharedBuild(p)
	require.NoError(t, err)
	require.Equal(t, 8, len(doc.Bodies()))
	for _, b := range []*decad.Body{m.Brackets[0], m.Brackets[1], m.Rail, m.Plaque, m.Bolts[0].Body, m.Bolts[1].Body,
		m.Nuts[0].Body, m.Nuts[1].Body} {
		require.True(t, b.IsSolid())
	}
	require.Len(t, m.Sketches, 12)
	for _, s := range m.Sketches {
		requireFullyConstrained(t, s)
	}

	byName := map[string][]*decad.Body{}
	for _, child := range doc.Root().Children() {
		byName[child.Name()] = child.Bodies()
	}
	require.Len(t, byName, 5)
	require.Equal(t, []*decad.Body{m.Brackets[0], m.Brackets[1]}, byName[plaquestand.BracketsContainerName])
	require.Equal(t, []*decad.Body{m.Rail}, byName[plaquestand.RailContainerName])
	require.Equal(t, []*decad.Body{m.Bolts[0].Body, m.Bolts[1].Body}, byName[plaquestand.BoltsContainerName])
	require.Equal(t, []*decad.Body{m.Nuts[0].Body, m.Nuts[1].Body}, byName[plaquestand.NutsContainerName])
	require.Equal(t, []*decad.Body{m.Plaque}, byName[plaquestand.PlaqueContainerName])
	require.NoError(t, doc.VerifyContainers().Err())
	return doc, m
}

// shoelace returns the absolute area of the closed loop in (Y, Z).
func shoelace(loop []r3.Vec) float64 {
	sum := 0.0
	for i, a := range loop {
		b := loop[(i+1)%len(loop)]
		sum += a.Y*b.Z - b.Y*a.Z
	}
	return math.Abs(sum) / 2
}

func bounds(t *testing.T, b *decad.Body) (decad.Box, float64) {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err)
	return box, box.Bound.Base() + 1e-6
}
