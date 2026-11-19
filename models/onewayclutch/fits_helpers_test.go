package onewayclutch_test

import (
	"math"
	"sort"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/onewayclutch"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// Tolerances of the fit measurements, in mm. A section is cut at the finest chord of fitSectionTols that its body
// holds: the exact parts and the chamfered hubs (held mesh bound 0.006) take 0.01, and the ring's held mesh bound of
// 0.0285 takes 0.03. A coarser chord makes the section less exact (a bolt hole's polygon shrinks by the sagitta, and
// the proven off-surface bound grows into every margin). The printed-part meshes are 0.03 for the same reason, and only
// a body whose held bound refuses 0.03 falls back to fitCoarseTol. A mesh's bound is charged against every gap: the
// 8 mm hub to ring gap read 0.3476 against the 0.35 limit while the hub's chamfer held 0.045, and reads 0.372 now.
const (
	fitMeshTol   = 0.03
	fitCoarseTol = 0.035
	// fitCut is the angle of the axial section plane. It is off every lock face, which lie at 10 + 20k degrees (the
	// 20 degree tooth pitch), so the plane cuts the teeth through their ramps and never lies along one.
	fitCut = 7.0
	// midHub is the hub's mid-height; the ring's tooth band shares it.
	midHub = (onewayclutch.HubFrom + onewayclutch.HubFrom + onewayclutch.HubHeight) / 2
)

// fitBackRimZ is the middle of the back plate's rim where it overlaps the ring's counterbore: from the end of the
// plate's outer chamfer (z 0.5) to the cone's start on the plate's rim (z 3.0 on the 605ZZ clutch); the cover's is its
// mirror in midHub.
func fitBackRimZ(dm onewayclutch.Dimensions) float64 {
	return (onewayclutch.PlateChamfer + onewayclutch.BackPlateThickness - onewayclutch.Recess -
		(dm.PlateRadius - dm.ToothRoot)) / 2
}

// fitSectionTols are the chord tolerances fitSection tries, finest first.
var fitSectionTols = []float64{0.01, 0.02, fitMeshTol, fitCoarseTol}

type fitCase struct {
	name string
	p    onewayclutch.Params
}

// fitCases lists every preset in both directions, the wide bore on the 605ZZ preset and the 8 mm presets at 0.5 mm.
func fitCases() []fitCase {
	var out []fitCase
	for _, s := range onewayclutch.Shafts() {
		for _, d := range []onewayclutch.Direction{onewayclutch.Clockwise, onewayclutch.CounterClockwise} {
			p := onewayclutch.DefaultParams()
			p.Shaft, p.Freewheel = s.Name, d
			out = append(out, fitCase{name: s.Name + "/" + d.String(), p: p})
		}
	}
	wide := onewayclutch.DefaultParams()
	wide.BoreClearance = 0.35
	out = append(out, fitCase{name: "605ZZ/bore-0.35", p: wide})
	for _, w := range wideBoreCases() {
		out = append(out, fitCase{name: w.name, p: w.params})
	}
	return out
}

func fitBuild(t *testing.T, c fitCase) *onewayclutch.Model {
	t.Helper()
	m, _ := buildCase(t, c.p)
	return m
}

func fitFrame(t *testing.T, origin, u, v r3.Vec) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(origin, u, v)
	require.NoError(t, err)
	return f
}

func fitSection(t *testing.T, body *decad.Body, plane r3.Frame) decad.SectionOutline {
	t.Helper()
	var s decad.SectionOutline
	var err error
	for _, tol := range fitSectionTols {
		if s, err = body.SectionOutline(t.Context(), plane, units.Millimeters(tol)); err == nil {
			break
		}
		var bound *decad.MeshBoundError
		require.ErrorAs(t, err, &bound, "only a held mesh bound refuses a section")
	}
	require.NoError(t, err)
	require.NotEmpty(t, s.Loops)
	return s
}

// fitPlane is the horizontal section plane at height z.
func fitPlane(t *testing.T, z float64) r3.Frame {
	t.Helper()
	return fitFrame(t, r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
}

// fitAxial is the axial section plane at angle deg: u runs outward along the angle, v along +Z.
func fitAxial(t *testing.T, deg float64) r3.Frame {
	t.Helper()
	s, c := math.Sincos(deg * math.Pi / 180)
	return fitFrame(t, r3.Vec{}, r3.Vec{X: c, Y: s}, r3.Vec{Z: 1})
}

// fitPoints returns every vertex of every loop.
func fitPoints(s decad.SectionOutline) []r3.Vec {
	var out []r3.Vec
	for _, loop := range s.Loops {
		out = append(out, loop...)
	}
	return out
}

func fitSegDist(p, a, b r3.Vec) float64 {
	e := b.Sub(a)
	l := e.Dot(e)
	f := 0.0
	if l > 0 {
		f = math.Max(0, math.Min(1, p.Sub(a).Dot(e)/l))
	}
	return p.Sub(a.Add(e.Scale(f))).Len()
}

// fitNearest returns the least distance from p to any segment of the loops.
func fitNearest(p r3.Vec, loops [][]r3.Vec) float64 {
	best := math.Inf(1)
	for _, loop := range loops {
		for i, a := range loop {
			best = math.Min(best, fitSegDist(p, a, loop[(i+1)%len(loop)]))
		}
	}
	return best
}

// fitBoreLoop returns the loop of the section that is the D bore: the one wholly within 6 mm of the axis.
func fitBoreLoop(t *testing.T, s decad.SectionOutline) []r3.Vec {
	t.Helper()
	var found [][]r3.Vec
	for _, loop := range s.Loops {
		inside := true
		for _, p := range loop {
			inside = inside && math.Hypot(p.X, p.Y) < 6
		}
		if inside {
			found = append(found, loop)
		}
	}
	require.Len(t, found, 1, "the D bore loop")
	return found[0]
}

// fitDistanceToD returns the distance from p to the nominal D: the disk of radius r cut by the half plane y <= flat.
func fitDistanceToD(p r3.Vec, r, flat float64) float64 {
	inD := func(q r3.Vec) bool { return math.Hypot(q.X, q.Y) <= r+1e-9 && q.Y <= flat+1e-9 }
	corner := math.Sqrt(r*r - flat*flat)
	candidates := []r3.Vec{
		{X: corner, Y: flat}, {X: -corner, Y: flat},
		{X: p.X, Y: math.Min(p.Y, flat)}, // onto the flat's half plane
	}
	if n := math.Hypot(p.X, p.Y); n > 0 {
		candidates = append(candidates, r3.Vec{X: p.X * r / n, Y: p.Y * r / n}) // onto the disk
	}
	best := math.Inf(1)
	for _, q := range candidates {
		q.Z = 0
		if inD(q) {
			best = math.Min(best, math.Hypot(p.X-q.X, p.Y-q.Y))
		}
	}
	return best
}

// requireBoreFit requires every point of the body's bore, in the section at z, to lie the bore clearance outside the
// nominal D shaft, on the round and on the flat (the press fit, P1 and P2).
func requireBoreFit(t *testing.T, name string, body *decad.Body, z float64, m *onewayclutch.Model, c float64) {
	t.Helper()
	s := fitSection(t, body, fitPlane(t, z))
	loop := fitBoreLoop(t, s)
	off := s.OffSurface.Base() + 1e-6
	shaft := m.Dimensions.Shaft
	flat := shaft.Diameter/2 - shaft.FlatDepth
	lo := math.Inf(1)
	for _, p := range loop {
		d := fitDistanceToD(p, shaft.Diameter/2, flat)
		lo = math.Min(lo, d)
		// Each vertex lies on the round, c outside it, or on the flat, c outside it. The two meet at the bore's
		// corners, which stand more than c from the shaft's corners; that is the modeled radial clearance.
		onRound := math.Abs(math.Hypot(p.X, p.Y)-(shaft.Diameter/2+c)) <= off
		onFlat := math.Abs(p.Y-(flat+c)) <= off
		require.True(t, onRound || onFlat, "%s bore: vertex (%.3f, %.3f) is neither c outside the round nor the flat",
			name, p.X, p.Y)
	}
	t.Logf("%s bore: least clearance %.4f (modeled %.2f, off-surface %.4f)", name, lo, c, off)
	require.GreaterOrEqual(t, lo, c-off, "%s bore: least clearance", name)
}

// fitMeshOf tessellates body at the first of tols that its held mesh bound allows.
func fitMeshOf(t *testing.T, body *decad.Body, tols ...float64) meshgap.Mesh {
	t.Helper()
	var mesh *decad.Mesh
	var err error
	for _, tol := range tols {
		if mesh, err = body.Tessellate(t.Context(), units.Millimeters(tol), decad.WithVerification(decad.VerifyAll)); err == nil {
			break
		}
		var bound *decad.MeshBoundError
		require.ErrorAs(t, err, &bound, "only a held mesh bound refuses a mesh")
	}
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

func fitBodies(m *onewayclutch.Model) map[string]*decad.Body {
	out := map[string]*decad.Body{"hub": m.Hub, "ring": m.Ring, "back plate": m.BackPlate, "cover": m.Cover}
	for i := range m.Bolts {
		out["bolt "+string(rune('1'+i))] = m.Bolts[i].Body
		out["nut "+string(rune('1'+i))] = m.Nuts[i].Body
	}
	return out
}

// requireFitPairs measures the gap between every pair of bodies on the meshes: T1 (hub to each plate), the ring to
// each plate, R5 (hub to ring) and no overlap anywhere else. A bolt rests in its nut and on its pocket floor by design.
func requireFitPairs(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	bodies := fitBodies(m)
	names := make([]string, 0, len(bodies))
	meshes := map[string]meshgap.Mesh{}
	for name, body := range bodies {
		names = append(names, name)
		tols := []float64{fitMeshTol, fitCoarseTol}
		if len(name) == 5 { // a bolt or a nut: a 0.05 mm mesh is fine
			tols = []float64{0.05}
		}
		meshes[name] = fitMeshOf(t, body, tols...)
	}
	sort.Strings(names)
	minimum := func(a, b string) float64 {
		switch {
		case a == "back plate" && b == "hub", a == "cover" && b == "hub":
			return onewayclutch.TightFit - onewayclutch.FitMargin // Tight 0.2
		case a == "back plate" && b == "ring", a == "cover" && b == "ring", a == "hub" && b == "ring":
			return onewayclutch.RegularFit - onewayclutch.FitMargin // Regular 0.35
		}
		return 0
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			x, y := a, b // names are sorted, so x < y
			if isBoltNutPair(a, b) || isBoltCoverPair(a, b) {
				continue
			}
			gap, clear, err := meshes[a].Exceeds(meshes[b], 0)
			require.NoError(t, err)
			require.True(t, clear, "%s and %s overlap or touch", a, b)
			if want := minimum(x, y); want > 0 {
				t.Logf("gap %s / %s: proven at least %.3f", a, b, gap)
				require.GreaterOrEqual(t, gap, want, "%s to %s", a, b)
			}
		}
	}
}

// fitRadial returns the vertices of s with radial coordinate (the section's u) in [lo, hi] as (radial, z) pairs.
func fitRadial(s decad.SectionOutline, lo, hi float64) [][2]float64 {
	u := s.Plane.U()
	var out [][2]float64
	for _, p := range fitPoints(s) {
		if r := p.Dot(u); r >= lo && r <= hi {
			out = append(out, [2]float64{r, p.Z})
		}
	}
	return out
}

func fitZRange(pts [][2]float64, below, above float64) (lo, hi float64, n int) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		if p[1] < below || p[1] > above {
			continue
		}
		lo, hi, n = math.Min(lo, p[1]), math.Max(hi, p[1]), n+1
	}
	return lo, hi, n
}

// fitAxialFits measures R1, R3 and R4 in the axial plane, each side of the hub, and returns the ring's tooth band
// height. R1 is the hub's end to the plate annulus, R3 the plate cone to the ring cone (normal) and R4 the ring's
// tooth end to the plate annulus.
func fitAxialFits(t *testing.T, m *onewayclutch.Model) (bandHeight float64) {
	t.Helper()
	plane := fitAxial(t, fitCut)
	ring := fitSection(t, m.Ring, plane)
	back, cover := fitSection(t, m.BackPlate, plane), fitSection(t, m.Cover, plane)
	hub, err := m.Hub.Bounds()
	require.NoError(t, err)
	hubBound := hub.Bound.Base()
	const regular = onewayclutch.RegularFit - onewayclutch.FitMargin

	// The annulus floor is the plate's vertices from the boss bevel out to the cone.
	dm := m.Dimensions
	annulusLo, annulusHi := dm.BossRadius+0.3, dm.ToothRoot-0.1
	_, backTop, n := fitZRange(fitRadial(back, annulusLo, annulusHi), 0, midHub)
	require.NotZero(t, n)
	coverBottom, _, n := fitZRange(fitRadial(cover, annulusLo, annulusHi), midHub, 30)
	require.NotZero(t, n)
	// The ring's tooth end faces: its vertices from the tooth tips out to the root radius.
	toothLo, _, n := fitZRange(fitRadial(ring, dm.ToothTip-0.1, dm.ToothRoot+0.01), 0, midHub)
	require.NotZero(t, n)
	_, toothHi, n := fitZRange(fitRadial(ring, dm.ToothTip-0.1, dm.ToothRoot+0.01), midHub, 30)
	require.NotZero(t, n)

	off := func(a, b decad.SectionOutline) float64 { return a.OffSurface.Base() + b.OffSurface.Base() }
	r1Back := hub.Min.Z - hubBound - backTop - back.OffSurface.Base()
	r1Cover := coverBottom - cover.OffSurface.Base() - (hub.Max.Z + hubBound)
	r4Back := toothLo - backTop - off(ring, back)
	r4Cover := coverBottom - toothHi - off(ring, cover)
	t.Logf("R1 pawl to annulus: back %.3f cover %.3f (modeled 0.65)", r1Back, r1Cover)
	t.Logf("R4 tooth end to annulus: back %.3f cover %.3f (modeled 0.85)", r4Back, r4Cover)
	for name, gap := range map[string]float64{"R1 back": r1Back, "R1 cover": r1Cover, "R4 back": r4Back, "R4 cover": r4Cover} {
		require.GreaterOrEqual(t, gap, regular, name)
	}

	// R3: from the cone's inner vertex of each plate to the ring's cone, the nearest ring segment.
	for name, plate := range map[string]decad.SectionOutline{"back": back, "cover": cover} {
		best, found := math.Inf(1), 0
		for _, p := range fitPoints(plate) {
			if r := p.Dot(plane.U()); math.Abs(r-dm.ToothRoot) <= 0.02 {
				best, found = math.Min(best, fitNearest(p, ring.Loops)), found+1
			}
		}
		require.NotZero(t, found, "R3 %s: the cone's inner vertex", name)
		best -= off(plate, ring)
		t.Logf("R3 %s cone: %.3f (modeled 0.60)", name, best)
		require.GreaterOrEqual(t, best, regular, "R3 "+name)
	}
	return toothHi - toothLo
}

// fitRimFits measures R2, the ring counterbore to each plate rim, in the section through the middle of the rim.
func fitRimFits(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	rimZ := fitBackRimZ(m.Dimensions)
	for name, c := range map[string]struct {
		plate *decad.Body
		z     float64
	}{"back": {m.BackPlate, rimZ}, "cover": {m.Cover, 2*midHub - rimZ}} {
		plane := fitPlane(t, c.z)
		plate, ring := fitSection(t, c.plate, plane), fitSection(t, m.Ring, plane)
		gap, err := plate.Clearance(ring)
		require.NoError(t, err)
		require.False(t, gap.Overlaps, "R2 %s rim overlaps the ring", name)
		t.Logf("R2 %s rim: %.3f (modeled 0.40)", name, gap.Lower().Base())
		require.GreaterOrEqual(t, gap.Lower().Base(), onewayclutch.RegularFit-onewayclutch.FitMargin, "R2 "+name)
	}
}

// fitMapAngle maps an angle of the counter-clockwise layout to the built one for sign sigma; it is its own inverse.
func fitMapAngle(sigma, layoutDeg float64) float64 { return 90 + sigma*(layoutDeg-90) }

// fitArmFits measures L1: from each ring tooth tip to the hub's arm outer arcs, in the section at mid-height.
func fitArmFits(t *testing.T, m *onewayclutch.Model) {
	t.Helper()
	plane := fitPlane(t, midHub)
	dm := m.Dimensions
	hub, ring := fitSection(t, m.Hub, plane), fitSection(t, m.Ring, plane)
	var arcs [][]r3.Vec // the arc segments, two-point loops, so fitNearest reads them as one segment each way
	for _, loop := range hub.Loops {
		for i, a := range loop {
			b := loop[(i+1)%len(loop)]
			if math.Abs(math.Hypot(a.X, a.Y)-dm.ArmOuter) < 0.02 && math.Abs(math.Hypot(b.X, b.Y)-dm.ArmOuter) < 0.02 {
				arcs = append(arcs, []r3.Vec{a, b})
			}
		}
	}
	require.NotEmpty(t, arcs)
	best, tips := math.Inf(1), 0
	for _, p := range fitPoints(ring) {
		if math.Abs(math.Hypot(p.X, p.Y)-dm.ToothTip) < 0.02 {
			best, tips = math.Min(best, fitNearest(p, arcs)), tips+1
		}
	}
	require.GreaterOrEqual(t, tips, onewayclutch.Teeth, "ring tooth tip vertices")
	best -= hub.OffSurface.Base() + ring.OffSurface.Base()
	t.Logf("L1 arm to tooth tips: %.3f (modeled 0.55)", best)
	require.GreaterOrEqual(t, best, onewayclutch.LooseFit-onewayclutch.FitMargin)
}

// fitCrossings returns, sorted, the radii where the ray from the axis at deg crosses the section's loops.
func fitCrossings(s decad.SectionOutline, deg float64) []float64 {
	sn, cs := math.Sincos(deg * math.Pi / 180)
	cross := func(ax, ay, bx, by float64) float64 { return ax*by - ay*bx }
	var out []float64
	for _, loop := range s.Loops {
		for i, a := range loop {
			b := loop[(i+1)%len(loop)]
			ex, ey := b.X-a.X, b.Y-a.Y
			den := cross(cs, sn, ex, ey)
			if math.Abs(den) < 1e-12 {
				continue
			}
			tt := cross(a.X, a.Y, ex, ey) / den
			ss := cross(a.X, a.Y, cs, sn) / den
			if ss >= 0 && ss < 1 && tt > 0 {
				out = append(out, tt)
			}
		}
	}
	sort.Float64s(out)
	return out
}

func fitAngle(p r3.Vec) float64 { return math.Atan2(p.Y, p.X) * 180 / math.Pi }
