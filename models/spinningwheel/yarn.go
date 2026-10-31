package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) yarnHook(name string, centre, axis r3.Vec, bottomZ, phase float64) r3.Vec {
	if b.err != nil {
		return r3.Vec{}
	}
	n, ok := axis.Normalize()
	if !ok {
		b.fail("orient "+name, fmt.Errorf("invalid hook axis"))
		return r3.Vec{}
	}
	down := r3.Vec{Z: -1}
	e, ok := down.Sub(n.Scale(down.Dot(n))).Normalize()
	if !ok {
		b.fail("orient "+name, fmt.Errorf("hook plane has no downward radial direction"))
		return r3.Vec{}
	}
	attachment := centre.Add(e.Scale(4))
	bottom := r3.Vec{X: attachment.X, Y: attachment.Y, Z: bottomZ}
	b.part(name, "guide", "flyer", func() *decad.Body {
		hook := b.torus(4, 1, -5*math.Pi/6, 5*math.Pi/3, r3.Vec{})
		return b.place(hook, frame(
			b.headPoint(centre.X, centre.Y, centre.Z, phase),
			headDirection(e, phase), headDirection(n.Scale(-1), phase)))
	})
	b.part(name+"-stem", "guide", "flyer", func() *decad.Body {
		return b.span(
			b.headPoint(bottom.X, bottom.Y, bottom.Z, phase),
			b.headPoint(attachment.X, attachment.Y, attachment.Z, phase), 1)
	})
	return bottom
}

func (b *builder) yarn() {
	if b.err != nil {
		return
	}
	s, phase := b.p.Scale, b.a.State.FlyerAngle
	selected := r3.Vec{X: 58 * s, Y: -160 * s, Z: 22 * s}
	radius := 31 * s
	distance2 := selected.X*selected.X + selected.Z*selected.Z
	radial := radius * radius / distance2
	tangent := radius * math.Sqrt(distance2-radius*radius) / distance2
	contact := r3.Vec{
		X: radial*selected.X + tangent*selected.Z,
		Y: selected.Y,
		Z: radial*selected.Z - tangent*selected.X,
	}
	outgoing, ok := contact.Sub(selected).Normalize()
	if !ok {
		b.fail("route yarn leader", fmt.Errorf("invalid tangent-to-pack strand"))
		return
	}
	for _, side := range []struct {
		name string
		x    float64
	}{{"negative", -58}, {"positive", 58}} {
		for i, y := range []float64{-190, -172, -160, -142, -124, -106} {
			axis := r3.Vec{Y: 1}
			if side.x > 0 && y == -160 {
				axis = axis.Add(outgoing)
			}
			b.yarnHook(fmt.Sprintf("yarn-hook-%s-%d", side.name, i+1),
				r3.Vec{X: side.x * s, Y: y * s, Z: 22 * s}, axis, 10*s, phase)
		}
	}
	incoming, ok := (r3.Vec{X: 58, Y: 3}).Normalize()
	if !ok {
		b.fail("route front yarn guide", fmt.Errorf("invalid incoming strand"))
		return
	}
	front := r3.Vec{X: 58 * s, Y: -224 * s, Z: 22 * s}
	stalkStart := b.yarnHook("yarn-front-guide", front, incoming.Add(r3.Vec{Y: 1}), 12*s, phase)
	b.part("yarn-front-guide-stalk", "guide", "flyer", func() *decad.Body {
		return b.span(
			b.headPoint(stalkStart.X, stalkStart.Y, stalkStart.Z, phase),
			b.headPoint(stalkStart.X, -215*s, stalkStart.Z, phase), 1)
	})
	waypoints := []r3.Vec{
		{Y: -405 * s},
		{Y: -335 * s},
		{Y: -227 * s},
		{Y: -227 * s, Z: 22 * s},
		front,
		selected,
		contact,
	}
	for i := 0; i+1 < len(waypoints); i++ {
		start, end := waypoints[i], waypoints[i+1]
		b.part(fmt.Sprintf("yarn-leader-%d", i+1), "yarn", "flyer", func() *decad.Body {
			return b.span(
				b.headPoint(start.X, start.Y, start.Z, phase),
				b.headPoint(end.X, end.Y, end.Z, phase), 0.35)
		})
	}
}
