package escapement

import (
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-3d/decad"
)

type sharedBuild struct {
	doc *decad.Document
	m   *Movement
	err error
}

var (
	sharedOnce  sync.Once
	sharedValue sharedBuild
)

// sharedDefault builds the default movement once for the whole test binary.
func sharedDefault(t *testing.T) (*decad.Document, *Movement, error) {
	t.Helper()
	sharedOnce.Do(func() {
		start := time.Now()
		doc := NewDocument()
		m, err := Build(t.Context(), doc, DefaultParams())
		sharedValue = sharedBuild{doc, m, err}
		t.Logf("default build took %v", time.Since(start))
	})
	return sharedValue.doc, sharedValue.m, sharedValue.err
}
