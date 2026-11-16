package turbinefan_test

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/handcrankfan"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanClampBolt checks the clamp bolt across the tab: an M2 x 16 whose head's top face is flush with the +X
// side face and inside its pocket, which crosses the slit and whose tip ends flush with the -X side face, and whose nut
// lies inside its hexagonal pocket, below the -X side face; neither bolt nor nut shares volume with the hub in sections
// along and across the bolt. It also checks the pocket: open to the -X side face, a hexagon whose flats the nut's
// corners cannot pass, so the nut cannot turn, with room for the nut's corners.
func TestTurbineFanClampBolt(t *testing.T) {
	eachCase(t, []string{"605ZZ", "608ZZ"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm, site := m.Dimensions, m.ClampSite
		mid := float64(turbinefan.HubMid)
		w := float64(turbinefan.TabHalfWidth)
		floor := -w + m2fastener.NutCavityHeight // the nut pocket's floor
		bb, err := m.ClampBolt.Body.Bounds()
		require.NoError(t, err)
		tol := bb.Bound.Base() + 2e-3
		require.Equal(t, turbinefan.ClampBolt, site.Length)
		require.Contains(t, m2bolt.StandardLengths(), site.Length)
		require.InDelta(t, w, bb.Max.X, tol, "head top flush with the +X side face")
		require.InDelta(t, w-site.Tip(), bb.Min.X, tol, "tip")
		require.InDelta(t, -w, bb.Min.X, tol, "tip flush with the -X side face")
		require.Less(t, math.Max(math.Max(bb.Max.Y-dm.ClampY, dm.ClampY-bb.Min.Y), math.Max(bb.Max.Z-mid, mid-bb.Min.Z)),
			m2fastener.HeadPocketRadius, "head inside its pocket")
		require.Less(t, bb.Min.X, -turbinefan.SlitWidth/2, "the bolt crosses the slit")
		nb, err := m.ClampNut.Body.Bounds()
		require.NoError(t, err)
		ntol := nb.Bound.Base() + 2e-3
		require.GreaterOrEqual(t, nb.Min.X, -w+m2fastener.NutClearance-ntol, "nut below the -X side face")
		require.LessOrEqual(t, nb.Max.X, floor-m2fastener.NutClearance+ntol, "nut inside its pocket")
		require.Less(t, bb.Min.X, nb.Min.X, "the bolt runs through the whole nut")
		for _, d := range []float64{nb.Min.Z - mid, nb.Max.Z - mid} {
			require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius, "nut: corners inside its pocket")
		}
		for _, d := range []float64{nb.Min.Y - dm.ClampY, nb.Max.Y - dm.ClampY} {
			require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius*math.Sqrt(3)/2+ntol, "nut: flats inside its pocket")
		}
		requireClampPocket(t, m)
		frames := map[string]r3.Frame{}
		for _, z := range []float64{7.8, 8.8, 10, 11.2, 12.2} {
			frames[fmt.Sprintf("z %g", z)] = atZ(t, z)
		}
		for _, x := range []float64{9.5, 8, 6, 3, 0, -3, -6, -7.9, -8.3, -9, -9.7} {
			frames[fmt.Sprintf("x %g", x)] = atX(t, x)
		}
		for name, f := range frames {
			requireNoOverlap(t, m.Hub, f, "clamp fastener at "+name, m.ClampBolt.Body, m.ClampNut.Body)
		}
		// 1 mm or more of material round the pockets: the tab's flat side faces beyond the head pocket's ring and the
		// nut pocket's flats, the web between the hole and the bore, and under the head pocket.
		flat := m2fastener.NutCavityRadius * math.Sqrt(3) / 2
		for name, wall := range map[string]float64{
			"between the hole and the bore":              -dm.ClampY - m2fastener.HoleRadius - dm.BoreRadius,
			"of -X side face beyond the nut pocket":      dm.ClampY - flat - dm.SideEnd,
			"of side face beyond the head pocket":        dm.ClampY - m2fastener.HeadPocketRadius - dm.SideEnd,
			"between the head pocket and each face":      mid - m2fastener.HeadPocketRadius,
			"between the nut pocket and each face":       mid - m2fastener.NutCavityRadius,
			"under the head pocket to the slit":          w - m2fastener.HeadPocketDepth - turbinefan.SlitWidth/2,
			"between the nut pocket floor and the bore ": math.Hypot(floor, math.Min(0, dm.ClampY+flat)) - dm.BoreRadius,
		} {
			require.GreaterOrEqual(t, wall, 1.0-1e-9, "material %s", name)
		}
	})
}

// requireClampPocket checks the clamp nut's pocket in sections of the hub. Across the bolt, 1 mm inside the -X side
// face, its loop round the bolt's axis comes no nearer the axis than the nut's flats allow and no farther than the
// circle through the pocket's corners, and its least distance is under the nut's corner radius, so the nut cannot turn
// in it while its corners keep their room. Along the bolt at mid thickness, the pocket opens to the -X side face and the
// hub is solid beyond its floor and beyond its corners.
func requireClampPocket(t *testing.T, m *turbinefan.Model) {
	t.Helper()
	dm := m.Dimensions
	mid := float64(turbinefan.HubMid)
	w := float64(turbinefan.TabHalfWidth)
	floor := -w + m2fastener.NutCavityHeight
	across := section(t, m.Hub, atX(t, -w+1))
	bound := across.OffSurface.Base() + 1e-3
	near, far := math.Inf(1), math.Inf(-1)
	up, side := 0.0, 0.0
	found := false
	for _, l := range across.Loops {
		var ds []float64
		for _, v := range l {
			ds = append(ds, math.Hypot(v.Y-dm.ClampY, v.Z-mid))
		}
		if slices.Max(ds) > m2fastener.NutCavityRadius+0.5 {
			continue
		}
		found = true
		for _, v := range l {
			up, side = math.Max(up, math.Abs(v.Z-mid)), math.Max(side, math.Abs(v.Y-dm.ClampY))
		}
		for i := range l {
			a, b := l[i], l[(i+1)%len(l)]
			yz := func(v r3.Vec) r3.Vec { return r3.Vec{X: v.Y - dm.ClampY, Y: v.Z - mid} }
			near = math.Min(near, distanceToLoop(r3.Vec{}, []r3.Vec{yz(a), yz(b)}))
			far = math.Max(far, math.Hypot(a.Y-dm.ClampY, a.Z-mid))
		}
	}
	require.True(t, found, "the nut pocket's loop round the clamp bolt's axis")
	// A diagonal lies along Z: a corner points up and one down, and the flats are the +Y and -Y sides.
	require.InDelta(t, m2fastener.NutCavityRadius, up, bound, "nut pocket: a corner up and one down")
	require.InDelta(t, m2fastener.NutCavityRadius*math.Sqrt(3)/2, side, bound, "nut pocket: flats on the sides")
	require.InDelta(t, m2fastener.NutCavityRadius*math.Sqrt(3)/2, near, bound, "nut pocket: its flats")
	require.Less(t, near, m2nut.AcrossCorners()/2, "nut pocket: the nut's corners cannot pass its flats")
	require.LessOrEqual(t, far, m2fastener.NutCavityRadius+bound, "nut pocket: its corners")
	require.Greater(t, far, m2nut.AcrossCorners()/2+0.05, "nut pocket: room for the nut's corners")
	along := section(t, m.Hub, atZ(t, mid))
	xy := func(v r3.Vec) (float64, float64) { return v.X, v.Y }
	require.False(t, inSection(along, xy, -w+0.3, dm.ClampY+1.8), "nut pocket open to the -X side face")
	require.True(t, inSection(along, xy, -w+0.3, dm.ClampY+m2fastener.NutCavityRadius+0.5), "hub beyond the nut pocket's corner")
	require.True(t, inSection(along, xy, floor+0.4, dm.ClampY+1.8), "hub beyond the nut pocket's floor")
}

// TestTurbineFanBladeBolts checks each blade bolt: the head's top face is flush with the front face and inside its
// pocket, the tip stays inside the hole in the hub, the nut lies inside its cavity, the hub encloses the nut on all
// sides but the bolt hole, and neither bolt nor nut shares volume with the hub or with its blade.
func TestTurbineFanBladeBolts(t *testing.T) {
	eachCase(t, []string{"605ZZ", "608ZZ"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		for k := 1; k <= turbinefan.BladeCount; k++ {
			t.Run(fmt.Sprintf("blade-%d", k), func(t *testing.T) {
				site := m.BladeSites[k-1]
				u := bladeDir(k)
				line := r3.Vec{X: u.X * handcrankfan.BladeBoltAt, Y: u.Y * handcrankfan.BladeBoltAt}
				bolt, nut := m.BladeBolts[k-1].Body, m.BladeNuts[k-1].Body
				require.Equal(t, handcrankfan.BladeBoltLength, site.Length)
				require.Contains(t, m2bolt.StandardLengths(), site.Length)
				bb, err := bolt.Bounds()
				require.NoError(t, err)
				tol := bb.Bound.Base() + 2e-3
				require.InDelta(t, turbinefan.HubLength, bb.Max.Z, tol, "head top flush with the front face")
				require.InDelta(t, turbinefan.HubLength-site.Tip(), bb.Min.Z, tol, "tip")
				require.Greater(t, bb.Min.Z, 0.0, "tip inside the hub")
				for _, d := range []float64{bb.Min.X - line.X, bb.Max.X - line.X, bb.Min.Y - line.Y, bb.Max.Y - line.Y} {
					require.Less(t, math.Abs(d), m2fastener.HeadPocketRadius, "head inside its pocket")
				}
				nb, err := nut.Bounds()
				require.NoError(t, err)
				ntol := nb.Bound.Base() + 2e-3
				lo, hi := turbinefan.HubLength-site.CavityFar(), turbinefan.HubLength-site.Cavity
				require.GreaterOrEqual(t, nb.Min.Z, lo-ntol)
				require.LessOrEqual(t, nb.Max.Z, hi+ntol)
				require.InDelta(t, 2.4, lo, 1e-9, "the cavity's low face")
				require.InDelta(t, 4.4, hi, 1e-9, "the cavity's high face")
				for _, d := range []float64{nb.Min.X - line.X, nb.Max.X - line.X, nb.Min.Y - line.Y, nb.Max.Y - line.Y} {
					require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius, "nut inside its cavity")
				}
				// The hub holds the nut: material above and below the cavity beside the bolt hole, and all round it; the cavity
				// and the bolt hole are empty. The cavity's faces are within 0.5 mm of these points, where the faceted hub's
				// point location is undecided, so the sections read them.
				xy := func(v r3.Vec) (float64, float64) { return v.X, v.Y }
				solid := func(z, r, a float64) bool {
					return inSection(section(t, m.Hub, atZ(t, z)), xy, line.X+r*math.Cos(a), line.Y+r*math.Sin(a))
				}
				for _, z := range []float64{1.9, 4.9} {
					for i := range 8 {
						a := float64(i) * math.Pi / 4
						require.True(t, solid(z, 2.0, a), "the hub closes the cavity at z %g, angle %g", z, a)
						require.False(t, solid(z, 0.5, a), "the bolt hole at z %g", z)
					}
				}
				for i := range 8 {
					a := float64(i) * math.Pi / 4
					require.True(t, solid(3.4, 3.0, a), "the hub round the cavity at angle %g", a)
					require.False(t, solid(3.4, 2.0, a), "the cavity at angle %g is empty", a)
				}
				frames := map[string]r3.Frame{"axis plane": radial(t, k)}
				for _, z := range []float64{1, 3.4, 10, 19} {
					frames[fmt.Sprintf("z %g", z)] = atZ(t, z)
				}
				for name, f := range frames {
					requireNoOverlap(t, m.Hub, f, fmt.Sprintf("blade fastener %d at %s", k, name), bolt, nut)
					requireNoOverlap(t, m.Blades[k-1], f, fmt.Sprintf("blade fastener %d with its blade at %s", k, name), bolt, nut)
				}
				// 1 mm or more of hub under the head pocket, round the cavity and to the back face.
				require.GreaterOrEqual(t, handcrankfan.BladeBoltCavity-m2fastener.HeadPocketDepth, 1.0)
			})
		}
	})
}
