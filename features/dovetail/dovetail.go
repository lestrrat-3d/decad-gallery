// Package dovetail cuts a straight sliding dovetail socket into one host body and adds the matching tail to
// another. Both come from one Params, and the tail and the socket have the same profile at exact nominal size, so the
// halves have no gap. A fit clearance is the caller's to define.
package dovetail

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Defaults and limits in millimeters and degrees.
const (
	DefaultTailWidth = 10.0
	DefaultAngle     = 10.0
	DefaultLength    = 16.0
	DefaultDepth     = 6.0

	// MinAngle and MaxAngle bound the dovetail angle. Below 5 degrees the tail barely resists being pulled apart
	// across the joint. Above 20 degrees the tail's wide end is much wider than its neck, so the tail's corners at
	// the wide end and the socket's lips turn sharper than printed layers hold.
	MinAngle = 5.0
	MaxAngle = 20.0
	// MinTailWidth and MinDepth keep the neck and the tail's height printable.
	MinTailWidth = 3.0
	MinDepth     = 3.0
	// MinLength keeps the joint long enough to guide the slide.
	MinLength = 5.0
	// MinWall is the least host material the joint leaves beside its widest end and beyond its depth.
	MinWall = 0.8
	// Stock is how far a tool reaches past a face it must open or join: the tail's root sinks Stock into its host,
	// and the socket cutter starts Stock before the face and runs Stock past each end.
	Stock = 1.0

	tolerance = 1e-6
)

// Carrier names the host block that carries the tail.
type Carrier int

const (
	// TailOnFirst puts the tail on the first host, which lies on the -N side of the joint frame. The tail rises
	// toward +N into a socket in the second host.
	TailOnFirst Carrier = iota
	// TailOnSecond puts the tail on the second host, which lies on the +N side. The tail hangs toward -N into a
	// socket in the first host.
	TailOnSecond
)

func (c Carrier) String() string {
	switch c {
	case TailOnFirst:
		return "first"
	case TailOnSecond:
		return "second"
	}
	return fmt.Sprintf("Carrier(%d)", int(c))
}

// Params specifies the joint. The tail and the socket are both built from it.
type Params struct {
	// TailWidth is the tail's width at its narrow end, where it leaves its host (the neck).
	TailWidth float64
	// Angle is the dovetail angle in degrees: each flank leans this far from the depth direction.
	Angle float64
	// Length is the joint's length along the slide direction.
	Length float64
	// Depth is how far the tail stands from the mating face, and how deep the socket is.
	Depth float64
	// TailOn selects the host that carries the tail.
	TailOn Carrier
}

// DefaultParams returns a 10 mm neck, 10 degree, 16 mm long, 6 mm deep joint with the tail on the first host.
func DefaultParams() Params {
	return Params{TailWidth: DefaultTailWidth, Angle: DefaultAngle, Length: DefaultLength, Depth: DefaultDepth}
}

// TipWidth returns the tail's width at its wide end, the far end from its host: TailWidth + 2 Depth tan(Angle).
func (p Params) TipWidth() float64 {
	return p.TailWidth + 2*p.Depth*math.Tan(p.Angle*math.Pi/180)
}

// Validate rejects values outside the limits above.
func (p Params) Validate() error {
	if !(p.Angle >= MinAngle && p.Angle <= MaxAngle) {
		return fmt.Errorf("dovetail: Angle must lie in [%g, %g] degrees, got %g", MinAngle, MaxAngle, p.Angle)
	}
	if !(p.TailWidth >= MinTailWidth) {
		return fmt.Errorf("dovetail: TailWidth must be at least %g mm, got %g", MinTailWidth, p.TailWidth)
	}
	if !(p.Depth >= MinDepth) {
		return fmt.Errorf("dovetail: Depth must be at least %g mm, got %g", MinDepth, p.Depth)
	}
	if !(p.Length >= MinLength) {
		return fmt.Errorf("dovetail: Length must be at least %g mm, got %g", MinLength, p.Length)
	}
	if p.TailOn != TailOnFirst && p.TailOn != TailOnSecond {
		return fmt.Errorf("dovetail: TailOn must be TailOnFirst or TailOnSecond, got %d", int(p.TailOn))
	}
	if math.IsInf(p.TailWidth+p.Depth+p.Length, 0) || math.IsNaN(p.TailWidth+p.Depth+p.Length) {
		return fmt.Errorf("dovetail: dimensions must be finite")
	}
	return nil
}

// side returns +1 when the tail points toward +N (tail on the first host) and -1 otherwise.
func (p Params) side() float64 {
	if p.TailOn == TailOnFirst {
		return 1
	}
	return -1
}

// Joint is the result of Apply. First and Second replace the hosts passed in, which Apply retires.
type Joint struct {
	// First and Second are the finished hosts.
	First, Second *decad.Body
	// TailSketch is the profile sketch of the tail. SocketSketch is the profile sketch of the socket cutter.
	TailSketch, SocketSketch *sketch.Sketch
}

// Sketches returns every sketch Apply made, each already verified fully constrained.
func (j *Joint) Sketches() []*sketch.Sketch { return []*sketch.Sketch{j.TailSketch, j.SocketSketch} }

// TailBody returns the host that carries the tail, given the Params that built j.
func (j *Joint) TailBody(p Params) *decad.Body {
	if p.TailOn == TailOnFirst {
		return j.First
	}
	return j.Second
}

// SocketBody returns the host that holds the socket, given the Params that built j.
func (j *Joint) SocketBody(p Params) *decad.Body {
	if p.TailOn == TailOnFirst {
		return j.Second
	}
	return j.First
}

// Apply adds the tail to one host and cuts the socket into the other.
//
// The frame f places the joint. Its origin is the centre of the joint's start on the mating plane, U runs across the
// joint, V runs along the slide direction, and N = U x V points from the first host into the second. The mating plane
// is the frame's plane N = 0. The first host lies on the -N side of it and the second on the +N side. The joint runs
// from V = 0 to V = Length. Both hosts must end flush with those two planes (a through joint), cover the joint's
// widest end plus MinWall each side, and the socket's host must reach Depth + MinWall past the mating plane. The tail's
// host must reach Stock behind it.
//
// Apply retires first and second, as decad.Union and decad.Cut do, and returns the finished bodies.
func Apply(ctx context.Context, doc *decad.Document, p Params, f r3.Frame, first, second *decad.Body) (*Joint, error) {
	if ctx == nil {
		return nil, fmt.Errorf("dovetail: context must not be nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("dovetail: document must not be nil")
	}
	if first == nil || second == nil {
		return nil, fmt.Errorf("dovetail: both hosts must be given")
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if !f.IsValid() {
		return nil, fmt.Errorf("dovetail: frame must be valid")
	}
	if err := p.checkHost(f, first, -1); err != nil {
		return nil, fmt.Errorf("dovetail: first host: %w", err)
	}
	if err := p.checkHost(f, second, 1); err != nil {
		return nil, fmt.Errorf("dovetail: second host: %w", err)
	}
	tailHost, socketHost := first, second
	if p.TailOn == TailOnSecond {
		tailHost, socketHost = second, first
	}

	tail, tailSketch, err := p.tool(ctx, doc, f, 0, p.Length, "tail")
	if err != nil {
		return nil, err
	}
	socket, socketSketch, err := p.tool(ctx, doc, f, -Stock, p.Length+2*Stock, "socket")
	if err != nil {
		return nil, err
	}
	// The tail's wide-end edges are convex corners, so they take the chamfer before the tail joins its host.
	tail, err = chamferEdges(ctx, tail, p.convexEdges(f, p.TipWidth()/2, p.side()*p.Depth), "tail wide end")
	if err != nil {
		return nil, err
	}
	tailed, err := decad.Union(ctx, tailHost, tail)
	if err != nil {
		return nil, fmt.Errorf("dovetail: add tail to its host: %w", err)
	}
	cut, err := decad.Cut(ctx, socketHost, socket)
	if err != nil {
		return nil, fmt.Errorf("dovetail: cut socket into its host: %w", err)
	}
	// The socket's lips are convex corners of its host, so they take the chamfer after the cut. The socket floor and
	// the tail's root are concave corners and stay sharp.
	cut, err = chamferEdges(ctx, cut, p.convexEdges(f, p.TailWidth/2, 0), "socket lips")
	if err != nil {
		return nil, err
	}
	j := &Joint{TailSketch: tailSketch, SocketSketch: socketSketch}
	if p.TailOn == TailOnFirst {
		j.First, j.Second = tailed, cut
	} else {
		j.First, j.Second = cut, tailed
	}
	return j, nil
}

// checkHost requires host to span the joint in the frame. dir is -1 for the first host and +1 for the second.
func (p Params) checkHost(f r3.Frame, host *decad.Body, dir float64) error {
	box, err := host.Bounds()
	if err != nil {
		return fmt.Errorf("read bounds: %w", err)
	}
	tol := box.Bound.Base() + tolerance
	lo := r3.Vec{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	hi := r3.Vec{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				l := f.ToLocal(r3.Vec{X: x, Y: y, Z: z})
				lo = r3.Vec{X: math.Min(lo.X, l.X), Y: math.Min(lo.Y, l.Y), Z: math.Min(lo.Z, l.Z)}
				hi = r3.Vec{X: math.Max(hi.X, l.X), Y: math.Max(hi.Y, l.Y), Z: math.Max(hi.Z, l.Z)}
			}
		}
	}
	if math.Abs(lo.Y) > tol || math.Abs(hi.Y-p.Length) > tol {
		return fmt.Errorf("must span V from 0 to %g, spans %g to %g", p.Length, lo.Y, hi.Y)
	}
	reach := p.TipWidth()/2 + MinWall
	if lo.X > -reach+tol || hi.X < reach-tol {
		return fmt.Errorf("must cover U within +-%g, covers %g to %g", reach, lo.X, hi.X)
	}
	// The host's far face from the mating plane, measured outward from it.
	near, far := hi.Z, -lo.Z
	if dir > 0 {
		near, far = -lo.Z, hi.Z
	}
	if math.Abs(near) > tol {
		return fmt.Errorf("must end at the mating plane, ends %g from it", near)
	}
	need := Stock
	if (dir < 0) == (p.TailOn == TailOnSecond) { // the socket's host
		need = p.Depth + MinWall
	}
	if far < need-tol {
		return fmt.Errorf("must reach %g from the mating plane, reaches %g", need, far)
	}
	return nil
}

// tool builds the cutter or the tail: the joint profile extruded along V from y0 for length.
//
// The profile lies in the plane spanned by U and s*N, with its origin on the mating plane's centre line at V = y0 +
// the frame origin. It is a hexagon: a neck of width TailWidth running Stock into the host behind the mating plane,
// then flanks leaning Angle outward to the wide end Depth away. Both the tail and the socket use it, so they
// are the same shape.
func (p Params) tool(ctx context.Context, doc *decad.Document, f r3.Frame, y0, length float64, what string) (
	*decad.Body, *sketch.Sketch, error) {
	s := p.side()
	origin := f.Origin().Add(f.V().Scale(y0))
	frame, err := r3.NewFrame(origin, f.U(), f.N().Scale(s))
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail: %s frame: %w", what, err)
	}
	w := sketch.NewWorld()
	plane, err := w.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail: %s plane: %w", what, err)
	}
	sk, err := w.CreateSketch(plane)
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail: %s sketch: %w", what, err)
	}
	half, tip := p.TailWidth/2, p.TipWidth()/2
	a := sk.CreatePoint(-half, -Stock)
	b := sk.CreatePoint(half, -Stock)
	c := sk.CreatePoint(half, 0)
	d := sk.CreatePoint(tip, p.Depth)
	e := sk.CreatePoint(-tip, p.Depth)
	g := sk.CreatePoint(-half, 0)
	ab, bc, cd := sk.CreateLine(a, b), sk.CreateLine(b, c), sk.CreateLine(c, d)
	de, eg, ga := sk.CreateLine(d, e), sk.CreateLine(e, g), sk.CreateLine(g, a)
	sk.AddConstraint(
		sketch.NewHorizontal(ab), sketch.NewHorizontal(de), sketch.NewVertical(bc), sketch.NewVertical(ga),
		// The origin is the centre of the neck on the mating plane.
		sketch.NewMidpointOf(sk.Origin(), g, c), sketch.NewHorizontalPoints(g, c),
		sketch.NewDistance(g, c, p.TailWidth),
		sketch.NewVerticalDistance(b, c, Stock),
		sketch.NewVerticalDistance(c, d, p.Depth),
		// The flanks lean outward by Angle from the neck's sides, the right one clockwise and the left one the other way.
		sketch.NewAngle(bc, cd, -p.Angle), sketch.NewAngle(ga, eg, p.Angle),
	)
	if _, err := sk.Solve(ctx); err != nil {
		return nil, nil, fmt.Errorf("dovetail: solve %s sketch: %w", what, err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, sk); err != nil {
		return nil, nil, fmt.Errorf("dovetail: verify %s sketch: %w", what, err)
	}
	var chosen *sketch.Profile
	for _, candidate := range sk.Profiles() {
		if !candidate.Valid || len(candidate.Holes) != 0 {
			continue
		}
		if chosen != nil {
			return nil, nil, fmt.Errorf("dovetail: select %s profile: several valid profiles", what)
		}
		chosen = candidate
	}
	if chosen == nil {
		return nil, nil, fmt.Errorf("dovetail: select %s profile: no valid profile", what)
	}
	// The plane's normal is U x (s N) = -s V, so the profile extrudes toward +V against it when s > 0.
	dir := decad.Against
	if s < 0 {
		dir = decad.Along
	}
	body, err := decad.Extrude(ctx, doc, sk, chosen, decad.Distance{D: units.Millimeters(length), Dir: dir})
	if err != nil {
		return nil, nil, fmt.Errorf("dovetail: extrude %s: %w", what, err)
	}
	return body, sk, nil
}
