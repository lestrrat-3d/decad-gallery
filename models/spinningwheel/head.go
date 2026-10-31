package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func headDirection(v r3.Vec, angle float64) r3.Vec {
	angle = math.Remainder(angle, 2*math.Pi)
	c, s := math.Cos(angle), math.Sin(angle)
	return r3.Vec{X: c*v.X - s*v.Z, Y: v.Y, Z: s*v.X + c*v.Z}
}

func (b *builder) headPoint(x, y, z, angle float64) r3.Vec {
	s := b.p.Scale
	return headDirection(r3.Vec{X: x, Y: y, Z: z}, angle).
		Add(r3.Vec{X: -140 * s, Y: b.a.State.HeadShift, Z: 690 * s})
}

func (b *builder) headFrame(y, angle float64) r3.Frame {
	s := b.p.Scale
	return shaftFrame(-140*s, y+b.a.State.HeadShift, 690*s, math.Remainder(angle, 2*math.Pi))
}

func (b *builder) headRing(name, group, motion string, inner, outer, y0, y1, angle float64) {
	b.part(name, group, motion, func() *decad.Body {
		return b.annulus(inner, outer, y1-y0, b.headFrame(y1, angle))
	})
}

func (b *builder) head() {
	s := b.p.Scale
	flyer, bobbin := b.a.State.FlyerAngle, b.a.State.BobbinAngle

	b.part("flyer-shaft", "shaft", "flyer", func() *decad.Body {
		body := b.revolve([]point2{
			{4, 0}, {4, 179 * s}, {8, 179 * s}, {8, 309 * s},
			{5, 309 * s}, {5, 182 * s}, {0, 182 * s}, {0, 0},
		}, frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1}))
		// The unplaced shaft's radial +Y becomes reference +Z.
		port := b.cylinder(3, 9, axisFrame(r3.Vec{Z: 201 * s}, r3.Vec{Y: 1}))
		body = b.cut(body, port)
		body = b.flatShaft(body,
			b.box(6, 9, -9, 9, 182*s, 196*s),
			b.box(6, 9, -9, 9, 223*s, 257*s))
		return b.place(body, b.headFrame(-26*s, flyer))
	})

	b.part("flyer-drive-whorl", "whorl", "flyer", func() *decad.Body {
		outer := make([]float64, 4)
		root := make([]float64, 4)
		for i, ratio := range []float64{17, 12, 9, 7} {
			root[i] = b.a.State.WheelPitch/ratio - math.Sqrt(5)
			outer[i] = root[i] + 6*s
		}
		profile := []point2{{0, 0}, {outer[0], 0}}
		for i := range outer {
			centre := (4 + 8*float64(i)) * s
			profile = append(profile,
				point2{outer[i], centre - 3*s},
				point2{root[i], centre},
				point2{outer[i], centre + 3*s})
			if i+1 < len(outer) {
				profile = append(profile, point2{outer[i+1], centre + 3*s})
			}
		}
		profile = append(profile, point2{outer[3], 32 * s}, point2{0, 32 * s})
		local := frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		body := b.revolve(profile, local)
		body = b.cut(body, b.dTool(8.1, 6.1, 32*s, local))
		return b.place(body, b.headFrame(-250*s, flyer))
	})

	b.part("flyer-crossbar", "flyer", "flyer", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -74*s, -16*s, 74*s, 16*s)
		dOutline(section, 8.1, 6.1)
		return b.extrude(section, 1, 10*s, b.headFrame(-210*s, flyer))
	})
	for i, limits := range [][2]float64{{-74, -56}, {56, 74}} {
		name := "flyer-arm-1"
		if i == 1 {
			name = "flyer-arm-2"
		}
		b.part(name, "flyer", "flyer", func() *decad.Body {
			return b.panel(limits[0]*s, -10*s, limits[1]*s, 10*s, 135*s, b.headFrame(-85*s, flyer))
		})
	}

	b.part("front-maiden", "frame", "fixed", func() *decad.Body {
		return b.panel(-20*s, -105*s, 20*s, 25*s, 28*s,
			b.headFrame(-286*s, 0), hole{0, 0, 13})
	})
	b.part("rear-maiden", "frame", "fixed", func() *decad.Body {
		return b.panel(-20*s, -105*s, 20*s, 25*s, 24*s,
			b.headFrame(-30*s, 0), hole{0, 0, 11})
	})
	b.headRing("front-maiden-bush", "bush", "fixed", 8.1, 13, -300*s-10, -300*s+10, 0)
	b.headRing("rear-flyer-bearing", "bearing", "fixed", 4, 11, -48*s, -48*s+7, 0)
	b.headRing("rear-flyer-front-collar", "collar", "flyer", 4, 6, -54*s-3.2, -54*s-0.2, flyer)
	b.headRing("rear-flyer-front-thrust-spacer", "spacer", "flyer", 4, 6, -54*s-0.2, -48*s, flyer)
	b.headRing("rear-flyer-rear-thrust-spacer", "spacer", "flyer", 4, 6, -48*s+7, -30*s, flyer)
	b.headRing("rear-flyer-rear-collar", "collar", "flyer", 4, 6, -30*s+0.2, -30*s+3.2, flyer)
	b.headRing("front-sleeve-retainer", "collar", "flyer", 8, 10, -314*s-2.2, -314*s-0.2, flyer)
	b.headRing("whorl-front-collar", "collar", "flyer", 8, 10, -282*s-2.2, -282*s-0.2, flyer)
	b.headRing("whorl-rear-collar", "collar", "flyer", 8, 10, -250*s+0.2, -250*s+2.2, flyer)

	b.part("bobbin", "bobbin", "bobbin", func() *decad.Body {
		return b.revolve([]point2{
			{45 * s, 0}, {45 * s, 3.5 * s}, {42 * s, 5 * s},
			{45 * s, 6.5 * s}, {45 * s, 10 * s},
			{17 * s, 10 * s}, {17 * s, 130 * s},
			{45 * s, 130 * s}, {45 * s, 140 * s},
			{11, 140 * s}, {11, 130 * s}, {6, 130 * s},
			{6, 10 * s}, {11, 10 * s}, {11, 0},
		}, b.headFrame(-64*s, bobbin))
	})
	b.headRing("bobbin-yarn-pack", "yarn-pack", "bobbin", 17*s, 31*s, -190*s, -78*s, bobbin)
	b.part("bobbin-witness-bar", "marker", "bobbin", func() *decad.Body {
		return b.panel(-2*s, 29*s, 2*s, 33*s, 8*s, b.headFrame(-130*s, bobbin))
	})
	b.headRing("bobbin-front-bearing", "bearing", "fixed", 4, 11, -194*s-7, -194*s, 0)
	b.headRing("bobbin-rear-bearing", "bearing", "fixed", 4, 11, -74*s, -74*s+7, 0)
	b.headRing("bobbin-inner-race-sleeve", "spacer", "flyer", 4, 5.5, -194*s, -74*s, flyer)
	b.headRing("bobbin-front-thrust-sleeve", "spacer", "flyer", 4, 6, -205*s+0.2, -194*s-7-0.2, flyer)
	b.headRing("bobbin-rear-removable-collar", "collar", "flyer", 4, 6, -74*s+7+0.2, -74*s+7+3.2, flyer)
}
