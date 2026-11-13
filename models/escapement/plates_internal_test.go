package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
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
	require.False(t, inside(t, back, 57.9, 0, -5.9), "back plate rear edge")
	require.True(t, inside(t, back, 57.4, 0, -5.4), "back plate wall below the chamfer")
	require.False(t, inside(t, back, 57.9, 0, -0.1), "back plate inner edge")
	require.False(t, inside(t, front, 57.9, 0, 28.1), "front plate inner edge")
	require.False(t, inside(t, front, 57.9, 0, 33.9), "front plate front edge")
	require.True(t, inside(t, front, 57.4, 0, 33.4), "front plate wall below the chamfer")
	// The bearing seats carry a seat chamfer on the inner face and the lip bores a rim chamfer on the outer face.
	for _, a := range []Pt{g.O, g.P, g.G} {
		require.False(t, inside(t, back, a.X+7.4, a.Y, -0.1), "back seat rim chamfer")
		require.False(t, inside(t, back, a.X+5.1, a.Y, -5.9), "back lip rim chamfer")
		require.True(t, inside(t, back, a.X+5.6, a.Y, -5.4), "back plate below the lip rim chamfer")
		require.False(t, inside(t, front, a.X+7.4, a.Y, 28.1), "front seat rim chamfer")
		require.False(t, inside(t, front, a.X+5.1, a.Y, 33.9), "front lip rim chamfer")
		require.True(t, inside(t, front, a.X+5.6, a.Y, 33.4), "front plate below the lip rim chamfer")
	}
}

// TestPlateWindows checks by section outline at mid-plane that each plate is open where the mechanism shows through and
// solid in the spine, the hub rings and the rims.
func TestPlateWindows(t *testing.T) {
	b := testBuilder(t)
	back, err := b.buildBackPlate()
	require.NoError(t, err)
	front, err := b.buildFrontPlate()
	require.NoError(t, err)
	open := [][2]float64{{29.5114, 27.5334}, {28.2168, 26.2738}, {37.1764, 37.3885}, {40, 0}, {20, 0}, {30, -45.75},
		{37, -45.75}, {12.5, -45.75}, {10.8, 0}}
	material := [][2]float64{{8.5, 0}, {8.5, 56.5685}, {8.5, -45.75}, {0, 30}, {47, 74}, {-47, 74}, {47, -88}, {-47, -88}}
	for name, c := range map[string]struct {
		body *decad.Body
		z    float64
		more [][2]float64
	}{"back": {back, -3, nil}, "front": {front, 31, [][2]float64{{0, 75}}}} {
		for _, p := range open {
			for _, sx := range []float64{1, -1} {
				require.False(t, inside(t, c.body, sx*p[0], p[1], c.z), "%s plate: window at (%v, %v)", name, sx*p[0], p[1])
			}
		}
		for _, p := range append(material, c.more...) {
			require.True(t, inside(t, c.body, p[0], p[1], c.z), "%s plate: material at (%v, %v)", name, p[0], p[1])
		}
	}
}

// TestPlateMaterial checks the wall between each window and the features near it.
func TestPlateMaterial(t *testing.T) {
	b := testBuilder(t)
	back, err := b.buildBackPlate()
	require.NoError(t, err)
	front, err := b.buildFrontPlate()
	require.NoError(t, err)
	g := &b.g
	for name, c := range map[string]struct {
		body *decad.Body
		z    float64
	}{"back": {back, -3}, "front": {front, 31}} {
		o := sectionAt(t, c.body, c.z)
		loops := loopsBy(o)
		// The two windows are the loops that reach beyond the spine and below the top rim: the ones with a vertex at x = 44.
		var windows [][]Pt
		for _, l := range loops {
			for _, v := range l {
				if math.Abs(math.Abs(v.X)-windowSide) < 0.01 && v.Y > -77 && v.Y < 57 {
					windows = append(windows, l)
					break
				}
			}
		}
		require.Len(t, windows, 2, "%s plate windows", name)
		gap := func(centre Pt, r float64) float64 {
			m := math.Inf(1)
			for _, w := range windows {
				for i := range w {
					m = math.Min(m, distPointSeg(centre, w[i], w[(i+1)%len(w)]))
				}
			}
			return m - r
		}
		off := o.OffSurface.Base()
		for _, a := range []Pt{g.O, g.P, g.G} {
			require.GreaterOrEqual(t, gap(a, seatRadius), 2.0-off, "%s plate: pocket at %v to a window", name, a)
		}
		for _, p := range pillars {
			require.GreaterOrEqual(t, gap(p, 1.3), 3.0-off, "%s plate: bolt hole at %v to a window", name, p)
		}
		if name == "front" {
			require.GreaterOrEqual(t, gap(g.S, 2.55), 5.0-off, "socket to a window")
		}
	}
}
