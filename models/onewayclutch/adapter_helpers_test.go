package onewayclutch_test

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// requireAdapter runs the adapter checks for the clutch p.
func requireAdapter(t *testing.T, p onewayclutch.Params) {
	// The cached default clutch lives in its own document, so this test builds its own pair.
	doc := onewayclutch.NewDocument()
	m, err := onewayclutch.Build(t.Context(), doc, p)
	require.NoError(t, err)
	a, err := onewayclutch.BuildAdapter(t.Context(), doc, p)
	require.NoError(t, err)
	dm := m.Dimensions

	t.Run("bodies and containers", func(t *testing.T) {
		require.Len(t, doc.Bodies(), 17)
		require.NoError(t, doc.VerifyContainers().Err())
		want := map[string][]*decad.Body{
			onewayclutch.HubContainerName:     {m.Hub},
			onewayclutch.RingContainerName:    {m.Ring},
			onewayclutch.PlatesContainerName:  {m.BackPlate, m.Cover},
			onewayclutch.AdapterContainerName: {a.Body},
			onewayclutch.BoltsContainerName:   nil,
			onewayclutch.NutsContainerName:    nil,
		}
		for i := range 3 {
			want[onewayclutch.BoltsContainerName] = append(want[onewayclutch.BoltsContainerName], m.Bolts[i].Body)
			want[onewayclutch.NutsContainerName] = append(want[onewayclutch.NutsContainerName], m.Nuts[i].Body)
		}
		for i := range 3 {
			want[onewayclutch.BoltsContainerName] = append(want[onewayclutch.BoltsContainerName], a.Bolts[i].Body)
			want[onewayclutch.NutsContainerName] = append(want[onewayclutch.NutsContainerName], a.Nuts[i].Body)
		}
		children := doc.Root().Children()
		require.Len(t, children, len(want))
		for _, box := range children {
			bodies, ok := want[box.Name()]
			require.True(t, ok, "unexpected container %q", box.Name())
			require.ElementsMatch(t, bodies, box.Bodies(), box.Name())
		}
	})

	t.Run("sketches", func(t *testing.T) {
		// The annulus and the six head pockets: three at the ears, three at the face holes.
		require.Len(t, a.Sketches, 1+2*onewayclutch.Ears)
		for i, s := range a.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "adapter sketch %d", i)
		}
		for i, s := range m.Sketches {
			require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s), "clutch sketch %d", i)
		}
	})

	t.Run("shape", func(t *testing.T) {
		box, err := a.Body.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-3
		require.InDelta(t, onewayclutch.AdapterFrom, box.Min.Z, tol)
		require.InDelta(t, onewayclutch.AdapterTo, box.Max.Z, tol)
		require.InDelta(t, dm.AdapterRadius, box.Max.Y, tol, "an ear tip's radius, at 90 degrees")
		require.InDelta(t, -dm.AdapterRadius, box.Min.Y, tol)
		// Both faces are chamfered 0.5 mm, so a section 0.25 mm inside each is a circle 0.25 mm in.
		for name, z := range map[string]float64{"back": onewayclutch.AdapterFrom + 0.25, "front": onewayclutch.AdapterTo - 0.25} {
			s := fitSection(t, a.Body, fitPlane(t, z))
			outer, inner := 0.0, math.Inf(1)
			for _, p := range fitPoints(s) {
				r := math.Hypot(p.X, p.Y)
				if r > 20 {
					outer = math.Max(outer, r)
				}
				if r > 3 && r < dm.AdapterInner+1 { // the access holes start at BoltCircle - HoleRadius
					inner = math.Min(inner, r)
				}
			}
			require.InDelta(t, dm.AdapterRadius-0.25, outer, 0.03, "%s face: outer edge", name)
			require.InDelta(t, dm.AdapterInner+0.25, inner, 0.03, "%s face: inner edge", name)
		}
		// The face is over 400 mm², so the chamfer is required.
		area := math.Pi*(dm.AdapterRadius*dm.AdapterRadius-dm.AdapterInner*dm.AdapterInner) -
			9*math.Pi*m2fastener.HoleRadius*m2fastener.HoleRadius
		require.Greater(t, area, 400.0)
	})

	t.Run("fasteners", func(t *testing.T) {
		ring, err := m.Ring.Bounds()
		require.NoError(t, err)
		adapter, err := a.Body.Bounds()
		require.NoError(t, err)
		for i, site := range a.Sites {
			label := fmt.Sprintf("mount site %d", i+1)
			require.Contains(t, m2fastener.BoltSize.StandardLengths, site.Length, label)
			bb, err := a.Bolts[i].Body.Bounds()
			require.NoError(t, err)
			tol := bb.Bound.Base() + 2e-3
			// The head is flush with the adapter's front face and inside its pocket.
			require.InDelta(t, adapter.Max.Z, bb.Max.Z, tol, "%s: head flush", label)
			require.InDelta(t, onewayclutch.AdapterTo, bb.Max.Z, tol, "%s: head at the front face", label)
			require.InDelta(t, m2fastener.BoltSize.SocketCap.HeadDiameter/2, bb.Max.X-site.Head.X, tol, "%s: bolt centred", label)
			require.LessOrEqual(t, bb.Max.X-site.Head.X, m2fastener.HeadPocketRadius, "%s: head inside its pocket", label)
			// With the seams open the tip stands 0.25 inside the ring's rear face and never outside it. Tightened, the
			// seam closes and the tip ends flush.
			require.InDelta(t, ring.Min.Z+onewayclutch.Seam, bb.Min.Z, tol, "%s: tip inside the rear face", label)
			require.GreaterOrEqual(t, bb.Min.Z-onewayclutch.Seam, ring.Min.Z-tol, "%s: tightened tip not outside the face", label)
			// The nut sits inside its pocket: below the pocket's floor and inside the ring's rear face.
			nb, err := a.Nuts[i].Body.Bounds()
			require.NoError(t, err)
			ntol := nb.Bound.Base() + 2e-3
			nearZ, farZ := site.At(site.Cavity).Z, site.At(site.CavityFar()).Z
			require.LessOrEqual(t, nb.Max.Z, nearZ-m2fastener.NutClearance+ntol, "%s: nut below the pocket's floor", label)
			require.GreaterOrEqual(t, nb.Min.Z, farZ+m2fastener.NutClearance-ntol, "%s: nut inside the rear face", label)
			require.InDelta(t, ring.Min.Z+m2fastener.NutClearance, nb.Min.Z, ntol+0.01, "%s: nut 0.2 inside the face", label)
			for _, d := range []float64{nb.Max.X - site.Head.X, site.Head.X - nb.Min.X, nb.Max.Y - site.Head.Y, site.Head.Y - nb.Min.Y} {
				require.LessOrEqual(t, d, m2fastener.NutCavityRadius+ntol, "%s: nut inside its pocket's corners", label)
			}
			// The tip reaches the nut.
			require.Less(t, bb.Min.Z, site.At(site.Cavity+m2fastener.NutClearance).Z, "%s: tip reaches the nut", label)
		}
	})

	t.Run("clearances", func(t *testing.T) {
		// Sections across each bolt: the shank in the ear bore and the adapter hole, the head in its pocket, the nut in
		// its pocket.
		type probe struct {
			name      string
			part      *decad.Body
			with      *decad.Body
			z         float64
			clearance float64
		}
		for i, site := range a.Sites {
			bolt, nut := a.Bolts[i].Body, a.Nuts[i].Body
			probes := []probe{
				{"bolt/adapter hole", a.Body, bolt, onewayclutch.AdapterFrom + 1, 0.28},
				{"bolt/adapter head pocket", a.Body, bolt, site.At(m2fastener.HeadPocketDepth / 2).Z, 0.28},
				{"bolt/ring ear bore", m.Ring, bolt, onewayclutch.RingTo / 2, 0.28},
				{"nut/ring pocket", m.Ring, nut, (site.At(site.Cavity).Z + site.At(site.CavityFar()).Z) / 2, 0.15},
				{"bolt/ring pocket", m.Ring, bolt, (site.At(site.Cavity).Z + site.At(site.CavityFar()).Z) / 2, 0.5},
			}
			for _, pr := range probes {
				f := atZ(t, pr.z)
				x, err := pr.with.SectionOutline(t.Context(), f, units.Millimeters(0.03))
				require.NoError(t, err, pr.name)
				y, err := pr.part.SectionOutline(t.Context(), f, units.Millimeters(0.03))
				require.NoError(t, err, pr.name)
				c, err := x.Clearance(y)
				require.NoError(t, err, pr.name)
				require.False(t, c.Overlaps, "mount site %d: %s overlaps at z %g", i+1, pr.name, pr.z)
				require.GreaterOrEqual(t, c.Distance.Value.Base(), pr.clearance, "mount site %d: %s at z %g", i+1, pr.name, pr.z)
			}
		}
	})

	t.Run("no overlaps and the seam", func(t *testing.T) {
		bodies := map[string]*decad.Body{
			"adapter": a.Body, "hub": m.Hub, "ring": m.Ring, "back plate": m.BackPlate, "cover": m.Cover,
		}
		for i := range 3 {
			bodies[fmt.Sprintf("mount bolt %d", i+1)] = a.Bolts[i].Body
			bodies[fmt.Sprintf("mount nut %d", i+1)] = a.Nuts[i].Body
			bodies[fmt.Sprintf("bolt %d", i+1)] = m.Bolts[i].Body
			bodies[fmt.Sprintf("nut %d", i+1)] = m.Nuts[i].Body
		}
		names := make([]string, 0, len(bodies))
		for name := range bodies {
			names = append(names, name)
		}
		sort.Strings(names)
		meshes := map[string]meshgap.Mesh{}
		for _, name := range names {
			tol := 0.05
			if name == "ring" || name == "adapter" {
				tol = 0.03
			}
			meshes[name] = fitMeshOf(t, bodies[name], tol)
		}
		isMountBolt := func(s string) bool { return len(s) == 12 && s[:11] == "mount bolt " }
		isMountNut := func(s string) bool { return len(s) == 11 && s[:10] == "mount nut " }
		for i, x := range names {
			for _, y := range names[i+1:] {
				switch {
				case isBoltNutPair(x, y), isBoltCoverPair(x, y):
					continue
				case isMountBolt(x) && y == "adapter", isMountBolt(y) && x == "adapter":
					// The head rests on its pocket's floor, so the meshes touch; "clearances" measures the pair.
					continue
				case isMountBolt(x) && isMountNut(y), isMountBolt(y) && isMountNut(x):
					continue
				case isMountBolt(x) && len(y) == 6 && y[:4] == "nut ", isMountBolt(y) && len(x) == 6 && x[:4] == "nut ":
					continue
				}
				gap, clear, err := meshes[x].Exceeds(meshes[y], 0)
				require.NoError(t, err)
				require.True(t, clear, "%s and %s overlap or touch (gap bound %g)", x, y, gap)
				if x == "adapter" && y == "ring" || y == "adapter" && x == "ring" {
					// A clamped seam of 0.25 mm, proven to 0.20 or more.
					require.GreaterOrEqual(t, gap, onewayclutch.Seam-onewayclutch.FitMargin, "the seam")
					t.Logf("adapter / ring seam: proven at least %.3f", gap)
				}
			}
		}
	})

	t.Run("walls", func(t *testing.T) {
		// Round each head pocket: the section through the heads shows three pockets, the outer edge and the inner edge.
		z := a.Sites[0].At(m2fastener.HeadPocketDepth / 2).Z
		s := fitSection(t, a.Body, fitPlane(t, z))
		// The outer edge, the bore, the three ear holes in their pockets, the three face holes and the three access holes.
		require.Equal(t, 2+3*onewayclutch.Ears, len(s.Loops))
		found := 0
		for i, loop := range s.Loops {
			c, near := r3.Vec{}, false
			for _, site := range a.Sites {
				if math.Hypot(loop[0].X-site.Head.X, loop[0].Y-site.Head.Y) < m2fastener.HeadPocketRadius+0.1 {
					c, near = site.Head, true
				}
			}
			if !near {
				continue
			}
			found++
			for _, p := range loop {
				require.InDelta(t, m2fastener.HeadPocketRadius, math.Hypot(p.X-c.X, p.Y-c.Y), 0.03, "pocket round")
			}
			wall := math.Inf(1)
			for j, other := range s.Loops {
				if j != i {
					for _, p := range loop {
						wall = math.Min(wall, fitNearest(p, [][]r3.Vec{other}))
					}
				}
			}
			t.Logf("head pocket wall %.3f", wall)
			require.GreaterOrEqual(t, wall, 1.5-0.03, "wall round a head pocket")
		}
		require.Equal(t, onewayclutch.Ears, found)
		// Material under each head pocket: its floor to the adapter's back face.
		floor := a.Sites[0].At(m2fastener.HeadPocketDepth).Z
		require.GreaterOrEqual(t, floor-onewayclutch.AdapterFrom, 4.0-1e-9, "material under a head pocket")
	})

	t.Run("face interface and access holes", func(t *testing.T) {
		// A section through the rear face pockets: each face hole's head pocket is a circle about its site, and the wall
		// to every other loop is 1.5 mm or more.
		z := dm.FaceSites[0].At(m2fastener.HeadPocketDepth / 2).Z
		s := fitSection(t, a.Body, fitPlane(t, z))
		pockets := 0
		for i, loop := range s.Loops {
			for _, site := range dm.FaceSites {
				if math.Hypot(loop[0].X-site.Head.X, loop[0].Y-site.Head.Y) > m2fastener.HeadPocketRadius+0.1 {
					continue
				}
				pockets++
				for _, p := range loop {
					require.InDelta(t, m2fastener.HeadPocketRadius, math.Hypot(p.X-site.Head.X, p.Y-site.Head.Y), 0.03)
				}
				wall := math.Inf(1)
				for j, other := range s.Loops {
					if j == i {
						continue
					}
					for _, p := range loop {
						wall = math.Min(wall, fitNearest(p, [][]r3.Vec{other}))
					}
				}
				require.GreaterOrEqual(t, wall, 1.5-0.03, "wall round a face head pocket")
			}
		}
		require.Equal(t, onewayclutch.Ears, pockets)
		// Each face bolt's head would end flush with the rear face, and 1 mm or more of material lies under its pocket.
		for i, site := range dm.FaceSites {
			require.Contains(t, m2fastener.BoltSize.StandardLengths, site.Length, "face site %d", i+1)
			require.InDelta(t, onewayclutch.AdapterFrom, site.Head.Z, 1e-9, "face site %d: head flush with the rear face", i+1)
			require.GreaterOrEqual(t, onewayclutch.AdapterThickness-m2fastener.HeadPocketDepth, 1.0-1e-9, "material under a pocket")
		}
		// The section through the adapter's mid plane shows each face hole, centred on its site, and each access hole
		// centred over a clutch bolt head; a 1.5 mm hex key (1.73 mm across corners) passes the access hole.
		mid := fitSection(t, a.Body, fitPlane(t, onewayclutch.AdapterFrom+onewayclutch.AdapterThickness/2))
		require.Equal(t, 2+3*onewayclutch.Ears, len(mid.Loops))
		require.Greater(t, 2*m2fastener.HoleRadius, 1.5*2/math.Sqrt(3))
		centres := make([]r3.Vec, 0, 2*onewayclutch.Ears)
		for _, site := range dm.Sites {
			centres = append(centres, site.Head)
		}
		for _, site := range dm.FaceSites {
			centres = append(centres, r3.Vec{X: site.Head.X, Y: site.Head.Y})
		}
		for _, c := range centres {
			found := false
			for _, loop := range mid.Loops {
				if math.Hypot(loop[0].X-c.X, loop[0].Y-c.Y) > m2fastener.HoleRadius+0.1 {
					continue
				}
				found = true
				for _, p := range loop {
					require.InDelta(t, m2fastener.HoleRadius, math.Hypot(p.X-c.X, p.Y-c.Y), 0.03, "hole round at %v", c)
				}
			}
			require.True(t, found, "no hole at %v", c)
		}
		// No access hole or face hole breaks into the bore: 1.5 mm of material remains to the bore's chamfer.
		for _, c := range centres {
			require.GreaterOrEqual(t, math.Hypot(c.X, c.Y)-m2fastener.HoleRadius-dm.AdapterInner, 1.5)
		}
	})

	t.Run("corners", func(t *testing.T) {
		// Printed front face down, axis vertical, with no exemption: an annulus with round holes has no inside corner.
		fs, err := cornercheck.Check(t.Context(), a.Body, r3.Vec{Z: 1}, nil)
		require.NoError(t, err)
		require.Empty(t, fs, cornercheck.Report(fs, 5))
	})
}
