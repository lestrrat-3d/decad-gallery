package edgefinish_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/edgefinish"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// slab extrudes a w by h rectangle with a hole of radius holeR (0 for none) along +Y from Y 0, thick long. The sketch
// plane has U = +X and V = -Z.
func slab(t *testing.T, doc *decad.Document, w, h, holeR, thick float64) *decad.Body {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Z: -1})
	require.NoError(t, err)
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(f)
	require.NoError(t, err)
	s, err := world.CreateSketch(plane)
	require.NoError(t, err)
	r := s.CreateRectangle(5, 5, 5+w, 5+h)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, 5),
		sketch.NewVerticalDistance(s.Origin(), r.A, 5),
		sketch.NewHorizontalDistance(r.A, r.B, w),
		sketch.NewVerticalDistance(r.B, r.C, h),
	)
	holes := 0
	if holeR > 0 {
		holes = 1
		c := s.CreatePoint(5+w/2, 5+h/2)
		s.AddConstraint(
			sketch.NewHorizontalDistance(s.Origin(), c, 5+w/2),
			sketch.NewVerticalDistance(s.Origin(), c, 5+h/2),
		)
		s.AddConstraint(sketch.NewRadius(s.CreateCircle(c, holeR), holeR))
	}
	_, err = s.Solve(t.Context())
	require.NoError(t, err)
	var chosen *sketch.Profile
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == holes {
			chosen = p
		}
	}
	require.NotNil(t, chosen)
	body, err := decad.Extrude(t.Context(), doc, s, chosen, decad.Distance{D: units.Millimeters(thick), Dir: decad.Along})
	require.NoError(t, err)
	return body
}

// loopEdges returns the total edge count of the kept loops of the large faces at the offsets.
func loopEdges(body *decad.Body, offsets []float64, keep edgefinish.LoopFilter) int {
	n := 0
	for _, f := range body.Faces() {
		plane, ok := f.Surface().(decad.Plane)
		if !ok || math.Abs(math.Abs(plane.Frame.N().Y)-1) > 1e-9 {
			continue
		}
		area, err := f.Area()
		if err != nil || area.Value.Base() < edgefinish.LargeFaceArea {
			continue
		}
		at := plane.Frame.Origin().Y
		match := false
		for _, o := range offsets {
			match = match || math.Abs(at-o) < 1e-6
		}
		if !match {
			continue
		}
		for _, l := range f.Loops() {
			if keep(l, l.Edges()) {
				n += len(l.Edges())
			}
		}
	}
	return n
}

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
}
