package escapement

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlates(t *testing.T) {
	b := testBuilder(t)
	back, err := b.buildBackPlate()
	require.NoError(t, err)
	require.True(t, back.IsSolid())
	front, err := b.buildFrontPlate()
	require.NoError(t, err)
	require.True(t, front.IsSolid())

	// Back plate: plate z -6..0, pillars to 28, seats 5 deep from the inner face, lip bores, mounting holes.
	g := &b.g
	for _, a := range []Pt{g.O, g.P, g.G} {
		require.False(t, inside(t, back, a.X+6, a.Y, -2), "bearing seat open")
		require.True(t, inside(t, back, a.X+6, a.Y, -5.5), "the 1 mm lip under the seat wall is solid")
		require.False(t, inside(t, back, a.X+4, a.Y, -5.5), "lip bore through")
	}
	for _, h := range mountHoles {
		require.False(t, inside(t, back, h.X+1, h.Y, -3), "mounting hole")
		require.True(t, inside(t, back, h.X+3, h.Y, -3), "plate around the mounting hole")
	}
	for _, p := range pillars {
		require.True(t, inside(t, back, p.X+4, p.Y, 10), "pillar")
		require.False(t, inside(t, back, p.X+0.8, p.Y, 22), "bolt hole in the pillar")
		require.False(t, inside(t, back, p.X+2.3, p.Y, 25), "nut cavity")
		require.False(t, inside(t, back, p.X+0.8, p.Y, 27), "the bolt hole runs through the roof")
		require.True(t, inside(t, back, p.X+2.0, p.Y, 26.5), "cavity roof")
		require.True(t, inside(t, back, p.X+2.6, p.Y, 25), "cavity wall")
		require.True(t, inside(t, back, p.X+0.8, p.Y, 20), "pillar below the hole")
	}
	// Front plate: z 28..34, seats open at the inner face, bolt holes, head pockets, socket.
	for _, a := range []Pt{g.O, g.P, g.G} {
		require.False(t, inside(t, front, a.X+6, a.Y, 29), "front bearing seat")
		require.False(t, inside(t, front, a.X+4, a.Y, 33.5), "lip bore")
		require.True(t, inside(t, front, a.X+6, a.Y, 33.5), "lip")
	}
	for _, p := range pillars {
		require.False(t, inside(t, front, p.X+1.0, p.Y, 30), "bolt hole")
		require.False(t, inside(t, front, p.X+2.0, p.Y, 33), "head pocket")
		require.True(t, inside(t, front, p.X+2.0, p.Y, 31), "plate under the head pocket")
		require.True(t, inside(t, front, p.X+2.4, p.Y, 33), "plate beside the head pocket")
	}
	require.False(t, inside(t, front, g.S.X+2.0, g.S.Y, 32), "suspension socket")
	require.True(t, inside(t, front, g.S.X-2.7, g.S.Y, 32), "plate round the socket")
	// The 0.5 mm chamfers cut the outer edges of both faces of both plates.
	require.False(t, inside(t, back, 69.9, 0, -5.9), "back plate rear edge")
	require.True(t, inside(t, back, 69.4, 0, -5.4), "back plate wall below the chamfer")
	require.False(t, inside(t, back, 69.9, 0, -0.1), "back plate inner edge")
	require.False(t, inside(t, front, 69.9, 0, 28.1), "front plate inner edge")
	require.False(t, inside(t, front, 69.9, 0, 33.9), "front plate front edge")
	require.True(t, inside(t, front, 69.4, 0, 33.4), "front plate wall below the chamfer")
	// The bearing seats carry a seat chamfer on the inner face and the lip bores a rim chamfer on the outer face.
	require.False(t, inside(t, back, 7.4, 0, -0.1), "back seat rim chamfer")
	require.False(t, inside(t, back, 5.1, 0, -5.9), "back lip rim chamfer")
	require.False(t, inside(t, front, 7.4, 0, 28.1), "front seat rim chamfer")
	require.False(t, inside(t, front, 5.1, 0, 33.9), "front lip rim chamfer")
}
