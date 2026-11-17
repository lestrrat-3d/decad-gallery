package swisslever

import (
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// chamferRefusal is the decad refusal the balance wheel's rim and window chamfer gets until decad chamfers the edges
// the wheel's boss leaves as faceted curves.
const chamferRefusal = "cannot select a decad.FacetedCurve edge"

// switches records whether decad refuses the balance wheel's chamfer today: it holds the refusal text, or is empty
// when decad accepts it. Once it does, the movement is built exactly as Build builds it.
type switches struct {
	chamfer string
}

var (
	switchOnce  sync.Once
	switchValue switches
)

// movementSwitches probes the lever and the balance wheel once. The lever must build; the balance wheel may fail only
// at its edge chamfer with the one known refusal.
func movementSwitches(t *testing.T) switches {
	t.Helper()
	switchOnce.Do(func() {
		_, err := partBuilder(t).buildLever()
		require.NoError(t, err, "the lever builds in full")
		_, err = partBuilder(t).buildBalanceWheel()
		if err != nil {
			require.ErrorContains(t, err, "balance wheel edges", "only the balance wheel's chamfer may fail")
			require.ErrorContains(t, err, chamferRefusal)
			switchValue.chamfer = err.Error()
		}
	})
	return switchValue
}

// configure returns the builder hook that leaves out the chamfer decad refuses; nil when it refuses none.
func (s switches) configure() func(*builder) {
	if s.chamfer == "" {
		return nil
	}
	return func(b *builder) { b.skipBalanceChamfer = true }
}

// buildMovement builds the movement for p as Build does, leaving out only what decad refuses.
func buildMovement(t *testing.T, p Params) (*decad.Document, *Movement, error) {
	t.Helper()
	doc := NewDocument()
	m, err := buildWith(t.Context(), doc, p, movementSwitches(t).configure())
	return doc, m, err
}

type sharedBuild struct {
	doc *decad.Document
	m   *Movement
	err error
}

var (
	sharedMu    sync.Mutex
	sharedCache = map[float64]sharedBuild{}
)

// sharedAt builds the default movement posed at phase once for the whole test binary.
func sharedAt(t *testing.T, phase float64) (*decad.Document, *Movement) {
	t.Helper()
	sharedMu.Lock()
	defer sharedMu.Unlock()
	v, ok := sharedCache[phase]
	if !ok {
		start := time.Now()
		p := DefaultParams()
		p.Phase = phase
		doc, m, err := buildMovement(t, p)
		v = sharedBuild{doc, m, err}
		sharedCache[phase] = v
		t.Logf("build at phase %v took %v", phase, time.Since(start))
	}
	require.NoError(t, v.err)
	return v.doc, v.m
}

// sharedDefault returns the default movement.
func sharedDefault(t *testing.T) (*decad.Document, *Movement) {
	t.Helper()
	return sharedAt(t, DefaultParams().Phase)
}
