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
	for _, leg := range []struct {
		name string
		x    float64
		yz   []point2
		heel bool
	}{
		{"head-leg", -280, []point2{{-220, 0}, {-180, 0}, {120, 500}, {-100, 500}}, true},
		{"front-wheel-leg", 200, []point2{{-230, 0}, {-190, 0}, {-25, 550}, {-65, 550}}, true},
		{"rear-wheel-leg", 200, []point2{{190, 0}, {230, 0}, {65, 550}, {25, 550}}, false},
	} {
		b.part(leg.name, "frame", "fixed", func() *decad.Body {
			section := b.section(false)
			polygon(section, scaled(s, leg.yz...))
			holes := 0
			if leg.heel {
				circle(section, -190*s, 60*s, 11)
				holes = 1
			}
			return b.extrude(section, holes, 40*s,
				frame(r3.Vec{X: leg.x * s}, r3.Vec{Y: 1}, r3.Vec{Z: 1}))
		})
	}
	for _, side := range []struct {
		label string
		y     float64
	}{
		{"front", -55}, {"rear", 85},
	} {
		f := frame(r3.Vec{Y: side.y * s}, r3.Vec{X: 1}, r3.Vec{Z: 1})
		b.part(side.label+"-bench-rail", "frame", "fixed", func() *decad.Body {
			section := b.section(false)
			polygon(section, scaled(s, point2{-280, 465}, point2{240, 375},
				point2{240, 410}, point2{-280, 500}))
			return b.extrude(section, 0, 30*s, f)
		})
		b.part(side.label+"-head-wedge", "frame", "fixed", func() *decad.Body {
			section := b.section(false)
			polygon(section, scaled(s, point2{-195, 500 - 85*90.0/520},
				point2{-105, 500 - 175*90.0/520}, point2{-105, 525}, point2{-195, 525}))
			wedge := b.extrude(section, 0, 30*s, f)
			if side.label == "front" {
				// Both clamp axes stay fixed while the slotted base slides in Y.
				passages := make([]*decad.Body, 0, 2)
				for _, x := range []float64{-180, -120} {
					passages = append(passages, b.cylinder(3.3, 80*s,
						frame(r3.Vec{X: x * s, Y: -60 * s, Z: 450 * s},
							r3.Vec{X: 1}, r3.Vec{Y: 1})))
				}
				wedge = b.cut(wedge, b.union(passages...))
			}
			return wedge
		})
	}
	for i, cap := range []struct {
		label   string
		y       float64
		bearing float64
	}{
		{"front", -25, -45*s + 7}, {"rear", 55, 45 * s},
	} {
		b.part(cap.label+"-wheel-bearing-cap", "frame", "fixed", func() *decad.Body {
			return b.panel(200*s, 550*s, 240*s, 610*s, 30*s,
				frame(r3.Vec{Y: cap.y * s}, r3.Vec{X: 1}, r3.Vec{Z: 1}),
				hole{220 * s, 580 * s, 11})
		})
		b.part(fmt.Sprintf("wheel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.annulus(4, 11, 7, shaftFrame(220*s, cap.bearing, 580*s, 0))
		})
	}
	for i, x := range []float64{-260, 220} {
		b.part(fmt.Sprintf("heel-bearing-%d", i+1), "bearing", "fixed", func() *decad.Body {
			return b.annulus(4, 11, 7, frame(r3.Vec{X: x*s - 3.5, Y: -190 * s, Z: 60 * s},
				r3.Vec{Y: 1}, r3.Vec{Z: 1}))
		})
	}
	for _, ring := range []struct {
		name, group string
		y, length   float64
	}{
		{"wheel-axle-front-collar", "collar", -55 * s, 3},
		{"wheel-axle-rear-collar", "collar", 55*s + 3, 3},
		{"wheel-axle-front-thrust-spacer", "spacer", -45 * s, 10 * s},
		{"wheel-axle-rear-thrust-spacer", "spacer", 55 * s, 10 * s},
	} {
		b.part(ring.name, ring.group, "wheel", func() *decad.Body {
			return b.annulus(4, 6, ring.length, shaftFrame(220*s, ring.y, 580*s, 0))
		})
	}
}

func (b *builder) adjustment() {
	s, shift := b.p.Scale, b.a.State.HeadShift
	alignment := shift - 278*s
	b.tilt = false
	b.part("mother-base", "frame", "fixed", func() *decad.Body {
		section := b.section(false)
		polygon(section, []point2{
			{-195 * s, -85*s + alignment}, {-30 * s, -85*s + alignment},
			{-30 * s, -45*s + alignment}, {-105 * s, -45*s + alignment},
			{-105 * s, 280*s + alignment}, {-195 * s, 280*s + alignment},
		})
		if b.err != nil {
			return nil
		}
		for _, x := range []float64{-180, -120} {
			low, high := -75*s+alignment, -25*s+alignment
			slot, err := section.CreateSlot(x*s, low, x*s, high, 3.3)
			if err != nil {
				b.fail("create base alignment slot", err)
				return nil
			}
			section.AddConstraint(
				sketch.NewHorizontalDistance(section.Origin(), slot.C1, x*s),
				sketch.NewVerticalDistance(section.Origin(), slot.C1, low),
				sketch.NewHorizontalDistance(section.Origin(), slot.C2, x*s),
				sketch.NewVerticalDistance(section.Origin(), slot.C2, high),
				sketch.NewDistance(slot.C1, slot.L1.Start, 3.3),
			)
		}
		return b.extrude(section, 2, 15*s,
			frame(r3.Vec{Z: 525 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	})
	for i, x := range []float64{-180, -120} {
		b.part(fmt.Sprintf("mother-alignment-clamp-%d", i+1), "adjustment", "fixed", func() *decad.Body {
			return b.motherBolt(x*s, -60*s, 515*s, 540*s, 3)
		})
	}
	for i, start := range []float64{-300, -90} {
		prefix := fmt.Sprintf("mother-hinge-%d-", i+1)
		q := start*s + shift
		b.tilt = false
		for j, interval := range [][2]float64{{q, q + 15*s}, {q + 32*s, q + 47*s}} {
			b.part(fmt.Sprintf("%sfixed-knuckle-%d", prefix, j+1), "hinge", "fixed", func() *decad.Body {
				return b.annulus(2.5, 5, interval[1]-interval[0],
					shaftFrame(-172*s, interval[1], 555*s, 0))
			})
			b.part(fmt.Sprintf("%sfixed-bracket-%d", prefix, j+1), "hinge", "fixed", func() *decad.Body {
				return b.box(-190*s, -175*s, interval[0], interval[1], 540*s, 555*s)
			})
		}
		b.part(prefix+"pin", "hinge", "fixed", func() *decad.Body {
			return b.cylinder(2.5, 49*s, shaftFrame(-172*s, q+48*s, 555*s, 0))
		})
		b.tilt = true
		b.part(prefix+"moving-knuckle", "hinge", "fixed", func() *decad.Body {
			return b.annulus(2.5, 5, 15*s, shaftFrame(-172*s, q+31*s, 555*s, 0))
		})
		b.part(prefix+"moving-leaf", "hinge", "fixed", func() *decad.Body {
			leaf := b.box(-169*s, -158*s, q+16*s, q+31*s, 555*s, 558*s)
			relief := b.cylinder(2.7, 15*s, shaftFrame(-172*s, q+31*s, 555*s, 0))
			return b.cut(leaf, relief)
		})
	}
	b.tilt = true
	b.part("mother-upper-bar", "frame", "fixed", func() *decad.Body {
		holes := make([]hole, 0, 4)
		for _, front := range []float64{-314, -54} {
			for _, x := range []float64{-150, -130} {
				holes = append(holes, hole{x * s, front*s - 6 + shift, 3.3})
			}
		}
		return b.panel(-167*s, -330*s+shift, -118*s, -20*s+shift, 15*s,
			frame(r3.Vec{Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}), holes...)
	})
	b.part("mother-jack-arm", "frame", "fixed", func() *decad.Body {
		arm := b.panel(-172*s, -342*s+shift, -35*s, -318*s+shift, 15*s,
			frame(r3.Vec{Z: 555 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}),
			hole{-50 * s, -330*s + shift, 4.3})
		pocket := b.box(-50*s-8.1, -50*s+8.1, -330*s+shift-8.1, -330*s+shift+8.1,
			555*s, 555*s+8)
		return b.cut(arm, pocket)
	})
	b.part("mother-jack-square-insert", "adjustment", "fixed", func() *decad.Body {
		return b.panel(-8, -8, 8, 8, 8,
			frame(r3.Vec{X: -50 * s, Y: -330*s + shift, Z: 555 * s},
				r3.Vec{X: 1}, r3.Vec{Y: 1}), hole{0, 0, 4})
	})
	theta := b.a.State.HeadTilt
	h := (15*s + 122*s*math.Sin(theta)) / math.Cos(theta)
	tip := 555*s - h
	phase := math.Remainder(2*math.Pi*(h-15*s)/1.25, 2*math.Pi)
	u, v := r3.Vec{X: math.Cos(phase), Y: math.Sin(phase)}, r3.Vec{X: -math.Sin(phase), Y: math.Cos(phase)}
	jackOrigin := r3.Vec{X: -50 * s, Y: -330*s + shift, Z: tip}
	b.part("mother-jack-screw", "adjustment", "fixed", func() *decad.Body {
		local := frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		shaft := b.revolve([]point2{{0, 0}, {4, 4}, {4, 58 * s}, {0, 58 * s}}, local)
		shaft = b.flatShaft(shaft, b.box(3, 5, -5, 5, 42*s, 52*s))
		return b.place(shaft, frame(jackOrigin, u, v))
	})
	b.part("mother-jack-knob", "adjustment", "fixed", func() *decad.Body {
		origin := jackOrigin.Add(r3.Vec{Z: 42 * s})
		return b.dAnnulus(4.1, 3.1, 12*s, 10*s, frame(origin, u, v))
	})
	for i, post := range [][2]float64{{-314, -286}, {-54, -30}} {
		prefix := fmt.Sprintf("maiden-foot-%d", i+1)
		boltY := post[0]*s - 6 + shift
		b.part(prefix, "frame", "fixed", func() *decad.Body {
			return b.panel(-175*s, post[0]*s-12+shift, -105*s, post[1]*s+2+shift, 15*s,
				frame(r3.Vec{Z: 570 * s}, r3.Vec{X: 1}, r3.Vec{Y: 1}),
				hole{-150 * s, boltY, 3.3}, hole{-130 * s, boltY, 3.3})
		})
		for j, x := range []float64{-150, -130} {
			b.part(fmt.Sprintf("%s-bolt-%d", prefix, j+1), "adjustment", "fixed", func() *decad.Body {
				return b.motherBolt(x*s, boltY, 560*s, 585*s, 4.2)
			})
		}
	}
	b.tilt = false
}

func (b *builder) motherBolt(x, y, z0, z1, headHeight float64) *decad.Body {
	length := z1 - z0
	return b.revolve([]point2{
		{0, 0}, {3, 0}, {3, length}, {6, length}, {6, length + headHeight}, {0, length + headHeight},
	}, frame(r3.Vec{X: x, Y: y, Z: z0}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
}
