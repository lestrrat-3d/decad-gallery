package onewayclutch_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// fastenerCases are every shaft preset in both directions, and a loose bore.
func fastenerCases() map[string]onewayclutch.Params {
	out := map[string]onewayclutch.Params{}
	for _, sh := range onewayclutch.Shafts() {
		for _, dir := range []onewayclutch.Direction{onewayclutch.Clockwise, onewayclutch.CounterClockwise} {
			p := onewayclutch.DefaultParams()
			p.Shaft, p.Freewheel = sh.Name, dir
			out[fmt.Sprintf("fasteners %s %s", sh.Name, dir)] = p
		}
	}
	p := onewayclutch.DefaultParams()
	p.BoreClearance = 0.35
	out["fasteners 605ZZ loose"] = p
	return out
}

func TestOneWayClutchFasteners(t *testing.T) {
	for name, p := range fastenerCases() {
		t.Run(name, func(t *testing.T) {
			m, _ := buildCase(t, p)
			for i, site := range m.Sites {
				requireSiteFastener(t, m, i, site)
				requireSitePocket(t, m, i, site)
				requireSiteWalls(t, m, i, site)
				requireSiteClear(t, m, i, site)
			}
			requireHubBoltWalls(t, m)
		})
	}
}

// requireSiteFastener checks one site's bolt and nut by bounds: the head flush with the cover's outer face and inside
// its pocket, the tip flush with the back plate's outer face, the nut inside its hex pocket.
func requireSiteFastener(t *testing.T, m *onewayclutch.Model, i int, site m2fastener.Site) {
	t.Helper()
	label := fmt.Sprintf("site %d", i+1)
	require.Contains(t, m2bolt.StandardLengths(), site.Length, label)
	require.Equal(t, 0.0, site.Dir.X, label)
	require.Equal(t, 0.0, site.Dir.Y, label)
	require.Equal(t, -1.0, site.Dir.Z, label)

	cover, err := m.Cover.Bounds()
	require.NoError(t, err)
	bb, err := m.Bolts[i].Body.Bounds()
	require.NoError(t, err)
	tol := bb.Bound.Base() + 2e-3
	headZ, tipZ := site.Head.Z, site.At(site.Tip()).Z
	require.InDelta(t, cover.Max.Z, bb.Max.Z, tol, "%s: head top flush with the cover's outer face", label)
	require.InDelta(t, headZ, bb.Max.Z, tol, "%s: head top at the site", label)
	require.InDelta(t, tipZ, bb.Min.Z, tol, "%s: tip", label)
	require.GreaterOrEqual(t, bb.Min.Z, -tol, "%s: tip above z 0", label)
	back0, err := m.BackPlate.Bounds()
	require.NoError(t, err)
	// With the seams open (the model) the tip lies 2*Seam inside the back plate's outer face. Tightening closes both
	// seams and pulls the tip 2*Seam toward the back face: it then ends flush, never outside, and the nut, seated on its
	// pocket's floor, is fully engaged.
	require.InDelta(t, back0.Min.Z+2*onewayclutch.Seam, bb.Min.Z, tol, "%s: the tip lies 2*Seam inside the back face", label)
	require.GreaterOrEqual(t, bb.Min.Z-2*onewayclutch.Seam, back0.Min.Z-tol, "%s: the tightened tip is not outside the back face", label)
	floorZ := site.At(site.Cavity).Z
	require.LessOrEqual(t, bb.Min.Z-2*onewayclutch.Seam, floorZ-m2nut.Thickness+tol, "%s: the tightened tip is past the nut", label)
	require.Less(t, bb.Min.Z, site.At(site.Cavity+m2fastener.NutClearance).Z, "%s: tip reaches the nut", label)
	// Head inside its pocket: the bolt's radius from the site's line stays within the pocket.
	for _, d := range []float64{bb.Max.X - site.Head.X, site.Head.X - bb.Min.X, bb.Max.Y - site.Head.Y, site.Head.Y - bb.Min.Y} {
		require.LessOrEqual(t, d, m2fastener.HeadPocketRadius, "%s: head inside its pocket", label)
	}
	require.InDelta(t, m2bolt.HeadDiameter/2, bb.Max.X-site.Head.X, tol, "%s: bolt centred on the site", label)

	nb, err := m.Nuts[i].Body.Bounds()
	require.NoError(t, err)
	ntol := nb.Bound.Base() + 2e-3
	nearZ, farZ := site.At(site.Cavity).Z, site.At(site.CavityFar()).Z // nearZ > farZ
	require.LessOrEqual(t, nb.Max.Z, nearZ-m2fastener.NutClearance+ntol, "%s: nut below the cavity's near face", label)
	require.GreaterOrEqual(t, nb.Min.Z, farZ+m2fastener.NutClearance-ntol, "%s: nut above the cavity's far face", label)
	for _, d := range []float64{nb.Max.X - site.Head.X, site.Head.X - nb.Min.X, nb.Max.Y - site.Head.Y, site.Head.Y - nb.Min.Y} {
		require.LessOrEqual(t, d, m2fastener.NutCavityRadius+ntol, "%s: nut inside its cavity round", label)
	}
	back, err := m.BackPlate.Bounds()
	require.NoError(t, err)
	require.InDelta(t, back.Min.Z+m2fastener.NutClearance, nb.Min.Z, ntol+0.01,
		"%s: the nut's outer face lies 0.2 above the back plate's outer face", label)
	require.Less(t, nb.Max.Z, back.Max.Z, "%s: nut inside the back plate", label)
}

// requireSitePocket checks the back plate's section at z 1.0 at one site: a hexagon with its corners 2.51 mm from the
// site's line along Corner and its flats 2.174 mm from it, open to the outer face (the section sits between the
// pocket's floor at z 2.0 and the face at z 0, so the pocket is a hole of the plate's section there).
func requireSitePocket(t *testing.T, m *onewayclutch.Model, i int, site m2fastener.Site) {
	t.Helper()
	label := fmt.Sprintf("site %d", i+1)
	s, err := m.BackPlate.SectionOutline(t.Context(), atZ(t, 1.0), units.Millimeters(0.01))
	require.NoError(t, err)
	var hex []r3.Vec
	for _, loop := range s.Loops {
		lo, hi := r3.Vec{X: math.Inf(1), Y: math.Inf(1)}, r3.Vec{X: math.Inf(-1), Y: math.Inf(-1)}
		for _, p := range loop {
			lo, hi = r3.Vec{X: math.Min(lo.X, p.X), Y: math.Min(lo.Y, p.Y)}, r3.Vec{X: math.Max(hi.X, p.X), Y: math.Max(hi.Y, p.Y)}
		}
		if math.Hypot((lo.X+hi.X)/2-site.Head.X, (lo.Y+hi.Y)/2-site.Head.Y) < 0.3 && hi.X-lo.X < 6 {
			hex = loop
		}
	}
	require.NotEmpty(t, hex, "%s: a hole at the site", label)
	corner := site.Corner
	flat := r3.Vec{X: -corner.Y, Y: corner.X}
	var alongMax, acrossMax float64
	for _, p := range hex {
		d := r3.Vec{X: p.X - site.Head.X, Y: p.Y - site.Head.Y}
		alongMax = math.Max(alongMax, math.Abs(d.Dot(corner)))
		acrossMax = math.Max(acrossMax, math.Abs(d.Dot(flat)))
	}
	require.InDelta(t, m2fastener.NutCavityRadius, alongMax, 0.02, "%s: hex corners along Corner", label)
	require.InDelta(t, m2fastener.NutCavityRadius*math.Sqrt(3)/2, acrossMax, 0.02, "%s: hex flats", label)
}

// requireSiteWalls checks the material round the pockets and cavities: 1 mm or more under a head pocket and round
// each cavity, and the press wall from the bore (1.6 mm) at every preset.
func requireSiteWalls(t *testing.T, m *onewayclutch.Model, i int, site m2fastener.Site) {
	t.Helper()
	label := fmt.Sprintf("site %d", i+1)
	dm := m.Dimensions
	back, err := m.BackPlate.Bounds()
	require.NoError(t, err)
	cover, err := m.Cover.Bounds()
	require.NoError(t, err)
	// Material under the head pocket: the pocket's floor to the cover's inner face (the recessed annulus is the
	// lowest face the pocket's disc reaches, at CoverFrom + Recess).
	floor := site.At(m2fastener.HeadPocketDepth).Z
	require.GreaterOrEqual(t, floor-(cover.Min.Z+onewayclutch.Recess), 1.0, "%s: material under the head pocket", label)
	// Material beside the pocket: above it under the hub-side face (the recessed annulus).
	require.GreaterOrEqual(t, (back.Max.Z-onewayclutch.Recess)-site.At(site.Cavity).Z, 1.0,
		"%s: material above the cavity", label)
	// Radial walls from the plate rim.
	require.GreaterOrEqual(t, dm.PlateRadius-(math.Hypot(site.Head.X, site.Head.Y)+m2fastener.NutCavityRadius), 1.0,
		"%s: wall to the plate rim", label)

	// Wall from the D bore to each cavity and pocket, from the bore's outline: the round arc and the flat on +Y.
	d := distanceToBore(site.Head.X, site.Head.Y, dm.BoreRadius, dm.BoreFlat)
	// The back plate's pockets are turned so that a flat, not a corner, faces the bore (Corner is at the bolt's angle plus
	// 30 degrees), which is what keeps the 8 mm presets' wall at the press wall.
	require.GreaterOrEqual(t, d-m2fastener.NutCavityRadius*math.Sqrt(3)/2, onewayclutch.PressWall-1e-9, "%s: wall bore to nut cavity", label)
	require.GreaterOrEqual(t, d-m2fastener.HeadPocketRadius, onewayclutch.PressWall-1e-9, "%s: wall bore to head pocket", label)
}

// requireHubBoltWalls checks the hub's section at mid-height: each bolt hole keeps the press wall (1.6 mm) to the
// hub's outer loop, which holds the core, the slots and the arms.
func requireHubBoltWalls(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	dm := m.Dimensions
	s := fitSection(t, m.Hub, fitPlane(t, midHub))
	var outer []r3.Vec
	var holes [][]r3.Vec
	for _, loop := range s.Loops {
		switch efKindOf(loop, dm) {
		case efOuter:
			outer = loop
		case efHole:
			holes = append(holes, loop)
		}
	}
	require.NotNil(t, outer, "the hub's outer loop")
	require.Len(t, holes, len(onewayclutch.BoltAngles), "the hub's bolt holes")
	for i, hole := range holes {
		wall := math.Inf(1)
		for _, p := range hole {
			wall = math.Min(wall, fitNearest(p, [][]r3.Vec{outer}))
		}
		wall -= s.OffSurface.Base()
		t.Logf("bolt hole %d: wall to the hub's outer loop %.3f", i+1, wall)
		require.GreaterOrEqual(t, wall, onewayclutch.PressWall, "bolt hole %d: wall to the hub's outer loop", i+1)
	}
}

// distanceToBore returns the distance from (x, y) outside the D bore, a circle of radius r cut flat at y = f, to it.
func distanceToBore(x, y, r, f float64) float64 {
	best := math.Inf(1)
	const n = 7200
	for k := 0; k < n; k++ {
		a := 2 * math.Pi * float64(k) / n
		px, py := r*math.Cos(a), r*math.Sin(a)
		if py > f {
			continue
		}
		best = math.Min(best, math.Hypot(x-px, y-py))
	}
	half := math.Sqrt(r*r - f*f)
	for k := 0; k <= n; k++ {
		px := -half + 2*half*float64(k)/n
		best = math.Min(best, math.Hypot(x-px, y-f))
	}
	return best
}

// atZ is the plane z = const, normal +Z.
func atZ(t *testing.T, z float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}

// requireSiteClear checks, in sections across the bolt, that the bolt and the nut share no volume with the parts they
// pass through and keep the fastener clearances: 0.3 mm round the shank and the head, 0.2 mm round the nut.
func requireSiteClear(t *testing.T, m *onewayclutch.Model, i int, site m2fastener.Site) {
	t.Helper()
	type probe struct {
		part      *decad.Body
		with      *decad.Body
		name      string
		z         float64
		clearance float64
	}
	bolt, nut := m.Bolts[i].Body, m.Nuts[i].Body
	headMid := site.At(m2fastener.HeadPocketDepth / 2).Z
	nutMid := (site.At(site.Cavity).Z + site.At(site.CavityFar()).Z) / 2
	probes := []probe{
		{m.Cover, bolt, "bolt/cover head", headMid, 0.28},
		{m.Cover, bolt, "bolt/cover shank", site.At(m2fastener.HeadPocketDepth + 1).Z, 0.28},
		{m.Hub, bolt, "bolt/hub", onewayclutch.HubFrom + onewayclutch.HubHeight/2, 0.28},
		{m.BackPlate, bolt, "bolt/back plate", site.At(site.Cavity - 0.5).Z, 0.28},
		{m.BackPlate, nut, "nut/back plate", nutMid, 0.15},
		{m.BackPlate, bolt, "bolt/back plate pocket", nutMid, 0.5},
	}
	for _, pr := range probes {
		f := atZ(t, pr.z)
		// A chamfered hub is faceted and holds no section finer than 0.035. Every other pair is exact and takes
		// 0.01: a coarser chord shrinks the bolt hole's polygon and the clearance read from it (0.272 at 0.03 for the
		// 0.3 mm of the 8 mm hub's holes).
		tol := 0.01
		if pr.part == m.Hub && hubChamfered(t, m) {
			tol = fitCoarseTol
		}
		a, err := pr.with.SectionOutline(t.Context(), f, units.Millimeters(tol))
		require.NoError(t, err, pr.name)
		b, err := pr.part.SectionOutline(t.Context(), f, units.Millimeters(tol))
		require.NoError(t, err, pr.name)
		c, err := a.Clearance(b)
		require.NoError(t, err, pr.name)
		require.False(t, c.Overlaps, "site %d: %s overlaps at z %g", i+1, pr.name, pr.z)
		require.GreaterOrEqual(t, c.Distance.Value.Base(), pr.clearance, "site %d: %s at z %g", i+1, pr.name, pr.z)
	}
	// The nut stands clear of the hub and the cover by its z range, and of the ring by its radius (the ring teeth start at ToothRoot).
	nb, err := nut.Bounds()
	require.NoError(t, err)
	for name, other := range map[string]*decad.Body{"hub": m.Hub, "cover": m.Cover} {
		ob, err := other.Bounds()
		require.NoError(t, err, name)
		require.Less(t, nb.Max.Z, ob.Min.Z+1e-3, "site %d: nut clear of the %s", i+1, name)
	}
	require.Less(t, math.Hypot(site.Head.X, site.Head.Y)+m2fastener.NutCavityRadius, m.Dimensions.ToothRoot, "site %d: nut inside the ring teeth", i+1)
}
