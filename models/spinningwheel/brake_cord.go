package spinningwheel

import (
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) brakeCord(left, right, leftEnd, rightEnd, z, radius, phase float64) *decad.Body {
	if b.err != nil {
		return nil
	}

	scale := b.p.Scale
	travel := b.p.BrakeTravel
	h := 216*scale - 13.4 - travel/2
	path, err := decad.NewPath(r3.Vec{},
		decad.LineTo{End: r3.Vec{Z: h}},
		decad.ArcAround{
			Center: r3.Vec{Y: -5, Z: h},
			Axis:   r3.Vec{X: 1},
			End:    r3.Vec{Y: -5, Z: h + 5},
		},
		decad.LineTo{End: r3.Vec{Y: -90 * scale, Z: h + 5}},
		decad.ArcAround{
			Center: r3.Vec{X: radius, Y: -90 * scale, Z: h + 5},
			Axis:   r3.Vec{Z: 1},
			End:    r3.Vec{X: 2 * radius, Y: -90 * scale, Z: h + 5},
		},
		decad.LineTo{End: r3.Vec{X: 2 * radius, Y: -5, Z: h + 5}},
		decad.ArcAround{
			Center: r3.Vec{X: 2 * radius, Y: -5, Z: h},
			Axis:   r3.Vec{X: -1},
			End:    r3.Vec{X: 2 * radius, Z: h},
		},
		decad.LineTo{End: r3.Vec{X: 2 * radius, Z: -travel}},
	)
	if err != nil {
		b.fail("create brake cord path", err)
		return nil
	}

	s := b.section(false)
	circle(s, 0, 0, 0.6)
	p := b.profile(s, 0)
	if b.err != nil {
		return nil
	}
	cord, err := decad.Sweep(b.ctx, b.doc, s, p, path)
	if err != nil {
		b.fail("sweep brake cord", err)
		return nil
	}
	cord = b.place(cord, frame(
		r3.Vec{X: left, Y: leftEnd + 8.4, Z: z},
		r3.Vec{X: 1}, r3.Vec{Z: -1},
	))
	if b.err != nil {
		return nil
	}

	leftCentre := r3.Vec{X: left, Y: leftEnd + 6, Z: z}
	leftEye := b.brakeEye(2.4, 0.6, leftCentre, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if b.err != nil {
		return nil
	}
	rightCentre := r3.Vec{X: right, Y: rightEnd + 6, Z: z}
	rightEye := b.brakeEye(2.4, 0.6, rightCentre, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if b.err != nil {
		return nil
	}
	rightEye = b.brakeRotate(rightEye, rightCentre, phase)
	return b.refine(b.union(cord, leftEye, rightEye), 0.12)
}
