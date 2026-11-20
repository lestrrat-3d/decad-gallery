package fastener_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func requireFullyConstrained(t *testing.T, s *sketch.Sketch) {
	t.Helper()
	require.NotNil(t, s)
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "status %s", report.Status)
	require.Zero(t, report.DOF)
	require.Zero(t, len(report.Redundant))
	require.Zero(t, len(report.Conflicts))
	require.NoError(t, report.Check())
}

// socketCapVolume integrates the blank's axial section, adds the underhead fillet, and removes the socket.
func socketCapVolume(b fastener.BoltSize, length float64) float64 {
	sc := b.SocketCap
	shankR, headR := b.ShankDiameter/2, sc.HeadDiameter/2
	chamferHeight := sc.HeadChamferWidth * math.Tan(sc.HeadChamferAngle*math.Pi/180)
	section := [][2]float64{
		{0, -length}, {shankR - b.TipChamfer, -length}, {shankR, -length + b.TipChamfer}, {shankR, 0},
		{headR, 0}, {headR, b.HeadHeight - chamferHeight}, {headR - sc.HeadChamferWidth, b.HeadHeight},
		{0, b.HeadHeight},
	}
	blank := 0.0
	for i := range section {
		a, c := section[i], section[(i+1)%len(section)]
		blank += (a[0]*a[0] + a[0]*c[0] + c[0]*c[0]) * (c[1] - a[1])
	}
	blank = math.Abs(blank) * math.Pi / 3
	// The fillet fills the corner between the shank and the underside: area r^2(1-pi/4) around radius shankR.
	r := sc.UnderheadRadius
	filletArea := r * r * (1 - math.Pi/4)
	fillet := 2 * math.Pi * (shankR + r*0.78) * filletArea
	// The round pilot well lies inside the hexagonal socket, so the socket is the hexagonal prism, the pilot below its
	// floor, and the cone.
	hexArea := math.Sqrt(3) / 2 * sc.SocketAcrossFlats * sc.SocketAcrossFlats
	baseR := sc.DrillPointBaseRadius
	coneHeight := baseR / math.Tan(sc.DrillPointAngle/2*math.Pi/180)
	cone := math.Pi * baseR * baseR * coneHeight / 3
	pilot := math.Pi * baseR * baseR * sc.PilotDepth
	return blank + fillet - hexArea*sc.SocketDepth - pilot - cone
}

// hexHeadVolume integrates the chamfered hexagonal head with the midpoint rule over the corner chamfer ring.
func hexHeadVolume(b fastener.BoltSize) float64 {
	const steps = 20000
	s, e := b.Hex.AcrossFlats, b.Hex.AcrossCorners()
	dw := b.Hex.BearingDiameter
	slope := math.Tan(fastener.HexChamferAngle * math.Pi / 180)
	depth := (e/2 - dw/2) * slope
	hexArea := math.Sqrt(3) / 2 * s * s
	removed := 0.0
	dz := depth / steps
	for i := 0; i < steps; i++ {
		z := (float64(i) + 0.5) * dz
		radius := dw/2 + (depth-z)/slope
		removed += (hexArea - hexSection(radius, s)) * dz
	}
	return hexArea*b.HeadHeight - 2*removed
}

// hexSection returns the area of the regular hexagon of across-flats s that lies inside the circle of radius r.
func hexSection(r, s float64) float64 {
	const steps = 4000
	apothem := s / 2
	// Integrate over one 30 degree wedge from a flat's midpoint to a corner, then multiply by twelve.
	area := 0.0
	dtheta := (math.Pi / 6) / steps
	for i := 0; i < steps; i++ {
		theta := (float64(i) + 0.5) * dtheta
		rho := math.Min(r, apothem/math.Cos(theta))
		area += 0.5 * rho * rho * dtheta
	}
	return 12 * area
}

func hexBoltVolume(b fastener.BoltSize, length float64) float64 {
	r, c := b.ShankDiameter/2, b.TipChamfer
	tipRing := 2 * math.Pi * (r - c/3) * (c * c / 2)
	return hexHeadVolume(b) + math.Pi*r*r*length - tipRing
}

func faceKinds(body *decad.Body) map[decad.SurfaceKind]int {
	counts := map[decad.SurfaceKind]int{}
	for _, f := range body.Faces() {
		counts[f.Surface().Kind()]++
	}
	return counts
}

func TestBoltPresets(t *testing.T) {
	for _, name := range fastener.Names() {
		preset := mustLookup(t, name)
		for _, length := range preset.Bolt.StandardLengths {
			doc := decad.New()
			b, err := fastener.BuildBolt(t.Context(), doc, fastener.BoltParams{Preset: name, Length: length})
			require.NoError(t, err, "%s x%g", name, length)
			require.Len(t, doc.Bodies(), 1)
			require.True(t, b.Body.IsSolid())
			for _, s := range b.Sketches() {
				requireFullyConstrained(t, s)
			}

			box, err := b.Body.Bounds()
			require.NoError(t, err)
			tol := box.Bound.Base() + 1e-8
			require.InDelta(t, -length, box.Min.Z, tol)
			require.InDelta(t, preset.Bolt.HeadHeight, box.Max.Z, tol)

			volume, err := b.Body.Volume()
			require.NoError(t, err)
			counts := faceKinds(b.Body)
			switch preset.Bolt.Head {
			case fastener.HeadSocketCap:
				require.Len(t, b.Sketches(), 2)
				require.Nil(t, b.Head)
				require.InDelta(t, socketCapVolume(preset.Bolt, length), volume.Value.Base(), volume.Bound.Base()+0.01)
				require.Equal(t, 10, counts[decad.KindPlane], "tip, underside, top, socket floor and six socket flats")
				require.Equal(t, 3, counts[decad.KindCylinder], "shank, head and pilot well")
				require.Equal(t, 2, counts[decad.KindCone], "tip chamfer and head chamfer")
				// Decad's boolean keeps the cone that closes the pilot well as a faceted face; the volume check bounds it.
				require.Equal(t, 1, counts[decad.KindFaceted], "drill point")
				require.Equal(t, 1, counts[decad.KindTorus], "underhead fillet")
				require.Len(t, b.Body.Faces(), 17)
			case fastener.HeadHex:
				require.Len(t, b.Sketches(), 2)
				require.Nil(t, b.Socket)
				require.InDelta(t, hexBoltVolume(preset.Bolt, length), volume.Value.Base(), volume.Bound.Base()+1e-3)
				require.Equal(t, 3, counts[decad.KindCone])
				require.Equal(t, 1, counts[decad.KindCylinder])
				require.Equal(t, len(b.Body.Faces()), counts[decad.KindPlane]+counts[decad.KindCone]+counts[decad.KindCylinder])
				e, s := preset.Bolt.Hex.AcrossCorners(), preset.Bolt.Hex.AcrossFlats
				require.InDelta(t, -e/2, box.Min.X, tol)
				require.InDelta(t, e/2, box.Max.X, tol)
				require.InDelta(t, -s/2, box.Min.Y, tol)
				require.InDelta(t, s/2, box.Max.Y, tol)
			}
		}
	}
}

func TestBoltSocketCapDefault(t *testing.T) {
	doc := decad.New()
	b, err := fastener.BuildBolt(t.Context(), doc, fastener.DefaultBoltParams())
	require.NoError(t, err)
	require.Equal(t, []*sketch.Sketch{b.Profile, b.Socket}, b.Sketches())
	box, err := b.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, -3.8/2, box.Min.X, tol)
	require.InDelta(t, 3.8/2, box.Max.X, tol)
	require.InDelta(t, -10, box.Min.Z, tol)
	require.InDelta(t, 2, box.Max.Z, tol)

	mesh, err := b.Body.Tessellate(t.Context(), units.Millimeters(.03), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestBoltHexHeadDefault(t *testing.T) {
	doc := decad.New()
	b, err := fastener.BuildBolt(t.Context(), doc, fastener.BoltParams{Preset: "M5", Length: 25})
	require.NoError(t, err)
	require.Equal(t, []*sketch.Sketch{b.Head, b.Profile}, b.Sketches())
	mesh, err := b.Body.Tessellate(t.Context(), units.Millimeters(.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
}

func TestBoltPlacement(t *testing.T) {
	doc := decad.New()
	first, err := fastener.BuildBolt(t.Context(), doc, fastener.DefaultBoltParams())
	require.NoError(t, err)
	// +Z of the bolt frame maps to world +X: the head sits at x in [10, 12] and the shank runs to x = 0.
	frame, err := r3.NewFrame(r3.NewVec(10, 5, 0), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	placed, err := fastener.BuildBolt(t.Context(), doc, fastener.BoltParams{Preset: "M2", Length: 10, Placement: &frame})
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 2)
	box, err := placed.Body.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-8
	require.InDelta(t, 0, box.Min.X, tol)
	require.InDelta(t, 12, box.Max.X, tol)
	require.InDelta(t, 5-1.9, box.Min.Y, tol)
	require.InDelta(t, 5+1.9, box.Max.Y, tol)
	_, err = first.Body.Bounds()
	require.NoError(t, err, "the first bolt is untouched")
	requireFullyConstrained(t, placed.Profile)

	// Local +Z maps to world +X, local +X to world +Y, origin at (10, 20, 30).
	m5 := mustLookup(t, "M5").Bolt
	frame, err = r3.NewFrame(r3.NewVec(10, 20, 30), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	hex, err := fastener.BuildBolt(t.Context(), doc, fastener.BoltParams{Preset: "M5", Length: 25, Placement: &frame})
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 3)
	box, err = hex.Body.Bounds()
	require.NoError(t, err)
	tol = box.Bound.Base() + 1e-8
	e, s := m5.Hex.AcrossCorners(), m5.Hex.AcrossFlats
	require.InDelta(t, 10-25.0, box.Min.X, tol)
	require.InDelta(t, 10+m5.HeadHeight, box.Max.X, tol)
	require.InDelta(t, 20-e/2, box.Min.Y, tol)
	require.InDelta(t, 20+e/2, box.Max.Y, tol)
	require.InDelta(t, 30-s/2, box.Min.Z, tol)
	require.InDelta(t, 30+s/2, box.Max.Z, tol)
}

func TestBoltRejectsInvalidInputs(t *testing.T) {
	for _, item := range []struct {
		name string
		p    fastener.BoltParams
	}{
		{"zero", fastener.BoltParams{}},
		{"empty-preset", fastener.BoltParams{Length: 10}},
		{"unknown-preset", fastener.BoltParams{Preset: "M3", Length: 10}},
		{"negative", fastener.BoltParams{Preset: "M2", Length: -8}},
		{"between-standard-lengths", fastener.BoltParams{Preset: "M2", Length: 8}},
		{"too-short", fastener.BoltParams{Preset: "M2", Length: 5.9}},
		{"too-long", fastener.BoltParams{Preset: "M2", Length: 25.1}},
		{"m5-off-list", fastener.BoltParams{Preset: "M5", Length: 22}},
		{"m2-length-on-m5", fastener.BoltParams{Preset: "M5", Length: 10}},
		{"nan", fastener.BoltParams{Preset: "M2", Length: math.NaN()}},
		{"inf", fastener.BoltParams{Preset: "M5", Length: math.Inf(1)}},
		{"invalid-frame", fastener.BoltParams{Preset: "M2", Length: 10, Placement: &r3.Frame{}}},
		{"invalid-frame-m5", fastener.BoltParams{Preset: "M5", Length: 25, Placement: &r3.Frame{}}},
	} {
		t.Run(item.name, func(t *testing.T) {
			doc := decad.New()
			_, err := fastener.BuildBolt(t.Context(), doc, item.p)
			require.Error(t, err)
			require.Zero(t, len(doc.Bodies()))
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err := fastener.BuildBolt(ctx, doc, fastener.DefaultBoltParams())
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, len(doc.Bodies()))
	_, err = fastener.BuildBolt(t.Context(), nil, fastener.DefaultBoltParams())
	require.ErrorContains(t, err, "document")
	_, err = fastener.BuildBolt(nil, doc, fastener.DefaultBoltParams()) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
