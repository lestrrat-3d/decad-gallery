package swisslever

import (
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// bossRefusal is the decad refusal the balance wheel's hub boss gets until decad fillets a boss root that crosses
// the wheel's window arcs.
const bossRefusal = "the boss's root circle is not an edge of the result"

// switches records which of the two blocked operations decad refuses today. Each holds the refusal text, or is empty
// when decad accepts the operation. Once decad accepts both, the movement is built exactly as Build builds it.
type switches struct {
	dartRoot, boss string
}

var (
	switchOnce  sync.Once
	switchValue switches
)

// movementSwitches probes the lever and the balance wheel once. It tolerates only the two known refusals.
func movementSwitches(t *testing.T) switches {
	t.Helper()
	switchOnce.Do(func() {
		_, err := partBuilder(t).buildLever()
		if err != nil {
			require.ErrorContains(t, err, "dart root fillet", "only the dart root fillet may fail")
			require.ErrorContains(t, err, dartRootRefusal)
			switchValue.dartRoot = err.Error()
		}
		_, err = partBuilder(t).buildBalanceWheel()
		if err != nil {
			require.ErrorContains(t, err, "balance boss", "only the balance boss may fail")
			require.ErrorContains(t, err, bossRefusal)
			switchValue.boss = err.Error()
		}
	})
	return switchValue
}

// configure returns the builder hook that leaves out each operation decad refuses; nil when it refuses none.
func (s switches) configure() func(*builder) {
	if s.dartRoot == "" && s.boss == "" {
		return nil
	}
	return func(b *builder) {
		b.skipDartRoot = s.dartRoot != ""
		b.skipBalanceBoss = s.boss != ""
	}
}

// hasBoss reports whether the built balance wheel carries its hub boss.
func (s switches) hasBoss() bool { return s.boss == "" }

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
