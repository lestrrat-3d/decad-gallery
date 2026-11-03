package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

func (b *builder) frame() {
	s := b.p.Scale
	b.tilt = false
	legs := []struct {
		name    string
		x0, x1  float64
		outline []point2
		heel    bool
	}{
		{"head-leg", -280, -240, []point2{{-220, 0}, {-180, 0}, {120, 500}, {-100, 500}}, true},
		{"front-wheel-leg", 200, 240, []point2{{-230, 0}, {-190, 0}, {-25, 550}, {-65, 550}}, true},
		{"rear-wheel-leg", 200, 240, []point2{{190, 0}, {230, 0}, {65, 550}, {25, 550}}, false},
	}
	for _, leg := range legs {
		b.part(leg.name, "frame", "fixed", func() *decad.Body {
			section := b.section(false)
			polygon(section, scaled(s, leg.outline...))
			holes := 0
			// The heel bore is drawn at stock radius9; the bearing's radius11 seat is cut from the inboard side
			// (frameJoinery), leaving a shoulder on the outer race's outboard face.
			if leg.heel {
				circle(section, -190*s, 60*s, 9)
				holes = 1
			}
			return b.extrude(section, holes, (leg.x1-leg.x0)*s,
				frame(r3.Vec{X: leg.x0 * s}, r3.Vec{Y: 1}, r3.Vec{Z: 1}))
		})
	}
	for i, y0 := range []float64{-85, 55} {
		name := []string{"front-bench-rail", "rear-bench-rail"}[i]
		b.part(name, "frame", "fixed", func() *decad.Body {
			section := b.section(false)
			polygon(section, scaled(s, point2{-280, 465}, point2{240, 375}, point2{240, 410}, point2{-280, 500}))
			return b.extrude(section, 0, 30*s,
				frame(r3.Vec{Y: (y0 + 30) * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
		})
		b.part([]string{"front-head-wedge", "rear-head-wedge"}[i], "frame", "fixed", func() *decad.Body {
			// The wedge is the box above the bench rail's top line: the box minus a prism of the rail's own
			// section in the rail's own frame, so Verify can prove the pair apart from the Cut history.
			section := b.section(false)
			polygon(section, scaled(s, point2{-195, 455}, point2{-105, 455}, point2{-105, 525}, point2{-195, 525}))
			blank := b.extrude(section, 0, 30*s,
				frame(r3.Vec{Y: (y0 + 30) * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
			railSection := b.section(false)
			polygon(railSection, scaled(s, point2{-280, 465}, point2{240, 375}, point2{240, 410}, point2{-280, 500}))
			railTool := b.extrude(railSection, 0, 30*s,
				frame(r3.Vec{Y: (y0 + 30) * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
			body := b.cut(blank, railTool)
			if i != 0 {
				return body
			}
			var tools []*decad.Body
			for _, x := range []float64{-180, -120} {
				tools = append(tools, b.cylinder(3.3, 100*s,
					frame(r3.Vec{X: x * s, Y: -70 * s, Z: 450 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1})))
			}
			return b.cut(body, b.union(tools...))
		})
	}
	// The round wheel-axis parts ride the wheel lift; each cap's bearing bore is cut by a tool on that
	// lift, running 5 past both cap faces, so the bore and the bearing share one exact placement.
	wl := b.wheelLift()
	for i, y0 := range []float64{-55, 25} {
		name := []string{"front-wheel-bearing-cap", "rear-wheel-bearing-cap"}[i]
		b.part(name, "frame", "fixed", func() *decad.Body {
			body := b.panel(200*s, 535*s, 240*s, 610*s, 30*s,
				frame(r3.Vec{Y: (y0 + 30) * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
			// The bore is stepped: the bearing's stock radius11 seat runs from its outboard face out past the
			// cap's inboard face, and a radius9 bore through the rest leaves a shoulder on the outer race.
			seat0, seat1 := -45*s, (y0+30)*s+5
			if i == 1 {
				seat0, seat1 = 25*s-5, 45*s
			}
			return b.cut(body, b.union(
				b.liftCylinder(wl, wheelAt(0, 0, (y0+30)*s+5), 9, 30*s+10),
				b.liftCylinder(wl, wheelAt(0, 0, seat1), 11, seat1-seat0)))
		})
	}
	for i, y0 := range []float64{-45 * s, 45*s - 7} {
		b.part(fmt.Sprintf("wheel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.liftAnnulus(wl, wheelAt(0, 0, y0+7), 4, 11, 7)
		})
		// A stock retaining ring in the seat holds the outer race's inboard face against the shoulder.
		ring := [2]float64{y0 + 7, y0 + 8}
		if i == 1 {
			ring = [2]float64{y0 - 1, y0}
		}
		b.part(fmt.Sprintf("wheel-bearing-%d-ring", i+1), "retainer", "fixed", func() *decad.Body {
			return b.liftAnnulus(wl, wheelAt(0, 0, ring[1]), 9, 11, ring[1]-ring[0])
		})
	}
	hl := b.heelLift()
	for i, x := range []float64{-260, 220} {
		b.part(fmt.Sprintf("heel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.liftAnnulus(hl, level(0, 0, x*s-3.5), 4, 11, 7)
		})
		// A stock retaining ring in the seat holds the outer race's inboard face.
		ring := x*s + 3.5
		if i == 1 {
			ring = x*s - 3.5 - 1
		}
		b.part(fmt.Sprintf("heel-bearing-%d-ring", i+1), "retainer", "fixed", func() *decad.Body {
			return b.liftAnnulus(hl, level(0, 0, ring), 9, 11, 1)
		})
	}
	for i, interval := range [][2]float64{{-55*s - 3, -55 * s}, {55 * s, 55*s + 3}} {
		b.part([]string{"wheel-axle-front-collar", "wheel-axle-rear-collar"}[i], "collar", "wheel", func() *decad.Body {
			return b.liftAnnulus(wl, wheelAt(0, 0, interval[1]), 4, 6, interval[1]-interval[0])
		})
	}
	for i, interval := range [][2]float64{{-55 * s, -45 * s}, {45 * s, 55 * s}} {
		b.part([]string{"wheel-axle-front-thrust-spacer", "wheel-axle-rear-thrust-spacer"}[i], "spacer", "wheel", func() *decad.Body {
			return b.liftAnnulus(wl, wheelAt(0, 0, interval[1]), 4, 6, interval[1]-interval[0])
		})
	}
}

func (b *builder) adjustment() {
	s := b.p.Scale
	shift := b.a.State.HeadShift
	alignment := shift - 278*s
	b.tilt = false
	b.part("mother-base", "frame", "fixed", func() *decad.Body {
		section := b.section(false)
		polygon(section, []point2{
			{-195 * s, -85*s + alignment}, {-30 * s, -85*s + alignment},
			{-30 * s, -45*s + alignment}, {-105 * s, -45*s + alignment},
			{-105 * s, 280*s + alignment}, {-195 * s, 280*s + alignment},
		})
		for _, x := range []float64{-180, -120} {
			b.alignmentSlot(section, x*s, -75*s+alignment, -25*s+alignment)
		}
		return b.extrude(section, 2, 15*s,
			frame(r3.Vec{Z: 525 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	})
	for i, x := range []float64{-180, -120} {
		b.part(fmt.Sprintf("mother-alignment-clamp-%d", i+1), "fastener", "fixed", func() *decad.Body {
			return b.verticalBolt(lift{xf: r3.Identity()}, x*s, -70*s, 515*s, 540*s, 3)
		})
		b.part(fmt.Sprintf("mother-alignment-insert-%d", i+1), "insert", "fixed", func() *decad.Body {
			return b.panel(-5, -5, 5, 5, 8,
				frame(r3.Vec{X: x * s, Y: -70 * s, Z: 525*s - 8}, r3.Vec{X: 1}, r3.Vec{Y: 1}), hole{0, 0, 3})
		})
	}
	for i, q := range []float64{-300*s + shift, -90*s + shift} {
		b.hinge(i+1, q)
	}
	b.tilt = true
	hl := b.headLift()
	b.liftedPart("mother-upper-bar", "frame", "fixed", func() *decad.Body {
		var holes []hole
		for _, front := range []float64{-314, -54} {
			for _, x := range []float64{-150, -130} {
				holes = append(holes, hole{x * s, front*s - 6.2 + shift, 3.3})
			}
		}
		return b.liftPanel(hl, level(0, 0, 555*s), b.boardEdge(), -330*s+shift, -118*s, -20*s+shift, 15*s, holes...)
	})
	b.liftedPart("mother-jack-arm", "adjustment", "fixed", func() *decad.Body {
		body := b.liftPanel(hl, level(0, 0, 555*s), -172*s, -342*s+shift, -35*s, -318*s+shift, 15*s,
			hole{-50 * s, -330*s + shift, 4.3})
		return b.cut(body, b.localStock(hl, -50*s-8.1, -50*s+8.1,
			-330*s+shift-8.1, -330*s+shift+8.1, 555*s, 555*s+8))
	})
	b.liftedPart("mother-jack-square-insert", "insert", "fixed", func() *decad.Body {
		x, y := -50*s, -330*s+shift
		return b.liftPanel(hl, level(0, 0, 555*s), x-8, y-8, x+8, y+8, 8, hole{x, y, 4})
	})
	theta := b.a.State.HeadTilt
	h := (15*s + 122*s*math.Sin(theta)) / math.Cos(theta)
	phase := math.Remainder(2*math.Pi*(h-15*s)/1.25, 2*math.Pi)
	axis := frame(r3.Vec{X: -50 * s, Y: -330*s + shift, Z: 555*s - h},
		r3.Vec{X: math.Cos(phase), Y: math.Sin(phase)}, r3.Vec{X: -math.Sin(phase), Y: math.Cos(phase)})
	// The jack screw and its knob share one lift, the screw's phased axis frame then the head tilt, and every
	// piece is placed on it before its Boolean.
	jl := b.tilted(b.liftOf(axis))
	b.liftedPart("mother-jack-screw", "adjustment", "fixed", func() *decad.Body {
		body := b.liftRevolve(jl, zAxis(0, 0, 0), []point2{{0, 0}, {4, 4}, {4, 58 * s}, {0, 58 * s}})
		// The knob flat runs 5 past the screw's top face it opens.
		return b.cut(body, b.localStock(jl, 3, 5, -5, 5, 42*s, 58*s+5))
	})
	b.liftedPart("mother-jack-knob", "adjustment", "fixed", func() *decad.Body {
		knob := b.sectionOn(level(0, 0, 42*s))
		circle(knob, 0, 0, 12*s)
		dOutline(knob, 4.1, 3.1)
		return b.liftExtrude(knob, 1, 10*s, jl)
	})
	for i, post := range [][2]float64{{-314, -286}, {-54, -30}} {
		footY := post[0]*s - 6.2 + shift
		b.liftedPart(fmt.Sprintf("maiden-foot-%d", i+1), "frame", "fixed", func() *decad.Body {
			return b.liftPanel(hl, level(0, 0, 570*s), -175*s, post[0]*s-12+shift, -105*s, post[1]*s+2+shift, 15*s,
				hole{-150 * s, footY, 3.3}, hole{-130 * s, footY, 3.3})
		})
		for j, x := range []float64{-150, -130} {
			b.liftedPart(fmt.Sprintf("maiden-foot-%d-bolt-%d", i+1, j+1), "fastener", "fixed", func() *decad.Body {
				return b.verticalBolt(hl, x*s, footY, 560*s, 585*s, 4.2)
			})
			b.liftedPart(fmt.Sprintf("maiden-foot-%d-insert-%d", i+1, j+1), "insert", "fixed", func() *decad.Body {
				return b.liftPanel(hl, frame(r3.Vec{X: x * s, Y: footY, Z: 570*s - 8}, r3.Vec{X: 1}, r3.Vec{Y: 1}),
					-5, -5, 5, 5, 8, hole{0, 0, 3})
			})
		}
	}
	b.tilt = false
}

// boardEdge is the upper board's hinge-side X: 5*Scale from the hinge axis, but never inside the stock radius5
// knuckles, which do not scale. Below Scale 1 the edge recedes to touch the knuckles instead of crossing them.
func (b *builder) boardEdge() float64 {
	s := b.p.Scale
	return -172*s + max(5*s, 5)
}

// fixedBracket is a fixed hinge bracket: one prism along Y whose section is the bracket's rectangle joined to
// its stock radius5 knuckle, with the knuckle's radius2.7 pin bore, cut by a vertical screw hole that runs 5
// past both bracket faces. One prism needs no Union, so Verify reads the bracket's faces directly.
func (b *builder) fixedBracket(y0, y1, middle float64) *decad.Body {
	s := b.p.Scale
	sk := b.sectionOn(frame(r3.Vec{Y: y1}, r3.Vec{X: 1}, r3.Vec{Z: 1}))
	if sk == nil {
		return nil
	}
	cx, cz, r := -172*s, 555*s, 5.0
	d := math.Sqrt(r*r - 9*s*s)
	centre := sk.CreatePoint(cx, cz)
	a := sk.CreatePoint(-190*s, 540*s)
	bb := sk.CreatePoint(-175*s, 540*s)
	c := sk.CreatePoint(-175*s, cz-d)
	dd := sk.CreatePoint(cx-r, cz)
	e := sk.CreatePoint(-190*s, cz)
	arc := sk.CreateArc(centre, c, dd)
	ab, bc := sk.CreateLine(a, bb), sk.CreateLine(bb, c)
	de, ea := sk.CreateLine(dd, e), sk.CreateLine(e, a)
	bore := sk.CreateCircle(centre, 2.7)
	sk.AddConstraint(
		sketch.NewHorizontalDistance(sk.Origin(), centre, cx),
		sketch.NewVerticalDistance(sk.Origin(), centre, cz),
		sketch.NewHorizontalDistance(sk.Origin(), a, -190*s),
		sketch.NewVerticalDistance(sk.Origin(), a, 540*s),
		sketch.NewHorizontal(ab),
		sketch.NewHorizontalDistance(a, bb, 15*s),
		sketch.NewVertical(bc),
		sketch.NewRadius(arc, r),
		sketch.NewHorizontalPoints(centre, dd),
		sketch.NewHorizontal(de),
		sketch.NewVertical(ea),
		sketch.NewRadius(bore, 2.7),
	)
	body := b.liftExtrude(sk, 1, y1-y0, lift{xf: r3.Identity()})
	return b.cut(body, b.liftCylinder(lift{xf: r3.Identity()}, level(-183*s, middle, 540*s-5), 1.6, 15*s+10))
}

func (b *builder) alignmentSlot(section *sketch.Sketch, x, y0, y1 float64) {
	if b.err != nil {
		return
	}
	slot, err := section.CreateSlot(x, y0, x, y1, 3.3)
	if err != nil {
		b.fail("create alignment slot", err)
		return
	}
	section.AddConstraint(
		sketch.NewHorizontalDistance(section.Origin(), slot.C1, x),
		sketch.NewVerticalDistance(section.Origin(), slot.C1, y0),
		sketch.NewHorizontalDistance(slot.C1, slot.C2, 0),
		sketch.NewVerticalDistance(slot.C1, slot.C2, y1-y0),
		sketch.NewRadius(slot.A1, 3.3),
	)
}

func (b *builder) verticalBolt(l lift, x, y, z0, z1, headHeight float64) *decad.Body {
	length := z1 - z0
	return b.liftRevolve(l, frame(r3.Vec{X: x, Y: y, Z: z0}, r3.Vec{X: 1}, r3.Vec{Z: 1}),
		[]point2{{0, 0}, {3, 0}, {3, length}, {6, length}, {6, length + headHeight}, {0, length + headHeight}})
}

func (b *builder) hinge(index int, q float64) {
	s := b.p.Scale
	prefix := fmt.Sprintf("mother-hinge-%d-", index)
	b.tilt = false
	for j, interval := range [][2]float64{{q, q + 15*s}, {q + 32*s, q + 47*s}} {
		middle := (interval[0] + interval[1]) / 2
		b.part(fmt.Sprintf("%sfixed-bracket-%d", prefix, j+1), "hinge", "fixed", func() *decad.Body {
			return b.fixedBracket(interval[0], interval[1], middle)
		})
		b.part(fmt.Sprintf("%sfixed-mount-%d", prefix, j+1), "fastener", "fixed", func() *decad.Body {
			return b.revolve([]point2{{0, 0}, {1.5, 0}, {1.5, 23 * s}, {3, 23 * s},
				{3, 23*s + 1.5}, {0, 23*s + 1.5}},
				frame(r3.Vec{X: -183 * s, Y: middle, Z: 532 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
		})
	}
	b.part(prefix+"pin", "pin", "fixed", func() *decad.Body {
		return b.revolve([]point2{{0, 0}, {4, 0}, {4, 2}, {2.5, 2},
			{2.5, 47*s + 5.2}, {0, 47*s + 5.2}},
			frame(r3.Vec{X: -172 * s, Y: q - 2, Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Z: -1}))
	})
	b.part(prefix+"nut", "fastener", "fixed", func() *decad.Body {
		return b.annulus(2.5, 4, 3, shaftFrame(-172*s, q+47*s+3, 555*s, 0))
	})
	b.tilt = true
	b.liftedPart(prefix+"moving-leaf", "hinge", "fixed", func() *decad.Body {
		// Leaf, knuckle and pin bore share one plane so their caps stay co-directional prisms, and every piece is
		// placed on the head lift before its Boolean. The pin bore and the screw hole run past the faces they open.
		hl := b.headLift()
		plane := func(y float64) r3.Frame { return frame(r3.Vec{X: -172 * s, Y: y, Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}) }
		leaf := b.liftPanel(hl, plane(q+31*s), 3*s, 0, 14*s, 3*s, 15*s)
		knuckle := b.liftCylinder(hl, plane(q+31*s), 5, 15*s)
		body := b.cut(b.union(leaf, knuckle), b.liftCylinder(hl, plane(q+31*s+5), 2.7, 15*s+10))
		screw := b.liftCylinder(hl, level(-162*s, q+23.5*s, 555*s-1), 1.6, 3*s+2)
		return b.cut(body, screw)
	})
	// The moving mount turns about +Z on the head lift, on the axis of its upper-board pilot (frameJoinery).
	b.liftedPart(prefix+"moving-mount", "fastener", "fixed", func() *decad.Body {
		return b.liftRevolve(b.headLift(), frame(r3.Vec{X: -162 * s, Y: q + 23.5*s, Z: 555*s - 1.5}, r3.Vec{X: 1}, r3.Vec{Z: 1}),
			[]point2{{0, 0}, {3, 0}, {3, 1.5}, {1.5, 1.5}, {1.5, 11*s + 1.5}, {0, 11*s + 1.5}})
	})
	b.tilt = false
}
