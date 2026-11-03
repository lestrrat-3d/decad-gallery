package spinningwheel

import (
	"fmt"
	"math"
	"strings"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) finishJoinery() {
	if b.err != nil {
		return
	}
	b.tilt = false
	b.frameJoinery()
	b.treadleJoinery()
	b.guideSeats()
	b.brakeSeats()
	b.collarScrews()
	b.mountInventory()
}

func (b *builder) partIndex(name string) int {
	if b.err != nil {
		return -1
	}
	for i := range b.a.Parts {
		if b.a.Parts[i].Name == name {
			return i
		}
	}
	b.name = name
	b.fail("find manufactured part", fmt.Errorf("missing required part %q", name))
	return -1
}

// Each tool is fresh stock, never a borrowed final part with a through-passage.
// A single union lets the native evaluator see all cuts for this host together.
func (b *builder) machine(name string, tools ...*decad.Body) {
	i := b.partIndex(name)
	if i < 0 {
		return
	}
	b.name = name
	body := b.cut(b.a.Parts[i].Body, b.union(tools...))
	if b.err != nil {
		return
	}
	if body == nil || !body.IsSolid() {
		b.fail("finish joinery", fmt.Errorf("machining did not return a solid"))
		return
	}
	b.a.Parts[i].Body = body
}

// localStock is a box of l's local coordinates, sketched at level z0.
func (b *builder) localStock(l lift, x0, x1, y0, y1, z0, z1 float64) *decad.Body {
	return b.liftPanel(l, level(0, 0, z0), x0, y0, x1, y1, z1-z0)
}

// headStock is a box of the head lift, its Y range given before HeadShift.
func (b *builder) headStock(x0, x1, y0, y1, z0, z1 float64) *decad.Body {
	return b.localStock(b.headLift(), x0, x1, y0+b.a.State.HeadShift, y1+b.a.State.HeadShift, z0, z1)
}

// headBore is a cylinder of the head lift from origin (Y before HeadShift) along +Z or +Y.
func (b *builder) headBore(radius, length float64, origin, axis r3.Vec) *decad.Body {
	origin.Y += b.a.State.HeadShift
	f := level(origin.X, origin.Y, origin.Z)
	if axis != (r3.Vec{Z: 1}) {
		f = frame(origin, r3.Vec{Z: 1}, r3.Vec{X: 1})
	}
	return b.liftCylinder(b.headLift(), f, radius, length)
}

// maidenStock is a maiden's own section without its bore, the tool that cuts the maiden's foot socket: the
// maiden then lies in the tool that opened the socket. It reaches past the foot's top face by the maiden's
// height, and its other faces are the socket's walls and floor.
func (b *builder) maidenStock(y, length float64) *decad.Body {
	s := b.p.Scale
	return b.liftPanel(b.headLift(), b.headAt(y), -20*s, -115*s, 20*s, 25*s, length)
}

// railStock is the rail's socket tool. Its edges follow the rail's lines (slope -90/520) but run
// 52 past the rail's ends at X -280 and 240, so the tool crosses each leg's end cap instead of sharing
// its plane; the part of the tool inside a leg, and so the cut socket, is unchanged.
func (b *builder) railStock(y float64) *decad.Body {
	s := b.p.Scale
	sk := b.section(false)
	polygon(sk, scaled(s, point2{-332, 474}, point2{292, 366}, point2{292, 401}, point2{-332, 509}))
	return b.extrude(sk, 0, 30*s, frame(r3.Vec{Y: y * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
}

// capStock is a bearing cap's socket tool in the cap's own section plane, so the cap lies in the tool that opened
// its socket: the cap's section widened 10 along X, over the cap's Y span from (y0)*Scale run front more toward -Y
// and back more toward +Y.
func (b *builder) capStock(y0, front, back float64) *decad.Body {
	s := b.p.Scale
	return b.panel(190*s, 535*s, 250*s, 610*s, 30*s+front+back,
		frame(r3.Vec{Y: (y0+30)*s + back}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
}

func (b *builder) frameJoinery() {
	s := b.p.Scale
	// XZ profiles extrude toward -Y, so their origins name the rear faces.
	b.machine("head-leg", b.railStock(-55), b.railStock(85))
	// Each cap socket runs 5 past the leg's top corner on the side away from the rails, which it opens; flush
	// with that corner, the Cut needed Decad's gap shift. The leg has no material there, so the socket is the same.
	b.machine("front-wheel-leg", b.railStock(-55), b.capStock(-55, 0, 5))
	b.machine("rear-wheel-leg", b.railStock(85), b.capStock(25, 5, 0))

	var wedgeTools []*decad.Body
	for _, x := range []float64{-180, -120} {
		wedgeTools = append(wedgeTools,
			b.box(x*s-5.1, x*s+5.1, -70*s-5.1, -70*s+5.1, 525*s-8, 525*s+5))
	}
	b.machine("front-head-wedge", wedgeTools...)

	// The pilots run 5 past the base top face they open, so they never end flush with it.
	var basePilots []*decad.Body
	// mother-upper-bar is machined one seat per pass. Fused, the four foot-insert sockets share their
	// X, Y and Z planes pairwise, and Decad's inside test finds every axis ray ambiguous. The hinge
	// stock and the sockets run 5 past the bar face they open, so no pass leaves a flush face.
	var barPasses [][]*decad.Body
	for _, q := range []float64{-300, -90} {
		for _, interval := range [][2]float64{{0, 15}, {32, 47}} {
			y := (q+(interval[0]+interval[1])/2)*s + b.a.State.HeadShift
			basePilots = append(basePilots, b.cylinder(1.5, 8*s+5,
				axisFrame(r3.Vec{X: -183 * s, Y: y, Z: 532 * s}, r3.Vec{Z: 1})))
		}
		barPasses = append(barPasses, []*decad.Body{
			b.headStock(-169*s, -158*s, (q+16)*s, (q+31)*s, 555*s-5, 558*s),
			// The screw pilot starts 5 below the leaf pocket's ceiling it opens, not flush with it.
			b.liftCylinder(b.headLift(), level(-162*s, q*s+b.a.State.HeadShift+23.5*s, 558*s-5), 1.5, 8*s+5)})
	}
	b.machine("mother-base", basePilots...)
	b.machine("mother-jack-arm", b.headStock(b.boardEdge(), -118*s, -330*s, -20*s, 555*s-5, 570*s+5))
	barPasses = append(barPasses, []*decad.Body{b.headStock(-195*s, -85*s, -80*s, -60*s, 565*s, 580*s)})
	var rearCrossbarTools []*decad.Body
	for i, post := range [][2]float64{{-314, -286}, {-54, -30}} {
		footName := fmt.Sprintf("maiden-foot-%d", i+1)
		b.machine(footName, b.maidenStock([]float64{-286 * s, -30 * s}[i], []float64{28 * s, 24 * s}[i]))
		for _, x := range []float64{-150, -130} {
			y := post[0]*s - 6.2
			barPasses = append(barPasses, []*decad.Body{b.headStock(x*s-5.1, x*s+5.1, y-5.1, y+5.1, 570*s-8, 570*s+5)})
			if i == 1 {
				rearCrossbarTools = append(rearCrossbarTools,
					b.headStock(x*s-5.1, x*s+5.1, y-5.1, y+5.1, 570*s-8, 570*s))
			}
		}
	}
	for _, tools := range barPasses {
		b.machine("mother-upper-bar", tools...)
	}
	rearCrossbarTools = append(rearCrossbarTools,
		b.headStock(-175*s, -105*s, -54*s-12, -30*s+2, 570*s, 585*s),
		b.liftCylinder(b.headLift(), level(-162*s, -90*s+b.a.State.HeadShift+23.5*s, 558*s-5), 1.5, 8*s+5))
	rb := 42*s + 0.6*math.Sqrt(5)
	eyeY, eyeZ0, eyeZ1 := b.routingEyeStem()
	for _, x := range []float64{-140*s - rb, -140*s + rb} {
		rearCrossbarTools = append(rearCrossbarTools,
			b.liftCylinder(b.headLift(), level(x, eyeY, eyeZ0), 0.85, eyeZ1-eyeZ0))
	}
	b.machine("brake-rear-crossbar", rearCrossbarTools...)
	// The rod socket runs 5 past the boss's end face it opens.
	// The bearing bore runs 5 past both boss faces on the cross pin's gimbal lift, the lift of the bearings in it.
	b.machine("footman-head-boss", b.localStock(b.footmanLift(), 460*s-5, 476*s, -10*s, 10*s, -5*s, 5*s),
		b.liftCylinder(b.gimbalLift(), crossAt(32*s, 12*s+5), 7, 24*s+10))

	// The radial spoke blanks seat in the hub and the rim rather than sharing material.
	var hubTools, rimTools []*decad.Body
	for i := range 6 {
		a := math.Remainder(b.p.WheelAngle, 2*math.Pi) + float64(i)*math.Pi/3
		f := shaftFrame(220*s, 6*s, 580*s, a)
		hubTools = append(hubTools, b.panel(25*s, -9*s, 260*s, 9*s, 12*s, f))
		rimTools = append(rimTools, b.panel(25*s, -9*s, 260*s, 9*s, 12*s, f))
	}
	b.machine("wheel-hub", hubTools...)
	b.machine("wheel-rim", rimTools...)
}

// diagonalStock is the diagonal rail's lap tool between local z0 and z1 (scaled); pad0 and pad1 run it that many mm
// further out so it never ends flush with the host face it opens.
func (b *builder) diagonalStock(z0, z1, pad0, pad1 float64) *decad.Body {
	s := b.p.Scale
	f := b.a.State.PedalFrame
	u, _ := (r3.Vec{X: 330, Y: 450}).Normalize()
	v := r3.Vec{X: -u.Y, Y: u.X}
	return b.panel(0, -12*s, math.Hypot(330, 450)*s, 12*s, (z1-z0)*s+pad0+pad1,
		frame(f.ToWorld(r3.Vec{X: -160 * s, Z: z0*s - pad0}),
			f.U().Scale(u.X).Add(f.V().Scale(u.Y)), f.U().Scale(v.X).Add(f.V().Scale(v.Y))))
}

func (b *builder) treadleJoinery() {
	s := b.p.Scale
	f := b.treadleLift()
	b.machine("treadle-heel-rail",
		b.localStock(f, 5*s, 115*s, -30*s, 240*s, 0, 12*s+5),
		b.diagonalStock(-12, 0, 5, 0))
	// The keyed shaft runs along the treadle's local X; radial +X of the key section is local Y.
	b.machine("treadle",
		b.localStock(f, -233*s, 193*s, -12*s, 12*s, -12*s-5, 0),
		b.diagonalStock(-12, 0, 5, 0), b.liftKeyTool(f, heelAt(5*s-5, 0, 0), 4.1, 1.1, 4.9, 110*s+10))
	b.machine("treadle-actuating-rail",
		b.localStock(f, -233*s, 193*s, -12*s, 12*s, 0, 12*s+5),
		b.localStock(f, 5*s, 115*s, -30*s, 240*s, 0, 12*s+5),
		b.localStock(f, 160*s, 220*s, 438*s, 462*s, -12*s-5, 0))
	// The glued rail never touches the heel shaft or its key, so it gets a clearance bore along the shaft.
	b.machine("treadle-actuating-rail", b.liftCylinder(f, heelAt(-290*s, 0, 0), 5.2, 560*s))
	// The half-lap stock runs 5 past the block's side and bottom faces, so it never shares their planes.
	b.machine("treadle-connector-block",
		b.localStock(f, 160*s, 220*s, 438*s-5, 462*s+5, -12*s-5, 0),
		b.liftCylinder(f, level(220*s, 450*s, 3*s), 3.2, 10*s))
	// The flexure bore runs along the rod's local X from 19s.
	b.machine("footman", b.liftCylinder(b.footmanLift(), frame(r3.Vec{X: 19 * s}, r3.Vec{Y: 1}, r3.Vec{Z: 1}), 3.2, 14*s))
	if i := b.partIndex("footman"); i >= 0 && b.err == nil {
		b.name = "footman"
		b.a.Parts[i].Body = b.refine(b.a.Parts[i].Body, 0.01)
	}
}

func (b *builder) guideSeats() {
	s := b.p.Scale
	fl := b.flyerLift()
	p := r3.Vec{X: 58 * s, Z: 22 * s}
	r := 31 * s
	q := p.Dot(p)
	tangent := p.Scale(r * r / q).Sub(r3.Vec{X: -p.Z, Z: p.X}.Scale(r * math.Sqrt(q-r*r) / q))
	outgoing, _ := tangent.Sub(p).Normalize()
	for side, x := range []float64{-58, 58} {
		var tools []*decad.Body
		for _, y := range []float64{-190, -172, -160, -142, -124, -106} {
			axis := r3.Vec{Y: 1}
			if side == 1 && y == -160 {
				axis, _ = axis.Add(outgoing).Normalize()
			}
			down := r3.Vec{Z: -1}
			e, _ := down.Sub(axis.Scale(down.Dot(axis))).Normalize()
			attachment := (r3.Vec{X: x * s, Y: y * s, Z: 22 * s}).Add(e.Scale(4))
			// The seat is the stem's own stock run 4s deeper: it runs along the flyer's +Z, local +Y, from flyer
			// point (x, y, 6s) to the stem's top, so the stem lies in the tool that opened its seat.
			tools = append(tools, b.liftCylinder(fl, frame(r3.Vec{X: attachment.X, Y: 6 * s, Z: -attachment.Y},
				r3.Vec{Z: 1}, r3.Vec{X: 1}), hookStemRadius, attachment.Z-6*s))
		}
		b.machine(fmt.Sprintf("flyer-arm-%d", side+1), tools...)
	}
	var crossbarTools []*decad.Body
	// Each socket is its arm's own stock, run 5 past the crossbar's end and its far face, so the arm lies in
	// the tool that opened its socket and no tool face is flush with a crossbar face.
	for _, xr := range [][2]float64{{-74*s - 5, -56 * s}, {56 * s, 74*s + 5}} {
		crossbarTools = append(crossbarTools,
			b.liftPanel(fl, level(0, 0, 85*s), xr[0], -10*s, xr[1], 10*s, 135*s+5))
	}
	// The front guide's seat is its stalk's own stock run 1s deeper: along world +Y, local -Z, from the stalk's
	// start at flyer point (58s, -224s, 12s) to -214s, so the stalk lies in the tool that opened its seat.
	crossbarTools = append(crossbarTools, b.liftCylinder(fl, frame(r3.Vec{X: 58 * s, Y: 12 * s, Z: 224 * s},
		r3.Vec{Y: 1}, r3.Vec{X: 1}), 1, -214*s-(-224*s)))
	b.machine("flyer-crossbar", crossbarTools...)
}

func (b *builder) brakeSeats() {
	s := b.p.Scale
	r := 42*s + 0.6*math.Sqrt(5)
	x0, x1 := -140*s-r, -140*s+r
	yf := -338 * s
	// The left anchor terminates six stock millimetres inside the control rail.
	anchor0, anchor1 := b.anchorStem()
	var controlTools []*decad.Body
	// The insert's socket is sketched in the insert's own plane orientation, 1 in front of the rail's front face.
	controlTools = append(controlTools,
		b.liftPanel(b.headLift(), frame(r3.Vec{X: x1, Y: yf + b.a.State.HeadShift - 1, Z: 600 * s}, r3.Vec{X: 1}, r3.Vec{Z: -1}),
			-4.1, -4.1, 4.1, 4.1, 5),
		b.liftCylinder(b.headLift(), frame(r3.Vec{X: x0, Y: anchor0, Z: 600 * s}, r3.Vec{Z: 1}, r3.Vec{X: 1}),
			0.85, anchor1-anchor0))
	// The mount pilots run 5 past the rail's front face they open; starting flush with it, the rounded tool
	// left a paper-thin disk of the face as a separate piece at Scale 1.2.
	for _, x := range []float64{x1 - 6, x1 + 6} {
		controlTools = append(controlTools,
			b.headBore(1, 7+5, r3.Vec{X: x, Y: yf - 5, Z: 600 * s}, r3.Vec{Y: 1}))
	}
	// The front maiden foot's bolt heads (stock height 4.2 above 585*Scale) reach the rail's bottom face at
	// 590*Scale below Scale 0.84. There the rail gets a stock radius6 head relief, 5 below its bottom face up to
	// the head tops; above that scale the heads stay clear and the rail is uncut.
	if top := 585*s + 4.2; top > 590*s {
		for _, x := range []float64{-150 * s, -130 * s} {
			controlTools = append(controlTools,
				b.headBore(6, top-(590*s-5), r3.Vec{X: x, Y: -314*s - 6.2, Z: 590*s - 5}, r3.Vec{Z: 1}))
		}
	}
	b.machine("brake-control-rail", controlTools...)
}

type collarSeat struct {
	name           string
	centre, radial r3.Vec
	inner, outer   float64
	// hole, when set, is the tapped hole's start on the collar's axis in l's local coordinates, its
	// normal the radial direction; the setscrew then starts on the same frame, raised by inner.
	l    lift
	hole *r3.Frame
}

func (b *builder) collarScrews() {
	s := b.p.Scale
	var seats []collarSeat
	wl := b.wheelLift()
	// The wheel's setscrews point along local -X, from the axle away from the crank pin.
	wheel := func(name string, x, y, inner, outer float64) {
		hole := frame(r3.Vec{X: x, Z: -y}, r3.Vec{Z: 1}, r3.Vec{Y: 1})
		seats = append(seats, collarSeat{name: name, inner: inner, outer: outer, l: wl, hole: &hole})
	}
	wheel("wheel-axle-front-collar", 0, -55*s-1.5, 4, 6)
	wheel("wheel-axle-rear-collar", 0, 55*s+1.5, 4, 6)
	wheel("wheel-crank-front-collar", 0, 78*s-1.7, 4, 6)
	wheel("wheel-crank-rear-collar", 0, 86*s+1.7, 4, 6)
	wheel("wheel-crank-pin-collar-1", 25*s, 78*s-1.7, 2.5, 4.5)
	wheel("wheel-crank-pin-collar-2", 25*s, 105*s+4.2, 2.5, 4.5)
	tl := b.treadleLift()
	for i, x := range []float64{-260*s - 5.2, 220*s + 5.2} {
		// The heel setscrews point along the treadle's local -Y, away from the toe.
		hole := frame(r3.Vec{X: x}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		seats = append(seats, collarSeat{name: fmt.Sprintf("heel-collar-%d", i+1), inner: 4, outer: 6, l: tl, hole: &hole})
	}
	gl := b.gimbalLift()
	for i, v := range []float64{-27*s - 1.7, 27*s + 1.7} {
		// The cross-pin setscrews point along the gimbal's local -X, back toward the crank pin.
		hole := frame(r3.Vec{X: 32 * s, Y: v}, r3.Vec{Z: 1}, r3.Vec{Y: 1})
		seats = append(seats, collarSeat{name: fmt.Sprintf("gimbal-cross-pin-collar-%d", i+1),
			inner: 2.5, outer: 4.5, l: gl, hole: &hole})
	}
	// The flyer's setscrews turn with the flyer, so their holes ride the flyer lift along its local -X.
	fl := b.flyerLift()
	for _, item := range []struct {
		name         string
		y, in, outer float64
	}{
		{"rear-flyer-front-collar", -54*s - 1.7, 4, 6},
		{"rear-flyer-rear-collar", -30*s + 1.7, 4, 6},
		{"front-sleeve-retainer", -314*s - 2.2, 8, 10},
		{"whorl-front-collar", -282*s - 2.2, 8, 10},
		{"whorl-rear-collar", -250*s + 2.2, 8, 10},
		{"crossbar-rear-collar", -210*s + 2.2, 8, 10},
		{"bobbin-rear-removable-collar", -74*s + 8.7, 4, 6},
	} {
		hole := frame(r3.Vec{Z: -item.y}, r3.Vec{Z: 1}, r3.Vec{Y: 1})
		seats = append(seats, collarSeat{name: item.name, inner: item.in, outer: item.outer, l: fl, hole: &hole})
	}
	for _, seat := range seats {
		i := b.partIndex(seat.name)
		if i < 0 {
			return
		}
		motion := b.a.Parts[i].Motion
		screw := math.Sqrt(seat.outer*seat.outer-1) - seat.inner
		// The tapped hole's tool runs 5 past the collar's outer surface; ending at that radius, its end
		// disk would lie tangent to the outer cylinder.
		if seat.hole != nil {
			b.machine(seat.name, b.liftCylinder(seat.l, *seat.hole, 1, seat.outer+5))
			b.part(seat.name+"-setscrew", "fastener", motion, func() *decad.Body {
				return b.liftCylinder(seat.l, raise(*seat.hole, seat.inner), 1, screw)
			})
			continue
		}
		b.machine(seat.name, b.cylinder(1, seat.outer+5,
			axisFrame(seat.centre, seat.radial)))
		b.part(seat.name+"-setscrew", "fastener", motion, func() *decad.Body {
			return b.cylinder(1, screw, axisFrame(seat.centre.Add(seat.radial.Scale(seat.inner)), seat.radial))
		})
	}
}

func (b *builder) mountInventory() {
	if b.err != nil {
		return
	}
	defined := make(map[string]Mount)
	add := func(name, kind string, supports, fasteners []string, clearance float64) {
		if _, exists := defined[name]; exists {
			b.name = name
			b.fail("record mount", fmt.Errorf("duplicate mount definition"))
			return
		}
		defined[name] = Mount{Part: name, Kind: kind, Supports: supports, Fasteners: fasteners, Clearance: clearance}
	}
	refs := func(names ...string) []string { return names }
	for _, name := range []string{"head-leg", "front-wheel-leg", "rear-wheel-leg"} {
		add(name, "ground", nil, nil, 0)
	}
	add("front-bench-rail", "glued-socket", refs("head-leg", "front-wheel-leg"), nil, 0)
	add("rear-bench-rail", "glued-socket", refs("head-leg", "rear-wheel-leg"), nil, 0)
	add("front-head-wedge", "glued-face", refs("front-bench-rail"), nil, 0)
	add("rear-head-wedge", "glued-face", refs("rear-bench-rail"), nil, 0)
	add("front-wheel-bearing-cap", "glued-socket", refs("front-wheel-leg"), nil, 0)
	add("rear-wheel-bearing-cap", "glued-socket", refs("rear-wheel-leg"), nil, 0)
	for i, cap := range []string{"front-wheel-bearing-cap", "rear-wheel-bearing-cap"} {
		add(fmt.Sprintf("wheel-bearing-%d", i+1), "press-fit", refs(cap), nil, 0)
	}
	for i, leg := range []string{"head-leg", "front-wheel-leg"} {
		add(fmt.Sprintf("heel-bearing-%d", i+1), "press-fit", refs(leg), nil, 0)
	}
	add("mother-base", "bolted-sliding-alignment", refs("front-head-wedge", "rear-head-wedge"),
		refs("mother-alignment-clamp-1", "mother-alignment-clamp-2", "mother-alignment-insert-1", "mother-alignment-insert-2"), 0.3)
	for i := 1; i <= 2; i++ {
		insert := fmt.Sprintf("mother-alignment-insert-%d", i)
		bolt := fmt.Sprintf("mother-alignment-clamp-%d", i)
		add(insert, "captured-square-insert", refs("front-head-wedge", "mother-base"), refs(bolt), 0.1)
		add(bolt, "threaded-clamp", refs(insert, "mother-base"), nil, 0)
	}
	var hingeLeaves, movingScrews []string
	for n := 1; n <= 2; n++ {
		prefix := fmt.Sprintf("mother-hinge-%d-", n)
		pin, nut := prefix+"pin", prefix+"nut"
		for j := 1; j <= 2; j++ {
			bracket := fmt.Sprintf("%sfixed-bracket-%d", prefix, j)
			screw := fmt.Sprintf("%sfixed-mount-%d", prefix, j)
			add(bracket, "screwed-hinge-leaf", refs("mother-base"), refs(screw), 0.1)
			add(screw, "wood-screw-pilot", refs("mother-base", bracket), nil, 0)
		}
		leaf, screw := prefix+"moving-leaf", prefix+"moving-mount"
		add(leaf, "screwed-revolute-leaf", refs("mother-upper-bar", prefix+"fixed-bracket-1", prefix+"fixed-bracket-2"),
			refs(screw, pin, nut), 0.2)
		add(screw, "wood-screw-pilot", refs("mother-upper-bar", leaf), nil, 0)
		add(pin, "headed-hinge-pin", refs(prefix+"fixed-bracket-1", prefix+"fixed-bracket-2"), refs(nut), 0.2)
		add(nut, "threaded-pin-retention", refs(pin), nil, 0)
		hingeLeaves = append(hingeLeaves, leaf)
		movingScrews = append(movingScrews, screw)
	}
	add("mother-upper-bar", "screwed-hinge-and-jack-support", append(refs("mother-jack-screw"), hingeLeaves...), movingScrews, 0)
	add("mother-jack-arm", "glued-socket", refs("mother-upper-bar"), nil, 0)
	add("mother-jack-square-insert", "captured-square-insert", refs("mother-jack-arm"), nil, 0.1)
	add("mother-jack-screw", "threaded-jack-and-point-contact", refs("mother-jack-square-insert", "mother-base"), nil, 0)
	add("mother-jack-knob", "bonded-keyed", refs("mother-jack-screw"), nil, 0.1)
	for i, post := range []string{"front-maiden", "rear-maiden"} {
		foot := fmt.Sprintf("maiden-foot-%d", i+1)
		var bolts []string
		for j := 1; j <= 2; j++ {
			bolt := fmt.Sprintf("%s-bolt-%d", foot, j)
			insert := fmt.Sprintf("%s-insert-%d", foot, j)
			bolts = append(bolts, bolt)
			add(insert, "captured-square-insert", refs("mother-upper-bar", foot), refs(bolt), 0.1)
			add(bolt, "threaded-foot-clamp", refs(insert, foot), nil, 0)
		}
		add(foot, "removable-bolted-foot", refs("mother-upper-bar"), bolts, 0.3)
		add(post, "glued-tenon", refs(foot), nil, 0)
	}

	add("wheel-axle", "bearing-journal", refs("wheel-bearing-1", "wheel-bearing-2"),
		refs("wheel-axle-front-collar", "wheel-axle-rear-collar"), 0)
	add("wheel-hub", "keyed-axially-captured", refs("wheel-axle", "wheel-hub-key", "wheel-hub-front-spacer", "wheel-hub-rear-spacer"),
		refs("wheel-axle-front-collar", "wheel-axle-rear-collar"), 0.1)
	add("wheel-crank", "keyed-axially-captured", refs("wheel-axle", "wheel-crank-key"),
		refs("wheel-crank-front-collar", "wheel-crank-rear-collar"), 0.1)
	add("wheel-hub-key", "removable-parallel-key", refs("wheel-axle", "wheel-hub"), nil, 0.1)
	add("wheel-crank-key", "removable-parallel-key", refs("wheel-axle", "wheel-crank"), nil, 0.1)
	add("wheel-rim", "glued-spoke-sockets", refs("wheel-spoke-1", "wheel-spoke-2", "wheel-spoke-3", "wheel-spoke-4", "wheel-spoke-5", "wheel-spoke-6"), nil, 0)
	for i := 1; i <= 6; i++ {
		add(fmt.Sprintf("wheel-spoke-%d", i), "glued-hub-and-rim-sockets", refs("wheel-hub", "wheel-rim"), nil, 0)
	}
	for _, pair := range [][2]string{
		{"wheel-axle-front-thrust-spacer", "wheel-bearing-1"}, {"wheel-axle-rear-thrust-spacer", "wheel-bearing-2"},
		{"wheel-hub-front-spacer", "wheel-bearing-1"}, {"wheel-hub-rear-spacer", "wheel-bearing-2"},
	} {
		add(pair[0], "captive-thrust-spacer", refs("wheel-axle", pair[1]), nil, 0)
	}
	add("wheel-crank-pin", "retained-pin", refs("wheel-crank"), refs("wheel-crank-pin-collar-1", "wheel-crank-pin-collar-2"), 0.1)
	add("wheel-crank-pin-spacer", "captive-inner-race-spacer", refs("wheel-crank-pin", "wheel-crank", "gimbal-crank-605"), nil, 0)
	add("gimbal-crank-605", "press-fit-and-journal", refs("gimbal-bridge", "wheel-crank-pin"),
		refs("gimbal-crank-retainer-1", "gimbal-crank-retainer-2"), 0)
	add("gimbal-bridge", "bearing-clevis", refs("gimbal-crank-605", "wheel-crank-pin"),
		refs("gimbal-crank-retainer-1", "gimbal-crank-retainer-2"), 0)
	for i := 1; i <= 2; i++ {
		plate := fmt.Sprintf("gimbal-crank-retainer-%d", i)
		bolt, nut := fmt.Sprintf("gimbal-retainer-bolt-%d", i), fmt.Sprintf("gimbal-retainer-nut-%d", i)
		add(plate, "bolted-bearing-end-plate", refs("gimbal-bridge"), refs("gimbal-retainer-bolt-1", "gimbal-retainer-bolt-2"), 0)
		add(bolt, "headed-tie-bolt", refs("gimbal-bridge", "gimbal-crank-retainer-1", "gimbal-crank-retainer-2"), refs(nut), 0.1)
		add(nut, "threaded-tie-nut", refs(bolt), nil, 0)
	}
	add("gimbal-cross-pin", "retained-clevis-pin", refs("gimbal-bridge"),
		refs("gimbal-cross-pin-collar-1", "gimbal-cross-pin-collar-2"), 0.1)
	for i := 1; i <= 2; i++ {
		bearing := fmt.Sprintf("footman-cross-605-%d", i)
		clearance := 0.0
		if i == 1 {
			clearance = crossGap
		}
		add(bearing, "press-fit-and-journal", refs("footman-head-boss", "gimbal-cross-pin"),
			refs("footman-cross-retainer-1", "footman-cross-retainer-2"), clearance)
		add(fmt.Sprintf("gimbal-cross-spacer-%d", i), "captive-inner-race-spacer", refs("gimbal-cross-pin", "gimbal-bridge", bearing), nil, 0)
		add(fmt.Sprintf("footman-cross-retainer-%d", i), "bolted-bearing-end-plate", refs("footman-head-boss"), refs("footman-retainer-bolt"), 0)
	}
	add("footman-cross-spacer", "captive-inner-race-spacer", refs("gimbal-cross-pin", "footman-cross-605-1", "footman-cross-605-2"), nil, 0)
	add("footman-head-boss", "glued-rod-socket-and-cross-journal", refs("footman", "gimbal-cross-pin"), refs("footman-retainer-bolt"), 0)
	add("footman", "glued-boss-socket-and-clamped-flexure", refs("footman-head-boss", "treadle-flex-connector"), refs("footman-upper-clamp-pin", "footman-upper-clamp-nut"), 0.2)
	add("footman-retainer-bolt", "headed-tie-bolt", refs("footman-head-boss", "footman", "footman-cross-retainer-1", "footman-cross-retainer-2"), refs("footman-retainer-nut"), 0.1)
	add("footman-retainer-nut", "threaded-tie-nut", refs("footman-retainer-bolt"), nil, 0)
	add("heel-pin", "bearing-journal", refs("heel-bearing-1", "heel-bearing-2"), refs("heel-collar-1", "heel-collar-2"), 0)
	add("heel-key", "removable-parallel-key", refs("heel-pin", "treadle-heel-rail", "treadle"), nil, 0.1)
	add("treadle-heel-rail", "keyed-and-glued-half-lap", refs("heel-pin", "heel-key", "treadle"), refs("heel-collar-1", "heel-collar-2"), 0.1)
	add("treadle", "keyed-and-glued-half-laps", refs("heel-pin", "heel-key", "treadle-heel-rail", "treadle-actuating-rail"), nil, 0.1)
	add("treadle-actuating-rail", "glued-half-laps", refs("treadle", "treadle-heel-rail", "treadle-rear-crossarm"), nil, 0)
	add("treadle-rear-crossarm", "glued-half-laps", refs("treadle-actuating-rail", "treadle-connector-block"), nil, 0)
	add("treadle-connector-block", "glued-half-lap", refs("treadle-rear-crossarm"), nil, 0)
	for i := 1; i <= 2; i++ {
		add(fmt.Sprintf("heel-spacer-%d", i), "captive-inner-race-spacer", refs("heel-pin", fmt.Sprintf("heel-bearing-%d", i), "treadle-heel-rail"), nil, 0.2)
	}
	add("treadle-flex-connector", "clamped-compliant-sockets", refs("treadle-connector-block", "footman"),
		refs("treadle-lower-clamp-pin", "treadle-lower-clamp-nut", "footman-upper-clamp-pin", "footman-upper-clamp-nut"), 0.2)
	for _, clamp := range [][2]string{{"treadle-lower-clamp", "treadle-connector-block"}, {"footman-upper-clamp", "footman"}} {
		add(clamp[0]+"-pin", "headed-cross-clamp", refs(clamp[1], "treadle-flex-connector"), refs(clamp[0]+"-nut"), 0.2)
		add(clamp[0]+"-nut", "threaded-cross-clamp-nut", refs(clamp[0]+"-pin", clamp[1]), nil, 0)
	}
	for _, name := range []string{"drive-band-wheel-wrap", "drive-band-flyer-wrap", "drive-band-span-1", "drive-band-span-2"} {
		add(name, "single-drive-band-tangent-cap-seams", refs("wheel-rim", "flyer-drive-whorl"), nil, bandGap)
	}

	add("front-maiden-bush", "press-fit-sleeve", refs("front-maiden"), refs("front-sleeve-retainer"), 0)
	add("rear-flyer-bearing", "press-fit-bearing", refs("rear-maiden"), refs("rear-flyer-front-collar", "rear-flyer-rear-collar"), 0)
	add("flyer-shaft", "sleeve-and-bearing-journals", refs("front-maiden-bush", "rear-flyer-bearing"),
		refs("front-sleeve-retainer", "rear-flyer-front-collar", "rear-flyer-rear-collar"), 0.1)
	add("flyer-whorl-key", "removable-parallel-key", refs("flyer-shaft", "flyer-drive-whorl"), nil, 0.1)
	add("flyer-crossbar-key", "removable-parallel-key", refs("flyer-shaft", "flyer-crossbar"), nil, 0.1)
	add("flyer-drive-whorl", "keyed-axially-captured", refs("flyer-shaft", "flyer-whorl-key"), refs("whorl-front-collar", "whorl-rear-collar"), 0.1)
	add("flyer-crossbar", "keyed-shoulder-and-rear-collar", refs("flyer-shaft", "flyer-crossbar-key"), refs("crossbar-rear-collar"), 0.1)
	for i := 1; i <= 2; i++ {
		add(fmt.Sprintf("flyer-arm-%d", i), "glued-crossbar-socket", refs("flyer-crossbar"), nil, 0)
	}
	for _, pair := range [][2]string{{"rear-flyer-front-thrust-spacer", "rear-flyer-bearing"}, {"rear-flyer-rear-thrust-spacer", "rear-flyer-bearing"},
		{"bobbin-front-thrust-sleeve", "bobbin-front-bearing"}} {
		add(pair[0], "captive-inner-race-spacer", refs("flyer-shaft", pair[1]), nil, 0.2)
	}
	add("bobbin-inner-race-sleeve", "captive-inner-race-spacer", refs("flyer-shaft", "bobbin-front-bearing", "bobbin-rear-bearing"), nil, 0)
	for _, name := range []string{"bobbin-front-bearing", "bobbin-rear-bearing"} {
		add(name, "press-fit-independent-journal", refs("bobbin", "flyer-shaft"), refs("bobbin-rear-removable-collar"), 0)
	}
	add("bobbin", "independent-bearing-journals", refs("bobbin-front-bearing", "bobbin-rear-bearing", "flyer-shaft"), refs("bobbin-rear-removable-collar"), 0.2)
	add("bobbin-yarn-pack", "wound-pack", refs("bobbin"), nil, 0)
	add("bobbin-witness-bar", "bonded-curved-label", refs("bobbin-yarn-pack"), nil, 0)
	for i := 1; i <= 12; i++ {
		arm := "flyer-arm-1"
		if i > 6 {
			arm = "flyer-arm-2"
		}
		add(fmt.Sprintf("yarn-hook-%d", i), "bonded-thread-envelope-seat", refs(arm), nil, 0)
	}
	add("yarn-front-guide", "bonded-thread-envelope-stalk-seat", refs("flyer-crossbar"), nil, 0)
	add("yarn-leader", "process-yarn-with-free-input-end", refs("flyer-shaft", "yarn-front-guide", "yarn-hook-9", "bobbin-yarn-pack"), nil, 0)

	add("brake-rear-crossbar", "glued-top-seat", refs("mother-upper-bar"), nil, 0)
	add("brake-front-upright", "glued-face", refs("mother-jack-arm"), nil, 0)
	add("brake-control-rail", "glued-face", refs("brake-front-upright"), nil, 0)
	add("brake-insert", "captured-square-insert", refs("brake-control-rail"), refs("brake-insert-retaining-plate"), 0.1)
	add("brake-insert-retaining-plate", "screwed-insert-retainer", refs("brake-control-rail"), refs("brake-insert-mount-1", "brake-insert-mount-2"), 0)
	for i := 1; i <= 2; i++ {
		add(fmt.Sprintf("brake-insert-mount-%d", i), "wood-screw-pilot", refs("brake-control-rail", "brake-insert-retaining-plate"), nil, 0)
	}
	add("brake-left-anchor", "bonded-blind-stem-seat", refs("brake-control-rail"), nil, 0.05)
	add("brake-takeup-screw", "threaded-adjuster-and-integral-clevis", refs("brake-insert"), nil, 0)
	add("brake-takeup-knob", "bonded-keyed", refs("brake-takeup-screw"), nil, 0.1)
	for _, side := range []string{"left", "right"} {
		clevis := "brake-left-anchor"
		if side == "right" {
			clevis = "brake-takeup-screw"
		}
		pin := "brake-front-pin-" + side
		add(pin, "peened-clevis-rivet", refs(clevis), nil, 0.1)
		add("brake-spring-"+side, "captive-front-eye-and-clevis-pin", refs(clevis, "brake-band"), refs(pin), 0.2)
		add("brake-routing-eye-"+side, "bonded-blind-stem-seat", refs("brake-rear-crossbar"), nil, 0.05)
	}
	add("brake-band", "one-cord-with-captive-terminal-eyes", refs("brake-spring-left", "brake-spring-right", "brake-routing-eye-left", "brake-routing-eye-right", "bobbin"), nil, 0)

	for _, collar := range []struct {
		name, shaft string
	}{
		{"wheel-axle-front-collar", "wheel-axle"}, {"wheel-axle-rear-collar", "wheel-axle"},
		{"wheel-crank-front-collar", "wheel-axle"}, {"wheel-crank-rear-collar", "wheel-axle"},
		{"wheel-crank-pin-collar-1", "wheel-crank-pin"}, {"wheel-crank-pin-collar-2", "wheel-crank-pin"},
		{"gimbal-cross-pin-collar-1", "gimbal-cross-pin"}, {"gimbal-cross-pin-collar-2", "gimbal-cross-pin"},
		{"heel-collar-1", "heel-pin"}, {"heel-collar-2", "heel-pin"},
		{"rear-flyer-front-collar", "flyer-shaft"}, {"rear-flyer-rear-collar", "flyer-shaft"},
		{"front-sleeve-retainer", "flyer-shaft"}, {"whorl-front-collar", "flyer-shaft"},
		{"whorl-rear-collar", "flyer-shaft"}, {"crossbar-rear-collar", "flyer-shaft"},
		{"bobbin-rear-removable-collar", "flyer-shaft"},
	} {
		add(collar.name, "radial-set-screw-collar", refs(collar.shaft), refs(collar.name+"-setscrew"), 0)
		add(collar.name+"-setscrew", "radial-thread-envelope", refs(collar.name, collar.shaft), nil, 0)
	}

	live := make(map[string]struct{}, len(b.a.Parts))
	for _, part := range b.a.Parts {
		live[part.Name] = struct{}{}
	}
	for _, part := range b.a.Parts {
		m, ok := defined[part.Name]
		if !ok {
			b.name = part.Name
			b.fail("record mount", fmt.Errorf("no specified physical mount"))
			return
		}
		for _, reference := range append(append([]string{}, m.Supports...), m.Fasteners...) {
			if _, exists := live[reference]; !exists {
				b.name = part.Name
				b.fail("record mount", fmt.Errorf("missing referenced part %q", reference))
				return
			}
		}
		b.a.Mounts = append(b.a.Mounts, m)
	}
	// Supports form cycles at captured joints. Propagate the known ground paths
	// instead of treating a repeated node as evidence of ground support.
	grounded := make(map[string]struct{})
	for _, m := range b.a.Mounts {
		if m.Kind == "ground" {
			grounded[m.Part] = struct{}{}
		}
	}
	for changed := true; changed; {
		changed = false
		for _, m := range b.a.Mounts {
			if _, exists := grounded[m.Part]; exists {
				continue
			}
			for _, support := range m.Supports {
				if _, exists := grounded[support]; exists {
					grounded[m.Part] = struct{}{}
					changed = true
					break
				}
			}
		}
	}
	for _, m := range b.a.Mounts {
		if _, exists := grounded[m.Part]; !exists {
			b.name = m.Part
			b.fail("verify mount support", fmt.Errorf("no support path to a ground leg: %s", strings.Join(m.Supports, ", ")))
			return
		}
	}
}
