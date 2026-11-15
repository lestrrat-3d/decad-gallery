package onewayclutch_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
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
			m, _ := buildCase(t, name, p)
			for i, site := range m.Sites {
				requireSiteFastener(t, m, i, site)
				requireSiteWalls(t, m, i, site)
				requireSiteClear(t, m, i, site)
			}
		})
	}
}

// requireSiteFastener checks one site's bolt and nut by bounds: the head flush with the cover's outer face and inside
// its pocket, the tip inside the cavity's far face and above z 0, the nut inside its cavity.
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
	require.LessOrEqual(t, bb.Min.Z, site.At(site.CavityFar()).Z+tol, "%s: tip at or below the cavity's far face", label)
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
	require.Greater(t, nb.Min.Z, back.Min.Z+m2fastener.Stock*0.5, "%s: back plate encloses the nut from below", label)
	require.Less(t, nb.Max.Z, back.Max.Z, "%s: nut inside the back plate", label)
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
	// Material beside the cavity: below it, and above it under the hub-side face (the recessed annulus).
	require.GreaterOrEqual(t, site.At(site.CavityFar()).Z-back.Min.Z, 1.0, "%s: material below the cavity", label)
	require.GreaterOrEqual(t, (back.Max.Z-onewayclutch.Recess)-site.At(site.Cavity).Z, 1.0,
		"%s: material above the cavity", label)
	// Radial walls from the plate rim.
	require.GreaterOrEqual(t, onewayclutch.PlateRadius-(math.Hypot(site.Head.X, site.Head.Y)+m2fastener.NutCavityRadius), 1.0,
		"%s: wall to the plate rim", label)

	// Wall from the D bore to each cavity and pocket, from the bore's outline: the round arc and the flat on +Y.
	d := distanceToBore(site.Head.X, site.Head.Y, dm.BoreRadius, dm.BoreFlat)
	require.GreaterOrEqual(t, d-m2fastener.NutCavityRadius, onewayclutch.PressWall-1e-9, "%s: wall bore to nut cavity", label)
	require.GreaterOrEqual(t, d-m2fastener.HeadPocketRadius, onewayclutch.PressWall-1e-9, "%s: wall bore to head pocket", label)
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
		{m.Hub, bolt, "bolt/hub", 10, 0.28},
		{m.BackPlate, bolt, "bolt/back plate", site.At(site.Cavity - 0.5).Z, 0.28},
		{m.BackPlate, nut, "nut/back plate", nutMid, 0.18},
		{m.BackPlate, bolt, "bolt/back plate cavity", nutMid, 0.5},
	}
	for _, pr := range probes {
		f := atZ(t, pr.z)
		a, err := pr.with.SectionOutline(t.Context(), f, units.Millimeters(0.01))
		require.NoError(t, err, pr.name)
		b, err := pr.part.SectionOutline(t.Context(), f, units.Millimeters(0.01))
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
	require.Less(t, math.Hypot(site.Head.X, site.Head.Y)+m2fastener.NutCavityRadius, onewayclutch.ToothRoot, "site %d: nut inside the ring teeth", i+1)
}
