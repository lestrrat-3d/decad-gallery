package planetarygearbox

// CI: isolated-tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCorners applies "Corners and roots" to every printed part of the default gearbox.
func TestCorners(t *testing.T) {
	cornersDefault.once.Do(func() {
		cornersDefault.box, cornersDefault.err = Build(t.Context(), NewDocument(), DefaultParams())
	})
	require.NoError(t, cornersDefault.err)
	requireCorners(t, cornersDefault.box, DefaultParams())
}

func TestCornersPlanet605Shaft608(t *testing.T) { requireCornerKit(t, "planet605-shaft608") }
func TestCornersPlanet608Shaft605(t *testing.T) { requireCornerKit(t, "planet608-shaft605") }
func TestCornersPlanet608Shaft608(t *testing.T) { requireCornerKit(t, "planet608-shaft608") }
func TestCornersEarCount0(t *testing.T)         { requireCornerEars(t, 0) }
func TestCornersEarCount1(t *testing.T)         { requireCornerEars(t, 1) }
func TestCornersEarCount6(t *testing.T)         { requireCornerEars(t, 6) }

func TestCornersTwoStages(t *testing.T) {
	p := DefaultParams()
	p.Stages = 2
	requireCornerVariant(t, p)
}
