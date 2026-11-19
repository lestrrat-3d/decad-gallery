package turbinefan_test

// CI: isolated-tests

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// TestTurbineFanSetScrew checks the set screw: an M2 (M2 x 10 at 192 mm) on the line x = 0, z = SetScrewZ, whose tip stands on the pushed
// shaft's flat, whose head floats over a rounded counterbore floor and stays inside the round face, whose nut sits in a
// hex slot open to the back face, and which share no volume with the hub.
func TestTurbineFanSetScrew(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		p, dm, site := c.params(), m.Dimensions, m.SetScrewSite
		sz := dm.Size
		z0 := float64(turbinefan.SetScrewZ)
		require.Equal(t, dm.SetScrewBolt, site.Length)
		require.Contains(t, m2bolt.StandardLengths(), site.Length)

		bb, err := m.SetScrew.Body.Bounds()
		require.NoError(t, err)
		tol := bb.Bound.Base() + 2e-3
		require.InDelta(t, -dm.SetScrewHead, bb.Min.Y, tol, "head top")
		require.InDelta(t, -dm.SetScrewTip, bb.Max.Y, tol, "tip")
		require.InDelta(t, p.ShaftDiameter/2-p.FlatDepth-p.BoreClearance, dm.SetScrewTip, 1e-12)
		require.InDelta(t, 2*p.BoreClearance, dm.BoreFlat-dm.SetScrewTip, 1e-12, "the tip stands 2c inside the bore's flat")
		for _, d := range []float64{bb.Min.X, bb.Max.X, bb.Min.Z - z0, bb.Max.Z - z0} {
			require.Less(t, math.Abs(d), m2fastener.HeadPocketRadius, "head inside its counterbore")
		}
		require.LessOrEqual(t, m2bolt.ShankDiameter, dm.ShaftFlatWidth, "the tip lies within the shaft's flat")

		// Nothing stands out: every vertex of the bolt is inside the round face.
		out := 0.0
		for _, v := range meshOf(t, m.SetScrew.Body, 0.01).Vertices() {
			out = math.Max(out, math.Hypot(v.X, v.Y))
		}
		require.Less(t, out, sz.HubRadius)
		t.Logf("%s: set screw M2 x %g, the head stays %.2f inside the round face", c.name, dm.SetScrewBolt,
			sz.HubRadius-dm.SetScrewHead)

		// The head floats over a rounded floor.
		require.InDelta(t, turbinefan.SetScrewTravel, (dm.SetScrewHead-m2fastener.HeadPocketDepth)-dm.CounterboreFloor, 1e-12)
		plane := section(t, m.Hub, atZ(t, z0))
		xy := func(v r3.Vec) (float64, float64) { return v.X, v.Y }
		floor := dm.CounterboreFloor
		require.True(t, inSection(plane, xy, 1.5, -(floor-0.2)), "hub below the counterbore floor")
		require.False(t, inSection(plane, xy, 1.5, -(floor+0.2)), "the counterbore above its floor")
		require.True(t, inSection(plane, xy, 2.15, -(floor+0.1)), "hub inside the floor's round")
		require.False(t, inSection(plane, xy, 2.15, -(floor+0.6)), "the counterbore past the round")

		// The nut sits in its slot.
		nb, err := m.SetScrewNut.Body.Bounds()
		require.NoError(t, err)
		ntol := nb.Bound.Base() + 2e-3
		require.GreaterOrEqual(t, nb.Min.Y, -(dm.BoreFlat+3.6)+m2fastener.NutClearance-ntol)
		require.LessOrEqual(t, nb.Max.Y, -(dm.BoreFlat+turbinefan.PressWall)-m2fastener.NutClearance+ntol)
		require.LessOrEqual(t, math.Max(-nb.Min.X, nb.Max.X), flatsHalf+ntol, "nut flats inside the slot's sides")
		require.LessOrEqual(t, math.Max(z0-nb.Min.Z, nb.Max.Z-z0), m2fastener.NutCavityRadius+ntol, "nut corners inside the slot")
		require.Less(t, bb.Min.Y, nb.Min.Y, "the bolt runs through the whole nut")
		require.Greater(t, bb.Max.Y, nb.Max.Y, "the bolt runs through the whole nut")

		// The slot: open to the back face, a hexagonal end, straight sides.
		slotY := -(dm.BoreFlat + 2.6)
		across := section(t, m.Hub, atY(t, slotY))
		zx := func(v r3.Vec) (float64, float64) { return v.Z, v.X }
		abound := across.OffSurface.Base() + 1e-3
		require.False(t, inSection(across, zx, 0.3, 0), "the slot is open to the back face")
		require.False(t, inSection(across, zx, z0, 0), "the slot at the bolt's line")
		require.False(t, inSection(across, zx, z0+2.3, 0), "the slot's end")
		require.True(t, inSection(across, zx, z0+m2fastener.NutCavityRadius+0.4, 0), "hub beyond the slot's end")
		for _, x := range []float64{-(flatsHalf + 0.4), flatsHalf + 0.4} {
			require.True(t, inSection(across, zx, 2.0, x), "hub beside the slot")
		}
		least := math.Inf(1)
		for _, l := range across.Loops {
			for _, v := range l {
				if v.Z >= 1.0 && v.Z <= z0 {
					least = math.Min(least, math.Abs(v.X))
				}
			}
		}
		require.InDelta(t, flatsHalf, least, abound, "the slot's straight sides")
		require.Less(t, least, m2nut.AcrossCorners()/2, "the nut's corners cannot pass the sides")

		frames := map[string]r3.Frame{"x 0": atX(t, 0), "x -1": atX(t, -1), "x 1": atX(t, 1), "z": atZ(t, z0),
			"z-1": atZ(t, z0-1), "z+1": atZ(t, z0+1), "y flat": atY(t, -(dm.BoreFlat + 0.8)), "y slot": atY(t, slotY),
			"y floor": atY(t, -(floor + 1))}
		for name, f := range frames {
			requireNoOverlap(t, m.Hub, f, "set screw at "+name, m.SetScrew.Body, m.SetScrewNut.Body)
		}

		// Material round the pockets.
		require.GreaterOrEqual(t, site.Cavity-m2fastener.HeadPocketDepth-turbinefan.SetScrewTravel, 1.0)
		require.InDelta(t, turbinefan.PressWall, dm.SetScrewHead-site.CavityFar()-dm.BoreFlat, 1e-12)
		require.GreaterOrEqual(t, z0-m2fastener.HeadPocketRadius, 1.0)
		require.GreaterOrEqual(t, dm.BoreFlatWidth, 2*(m2fastener.HoleRadius+0.1))
	})
}

// TestTurbineFanBladeBolts checks each blade bolt: the head's top face is flush with the front face and inside its
// pocket, the tip stays inside the hole in the hub, the nut lies inside its slot, which is open to the round face and
// closed above and below, and neither bolt nor nut shares volume with the hub or with its blade.
func TestTurbineFanBladeBolts(t *testing.T) {
	eachCase(t, []string{"605ZZ", "608ZZ"}, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		for k := 1; k <= turbinefan.BladeCount; k++ {
			t.Run(fmt.Sprintf("blade-%d", k), func(t *testing.T) {
				sz := m.Dimensions.Size
				site := m.BladeSites[k-1]
				u := bladeDir(k)
				v := r3.Vec{X: -u.Y, Y: u.X}
				line := r3.Vec{X: u.X * sz.BladeBoltAt, Y: u.Y * sz.BladeBoltAt}
				bolt, nut := m.BladeBolts[k-1].Body, m.BladeNuts[k-1].Body
				require.Equal(t, sz.BladeBoltLength, site.Length)
				require.Contains(t, m2bolt.StandardLengths(), site.Length)
				bb, err := bolt.Bounds()
				require.NoError(t, err)
				tol := bb.Bound.Base() + 2e-3
				require.InDelta(t, sz.HubLength, bb.Max.Z, tol, "head top flush with the front face")
				require.InDelta(t, sz.HubLength-site.Tip(), bb.Min.Z, tol, "tip")
				require.GreaterOrEqual(t, bb.Min.Z, turbinefan.BladeBoltTipFloor-tol, "tip inside the hub")
				for _, d := range []float64{bb.Min.X - line.X, bb.Max.X - line.X, bb.Min.Y - line.Y, bb.Max.Y - line.Y} {
					require.Less(t, math.Abs(d), m2fastener.HeadPocketRadius, "head inside its pocket")
				}
				nb, err := nut.Bounds()
				require.NoError(t, err)
				ntol := nb.Bound.Base() + 2e-3
				lo, hi := sz.HubLength-site.CavityFar(), sz.HubLength-site.Cavity
				require.InDelta(t, sz.HubLength-sz.BladeBoltCavity-m2fastener.NutCavityHeight, lo, 1e-9, "the slot's low face")
				require.InDelta(t, sz.HubLength-sz.BladeBoltCavity, hi, 1e-9, "the slot's high face")
				require.GreaterOrEqual(t, lo, 2.4-1e-9, "the nut slot keeps 2.4 of hub under it")
				require.GreaterOrEqual(t, nb.Min.Z, lo+m2fastener.NutClearance-ntol)
				require.LessOrEqual(t, nb.Max.Z, hi-m2fastener.NutClearance+ntol)
				for _, w := range meshOf(t, nut, 0.01).Vertices() {
					d := w.Sub(line)
					require.LessOrEqual(t, math.Abs(d.Dot(v)), flatsHalf+ntol, "nut flats inside the slot's sides")
					require.LessOrEqual(t, math.Abs(d.Dot(u)), m2fastener.NutCavityRadius+ntol, "nut corners inside the slot")
				}
				// The slot, in its mid section: open to the rim, closed at its sides and its end. Above and below it
				// the hub is solid, and the bolt's hole is empty.
				xy := func(w r3.Vec) (float64, float64) { return w.X, w.Y }
				mid := section(t, m.Hub, atZ(t, (lo+hi)/2))
				at := func(p r3.Vec) bool { return inSection(mid, xy, p.X, p.Y) }
				require.False(t, at(line), "the slot at the bolt's line")
				require.False(t, at(line.Add(u.Scale(3))), "the slot beyond the bolt")
				require.False(t, at(u.Scale(sz.HubRadius-0.1)), "the slot is open to the rim")
				require.True(t, at(line.Add(v.Scale(flatsHalf+0.4))), "hub beside the slot")
				require.True(t, at(line.Sub(v.Scale(flatsHalf+0.4))), "hub beside the slot")
				require.True(t, at(u.Scale(sz.BladeBoltAt-m2fastener.NutCavityRadius-0.4)), "hub beyond the slot's end")
				for _, z := range []float64{lo - 0.5, hi + 0.5} {
					level := section(t, m.Hub, atZ(t, z))
					require.True(t, inSection(level, xy, line.X+2*u.X, line.Y+2*u.Y), "hub under and over the slot at z %g", z)
					require.False(t, inSection(level, xy, line.X, line.Y), "the bolt's hole at z %g", z)
				}
				frames := map[string]r3.Frame{"axis plane": radial(t, k)}
				for _, z := range []float64{1, (lo + hi) / 2, sz.HubMid, sz.HubLength - 1} {
					frames[fmt.Sprintf("z %g", z)] = atZ(t, z)
				}
				for name, f := range frames {
					requireNoOverlap(t, m.Hub, f, fmt.Sprintf("blade fastener %d at %s", k, name), bolt, nut)
					requireNoOverlap(t, m.Blades[k-1], f, fmt.Sprintf("blade fastener %d with its blade at %s", k, name), bolt, nut)
				}
			})
		}
	})
}

// TestTurbineFanAccess requires the set screw's path from -Y and each blade's nut path to stay clear of every blade, so
// the screw and the nuts reach their seats with the hub on its shaft.
func TestTurbineFanAccess(t *testing.T) {
	eachCase(t, pitchCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		sz := m.Dimensions.Size
		reach := m2fastener.HeadPocketRadius + 1
		for k := 1; k <= turbinefan.BladeCount; k++ {
			verts := meshOf(t, m.Blades[k-1], 0.1).Vertices()
			for _, w := range verts {
				if math.Abs(w.X) < reach && w.Y < -(sz.HubRadius-1) && math.Abs(w.Z-turbinefan.SetScrewZ) < reach {
					require.FailNow(t, fmt.Sprintf("a blade %d vertex blocks the set screw's path at %v", k, w))
				}
			}
			requireAccess(t, sz, verts, k)
		}
	})
}

// TestTurbineFanAdapterCuts checks the hub's two adapter bolt cuts on the ShaftMount hub, which has them in both modes: a
// blind clearance hole from the back face to z 6.75 and a hex nut slot open to the round face, 1.5 mm in front of the
// back face, with the wall to the bore and the 1 mm of material round it.
func TestTurbineFanAdapterCuts(t *testing.T) {
	eachCase(t, boreCases, func(t *testing.T, c fanCase, m *turbinefan.Model) {
		dm := m.Dimensions
		sz := dm.Size
		xy := func(v r3.Vec) (float64, float64) { return v.X, v.Y }
		for k, site := range dm.AdapterSites {
			u := site.Corner
			v := r3.Vec{X: -u.Y, Y: u.X}
			line := r3.Vec{X: site.Head.X, Y: site.Head.Y}
			require.Contains(t, m2bolt.StandardLengths(), site.Length)
			require.InDelta(t, onewayclutch.FaceCircle, line.Len(), 1e-9)
			// The slot spans z 1.5 to 3.5: FaceNutDepth under the back face's seam, below the set screw's line (z 4) and
			// the blade slots' faces (in other directions).
			lo := site.Head.Z + site.Cavity
			require.InDelta(t, onewayclutch.FaceNutDepth, lo, 1e-9, "1.5 mm of hub under the slot")
			hi := lo + m2fastener.NutCavityHeight
			require.Less(t, hi, float64(turbinefan.SetScrewZ), "the slot stays under the set screw's line")

			mid := section(t, m.Hub, atZ(t, (lo+hi)/2))
			at := func(p r3.Vec) bool { return inSection(mid, xy, p.X, p.Y) }
			require.False(t, at(line), "the slot at the bolt's line")
			require.False(t, at(u.Scale(sz.HubRadius-0.1)), "the slot is open to the round face")
			require.True(t, at(line.Add(v.Scale(flatsHalf+0.4))), "hub beside the slot")
			require.True(t, at(line.Sub(v.Scale(flatsHalf+0.4))), "hub beside the slot")
			require.True(t, at(u.Scale(onewayclutch.FaceCircle-m2fastener.NutCavityRadius-0.4)), "hub beyond the slot's end")
			// Wall from the slot's inner corner to the bore: PressWall or more.
			corner := onewayclutch.FaceCircle - m2fastener.NutCavityRadius
			require.GreaterOrEqual(t, corner-dm.BoreRadius, turbinefan.PressWall, "wall between the slot and the bore")
			// The hole is blind: open at the back face, closed from z 6.75; solid above and below the slot beside it.
			for _, z := range []float64{0.5, 1.0, 5.0, 6.5} {
				level := section(t, m.Hub, atZ(t, z))
				require.False(t, inSection(level, xy, line.X, line.Y), "the bolt's hole at z %g", z)
			}
			for _, z := range []float64{lo - 0.3, hi + 0.3} {
				level := section(t, m.Hub, atZ(t, z))
				require.True(t, inSection(level, xy, line.X+2*u.X, line.Y+2*u.Y), "hub round the slot at z %g", z)
			}
			require.Less(t, site.Head.Z+site.Tip(), 6.75, "the bolt's tip stays inside the blind hole")
			closed := section(t, m.Hub, atZ(t, 6.95))
			require.True(t, inSection(closed, xy, line.X, line.Y), "the hole's floor at z 6.95")
			t.Logf("%s adapter cut %d: slot z %.2f to %.2f", c.name, k+1, lo, hi)
		}
	})
}
