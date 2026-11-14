package cornercheck_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

const plateHeight = 4.0

var axisZ = r3.Vec{Z: 1}

type xy struct{ x, y float64 }

// extrude verifies the sketch, picks the profile with the given hole count and extrudes it plateHeight along +Z.
func extrude(t *testing.T, s *sketch.Sketch, holes int) *decad.Body {
	t.Helper()
	_, err := s.Solve(t.Context())
	require.NoError(t, err)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
	var chosen *sketch.Profile
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == holes {
			chosen = p
		}
	}
	require.NotNil(t, chosen)
	body, err := decad.Extrude(t.Context(), decad.New(), s, chosen,
		decad.Distance{D: units.Millimeters(plateHeight), Dir: decad.Along})
	require.NoError(t, err)
	return body
}

func newSketch(t *testing.T) *sketch.Sketch {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(f)
	require.NoError(t, err)
	s, err := world.CreateSketch(plane)
	require.NoError(t, err)
	return s
}

// polygon extrudes a closed polygon whose points are each dimensioned from the origin.
func polygon(t *testing.T, pts []xy) *decad.Body {
	t.Helper()
	s := newSketch(t)
	points := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		points[i] = sketchcheck.DimensionPoint(s, p.x, p.y)
	}
	for i := range pts {
		s.CreateLine(points[i], points[(i+1)%len(pts)])
	}
	return extrude(t, s, 0)
}

// axisPolygon extrudes an axis-aligned polygon whose edges sit by their distance from the origin, rounding corner i
// with radii[i] when it is not zero. Every edge is constrained by name, so the rounds keep the sketch constrained.
func axisPolygon(t *testing.T, pts []xy, radii []float64) *decad.Body {
	t.Helper()
	s := newSketch(t)
	n := len(pts)
	points := make([]*sketch.Point, n)
	for i, p := range pts {
		points[i] = s.CreatePoint(p.x, p.y)
	}
	lines := make([]*sketch.Line, n)
	for i := range pts {
		j := (i + 1) % n
		lines[i] = s.CreateLine(points[i], points[j])
		if pts[i].x == pts[j].x {
			s.AddConstraint(sketch.NewVertical(lines[i]), sketch.NewDistancePointLine(s.Origin(), lines[i], math.Abs(pts[i].x)))
			continue
		}
		s.AddConstraint(sketch.NewHorizontal(lines[i]), sketch.NewDistancePointLine(s.Origin(), lines[i], math.Abs(pts[i].y)))
	}
	for i, r := range radii {
		if r == 0 {
			continue
		}
		prev, next := lines[(i+n-1)%n], lines[i]
		_, err := sketch.CreateFillet(s, prev, next, points[i], r)
		require.NoError(t, err)
	}
	return extrude(t, s, 0)
}

// wedge extrudes an isosceles triangle on a 20 mm base whose apex angle is apex degrees, rounding the apex with r
// when it is not zero.
func wedge(t *testing.T, apex, r float64) *decad.Body {
	t.Helper()
	s := newSketch(t)
	base := (180 - apex) / 2
	a := sketchcheck.DimensionPoint(s, 5, 5)
	b := s.CreatePoint(25, 5)
	c := s.CreatePoint(15, 5+10*math.Tan(base*math.Pi/180))
	ab := s.CreateLine(a, b)
	bc := s.CreateLine(b, c)
	ac := s.CreateLine(a, c)
	s.AddConstraint(
		sketch.NewHorizontal(ab),
		sketch.NewHorizontalDistance(a, b, 20),
		sketch.NewAngle(ab, ac, base),
		sketch.NewAngle(ab, bc, 180-base),
	)
	if r > 0 {
		_, err := sketch.CreateFillet(s, bc, ac, c, r)
		require.NoError(t, err)
	}
	return extrude(t, s, 0)
}

// slotPlate extrudes a 20 mm square with a 1.5 mm slot in its top edge, ending in a radial line or a full round.
func slotPlate(t *testing.T, round bool) *decad.Body {
	t.Helper()
	if !round {
		return axisPolygon(t, []xy{
			{5, 5}, {25, 5}, {25, 25}, {15.75, 25}, {15.75, 17}, {14.25, 17}, {14.25, 25}, {5, 25},
		}, make([]float64, 8))
	}
	s := newSketch(t)
	pts := []xy{{5, 5}, {25, 5}, {25, 25}, {15.75, 25}}
	var points []*sketch.Point
	for _, p := range pts {
		points = append(points, sketchcheck.DimensionPoint(s, p.x, p.y))
	}
	center := sketchcheck.DimensionPoint(s, 15, 17)
	east := sketchcheck.DimensionPoint(s, 15.75, 17)
	west := s.CreatePoint(14.25, 17)
	s.AddConstraint(sketch.NewVerticalDistance(s.Origin(), west, 17))
	left := sketchcheck.DimensionPoint(s, 14.25, 25)
	last := sketchcheck.DimensionPoint(s, 5, 25)
	s.CreateLine(points[0], points[1])
	s.CreateLine(points[1], points[2])
	s.CreateLine(points[2], points[3])
	s.CreateLine(points[3], east)
	s.CreateArc(center, west, east)
	s.CreateLine(west, left)
	s.CreateLine(left, last)
	s.CreateLine(last, points[0])
	return extrude(t, s, 0)
}

// dBorePlate extrudes a plate with a D bore of radius 2.7 whose flat lies 2.2 from the axis, as the spiral spring has.
func dBorePlate(t *testing.T) *decad.Body {
	t.Helper()
	s := newSketch(t)
	r := s.CreateRectangle(-8, -8, 8, 8)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, -8),
		sketch.NewVerticalDistance(s.Origin(), r.A, -8),
		sketch.NewHorizontalDistance(r.A, r.B, 16),
		sketch.NewVerticalDistance(r.B, r.C, 16),
	)
	half := math.Sqrt(2.7*2.7 - 2.2*2.2)
	start := s.CreatePoint(2.2, half)
	end := s.CreatePoint(2.2, -half)
	arc := s.CreateArc(s.Origin(), start, end)
	flat := s.CreateLine(end, start)
	s.AddConstraint(
		sketch.NewRadius(arc, 2.7),
		sketch.NewVertical(flat),
		sketch.NewDistancePointLine(s.Origin(), flat, 2.2),
	)
	return extrude(t, s, 1)
}

func requireFindings(t *testing.T, body *decad.Body, kind cornercheck.Kind, count int) []cornercheck.Finding {
	t.Helper()
	mids, err := cornercheck.MidHeights(body, axisZ)
	require.NoError(t, err)
	require.Len(t, mids, 1)
	require.InDelta(t, plateHeight/2, mids[0], 1e-6)
	fs, err := cornercheck.Check(t.Context(), body, axisZ, nil)
	require.NoError(t, err)
	var got []cornercheck.Finding
	for _, f := range fs {
		if f.Kind == kind {
			got = append(got, f)
		}
	}
	require.Len(t, got, count, cornercheck.Report(fs, 5))
	return fs
}

func TestInsideCorners(t *testing.T) {
	lShape := []xy{{5, 5}, {25, 5}, {25, 15}, {15, 15}, {15, 25}, {5, 25}}
	round := func(r float64) []float64 { return []float64{0, 0, 0, r, 0, 0} }

	t.Run("sharp notch", func(t *testing.T) {
		fs := requireFindings(t, axisPolygon(t, lShape, round(0)), cornercheck.Inside, 1)
		require.Len(t, fs, 1)
		require.InDelta(t, 15, fs[0].At.X, 0.01)
		require.InDelta(t, 15, fs[0].At.Y, 0.01)
		require.InDelta(t, 90, fs[0].Turn, 1)
	})
	t.Run("round 0.5", func(t *testing.T) {
		requireFindings(t, axisPolygon(t, lShape, round(0.5)), cornercheck.Inside, 0)
	})
	t.Run("round 0.4", func(t *testing.T) {
		requireFindings(t, axisPolygon(t, lShape, round(0.4)), cornercheck.Inside, 1)
	})
	t.Run("45 degree turn", func(t *testing.T) {
		body := polygon(t, []xy{{5, 5}, {25, 5}, {25, 15}, {15, 15}, {5, 25}})
		fs, err := cornercheck.Check(t.Context(), body, axisZ, nil)
		require.NoError(t, err)
		for _, f := range fs {
			require.NotEqual(t, cornercheck.Inside, f.Kind, cornercheck.Report(fs, 5))
		}
	})
	t.Run("slot ending in a line", func(t *testing.T) {
		requireFindings(t, slotPlate(t, false), cornercheck.Inside, 2)
	})
	t.Run("slot ending in a round", func(t *testing.T) {
		requireFindings(t, slotPlate(t, true), cornercheck.Inside, 0)
	})
	t.Run("D bore", func(t *testing.T) {
		requireFindings(t, dBorePlate(t), cornercheck.Inside, 0)
	})
}

func TestTips(t *testing.T) {
	t.Run("60 degrees", func(t *testing.T) {
		requireFindings(t, wedge(t, 60, 0), cornercheck.Tip, 0)
	})
	t.Run("45 degrees", func(t *testing.T) {
		fs := requireFindings(t, wedge(t, 45, 0), cornercheck.Tip, 1)
		require.InDelta(t, 15, fs[0].At.X, 0.01)
		require.InDelta(t, 135, fs[0].Turn, 1)
	})
	t.Run("45 degrees blunted", func(t *testing.T) {
		requireFindings(t, wedge(t, 45, 0.4), cornercheck.Tip, 0)
	})
}

func TestFacetedBody(t *testing.T) {
	doc := decad.New()
	box := func(x0, y0, x1, y1 float64) *decad.Body {
		s := newSketch(t)
		r := s.CreateRectangle(x0, y0, x1, y1)
		s.AddConstraint(
			sketch.NewHorizontalDistance(s.Origin(), r.A, x0),
			sketch.NewVerticalDistance(s.Origin(), r.A, y0),
			sketch.NewHorizontalDistance(r.A, r.B, x1-x0),
			sketch.NewVerticalDistance(r.B, r.C, y1-y0),
		)
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
		_, err := s.Solve(t.Context())
		require.NoError(t, err)
		body, err := decad.Extrude(t.Context(), doc, s, s.Profiles()[0],
			decad.Distance{D: units.Millimeters(plateHeight), Dir: decad.Along})
		require.NoError(t, err)
		return body
	}
	union, err := decad.Union(t.Context(), box(5, 5, 25, 15), box(5, 5, 15, 25))
	require.NoError(t, err)
	// The union's only inside corner is the notch at (15, 15).
	fs := requireFindings(t, union, cornercheck.Inside, 1)
	require.InDelta(t, 15, fs[0].At.X, 0.01)
	require.InDelta(t, 15, fs[0].At.Y, 0.01)
}

func TestExemptions(t *testing.T) {
	body := axisPolygon(t, []xy{{5, 5}, {25, 5}, {25, 15}, {15, 15}, {15, 25}, {5, 25}}, make([]float64, 6))
	notch := cornercheck.Exempt{Reason: "notch", Near: cornercheck.Disc(r3.Vec{X: 15, Y: 15}, axisZ, 0.5)}
	away := cornercheck.Exempt{Reason: "unrelated spot", Near: cornercheck.Disc(r3.Vec{X: 5, Y: 5}, axisZ, 0.5)}

	fs, err := cornercheck.Check(t.Context(), body, axisZ, []cornercheck.Exempt{notch})
	require.NoError(t, err)
	require.Empty(t, fs, cornercheck.Report(fs, 5))

	_, err = cornercheck.Check(t.Context(), body, axisZ, []cornercheck.Exempt{notch, away})
	require.ErrorContains(t, err, "unrelated spot")

	ring := cornercheck.Annulus(r3.Vec{X: 15, Y: 15}, axisZ, 0, 0.5)
	require.True(t, ring(r3.Vec{X: 15.2, Y: 15, Z: 99}))
	require.False(t, ring(r3.Vec{X: 16, Y: 15}))
}
