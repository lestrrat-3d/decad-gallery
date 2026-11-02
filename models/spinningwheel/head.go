package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

func headDirection(v r3.Vec, angle float64) r3.Vec {
	angle = math.Remainder(angle, 2*math.Pi)
	c, s := math.Cos(angle), math.Sin(angle)
	return r3.Vec{X: c*v.X - s*v.Z, Y: v.Y, Z: s*v.X + c*v.Z}
}

func (b *builder) headPoint(x, y, z, angle float64) r3.Vec {
	s := b.p.Scale
	v := headDirection(r3.Vec{X: x, Y: y, Z: z}, angle)
	return v.Add(r3.Vec{X: -140 * s, Y: b.a.State.HeadShift, Z: 690 * s})
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
	a := b.a.State.FlyerAngle
	ba := b.a.State.BobbinAngle
	local := frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})

	b.part("flyer-shaft", "shaft", "flyer", func() *decad.Body {
		body := b.revolve([]point2{
			{0, 0}, {4, 0}, {4, 179 * s}, {8, 179 * s},
			{8, 194 * s}, {10, 194 * s}, {10, 196 * s}, {8, 196 * s},
			{8, 309 * s}, {5, 309 * s}, {5, 182 * s}, {0, 182 * s},
		}, local)
		port := b.cylinder(3, 9, axisFrame(r3.Vec{Z: 201 * s}, r3.Vec{Y: 1}))
		body = b.cut(body, port)
		body = b.flatShaft(body,
			b.box(7.2, 9, -1.5, 1.5, 225*s, 255*s),
			b.box(7.2, 9, -1.5, 1.5, 185*s, 193*s))
		return b.place(body, b.headFrame(-26*s, a))
	})

	for _, key := range []struct {
		name   string
		y0, y1 float64
	}{
		{"flyer-whorl-key", -281 * s, -251 * s},
		{"flyer-crossbar-key", -219 * s, -211 * s},
	} {
		b.part(key.name, "key", "flyer", func() *decad.Body {
			return b.panel(7.2, -1.5, 8.7, 1.5, key.y1-key.y0, b.headFrame(key.y1, a))
		})
	}

	b.part("flyer-drive-whorl", "whorl", "flyer", func() *decad.Body {
		profile := []point2{{0, 0}}
		for i, ratio := range []float64{17, 12, 9, 7} {
			root := b.a.State.WheelPitch/ratio - math.Sqrt(5)
			outer := root + 6*s
			centre := float64(4+8*i) * s
			start := float64(8*i-1) * s
			if i == 0 {
				start = 0
			}
			profile = append(profile, point2{outer, start}, point2{outer, centre - 3*s},
				point2{root, centre}, point2{outer, centre + 3*s})
			if i == 3 {
				profile = append(profile, point2{outer, 32 * s})
			}
		}
		profile = append(profile, point2{0, 32 * s})
		body := b.revolve(profile, local)
		body = b.cut(body, b.keyTool(8.1, 1.6, 8.8, 32*s, local))
		return b.place(body, b.headFrame(-250*s, a))
	})

	b.part("flyer-crossbar", "flyer", "flyer", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -74*s, -16*s, 74*s, 16*s)
		keyedOutline(section, 8.1, 1.6, 8.8)
		return b.extrude(section, 1, 10*s, b.headFrame(-210*s, a))
	})
	for i, bounds := range [][2]float64{{-74, -56}, {56, 74}} {
		name := "flyer-arm-1"
		if i == 1 {
			name = "flyer-arm-2"
		}
		b.part(name, "flyer", "flyer", func() *decad.Body {
			return b.panel(bounds[0]*s, -10*s, bounds[1]*s, 10*s, 135*s, b.headFrame(-85*s, a))
		})
	}

	b.part("front-maiden", "frame", "fixed", func() *decad.Body {
		return b.panel(-20*s, -115*s, 20*s, 25*s, 28*s, b.headFrame(-286*s, 0), hole{r: 13})
	})
	b.part("rear-maiden", "frame", "fixed", func() *decad.Body {
		return b.panel(-20*s, -115*s, 20*s, 25*s, 24*s, b.headFrame(-30*s, 0), hole{r: 11})
	})
	b.headRing("front-maiden-bush", "bush", "fixed", 8.1, 13, -300*s-10, -300*s+10, 0)
	b.headRing("rear-flyer-bearing", "bearing", "fixed", 4, 11, -48*s, -48*s+7, 0)
	b.headRing("rear-flyer-front-collar", "collar", "flyer", 4, 6, -54*s-3.2, -54*s-0.2, a)
	b.headRing("rear-flyer-front-thrust-spacer", "spacer", "flyer", 4, 6, -54*s-0.2, -48*s, a)
	b.headRing("rear-flyer-rear-thrust-spacer", "spacer", "flyer", 4, 6, -48*s+7, -30*s, a)
	b.headRing("rear-flyer-rear-collar", "collar", "flyer", 4, 6, -30*s+0.2, -30*s+3.2, a)
	b.headRing("front-sleeve-retainer", "collar", "flyer", 8, 10, -314*s-4.2, -314*s-0.2, a)
	b.headRing("whorl-front-collar", "collar", "flyer", 8, 10, -282*s-4.2, -282*s-0.2, a)
	b.headRing("whorl-rear-collar", "collar", "flyer", 8, 10, -250*s+0.2, -250*s+4.2, a)
	b.headRing("crossbar-rear-collar", "collar", "flyer", 8, 10, -210*s+0.2, -210*s+4.2, a)

	b.part("bobbin", "bobbin", "bobbin", func() *decad.Body {
		return b.revolve([]point2{
			{11, 0}, {45 * s, 0}, {45 * s, 3.5 * s}, {42 * s, 5 * s},
			{45 * s, 6.5 * s}, {45 * s, 10 * s}, {17 * s, 10 * s},
			{17 * s, 130 * s}, {45 * s, 130 * s}, {45 * s, 140 * s},
			{11, 140 * s}, {11, 130 * s}, {6, 130 * s}, {6, 10 * s}, {11, 10 * s},
		}, b.headFrame(-64*s, ba))
	})
	b.headRing("bobbin-front-bearing", "bearing", "fixed", 4, 11, -194*s-7, -194*s, 0)
	b.headRing("bobbin-rear-bearing", "bearing", "fixed", 4, 11, -74*s, -74*s+7, 0)
	b.headRing("bobbin-inner-race-sleeve", "spacer", "flyer", 4, 5.5, -194*s, -74*s, a)
	b.headRing("bobbin-front-thrust-sleeve", "spacer", "flyer", 4, 6, -205*s+0.2, -194*s-7-0.2, a)
	b.headRing("bobbin-rear-removable-collar", "collar", "flyer", 4, 6, -74*s+7.2, -74*s+10.2, a)
	b.headRing("bobbin-yarn-pack", "yarn-pack", "bobbin", 17*s, 31*s-0.35, -190*s, -78*s, ba)
	b.part("bobbin-witness-bar", "marker", "bobbin", func() *decad.Body {
		radius := 31*s - 0.35
		half := math.Asin(2 * s / radius)
		section := b.section(true)
		rectangle(section, radius, 0, radius+0.2, 8*s)
		profile := b.profile(section, 0)
		if b.err != nil {
			return nil
		}
		body, err := decad.Revolve(b.ctx, b.doc, section, profile,
			decad.ConstructionAxis{Dir: r3.Vec{Z: 1}},
			decad.AngleExtent{A: units.Radians(2 * half), Dir: decad.Along})
		b.fail("revolve bobbin winding label", err)
		if b.err != nil {
			return nil
		}
		return b.place(body, b.headFrame(-130*s, ba+math.Pi/2-half))
	})
}
