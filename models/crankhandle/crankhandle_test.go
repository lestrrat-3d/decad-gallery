package crankhandle_test

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// crankCase is one parameter case. Every case changes the arm's outline or its bore. slitRound marks the case whose
// held arm mesh chords the slit's end round coarsely enough for cornercheck to read it (instructions.md, "Retained
// sharp corners and edges").
type crankCase struct {
	name      string
	params    func() crankhandle.Params
	slitRound bool
}

var crankCases = []crankCase{
	{"605ZZ", crankhandle.Params605ZZ, false},
	{"608-2RS", crankhandle.Params6082RS, false},
	{"regular-bore", func() crankhandle.Params {
		p := crankhandle.Params605ZZ()
		p.BoreClearance = 0.35
		return p
	}, false},
	{"long-throw", func() crankhandle.Params {
		p := crankhandle.Params6082RS()
		p.Throw = 80
		return p
	}, true},
}

type builtCase struct {
	m   *crankhandle.Model
	doc *decad.Document
}

// builtCases holds each case's build, shared by the tests: Build takes seconds and the tests only read its bodies.
var builtCases = map[string]builtCase{}

// built builds a case on first use and checks what every case shares: eleven bodies (the arm, the pin, the handle, four
// bolts and four nuts), each in its container, and every sketch fully constrained.
func built(t *testing.T, c crankCase) (*crankhandle.Model, *decad.Document) {
	t.Helper()
	if b, ok := builtCases[c.name]; ok {
		return b.m, b.doc
	}
	doc := crankhandle.NewDocument()
	m, err := crankhandle.Build(t.Context(), doc, c.params())
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 11)
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
				crankhandle.BoltsContainerName:  {m.Bolt.Body, m.CrossBolts[0].Body, m.CrossBolts[1].Body, m.ClampBolt.Body},
				crankhandle.NutsContainerName:   {m.Nut.Body, m.CrossNuts[0].Body, m.CrossNuts[1].Body, m.ClampNut.Body},
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
	// The end arc reaches 13.9 mm behind the axis for the 5 mm shaft and 15.4 mm for the 8 mm one.
	require.InDelta(t, 13.9, m.Dimensions.Reach, 0.02)
	large, err := crankhandle.Params6082RS().Derive()
	require.NoError(t, err)
	require.InDelta(t, 15.4, large.Reach, 0.02)
	armBox, err := m.Arm.Bounds()
	require.NoError(t, err)
	require.InDelta(t, -m.Dimensions.Reach, armBox.Min.Y, armBox.Bound.Base()+1e-3, "the end arc's reach")
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
// outside the shaft's D section, the bore's floor is 1 mm thick apart from the slit's round end, and the wall round the
// bore is at least 1.6 mm, except the ClampHoleWall web over the clamp bolt's hole at mid thickness. The slit joins the
// bore to the outline, so the bore is read as the points within r_b of the axis outside the slit.
func TestCrankHandleBoreFit(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			rs, fs := p.ShaftDiameter/2, p.ShaftDiameter/2-p.FlatDepth
			half := crankhandle.SlitWidth / 2
			for z, least := range map[float64]float64{1: crankhandle.PressWall, 4: crankhandle.ClampHoleWall, 6.5: crankhandle.PressWall} {
				o := section(t, m.Arm, atZ(t, z))
				bound := o.OffSurface.Base() + 1e-6
				inSlit := func(v r3.Vec) bool { return math.Abs(v.X) <= half+bound && v.Y < 0 }
				onBore := func(v r3.Vec) bool { return math.Hypot(v.X, v.Y) < m.Dimensions.BoreRadius+0.1 && !inSlit(v) }
				var bore []r3.Vec
				for _, l := range o.Loops {
					for _, v := range l {
						if onBore(v) {
							bore = append(bore, v)
						}
					}
				}
				require.NotEmpty(t, bore, "z %g: bore", z)
				lo, hi := math.Inf(1), math.Inf(-1)
				for _, v := range bore {
					// Outside the shaft's convex D section, the distance is at least the larger of the distances outside
					// the round and outside the flat's half plane.
					gap := math.Max(math.Hypot(v.X, v.Y)-rs, v.Y-fs)
					lo, hi = math.Min(lo, gap), math.Max(hi, gap)
				}
				require.GreaterOrEqual(t, lo, p.BoreClearance-bound, "z %g", z)
				require.LessOrEqual(t, hi, p.BoreClearance+bound, "z %g", z)
				// The wall: from the bore to every edge of the section that is neither the bore nor the slit.
				wall := math.Inf(1)
				for _, l := range o.Loops {
					for i := range l {
						a, b := l[i], l[(i+1)%len(l)]
						if onBore(a) || onBore(b) || inSlit(a) || inSlit(b) {
							continue
						}
						for _, v := range bore {
							wall = math.Min(wall, distanceToLoop(v, []r3.Vec{a, b}))
						}
					}
				}
				require.GreaterOrEqual(t, wall, least-bound, "z %g: wall round the bore", z)
			}
			// At the outer face the floor closes the bore everywhere but the slit's round end.
			o := section(t, m.Arm, atZ(t, crankhandle.ArmThickness-0.25))
			bound := o.OffSurface.Base() + 1e-3
			for _, l := range o.Loops {
				for _, v := range l {
					if math.Abs(v.X) <= half+bound {
						continue
					}
					require.Greater(t, math.Hypot(v.X, v.Y), m.Dimensions.BoreRadius+0.5, "bore through the floor at %v", v)
				}
			}
		})
	}
}

// TestCrankHandleSlit checks that the slit closes the bore: low in the arm one loop of its section runs from the bore
// down the slit's two walls, SlitWidth apart, to the end arc, so tightening the clamp bolt narrows the bore; and in the
// floor the slit ends in its full round SlitInBore past the bore's wall.
func TestCrankHandleSlit(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			dm := m.Dimensions
			half := crankhandle.SlitWidth / 2
			o := section(t, m.Arm, atZ(t, 1))
			bound := o.OffSurface.Base() + 1e-3
			joined := false
			for _, l := range o.Loops {
				bore, end := false, false
				for _, v := range l {
					bore = bore || math.Hypot(v.X, v.Y) < dm.BoreRadius+0.1
					end = end || v.Y < -dm.Reach+1
				}
				joined = joined || bore && end
			}
			require.True(t, joined, "one loop holds the bore and the end arc")
			gap := math.Inf(1)
			for _, l := range o.Loops {
				for _, v := range l {
					if v.Y < -dm.Reach+1.5 || v.Y > -dm.BoreRadius-0.3 {
						continue
					}
					gap = math.Min(gap, math.Abs(v.X))
				}
			}
			require.InDelta(t, half, gap, bound, "the slit's walls")
			floor := section(t, m.Arm, atZ(t, crankhandle.ArmThickness-0.5))
			top := math.Inf(-1)
			for _, l := range floor.Loops {
				for _, v := range l {
					if math.Abs(v.X) <= half+bound && v.Y < 0 {
						top = math.Max(top, v.Y)
					}
				}
			}
			require.InDelta(t, -dm.BoreRadius+crankhandle.SlitInBore, top, floor.OffSurface.Base()+1e-3, "the slit's round end")
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
			requireClampBolt(t, m)
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

// TestCrankHandleSplitPin checks the pin against "Split shafts": split by the plane through its axis and local X,
// it is mirror-symmetric apart from the cross bolts' pockets, each half prints on its cut face with only
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
// 0.25 mm short of the arm's ends and of its sides, round the pin's end and the clamp's end arc alike.
func TestCrankHandleEdgeFinishing(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			for _, z := range []float64{0.25, crankhandle.ArmThickness - 0.25} {
				o := section(t, m.Arm, atZ(t, z))
				bound := o.OffSurface.Base() + 1e-3
				lo, hi := math.Inf(1), math.Inf(-1)
				left, right := math.Inf(1), math.Inf(-1)
				for _, l := range o.Loops {
					for _, v := range l {
						lo, hi = math.Min(lo, v.Y), math.Max(hi, v.Y)
						left, right = math.Min(left, v.X), math.Max(right, v.X)
					}
				}
				require.InDelta(t, -m.Dimensions.Reach+0.25, lo, bound, "z %g: end arc", z)
				require.InDelta(t, p.Throw+crankhandle.ArmRadius-0.25, hi, bound, "z %g: pin end", z)
				require.InDelta(t, -crankhandle.ArmRadius+0.25, left, bound, "z %g: -X side", z)
				require.InDelta(t, crankhandle.ArmRadius-0.25, right, bound, "z %g: +X side", z)
			}
		})
	}
}

// TestCrankHandleBoreRim checks the 0.5 mm chamfer on the bore's rim in the back face, which the slit opens into two
// arcs and the flat. A section z above the face runs along the arc of radius r_b + d - z and along the flat at
// f_b + d - z, where d is the chamfer size, and the bevel meets each slit wall where that arc does.
func TestCrankHandleBoreRim(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			rb, fb := m.Dimensions.BoreRadius, m.Dimensions.BoreFlat
			half := crankhandle.SlitWidth / 2
			for _, z := range []float64{0.05, 0.25, 0.45} {
				d := crankhandle.ChamferSize - z
				o := section(t, m.Arm, atZ(t, z))
				tol := o.OffSurface.Base() + 1e-3
				var rim, walls []r3.Vec
				for _, l := range o.Loops {
					for _, v := range l {
						if math.Hypot(v.X, v.Y) > rb+crankhandle.ChamferSize+0.2 {
							continue
						}
						if v.Y < 0 && math.Abs(math.Abs(v.X)-half) <= tol {
							walls = append(walls, v)
							continue
						}
						if v.Y < 0 && math.Abs(v.X) < half {
							continue
						}
						rim = append(rim, v)
					}
				}
				require.NotEmpty(t, rim, "z %g: rim", z)
				arcs, flats := 0, 0
				for _, v := range rim {
					r := math.Hypot(v.X, v.Y)
					onArc := math.Abs(r-(rb+d)) <= tol && v.Y <= fb+d+tol
					onFlat := math.Abs(v.Y-(fb+d)) <= tol && r <= rb+d+tol
					require.True(t, onArc || onFlat, "z %g: rim point %v is off the arc (r %g) and the flat (y %g)", z, v, rb+d, fb+d)
					if onArc {
						arcs++
					}
					if onFlat {
						flats++
					}
				}
				require.Positive(t, arcs, "z %g: arc points", z)
				require.Positive(t, flats, "z %g: flat points", z)
				want := -math.Sqrt((rb+d)*(rb+d) - half*half)
				meets := 0
				for _, v := range walls {
					if math.Abs(v.Y-want) <= tol {
						meets++
					}
				}
				require.GreaterOrEqual(t, meets, 2, "z %g: the bevel meets both slit walls at y %g, walls %v", z, want, walls)
			}
		})
	}
}

// TestCrankHandleCorners applies "Corners and roots" to every printed part in its print orientation, axis +Z: the arm
// flat, the pin upright and the handle upright. The bore's rim in the back face is chamfered, so it holds no corner.
// The arm's hexagonal pocket keeps sharp corners to match the pin's
// sharp stub, a listed joinery pair, and the clamp bolt's head pocket keeps its sharp floor rim, which the 0.3 mm head
// clearance leaves no room to round. The pin's cross nut pockets keep the six sharp corners that mate with their nuts.
// All are listed in instructions.md. The six corners of the clamp nut's hexagonal pocket are listed there too, but they
// run parallel to the print bed, so no layer section reads them and an exemption for them would claim no finding,
// which cornercheck rejects. The corner's pointed roof does make a layer read the floor's round at a slant, which the
// arm's list exempts. The six corners of each cross nut's pocket run along the bolt, normal to the split face.
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
				"arm":    {m.Arm, r3.Vec{Z: 1}, armExempt(c, m, pocket)},
				"pin":    {m.Pin, m.SplitNormal, crossPocketCorners(p, m)},
				"handle": {m.Handle, r3.Vec{Z: 1}, nil},
			} {
				fs, err := cornercheck.Check(t.Context(), item.body, item.up, item.exempt)
				require.NoError(t, err, name)
				require.Empty(t, fs, "%s: %s", name, cornercheck.Report(fs, 8))
			}
		})
	}
}

// armExempt lists the arm's retained corners for case c: the hexagonal pocket, the clamp bolt's head pocket floor rim,
// and in a case marked slitRound the slit's end round in the bore's floor.
func armExempt(c crankCase, m *crankhandle.Model, pocket cornercheck.Exempt) []cornercheck.Exempt {
	dm := m.Dimensions
	floor := crankhandle.ArmRadius - dm.ClampSite.Cavity
	out := []cornercheck.Exempt{pocket, {
		// The pocket's pointed roof and floor slopes lie 30 degrees off the bed, so a layer cuts the 0.6 mm round of the
		// floor's outline at a slant and reads it as a turn over 45 degrees (instructions.md, "Retained sharp corners").
		Reason: "clamp nut pocket floor round cut at a slant",
		Near: func(v r3.Vec) bool {
			return v.X < floor-0.05 && v.X > floor-crankhandle.NutPocketRound-0.1 &&
				math.Hypot(v.Y-dm.ClampY, v.Z-crankhandle.ArmThickness/2) <= m2fastener.NutCavityRadius+0.05
		},
	}, {
		Reason: "clamp head pocket floor rim",
		Near: func(v r3.Vec) bool {
			return math.Abs(v.X-(crankhandle.ArmRadius-m2fastener.HeadPocketDepth)) < 0.05 &&
				math.Hypot(v.Y-dm.ClampY, v.Z-crankhandle.ArmThickness/2) <= m2fastener.HeadPocketRadius+0.05
		},
	}}
	if c.slitRound {
		half := crankhandle.SlitWidth / 2
		out = append(out, cornercheck.Exempt{
			Reason: "slit end round in the bore's floor",
			Near: func(v r3.Vec) bool {
				return math.Hypot(v.X, v.Y-(-dm.BoreRadius+crankhandle.SlitInBore-half)) <= half+0.05 &&
					v.Z > crankhandle.ArmThickness-crankhandle.BoreFloor
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

// TestCrankHandleCrossPocketCorners proves that a section of the pin 1 mm inside the -Y flat shows exactly six findings
// at each cross nut pocket's corners, one in each of the six corner directions, so the exemption names real corners.
func TestCrankHandleCrossPocketCorners(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			fs, err := cornercheck.Section(t.Context(), m.Pin, m.SplitNormal, p.Throw-crankhandle.CrossFlat+1)
			require.NoError(t, err)
			for i, z := range crankhandle.CrossZ {
				near := crossPocketNear(p, i)
				var sectors []int
				for _, f := range fs {
					if !near(f.At) {
						continue
					}
					deg := math.Atan2(f.At.X, f.At.Z-z)*180/math.Pi - 30
					sectors = append(sectors, int(math.Round(math.Mod(deg+360, 360)/60))%6)
				}
				slices.Sort(sectors)
				require.Equal(t, []int{0, 1, 2, 3, 4, 5}, sectors, "cross nut pocket %d: findings by corner direction", i+1)
			}
		})
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
		"clamp bolt": {local.ClampBolt.Body, placed.ClampBolt.Body},
		"clamp nut":  {local.ClampNut.Body, placed.ClampNut.Body},
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
	require.InDelta(t, 0, placed.ClampSite.Head.Sub(xf.Apply(local.ClampSite.Head)).Len(), 1e-9)
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

// atX is the plane x = const of the local frame, normal +X, across the clamp bolt's line.
func atX(t *testing.T, x float64) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	return f
}

// requireClampBolt checks the clamp bolt across the arm's hub: an M2 x 16 whose head's top face is flush with the +X
// side face and inside its pocket, which crosses the slit and whose tip ends inside the -X side face, and whose nut
// lies inside its hexagonal pocket, below the -X side face; neither bolt nor nut shares volume with the arm in sections
// along and across the bolt. It also checks the pocket: open to the -X side face, a hexagon whose flats the nut's
// corners cannot pass, so the nut cannot turn, with room for the nut's corners.
func requireClampBolt(t *testing.T, m *crankhandle.Model) {
	t.Helper()
	dm, site := m.Dimensions, m.ClampSite
	mid := crankhandle.ArmThickness / 2
	floor := crankhandle.ArmRadius - site.Cavity // the nut pocket's floor
	bb, err := m.ClampBolt.Body.Bounds()
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
	nb, err := m.ClampNut.Body.Bounds()
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
	requireClampPocket(t, m)
	frames := map[string]r3.Frame{}
	for _, z := range []float64{2.3, 3.2, 4, 4.8, 5.7} {
		frames[fmt.Sprintf("z %g", z)] = atZ(t, z)
	}
	for _, x := range []float64{9.5, 8.5, 6, 3, 0, -3, -6, -7.9, -8.3, -9, -9.7} {
		frames[fmt.Sprintf("x %g", x)] = atX(t, x)
	}
	for name, f := range frames {
		a := section(t, m.Arm, f)
		for _, part := range []*decad.Body{m.ClampBolt.Body, m.ClampNut.Body} {
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
func requireClampPocket(t *testing.T, m *crankhandle.Model) {
	t.Helper()
	dm := m.Dimensions
	mid := crankhandle.ArmThickness / 2
	floor := crankhandle.ArmRadius - m.ClampSite.Cavity
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

// TestCrankHandleClampWalls checks that every wall round the clamp's cuts is at least 1 mm: the web between the bolt's
// hole and the bore, the arm round the nut pocket (to the bore, from its floor to the outline, to both faces, and the
// flat -X side face beyond its corner), the flat side face beyond the head pocket's ring, and the arm round and under
// the head pocket. It reads a shaft near the largest the arm takes (16.6 mm of the 16.7 limit) as well.
func TestCrankHandleClampWalls(t *testing.T) {
	widest := crankhandle.Params605ZZ()
	widest.ShaftDiameter = 16.6
	cases := append(append([]crankCase{}, crankCases...),
		crankCase{name: "widest-shaft", params: func() crankhandle.Params { return widest }})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dm, err := c.params().Derive()
			require.NoError(t, err)
			rad := m2fastener.NutCavityRadius
			flat := rad * math.Sqrt(3) / 2
			floor := crankhandle.ArmRadius - dm.ClampSite.Cavity
			pocketOutline := math.Inf(1)
			for _, y := range []float64{dm.ClampY - flat, dm.ClampY + flat} {
				pocketOutline = math.Min(pocketOutline, clampOutlineDistance(dm, floor, y))
			}
			mid := crankhandle.ArmThickness / 2
			for name, wall := range map[string]float64{
				"between the hole and the bore":                  -dm.ClampY - m2fastener.HoleRadius - dm.BoreRadius,
				"between the nut pocket and the bore":            math.Hypot(floor, math.Min(0, dm.ClampY+flat)) - dm.BoreRadius,
				"between the nut pocket's floor and the outline": pocketOutline,
				"between the nut pocket and each face":           mid - rad,
				"of -X side face beyond the nut pocket":          dm.ClampY - flat - dm.SideEnd,
				"of side face beyond the head pocket":            dm.ClampY - m2fastener.HeadPocketRadius - dm.SideEnd,
				"between the head pocket and each face":          mid - m2fastener.HeadPocketRadius,
				"under the head pocket to the slit": crankhandle.ArmRadius - m2fastener.HeadPocketDepth -
					crankhandle.SlitWidth/2,
			} {
				require.GreaterOrEqual(t, wall, 1.0-1e-9, "material %s", name)
			}
		})
	}
}
