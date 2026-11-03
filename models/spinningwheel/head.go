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

// fixedRing is a round ring on the flyer axis, on the head lift, from head Y y0 to y1.
func (b *builder) fixedRing(name, group, motion string, inner, outer, y0, y1 float64) {
	hl := b.headLift()
	// The length is taken between the shifted ends, the planes' own Y values: neighbouring rings that share an end
	// then meet at exactly one level.
	shift := b.a.State.HeadShift
	b.liftedPart(name, group, motion, func() *decad.Body {
		return b.liftAnnulus(hl, b.headAt(y1), inner, outer, (y1+shift)-(y0+shift))
	})
}

func (b *builder) head() {
	s := b.p.Scale
	ba := b.a.State.BobbinAngle
	fl := b.flyerLift()

	// The shaft and its tools are placed on the flyer lift before they are cut, so the cuts stay readable.
	// The shaft starts at head Y -26s, local level 26s.
	b.liftedPart("flyer-shaft", "shaft", "flyer", func() *decad.Body {
		// The shaft's revolve is round, so it rides the fixed head lift with its rings and bearings, on one
		// exact axis line; only its port and flats are cut by tools on the flyer lift.
		z := 26 * s
		o := b.headAt(-z).Origin()
		body := b.liftRevolve(b.headLift(), frame(o, r3.Vec{X: 1}, r3.Vec{Y: -1}), []point2{
			{0, 0}, {4, 0}, {4, 179 * s}, {8, 179 * s},
			{8, 194 * s}, {10, 194 * s}, {10, 196 * s}, {8, 196 * s},
			{8, 309 * s}, {5, 309 * s}, {5, 182 * s}, {0, 182 * s},
		})
		port := b.liftCylinder(fl, frame(r3.Vec{Z: z + 201*s}, r3.Vec{Z: 1}, r3.Vec{X: 1}), 3, 9)
		body = b.cut(body, port)
		return b.flatShaft(body,
			b.localStock(fl, 7.2, 9, -1.5, 1.5, z+225*s, z+255*s),
			b.localStock(fl, 7.2, 9, -1.5, 1.5, z+185*s, z+193*s))
	})

	for _, key := range []struct {
		name   string
		y0, y1 float64
	}{
		{"flyer-whorl-key", -281 * s, -251 * s},
		{"flyer-crossbar-key", -219 * s, -211 * s},
	} {
		b.liftedPart(key.name, "key", "flyer", func() *decad.Body {
			return b.liftPanel(fl, level(0, 0, -key.y1), 7.2, -1.5, 8.7, 1.5, key.y1-key.y0)
		})
	}

	b.liftedPart("flyer-drive-whorl", "whorl", "flyer", func() *decad.Body {
		profile := []point2{{0, 0}}
		for i, ratio := range []float64{17, 12, 9, 7} {
			root := b.a.State.WheelPitch/ratio - math.Sqrt(5)
			outer := root + 6*s
			// Every axial station is an integer multiple of s, so one groove's end and the next groove's
			// start are the same float. Computed as centre+3*s versus (8i-1)*s they differ by an ulp at
			// s=0.8, which folds the outer run and keeps Decad off its exact revolved-bore route.
			z := func(k int) float64 { return float64(k) * s }
			start := z(8*i - 1)
			if i == 0 {
				start = 0
			}
			profile = append(profile, point2{outer, start}, point2{outer, z(8*i + 1)},
				point2{root, z(8*i + 4)}, point2{outer, z(8*i + 7)})
			if i == 3 {
				profile = append(profile, point2{outer, 32 * s})
			}
		}
		profile = append(profile, point2{0, 32 * s})
		// The whorl starts at head Y -250s, local level 250s. It rides the flyer wrap's lift, and its keyway is
		// sketched turned onto the flyer's angle in the same plane, so revolve and keyway share one placement.
		wl := b.whorlLift()
		body := b.liftRevolve(wl, zAxis(0, 0, 250*s), profile)
		key := b.sectionOn(level(0, 0, 250*s))
		err := turnedKeyedOutline(key, 8.1, 1.6, 8.8, math.Remainder(b.a.State.FlyerAngle-b.whorlStart(), 2*math.Pi))
		b.fail("turn whorl keyway", err)
		return b.cut(body, b.liftExtrude(key, 0, 32*s, wl))
	})

	b.liftedPart("flyer-crossbar", "flyer", "flyer", func() *decad.Body {
		section := b.sectionOn(level(0, 0, 210*s))
		rectangle(section, -74*s, -16*s, 74*s, 16*s)
		keyedOutline(section, 8.1, 1.6, 8.8)
		return b.liftExtrude(section, 1, 10*s, fl)
	})
	for i, bounds := range [][2]float64{{-74, -56}, {56, 74}} {
		name := "flyer-arm-1"
		if i == 1 {
			name = "flyer-arm-2"
		}
		b.liftedPart(name, "flyer", "flyer", func() *decad.Body {
			return b.liftPanel(fl, level(0, 0, 85*s), bounds[0]*s, -10*s, bounds[1]*s, 10*s, 135*s)
		})
	}

	hl := b.headLift()
	b.liftedPart("front-maiden", "frame", "fixed", func() *decad.Body {
		return b.liftPanel(hl, b.headAt(-286*s), -20*s, -115*s, 20*s, 25*s, 28*s, hole{r: 13})
	})
	b.liftedPart("rear-maiden", "frame", "fixed", func() *decad.Body {
		// The bore is stock radius9 with the bearing's radius11 seat cut from the bearing's front face out past the
		// maiden's rear face, leaving a shoulder on the outer race's front face.
		// The seat's length is taken between the shifted planes, so its shoulder lands on the bearing's front face.
		shift := b.a.State.HeadShift
		body := b.liftPanel(hl, b.headAt(-30*s), -20*s, -115*s, 20*s, 25*s, 24*s, hole{r: 9})
		return b.cut(body, b.liftCylinder(hl, b.headAt(-30*s+5), 11, (-30*s+5+shift)-(-48*s+shift)))
	})
	b.fixedRing("front-maiden-bush", "bush", "fixed", 8.1, 13, -300*s-10, -300*s+10)
	b.fixedRing("rear-flyer-bearing", "bearing", "fixed", 4, 11, -48*s, -48*s+7)
	// A stock retaining ring in the seat holds the outer race's rear face.
	b.fixedRing("rear-flyer-bearing-ring", "retainer", "fixed", 9, 11, -48*s+7, -48*s+8)
	// The flyer's rings are round, so they ride the fixed head lift with the bearings they bear on.
	b.fixedRing("rear-flyer-front-collar", "collar", "flyer", 4, 6, -54*s-3.2, -54*s-0.2)
	b.fixedRing("rear-flyer-front-thrust-spacer", "spacer", "flyer", 4, 6, -54*s-0.2, -48*s)
	b.fixedRing("rear-flyer-rear-thrust-spacer", "spacer", "flyer", 4, 6, -48*s+7, -30*s)
	b.fixedRing("rear-flyer-rear-collar", "collar", "flyer", 4, 6, -30*s+0.2, -30*s+3.2)
	b.fixedRing("front-sleeve-retainer", "collar", "flyer", 8, 10, -314*s-4.2, -314*s-0.2)
	b.fixedRing("whorl-front-collar", "collar", "flyer", 8, 10, -282*s-4.2, -282*s-0.2)
	b.fixedRing("whorl-rear-collar", "collar", "flyer", 8, 10, -250*s+0.2, -250*s+4.2)
	b.fixedRing("crossbar-rear-collar", "collar", "flyer", 8, 10, -210*s+0.2, -210*s+4.2)

	// The bobbin and its yarn pack are round, so they ride the fixed head lift with their bearings. The bobbin
	// turns about the flyer axis, from head Y -64s toward -Y.
	b.liftedPart("bobbin", "bobbin", "bobbin", func() *decad.Body {
		o := b.headAt(-64 * s).Origin()
		return b.liftRevolve(hl, frame(o, r3.Vec{X: 1}, r3.Vec{Y: -1}), []point2{
			{11, 0}, {45 * s, 0}, {45 * s, 3.5 * s}, {42 * s, 5 * s},
			{45 * s, 6.5 * s}, {45 * s, 10 * s}, {17 * s, 10 * s},
			{17 * s, 130 * s}, {45 * s, 130 * s}, {45 * s, 140 * s},
			{11, 140 * s}, {11, 130 * s}, {6, 130 * s}, {6, 10 * s}, {11, 10 * s},
		})
	})
	b.fixedRing("bobbin-front-bearing", "bearing", "fixed", 4, 11, -194*s-7, -194*s)
	b.fixedRing("bobbin-rear-bearing", "bearing", "fixed", 4, 11, -74*s, -74*s+7)
	b.fixedRing("bobbin-inner-race-sleeve", "spacer", "flyer", 4, 5.5, -194*s, -74*s)
	b.fixedRing("bobbin-front-thrust-sleeve", "spacer", "flyer", 4, 6, -205*s+0.2, -194*s-7-0.2)
	b.fixedRing("bobbin-rear-removable-collar", "collar", "flyer", 4, 6, -74*s+7.2, -74*s+10.2)
	b.fixedRing("bobbin-yarn-pack", "yarn-pack", "bobbin", 17*s, 31*s-0.35, -190*s, -78*s)
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
