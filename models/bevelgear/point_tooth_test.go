package bevelgear_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/bevelgear"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPointToothFromRealBackConeProfile(t *testing.T) {
	for _, tc := range []struct {
		name             string
		teeth, mateTeeth int
		spacing          float64
		shaftAngle       float64
	}{
		{name: "eight equal", teeth: 8, mateTeeth: 8},
		{name: "sixteen twelve spaced", teeth: 16, mateTeeth: 12, spacing: 0.5},
		{name: "sixteen twelve oblique", teeth: 16, mateTeeth: 12, shaftAngle: 70 * math.Pi / 180},
		{name: "sixty embedded", teeth: 60, mateTeeth: 60},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := spiralDefaults()
			p.Teeth, p.MateTeeth, p.Samples = tc.teeth, tc.mateTeeth, 5
			p.ToothSpacing = tc.spacing
			if tc.shaftAngle > 0 {
				p.ShaftAngle = tc.shaftAngle
			}
			doc := decad.New()
			tooth, err := bevelgear.BuildPointTooth(t.Context(), doc, p)
			require.NoError(t, err)
			require.True(t, tooth.IsSolid())
			require.Len(t, doc.Bodies(), 1)
			volume, err := tooth.Volume()
			require.NoError(t, err)
			require.Greater(t, volume.Value.Base(), volume.Bound.Base())
			mesh, err := tooth.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
			require.Equal(t, 2, meshEuler(mesh))
			foundApex := false
			for _, vertex := range mesh.Vertices() {
				if vertex.X == 0 && vertex.Y == 0 && vertex.Z == 0 {
					foundApex = true
					break
				}
			}
			require.True(t, foundApex, "the real loft must contain the exact world apex")
		})
	}
}
