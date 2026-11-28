package screwgear_test

import (
	"context"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/screwgear"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// The tests that only read a mechanism (volumes, bounds, sections, faces, meshes) share one build per parameter set,
// built once per test binary. A test that cuts, places or otherwise changes the document builds its own with build.

type sharedMechanism struct {
	once sync.Once
	m    *screwgear.Mechanism
	doc  *decad.Document
}

var sharedMechanisms sync.Map // key string -> *sharedMechanism

// sharedBuild returns the mechanism for p, built and checked once under key. Callers must not change its bodies.
// The build runs on a context that outlives the test that triggers it.
func sharedBuild(t *testing.T, key string, p screwgear.Params) (*screwgear.Mechanism, *decad.Document) {
	t.Helper()
	v, _ := sharedMechanisms.LoadOrStore(key, &sharedMechanism{})
	e := v.(*sharedMechanism)
	e.once.Do(func() {
		doc := screwgear.NewDocument()
		m, err := screwgear.Build(context.WithoutCancel(t.Context()), doc, p)
		require.NoError(t, err)
		checkBuild(t, doc, m)
		e.m, e.doc = m, doc
	})
	require.NotNil(t, e.m, "the shared %s build failed in the test that built it", key)
	return e.m, e.doc
}

func sharedDefault(t *testing.T) (*screwgear.Mechanism, *decad.Document) {
	t.Helper()
	return sharedBuild(t, "default", screwgear.DefaultParams())
}

func sharedShort(t *testing.T) (*screwgear.Mechanism, *decad.Document) {
	t.Helper()
	return sharedBuild(t, "short", shortParams())
}

// placedFrame is the placement the placed build uses: U = +Y and V = +Z, so the normal is world X.
func placedFrame(t *testing.T) r3.Frame {
	t.Helper()
	frame, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	return frame
}

func placedParams(t *testing.T) screwgear.Params {
	t.Helper()
	frame := placedFrame(t)
	p := shortParams()
	p.Placement = &frame
	return p
}

func sharedPlaced(t *testing.T) (*screwgear.Mechanism, *decad.Document) {
	t.Helper()
	return sharedBuild(t, "placed", placedParams(t))
}
