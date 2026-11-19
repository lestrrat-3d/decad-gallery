package dovetail_test

// CI: isolated-tests

import (
	"context"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/features/dovetail"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	p := dovetail.DefaultParams()
	require.NoError(t, p.Validate())
	require.InDelta(t, 10, p.TailWidth, 1e-12)
	require.Equal(t, dovetail.Slope{Rise: 7, Run: 24}, p.Slope)
	require.InDelta(t, 16, p.Length, 1e-12)
	require.InDelta(t, 6, p.Depth, 1e-12)
	require.Equal(t, dovetail.TailOnFirst, p.TailOn)
	require.Equal(t, 13.5, p.TipWidth())
	require.Equal(t, 1025.0/2048, p.ChamferSize())
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

// TestHalvesMateExactly requires the halves, which touch along the whole joint with no gap, to share no volume, by
// decad's own pair checks: every vertex is exact, Document.Verify reads Sound with no interference, the intersection
// of the halves is empty, and their union has exactly the sum of their volumes.
func TestHalvesMateExactly(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			doc, d := build(t, p)
			for _, body := range []*decad.Body{d.First, d.Second} {
				for _, v := range body.Vertices() {
					require.Zero(t, v.Position().Bound.Base(), "a vertex is not exact")
				}
			}
			gap, separated, err := meshOf(t, d.First).Exceeds(meshOf(t, d.Second), 0)
			require.NoError(t, err)
			require.False(t, separated, "the halves touch: boundary gap lower bound %g", gap)
			require.Zero(t, gap)

			report, err := doc.Verify(t.Context())
			require.NoError(t, err)
			require.True(t, report.Status == decad.Sound, "Verify status %s", report.Status)
			require.Zero(t, len(report.Interferences), "interferences")
			require.Zero(t, len(report.Diagnostics), "diagnostics")

			tailV, err := d.TailBlock.Volume()
			require.NoError(t, err)
			socketV, err := d.SocketBlock.Volume()
			require.NoError(t, err)
			// The boolean operations need both halves in one container.
			both, err := doc.Root().NewChild("Both")
			require.NoError(t, err)
			require.NoError(t, both.AddBody(d.TailBlock))
			require.NoError(t, both.AddBody(d.SocketBlock))
			_, err = decad.Intersect(t.Context(), d.TailBlock, d.SocketBlock)
			var be *decad.BooleanError
			require.ErrorAs(t, err, &be, "Intersect of the halves must fail as empty")
			require.True(t, be.Code == decad.BooleanEmpty, "Intersect code %s", be.Code)
			union, err := decad.Union(t.Context(), d.TailBlock, d.SocketBlock)
			require.NoError(t, err)
			unionV, err := union.Volume()
			require.NoError(t, err)
			require.Equal(t, tailV.Value.Base()+socketV.Value.Base(), unionV.Value.Base(), "union volume differs from the sum")
		})
	}
}

// TestEndFacesHaveNoSeam requires each block to have one face at each end of the joint: the tail's end face and its
// host's end face are one face with no edge between them, and the socket host's end face is one face too.
func TestEndFacesHaveNoSeam(t *testing.T) {
	for name, p := range cases() {
		t.Run(name, func(t *testing.T) {
			_, d := build(t, p)
			profile := (p.TailWidth+p.TipWidth())/2*p.Depth - 2*p.ChamferCut()
			block := d.Width * d.Height
			for _, c := range []struct {
				what string
				body *decad.Body
				area float64
			}{{"tail block", d.TailBlock, block + profile}, {"socket block", d.SocketBlock, block - profile - 4*p.ChamferCut()}} {
				for _, v := range []float64{0, p.Length} {
					faces := endFaces(c.body, v)
					require.Len(t, faces, 1, "%s: faces at V = %g", c.what, v)
					require.Len(t, faces[0].Loops(), 1, "%s: loops of the end face at V = %g", c.what, v)
					area, err := faces[0].Area()
					require.NoError(t, err)
					require.InDelta(t, c.area, area.Value.Base(), area.Bound.Base()+1e-6, "%s: end face area at V = %g", c.what, v)
				}
			}
		})
	}
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
		"slope zero":         func(p *dovetail.Params) { p.Slope = dovetail.Slope{} },
		"slope not a triple": func(p *dovetail.Params) { p.Slope = dovetail.Slope{Rise: 1, Run: 6} },
		"slope too steep":    func(p *dovetail.Params) { p.Slope = dovetail.Slope{Rise: 3, Run: 4} },
		"slope too shallow":  func(p *dovetail.Params) { p.Slope = dovetail.Slope{Rise: 33, Run: 544} },
		"slope scaled":       func(p *dovetail.Params) { p.Slope = dovetail.Slope{Rise: 14, Run: 48} },
		"depth inexact":      func(p *dovetail.Params) { p.Depth = 6.2 },
		"depth nan":          func(p *dovetail.Params) { p.Depth = math.NaN() },
		"width":              func(p *dovetail.Params) { p.TailWidth = dovetail.MinTailWidth - 0.1 },
		"depth":              func(p *dovetail.Params) { p.Depth = dovetail.MinDepth - 0.1 },
		"length":             func(p *dovetail.Params) { p.Length = dovetail.MinLength - 0.1 },
		"carrier":            func(p *dovetail.Params) { p.TailOn = dovetail.Carrier(7) },
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
	// The error for a refused slope names the allowed ones, and for an inexact depth the depth rule.
	p := good
	p.Slope = dovetail.Slope{Rise: 1, Run: 6}
	require.ErrorContains(t, p.Validate(), "7:24")
	p = good
	p.Depth = 6.2
	require.ErrorContains(t, p.Validate(), "power of two")
	_, err := dovetail.BuildDemo(t.Context(), nil, good)
	require.ErrorContains(t, err, "document")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = dovetail.BuildDemo(ctx, decad.New(), good)
	require.Error(t, err)
}
