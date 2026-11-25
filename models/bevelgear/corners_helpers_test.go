package bevelgear_test

import (
	"context"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

// rootCone is the default gear's root cone about the shaft: the radius at axial position x, in mm. Both members' roots
// lie on it, from 10.14 at x 11.36 to 14.60 at x 16.36.
func rootCone(x float64) float64 { return 10.14 + 0.892*(x-11.36) }

// sharedMember is one bevel member built once for the whole test binary, by whichever test asks first. Tests must
// only read the body.
type sharedMember struct {
	once sync.Once
	body *decad.Body
	err  error
}

var defaultStraight, defaultSpiral sharedMember

func (s *sharedMember) get(t *testing.T, doc func() *decad.Document, p bevelgear.Params) *decad.Body {
	t.Helper()
	s.once.Do(func() { s.body, s.err = bevelgear.Build(context.Background(), doc(), p) })
	require.NoError(t, s.err)
	return s.body
}

// defaultStraightMember is the default bored 31/31 straight member, built in a plain document.
func defaultStraightMember(t *testing.T) *decad.Body {
	t.Helper()
	return defaultStraight.get(t, func() *decad.Document { return decad.New() }, bevelgear.DefaultParams())
}

// defaultSpiralMember is the default bored 35° spiral member, built in a chain-tolerance document.
func defaultSpiralMember(t *testing.T) *decad.Body {
	t.Helper()
	return defaultSpiral.get(t, chainDocument, spiralDefaults())
}
