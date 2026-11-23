package turbinefan_test

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/turbinefan"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// poseTransform returns the transform of PrintPoseFor(sz, pitch, hand, tilt, depth).
func poseTransform(t *testing.T, sz turbinefan.Size, pitch float64, hand turbinefan.Hand, tilt, depth float64) (r3.Frame,
	r3.Transform) {
	t.Helper()
	f, err := turbinefan.PrintPoseFor(sz, pitch, hand, tilt, depth)
	require.NoError(t, err)
	xf, err := r3.FromFrame(f)
	require.NoError(t, err)
	return f, xf
}

// posedBlade builds the blade of hand at rotor d and pitch in a fresh NewDocument, finds its rest tilt, moves it into
// the print pose and returns the document, the posed blade, the tilt and the flat depth.
func posedBlade(t *testing.T, d, pitch float64, hand turbinefan.Hand) (*decad.Document, *decad.Body, float64, float64) {
	t.Helper()
	sz := mustSize(t, d)
	doc := turbinefan.NewDocument()
	build := turbinefan.BuildBlade
	if hand == turbinefan.LeftHand {
		build = turbinefan.BuildBladeMirrored
	}
	blade, err := build(t.Context(), doc, pitch, sz)
	require.NoError(t, err)
	tilt, depth, err := turbinefan.RestTilt(t.Context(), blade.Body, sz, pitch, hand)
	require.NoError(t, err)
	_, xf := poseTransform(t, sz, pitch, hand, tilt, depth)
	posed, err := decad.Transform(t.Context(), blade.Body, xf)
	require.NoError(t, err)
	return doc, posed, tilt, depth
}

// sectionAt cuts body with the plane z = h of the print frame at the finest chord the body allows, 0.02 to 0.2.
func sectionAt(t *testing.T, body *decad.Body, h float64) decad.SectionOutline {
	t.Helper()
	var o decad.SectionOutline
	var err error
	for _, tol := range []float64{0.02, 0.05, 0.2} {
		if o, err = body.SectionOutline(t.Context(), atZ(t, h), units.Millimeters(tol)); err == nil {
			return o
		}
	}
	require.NoError(t, err)
	return o
}

// loopArea returns the area of a closed polyline in the XY plane, and its centroid.
func loopArea(l []r3.Vec) (float64, r3.Vec) {
	a, cx, cy := 0.0, 0.0, 0.0
	for i := range l {
		p, q := l[i], l[(i+1)%len(l)]
		c := p.X*q.Y - q.X*p.Y
		a += c
		cx += (p.X + q.X) * c
		cy += (p.Y + q.Y) * c
	}
	if a == 0 {
		return 0, l[0]
	}
	return math.Abs(a) / 2, r3.Vec{X: cx / (3 * a), Y: cy / (3 * a)}
}

// insideLoop reports whether (x, y) lies inside the closed polyline, by the even-odd rule.
func insideLoop(l []r3.Vec, x, y float64) bool {
	in := false
	for i := range l {
		a, c := l[i], l[(i+len(l)-1)%len(l)]
		if (a.Y > y) != (c.Y > y) && x < (c.X-a.X)*(y-a.Y)/(c.Y-a.Y)+a.X {
			in = !in
		}
	}
	return in
}

// requireBoxNear requires two boxes to agree within tol on every face; mirror negates y of want first.
func requireBoxNear(t *testing.T, want, got decad.Box, tol float64, mirror bool, name string) {
	t.Helper()
	lo, hi := want.Min, want.Max
	if mirror {
		lo.Y, hi.Y = -want.Max.Y, -want.Min.Y
	}
	for i, pair := range [][2]float64{
		{lo.X, got.Min.X}, {lo.Y, got.Min.Y}, {lo.Z, got.Min.Z}, {hi.X, got.Max.X}, {hi.Y, got.Max.Y}, {hi.Z, got.Max.Z},
	} {
		require.InDelta(t, pair[0], pair[1], tol, "%s bound %d", name, i)
	}
}

// cradleCase is one harness case: rotor, pitch in degrees and hand.
type cradleCase struct {
	name     string
	rotor    float64
	pitchDeg float64
	hand     turbinefan.Hand
}

// cradleCases are the harness cases. The left hand at 192 / 25 pairs with the right hand for the mirror check.
var cradleCases = []cradleCase{
	{"128", 128, 25, turbinefan.RightHand},
	{"192", 192, 25, turbinefan.RightHand},
	{"192-left", 192, 25, turbinefan.LeftHand},
	{"192-steep-left", 192, 40, turbinefan.LeftHand},
	{"224-left", 224, 25, turbinefan.LeftHand},
	{"224-steep-left", 224, 40, turbinefan.LeftHand},
}

// builtCradle is one built harness case: its document, the posed blade, its rest tilt and flat depth and the harness.
type builtCradle struct {
	doc     *decad.Document
	blade   *decad.Body
	tilt    float64
	depth   float64
	harness *turbinefan.BladeHarness
	sz      turbinefan.Size
	hand    turbinefan.Hand
	err     error
}

var (
	builtCradleMu    sync.Mutex
	builtCradleCases = map[string]*builtCradle{}
)

// cradleFor builds a harness case on first use and shares it between the tests, which only read it. A build takes 12 to
// 30 s.
func cradleFor(t *testing.T, c cradleCase) *builtCradle {
	t.Helper()
	builtCradleMu.Lock()
	defer builtCradleMu.Unlock()
	if b, ok := builtCradleCases[c.name]; ok {
		require.NoError(t, b.err, c.name)
		return b
	}
	sz := mustSize(t, c.rotor)
	doc, blade, tilt, depth := posedBlade(t, c.rotor, c.pitchDeg*math.Pi/180, c.hand)
	h, err := turbinefan.BuildBladeHarness(t.Context(), doc, blade, sz, c.hand, tilt, depth)
	b := &builtCradle{doc: doc, blade: blade, tilt: tilt, depth: depth, harness: h, sz: sz, hand: c.hand, err: err}
	builtCradleCases[c.name] = b
	require.NoError(t, err, c.name)
	return b
}

// eachCradle runs fn as a subtest for each harness case.
func eachCradle(t *testing.T, fn func(t *testing.T, c cradleCase, b *builtCradle)) {
	t.Helper()
	for _, c := range cradleCases {
		t.Run(c.name, func(t *testing.T) { fn(t, c, cradleFor(t, c)) })
	}
}

// lowestCrossing returns the lowest z where the vertical line y meets the closed polyline (y, z taken from the points'
// Y and Z), and whether it meets it.
func lowestCrossing(l []r3.Vec, y float64) (float64, bool) {
	best, ok := math.Inf(1), false
	for i := range l {
		a, c := l[i], l[(i+1)%len(l)]
		if (a.Y-y)*(c.Y-y) > 0 || a.Y == c.Y {
			continue
		}
		if z := a.Z + (y-a.Y)/(c.Y-a.Y)*(c.Z-a.Z); z < best {
			best, ok = z, true
		}
	}
	return best, ok
}

// frame3 is the plane x = const of the print frame, U along +Y and V along +Z.
func frame3(x float64) r3.Frame {
	f, _ := r3.NewFrame(r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	return f
}

// pointSegment returns the distance in the XY plane from p to the segment a-c.
func pointSegment(p, a, c r3.Vec) float64 {
	dx, dy := c.X-a.X, c.Y-a.Y
	t := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		t = math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/l))
	}
	return math.Hypot(p.X-a.X-t*dx, p.Y-a.Y-t*dy)
}

// loopDistance returns the least distance in the XY plane between two closed polylines that do not cross: the least
// distance from any vertex of one to any segment of the other.
func loopDistance(p, q []r3.Vec) float64 {
	best := math.Inf(1)
	for _, pair := range [][2][]r3.Vec{{p, q}, {q, p}} {
		from, to := pair[0], pair[1]
		for _, v := range from {
			for k := range to {
				best = math.Min(best, pointSegment(v, to[k], to[(k+1)%len(to)]))
			}
		}
	}
	return best
}

// layerCheck is what slicing a body at every layer found: the number of layers, the loops with nothing in the
// layer below, and the outline point reaching farthest past the layer below.
type layerCheck struct {
	layers, islands int
	islandAt        []r3.Vec // up to three island centroids, z set to the layer
	worst           float64
	worstAt         r3.Vec
}

// checkLayers slices body at the middle of every layer of the given height from the bed up and compares each layer
// with the one below.
// A loop with no point inside the layer below, and no point of the layer below inside it, is an island; for every
// outline point outside the layer below it records the distance to that layer's outline.
func checkLayers(t *testing.T, body *decad.Body, layer float64) layerCheck {
	t.Helper()
	box, err := body.Bounds()
	require.NoError(t, err)
	inside := func(loops [][]r3.Vec, x, y float64) bool {
		in := false
		for _, l := range loops {
			if insideLoop(l, x, y) {
				in = !in
			}
		}
		return in
	}
	var out layerCheck
	var below [][]r3.Vec
	for z := layer / 2; z < box.Max.Z; z += layer {
		loops := sectionAt(t, body, z).Loops
		out.layers++
		for _, l := range loops {
			if below == nil {
				break
			}
			rests := false
			for _, p := range l {
				if rests = inside(below, p.X, p.Y); rests {
					break
				}
			}
			for _, bl := range below {
				for _, p := range bl {
					if rests {
						break
					}
					rests = insideLoop(l, p.X, p.Y)
				}
			}
			if !rests {
				out.islands++
				if len(out.islandAt) < 3 {
					_, cen := loopArea(l)
					out.islandAt = append(out.islandAt, r3.Vec{X: cen.X, Y: cen.Y, Z: z})
				}
			}
			for _, p := range l {
				if inside(below, p.X, p.Y) {
					continue
				}
				d := math.Inf(1)
				for _, bl := range below {
					for k := range bl {
						d = math.Min(d, pointSegment(p, bl[k], bl[(k+1)%len(bl)]))
					}
				}
				if d > out.worst {
					out.worst, out.worstAt = d, r3.Vec{X: p.X, Y: p.Y, Z: z}
				}
			}
		}
		below = loops
	}
	return out
}

// downward returns a triangle's angle from vertical, in degrees, and whether it points down more than 45 degrees
// (its unit normal's z below -sin 45° - slack).
func downward(v [3]r3.Vec, slack float64) (float64, bool) {
	n, ok := v[1].Sub(v[0]).Cross(v[2].Sub(v[0])).Normalize()
	if !ok || n.Z >= -math.Sin(math.Pi/4)-slack {
		return 0, false
	}
	return math.Asin(math.Min(1, -n.Z)) * 180 / math.Pi, true
}

// r3Lowest returns the lowest point of an outline's loops.
func r3Lowest(o decad.SectionOutline) r3.Vec {
	low := r3.Vec{Z: math.Inf(1)}
	for _, l := range o.Loops {
		for _, p := range l {
			if p.Z < low.Z {
				low = p
			}
		}
	}
	return low
}

// maxTop returns the greatest z of a top profile (x, z).
func maxTop(top [][2]float64) float64 {
	z := math.Inf(-1)
	for _, q := range top {
		z = math.Max(z, q[1])
	}
	return z
}

// loopSpanAtX returns the least and the greatest y where the line x = const meets the closed polyline in the XY plane.
func loopSpanAtX(l []r3.Vec, x float64) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for i := range l {
		a, c := l[i], l[(i+1)%len(l)]
		if (a.X-x)*(c.X-x) > 0 || a.X == c.X {
			continue
		}
		y := a.Y + (x-a.X)/(c.X-a.X)*(c.Y-a.Y)
		lo, hi = math.Min(lo, y), math.Max(hi, y)
	}
	return lo, hi
}
