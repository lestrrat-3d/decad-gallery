package bevelgear_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/stretchr/testify/require"
)

func TestBuildRejectsUnverifiedFullPatternCaseBeforeAddingBodies(t *testing.T) {
	p := bevelgear.DefaultParams()
	doc := decad.New()
	_, err := bevelgear.Build(t.Context(), doc, p)
	require.ErrorContains(t, err, "verified 8/8")
	require.Empty(t, doc.Bodies())
}
