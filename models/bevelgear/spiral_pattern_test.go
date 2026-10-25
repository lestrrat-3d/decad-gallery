package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralToothFullPattern(t *testing.T) {
	for _, member := range []struct {
		name   string
		pinion bool
	}{{name: "driving"}, {name: "pinion", pinion: true}} {
		t.Run(member.name, func(t *testing.T) {
			p := spiralDefaults()
			p.SpiralAngle = 35 * math.Pi / 180
			p.SpiralHand = "right"
			doc := decad.New()
			curved, err := buildSpiralTooth(t.Context(), doc, p, member.pinion)
			require.NoError(t, err)
			trimmed, err := trimSpiralTooth(t.Context(), doc, p, curved)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1)
			pattern, err := patternSpiralTooth(t.Context(), trimmed, p.Teeth)
			require.NoError(t, err)
			require.Len(t, pattern, 31)
			require.Len(t, doc.Bodies(), 31)
			for i, tooth := range pattern {
				require.True(t, tooth.IsSolid(), "tooth %d", i+1)
				mesh, err := tooth.Tessellate(t.Context(), units.Millimeters(0.1),
					decad.WithVerification(decad.VerifyAll))
				require.NoError(t, err, "tooth %d", i+1)
				require.True(t, mesh.BoundaryVerified(), "tooth %d", i+1)
				require.True(t, mesh.VolumeVerified(), "tooth %d", i+1)
			}
		})
	}
}
