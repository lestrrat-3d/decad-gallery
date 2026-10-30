package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) yarn() {
	if b.err != nil {
		return
	}
	s := b.p.Scale
	angle := math.Remainder(b.a.State.FlyerAngle, 2*math.Pi)
	point := func(p r3.Vec) r3.Vec {
		return b.headPoint(p.X, p.Y, p.Z, angle)
	}
	span := func(name, motion string, a, c r3.Vec) {
		b.part(name, "yarn", motion, func() *decad.Body {
			return b.span(point(a), point(c), 0.35)
		})
	}
	start := r3.Vec{Y: -350 * s}
	entrance := r3.Vec{Y: -280 * s}
	port := r3.Vec{Y: -227 * s}
	outlet := r3.Vec{Y: -227 * s, Z: 22 * s}
	front := r3.Vec{X: 58 * s, Y: -224 * s, Z: 22 * s}
	active := r3.Vec{X: 58 * s, Y: -160 * s, Z: 22 * s}
	span("yarn-incoming", "fixed", start, entrance)
	span("yarn-orifice", "fixed", entrance, port)
	span("yarn-outlet", "flyer", port, outlet)
	span("yarn-front-leader", "flyer", outlet, front)
	span("yarn-arm-leader", "flyer", front, active)

	p := r3.Vec{X: 58 * s, Z: 22 * s}
	radius := 31 * s
	p2 := p.Dot(p)
	tangent := p.Scale(radius * radius / p2).Sub(
		r3.Vec{X: -p.Z, Z: p.X}.Scale(radius * math.Sqrt(p2-radius*radius) / p2),
	)
	tangent.Y = active.Y
	span("yarn-pack-strand", "flyer", active, tangent)

	outgoing := tangent.Sub(active)
	outgoing = outgoing.Scale(1 / outgoing.Len())
	activeAxis := outgoing.Add(r3.Vec{Y: 1})
	activeAxis = activeAxis.Scale(1 / activeAxis.Len())
	for _, side := range []struct {
		name string
		x    float64
	}{
		{name: "negative", x: -58 * s},
		{name: "positive", x: 58 * s},
	} {
		for _, y := range []float64{-190, -160, -130, -100} {
			axis := r3.Vec{Y: 1}
			if side.x > 0 && y == -160 {
				axis = activeAxis
			}
			name := fmt.Sprintf("yarn-guide-%s-%.0f", side.name, -y)
			centre := r3.Vec{X: side.x, Y: y * s, Z: 22 * s}
			b.yarnGuide(name, centre, axis, 10*s, angle, false)
		}
	}
	frontIncoming := r3.Vec{X: 58, Y: 3}
	frontIncoming = frontIncoming.Scale(1 / frontIncoming.Len())
	frontAxis := frontIncoming.Add(r3.Vec{Y: 1})
	frontAxis = frontAxis.Scale(1 / frontAxis.Len())
	b.yarnGuide("yarn-guide-front", front, frontAxis, 12*s, angle, true)
}

func (b *builder) yarnGuide(name string, centre, axis r3.Vec, bottomZ, angle float64, front bool) {
	if b.err != nil {
		return
	}
	origin := b.headPoint(0, 0, 0, angle)
	worldCentre := b.headPoint(centre.X, centre.Y, centre.Z, angle)
	worldAxis := b.headPoint(axis.X, axis.Y, axis.Z, angle).Sub(origin)
	b.part(name, "yarn", "flyer", func() *decad.Body {
		return b.annulus(3, 5, 2, axisFrame(worldCentre.Sub(worldAxis), worldAxis))
	})
	down := r3.Vec{Z: -1}
	bottomDirection := down.Sub(axis.Scale(down.Dot(axis)))
	bottomDirection = bottomDirection.Scale(1 / bottomDirection.Len())
	attachment := centre.Add(bottomDirection.Scale(4))
	lower := r3.Vec{X: attachment.X, Y: attachment.Y, Z: bottomZ}
	b.part(name+"-stem", "yarn", "flyer", func() *decad.Body {
		return b.span(
			b.headPoint(lower.X, lower.Y, lower.Z, angle),
			b.headPoint(attachment.X, attachment.Y, attachment.Z, angle),
			1,
		)
	})
	if !front {
		return
	}
	stalkEnd := r3.Vec{X: lower.X, Y: -215 * b.p.Scale, Z: lower.Z}
	b.part(name+"-stalk", "yarn", "flyer", func() *decad.Body {
		return b.span(
			b.headPoint(lower.X, lower.Y, lower.Z, angle),
			b.headPoint(stalkEnd.X, stalkEnd.Y, stalkEnd.Z, angle),
			1,
		)
	})
}
