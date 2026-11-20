package fastener_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// nutVolume is the hexagonal prism, less the bore and the two chamfers. Each chamfer removes, at depth z
// from its face, the part of the hexagon outside the circle of radius dw/2 + z/tan(30 degrees).
func nutVolume(n fastener.NutSize) float64 {
	hexArea := math.Sqrt(3) / 2 * n.AcrossFlats * n.AcrossFlats
	bore := math.Pi * n.BoreDiameter * n.BoreDiameter / 4 * n.Thickness
	slope := math.Tan(fastener.HexChamferAngle * math.Pi / 180)
	bearingR := n.BearingDiameter / 2
	depth := (n.AcrossCorners()/2 - bearingR) * slope
	const steps = 20000
	chamfer := 0.0
	for i := range steps {
		z := (float64(i) + .5) / steps * depth
		chamfer += (hexArea - clippedHexArea(n, bearingR+z/slope)) * depth / steps
	}
	return hexArea*n.Thickness - bore - 2*chamfer
}

// clippedHexArea is the area of the nut's hexagon inside the circle of radius rho about its center.
func clippedHexArea(n fastener.NutSize, rho float64) float64 {
	apothem := n.AcrossFlats / 2
	if rho <= apothem {
		return math.Pi * rho * rho
	}
	if rho >= n.AcrossCorners()/2 {
		return math.Sqrt(3) / 2 * n.AcrossFlats * n.AcrossFlats
	}
	// Each of the six flats cuts a circular segment off the circle.
	half := math.Acos(apothem / rho)
	segment := rho * rho * (half - math.Sin(half)*math.Cos(half))
	return math.Pi*rho*rho - 6*segment
}

func TestNutPresets(t *testing.T) {
	for _, name := range fastener.Names() {
		n := mustLookup(t, name).Nut
		doc := decad.New()
		m, err := fastener.BuildNut(t.Context(), doc, fastener.NutParams{Preset: name})
		require.NoError(t, err, name)
		require.Len(t, doc.Bodies(), 1)
		require.True(t, m.Body.IsSolid())
		require.Len(t, m.Sketches(), 3)
		for _, s := range m.Sketches() {
			requireFullyConstrained(t, s)
		}

		box, err := m.Body.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-8
		require.InDelta(t, -n.AcrossCorners()/2, box.Min.X, tol, "corners lie on the X axis")
		require.InDelta(t, n.AcrossCorners()/2, box.Max.X, tol)
		require.InDelta(t, -n.AcrossFlats/2, box.Min.Y, tol, "flats face Y")
		require.InDelta(t, n.AcrossFlats/2, box.Max.Y, tol)
		require.InDelta(t, 0, box.Min.Z, tol)
		require.InDelta(t, n.Thickness, box.Max.Z, tol)

		volume, err := m.Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, nutVolume(n), volume.Value.Base(), volume.Bound.Base()+1e-3, name)

		counts := faceKinds(m.Body)
		require.Equal(t, 8, counts[decad.KindPlane], "six flats and two bearing faces")
		require.Equal(t, 2, counts[decad.KindCone], "two chamfers")
		require.Equal(t, 1, counts[decad.KindCylinder], "bore")
		require.Len(t, m.Body.Faces(), 11)

		mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		require.True(t, mesh.BoundaryVerified())
		require.True(t, mesh.VolumeVerified())
	}
}

func TestNutFitsBolt(t *testing.T) {
	for _, name := range fastener.Names() {
		p := mustLookup(t, name)
		require.Equal(t, p.Bolt.ShankDiameter, p.Nut.BoreDiameter, name)
	}
}

func TestNutPlacement(t *testing.T) {
	for _, name := range fastener.Names() {
		n := mustLookup(t, name).Nut
		doc := decad.New()
		first, err := fastener.BuildNut(t.Context(), doc, fastener.NutParams{Preset: name})
		require.NoError(t, err)
		// The nut's +Z maps to world +X with its corners along Z.
		frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
		require.NoError(t, err)
		placed, err := fastener.BuildNut(t.Context(), doc, fastener.NutParams{Preset: name, Placement: &frame})
		require.NoError(t, err)
		require.Len(t, doc.Bodies(), 2)
		box, err := placed.Body.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-8
		require.InDelta(t, 10, box.Min.X, tol)
		require.InDelta(t, 10+n.Thickness, box.Max.X, tol)
		require.InDelta(t, 5-n.AcrossCorners()/2, box.Min.Y, tol)
		require.InDelta(t, 5+n.AcrossCorners()/2, box.Max.Y, tol)
		_, err = first.Body.Bounds()
		require.NoError(t, err, "the first nut is untouched")
		requireFullyConstrained(t, placed.Hexagon)
	}
}

func TestNutRejectsInvalidInputs(t *testing.T) {
	for name, p := range map[string]fastener.NutParams{
		"zero":    {},
		"unknown": {Preset: "M3"},
		"frame":   {Preset: "M2", Placement: &r3.Frame{}},
	} {
		doc := decad.New()
		_, err := fastener.BuildNut(t.Context(), doc, p)
		require.Error(t, err, name)
		require.Zero(t, len(doc.Bodies()), name)
	}
	doc := decad.New()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := fastener.BuildNut(ctx, doc, fastener.DefaultNutParams())
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, len(doc.Bodies()))
	_, err = fastener.BuildNut(t.Context(), nil, fastener.DefaultNutParams())
	require.ErrorContains(t, err, "document")
	_, err = fastener.BuildNut(nil, doc, fastener.DefaultNutParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
