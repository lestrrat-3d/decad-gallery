package crankhandle_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
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

// built builds a case on first use and checks what every case shares: nine bodies (the arm, the pin, the handle, three
// bolts and three nuts), each in its container, and every sketch fully constrained.
func built(t *testing.T, c crankCase) (*crankhandle.Model, *decad.Document) {
	t.Helper()
	if b, ok := builtCases[c.name]; ok {
		return b.m, b.doc
	}
	doc := crankhandle.NewDocument()
	m, err := crankhandle.Build(t.Context(), doc, c.params())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 9)
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

func TestCrankHandleContainers(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			require.NoError(t, doc.VerifyContainers().Err())
			want := map[string][]*decad.Body{
				crankhandle.ArmContainerName:    {m.Arm},
				crankhandle.PinContainerName:    {m.Pin},
				crankhandle.HandleContainerName: {m.Handle},
				crankhandle.BoltsContainerName:  {m.Bolt.Body, m.CrossBolts[0].Body, m.CrossBolts[1].Body},
				crankhandle.NutsContainerName:   {m.Nut.Body, m.CrossNuts[0].Body, m.CrossNuts[1].Body},
			}
			children := doc.Root().Children()
			require.Len(t, children, len(want))
			for _, box := range children {
				bodies, ok := want[box.Name()]
				require.True(t, ok, "unexpected container %q", box.Name())
				require.Equal(t, bodies, box.Bodies())
			}
		})
	}
}

// TestCrankHandleDefault checks the presets, the default's bounds and verified meshes.
func TestCrankHandleDefault(t *testing.T) {
	p := crankhandle.DefaultParams()
	require.Equal(t, crankhandle.Params605ZZ(), p)
	require.Equal(t, ballbearing.Params605ZZ().BoreDiameter, p.ShaftDiameter)
	require.Equal(t, ballbearing.Params6082RS().BoreDiameter, crankhandle.Params6082RS().ShaftDiameter)
	m, _ := built(t, crankCases[0])
	require.InDelta(t, 2.55, m.Dimensions.BoreRadius, 1e-12)
	require.InDelta(t, 2.05, m.Dimensions.BoreFlat, 1e-12)
	for name, want := range map[string][2]r3.Vec{
		"arm":    {{X: -10, Y: -10, Z: 0}, {X: 10, Y: 65, Z: 8}},
		"pin":    {{X: -crankhandle.HeadR, Y: 55 - crankhandle.HeadR, Z: crankhandle.StubFrom}, {X: crankhandle.HeadR, Y: 55 + crankhandle.HeadR, Z: 58}},
		"handle": {{X: -9.3, Y: 55 - 9.3, Z: 8.4}, {X: 9.3, Y: 55 + 9.3, Z: 58.4}},
	} {
		body := map[string]*decad.Body{"arm": m.Arm, "pin": m.Pin, "handle": m.Handle}[name]
		box, err := body.Bounds()
		require.NoError(t, err)
		tol := box.Bound.Base() + 1e-3
		require.InDelta(t, want[0].X, box.Min.X, tol, name)
		require.InDelta(t, want[1].Y, box.Max.Y, tol, name)
		require.InDelta(t, want[0].Z, box.Min.Z, tol, name)
		require.InDelta(t, want[1].Z, box.Max.Z, tol, name)
		mesh, err := body.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err, name)
		require.True(t, mesh.BoundaryVerified(), name)
	}
}

// TestCrankHandleBoreFit measures the D bore against the nominal shaft: every point of the bore lies BoreClearance
// outside the shaft's D section, the bore's floor is 1 mm thick, and the wall round the bore is at least 1.6 mm.
func TestCrankHandleBoreFit(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			rs, fs := p.ShaftDiameter/2, p.ShaftDiameter/2-p.FlatDepth
			for _, z := range []float64{1, 3.5, 6.5} {
				o := section(t, m.Arm, atZ(t, z))
				bound := o.OffSurface.Base() + 1e-6
				var bore []r3.Vec
				for _, l := range o.Loops {
					if math.Hypot(l[0].X, l[0].Y) < m.Dimensions.BoreRadius+0.1 {
						bore = l
					}
				}
				require.NotNil(t, bore, "z %g: bore loop", z)
				least, most := math.Inf(1), math.Inf(-1)
				for _, v := range bore {
					// Outside the shaft's convex D section, the distance is at least the larger of the distances outside
					// the round and outside the flat's half plane.
					gap := math.Max(math.Hypot(v.X, v.Y)-rs, v.Y-fs)
					least, most = math.Min(least, gap), math.Max(most, gap)
				}
				require.GreaterOrEqual(t, least, p.BoreClearance-bound, "z %g", z)
				require.LessOrEqual(t, most, p.BoreClearance+bound, "z %g", z)
				wall := math.Inf(1)
				for _, l := range o.Loops {
					if &l[0] == &bore[0] {
						continue
					}
					for _, v := range bore {
						wall = math.Min(wall, distanceToLoop(v, l))
					}
				}
				require.GreaterOrEqual(t, wall, crankhandle.PressWall-bound, "z %g: wall round the bore", z)
			}
			// No bore at the outer face: the floor closes it.
			for _, l := range section(t, m.Arm, atZ(t, crankhandle.ArmThickness-0.25)).Loops {
				require.Greater(t, math.Hypot(l[0].X, l[0].Y), m.Dimensions.BoreRadius+0.5, "bore through the floor")
			}
		})
	}
}

// TestCrankHandleFits checks every fit between the printed parts: the pin's stub in the arm's pocket (Tight), the
// handle on the pin's shank and below its head (Regular), and the handle's gap to the arm (Regular). The handle is a
// solid of revolution about the pin axis, so a cut through that axis shows every gap it has at any angle: it turns
// freely when every gap there clears its class.
func TestCrankHandleFits(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			stub := clearance(t, m.Arm, m.Pin, atZ(t, 6))
			require.GreaterOrEqual(t, stub.Lower().Base(), 0.2, "stub in its pocket")
			require.InDelta(t, crankhandle.TightFit, stub.Distance.Value.Base(), 0.01, "stub in its pocket")
			for _, z := range []float64{12, 30, 50} {
				shank := clearance(t, m.Handle, m.Pin, atZ(t, z))
				require.GreaterOrEqual(t, shank.Lower().Base(), 0.35, "handle on the shank at z %g", z)
			}
			axis, err := r3.NewFrame(r3.Vec{Y: p.Throw}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
			require.NoError(t, err)
			for name, other := range map[string]*decad.Body{"pin": m.Pin, "arm": m.Arm} {
				cl := clearance(t, m.Handle, other, axis)
				require.GreaterOrEqual(t, cl.Lower().Base(), 0.35, "handle to %s through the pin axis", name)
			}
			// The pin's head is wider than the handle's bore and stands above the bore's top, so it holds the handle.
			require.Greater(t, crankhandle.HeadR, crankhandle.HandleProfile[0][1])
			require.Greater(t, crankhandle.HeadFrom, crankhandle.HandleFrom+crankhandle.HandleProfile[1][0])
		})
	}
}

// TestCrankHandleFasteners checks the pin bolt: the head's top face is flush with the arm's back face and inside its
// pocket, the tip and its hole stay inside the pin, the nut lies inside its cavity, and neither shares volume with a part.
func TestCrankHandleFasteners(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			bb, err := m.Bolt.Body.Bounds()
			require.NoError(t, err)
			tol := bb.Bound.Base() + 2e-3
			require.InDelta(t, 0, bb.Min.Z, tol, "head top flush with the back face")
			require.InDelta(t, m.Site.Tip(), bb.Max.Z, tol, "tip")
			require.Less(t, bb.Max.Z+m2fastener.Stock, crankhandle.ShankTo, "tip and its hole inside the pin")
			require.Less(t, math.Max(bb.Max.X, bb.Max.Y-p.Throw), m2fastener.HeadPocketRadius, "head inside its pocket")
			nb, err := m.Nut.Body.Bounds()
			require.NoError(t, err)
			require.GreaterOrEqual(t, nb.Min.Z, crankhandle.PinCavity)
			require.LessOrEqual(t, nb.Max.Z, crankhandle.PinCavity+m2fastener.NutCavityHeight)
			for _, d := range []float64{nb.Min.X, nb.Max.X, nb.Min.Y - p.Throw, nb.Max.Y - p.Throw} {
				require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius)
			}
			for _, z := range []float64{1, 3, 5.5, 7, 9, 11.5} {
				for _, host := range []*decad.Body{m.Arm, m.Pin} {
					for _, part := range []*decad.Body{m.Bolt.Body, m.Nut.Body} {
						a, b := section(t, host, atZ(t, z)), section(t, part, atZ(t, z))
						if len(a.Loops) == 0 || len(b.Loops) == 0 {
							continue
						}
						cl, err := a.Clearance(b)
						require.NoError(t, err)
						require.False(t, cl.Overlaps, "fastener overlaps a part at z %g", z)
					}
				}
			}
			// 1 mm or more of stub round the cavity and of arm under the head pocket.
			require.GreaterOrEqual(t, crankhandle.StubFlats/2-m2fastener.NutCavityRadius, 1.0)
			require.GreaterOrEqual(t, crankhandle.PocketFrom-m2fastener.HeadPocketDepth, 1.0)
			requireCrossBolts(t, p, m)
		})
	}
}

// atY is the plane y = const of the local frame, normal +Y: a layer of the pin's halves as they print.
func atY(t *testing.T, y float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Y: y}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	require.NoError(t, err)
	return f
}

// requireCrossBolts checks the two cross bolts that clamp the pin's halves: each head's top face is flush with the +Y
// flat and inside its pocket in the +Y half, each tip ends at the -Y flat, each nut lies inside its cavity in the -Y
// half, so the bolt crosses the seam, and no bolt or nut shares volume with the pin. It also checks 1 mm or more of
// pin round every pocket and cavity.
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
		require.Less(t, math.Max(math.Max(-bb.Min.X, bb.Max.X), math.Max(z-bb.Min.Z, bb.Max.Z-z)),
			m2fastener.HeadPocketRadius, "cross bolt %d: head inside its pocket", i+1)
		require.Less(t, bb.Min.Y, seam, "cross bolt %d crosses the seam", i+1)
		nb, err := m.CrossNuts[i].Body.Bounds()
		require.NoError(t, err)
		near := seam + crankhandle.CrossFlat - crankhandle.CrossCavity
		require.LessOrEqual(t, nb.Max.Y, near, "cross nut %d inside its cavity in the -Y half", i+1)
		require.GreaterOrEqual(t, nb.Min.Y, near-m2fastener.NutCavityHeight, "cross nut %d", i+1)
		for _, d := range []float64{nb.Min.X, nb.Max.X, nb.Min.Z - z, nb.Max.Z - z} {
			require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius, "cross nut %d", i+1)
		}
		for _, y := range []float64{3.5, 2.5, 1.0, -0.5, -1.5, -2.0, -2.5, -3.5} {
			a := section(t, m.Pin, atY(t, seam+y))
			for _, part := range []*decad.Body{m.CrossBolts[i].Body, m.CrossNuts[i].Body} {
				b := section(t, part, atY(t, seam+y))
				if len(a.Loops) == 0 || len(b.Loops) == 0 {
					continue
				}
				cl, err := a.Clearance(b)
				require.NoError(t, err)
				require.False(t, cl.Overlaps, "cross fastener %d overlaps the pin at y %g", i+1, y)
			}
		}
	}
	r, flat := crankhandle.ShankR, crankhandle.CrossFlat
	far := crankhandle.CrossCavity + m2fastener.NutCavityHeight - flat // the cavity's far face from the seam
	for name, wall := range map[string]float64{
		"round the cavity at its far face":   math.Sqrt(r*r-far*far) - m2fastener.NutCavityRadius,
		"round the head pocket at the flat":  math.Sqrt(r*r-flat*flat) - m2fastener.HeadPocketRadius,
		"of flat beyond the head pocket":     crankhandle.CrossFlatHalf - m2fastener.HeadPocketRadius,
		"under the head pocket to the seam":  flat - m2fastener.HeadPocketDepth,
		"between the cavity and the seam":    crankhandle.CrossCavity - flat,
		"between the cavity and the -Y flat": flat - far,
	} {
		require.GreaterOrEqual(t, wall, 1.0-1e-9, "material %s", name)
	}
	require.LessOrEqual(t, m.CrossSites[0].Tip(), 2*flat, "tip inside the -Y flat")
}

// TestCrankHandleSplitPin checks the pin against "Split shafts": split by the plane through its axis and local X,
// it is mirror-symmetric apart from the cross bolts' pockets and cavities, each half prints on its cut face with only
// bridgeable roofs, and each half stands HeadR high.
func TestCrankHandleSplitPin(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			require.InDelta(t, 1, m.SplitNormal.Y, 1e-12)
			var exempt []splitcheck.Line
			for _, site := range m.CrossSites {
				exempt = append(exempt, splitcheck.Line{Point: site.Head, Dir: site.Dir, Radius: m2fastener.NutCavityRadius + 0.1})
			}
			r, err := splitcheck.Check(t.Context(), splitcheck.Shaft{
				Body: m.Pin, Point: r3.Vec{Y: p.Throw}, Axis: r3.Vec{Z: 1}, Normal: m.SplitNormal, Exempt: exempt,
			}, 0.01)
			require.NoError(t, err)
			require.Empty(t, r.Violations)
			require.Positive(t, r.Bridges)
			require.LessOrEqual(t, r.Bridge, splitcheck.MaxBridge)
			for i, h := range r.HalfHeights {
				require.InDelta(t, crankhandle.HeadR, h, 0.01, "half %d", i+1)
			}
		})
	}
}

// TestCrankHandleEdgeFinishing checks the 0.5 mm chamfer on both faces' outlines: a section 0.25 mm inside a face is
// 0.25 mm short of the arm's ends.
func TestCrankHandleEdgeFinishing(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			for _, z := range []float64{0.25, crankhandle.ArmThickness - 0.25} {
				o := section(t, m.Arm, atZ(t, z))
				bound := o.OffSurface.Base() + 1e-3
				lo, hi := math.Inf(1), math.Inf(-1)
				for _, l := range o.Loops {
					for _, v := range l {
						lo, hi = math.Min(lo, v.Y), math.Max(hi, v.Y)
					}
				}
				require.InDelta(t, -crankhandle.ArmRadius+0.25, lo, bound, "z %g", z)
				require.InDelta(t, p.Throw+crankhandle.ArmRadius-0.25, hi, bound, "z %g", z)
			}
		})
	}
}

// TestCrankHandleCorners applies "Corners and roots" to every printed part in its print orientation, axis +Z: the arm
// flat, the pin upright and the handle upright. The arm's hexagonal pocket keeps sharp corners to match the pin's
// sharp stub, a listed joinery pair.
func TestCrankHandleCorners(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			pocket := cornercheck.Exempt{
				Reason: "hexagonal pocket corners",
				Near: cornercheck.Annulus(r3.Vec{Y: p.Throw}, r3.Vec{Z: 1},
					crankhandle.PocketFlats/math.Sqrt(3)-0.3, crankhandle.PocketFlats/math.Sqrt(3)+0.3),
			}
			for name, item := range map[string]struct {
				body   *decad.Body
				up     r3.Vec
				exempt []cornercheck.Exempt
			}{
				"arm":    {m.Arm, r3.Vec{Z: 1}, []cornercheck.Exempt{pocket}},
				"pin":    {m.Pin, m.SplitNormal, []cornercheck.Exempt{nutRounds(p)}},
				"handle": {m.Handle, r3.Vec{Z: 1}, nil},
			} {
				fs, err := cornercheck.Check(t.Context(), item.body, item.up, item.exempt)
				require.NoError(t, err, name)
				require.Empty(t, fs, "%s: %s", name, cornercheck.Report(fs, 8))
			}
		})
	}
}

// nutRounds lists the pin bolt's nut cavity rounds (m2fastener.RoundRadius, 0.55 mm), the largest that keep the nut's
// corner clearance. A layer 2 mm off the seam crosses them near their edge and reads them at about 52 degrees
// (instructions.md, "Retained sharp corners").
func nutRounds(p crankhandle.Params) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "pin bolt nut cavity rounds",
		Near: func(v r3.Vec) bool {
			return math.Hypot(v.X, v.Y-p.Throw) <= m2fastener.NutCavityRadius+0.05 &&
				v.Z >= crankhandle.PinCavity-0.05 && v.Z <= crankhandle.PinCavity+m2fastener.NutCavityHeight+0.05
		},
	}
}

func TestCrankHandlePlacement(t *testing.T) {
	frame, err := r3.NewFrame(r3.Vec{X: 10, Y: -5, Z: 7}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	local, _ := built(t, crankCases[0])
	p := crankhandle.DefaultParams()
	p.Placement = &frame
	placed, err := crankhandle.Build(t.Context(), crankhandle.NewDocument(), p)
	require.NoError(t, err)
	xf, err := r3.FromFrame(frame)
	require.NoError(t, err)
	for name, pair := range map[string][2]*decad.Body{
		"arm": {local.Arm, placed.Arm}, "pin": {local.Pin, placed.Pin}, "handle": {local.Handle, placed.Handle},
		"bolt": {local.Bolt.Body, placed.Bolt.Body}, "nut": {local.Nut.Body, placed.Nut.Body},
		"cross bolt": {local.CrossBolts[1].Body, placed.CrossBolts[1].Body},
		"cross nut":  {local.CrossNuts[1].Body, placed.CrossNuts[1].Body},
	} {
		lv, err := pair[0].Volume()
		require.NoError(t, err)
		pv, err := pair[1].Volume()
		require.NoError(t, err)
		require.InDelta(t, lv.Value.Base(), pv.Value.Base(), 1e-6*lv.Value.Base(), name)
		lb, err := pair[0].Bounds()
		require.NoError(t, err)
		pb, err := pair[1].Bounds()
		require.NoError(t, err)
		// The frame maps local X to world Y, Y to Z and Z to X.
		tol := lb.Bound.Base() + pb.Bound.Base() + 1e-6
		lo, hi := xf.Apply(lb.Min), xf.Apply(lb.Max)
		require.InDelta(t, math.Min(lo.X, hi.X), pb.Min.X, tol, name)
		require.InDelta(t, math.Max(lo.Y, hi.Y), pb.Max.Y, tol, name)
		require.InDelta(t, math.Min(lo.Z, hi.Z), pb.Min.Z, tol, name)
	}
	require.InDelta(t, 0, placed.Site.Head.Sub(xf.Apply(local.Site.Head)).Len(), 1e-9)
	require.InDelta(t, 0, placed.CrossSites[0].Head.Sub(xf.Apply(local.CrossSites[0].Head)).Len(), 1e-9)
	require.InDelta(t, 0, placed.SplitNormal.Sub(xf.ApplyDir(local.SplitNormal)).Len(), 1e-12)
}

func TestCrankHandleRejectsInvalidInputs(t *testing.T) {
	def := crankhandle.DefaultParams()
	for _, item := range []struct {
		name string
		edit func(p *crankhandle.Params)
	}{
		{"nan", func(p *crankhandle.Params) { p.Throw = math.NaN() }},
		{"inf", func(p *crankhandle.Params) { p.ShaftDiameter = math.Inf(1) }},
		{"zero-shaft", func(p *crankhandle.Params) { p.ShaftDiameter = 0 }},
		{"zero-flat", func(p *crankhandle.Params) { p.FlatDepth = 0 }},
		{"flat-too-deep", func(p *crankhandle.Params) { p.FlatDepth = 1.3 }},
		{"press-clearance", func(p *crankhandle.Params) { p.BoreClearance = 0.04 }},
		{"loose-clearance", func(p *crankhandle.Params) { p.BoreClearance = 0.51 }},
		{"shaft-too-wide", func(p *crankhandle.Params) { p.ShaftDiameter = 16.8 }},
		{"short-throw", func(p *crankhandle.Params) { p.Throw = 19.9 }},
		{"invalid-frame", func(p *crankhandle.Params) { p.Placement = &r3.Frame{} }},
	} {
		t.Run(item.name, func(t *testing.T) {
			p := def
			item.edit(&p)
			require.Error(t, p.Validate())
			doc := decad.New()
			_, err := crankhandle.Build(t.Context(), doc, p)
			require.ErrorContains(t, err, "crankhandle: ")
			require.Empty(t, doc.Bodies())
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	doc := decad.New()
	_, err := crankhandle.Build(ctx, doc, def)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, doc.Bodies())
	_, err = crankhandle.Build(t.Context(), nil, def)
	require.ErrorContains(t, err, "document")
	_, err = crankhandle.Build(nil, doc, def) //nolint:staticcheck
	require.ErrorContains(t, err, "context")
}
