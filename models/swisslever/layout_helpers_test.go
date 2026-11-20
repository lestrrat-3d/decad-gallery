package swisslever_test

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/swisslever"
	"github.com/stretchr/testify/require"
)

func near(t *testing.T, want, got float64, name string) {
	t.Helper()
	require.InDelta(t, want, got, 1e-4, name)
}

func nearPt(t *testing.T, want [2]float64, got swisslever.Pt, name string) {
	t.Helper()
	near(t, want[0], got.X, name+".X")
	near(t, want[1], got.Y, name+".Y")
}
