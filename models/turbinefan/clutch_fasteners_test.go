package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// earPlane is the plane through the shaft axis and an ear bolt's line, U along the line outward and V along +Z.
func earPlane(t *testing.T, u r3.Vec) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{}, u, r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}

// TestTurbineFanClutchEarBolts checks each ear bolt: its head is flush with the flange's front face and inside its
// pocket, its tip clears the clamp ring's back face, its nut lies inside the clamp ring's slot, which is open to the rim
// and closed above and below, and neither bolt nor nut shares volume with the hub or the clamp ring.
func TestTurbineFanClutchEarBolts(t *testing.T) {
	require.Contains(t, m2bolt.StandardLengths(), float64(turbinefan.EarBoltLength))
	require.GreaterOrEqual(t, turbinefan.FlangeThickness-m2fastener.HeadPocketDepth, 1.0)
	require.GreaterOrEqual(t, turbinefan.ClampThickness-m2fastener.NutCavityHeight-1, 1.0)
	require.GreaterOrEqual(t, onewayclutch.EarCircle-m2fastener.NutCavityRadius-turbinefan.ClampInnerRadius, 1.0)
	require.GreaterOrEqual(t, turbinefan.FlangeRadius-turbinefan.ChamferSize-onewayclutch.EarCircle-m2fastener.HeadPocketRadius, 1.0)
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, _ *decad.Document) {
		for k := 1; k <= onewayclutch.Ears; k++ {
			t.Run(fmt.Sprintf("ear-%d", k), func(t *testing.T) {
				site := m.EarSites[k-1]
				u := site.Corner
				v := r3.Vec{X: -u.Y, Y: u.X}
				line := site.Head
				bolt, nut := m.EarBolts[k-1].Body, m.EarNuts[k-1].Body
				bb, err := bolt.Bounds()
				require.NoError(t, err)
				tol := bb.Bound.Base() + 2e-3
				require.InDelta(t, 0, bb.Max.Z, tol, "head top flush with the flange's front face")
				require.InDelta(t, -27, bb.Min.Z, tol, "tip")
				require.Greater(t, bb.Min.Z, float64(turbinefan.ClampBack))
				require.GreaterOrEqual(t, bb.Min.Z-(turbinefan.ClampBack+2*turbinefan.ClutchSeam), 1.0-tol)
				for _, d := range []float64{bb.Min.X - line.X, bb.Max.X - line.X, bb.Min.Y - line.Y, bb.Max.Y - line.Y} {
					require.Less(t, math.Abs(d), m2fastener.HeadPocketRadius, "head inside its pocket")
				}
				nb, err := nut.Bounds()
				require.NoError(t, err)
				ntol := nb.Bound.Base() + 2e-3
				require.GreaterOrEqual(t, nb.Min.Z, -27.3-ntol)
				require.LessOrEqual(t, nb.Max.Z, -25.7+ntol)
				for _, w := range meshOf(t, nut, 0.01).Vertices() {
					d := w.Sub(line)
					require.LessOrEqual(t, math.Abs(d.Dot(v)), flatsHalf+ntol, "nut flats inside the slot's sides")
					require.LessOrEqual(t, math.Abs(d.Dot(u)), m2fastener.NutCavityRadius+ntol, "nut corners inside the slot")
				}
				xy := func(w r3.Vec) (float64, float64) { return w.X, w.Y }
				mid := section(t, m.Clamp, atZ(t, -26.5))
				at := func(p r3.Vec) bool { return inSection(mid, xy, p.X, p.Y) }
				require.False(t, at(line), "the slot at the bolt's line")
				require.False(t, at(u.Scale(turbinefan.ClampOuterRadius+0.2)), "the slot is open to the rim")
				require.True(t, at(line.Add(v.Scale(flatsHalf+0.4))), "clamp beside the slot")
				require.True(t, at(line.Sub(v.Scale(flatsHalf+0.4))), "clamp beside the slot")
				require.True(t, at(u.Scale(onewayclutch.EarCircle-m2fastener.NutCavityRadius-0.4)), "clamp beyond the slot's end")
				for _, z := range []float64{-28.0, -25.0} {
					level := section(t, m.Clamp, atZ(t, z))
					require.True(t, inSection(level, xy, line.X+2*u.X, line.Y+2*u.Y), "clamp under and over the slot at z %g", z)
					require.False(t, inSection(level, xy, line.X, line.Y), "the bolt's hole at z %g", z)
				}
				frames := map[string]r3.Frame{"axis plane": earPlane(t, u)}
				for _, z := range []float64{-1, -26.5, -28} {
					frames[fmt.Sprintf("z %g", z)] = atZ(t, z)
				}
				for name, f := range frames {
					requireNoOverlap(t, m.Hub, f, fmt.Sprintf("ear fastener %d with the hub at %s", k, name), bolt, nut)
					requireNoOverlap(t, m.Clamp, f, fmt.Sprintf("ear fastener %d with the clamp at %s", k, name), bolt, nut)
				}
			})
		}
	})
}
