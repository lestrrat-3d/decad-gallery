package crankhandle_test

// CI: isolated-tests

import (
	"context"
	"math"
	"slices"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/crankhandle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestCrankHandleContainers(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, doc := built(t, c)
			require.NoError(t, doc.VerifyContainers().Err())
			want := map[string][]*decad.Body{
				crankhandle.ArmContainerName:    {m.Arm},
				crankhandle.PinContainerName:    {m.Pin},
				crankhandle.HandleContainerName: {m.Handle},
				crankhandle.BoltsContainerName:  {m.Bolt.Body, m.CrossBolts[0].Body, m.CrossBolts[1].Body, m.ClampBolts[0].Body, m.ClampBolts[1].Body},
				crankhandle.NutsContainerName:   {m.Nut.Body, m.CrossNuts[0].Body, m.CrossNuts[1].Body, m.ClampNuts[0].Body, m.ClampNuts[1].Body},
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
		"arm":    {{X: -10, Y: -10, Z: 0}, {X: 10, Y: 65, Z: crankhandle.HubLength}},
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
// bore is at least 1.6 mm, except the web over a clamp bolt's hole at that hole's height. The slit joins the
// bore to the outline, so the bore is read as the points within r_b of the axis outside the slit.
func TestCrankHandleBoreFit(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			rs, fs := p.ShaftDiameter/2, p.ShaftDiameter/2-p.FlatDepth
			half := crankhandle.SlitWidth / 2
			web := math.Min(crankhandle.PressWall, -m.Dimensions.ClampY-m2fastener.HoleRadius-m.Dimensions.BoreBack)
			for z, least := range map[float64]float64{
				1: crankhandle.PressWall, crankhandle.ClampZ[0]: web, 6.5: crankhandle.PressWall, 10: crankhandle.PressWall,
				crankhandle.ClampZ[1]: web, 15: crankhandle.PressWall,
			} {
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
					if p.DoubleFlat {
						gap = math.Max(gap, -v.Y-fs)
					}
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
			// At the boss top the floor closes the bore everywhere but the slit's round end.
			o := section(t, m.Arm, atZ(t, crankhandle.HubLength-0.25))
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
					if v.Y < -dm.Reach+1.5 || v.Y > -dm.BoreBack-0.3 {
						continue
					}
					gap = math.Min(gap, math.Abs(v.X))
				}
			}
			require.InDelta(t, half, gap, bound, "the slit's walls")
			floor := section(t, m.Arm, atZ(t, crankhandle.HubLength-0.5))
			top := math.Inf(-1)
			for _, l := range floor.Loops {
				for _, v := range l {
					if math.Abs(v.X) <= half+bound && v.Y < 0 {
						top = math.Max(top, v.Y)
					}
				}
			}
			require.InDelta(t, -dm.BoreBack+crankhandle.SlitInBore, top, floor.OffSurface.Base()+1e-3, "the slit's round end")
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
			for i := range m.ClampSites {
				requireClampBolt(t, m, i)
			}
		})
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

// TestCrankHandleEdgeFinishing checks the 0.5 mm chamfers on the arm's faces. A section 0.25 mm inside the back face is
// 0.25 mm short of the arm's ends and sides all round. At the same depth below the 8 mm outer face the pin's end and the
// sides beyond the ramp's foot are 0.25 mm in. A section 0.25 mm under the boss top is short of the clamp's end arc as
// before and of the sides; and 0.1 mm under the boss top the ramp's edge is chamfered, which pulls its largest y in.
func TestCrankHandleEdgeFinishing(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			for _, z := range []float64{0.25, crankhandle.ArmThickness - 0.25, crankhandle.HubLength - 0.25} {
				o := section(t, m.Arm, atZ(t, z))
				bound := o.OffSurface.Base() + 1e-3
				lo, hi := math.Inf(1), math.Inf(-1)
				left, right := math.Inf(1), math.Inf(-1)
				for _, l := range o.Loops {
					for _, v := range l {
						if z == crankhandle.ArmThickness-0.25 && v.Y <= m.Dimensions.RampFoot+1 {
							continue
						}
						lo, hi = math.Min(lo, v.Y), math.Max(hi, v.Y)
						left, right = math.Min(left, v.X), math.Max(right, v.X)
					}
				}
				if z == crankhandle.ArmThickness-0.25 {
					require.InDelta(t, p.Throw+crankhandle.ArmRadius-0.25, hi, bound, "z %g: pin end", z)
					require.InDelta(t, -crankhandle.ArmRadius+0.25, left, bound, "z %g: -X side", z)
					require.InDelta(t, crankhandle.ArmRadius-0.25, right, bound, "z %g: +X side", z)
					continue
				}
				// The slit cuts the end arc's lowest point away: the section's lowest point lies on the arc at the slit's wall.
				arcR, half := crankhandle.ClampEndR-0.25, crankhandle.SlitWidth/2
				sag := arcR - math.Sqrt(arcR*arcR-half*half)
				require.InDelta(t, -m.Dimensions.Reach+0.25+sag, lo, bound, "z %g: end arc", z)
				if z < crankhandle.ArmThickness {
					require.InDelta(t, p.Throw+crankhandle.ArmRadius-0.25, hi, bound, "z %g: pin end", z)
				}
				require.InDelta(t, -crankhandle.ArmRadius+0.25, left, bound, "z %g: -X side", z)
				require.InDelta(t, crankhandle.ArmRadius-0.25, right, bound, "z %g: +X side", z)
			}
			// The boss top ends at the ramp's top edge, y = ArmRadius: the 0.5 mm chamfer there, between the top and the 30
			// degree ramp, pulls the section 0.1 mm under the top back to ArmRadius - s + 0.1 (s + s cos 30)/(s sin 30), where
			// s = ChamferSize. Unchamfered it would read ArmRadius + 0.173.
			s := crankhandle.ChamferSize
			rad := crankhandle.RampAngle * math.Pi / 180
			want := crankhandle.ArmRadius - s + 0.1*(s+s*math.Cos(rad))/(s*math.Sin(rad))
			o := section(t, m.Arm, atZ(t, crankhandle.HubLength-0.1))
			top := math.Inf(-1)
			for _, l := range o.Loops {
				for _, v := range l {
					top = math.Max(top, v.Y)
				}
			}
			require.InDelta(t, want, top, o.OffSurface.Base()+1e-3, "the ramp top's chamfer")
		})
	}
}

// TestCrankHandleBoreRim checks the 0.5 mm chamfer on the bore's rim in the back face, which the slit opens into two
// arcs and the flat, or with DoubleFlat two arcs and both flats. A section z above the face runs along the arc of radius
// r_b + d - z and along the flat at f_b + d - z (and -(f_b + d - z)), where d is the chamfer size. The bevel meets each
// slit wall where that arc does, or with DoubleFlat on the -Y flat at y = -(f_b + d - z).
func TestCrankHandleBoreRim(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			m, _ := built(t, c)
			rb, fb := m.Dimensions.BoreRadius, m.Dimensions.BoreFlat
			double := c.params().DoubleFlat
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
					onArc := math.Abs(r-(rb+d)) <= tol && v.Y <= fb+d+tol && (!double || v.Y >= -(fb+d)-tol)
					onFlat := math.Abs(v.Y-(fb+d)) <= tol && r <= rb+d+tol
					if double {
						onFlat = onFlat || math.Abs(v.Y+(fb+d)) <= tol && r <= rb+d+tol
					}
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
				if double {
					want = -(fb + d)
				}
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
				"arm":    {m.Arm, r3.Vec{Z: 1}, armExempt(m, pocket)},
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
		"clamp bolt": {local.ClampBolts[0].Body, placed.ClampBolts[0].Body},
		"clamp nut":  {local.ClampNuts[0].Body, placed.ClampNuts[0].Body},
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
	require.InDelta(t, 0, placed.ClampSites[0].Head.Sub(xf.Apply(local.ClampSites[0].Head)).Len(), 1e-9)
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
		{"short-throw", func(p *crankhandle.Params) { p.Throw = 35.2 }},
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
	t.Run("min-throw-accepted", func(t *testing.T) {
		p := def
		p.Throw = 35.3
		require.NoError(t, p.Validate())
	})
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
			floor := crankhandle.ArmRadius - dm.ClampSites[0].Cavity
			pocketOutline := math.Inf(1)
			for _, y := range []float64{dm.ClampY - flat, dm.ClampY + flat} {
				pocketOutline = math.Min(pocketOutline, clampOutlineDistance(dm, floor, y))
			}
			gap := crankhandle.ClampZ[1] - crankhandle.ClampZ[0]
			for name, wall := range map[string]float64{
				"between the nut pockets":                        gap - 2*rad,
				"between the head pockets":                       gap - 2*m2fastener.HeadPocketRadius,
				"between nut pocket 1 and the back face":         crankhandle.ClampZ[0] - rad,
				"between nut pocket 2 and the boss top":          crankhandle.HubLength - crankhandle.ClampZ[1] - rad,
				"between head pocket 1 and the back face":        crankhandle.ClampZ[0] - m2fastener.HeadPocketRadius,
				"between head pocket 2 and the boss top":         crankhandle.HubLength - crankhandle.ClampZ[1] - m2fastener.HeadPocketRadius,
				"between the hole and the bore":                  -dm.ClampY - m2fastener.HoleRadius - dm.BoreBack,
				"between the nut pocket and the bore":            math.Hypot(floor, math.Min(0, dm.ClampY+flat)) - dm.BoreRadius,
				"between the nut pocket's floor and the outline": pocketOutline,

				"of -X side face beyond the nut pocket": dm.ClampY - flat - dm.SideEnd,
				"of side face beyond the head pocket":   dm.ClampY - m2fastener.HeadPocketRadius - dm.SideEnd,

				"under the head pocket to the slit": crankhandle.ArmRadius - m2fastener.HeadPocketDepth -
					crankhandle.SlitWidth/2,
			} {
				require.GreaterOrEqual(t, wall, 1.0-1e-9, "material %s", name)
			}
		})
	}
}

// TestCrankHandleGrip pins the geometry that sets the clamp's grip on the shaft, so the capacity estimate in
// instructions.md ("Clamp capacity") cannot fall without a test failing: two clamp bolts at ClampZ, the bore's depth to
// within BoreFloor of the boss top, with DoubleFlat a second flat opposite the first, and the estimate's torque.
func TestCrankHandleGrip(t *testing.T) {
	for _, c := range crankCases {
		t.Run(c.name, func(t *testing.T) {
			p := c.params()
			m, _ := built(t, c)
			require.Len(t, m.ClampSites, 2)
			for i, site := range m.ClampSites {
				require.InDelta(t, crankhandle.ClampZ[i], site.Head.Z, 1e-9, "clamp bolt %d", i+1)
			}
			rb, fb := m.Dimensions.BoreRadius, m.Dimensions.BoreFlat
			depth := crankhandle.HubLength - crankhandle.BoreFloor
			// The bore ends at its floor: a section just under it holds bore points, one just over holds none within
			// BoreRadius - 0.5 of the axis outside the slit band.
			bore := func(z float64) int {
				o := section(t, m.Arm, atZ(t, z))
				n := 0
				for _, l := range o.Loops {
					for _, v := range l {
						if math.Hypot(v.X, v.Y) >= rb-0.5 && math.Hypot(v.X, v.Y) <= rb+0.1 && math.Abs(v.X) > crankhandle.SlitWidth/2+0.05 {
							n++
						}
					}
				}
				return n
			}
			require.Positive(t, bore(depth-0.1), "bore points under the floor")
			require.Zero(t, bore(depth+0.1), "bore points over the floor")
			if p.DoubleFlat {
				w := math.Sqrt(rb*rb - fb*fb)
				o := section(t, m.Arm, atZ(t, 8))
				tol := o.OffSurface.Base() + 1e-3
				for _, side := range []float64{1, -1} {
					found := false
					for _, l := range o.Loops {
						for _, v := range l {
							found = found || math.Abs(v.Y-side*fb) <= tol && math.Abs(v.X) <= w-0.2
						}
					}
					require.True(t, found, "bore flat at y %g", side*fb)
				}
			}
			// The capacity estimate: form fit on the flats plus friction from the bolts' preload, in N.mm.
			l := crankhandle.HubLength - crankhandle.BoreFloor - 0.25 - crankhandle.ChamferSize
			r := p.ShaftDiameter / 2
			w2 := r*r - (r-p.FlatDepth)*(r-p.FlatDepth)
			k := 1.0
			if p.DoubleFlat {
				k = 2
			}
			torque := k*30*l*w2/3 + 0.3*2*2*200*r
			require.GreaterOrEqual(t, l, 15.25, "engagement")
			if p.DoubleFlat {
				require.GreaterOrEqual(t, torque, 2600.0, "estimated holding torque, N.mm")
			}
		})
	}
}
