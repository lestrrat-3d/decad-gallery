package fastener_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/fastener"
	"github.com/lestrrat-3d/sketch"
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

func mustLookup(t *testing.T, name string) fastener.Preset {
	t.Helper()
	p, ok := fastener.Lookup(name)
	require.True(t, ok, name)
	return p
}
