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
			if leg.heel {
				circle(section, -190*s, 60*s, 11)
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
	for i, y0 := range []float64{-55, 25} {
		name := []string{"front-wheel-bearing-cap", "rear-wheel-bearing-cap"}[i]
		b.part(name, "frame", "fixed", func() *decad.Body {
			return b.panel(200*s, 535*s, 240*s, 610*s, 30*s,
				frame(r3.Vec{Y: (y0 + 30) * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}), hole{220 * s, 580 * s, 11})
		})
	}
	for i, y0 := range []float64{-45 * s, 45*s - 7} {
		b.part(fmt.Sprintf("wheel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.annulus(4, 11, 7, shaftFrame(220*s, y0+7, 580*s, 0))
		})
	}
	for i, x := range []float64{-260, 220} {
		b.part(fmt.Sprintf("heel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.annulus(4, 11, 7,
				frame(r3.Vec{X: x*s - 3.5, Y: -190 * s, Z: 60 * s}, r3.Vec{Y: 1}, r3.Vec{Z: 1}))
		})
	}
	for i, interval := range [][2]float64{{-55*s - 3, -55 * s}, {55 * s, 55*s + 3}} {
		b.part([]string{"wheel-axle-front-collar", "wheel-axle-rear-collar"}[i], "collar", "wheel", func() *decad.Body {
			return b.annulus(4, 6, interval[1]-interval[0],
				shaftFrame(220*s, interval[1], 580*s, 0))
		})
	}
	for i, interval := range [][2]float64{{-55 * s, -45 * s}, {45 * s, 55 * s}} {
		b.part([]string{"wheel-axle-front-thrust-spacer", "wheel-axle-rear-thrust-spacer"}[i], "spacer", "wheel", func() *decad.Body {
			return b.annulus(4, 6, interval[1]-interval[0],
				shaftFrame(220*s, interval[1], 580*s, 0))
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
			return b.verticalBolt(x*s, -70*s, 515*s, 540*s, 3)
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
	b.part("mother-upper-bar", "frame", "fixed", func() *decad.Body {
		var holes []hole
		for _, front := range []float64{-314, -54} {
			for _, x := range []float64{-150, -130} {
				holes = append(holes, hole{x * s, front*s - 6.2 + shift, 3.3})
			}
		}
		return b.panel(-167*s, -330*s+shift, -118*s, -20*s+shift, 15*s,
			frame(r3.Vec{Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}), holes...)
	})
	b.part("mother-jack-arm", "adjustment", "fixed", func() *decad.Body {
		body := b.panel(-172*s, -342*s+shift, -35*s, -318*s+shift, 15*s,
			frame(r3.Vec{Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}), hole{-50 * s, -330*s + shift, 4.3})
		return b.cut(body, b.box(-50*s-8.1, -50*s+8.1,
			-330*s+shift-8.1, -330*s+shift+8.1, 555*s, 555*s+8))
	})
	b.part("mother-jack-square-insert", "insert", "fixed", func() *decad.Body {
		return b.panel(-8, -8, 8, 8, 8,
			frame(r3.Vec{X: -50 * s, Y: -330*s + shift, Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}), hole{0, 0, 4})
	})
	theta := b.a.State.HeadTilt
	h := (15*s + 122*s*math.Sin(theta)) / math.Cos(theta)
	phase := math.Remainder(2*math.Pi*(h-15*s)/1.25, 2*math.Pi)
	axis := frame(r3.Vec{X: -50 * s, Y: -330*s + shift, Z: 555*s - h},
		r3.Vec{X: math.Cos(phase), Y: math.Sin(phase)}, r3.Vec{X: -math.Sin(phase), Y: math.Cos(phase)})
	b.part("mother-jack-screw", "adjustment", "fixed", func() *decad.Body {
		body := b.revolve([]point2{{0, 0}, {4, 4}, {4, 58 * s}, {0, 58 * s}},
			frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
		body = b.cut(body, b.box(3, 5, -5, 5, 42*s, 58*s))
		return b.place(body, axis)
	})
	b.part("mother-jack-knob", "adjustment", "fixed", func() *decad.Body {
		knobFrame := frame(axis.ToWorld(r3.Vec{Z: 42 * s}), axis.U(), axis.V())
		return b.dAnnulus(4.1, 3.1, 12*s, 10*s, knobFrame)
	})
	for i, post := range [][2]float64{{-314, -286}, {-54, -30}} {
		footY := post[0]*s - 6.2 + shift
		b.part(fmt.Sprintf("maiden-foot-%d", i+1), "frame", "fixed", func() *decad.Body {
			return b.panel(-175*s, post[0]*s-12+shift, -105*s, post[1]*s+2+shift, 15*s,
				frame(r3.Vec{Z: 570 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}),
				hole{-150 * s, footY, 3.3}, hole{-130 * s, footY, 3.3})
		})
		for j, x := range []float64{-150, -130} {
			b.part(fmt.Sprintf("maiden-foot-%d-bolt-%d", i+1, j+1), "fastener", "fixed", func() *decad.Body {
				return b.verticalBolt(x*s, footY, 560*s, 585*s, 4.2)
			})
			b.part(fmt.Sprintf("maiden-foot-%d-insert-%d", i+1, j+1), "insert", "fixed", func() *decad.Body {
				return b.panel(-5, -5, 5, 5, 8,
					frame(r3.Vec{X: x * s, Y: footY, Z: 570*s - 8}, r3.Vec{X: 1}, r3.Vec{Y: 1}), hole{0, 0, 3})
			})
		}
	}
	b.tilt = false
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

func (b *builder) verticalBolt(x, y, z0, z1, headHeight float64) *decad.Body {
	length := z1 - z0
	return b.revolve([]point2{{0, 0}, {3, 0}, {3, length}, {6, length},
		{6, length + headHeight}, {0, length + headHeight}},
		frame(r3.Vec{X: x, Y: y, Z: z0}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
}

func (b *builder) hinge(index int, q float64) {
	s := b.p.Scale
	prefix := fmt.Sprintf("mother-hinge-%d-", index)
	b.tilt = false
	for j, interval := range [][2]float64{{q, q + 15*s}, {q + 32*s, q + 47*s}} {
		middle := (interval[0] + interval[1]) / 2
		b.part(fmt.Sprintf("%sfixed-bracket-%d", prefix, j+1), "hinge", "fixed", func() *decad.Body {
			bracket := b.panel(-190*s, interval[0], -175*s, interval[1], 15*s,
				frame(r3.Vec{Z: 540 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}), hole{-183 * s, middle, 1.6})
			knuckle := b.annulus(2.7, 5, interval[1]-interval[0], shaftFrame(-172*s, interval[1], 555*s, 0))
			return b.union(bracket, knuckle)
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
	b.part(prefix+"moving-leaf", "hinge", "fixed", func() *decad.Body {
		// Leaf, knuckle and pin bore share one shaft-frame plane so their caps stay
		// co-directional prisms. The screw hole is cut last by a tool that clears both leaf faces.
		axis := shaftFrame(-172*s, q+31*s, 555*s, 0)
		leaf := b.panel(3*s, 0, 14*s, 3*s, 15*s, axis)
		knuckle := b.cylinder(5, 15*s, axis)
		body := b.cut(b.union(leaf, knuckle), b.cylinder(2.7, 15*s, axis))
		screw := b.cylinder(1.6, 3*s+2, frame(r3.Vec{X: -162 * s, Y: q + 23.5*s, Z: 555*s - 1}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
		return b.cut(body, screw)
	})
	b.part(prefix+"moving-mount", "fastener", "fixed", func() *decad.Body {
		return b.revolve([]point2{{0, 0}, {3, 0}, {3, 1.5}, {1.5, 1.5},
			{1.5, 11*s + 1.5}, {0, 11*s + 1.5}},
			frame(r3.Vec{X: -162 * s, Y: q + 23.5*s, Z: 555*s - 1.5}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	})
	b.tilt = false
}
