package plaquestand_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlaqueStandOutlineCases(t *testing.T) {
	const tol = 1e-9
	concave := map[int]bool{5: true, 9: true, 13: true}
	for _, tc := range cases() {
		t.Run(tc.name, func(t *testing.T) {
			dm, err := tc.params.Derive()
			require.NoError(t, err)
			o := dm.Outline
			require.Len(t, o, 18)

			convex, concaveSweep := 0.0, 0.0
			for i, s := range o {
				next := o[(i+1)%len(o)]
				require.InDelta(t, 0, s.Start.X, tol, "segment %d X", i)
				require.InDelta(t, 0, s.End.X, tol, "segment %d X", i)
				require.InDelta(t, next.Start.Y, s.End.Y, tol, "segment %d joins %d in Y", i, i+1)
				require.InDelta(t, next.Start.Z, s.End.Z, tol, "segment %d joins %d in Z", i, i+1)
				require.Equal(t, i%2 == 1, s.IsArc(), "segment %d alternates line and arc", i)
				if !s.IsArc() {
					require.Greater(t, s.End.Sub(s.Start).Len(), 0.0, "segment %d line length", i)
					continue
				}
				require.InDelta(t, s.Radius, s.Start.Sub(s.Center).Len(), tol, "arc %d start", i)
				require.InDelta(t, s.Radius, s.End.Sub(s.Center).Len(), tol, "arc %d end", i)
				require.Equal(t, !concave[i], s.Convex, "arc %d convex flag", i)
				if s.Convex {
					convex += s.Sweep()
					continue
				}
				concaveSweep += s.Sweep()
			}
			// Every join is tangent: each line's direction is perpendicular to the radius at both of its arc joints.
			for i := 1; i < len(o); i += 2 {
				s := o[i]
				prev, next := o[i-1], o[(i+1)%len(o)]
				dir := prev.End.Sub(prev.Start)
				require.InDelta(t, 0, dir.Dot(s.Start.Sub(s.Center))/(dir.Len()*s.Radius), tol, "arc %d start tangent", i)
				dir = next.End.Sub(next.Start)
				require.InDelta(t, 0, dir.Dot(s.End.Sub(s.Center))/(dir.Len()*s.Radius), tol, "arc %d end tangent", i)
			}
			// Sweep's asin form loses about 1e-7 on the half-turn arcs, so this sum is checked to 1e-6.
			require.InDelta(t, 2*math.Pi, convex-concaveSweep, 1e-6)
			require.InDelta(t, tc.area, o.Area(), 1e-6)
			require.InDelta(t, tc.length, o.Length(), 1e-6)
		})
	}
}
