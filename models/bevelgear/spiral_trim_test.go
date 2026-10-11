package bevelgear

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestSourceDefaultSpiralToothConicalTrim(t *testing.T) {
	for _, member := range []struct {
		name   string
		pinion bool
	}{
		{name: "driving"},
		{name: "pinion", pinion: true},
	} {
		t.Run(member.name, func(t *testing.T) {
			p := DefaultParams()
			p.SpiralAngle = 35 * math.Pi / 180
			p.SpiralHand = "right"
			doc := decad.New()
			curved, err := buildSpiralTooth(t.Context(), doc, p, member.pinion)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1, "the loft removes its eight slab scaffolds")
			trimmed, err := trimSpiralTooth(t.Context(), doc, p, curved)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1)
			require.True(t, trimmed.IsSolid())
			mesh, err := trimmed.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}

func TestSourceDefaultSpiralToothToeCut(t *testing.T) {
	for _, member := range []struct {
		name   string
		pinion bool
	}{
		{name: "driving"},
		{name: "pinion", pinion: true},
	} {
		t.Run(member.name, func(t *testing.T) {
			p := DefaultParams()
			p.SpiralAngle = 35 * math.Pi / 180
			p.SpiralHand = "right"
			doc := decad.New()
			curved, err := buildSpiralTooth(t.Context(), doc, p, member.pinion)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1)
			uncutBounds, err := curved.Bounds()
			require.NoError(t, err)
			geometry := p
			geometry.SpiralAngle = 0
			geometry.BoreDiameter = 0
			toeCone, err := BuildToeCone(t.Context(), doc, geometry)
			require.NoError(t, err)
			toeKeeper, err := decad.Cut(t.Context(), curved, toeCone)
			require.NoError(t, err)
			require.Len(t, doc.Bodies(), 1)
			require.True(t, toeKeeper.IsSolid())
			cutBounds, err := toeKeeper.Bounds()
			require.NoError(t, err)
			require.Greater(t, cutBounds.Min.X-uncutBounds.Min.X,
				uncutBounds.Bound.Base()+cutBounds.Bound.Base(),
				"the toe cone must remove the tooth's apex-side tip")
			mesh, err := toeKeeper.Tessellate(t.Context(), units.Millimeters(0.1),
				decad.WithVerification(decad.VerifyAll))
			require.NoError(t, err)
			require.True(t, mesh.BoundaryVerified())
			require.True(t, mesh.VolumeVerified())
		})
	}
}
