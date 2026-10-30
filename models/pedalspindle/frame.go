package pedalspindle

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

func corners() [4][2]float64 {
	return [4][2]float64{{-165, 308}, {90, 308}, {-165, 500}, {90, 500}}
}

func (b *builder) frameHoles() []hole {
	s := b.scale
	hs := []hole{{-90 * s, 380 * s, 22}, {12.3 * s, 380 * s, 22}, {12.3 * s, 482.3 * s, 14}}
	for _, p := range corners() {
		hs = append(hs, hole{p[0] * s, p[1] * s, 6.6})
	}
	return hs
}

func (b *builder) portal() *decad.Body {
	s := b.section(false, 0)
	if b.err != nil {
		return nil
	}
	k := b.scale
	coords := [][2]float64{{-175, 18}, {-155, 18}, {-155, 298}, {80, 298}, {80, 18}, {100, 18}, {100, 510}, {-175, 510}}
	for i := range coords {
		coords[i][0] *= k
		coords[i][1] *= k
	}
	ps, ls := polygon(s, coords, false)
	for _, i := range []int{0, 2, 4, 6} {
		s.AddConstraint(sketch.NewHorizontal(ls[i]))
	}
	for _, i := range []int{1, 3, 5, 7} {
		s.AddConstraint(sketch.NewVertical(ls[i]))
	}
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[0], -175*k),
		sketch.NewVerticalDistance(s.Origin(), ps[0], 18*k))
	for i, d := range []float64{20, 235, 20} {
		j := 2 * i
		s.AddConstraint(sketch.NewHorizontalDistance(ps[j], ps[j+1], d*k))
	}
	for i, d := range []float64{280, -280, 492} {
		j := 2*i + 1
		s.AddConstraint(sketch.NewVerticalDistance(ps[j], ps[j+1], d*k))
	}
	rectangle(s, -155*k, 318*k, 80*k, 362*k)
	rectangle(s, -155*k, 398*k, 80*k, 468.3*k)
	holes(s, b.frameHoles())
	return b.extrude(s, 9, 12*k)
}

func (b *builder) bolt() *decad.Body {
	s := b.section(true, 0)
	if b.err != nil {
		return nil
	}
	k := b.scale
	ps, ls := polygon(s, [][2]float64{{0, 0}, {3, 0}, {3, 125 * k}, {6, 125 * k}, {6, 125*k + 4.2}, {0, 125*k + 4.2}}, true)
	for _, i := range []int{0, 2, 4} {
		s.AddConstraint(sketch.NewHorizontal(ls[i]))
	}
	for _, i := range []int{1, 3, 5} {
		s.AddConstraint(sketch.NewVertical(ls[i]))
	}
	s.AddConstraint(sketch.NewHorizontalDistance(ps[0], ps[1], 3),
		sketch.NewVerticalDistance(ps[0], ps[2], 125*k),
		sketch.NewHorizontalDistance(ps[0], ps[3], 6),
		sketch.NewVerticalDistance(ps[3], ps[4], 4.2))
	return b.revolve(s)
}

func (b *builder) frame() {
	s := b.scale
	b.box("base", "frame", -270, 150, -240, 110, 0, 18)
	b.box("left-leg", "frame", -180, -156, 0, 18, 18, 290)
	b.box("right-leg", "frame", 80, 104, 0, 18, 18, 290)
	board := b.rect(-185*s, 290*s, 110*s, 525*s, 18*s, 0, b.frameHoles()...)
	b.add("bearing-board", "frame", b.place(board, 0, 18*s, 0, 0))
	b.add("front-bearing-frame", "frame", b.place(b.portal(), 0, -83*s, 0, 0))
	for i, p := range corners() {
		x, z := p[0]*s, p[1]*s
		b.ring(fmt.Sprintf("frame-standoff-%d", i), "frame", x, z, -83*s, 0, 20*s, 6.6, 0)
		b.add(fmt.Sprintf("frame-bolt-%d", i), "steel", b.place(b.bolt(), x, 30*s, z, 0))
		b.ring(fmt.Sprintf("frame-nut-%d", i), "steel", x, z, 18*s, 18*s+5.5, 12, 6, 0)
	}
}
