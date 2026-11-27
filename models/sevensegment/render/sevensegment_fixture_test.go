package render

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var sevenSegmentFixture struct {
	once  sync.Once
	parts partSet
	err   error
}

// sevenSegmentTestSet shares the immutable default meshes within a test process.
func sevenSegmentTestSet(t *testing.T) partSet {
	t.Helper()
	sevenSegmentFixture.once.Do(func() {
		sevenSegmentFixture.parts, sevenSegmentFixture.err = buildSevenSegmentSet(t.Context())
	})
	require.NoError(t, sevenSegmentFixture.err)
	return sevenSegmentFixture.parts
}
