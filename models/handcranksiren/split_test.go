package handcranksiren_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// splitTolerance is the chord tolerance of the split check's mesh.
const splitTolerance = 0.01

// requireSplitShafts checks each shaft and the crank pin against "Split shafts" in docs/model-pipeline.md. A shaft's
// split plane holds its axis and its flat's normal. Built at crank angle 0 the flat faces +Z and the plane's normal is
// +X; the pose turns both about −Y by the shaft's angle, so the normal is (cos a, 0, sin a).
func requireSplitShafts(t *testing.T, a *handcranksiren.Assembly, parts map[string]handcranksiren.Part) {
	t.Helper()
	s := a.State
	for _, shaft := range []struct {
		name   string
		point  r3.Vec
		angle  float64
		height float64 // the larger of the shaft and collar radii
		flats  int     // flat planes along the whole shaft
	}{
		{"crank-shaft", s.Crank, s.CrankAngle, 5.5, 2},
		{"intermediate-shaft", s.Intermediate, s.IntermediateAngle, 4.0, 1},
		{"rotor-shaft", s.Rotor, s.RotorAngle, 5.5, 1},
	} {
		t.Run(shaft.name, func(t *testing.T) {
			r, err := splitcheck.Check(t.Context(), splitcheck.Shaft{
				Body:  parts[shaft.name].Body,
				Point: shaft.point,
				Axis:  r3.Vec{Y: 1},
			}, splitTolerance)
			require.NoError(t, err)
			require.Empty(t, r.Violations)
			require.Equal(t, shaft.flats, r.Flats, "flats along the whole shaft")
			want := r3.Vec{X: math.Cos(shaft.angle), Z: math.Sin(shaft.angle)}
			require.InDelta(t, 1, math.Abs(r.Normal.Dot(want)), 1e-9, "split normal %v", r.Normal)
			for i, h := range r.HalfHeights {
				require.InDelta(t, shaft.height, h, splitTolerance, "half %d", i+1)
			}
		})
	}
	// The crank pin's split plane holds its axis and the hand's driving force, across the arm, so its normal runs along
	// the arm (models/crankhandle, "Split pin"). Its two cross bolts clamp the halves; their lines are exempt from the
	// mirror check with the nut cavity's radius, as crankhandle's own test exempts them.
	t.Run("crank-pin", func(t *testing.T) {
		axis, err := handcranksiren.PosePoint(a.Params, "crank", handcranksiren.PinAxis)
		require.NoError(t, err)
		fasteners, err := handcranksiren.Fasteners(a.Params)
		require.NoError(t, err)
		var exempt []splitcheck.Line
		for _, f := range fasteners {
			if strings.HasPrefix(f.Bolt, "crank-cross-bolt-") {
				exempt = append(exempt, splitcheck.Line{Point: f.Head, Dir: f.Dir, Radius: m2fastener.NutCavityRadius + 0.1})
			}
		}
		require.Len(t, exempt, 2)
		r, err := splitcheck.Check(t.Context(), splitcheck.Shaft{
			Body: parts["crank-pin"].Body, Point: axis, Axis: r3.Vec{Y: 1}, Normal: s.ArmDirection, Exempt: exempt,
		}, splitTolerance)
		require.NoError(t, err)
		require.Empty(t, r.Violations)
		require.LessOrEqual(t, r.Bridge, splitcheck.MaxBridge)
		for i, h := range r.HalfHeights {
			require.InDelta(t, crankhandle.HeadR, h, splitTolerance, "half %d", i+1)
		}
	})
}
