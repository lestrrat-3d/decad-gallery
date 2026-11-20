package clamparm_test

// CI: isolated-tests

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/clamparm"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestClampArmSwivel(t *testing.T) {
	body, sketches, err := clamparm.BuildSwivel(t.Context(), clamparm.NewDocument(), clamparm.DefaultParams())
	require.NoError(t, err)

	t.Run("solid, constrained, bounds", func(t *testing.T) {
		require.True(t, body.IsSolid())
		// The blank's two sketches, the J1 tongue tool, the J1 hole, the fork slot, the J2 hole and the J2 pocket.
		require.Len(t, sketches, 7)
		requireFullyConstrained(t, sketches)
		requireBounds(t, body, r3.NewVec(-10, -10, 0), r3.NewVec(80, 10, 20))
	})

	t.Run("tongue is 8 thick over the full width", func(t *testing.T) {
		requireLocated(t, body, true, r3.NewVec(10, 5, 7.5), r3.NewVec(10, -9.5, 4), r3.NewVec(20, 9.5, 7.5))
		requireLocated(t, body, false, r3.NewVec(10, 5, 8.5), r3.NewVec(10, -9.5, 12), r3.NewVec(20, 0, 15))
	})

	t.Run("shoulder is rounded with radius 4", func(t *testing.T) {
		// The round's centre is (26, 12): a point inside it is empty, one in the corner beyond it is material.
		requireLocated(t, body, false, r3.NewVec(27.5, 5, 9))
		requireLocated(t, body, true, r3.NewVec(29.5, 5, 8.5))
		requireLocated(t, body, true, r3.NewVec(35, 5, 15))
	})

	t.Run("J2 fork slot", func(t *testing.T) {
		requireLocated(t, body, false, r3.NewVec(75, 0, 10), r3.NewVec(75, 3.7, 10), r3.NewVec(75, -3.7, 10),
			r3.NewVec(44.8, 0, 10))
		requireLocated(t, body, true, r3.NewVec(75, 4.8, 10), r3.NewVec(75, -5, 10), r3.NewVec(43.2, 0, 10))
	})

	t.Run("J1 and J2 holes", func(t *testing.T) {
		requireLocated(t, body, false, r3.NewVec(0, 0, 4), r3.NewVec(2.4, 0, 4), r3.NewVec(70, -9, 10),
			r3.NewVec(70, 4.4, 12.4))
		requireLocated(t, body, true, r3.NewVec(3.2, 0, 4), r3.NewVec(70, 5, 13.2))
	})

	t.Run("J2 head pocket", func(t *testing.T) {
		// The hex pocket is open to y = -10 and its floor is at y = -6.5.
		requireLocated(t, body, false, r3.NewVec(70, -8, 10+3), r3.NewVec(70, -6.9, 13.3))
		requireLocated(t, body, true, r3.NewVec(70, -6.1, 13.5), r3.NewVec(70, -8, 19))
	})
}

func TestClampArmHead(t *testing.T) {
	body, sketches, err := clamparm.BuildHead(t.Context(), clamparm.NewDocument(), clamparm.DefaultParams())
	require.NoError(t, err)

	t.Run("solid, constrained, bounds", func(t *testing.T) {
		require.True(t, body.IsSolid())
		// The head outline, the boss, the two tongue cutters' sketches, the J4 hole and the stud hole.
		require.Len(t, sketches, 6)
		requireFullyConstrained(t, sketches)
		requireBounds(t, body, r3.NewVec(-10, -10, -10), r3.NewVec(55, 10, 12))
	})

	t.Run("tongue is 8 thick", func(t *testing.T) {
		requireLocated(t, body, true, r3.NewVec(10, 3.5, 5), r3.NewVec(10, -3.5, -5))
		requireLocated(t, body, false, r3.NewVec(10, 4.5, 5), r3.NewVec(10, -4.5, -5), r3.NewVec(10, 9, 0))
		requireLocated(t, body, true, r3.NewVec(25, 9, 0))
	})

	t.Run("boss is a cone from radius 6 to radius 4 over 2 mm", func(t *testing.T) {
		requireLocated(t, body, true, r3.NewVec(44.9, 0, 10.5), r3.NewVec(40, 4.9, 10.5), r3.NewVec(43.4, 0, 11.5))
		requireLocated(t, body, false, r3.NewVec(46.5, 0, 10.5), r3.NewVec(45.2, 0, 11.5), r3.NewVec(40, 0, 12.5))
	})

	t.Run("stud hole is 5.6 wide through boss and head", func(t *testing.T) {
		requireLocated(t, body, false, r3.NewVec(40, 0, 0), r3.NewVec(42.4, 0, 5), r3.NewVec(40, 2.4, -9),
			r3.NewVec(40, 0, 11.5))
		requireLocated(t, body, true, r3.NewVec(43.2, 0, 5), r3.NewVec(40, 3.2, -5))
	})

	t.Run("front corners are rounded with radius 2", func(t *testing.T) {
		requireLocated(t, body, false, r3.NewVec(54.9, 0, 9.9))
		requireLocated(t, body, true, r3.NewVec(54.5, 0, 9.0), r3.NewVec(54.9, 0, 0))
	})
}
