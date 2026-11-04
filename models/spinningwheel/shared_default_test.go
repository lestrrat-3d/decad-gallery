package spinningwheel_test

import (
	"context"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/stretchr/testify/require"
)

var sharedDefault struct {
	once sync.Once
	a    *spinningwheel.Assembly
	err  error
}

// sharedDefaultAssembly builds the default wheel once per test binary. A default Build costs about a
// minute, and the tests that use this helper only read the assembly. A test that adds a body to the
// document must remove it again before it returns, and the tests do not run in parallel, so no test
// sees another test's body.
func sharedDefaultAssembly(t *testing.T) *spinningwheel.Assembly {
	t.Helper()
	sharedDefault.once.Do(func() {
		// The first caller's context ends with its test, but later tests reuse the result.
		ctx := context.WithoutCancel(t.Context())
		sharedDefault.a, sharedDefault.err = spinningwheel.Build(ctx, spinningwheel.NewDocument(), spinningwheel.DefaultParams())
	})
	require.NoError(t, sharedDefault.err)
	return sharedDefault.a
}
