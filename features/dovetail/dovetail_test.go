package dovetail_test

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/features/dovetail"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// cases covers every parameter branch: both carriers, both ends of the angle range, and the smallest and a large joint.
func cases() map[string]dovetail.Params {
	second := dovetail.DefaultParams()
	second.TailOn = dovetail.TailOnSecond
	minAngle := dovetail.DefaultParams()
	minAngle.Angle = dovetail.MinAngle
	maxAngle := dovetail.DefaultParams()
	maxAngle.Angle = dovetail.MaxAngle
	smallest := dovetail.Params{TailWidth: dovetail.MinTailWidth, Angle: 15, Length: dovetail.MinLength,
		Depth: dovetail.MinDepth}
	large := dovetail.Params{TailWidth: 20, Angle: 12, Length: 30, Depth: 10, TailOn: dovetail.TailOnSecond}
	return map[string]dovetail.Params{
		"default": dovetail.DefaultParams(), "tail on second": second, "min angle": minAngle, "max angle": maxAngle,
		"smallest": smallest, "large": large,
	}
}

func build(t *testing.T, p dovetail.Params) (*decad.Document, *dovetail.Demo) {
	t.Helper()
	doc := decad.New()
	d, err := dovetail.BuildDemo(t.Context(), doc, p)
	require.NoError(t, err)
	return doc, d
}

func requireFullyConstrained(t *testing.T, name string, s *sketch.Sketch) {
	t.Helper()
	report := s.Verify(t.Context())
	require.True(t, report.Status == sketch.FullyConstrained, "%s: status %s", name, report.Status)
	require.Zero(t, report.DOF, name)
	require.Zero(t, len(report.Redundant), name)
	require.Zero(t, len(report.Conflicts), name)
	require.NoError(t, report.Check(), name)
}

func TestDefaults(t *testing.T) {
	p := dovetail.DefaultParams()
	require.NoError(t, p.Validate())
	require.InDelta(t, 10, p.TailWidth, 1e-12)
	require.InDelta(t, 10, p.Angle, 1e-12)
	require.InDelta(t, 16, p.Length, 1e-12)
	require.InDelta(t, 6, p.Depth, 1e-12)
	require.Equal(t, dovetail.TailOnFirst, p.TailOn)
	require.InDelta(t, 10+12*math.Tan(10*math.Pi/180), p.TipWidth(), 1e-12)
}

func TestSketchesFullyConstrained(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			require.Len(t, d.Sketches, 4, "two block outlines and the tail and socket profiles")
			for i, s := range d.Sketches {
				requireFullyConstrained(t, []string{"first block", "second block", "tail", "socket"}[i], s)
			}
		})
	}
}

func TestBoundsAndVolumes(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			require.True(t, d.First.IsSolid())
			require.True(t, d.Second.IsSolid())
			h, w := d.Height, d.Width
			area := (p.TailWidth + p.TipWidth()) / 2 * p.Depth
			blank := w * p.Length * h
			// First spans N from -h to 0 and Second from 0 to h, except the tail's host, which also holds the tail.
			firstLo, firstHi, secondLo, secondHi := -h, 0.0, 0.0, h
			if p.TailOn == dovetail.TailOnFirst {
				firstHi = p.Depth
			} else {
				secondLo = -p.Depth
			}
			for _, c := range []struct {
				body   *decad.Body
				lo, hi float64
			}{{d.First, firstLo, firstHi}, {d.Second, secondLo, secondHi}} {
				box, err := c.body.Bounds()
				require.NoError(t, err)
				tol := box.Bound.Base() + 1e-6
				require.InDelta(t, -w/2, box.Min.X, tol)
				require.InDelta(t, w/2, box.Max.X, tol)
				require.InDelta(t, 0, box.Min.Y, tol)
				require.InDelta(t, p.Length, box.Max.Y, tol)
				require.InDelta(t, c.lo, box.Min.Z, tol)
				require.InDelta(t, c.hi, box.Max.Z, tol)
			}
			// The tail adds its profile less its two wide-end chamfers. The socket loses the full profile and its two
			// lip chamfers besides.
			tailV, err := d.TailBlock.Volume()
			require.NoError(t, err)
			socketV, err := d.SocketBlock.Volume()
			require.NoError(t, err)
			require.InDelta(t, blank+(area-2*p.ChamferCut())*p.Length, tailV.Value.Base(),
				tailV.Bound.Base()+1e-6)
			require.InDelta(t, blank-(area+2*p.ChamferCut())*p.Length, socketV.Value.Base(), socketV.Bound.Base()+1e-6)
		})
	}
}

// profileCorners returns the dovetail profile's four outline corners in the joint frame's (U, N) coordinates for a
// tail leaving its host at N = 0 toward direction s: the neck's two corners on the mating plane and the two at the
// wide end.
func profileCorners(p dovetail.Params, s float64) (neck, wide [2]r3.Vec) {
	nk, wd := p.TailWidth/2, p.TipWidth()/2
	neck = [2]r3.Vec{{X: -nk}, {X: nk}}
	wide = [2]r3.Vec{{X: -wd, Z: s * p.Depth}, {X: wd, Z: s * p.Depth}}
	return neck, wide
}

// chamferPoints returns the four points where each chamfer meets its two faces, in (U, N) coordinates. The tail's
// wide-end chamfers run from the wide-end face to the flank, and the socket's lip chamfers from the mating face to the
// flank. Each chamfer has ChamferSize setback on both faces.
func chamferPoints(p dovetail.Params, s float64) (tail, socket []r3.Vec) {
	a := p.Angle * math.Pi / 180
	c, sin, cos := dovetail.ChamferSize, math.Sin(a), math.Cos(a)
	nk, wd := p.TailWidth/2, p.TipWidth()/2
	for _, su := range []float64{-1, 1} {
		tail = append(tail, r3.Vec{X: su * (wd - c), Z: s * p.Depth}, r3.Vec{X: su * (wd - c*sin), Z: s * (p.Depth - c*cos)})
		socket = append(socket, r3.Vec{X: su * (nk + c)}, r3.Vec{X: su * (nk + c*sin), Z: s * c * cos})
	}
	return tail, socket
}

// requireOutlineHas requires the section outline at mid-length to pass within tol of each point.
func requireOutlineHas(t *testing.T, ctx context.Context, name string, body *decad.Body, length float64, pts []r3.Vec,
	tol float64) {
	t.Helper()
	outline, err := cornercheck.Outline(ctx, body, r3.Vec{Y: 1}, length/2)
	require.NoError(t, err)
	for _, pt := range pts {
		require.LessOrEqual(t, nearestVertex(outline, pt), tol, "%s: no outline vertex near (%.3f, %.3f)", name, pt.X, pt.Z)
	}
}

// requireOutlineLacks requires the section outline at mid-length to have no vertex within clear of any point.
func requireOutlineLacks(t *testing.T, ctx context.Context, name string, body *decad.Body, length float64, pts []r3.Vec, clear float64) {
	t.Helper()
	outline, err := cornercheck.Outline(ctx, body, r3.Vec{Y: 1}, length/2)
	require.NoError(t, err)
	for _, pt := range pts {
		require.Greater(t, nearestVertex(outline, pt), clear, "%s: outline vertex at sharp corner (%.3f, %.3f)", name, pt.X, pt.Z)
	}
}

func nearestVertex(outline cornercheck.SectionOutline, pt r3.Vec) float64 {
	least := math.Inf(1)
	for _, loop := range outline.Loops {
		for _, v := range loop {
			least = math.Min(least, math.Hypot(v.X-pt.X, v.Z-pt.Z))
		}
	}
	return least
}

// TestTailAndSocketShareProfile requires the tail's concave neck corners and the socket's concave floor corners to stay
// sharp at nominal size, and the tail's wide-end corners and the socket's lips to carry the chamfer.
func TestTailAndSocketShareProfile(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			s := 1.0
			if p.TailOn == dovetail.TailOnSecond {
				s = -1
			}
			neck, wide := profileCorners(p, s)
			tailChamfer, socketChamfer := chamferPoints(p, s)
			ctx := t.Context()
			// Tail block: sharp root (concave), chamfered wide end (convex).
			requireOutlineHas(t, ctx, "tail block root", d.TailBlock, p.Length, neck[:], 0.02)
			requireOutlineHas(t, ctx, "tail block chamfer", d.TailBlock, p.Length, tailChamfer, 0.02)
			requireOutlineLacks(t, ctx, "tail block wide end", d.TailBlock, p.Length, wide[:], 0.1)
			// Socket block: chamfered lips (convex), sharp floor corners (concave).
			requireOutlineHas(t, ctx, "socket block floor", d.SocketBlock, p.Length, wide[:], 0.02)
			requireOutlineHas(t, ctx, "socket block chamfer", d.SocketBlock, p.Length, socketChamfer, 0.02)
			requireOutlineLacks(t, ctx, "socket block lip", d.SocketBlock, p.Length, neck[:], 0.1)
		})
	}
}

// edgeAlongV reports whether body has a straight edge along V through (u, n) in the demo frame, from V = 0 to length.
func edgeAlongV(body *decad.Body, pt r3.Vec, length float64) bool {
	const tol = 1e-4
	at := func(v r3.Vec, y float64) bool {
		return math.Abs(v.X-pt.X) < tol && math.Abs(v.Z-pt.Z) < tol && math.Abs(v.Y-y) < tol
	}
	for _, e := range body.Edges() {
		a, b := e.Start().Position().Value, e.End().Position().Value
		if at(a, 0) && at(b, length) || at(a, length) && at(b, 0) {
			return true
		}
	}
	return false
}

// TestConvexEdgesChamferedConcaveSharp reads the edges of each block: no straight edge stays at a convex joint corner,
// one stands at each chamfer's two ends, and the concave corners keep their edges.
func TestConvexEdgesChamferedConcaveSharp(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			s := 1.0
			if p.TailOn == dovetail.TailOnSecond {
				s = -1
			}
			neck, wide := profileCorners(p, s)
			tailChamfer, socketChamfer := chamferPoints(p, s)
			for _, c := range []struct {
				what  string
				body  *decad.Body
				pts   []r3.Vec
				found bool
			}{
				{"tail block concave root", d.TailBlock, neck[:], true},
				{"tail block convex wide end", d.TailBlock, wide[:], false},
				{"tail block chamfer ends", d.TailBlock, tailChamfer, true},
				{"socket block concave floor", d.SocketBlock, wide[:], true},
				{"socket block convex lip", d.SocketBlock, neck[:], false},
				{"socket block chamfer ends", d.SocketBlock, socketChamfer, true},
			} {
				for _, pt := range c.pts {
					require.Equal(t, c.found, edgeAlongV(c.body, pt, p.Length), "%s at (%.3f, %.3f)", c.what, pt.X, pt.Z)
				}
			}
		})
	}
}

// meshOf tessellates body with a held bound and prepares it for a distance query.
func meshOf(t *testing.T, body *decad.Body) meshgap.Mesh {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.02), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified())
	require.True(t, mesh.VolumeVerified())
	mg, err := meshgap.New(mesh.Vertices(), mesh.Triangles(), mesh.Bound().Base())
	require.NoError(t, err)
	return mg
}

// TestHalvesMateWithoutOverlap requires the halves to touch with no gap (the nominal fit) and to share no volume.
// meshgap reads boundary distance only, so it proves the zero gap and cannot tell touching from overlapping. Overlap
// is excluded two ways: decad classifies a grid of points across the mid-length section and none lies inside both
// halves, and TestBoundsAndVolumes shows the tail adds the volume the socket removes. decad's pair checks
// (Document.Verify, Union, Intersect) refuse two halves in exact contact, so they cannot be used here.
func TestHalvesMateWithoutOverlap(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			gap, separated, err := meshOf(t, d.First).Exceeds(meshOf(t, d.Second), 0)
			require.NoError(t, err)
			require.False(t, separated, "the halves touch: boundary gap lower bound %g", gap)
			require.Zero(t, gap)

			var insideTail, insideSocket, insideBoth int
			const step = 0.4
			for u := -d.Width/2 + step/2; u < d.Width/2; u += step {
				for n := -d.Height - step/2; n < d.Height+p.Depth; n += step {
					pt := r3.Vec{X: u, Y: p.Length / 2, Z: n}
					inTail := locate(t, d.TailBlock, pt)
					inSocket := locate(t, d.SocketBlock, pt)
					if inTail == decad.PointInside {
						insideTail++
					}
					if inSocket == decad.PointInside {
						insideSocket++
					}
					if inTail == decad.PointInside && inSocket == decad.PointInside {
						insideBoth++
					}
				}
			}
			require.Zero(t, insideBoth, "points inside both halves")
			require.Greater(t, insideTail, 20)
			require.Greater(t, insideSocket, 20)
		})
	}
}

func locate(t *testing.T, body *decad.Body, pt r3.Vec) decad.PointLocation {
	t.Helper()
	loc, err := body.LocatePoint(t.Context(), pt, units.Millimeters(0.02))
	require.NoError(t, err)
	return loc
}

func TestBodyContainers(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			doc, d := build(t, p)
			var names []string
			for _, c := range doc.Root().Children() {
				names = append(names, c.Name())
			}
			require.ElementsMatch(t, []string{dovetail.TailBlocksContainerName, dovetail.SocketBlocksContainerName}, names)
			for want, body := range map[string]*decad.Body{dovetail.TailBlocksContainerName: d.TailBlock,
				dovetail.SocketBlocksContainerName: d.SocketBlock} {
				c, err := doc.ContainerOf(body)
				require.NoError(t, err)
				require.Equal(t, want, c.Name())
			}
			require.Len(t, doc.Bodies(), 2)
			require.NoError(t, doc.VerifyContainers().Err())
		})
	}
}

// TestCorners applies "Corners and roots" to each block, printed with the slide direction up so the profile lies in
// every layer. The only sharp corners are the two concave pairs listed in instructions.md: the tail's root corners and
// the socket's floor corners. The convex corners, the tail's wide end and the socket's lips, carry the chamfer.
func TestCorners(t *testing.T) {
	axis := r3.Vec{Y: 1}
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			s := 1.0
			if p.TailOn == dovetail.TailOnSecond {
				s = -1
			}
			neck, wide := profileCorners(p, s)
			near := func(c r3.Vec) cornercheck.Exempt {
				return cornercheck.Exempt{Reason: "mating corner", Near: cornercheck.Disc(c, axis, 0.3)}
			}
			fs, err := cornercheck.Check(t.Context(), d.TailBlock, axis, []cornercheck.Exempt{near(neck[0]), near(neck[1])})
			require.NoError(t, err)
			require.Empty(t, fs, "tail block: %s", cornercheck.Report(fs, 5))
			fs, err = cornercheck.Check(t.Context(), d.SocketBlock, axis, []cornercheck.Exempt{near(wide[0]), near(wide[1])})
			require.NoError(t, err)
			require.Empty(t, fs, "socket block: %s", cornercheck.Report(fs, 5))
		})
	}
}

func TestSlidesApart(t *testing.T) {
	// A pull along the slide direction frees the halves: moving the socket block by Length clears the tail.
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			moved, err := decad.Transform(t.Context(), d.SocketBlock, mustTranslate(t, r3.Vec{Y: p.Length + 1}))
			require.NoError(t, err)
			_, separated, err := meshOf(t, d.TailBlock).Exceeds(meshOf(t, moved), 0.5)
			require.NoError(t, err)
			require.True(t, separated, "the halves part along the slide direction")
		})
	}
}

func mustTranslate(t *testing.T, v r3.Vec) r3.Transform {
	t.Helper()
	xf, err := r3.Translation(v)
	require.NoError(t, err)
	return xf
}

// hostBlock extrudes a rectangular host from the mating plane of frame f: width along U, length along V, height along
// N toward direction dir.
func hostBlock(t *testing.T, doc *decad.Document, f r3.Frame, width, length, height float64, dir decad.Direction) *decad.Body {
	t.Helper()
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(f)
	require.NoError(t, err)
	sk, err := w.CreateSketch(plane)
	require.NoError(t, err)
	p0, p1 := sk.CreatePoint(-width/2, 0), sk.CreatePoint(width/2, 0)
	p2, p3 := sk.CreatePoint(width/2, length), sk.CreatePoint(-width/2, length)
	l0, l1, l2, l3 := sk.CreateLine(p0, p1), sk.CreateLine(p1, p2), sk.CreateLine(p2, p3), sk.CreateLine(p3, p0)
	sk.AddConstraint(sketch.NewHorizontal(l0), sketch.NewHorizontal(l2), sketch.NewVertical(l1), sketch.NewVertical(l3),
		sketch.NewMidpoint(sk.Origin(), l0), sketch.NewDistance(p0, p1, width), sketch.NewVerticalDistance(p1, p2, length))
	_, err = sk.Solve(t.Context())
	require.NoError(t, err)
	requireFullyConstrained(t, "host block", sk)
	var chosen *sketch.Profile
	for _, c := range sk.Profiles() {
		if c.Valid {
			chosen = c
		}
	}
	require.NotNil(t, chosen)
	body, err := decad.Extrude(t.Context(), doc, sk, chosen, decad.Distance{D: units.Millimeters(height), Dir: dir})
	require.NoError(t, err)
	return body
}

// TestApplyAtAnotherFrame applies the joint at a frame turned and moved from the world axes and checks the result in
// that frame, for each carrier.
func TestApplyAtAnotherFrame(t *testing.T) {
	f, err := r3.NewFrame(r3.Vec{X: 5, Y: 3, Z: 2}, r3.Vec{Y: 1}, r3.Vec{X: -1})
	require.NoError(t, err)
	for _, carrier := range []dovetail.Carrier{dovetail.TailOnFirst, dovetail.TailOnSecond} {
		t.Run(carrier.String(), func(t *testing.T) {
			p := dovetail.DefaultParams()
			p.TailOn = carrier
			doc := decad.New()
			w, h := p.TipWidth()+2*dovetail.DemoWall, p.Depth+dovetail.DemoWall
			first := hostBlock(t, doc, f, w, p.Length, h, decad.Against)
			second := hostBlock(t, doc, f, w, p.Length, h, decad.Along)
			j, err := dovetail.Apply(t.Context(), doc, p, f, first, second)
			require.NoError(t, err)
			require.Len(t, j.Sketches(), 2)
			for _, s := range j.Sketches() {
				requireFullyConstrained(t, "joint profile", s)
			}
			area := (p.TailWidth + p.TipWidth()) / 2 * p.Depth
			blank := w * p.Length * h
			tailV, err := j.TailBody(p).Volume()
			require.NoError(t, err)
			socketV, err := j.SocketBody(p).Volume()
			require.NoError(t, err)
			require.InDelta(t, blank+(area-2*p.ChamferCut())*p.Length, tailV.Value.Base(),
				tailV.Bound.Base()+1e-6)
			require.InDelta(t, blank-(area+2*p.ChamferCut())*p.Length, socketV.Value.Base(), socketV.Bound.Base()+1e-6)
			// The joint starts at world x = 5 and runs toward -X.
			box, err := j.First.Bounds()
			require.NoError(t, err)
			require.InDelta(t, 5-p.Length, box.Min.X, box.Bound.Base()+1e-6)
			require.InDelta(t, 5, box.Max.X, box.Bound.Base()+1e-6)
		})
	}
}

// TestApplyRefusesMisfitHosts requires a refusal, naming the problem, for hosts that do not span the joint.
func TestApplyRefusesMisfitHosts(t *testing.T) {
	p := dovetail.DefaultParams()
	f := dovetail.DemoFrame()
	w, h := p.TipWidth()+2*dovetail.DemoWall, p.Depth+dovetail.DemoWall
	for name, c := range map[string]struct {
		width, length, height float64
		want                  string
	}{
		"too long":    {w, p.Length + 2, h, "must span V"},
		"too narrow":  {p.TipWidth(), p.Length, h, "must cover U"},
		"too shallow": {w, p.Length, p.Depth, "must reach"},
	} {
		t.Run(name, func(t *testing.T) {
			doc := decad.New()
			first := hostBlock(t, doc, f, c.width, c.length, c.height, decad.Against)
			second := hostBlock(t, doc, f, c.width, c.length, c.height, decad.Along)
			_, err := dovetail.Apply(t.Context(), doc, p, f, first, second)
			require.ErrorContains(t, err, c.want)
		})
	}
}

func TestRejectsInvalidInputs(t *testing.T) {
	good := dovetail.DefaultParams()
	bad := map[string]func(*dovetail.Params){
		"angle low":  func(p *dovetail.Params) { p.Angle = dovetail.MinAngle - 0.1 },
		"angle high": func(p *dovetail.Params) { p.Angle = dovetail.MaxAngle + 0.1 },
		"angle nan":  func(p *dovetail.Params) { p.Angle = math.NaN() },
		"width":      func(p *dovetail.Params) { p.TailWidth = dovetail.MinTailWidth - 0.1 },
		"depth":      func(p *dovetail.Params) { p.Depth = dovetail.MinDepth - 0.1 },
		"length":     func(p *dovetail.Params) { p.Length = dovetail.MinLength - 0.1 },
		"carrier":    func(p *dovetail.Params) { p.TailOn = dovetail.Carrier(7) },
	}
	for name, mutate := range bad {
		p := good
		mutate(&p)
		require.Error(t, p.Validate(), name)
		doc := decad.New()
		_, err := dovetail.BuildDemo(t.Context(), doc, p)
		require.Error(t, err, name)
		require.Zero(t, len(doc.Bodies()), name)
	}
	_, err := dovetail.BuildDemo(t.Context(), nil, good)
	require.ErrorContains(t, err, "document")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = dovetail.BuildDemo(ctx, decad.New(), good)
	require.Error(t, err)
}
