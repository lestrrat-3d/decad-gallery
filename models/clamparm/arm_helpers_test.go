package clamparm_test

import (
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const locateTol = 0.05

// requireLocated requires that each point is inside the body when want is true and outside when false. The body's
// section by the plane z = p.Z is read from its mesh and p's (x, y) is tested against its outlines by the even-odd
// rule, so a probe must lie at least a few tolerances away from any surface.
func requireLocated(t *testing.T, body *decad.Body, want bool, pts ...r3.Vec) {
	t.Helper()
	for _, p := range pts {
		f, err := r3.NewFrame(p, r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
		require.NoError(t, err)
		out, err := body.SectionOutline(t.Context(), f, units.Millimeters(locateTol))
		require.NoError(t, err)
		crossings := 0
		for _, loop := range out.Loops {
			for i, a := range loop {
				b := loop[(i+1)%len(loop)]
				if (a.Y > p.Y) == (b.Y > p.Y) {
					continue
				}
				if a.X+(p.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X) > p.X {
					crossings++
				}
			}
		}
		require.Equal(t, want, crossings%2 == 1, "point %v", p)
	}
}
