package m2bolt_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/sketch"
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
