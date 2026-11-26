package planetarygearbox

import (
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/stretchr/testify/require"
)

// sharedDefault holds the one default gearbox build that the tests of both test packages read. The tests must not
// change it or add bodies to its document.
var sharedDefault struct {
	once sync.Once
	doc  *decad.Document
	box  Gearbox
	err  error
}

// SharedDefaultBuild builds the default gearbox once per test binary and returns that build on every call. It is
// exported for the planetarygearbox_test package, which compiles into the same binary.
func SharedDefaultBuild(t *testing.T) (*decad.Document, Gearbox) {
	t.Helper()
	sharedDefault.once.Do(func() {
		sharedDefault.doc = NewDocument()
		sharedDefault.box, sharedDefault.err = Build(t.Context(), sharedDefault.doc, DefaultParams())
	})
	require.NoError(t, sharedDefault.err)
	return sharedDefault.doc, sharedDefault.box
}
