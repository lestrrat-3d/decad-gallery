package crankhandle_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// build builds one case in a fresh document and checks what every case shares: three bodies, each in its container,
// and both sketches fully constrained.
func build(t *testing.T, p crankhandle.Params) (*crankhandle.Model, *decad.Document) {
	t.Helper()
	doc := decad.New()
	m, err := crankhandle.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.True(t, m.Crank.IsSolid())
	require.Len(t, doc.Bodies(), 3)
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.Plate))
	require.NoError(t, sketchcheck.VerifyFullyConstrained(t.Context(), m.Screw))
	return m, doc
}

// section cuts body by the plane z = const at the finest chord the body allows.
func section(t *testing.T, body *decad.Body, z float64) decad.SectionOutline {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	require.NoError(t, err)
	var o decad.SectionOutline
	for _, tol := range []float64{0.005, 0.02, 0.05} {
		if o, err = body.SectionOutline(t.Context(), f, units.Millimeters(tol)); err == nil {
			return o
		}
	}
	require.NoError(t, err)
	return o
}

// boreAndOuter returns the section's hole loop round the axis and its outer loop.
func boreAndOuter(t *testing.T, o decad.SectionOutline) ([]r3.Vec, []r3.Vec) {
	t.Helper()
	var bore, outer []r3.Vec
	for _, l := range o.Loops {
		near := true
		for _, v := range l {
			if math.Hypot(v.X, v.Y) > 6 {
				near = false
				break
			}
		}
		if near {
			bore = l
		} else {
			outer = l
		}
	}
	require.NotNil(t, bore, "bore loop")
	require.NotNil(t, outer, "outer loop")
	return bore, outer
}

func TestCrankHandleContainers(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			require.NoError(t, doc.VerifyContainers().Err())
			want := map[string]*decad.Body{
				crankhandle.CrankContainerName: m.Crank,
				crankhandle.BoltsContainerName: m.Bolt.Body,
				crankhandle.NutsContainerName:  m.Nut.Body,
			}
			children := doc.Root().Children()
			require.Len(t, children, len(want))
			for _, box := range children {
				body, ok := want[box.Name()]
				require.True(t, ok, "unexpected container %q", box.Name())
				require.Equal(t, []*decad.Body{body}, box.Bodies())
			}
		})
	}
}

// TestCrankHandleDefault checks the default crank's derived values, bounds and verified mesh.
func TestCrankHandleDefault(t *testing.T) {
	p := crankhandle.DefaultParams()
	require.Equal(t, crankhandle.Params605ZZ(), p)
	require.Equal(t, ballbearing.Params605ZZ().BoreDiameter, p.ShaftDiameter)
	require.Equal(t, ballbearing.Params6082RS().BoreDiameter, crankhandle.Params6082RS().ShaftDiameter)
	m, _ := built(t, crankCases[0])
	dm := m.Dimensions
	require.InDelta(t, 2.55, dm.BoreRadius, 1e-12)
	require.InDelta(t, 2.05, dm.BoreFlat, 1e-12)
	require.InDelta(t, 10.05, dm.TailDistance, 1e-12)
	require.InDelta(t, 5.0, dm.RootRadius, 1e-12)
	require.InDelta(t, 13.5, dm.EndRadius, 1e-12)

	box, err := m.Crank.Bounds()
	require.NoError(t, err)
	tol := box.Bound.Base() + 1e-6
	require.InDelta(t, -dm.EndRadius, box.Min.X, tol)
	require.InDelta(t, dm.EndRadius, box.Max.X, tol)
	require.InDelta(t, -dm.TailDistance, box.Min.Y, tol)
	require.InDelta(t, p.Throw+dm.EndRadius, box.Max.Y, tol)
	require.InDelta(t, 0, box.Min.Z, tol)
	require.InDelta(t, p.Thickness+p.GripLength, box.Max.Z, tol)

	mesh, err := m.Crank.Tessellate(t.Context(), units.Millimeters(.05), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
}

// TestCrankHandleBoreFit measures the D bore on the built crank against the nominal shaft: every point of the bore
// lies BoreClearance outside the shaft's D section, and the wall round the bore is at least 1.6 mm. The sections
// stand clear of the face chamfers and of the set screw's hole.
func TestCrankHandleBoreFit(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			rs, fs := p.ShaftDiameter/2, p.ShaftDiameter/2-p.FlatDepth
			for _, z := range []float64{1, p.Thickness/2 - 2, p.Thickness/2 + 2, p.Thickness - 1} {
				o := section(t, m.Crank, z)
				bound := o.OffSurface.Base() + 1e-6
				bore, outer := boreAndOuter(t, o)
				least, most := math.Inf(1), math.Inf(-1)
				for _, v := range bore {
					// The distance from a point outside the shaft's convex D section is at least the larger of its
					// distances outside the round and outside the flat's half plane.
					gap := math.Max(math.Hypot(v.X, v.Y)-rs, -fs-v.Y)
					least, most = math.Min(least, gap), math.Max(most, gap)
				}
				require.GreaterOrEqual(t, least, p.BoreClearance-bound, "z %g: bore clearance", z)
				require.LessOrEqual(t, most, p.BoreClearance+bound, "z %g: bore clearance", z)
				_ = outer
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
		})
	}
}

// TestCrankHandleFasteners checks the set screw: the head's top face is flush with the tail face and inside its
// pocket, the tip ends on the bore's flat, the nut lies inside its cavity, the cavity is closed round the nut, and
// neither part shares volume with the crank.
func TestCrankHandleFasteners(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			dm := m.Dimensions
			zb := dm.ScrewHeight
			bb, err := m.Bolt.Body.Bounds()
			require.NoError(t, err)
			tol := bb.Bound.Base() + 2e-3
			require.InDelta(t, -dm.TailDistance, bb.Min.Y, tol, "head top flush with the tail face")
			require.InDelta(t, -dm.BoreFlat, bb.Max.Y, tol, "tip on the bore's flat")
			require.Less(t, math.Max(bb.Max.X, bb.Max.Z-zb), m2fastener.HeadPocketRadius, "head inside its pocket")

			nb, err := m.Nut.Body.Bounds()
			require.NoError(t, err)
			near := -dm.TailDistance + crankhandle.CavityDepth
			require.GreaterOrEqual(t, nb.Min.Y, near)
			require.LessOrEqual(t, nb.Max.Y, near+m2fastener.NutCavityHeight)
			for _, d := range []float64{nb.Min.X, nb.Max.X, nb.Min.Z - zb, nb.Max.Z - zb} {
				require.LessOrEqual(t, math.Abs(d), m2fastener.NutCavityRadius)
			}

			for _, z := range []float64{zb - 2, zb - 1, zb - 0.4, zb, zb + 0.4, zb + 1, zb + 2} {
				crank := section(t, m.Crank, z)
				for name, body := range map[string]*decad.Body{"bolt": m.Bolt.Body, "nut": m.Nut.Body} {
					part := section(t, body, z)
					if len(part.Loops) == 0 {
						continue
					}
					cl, err := crank.Clearance(part)
					require.NoError(t, err)
					require.False(t, cl.Overlaps, "%s overlaps the crank at z %g", name, z)
				}
			}
			// The crank closes the cavity above and below the nut, and on its side toward the tail face and the bore.
			centre := near + m2fastener.NutCavityHeight/2
			for _, pt := range []r3.Vec{
				{X: 0, Y: centre, Z: zb + m2fastener.NutCavityRadius + 0.5},
				{X: 0, Y: centre, Z: zb - m2fastener.NutCavityRadius - 0.5},
				{X: m2fastener.NutCavityRadius + 0.5, Y: centre, Z: zb},
				{X: 2, Y: near - 0.5, Z: zb},
				{X: 2, Y: near + m2fastener.NutCavityHeight + 0.5, Z: zb},
			} {
				require.True(t, inside(t, m.Crank, pt), "material at %v", pt)
			}
			// 1 mm or more of material round the pocket and the cavity.
			require.GreaterOrEqual(t, dm.TailHalf-m2fastener.HeadPocketRadius, 1.0)
			require.GreaterOrEqual(t, zb-m2fastener.NutCavityRadius, 1.0)
			require.GreaterOrEqual(t, crankhandle.CavityDepth-m2fastener.HeadPocketDepth, 1.0)
			require.GreaterOrEqual(t, dm.Site.Tip()-dm.Site.CavityFar(), 1.0)
		})
	}
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

// inside reports whether pt lies in body's section at pt.Z.
func inside(t *testing.T, body *decad.Body, pt r3.Vec) bool {
	t.Helper()
	n := 0
	for _, l := range section(t, body, pt.Z).Loops {
		crossings := false
		for i := range l {
			a, b := l[i], l[(i+1)%len(l)]
			if (a.Y > pt.Y) != (b.Y > pt.Y) && pt.X < a.X+(pt.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y) {
				crossings = !crossings
			}
		}
		if crossings {
			n++
		}
	}
	return n%2 == 1
}

// TestCrankHandleEdgeFinishing checks the 0.5 mm chamfer on both faces' outlines: a section 0.25 mm inside a face
// is 0.25 mm narrower than the plate. The bore's rim is chamfered only where its loop is 20 mm or longer.
func TestCrankHandleEdgeFinishing(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			dm := m.Dimensions
			boreLoop := dm.BoreLoop
			for _, z := range []float64{0.25, p.Thickness - 0.25} {
				o := section(t, m.Crank, z)
				bound := o.OffSurface.Base() + 1e-3
				bore, outer := boreAndOuter(t, o)
				maxY, minY := math.Inf(-1), math.Inf(1)
				for _, v := range outer {
					maxY, minY = math.Max(maxY, v.Y), math.Min(minY, v.Y)
				}
				require.InDelta(t, p.Throw+dm.EndRadius-0.25, maxY, bound, "z %g: end", z)
				require.InDelta(t, -dm.TailDistance+0.25, minY, bound, "z %g: tail", z)
				far := 0.0
				for _, v := range bore {
					far = math.Max(far, math.Hypot(v.X, v.Y))
				}
				want := dm.BoreRadius
				if boreLoop >= 20 {
					want += 0.25
				}
				require.InDelta(t, want, far, bound, "z %g: bore rim (loop %.1f mm)", z, boreLoop)
			}
		})
	}
}

// TestCrankHandleGrip checks the grip's round and its root fillet, whose radius is half the thinner of the arm and
// the grip.
func TestCrankHandleGrip(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			dm := m.Dimensions
			require.InDelta(t, math.Min(p.Thickness, p.GripDiameter)/2, dm.RootRadius, 1e-12)
			radiusAt := func(z float64) float64 {
				o := section(t, m.Crank, z)
				require.Len(t, o.Loops, 1, "z %g", z)
				far := 0.0
				for _, v := range o.Loops[0] {
					far = math.Max(far, math.Hypot(v.X, v.Y-p.Throw))
				}
				return far
			}
			tol := 0.01
			rho := dm.RootRadius
			for _, h := range []float64{0.25, 0.5, 0.75} {
				want := p.GripDiameter/2 + rho - math.Sqrt(rho*rho-(rho-h*rho)*(rho-h*rho))
				require.InDelta(t, want, radiusAt(p.Thickness+h*rho), tol, "root fillet at %g of its height", h)
			}
			require.InDelta(t, p.GripDiameter/2, radiusAt(p.Thickness+(rho+p.GripLength-dm.TipRound)/2), tol, "grip")
		})
	}
}

func TestCrankHandlePlacement(t *testing.T) {
	frame, err := r3.NewFrame(r3.Vec{X: 10, Y: -5, Z: 7}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	p := crankhandle.DefaultParams()
	local, _ := built(t, crankCases[0])
	p.Placement = &frame
	placed, _ := build(t, p)
	xf, err := r3.FromFrame(frame)
	require.NoError(t, err)
	for name, pair := range map[string][2]*decad.Body{
		"crank": {local.Crank, placed.Crank}, "bolt": {local.Bolt.Body, placed.Bolt.Body},
		"nut": {local.Nut.Body, placed.Nut.Body},
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
}

func TestCrankHandleRejectsInvalidInputs(t *testing.T) {
	def := crankhandle.DefaultParams()
	for _, item := range []struct {
		name string
		edit func(p *crankhandle.Params)
	}{
		{"nan", func(p *crankhandle.Params) { p.Throw = math.NaN() }},
		{"inf", func(p *crankhandle.Params) { p.GripLength = math.Inf(1) }},
		{"zero-shaft", func(p *crankhandle.Params) { p.ShaftDiameter = 0 }},
		{"zero-flat", func(p *crankhandle.Params) { p.FlatDepth = 0 }},
		{"flat-too-deep", func(p *crankhandle.Params) { p.FlatDepth = 1.3 }},
		{"press-clearance", func(p *crankhandle.Params) { p.BoreClearance = 0.04 }},
		{"loose-clearance", func(p *crankhandle.Params) { p.BoreClearance = 0.51 }},
		{"narrow-flat", func(p *crankhandle.Params) { p.FlatDepth = 0.2 }},
		{"small-hub", func(p *crankhandle.Params) { p.HubDiameter = 21 }},
		{"thin", func(p *crankhandle.Params) { p.Thickness = 7 }},
		{"short-grip", func(p *crankhandle.Params) { p.GripLength = 7 }},
		{"short-throw", func(p *crankhandle.Params) { p.Throw = 25 }},
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
