package swisslever

import (
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"

	"github.com/stretchr/testify/require"
)

// plainLever builds the lever once.
var (
	leverOnce sync.Once
	leverB    *builder
	leverBody *decad.Body
	leverErr  error
)

func plainLever(t *testing.T) (*builder, *decad.Body) {
	t.Helper()
	leverOnce.Do(func() {
		leverB = partBuilder(t)
		leverBody, leverErr = leverB.buildLever()
	})
	require.NoError(t, leverErr)
	return leverB, leverBody
}
