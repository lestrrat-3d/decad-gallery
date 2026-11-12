package escapement

import (
	"testing"

	"github.com/lestrrat-3d/decad"
)

// SharedDefault returns the default movement every test shares, built once.
func SharedDefault(t *testing.T) (*decad.Document, *Movement, error) { return sharedDefault(t) }
