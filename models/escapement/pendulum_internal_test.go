package escapement

import (
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPendulumParts(t *testing.T) {
	for name, p := range anchorCases() {
		t.Run(name, func(t *testing.T) {
			g, err := Layout(p)
			require.NoError(t, err)
			b := newBuilder(t.Context(), NewDocument(), g)

			crutch, err := b.buildCrutch()
			require.NoError(t, err)
			require.True(t, crutch.IsSolid())
			f, err := r3.NewFrame(r3.NewVec(0, 0, 35), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
			require.NoError(t, err)
			o, err := crutch.SectionOutline(t.Context(), f, units.Millimeters(0.02))
			require.NoError(t, err)
			require.Less(t, o.PolygonArea().Base(), 400.0, "crutch area stays under the large-face limit")
			// The D socket is open, the pin stands on the end and the hub is kept.
			require.False(t, inside(t, crutch, g.P.X+2.0, g.P.Y, 35.5))
			require.True(t, inside(t, crutch, g.P.X-5.5, g.P.Y, 35.5))
			require.True(t, inside(t, crutch, g.P.X+0.6, g.P.Y-45, 40))
			require.False(t, inside(t, crutch, g.P.X+1.7, g.P.Y-45, 40))

			pend, err := b.buildPendulum()
			require.NoError(t, err)
			require.True(t, pend.IsSolid())
			lp := p.PendulumLength
			// The slot goes through, the pocket is open on the front face with a floor and a lip, the rod and bob are solid.
			require.False(t, inside(t, pend, g.S.X+0.5, g.S.Y-58.7, 40))
			require.False(t, inside(t, pend, g.S.X+6, g.S.Y, 41))
			require.True(t, inside(t, pend, g.S.X+6, g.S.Y, 38.25))
			require.False(t, inside(t, pend, g.S.X+4.5, g.S.Y, 38.25), "the lip bore is open through the floor")
			require.True(t, inside(t, pend, g.S.X+6, g.S.Y, 38.25))
			require.True(t, inside(t, pend, g.S.X+0.6, g.S.Y-lp, 40))
			// The 0.5 chamfer cuts the rod's corner on both faces.
			require.False(t, inside(t, pend, g.S.X+3.9, g.S.Y-80, 37.85))
			require.True(t, inside(t, pend, g.S.X+3.4, g.S.Y-80, 38.4))
			require.False(t, inside(t, pend, g.S.X+3.9, g.S.Y-80, 43.65))

			cap, err := b.buildCap()
			require.NoError(t, err)
			require.True(t, cap.IsSolid())
			require.False(t, inside(t, cap, g.S.X+2.0, g.S.Y, 45))
			require.True(t, inside(t, cap, g.S.X-3, g.S.Y, 45))
			require.NotEmpty(t, b.sketches)
		})
	}
}
