package spurgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchassert"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestGearSketchesAreFullyConstrained(t *testing.T) {
	t.Parallel()
	embedded := Params{Module: 0.5, Teeth: 60, PressureAngle: 20 * math.Pi / 180, Thickness: 2, Samples: 15}
	embeddedWide := Params{Module: 0.5, Teeth: 30, PressureAngle: 30 * math.Pi / 180, Thickness: 2, Samples: 5}
	for _, tc := range []struct {
		name string
		p    Params
	}{
		{"ordinary default", DefaultParams()},
		{"embedded sixty teeth", embedded},
		{"embedded thirty teeth wide pressure angle", embeddedWide},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := sketch.NewWorld()
			s, _, _, err := buildGearProfile(t.Context(), w, w.XY(), tc.p, derive(tc.p))
			require.NoError(t, err)
			sketchassert.FullyConstrained(t, "gear profile", s)
		})
	}
}

func TestToothSectionSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	for _, angle := range []float64{0, 0.3, -math.Pi / 2} {
		w := sketch.NewWorld()
		s, _, err := BuildToothSection(t.Context(), w, w.XY(), DefaultParams(), angle)
		require.NoError(t, err)
		sketchassert.FullyConstrained(t, "tooth section", s)
	}
}

func TestBoreSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	s, err := boreSketch(t.Context(), sketch.NewWorld(), 1, 5)
	require.NoError(t, err)
	sketchassert.FullyConstrained(t, "bore", s)
}
