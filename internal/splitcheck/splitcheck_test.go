package splitcheck_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// The test shafts run along +Y through the origin: radius 4, 40 long, with a flat 3.25 from the axis toward +Z.
const (
	radius     = 4.0
	flatOffset = 3.25
	length     = 40.0
	tolerance  = 0.01
)

var axisY = r3.Vec{Y: 1}

type shafts struct {
	t   *testing.T
	doc *decad.Document
}

func newShafts(t *testing.T) shafts { return shafts{t: t, doc: decad.New()} }

// frame returns the frame at origin with axes u and v; bodies extrude along u × v.
func (b shafts) frame(origin, u, v r3.Vec) r3.Frame {
	f, err := r3.NewFrame(origin, u, v)
	require.NoError(b.t, err)
	return f
}

// across is the plane at Y = y with U = +X and V = −Z, so bodies extrude along +Y.
func (b shafts) across(y float64) r3.Frame {
	return b.frame(r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: -1})
}

func (b shafts) sketch(f r3.Frame) *sketch.Sketch {
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(f)
	require.NoError(b.t, err)
	s, err := world.CreateSketch(plane)
	require.NoError(b.t, err)
	return s
}

func (b shafts) extrude(s *sketch.Sketch, d float64) *decad.Body {
	_, err := s.Solve(b.t.Context())
	require.NoError(b.t, err)
	require.NoError(b.t, sketchcheck.VerifyFullyConstrained(b.t.Context(), s))
	var chosen *sketch.Profile
	for _, p := range s.Profiles() {
		if p.Valid && len(p.Holes) == 0 {
			require.Nil(b.t, chosen, "one profile")
			chosen = p
		}
	}
	require.NotNil(b.t, chosen)
	body, err := decad.Extrude(b.t.Context(), b.doc, s, chosen, decad.Distance{D: units.Millimeters(d), Dir: decad.Along})
	require.NoError(b.t, err)
	return body
}

// cylinder extrudes a disc of radius r centred on f's origin for d along f's normal.
func (b shafts) cylinder(f r3.Frame, r, d float64) *decad.Body {
	s := b.sketch(f)
	s.AddConstraint(sketch.NewRadius(s.CreateCircle(s.Origin(), r), r))
	return b.extrude(s, d)
}

// box extrudes the rectangle u0..u1 by v0..v1 of f for d along f's normal. No edge may pass through the origin.
func (b shafts) box(f r3.Frame, u0, v0, u1, v1, d float64) *decad.Body {
	s := b.sketch(f)
	r := s.CreateRectangle(u0, v0, u1, v1)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), r.A, u0),
		sketch.NewVerticalDistance(s.Origin(), r.A, v0),
		sketch.NewHorizontalDistance(r.A, r.B, u1-u0),
		sketch.NewVerticalDistance(r.B, r.C, v1-v0),
	)
	return b.extrude(s, d)
}

func (b shafts) cut(host, tool *decad.Body) *decad.Body {
	body, err := decad.Cut(b.t.Context(), host, tool)
	require.NoError(b.t, err)
	return body
}

// dShaft builds the radius 4 shaft with its flat toward +Z (v < −3.25 in the across frame) over its whole length.
func (b shafts) dShaft() *decad.Body {
	round := b.cylinder(b.across(0), radius, length)
	return b.cut(round, b.box(b.across(-1), -5, -(flatOffset+3), 5, -flatOffset, length+2))
}

func (b shafts) check(body *decad.Body, reference r3.Vec) splitcheck.Report {
	r, err := splitcheck.Check(b.t.Context(), splitcheck.Shaft{Body: body, Axis: axisY, Reference: reference}, tolerance)
	require.NoError(b.t, err)
	return r
}

func requireViolation(t *testing.T, r splitcheck.Report, substring string) {
	t.Helper()
	for _, v := range r.Violations {
		if strings.Contains(v, substring) {
			return
		}
	}
	require.Failf(t, "missing violation", "want one containing %q, got %q", substring, r.Violations)
}

func TestCompliantShafts(t *testing.T) {
	t.Run("d_shaft_with_collar", func(t *testing.T) {
		b := newShafts(t)
		shaft := b.dShaft()
		body, err := decad.Union(t.Context(), shaft, b.cylinder(b.across(10), 5.5, 3))
		require.NoError(t, err)
		r := b.check(body, r3.Vec{})
		require.Empty(t, r.Violations)
		require.Equal(t, 1, r.Flats)
		require.InDelta(t, 1, math.Abs(r.Normal.X), 1e-9, "the split plane holds the axis and the flat's normal")
		require.LessOrEqual(t, r.Asymmetry, r.AsymmetryLimit)
		require.InDelta(t, 5.5, r.HalfHeights[0], tolerance)
		require.InDelta(t, 5.5, r.HalfHeights[1], tolerance)
		// A half-round faces nowhere downward. Only chords that cross the split plane at the bed seam lean, by half
		// their chord angle.
		require.Less(t, r.Overhang, 5.0)
	})
	t.Run("cross_hole_across_the_split_plane", func(t *testing.T) {
		b := newShafts(t)
		// The hole runs along X, normal to the split plane, so each half has a vertical hole.
		hole := b.cylinder(b.frame(r3.Vec{X: -5, Y: 20}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 1, 10)
		r := b.check(b.cut(b.dShaft(), hole), r3.Vec{})
		require.Empty(t, r.Violations)
	})
	t.Run("cross_hole_in_the_split_plane_bridges", func(t *testing.T) {
		b := newShafts(t)
		// The hole runs along Z, inside the split plane, so each half has a groove whose roof the printer bridges.
		hole := b.cylinder(b.frame(r3.Vec{Y: 20, Z: -5}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 1, 10)
		r := b.check(b.cut(b.dShaft(), hole), r3.Vec{})
		require.Empty(t, r.Violations)
		require.Positive(t, r.Bridges)
		require.InDelta(t, 2*math.Sin(math.Pi/4), r.Bridge, 0.05, "the roof's part steeper than 45 degrees")
	})
	t.Run("explicit_normal_and_exempt_bolt", func(t *testing.T) {
		b := newShafts(t)
		// A round shaft split by the plane Z = 0, named by Normal. A blind pocket from +Z reaches into the +Z half
		// only, round the line X = 0, Y = 20 along Z: the halves differ there and nowhere else.
		pocket := b.cylinder(b.frame(r3.Vec{Y: 20, Z: 2}, r3.Vec{X: 1}, r3.Vec{Y: 1}), 1.5, 3)
		body := b.cut(b.cylinder(b.across(0), radius, length), pocket)
		shaft := splitcheck.Shaft{Body: body, Axis: axisY, Normal: r3.Vec{Z: 1}}
		r, err := splitcheck.Check(t.Context(), shaft, tolerance)
		require.NoError(t, err)
		requireViolation(t, r, "mirror image")
		shaft.Exempt = []splitcheck.Line{{Point: r3.Vec{Y: 20}, Dir: r3.Vec{Z: 1}, Radius: 1.6}}
		r, err = splitcheck.Check(t.Context(), shaft, tolerance)
		require.NoError(t, err)
		require.Empty(t, r.Violations)
		require.Positive(t, r.Exempted)
		require.InDelta(t, 1, r.Normal.Z, 1e-12)
	})
	t.Run("round_shaft_with_reference", func(t *testing.T) {
		b := newShafts(t)
		r := b.check(b.cylinder(b.across(0), radius, length), r3.Vec{Z: 1})
		require.Empty(t, r.Violations)
		require.Zero(t, r.Flats)
		require.InDelta(t, 1, math.Abs(r.Normal.X), 1e-9)
	})
}

func TestShaftViolations(t *testing.T) {
	t.Run("round_shaft_without_reference", func(t *testing.T) {
		b := newShafts(t)
		requireViolation(t, b.check(b.cylinder(b.across(0), radius, length), r3.Vec{}), "no flat")
	})
	t.Run("axial_bore_too_wide_to_bridge", func(t *testing.T) {
		b := newShafts(t)
		// A radius 6 shaft with a radius 4.5 bore along its axis: each half has a groove on its cut face whose roof,
		// steeper than 45 degrees, is 6.36 mm across.
		shaft := b.cut(b.cylinder(b.across(0), 6, length), b.cylinder(b.across(-1), 4.5, length+2))
		r := b.check(shaft, r3.Vec{Z: 1})
		requireViolation(t, r, "wider than")
		require.LessOrEqual(t, r.Asymmetry, r.AsymmetryLimit, "the bore is symmetric; only its roof fails")
	})
	t.Run("second_flat_turned", func(t *testing.T) {
		b := newShafts(t)
		// A second flat toward +X on the first 8 mm: no one plane cuts both flats in half.
		r := b.check(b.cut(b.dShaft(), b.box(b.across(-1), flatOffset, -5, flatOffset+3, 5, 9)), r3.Vec{})
		requireViolation(t, r, "not parallel")
		reported := 0
		for _, v := range r.Violations {
			if strings.Contains(v, "not parallel") {
				reported++
			}
		}
		require.Equal(t, 1, reported, "each stray flat direction is reported once")
	})
	t.Run("off_centre_notch", func(t *testing.T) {
		b := newShafts(t)
		// A notch on the +X side only breaks the mirror symmetry.
		r := b.check(b.cut(b.dShaft(), b.box(b.across(15), 3, -1, 5, 1, 5)), r3.Vec{})
		requireViolation(t, r, "mirror image")
	})
	t.Run("thin_pin", func(t *testing.T) {
		b := newShafts(t)
		r := b.check(b.cylinder(b.across(0), 1.5, length), r3.Vec{Z: 1})
		requireViolation(t, r, "above its split face")
	})
}

func TestCheckRejectsInvalidInput(t *testing.T) {
	_, err := splitcheck.Check(t.Context(), splitcheck.Shaft{Axis: axisY}, tolerance)
	require.ErrorContains(t, err, "nil body")
	b := newShafts(t)
	_, err = splitcheck.Check(t.Context(), splitcheck.Shaft{Body: b.dShaft()}, tolerance)
	require.ErrorContains(t, err, "zero axis")
	_, err = splitcheck.Check(t.Context(), splitcheck.Shaft{Body: b.dShaft(), Axis: axisY, Normal: axisY}, tolerance)
	require.ErrorContains(t, err, "normal to the axis")
	_, err = splitcheck.Check(t.Context(), splitcheck.Shaft{Body: b.dShaft(), Axis: axisY,
		Exempt: []splitcheck.Line{{Dir: r3.Vec{Z: 1}}}}, tolerance)
	require.ErrorContains(t, err, "positive radius")
}
