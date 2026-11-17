package swisslever

import (
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// buildMovement builds the movement for p.
func buildMovement(t *testing.T, p Params) (*decad.Document, *Movement, error) {
	t.Helper()
	doc := NewDocument()
	m, err := Build(t.Context(), doc, p)
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
