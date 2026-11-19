package hexnut_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/hexnut"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestBuildSizes(t *testing.T) {
	full := hexnut.Options{Bore: true, ChamferLower: true, ChamferUpper: true}
	for name, size := range map[string]hexnut.Size{"M2": m2, "M5": m5} {
		t.Run(name, func(t *testing.T) {
			r, err := hexnut.Build(t.Context(), decad.New(), size, full, nil)
			require.NoError(t, err)
			require.True(t, r.Body.IsSolid())
			for _, s := range []*sketch.Sketch{r.Hexagon, r.LowerChamfer, r.UpperChamfer} {
				requireFullyConstrained(t, s)
			}
			box, err := r.Body.Bounds()
			require.NoError(t, err)
			tol := box.Bound.Base() + 1e-8
			require.InDelta(t, size.AcrossCorners()/2, box.Max.X, tol)
			require.InDelta(t, size.AcrossFlats/2, box.Max.Y, tol)
			require.InDelta(t, 0, box.Min.Z, tol)
			require.InDelta(t, size.Thickness, box.Max.Z, tol)
		})
	}
}

func TestBuildNoBoreIsSolidPrism(t *testing.T) {
	r, err := hexnut.Build(t.Context(), decad.New(), m5, hexnut.Options{}, nil)
	require.NoError(t, err)
	require.True(t, r.Body.IsSolid())
	require.Nil(t, r.LowerChamfer)
	require.Nil(t, r.UpperChamfer)
	requireFullyConstrained(t, r.Hexagon)
	volume, err := r.Body.Volume()
	require.NoError(t, err)
	want := math.Sqrt(3) / 2 * m5.AcrossFlats * m5.AcrossFlats * m5.Thickness
	require.InDelta(t, want, volume.Value.Base(), volume.Bound.Base()+1e-3)
	for _, face := range r.Body.Faces() {
		require.NotEqual(t, decad.KindCylinder, face.Surface().Kind())
	}
}

func TestBuildOneChamfer(t *testing.T) {
	r, err := hexnut.Build(t.Context(), decad.New(), m2, hexnut.Options{Bore: true, ChamferUpper: true}, nil)
	require.NoError(t, err)
	require.Nil(t, r.LowerChamfer)
	requireFullyConstrained(t, r.UpperChamfer)
}

func TestBuildRejectsInvalidInputs(t *testing.T) {
	_, err := hexnut.Build(t.Context(), nil, m2, hexnut.Options{}, nil)
	require.ErrorContains(t, err, "document")
	_, err = hexnut.Build(nil, decad.New(), m2, hexnut.Options{}, nil) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
	_, err = hexnut.Build(t.Context(), decad.New(), hexnut.Size{}, hexnut.Options{}, nil)
	require.ErrorContains(t, err, "size")
}
