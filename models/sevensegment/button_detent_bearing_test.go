package sevensegment

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestButtonDetentBearing(t *testing.T) {
	for _, phase := range []float64{0, 1, 10, 20, 109, 180, 251, 260, 271, 280, 301, 320, 331, 345, 354, 356, 358, 359} {
		doc := decad.New()
		detent, err := buildButtonDetent(t.Context(), doc, phase)
		require.NoError(t, err)
		cam := detentMesh(t, detent.Cam)
		r, d := buttonDetentPitch(phase)
		for _, offset := range []float64{.1, -.1} {
			frame, err := r3.NewFrame(r3.NewVec(10+r+offset, -105.75, 52), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
			require.NoError(t, err)
			bearing, err := ballbearing.Build(t.Context(), doc, ballbearing.Params{BoreDiameter: 3, OuterDiameter: 6, Width: 2.5, Placement: &frame})
			require.NoError(t, err)
			bearingMesh := detentMesh(t, bearing.Body)
			if offset > 0 {
				gap, ok, err := cam.Exceeds(bearingMesh, 0)
				require.NoError(t, err)
				require.True(t, ok, "phase %.0f: clearance lower bound %.6f", phase, gap)
				continue
			}
			// At a sampled station the surface point and outward normal rotate with the cam.
			normal := r3.NewVec(r/math.Hypot(r, d), 0, d/math.Hypot(r, d))
			point := r3.NewVec(10+r-3*normal.X, -107, 52-3*normal.Z)
			witness := point.Sub(normal.Scale(.04))
			anchor := point.Sub(normal.Scale(.6))
			distance, err := cam.PointDistance(anchor)
			require.NoError(t, err)
			require.Greater(t, distance, .56+cam.Bound()+1e-8, "phase %.0f", phase)
			location, err := detent.Cam.LocatePoint(t.Context(), anchor, units.Millimeters(.005))
			require.NoError(t, err)
			require.Equal(t, decad.PointInside, location, "phase %.0f cam anchor", phase)
			location, err = bearing.Body.LocatePoint(t.Context(), witness, units.Millimeters(.005))
			require.NoError(t, err)
			require.Equal(t, decad.PointInside, location, "phase %.0f bearing witness", phase)
		}
	}
}
