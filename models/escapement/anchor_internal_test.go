package escapement

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func anchorCases() map[string]Params {
	p := DefaultParams()
	wide, narrow := p, p
	wide.LockAngle, wide.LiftAngle, wide.PendulumLength = 1.25*deg, 3*deg, 120
	narrow.LiftAngle, narrow.PendulumLength = 2*deg, 250
	return map[string]Params{"default": p, "wide": wide, "narrow": narrow}
}

func TestAnchor(t *testing.T) {
	for name, p := range anchorCases() {
		t.Run(name, func(t *testing.T) {
			g, err := Layout(p)
			require.NoError(t, err)
			b := newBuilder(t.Context(), NewDocument(), g)
			body, err := b.buildAnchor()
			require.NoError(t, err)
			require.True(t, body.IsSolid())
			require.NotEmpty(t, b.sketches)

			f, err := r3.NewFrame(r3.NewVec(0, 0, 23.5), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
			require.NoError(t, err)
			o, err := body.SectionOutline(t.Context(), f, units.Millimeters(0.02))
			require.NoError(t, err)
			off := o.OffSurface.Base()
			outline := outerLoop2(o.Loops, g.P)
			for name, want := range map[string]Pt{"C_E": g.CE, "F_E": g.FE, "C_X": g.CX, "F_X": g.FX, "M_E": g.ME,
				"M_X": g.MX} {
				best := math.Inf(1)
				for i := range outline {
					best = math.Min(best, distPointSeg(want, outline[i], outline[(i+1)%len(outline)]))
				}
				require.Less(t, best, 0.02+off, name)
			}
		})
	}
}

// outerLoop2 returns the outline loop with the most extent, relative to the point origin.
func outerLoop2(loops [][]r3.Vec, origin Pt) []Pt {
	var best []Pt
	far := -1.0
	for _, l := range loops {
		pts := loopPoints(l)
		m := 0.0
		for i := range pts {
			pts[i] = pts[i].Sub(origin)
			m = math.Max(m, pts[i].Len())
		}
		if m > far {
			best, far = pts, m
		}
	}
	return best
}
