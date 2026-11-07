package m2bolt_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// expectedVolume integrates the blank's axial section, adds the underhead fillet, and removes the socket.
func expectedVolume(length float64) float64 {
	shankR, headR := m2bolt.ShankDiameter/2, m2bolt.HeadDiameter/2
	chamferHeight := m2bolt.HeadChamferWidth * math.Tan(m2bolt.HeadChamferAngle*math.Pi/180)
	section := [][2]float64{
		{0, -length}, {shankR - m2bolt.TipChamfer, -length}, {shankR, -length + m2bolt.TipChamfer}, {shankR, 0},
		{headR, 0}, {headR, m2bolt.HeadHeight - chamferHeight}, {headR - m2bolt.HeadChamferWidth, m2bolt.HeadHeight},
		{0, m2bolt.HeadHeight},
	}
	blank := 0.0
	for i := range section {
		a, b := section[i], section[(i+1)%len(section)]
		blank += (a[0]*a[0] + a[0]*b[0] + b[0]*b[0]) * (b[1] - a[1])
	}
	blank = math.Abs(blank) * math.Pi / 3
	// The fillet fills the corner between the shank and the underside: area r^2(1-pi/4) around radius shankR.
	r := m2bolt.UnderheadRadius
	filletArea := r * r * (1 - math.Pi/4)
	fillet := 2 * math.Pi * (shankR + r*0.78) * filletArea
	// The round pilot well lies inside the hexagonal socket, so the socket is the hexagonal prism, the pilot below its floor, and the cone.
	hexArea := math.Sqrt(3) / 2 * m2bolt.SocketAcrossFlats * m2bolt.SocketAcrossFlats
	coneHeight := m2bolt.DrillPointBaseRadius / math.Tan(m2bolt.DrillPointAngle/2*math.Pi/180)
	cone := math.Pi * m2bolt.DrillPointBaseRadius * m2bolt.DrillPointBaseRadius * coneHeight / 3
	pilot := math.Pi * m2bolt.DrillPointBaseRadius * m2bolt.DrillPointBaseRadius * m2bolt.PilotDepth
	return blank + fillet - hexArea*m2bolt.SocketDepth - pilot - cone
}

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}

func TestM2BoltDefault(t *testing.T) {
	doc := decad.New()
	m, err := m2bolt.Build(t.Context(), doc, m2bolt.DefaultParams())
	require.NoError(t, err)
	require.Equal(t, 1, len(doc.Bodies()))
	require.True(t, m.Body.IsSolid())
	for _, s := range []*sketch.Sketch{m.Profile, m.Socket} {
		requireFullyConstrained(t, s)
	}

	box, err := m.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, -m2bolt.HeadDiameter/2, box.Min.X, tol)
	require.InDelta(t, m2bolt.HeadDiameter/2, box.Max.X, tol)
	require.InDelta(t, -8, box.Min.Z, tol)
	require.InDelta(t, m2bolt.HeadHeight, box.Max.Z, tol)

	volume, err := m.Body.Volume()
	require.NoError(t, err)
	require.InDelta(t, expectedVolume(8), volume.Value.Base(), volume.Bound.Base()+0.01)

	counts := map[decad.SurfaceKind]int{}
	for _, face := range m.Body.Faces() {
		counts[face.Surface().Kind()]++
	}
	require.Equal(t, 10, counts[decad.KindPlane], "tip, underside, top, socket floor and six socket flats")
	require.Equal(t, 3, counts[decad.KindCylinder], "shank, head and pilot well")
	require.Equal(t, 2, counts[decad.KindCone], "tip chamfer and head chamfer")
	// Decad's boolean keeps the cone that closes the pilot well as a faceted face; the cut of the socket
	// converts it. The revolved profile holds it exactly, and the volume check above bounds the difference.
	require.Equal(t, 1, counts[decad.KindFaceted], "drill point")
	require.Equal(t, 1, counts[decad.KindTorus], "underhead fillet")
	require.Equal(t, 17, len(m.Body.Faces()))

	mesh, err := m.Body.Tessellate(t.Context(), units.Millimeters(.03), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestM2BoltLengths(t *testing.T) {
	for _, length := range []float64{m2bolt.MinLength, 5, 12.5, 30, m2bolt.MaxLength} {
		m, err := m2bolt.Build(t.Context(), decad.New(), m2bolt.Params{Length: length})
		require.NoError(t, err, "length %g", length)
		box, err := m.Body.Bounds()
		require.NoError(t, err)
		require.InDelta(t, -length, box.Min.Z, box.Bound.Base()+1e-8)
		volume, err := m.Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, expectedVolume(length), volume.Value.Base(), volume.Bound.Base()+0.01, "length %g", length)
		require.Equal(t, 17, len(m.Body.Faces()))
		requireFullyConstrained(t, m.Profile)
	}
}

func TestM2BoltPlacement(t *testing.T) {
	doc := decad.New()
	first, err := m2bolt.Build(t.Context(), doc, m2bolt.DefaultParams())
	require.NoError(t, err)
	// +Z of the bolt frame maps to world +X: the head sits at x in [10, 12] and the shank runs to x = 2.
	frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	placed, err := m2bolt.Build(t.Context(), doc, m2bolt.Params{Length: 8, Placement: &frame})
	require.NoError(t, err)
	require.Equal(t, 2, len(doc.Bodies()))
	box, err := placed.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, 2, box.Min.X, tol)
	require.InDelta(t, 12, box.Max.X, tol)
	require.InDelta(t, 5-1.9, box.Min.Y, tol)
	require.InDelta(t, 5+1.9, box.Max.Y, tol)
	_, err = first.Body.Bounds()
	require.NoError(t, err, "the first bolt is untouched")
	requireFullyConstrained(t, placed.Profile)
}

func TestM2BoltRejectsInvalidInputs(t *testing.T) {
	for _, item := range []struct {
		name string
		p    m2bolt.Params
	}{
		{"zero", m2bolt.Params{}},
		{"negative", m2bolt.Params{Length: -8}},
		{"too-short", m2bolt.Params{Length: m2bolt.MinLength - .1}},
		{"too-long", m2bolt.Params{Length: m2bolt.MaxLength + .1}},
		{"nan", m2bolt.Params{Length: math.NaN()}},
		{"inf", m2bolt.Params{Length: math.Inf(1)}},
		{"invalid-frame", m2bolt.Params{Length: 8, Placement: &r3.Frame{}}},
	} {
		t.Run(item.name, func(t *testing.T) {
			doc := decad.New()
			_, err := m2bolt.Build(t.Context(), doc, item.p)
			require.Error(t, err)
			require.Zero(t, len(doc.Bodies()))
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err := m2bolt.Build(ctx, doc, m2bolt.DefaultParams())
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, len(doc.Bodies()))
	_, err = m2bolt.Build(t.Context(), nil, m2bolt.DefaultParams())
	require.ErrorContains(t, err, "document")
	_, err = m2bolt.Build(nil, doc, m2bolt.DefaultParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
