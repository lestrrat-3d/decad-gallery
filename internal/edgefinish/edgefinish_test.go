package edgefinish_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestChamferLongAll(t *testing.T) {
	doc := decad.New()
	body := slab(t, doc, 40, 30, 5, 10)
	offsets := []float64{0, 10}
	want := loopEdges(body, offsets, edgefinish.Long(edgefinish.All))
	require.GreaterOrEqual(t, want, 10, "two outlines of four edges and two hole circles")

	out, n, err := edgefinish.Chamfer(t.Context(), body, r3.Vec{Y: 1}, offsets, edgefinish.Long(edgefinish.All), 0.5)
	require.NoError(t, err)
	require.Equal(t, want, n)
	require.True(t, out.IsSolid())

	// The side faces keep the slab's full extent; each chamfered edge gains one 45° bevel face.
	bounds, err := out.Bounds()
	require.NoError(t, err)
	require.InDelta(t, 5, bounds.Min.X, 1e-6)
	require.InDelta(t, 45, bounds.Max.X, 1e-6)
	tilted := 0
	for _, f := range out.Faces() {
		switch s := f.Surface().(type) {
		case decad.Plane:
			if math.Abs(math.Abs(s.Frame.N().Y)-math.Sqrt2/2) < 1e-6 {
				tilted++
			}
		case decad.Cone:
			half, err := s.HalfAngle.In(units.Radians(1).Unit())
			require.NoError(t, err)
			if math.Abs(half-math.Pi/4) < 1e-6 {
				tilted++
			}
		}
	}
	require.Equal(t, want, tilted, "one bevel face per chamfered edge")
}

func TestChamferLongOuter(t *testing.T) {
	doc := decad.New()
	body := slab(t, doc, 40, 30, 5, 10)
	_, n, err := edgefinish.Chamfer(t.Context(), body, r3.Vec{Y: 1}, []float64{0, 10}, edgefinish.Long(edgefinish.Outer), 0.5)
	require.NoError(t, err)
	require.Equal(t, 8, n)
}

func TestChamferNoQualifyingFace(t *testing.T) {
	doc := decad.New()
	body := slab(t, doc, 15, 15, 0, 10)
	_, _, err := edgefinish.Chamfer(t.Context(), body, r3.Vec{Y: 1}, []float64{0, 10}, edgefinish.Long(edgefinish.All), 0.5)
	require.ErrorContains(t, err, "no edge")
	require.ErrorIs(t, err, edgefinish.ErrNoEdge)
}

func TestRuns(t *testing.T) {
	doc := decad.New()
	body := slab(t, doc, 40, 15, 4, 10)
	checked := 0
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(math.Abs(plane.Frame.N().Y)-1) > 1e-9 {
			continue
		}
		for _, l := range f.Loops() {
			runs := edgefinish.Runs(l)
			if l.IsOuter() {
				require.Len(t, runs, 4, "a rectangle's corners split it into four runs")
				lengths := 0.0
				for _, r := range runs {
					require.False(t, r.Closed)
					lengths += r.Length
				}
				require.InDelta(t, 110, lengths, 1e-6)
			} else {
				require.Len(t, runs, 1)
				require.True(t, runs[0].Closed, "a circle has no corner")
				require.InDelta(t, 2*math.Pi*4, runs[0].Length, 1e-6)
			}
			checked++
		}
	}
	require.Equal(t, 4, checked)
}

func TestChamferRunsKeepsLongRunsOnly(t *testing.T) {
	doc := decad.New()
	body := slab(t, doc, 40, 15, 0, 10)
	offsets := []float64{0, 10}
	out, n, err := edgefinish.ChamferRuns(t.Context(), body, r3.Vec{Y: 1}, offsets, edgefinish.Outer, never, 0.5)
	require.NoError(t, err)
	require.Equal(t, 4, n, "the two 40 mm edges of each face; the 15 mm edges stay sharp")
	require.True(t, out.IsSolid())
	before, err := body.Volume()
	require.NoError(t, err)
	after, err := out.Volume()
	require.NoError(t, err)
	// Two faces, two 40 mm edges each, lose a 0.5 mm wedge; the mitred ends add a little back.
	lost := before.Value.Base() - after.Value.Base()
	require.InDelta(t, 4*40*0.5*0.5/2, lost, 1.0)
	require.Greater(t, lost, 0.0)
}

func TestChamferRunsWholeLoop(t *testing.T) {
	doc := decad.New()
	body := slab(t, doc, 40, 15, 0, 10)
	_, n, err := edgefinish.ChamferRuns(t.Context(), body, r3.Vec{Y: 1}, []float64{0, 10}, edgefinish.Outer,
		edgefinish.Long(edgefinish.All), 0.5)
	require.NoError(t, err)
	require.Equal(t, 8, n, "a whole-loop filter takes every edge")
}
