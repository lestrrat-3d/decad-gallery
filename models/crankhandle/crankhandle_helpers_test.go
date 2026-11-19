package crankhandle_test

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// crankCase is one parameter case. Every case changes the arm's outline or its bore.
type crankCase struct {
	name   string
	params func() crankhandle.Params
}

var crankCases = []crankCase{
	{"605ZZ", crankhandle.Params605ZZ},
	{"608-2RS", crankhandle.Params6082RS},
	{"regular-bore", func() crankhandle.Params {
		p := crankhandle.Params605ZZ()
		p.BoreClearance = 0.35
		return p
	}},
	{"long-throw", func() crankhandle.Params {
		p := crankhandle.Params6082RS()
		p.Throw = 80
		return p
	}},
}

type builtCase struct {
	m   *crankhandle.Model
	doc *decad.Document
}

// builtCases holds each case's build, shared by the tests: Build takes seconds and the tests only read its bodies.
var builtCases = map[string]builtCase{}

// built builds a case on first use and checks what every case shares: thirteen bodies (the arm, the pin, the handle, five
// bolts and five nuts), each in its container, and every sketch fully constrained.
func built(t *testing.T, c crankCase) (*crankhandle.Model, *decad.Document) {
	t.Helper()
	if b, ok := builtCases[c.name]; ok {
		return b.m, b.doc
	}
	doc := crankhandle.NewDocument()
	m, err := crankhandle.Build(t.Context(), doc, c.params())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 13)
	require.NotEmpty(t, m.Sketches)
	for _, s := range m.Sketches {
		require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), s))
	}
	builtCases[c.name] = builtCase{m: m, doc: doc}
	return m, doc
}

// section cuts body by the plane f at the finest chord the body allows.
func section(t *testing.T, body *decad.Body, f r3.Frame) decad.SectionOutline {
	t.Helper()
	var o decad.SectionOutline
	var err error
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		if o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			return o
		}
	}
	require.NoError(t, err)
	return o
}

// atZ is the plane z = const of the local frame.
func atZ(t *testing.T, z float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	return f
}

// clearance returns the polyline clearance between a and b in the plane f, failing on an overlap.
func clearance(t *testing.T, a, b *decad.Body, f r3.Frame) decad.SectionClearance {
	t.Helper()
	oa, ob := section(t, a, f), section(t, b, f)
	require.NotEmpty(t, oa.Loops)
	require.NotEmpty(t, ob.Loops)
	c, err := oa.Clearance(ob)
	require.NoError(t, err)
	require.False(t, c.Overlaps, "overlap at %v", f.Origin())
	return c
}

// distanceToLoop returns the least distance from v to the closed polyline l in the XY plane.
func distanceToLoop(v r3.Vec, l []r3.Vec) float64 {
	least := math.Inf(1)
	for i := range l {
		a, b := l[i], l[(i+1)%len(l)]
		ab := r3.Vec{X: b.X - a.X, Y: b.Y - a.Y}
		av := r3.Vec{X: v.X - a.X, Y: v.Y - a.Y}
		u := 0.0
		if d := ab.Dot(ab); d > 0 {
			u = math.Max(0, math.Min(1, av.Dot(ab)/d))
		}
		least = math.Min(least, math.Hypot(av.X-u*ab.X, av.Y-u*ab.Y))
	}
	return least
}

// atY is the plane y = const of the local frame, normal +Y: a layer of the pin's halves as they print.
func atY(t *testing.T, y float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Y: y}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	require.NoError(t, err)
	return f
}

// requireCrossBolts checks the two cross bolts that clamp the pin's halves: each head's top face is flush with the +Y
// flat and inside its pocket in the +Y half, each tip ends flush with the -Y flat, each nut lies inside its hexagonal
// pocket open to the -Y flat, below it, with the bolt through the whole nut, so the bolt crosses the seam, and no bolt or
// nut shares volume with the pin in sections across and along the bolt. It also checks 1 mm or more of pin round every
// pocket and the shape of the nut pockets (requireCrossPocket).
func requireCrossBolts(t *testing.T, p crankhandle.Params, m *crankhandle.Model) {
	t.Helper()
	seam := p.Throw
	for i, z := range crankhandle.CrossZ {
		bb, err := m.CrossBolts[i].Body.Bounds()
		require.NoError(t, err)
		tol := bb.Bound.Base() + 2e-3
		require.InDelta(t, seam+crankhandle.CrossFlat, bb.Max.Y, tol, "cross bolt %d: head top flush with the +Y flat", i+1)
		require.InDelta(t, seam+crankhandle.CrossFlat-m.CrossSites[i].Tip(), bb.Min.Y, tol, "cross bolt %d: tip", i+1)
		require.GreaterOrEqual(t, bb.Min.Y, seam-crankhandle.CrossFlat-tol, "cross bolt %d: tip inside the -Y flat", i+1)
		require.InDelta(t, seam-crankhandle.CrossFlat, bb.Min.Y, tol, "cross bolt %d: tip flush with the -Y flat", i+1)
		require.Less(t, math.Max(math.Max(-bb.Min.X, bb.Max.X), math.Max(z-bb.Min.Z, bb.Max.Z-z)),
			m2fastener.HeadPocketRadius, "cross bolt %d: head inside its pocket", i+1)
		require.Less(t, bb.Min.Y, seam, "cross bolt %d crosses the seam", i+1)
		nb, err := m.CrossNuts[i].Body.Bounds()
		require.NoError(t, err)
		floor := seam + crankhandle.CrossFlat - crankhandle.CrossCavity // the pocket's floor
		ntol := nb.Bound.Base() + 2e-3
		require.LessOrEqual(t, nb.Max.Y, floor-m2fastener.NutClearance+ntol, "cross nut %d inside its pocket", i+1)
		require.GreaterOrEqual(t, nb.Min.Y, seam-crankhandle.CrossFlat+m2fastener.NutClearance-ntol,
			"cross nut %d below the -Y flat", i+1)
		require.Less(t, bb.Min.Y, nb.Min.Y, "cross bolt %d runs through the whole nut", i+1)
		for _, d := range []float64{nb.Min.X, nb.Max.X} {
			require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius, "cross nut %d: corners inside its pocket", i+1)
		}
		for _, d := range []float64{nb.Min.Z - z, nb.Max.Z - z} {
			require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius*math.Sqrt(3)/2+ntol,
				"cross nut %d: flats inside its pocket", i+1)
		}
		for _, y := range []float64{3.5, 2.5, 1.0, -0.5, -1.5, -2.1, -2.5, -3.0, -3.5, -3.9} {
			requireNoOverlap(t, m.Pin, atY(t, seam+y), fmt.Sprintf("cross fastener %d at y %g", i+1, y),
				m.CrossBolts[i].Body, m.CrossNuts[i].Body)
		}
		for name, f := range map[string]r3.Frame{"x 0": atX(t, 0), "z": atZ(t, z)} {
			requireNoOverlap(t, m.Pin, f, fmt.Sprintf("cross fastener %d along its bolt at %s", i+1, name),
				m.CrossBolts[i].Body, m.CrossNuts[i].Body)
		}
	}
	requireCrossPocket(t, p, m)
	r, flat := crankhandle.ShankR, crankhandle.CrossFlat
	// The nut pocket's corners point across the pin, so the shank's round at the -Y flat leaves less than 1 mm beside
	// each corner. The exception is listed in instructions.md ("Walls"); the test pins its value.
	nutWall := math.Sqrt(r*r-flat*flat) - m2fastener.NutCavityRadius
	require.InDelta(t, 0.7344, nutWall, 0.0001, "wall round the nut pocket's corners at the -Y flat")
	require.Less(t, nutWall, 1.0, "the listed exception")
	for name, wall := range map[string]float64{
		"round the head pocket at the flat":   math.Sqrt(r*r-flat*flat) - m2fastener.HeadPocketRadius,
		"of flat beyond the head pocket":      crankhandle.CrossFlatHalf - m2fastener.HeadPocketRadius,
		"under the head pocket to the seam":   flat - m2fastener.HeadPocketDepth,
		"between the pocket's floor and seam": crankhandle.CrossCavity - flat,
	} {
		require.GreaterOrEqual(t, wall, 1.0-1e-9, "material %s", name)
	}
	require.LessOrEqual(t, m.CrossSites[0].Tip(), 2*flat, "tip inside the -Y flat")
}

// requireNoOverlap requires that no part shares volume with host in the section of both at frame f; name tells the
// case in a failure.
func requireNoOverlap(t *testing.T, host *decad.Body, f r3.Frame, name string, parts ...*decad.Body) {
	t.Helper()
	a := section(t, host, f)
	for _, part := range parts {
		b := section(t, part, f)
		if len(a.Loops) == 0 || len(b.Loops) == 0 {
			continue
		}
		cl, err := a.Clearance(b)
		require.NoError(t, err)
		require.False(t, cl.Overlaps, "%s: overlap", name)
	}
}

// requireCrossPocket checks each cross nut's pocket in sections of the pin, whose corners point across the pin (+/-X)
// and whose flats lie along its axis. Across the bolt, 1 mm inside the -Y flat, its loop round the bolt's line comes no nearer the line than the nut's flats allow and no farther than the circle
// through the pocket's corners, and its least distance is under the nut's corner radius, so the nut cannot turn in it
// while its corners keep their room. Along the bolt in the plane x = 0, the pocket opens to the -Y flat and the pin is
// solid beyond its floor and beyond its corners.
func requireCrossPocket(t *testing.T, p crankhandle.Params, m *crankhandle.Model) {
	t.Helper()
	seam := p.Throw
	across := section(t, m.Pin, atY(t, seam-crankhandle.CrossFlat+1))
	bound := across.OffSurface.Base() + 1e-3
	along := section(t, m.Pin, atX(t, 0))
	zy := func(v r3.Vec) (float64, float64) { return v.Z, v.Y }
	for i, z := range crankhandle.CrossZ {
		near, far := math.Inf(1), math.Inf(-1)
		across0, along0 := 0.0, 0.0
		found := false
		for _, l := range across.Loops {
			var ds []float64
			for _, v := range l {
				ds = append(ds, math.Hypot(v.X, v.Z-z))
			}
			if slices.Max(ds) > m2fastener.NutCavityRadius+0.5 {
				continue
			}
			found = true
			for _, v := range l {
				across0, along0 = math.Max(across0, math.Abs(v.X)), math.Max(along0, math.Abs(v.Z-z))
			}
			for j := range l {
				a, b := l[j], l[(j+1)%len(l)]
				xz := func(v r3.Vec) r3.Vec { return r3.Vec{X: v.X, Y: v.Z - z} }
				near = math.Min(near, distanceToLoop(r3.Vec{}, []r3.Vec{xz(a), xz(b)}))
				far = math.Max(far, math.Hypot(a.X, a.Z-z))
			}
		}
		require.True(t, found, "cross nut pocket %d: its loop round the bolt's line", i+1)
		// A diagonal runs across the pin: the corners point to +X and -X and the flats lie along the pin's axis.
		require.InDelta(t, m2fastener.NutCavityRadius, across0, bound, "nut pocket %d: corners across the pin", i+1)
		require.InDelta(t, m2fastener.NutCavityRadius*math.Sqrt(3)/2, along0, bound, "nut pocket %d: flats along the pin", i+1)
		require.InDelta(t, m2fastener.NutCavityRadius*math.Sqrt(3)/2, near, bound, "nut pocket %d: its flats", i+1)
		require.Less(t, near, m2nut.AcrossCorners()/2, "nut pocket %d: the nut's corners cannot pass its flats", i+1)
		require.LessOrEqual(t, far, m2fastener.NutCavityRadius+bound, "nut pocket %d: its corners", i+1)
		require.Greater(t, far, m2nut.AcrossCorners()/2+0.05, "nut pocket %d: room for the nut's corners", i+1)
		flatY := seam - crankhandle.CrossFlat
		require.False(t, inSection(along, zy, z+1.8, flatY+0.1), "nut pocket %d open to the -Y flat", i+1)
		require.True(t, inSection(along, zy, z+m2fastener.NutCavityRadius+0.5, flatY+0.1),
			"pin beyond nut pocket %d's corner", i+1)
		floor := seam + crankhandle.CrossFlat - crankhandle.CrossCavity
		require.True(t, inSection(along, zy, z+1.8, floor+0.4), "pin beyond nut pocket %d's floor", i+1)
	}
}

// armExempt lists the arm's retained corners: the hexagonal pocket, clamp nut pocket 1's floor round cut at a slant and
// each clamp bolt's head pocket floor rim. The slit's end round reads no finding at the 1.5 mm slit.
func armExempt(m *crankhandle.Model, pocket cornercheck.Exempt) []cornercheck.Exempt {
	dm := m.Dimensions
	out := []cornercheck.Exempt{pocket}
	for i, site := range dm.ClampSites {
		floor, z := crankhandle.ArmRadius-site.Cavity, site.Head.Z
		if i == 0 {
			// The pocket's pointed roof and floor slopes lie 30 degrees off the bed, so a layer cuts the 0.6 mm round of the
			// floor's outline at a slant and reads it as a turn over 45 degrees (instructions.md, "Retained sharp
			// corners"). Only pocket 1's layers read it: no layer section reads pocket 2's round, and an exemption that
			// claims no finding fails cornercheck.
			out = append(out, cornercheck.Exempt{
				Reason: "clamp nut pocket 1 floor round cut at a slant",
				Near: func(v r3.Vec) bool {
					return v.X < floor-0.05 && v.X > floor-crankhandle.NutPocketRound-0.1 &&
						math.Hypot(v.Y-dm.ClampY, v.Z-z) <= m2fastener.NutCavityRadius+0.05
				},
			})
		}
		out = append(out, cornercheck.Exempt{
			Reason: fmt.Sprintf("clamp head pocket %d floor rim", i+1),
			Near: func(v r3.Vec) bool {
				return math.Abs(v.X-(crankhandle.ArmRadius-m2fastener.HeadPocketDepth)) < 0.05 &&
					math.Hypot(v.Y-dm.ClampY, v.Z-z) <= m2fastener.HeadPocketRadius+0.05
			},
		})
	}
	return out
}

// crossPocketNear reports whether a finding point is at one of the six corners of cross nut pocket i: in the pocket's
// band along the bolt, within the circle through its corners, and within 10 degrees about the bolt's line of a corner
// direction, which points across the pin (30, 90, ... 330 degrees from local Z, so 90 is +X).
func crossPocketNear(p crankhandle.Params, i int) func(r3.Vec) bool {
	z := crankhandle.CrossZ[i]
	return func(v r3.Vec) bool {
		if v.Y < p.Throw-crankhandle.CrossFlat-0.05 || v.Y > p.Throw-crankhandle.CrossFlat+m2fastener.NutCavityHeight+0.05 {
			return false
		}
		if math.Hypot(v.X, v.Z-z) > m2fastener.NutCavityRadius+0.05 || math.Hypot(v.X, v.Z-z) < 2.2 {
			return false
		}
		deg := math.Abs(math.Atan2(v.X, v.Z-z))*180/math.Pi - 30
		rem := math.Mod(deg+60, 60)
		return math.Min(rem, 60-rem) <= 10
	}
}

// crossPocketCorners lists the six sharp corners of each cross nut pocket in the pin, the corners that mate with the
// nut's sharp corners (instructions.md, "Retained sharp corners").
func crossPocketCorners(p crankhandle.Params, m *crankhandle.Model) []cornercheck.Exempt {
	var out []cornercheck.Exempt
	for i := range m.CrossSites {
		out = append(out, cornercheck.Exempt{
			Reason: fmt.Sprintf("cross nut pocket %d corners", i+1),
			Near:   crossPocketNear(p, i),
		})
	}
	return out
}

// atX is the plane x = const of the local frame, normal +X, across the clamp bolt's line.
func atX(t *testing.T, x float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}

// requireClampBolt checks clamp bolt i across the arm's hub: an M2 x 16 whose head's top face is flush with the +X
// side face and inside its pocket, which crosses the slit and whose tip ends inside the -X side face, and whose nut
// lies inside its hexagonal pocket, below the -X side face; neither bolt nor nut shares volume with the arm in sections
// along and across the bolt. It also checks the pocket: open to the -X side face, a hexagon whose flats the nut's
// corners cannot pass, so the nut cannot turn, with room for the nut's corners.
func requireClampBolt(t *testing.T, m *crankhandle.Model, i int) {
	t.Helper()
	dm, site := m.Dimensions, m.ClampSites[i]
	mid := site.Head.Z
	floor := crankhandle.ArmRadius - site.Cavity // the nut pocket's floor
	bb, err := m.ClampBolts[i].Body.Bounds()
	require.NoError(t, err)
	tol := bb.Bound.Base() + 2e-3
	require.Equal(t, crankhandle.ClampBolt, site.Length)
	require.Contains(t, m2bolt.StandardLengths(), site.Length)
	require.InDelta(t, crankhandle.ArmRadius, bb.Max.X, tol, "clamp bolt: head top flush with the +X side face")
	require.InDelta(t, crankhandle.ArmRadius-site.Tip(), bb.Min.X, tol, "clamp bolt: tip")
	require.Greater(t, bb.Min.X, -crankhandle.ArmRadius+m2fastener.Stock, "clamp bolt: tip inside the -X side face")
	require.Less(t, math.Max(math.Max(bb.Max.Y-dm.ClampY, dm.ClampY-bb.Min.Y), math.Max(bb.Max.Z-mid, mid-bb.Min.Z)),
		m2fastener.HeadPocketRadius, "clamp bolt: head inside its pocket")
	require.Less(t, bb.Min.X, -crankhandle.SlitWidth/2, "clamp bolt crosses the slit")
	nb, err := m.ClampNuts[i].Body.Bounds()
	require.NoError(t, err)
	ntol := nb.Bound.Base() + 2e-3
	require.GreaterOrEqual(t, nb.Min.X, floor-m2fastener.NutCavityHeight-ntol, "clamp nut below the -X side face")
	require.LessOrEqual(t, nb.Max.X, floor-m2fastener.NutClearance+ntol, "clamp nut inside its pocket")
	require.Less(t, bb.Min.X, nb.Min.X, "clamp bolt runs through the whole nut")
	for _, d := range []float64{nb.Min.Z - mid, nb.Max.Z - mid} {
		require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius, "clamp nut: corners inside its pocket")
	}
	for _, d := range []float64{nb.Min.Y - dm.ClampY, nb.Max.Y - dm.ClampY} {
		require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius*math.Sqrt(3)/2+ntol,
			"clamp nut: flats inside its pocket")
	}
	requireClampPocket(t, m, i)
	frames := map[string]r3.Frame{}
	for _, dz := range []float64{-1.7, -0.8, 0, 0.8, 1.7} {
		frames[fmt.Sprintf("z %g", mid+dz)] = atZ(t, mid+dz)
	}
	for _, x := range []float64{9.5, 8.5, 6, 3, 0, -3, -6, -7.9, -8.3, -9, -9.7} {
		frames[fmt.Sprintf("x %g", x)] = atX(t, x)
	}
	for name, f := range frames {
		a := section(t, m.Arm, f)
		for _, part := range []*decad.Body{m.ClampBolts[i].Body, m.ClampNuts[i].Body} {
			b := section(t, part, f)
			if len(a.Loops) == 0 || len(b.Loops) == 0 {
				continue
			}
			cl, err := a.Clearance(b)
			require.NoError(t, err)
			require.False(t, cl.Overlaps, "clamp fastener overlaps the arm at %s", name)
		}
	}
}

// inSection reports whether the point (u, v) of the plane, given in the section's own coordinates by uv, lies in the
// section's material, by the even-odd rule over its loops.
func inSection(o decad.SectionOutline, uv func(r3.Vec) (float64, float64), u, v float64) bool {
	crossings := 0
	for _, l := range o.Loops {
		for i := range l {
			au, av := uv(l[i])
			bu, bv := uv(l[(i+1)%len(l)])
			if (av > v) != (bv > v) && u < au+(v-av)/(bv-av)*(bu-au) {
				crossings++
			}
		}
	}
	return crossings%2 == 1
}

// requireClampPocket checks the clamp nut's pocket in sections of the arm. Across the bolt, 1 mm inside the -X side
// face, its loop round the bolt's axis comes no nearer the axis than the nut's flats allow and no farther than the
// circle through the pocket's corners, and its least distance is under the nut's corner radius, so the nut cannot turn
// in it while its corners keep their room. Along the bolt at mid thickness, the pocket opens to the -X side face and
// the arm is solid beyond its floor and beyond its corners.
func requireClampPocket(t *testing.T, m *crankhandle.Model, i int) {
	t.Helper()
	dm := m.Dimensions
	mid := m.ClampSites[i].Head.Z
	floor := crankhandle.ArmRadius - m.ClampSites[i].Cavity
	across := section(t, m.Arm, atX(t, -crankhandle.ArmRadius+1))
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
	along := section(t, m.Arm, atZ(t, mid))
	xy := func(v r3.Vec) (float64, float64) { return v.X, v.Y }
	require.False(t, inSection(along, xy, -crankhandle.ArmRadius+0.3, dm.ClampY+1.8), "nut pocket open to the -X side face")
	require.True(t, inSection(along, xy, -crankhandle.ArmRadius+0.3, dm.ClampY+m2fastener.NutCavityRadius+0.5),
		"arm beyond the nut pocket's corner")
	require.True(t, inSection(along, xy, floor+0.4, dm.ClampY+1.8), "arm beyond the nut pocket's floor")
}

// clampOutlineDistance is the least distance from (x, y) to the arm's outline behind the shaft: the sides below y = 0,
// the corner arcs and the end arc, each arc sampled every 0.0005 rad.
func clampOutlineDistance(dm crankhandle.Dimensions, x, y float64) float64 {
	pt := r3.Vec{X: x, Y: y}
	cR, cB := dm.CornerCenter(), dm.EndCenter()
	pR := cB.Add(cR.Sub(cB).Scale(crankhandle.ClampEndR / cR.Sub(cB).Len()))
	least := math.Inf(1)
	for a := math.Atan2(pR.Y-cR.Y, pR.X-cR.X); a <= 0; a += 0.0005 {
		q := cR.Add(r3.Vec{X: crankhandle.ClampCornerR * math.Cos(a), Y: crankhandle.ClampCornerR * math.Sin(a)})
		least = math.Min(least, math.Min(pt.Sub(q).Len(), pt.Sub(r3.Vec{X: -q.X, Y: q.Y}).Len()))
	}
	halfAngle := math.Atan2(pR.X-cB.X, -(pR.Y - cB.Y))
	for a := -halfAngle; a <= halfAngle; a += 0.0005 {
		q := cB.Add(r3.Vec{X: crankhandle.ClampEndR * math.Sin(a), Y: -crankhandle.ClampEndR * math.Cos(a)})
		least = math.Min(least, pt.Sub(q).Len())
	}
	side := math.Max(dm.SideEnd, math.Min(0, y))
	for _, sx := range []float64{1, -1} {
		least = math.Min(least, pt.Sub(r3.Vec{X: sx * crankhandle.ArmRadius, Y: side}).Len())
	}
	return least
}
