package turbinefan_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// sitePlane is the plane through the shaft axis and a site's line, U along the line outward and V along +Z.
func sitePlane(t *testing.T, u r3.Vec) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{}, u, r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}

// TestTurbineFanClutchAdapterBolts checks each adapter bolt in the fan's local frame: an M2 x 10 whose head's top face is
// flush with the adapter's rear face (z -6.25), whose tip stays inside the hub's blind hole, whose nut lies inside the
// hub's slot, which is open to the round face and closed above and below, and neither bolt nor nut shares volume with the
// hub.
func TestTurbineFanClutchAdapterBolts(t *testing.T) {
	require.Contains(t, m2fastener.BoltSize.StandardLengths, float64(onewayclutch.FaceLength))
	eachClutchCase(t, func(t *testing.T, _ fanCase, m *turbinefan.Model, _ *decad.Document) {
		for k := 1; k <= 2; k++ {
			t.Run(fmt.Sprintf("adapter-%d", k), func(t *testing.T) {
				site := m.AdapterSites[k-1]
				u := site.Corner
				v := r3.Vec{X: -u.Y, Y: u.X}
				line := r3.Vec{X: site.Head.X, Y: site.Head.Y}
				bolt, nut := m.AdapterBolts[k-1].Body, m.AdapterNuts[k-1].Body
				require.Equal(t, onewayclutch.FaceLength, site.Length)
				bb, err := bolt.Bounds()
				require.NoError(t, err)
				tol := bb.Bound.Base() + 2e-3
				rear := -(onewayclutch.AdapterThickness + turbinefan.ClutchSeam)
				require.InDelta(t, rear, bb.Min.Z, tol, "head top flush with the adapter's rear face")
				require.InDelta(t, rear+site.Tip(), bb.Max.Z, tol, "tip")
				require.InDelta(t, 5.75, bb.Max.Z, tol, "tip, seam open")
				require.Less(t, bb.Max.Z, 6.75-0.5, "the tip stays inside the blind hole")
				for _, d := range []float64{bb.Min.X - line.X, bb.Max.X - line.X, bb.Min.Y - line.Y, bb.Max.Y - line.Y} {
					require.LessOrEqual(t, math.Abs(d), m2fastener.HeadPocketRadius, "head inside its pocket")
				}
				nb, err := nut.Bounds()
				require.NoError(t, err)
				ntol := nb.Bound.Base() + 2e-3
				lo, hi := site.Head.Z+site.Cavity, site.Head.Z+site.CavityFar()
				require.InDelta(t, 1.5, lo, 1e-9, "the slot's rear face")
				require.InDelta(t, 3.5, hi, 1e-9, "the slot's front face")
				require.GreaterOrEqual(t, nb.Min.Z, lo+m2fastener.NutClearance-ntol)
				require.LessOrEqual(t, nb.Max.Z, hi-m2fastener.NutClearance+ntol)
				out := 0.0
				for _, w := range meshOf(t, nut, 0.01).Vertices() {
					out = math.Max(out, math.Hypot(w.X, w.Y))
					d := w.Sub(line)
					require.LessOrEqual(t, math.Abs(d.Dot(v)), flatsHalf+ntol, "nut flats inside the slot's sides")
					require.LessOrEqual(t, math.Abs(d.Dot(u)), m2fastener.NutCavityRadius+ntol, "nut corners inside the slot")
				}
				require.Less(t, out, m.Dimensions.Size.HubRadius, "the nut stays inside the round face")
				frames := map[string]r3.Frame{"axis plane": sitePlane(t, u)}
				for _, z := range []float64{0.5, 2.5, 5.0, 6.0} {
					frames[fmt.Sprintf("z %g", z)] = atZ(t, z)
				}
				for name, f := range frames {
					requireNoOverlap(t, m.Hub, f, fmt.Sprintf("adapter fastener %d with the hub at %s", k, name), bolt, nut)
				}
			})
		}
	})
}
