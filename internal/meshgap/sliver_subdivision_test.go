package meshgap_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestClearanceNearSliver(t *testing.T) {
	cover := []r3.Vec{{X: -6.375360285807281, Y: .13534995811252873, Z: 29.53},
		{X: -6.376796875, Y: 7.809323881781584e-16, Z: 29.53},
		{X: -5.929469333022388, Y: 7.261505639328249e-16, Z: 30}}
	seam := []r3.Vec{{X: -6.4, Y: 7.83773951454305e-16, Z: 30},
		{X: -6.4, Y: 1.7797899989118203e-16, Z: 30},
		{X: -6.4, Y: 1.876166468478681e-16, Z: 29.93}}
	a, err := meshgap.New(cover, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	b, err := meshgap.New(seam, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	clear, err := a.ExceedsClearance(b, .06)
	require.NoError(t, err)
	require.True(t, clear, "the actual cover and seam triangles are separated")
	lower, clear, err := a.Exceeds(b, .06)
	require.NoError(t, err)
	require.True(t, clear)
	require.Greater(t, lower, .06)
	clear, err = b.ExceedsClearance(a, .06)
	require.NoError(t, err)
	require.True(t, clear)
	lower, clear, err = b.Exceeds(a, .06)
	require.NoError(t, err)
	require.True(t, clear)
	require.Greater(t, lower, .06)
	clear, err = a.ExceedsClearance(b, .5)
	require.NoError(t, err)
	require.False(t, clear, "a larger clearance must not be accepted")
}

func TestClearanceNearSliverContact(t *testing.T) {
	a, err := meshgap.New([]r3.Vec{{}, {X: 1}, {Y: 1}}, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	b, err := meshgap.New([]r3.Vec{{X: .2, Y: .2, Z: -1}, {X: .2, Y: .2, Z: 1},
		{X: .2000000000000001, Y: .2}}, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	clear, err := a.ExceedsClearance(b, 0)
	require.NoError(t, err)
	require.False(t, clear)
	_, clear, err = a.Exceeds(b, 0)
	require.NoError(t, err)
	require.False(t, clear)
}

func TestClearanceNearOverlappingSliverBoxes(t *testing.T) {
	cover := []r3.Vec{{X: -6.662265625, Y: 8.158922272725679e-16, Z: 29.53},
		{X: -6.193517049624522, Y: -.1314893959957203, Z: 30},
		{X: -6.194912663246269, Y: 7.586579964041938e-16, Z: 30}}
	seam := []r3.Vec{{X: -6.4, Y: 7.83773951454305e-16, Z: 30},
		{X: -6.4, Y: 1.7797899989118203e-16, Z: 30},
		{X: -6.4, Y: 1.876166468478681e-16, Z: 29.93}}
	a, err := meshgap.New(cover, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	b, err := meshgap.New(seam, [][3]int{{0, 1, 2}}, 0)
	require.NoError(t, err)
	lower, clear, err := a.Exceeds(b, .06116)
	require.NoError(t, err)
	require.True(t, clear)
	require.Greater(t, lower, .06116)
	clear, err = a.ExceedsClearance(b, .06116)
	require.NoError(t, err)
	require.True(t, clear)
}
