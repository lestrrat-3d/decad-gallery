package spinningwheel

import (
	"fmt"
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"math"
)

func (b *builder) brake() {
	s := b.p.Scale
	st := b.a.State
	travel := b.p.BrakeTravel
	radius := 42*s + .6*math.Sqrt(5)
	y := -69*s + st.HeadShift
	left, right := st.Flyer.X-radius, st.Flyer.X+radius
	top, bottom := 635*s+travel, 610*s
	e := 640*s - travel
	b.part("scotch-band-upper-wrap", "brake-band", "fixed", func() *decad.Body { return b.torus(radius, .6, 0, math.Pi, r3.Vec{X: st.Flyer.X, Y: y, Z: st.Flyer.Z}) })
	b.part("scotch-band-left-tail", "brake-band", "fixed", func() *decad.Body {
		return b.span(r3.Vec{X: left, Y: y, Z: top}, r3.Vec{X: left, Y: y, Z: st.Flyer.Z}, .6)
	})
	b.part("scotch-band-right-tail", "brake-band", "fixed", func() *decad.Body {
		return b.span(r3.Vec{X: right, Y: y, Z: e}, r3.Vec{X: right, Y: y, Z: st.Flyer.Z}, .6)
	})
	b.part("scotch-extension-spring", "spring", "fixed", func() *decad.Body {
		sk := b.section(true)
		if b.err != nil {
			return nil
		}
		circle(sk, 2.1, 0, .4)
		p := b.profile(sk, 0)
		if b.err != nil {
			return nil
		}
		body, err := decad.Coil(b.ctx, b.doc, sk, p, decad.ConstructionAxis{Dir: r3.Vec{Z: 1}}, units.Millimeters((top-bottom-.8)/6), units.Scalar(6))
		b.fail("coil extension spring", err)
		return b.place(body, frame(r3.Vec{X: left, Y: y, Z: bottom + .4}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
	})
	for i, z := range []float64{bottom, top} {
		coilZ := z + .4
		if i == 1 {
			coilZ = z - .4
		}
		b.part(fmt.Sprintf("spring-end-link-%d", i+1), "spring", "fixed", func() *decad.Body {
			return b.span(r3.Vec{X: left + 2.1, Y: y, Z: coilZ}, r3.Vec{X: left + 2.1, Y: y, Z: z}, .4)
		})
		b.part(fmt.Sprintf("spring-end-bridge-%d", i+1), "spring", "fixed", func() *decad.Body { return b.span(r3.Vec{X: left + 2.1, Y: y, Z: z}, r3.Vec{X: left, Y: y, Z: z}, .4) })
	}
	b.part("spring-wood-anchor", "fastener", "fixed", func() *decad.Body {
		return b.cylinder(.8, 4*s, axisFrame(r3.Vec{X: left, Y: y, Z: 606 * s}, r3.Vec{Z: 1}))
	})
	rodF := func(z float64) r3.Frame { return axisFrame(r3.Vec{X: right, Y: y, Z: z}, r3.Vec{Z: 1}) }
	b.part("brake-adjuster-thread-envelope", "thread-envelope", "fixed", func() *decad.Body {
		host := b.cylinder(2, e-4-(590*s-travel), rodF(590*s-travel))
		return b.cut(host, b.panel(1.3, -3, 3, 3, 5, rodF(e-10)))
	})
	b.part("brake-adjuster-knob", "knob", "fixed", func() *decad.Body { return b.dAnnulus(2.1, 1.4, 10*s, 5, rodF(e-10)) })
	b.part("brake-band-eye", "brake", "fixed", func() *decad.Body {
		return b.annulus(1.3, 2.5, 1.2, axisFrame(r3.Vec{X: right - .6, Y: y, Z: e - 2.5}, r3.Vec{X: 1}))
	})
	b.part("brake-square-insert-thread-envelope", "thread-envelope", "fixed", func() *decad.Body {
		sk := b.section(false)
		if b.err != nil {
			return nil
		}
		rectangle(sk, -4, -4, 4, 4)
		circle(sk, 0, 0, 2)
		return b.extrude(sk, 1, 4, rodF(595*s))
	})
}
