package escapement_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/escapement"
	"github.com/stretchr/testify/require"
)

// TestSetJoinsEveryPart checks that the parts set holds every body and that every body but the back plate is joined to
// another, so the derived explode can place it.
func TestSetJoinsEveryPart(t *testing.T) {
	b := defaultMovement(t)
	set, err := b.m.Set(b.doc, escapement.DefaultParams())
	require.NoError(t, err)
	require.Len(t, set.Parts(), len(expectedNames))

	joined := map[string]bool{}
	for _, j := range set.Joints() {
		joined[j.A], joined[j.B] = true, true
	}
	for _, e := range expectedNames {
		require.True(t, joined[e.name], "%s is in a joint", e.name)
	}
}
