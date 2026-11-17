package swisslever

import (
	"testing"

	"github.com/lestrrat-3d/decad"
)

// SharedAt returns the default movement posed at phase, built once per test binary. It leaves out only the operations
// decad refuses today, so the external tests run unchanged once decad accepts them.
func SharedAt(t *testing.T, phase float64) (*decad.Document, *Movement) { return sharedAt(t, phase) }

// BuildForTest builds the movement for p as Build does, leaving out only the operations decad refuses today.
func BuildForTest(t *testing.T, p Params) (*decad.Document, *Movement, error) {
	return buildMovement(t, p)
}
