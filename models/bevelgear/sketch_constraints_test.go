package bevelgear

import (
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchassert"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/stretchr/testify/require"
)

func TestVirtualToothSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	ordinary := DefaultParams()
	ordinary.Teeth, ordinary.MateTeeth, ordinary.Samples = 16, 12, 5
	embedded := DefaultParams()
	embedded.Teeth, embedded.MateTeeth = 60, 60
	for _, tc := range []struct {
		name string
		p    Params
	}{
		{"ordinary", ordinary},
		{"embedded", embedded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			world := sketch.NewWorld()
			s, _, err := BuildVirtualToothProfile(t.Context(), world, world.XY(), tc.p)
			require.NoError(t, err)
			sketchassert.FullyConstrained(t, "virtual tooth", s)
		})
	}
}

func TestBoreSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	s, err := boreSketch(t.Context(), sketch.NewWorld(), -1, 4)
	require.NoError(t, err)
	sketchassert.FullyConstrained(t, "bore", s)
}

func TestAxialSectionSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	p := DefaultParams()
	p.Teeth, p.MateTeeth, p.Samples = 8, 8, 5
	d, err := derive(p)
	require.NoError(t, err)
	section, err := blankSection(p, d)
	require.NoError(t, err)
	s, err := axialSectionSketch(t.Context(), sketch.NewWorld(), section, "blank")
	require.NoError(t, err)
	sketchassert.FullyConstrained(t, "blank section", s)
}

func TestSliceSheetSketchIsFullyConstrained(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		origin, along r3.Vec
	}{
		{"diagonal", r3.NewVec(1.5, -2, 0), r3.NewVec(0.6, 0.8, 0)},
		{"horizontal", r3.NewVec(1, 2, 0), r3.NewVec(1, 0, 0)},
		{"vertical", r3.NewVec(1, 2, 0), r3.NewVec(0, 1, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, err := sliceSheetSketch(t.Context(), sketch.NewWorld(), tc.origin, tc.along, 3)
			require.NoError(t, err)
			sketchassert.FullyConstrained(t, "slice sheet", s)
		})
	}
}
