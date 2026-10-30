package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) headPoint(x, y, z, angle float64) r3.Vec {
	angle = math.Remainder(angle, 2*math.Pi)
	c, s := math.Cos(angle), math.Sin(angle)
	return r3.Vec{
		X: b.a.State.Flyer.X + x*c - z*s,
		Y: y + b.a.State.HeadShift,
		Z: b.a.State.Flyer.Z + x*s + z*c,
	}
}

func (b *builder) head() {
	s := b.p.Scale
	state := b.a.State
	angle := math.Remainder(state.FlyerAngle, 2*math.Pi)
	bobbinAngle := math.Remainder(state.BobbinAngle, 2*math.Pi)
	local := frame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Y: 1})

	b.part("flyer-shaft", "shaft", "flyer", func() *decad.Body {
		body := b.revolve([]point2{
			{4, 0}, {4, 213 * s}, {8, 213 * s}, {8, 288 * s},
			{5, 288 * s}, {5, 216 * s}, {0, 216 * s}, {0, 0},
		}, local)
		body = b.cut(body, b.span(r3.Vec{Z: 235 * s}, r3.Vec{Y: 10, Z: 235 * s}, 3))
		frontCap := b.box(6, 9, -9, 9, 216*s, 230*s)
		rearCap := b.box(3, 5, -5, 5, 0, 37*s)
		body = b.flatShaft(body, frontCap, rearCap)
		return b.place(body, shaftFrame(state.Flyer.X, 8*s+state.HeadShift, state.Flyer.Z, angle))
	})

	b.part("flyer-drive-whorl", "whorl", "flyer", func() *decad.Body {
		ratios := [...]float64{17, 12, 9, 7}
		roots := [4]float64{}
		for i, ratio := range ratios {
			roots[i] = state.WheelPitch/ratio - math.Sqrt(5)
		}
		profile := []point2{{0, 0}, {roots[0] + 6*s, 0}, {roots[0] + 6*s, s}}
		for i, root := range roots {
			centre := float64(4+8*i) * s
			profile = append(profile, point2{root, centre}, point2{root + 6*s, centre + 3*s})
			if i+1 < len(roots) {
				profile = append(profile,
					point2{roots[i+1] + 6*s, centre + 3*s},
					point2{roots[i+1] + 6*s, centre + 5*s})
			}
		}
		profile = append(profile, point2{roots[3] + 6*s, 32 * s}, point2{0, 32 * s})
		body := b.revolve(profile, local)
		body = b.cut(body, b.dTool(4.1, 3.1, 32*s, local))
		return b.place(body, shaftFrame(state.Flyer.X, 4*s+state.HeadShift, state.Flyer.Z, angle))
	})

	b.part("flyer-crossbar", "flyer", "flyer", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -74*s, -16*s, 74*s, 16*s)
		dOutline(section, 8.1, 6.1)
		return b.extrude(section, 1, 10*s,
			shaftFrame(state.Flyer.X, -210*s+state.HeadShift, state.Flyer.Z, angle))
	})
	for _, arm := range []struct {
		name   string
		x0, x1 float64
	}{
		{"flyer-arm-1", -74, -56},
		{"flyer-arm-2", 56, 74},
	} {
		b.part(arm.name, "flyer", "flyer", func() *decad.Body {
			return b.panel(arm.x0*s, -10*s, arm.x1*s, 10*s, 135*s,
				shaftFrame(state.Flyer.X, -85*s+state.HeadShift, state.Flyer.Z, angle))
		})
	}

	for _, post := range []struct {
		name          string
		back, width   float64
		bearingRadius float64
	}{
		{"front-maiden-post", -231, 28, 13},
		{"rear-maiden-post", -30, 24, 11},
	} {
		b.part(post.name, "frame", "fixed", func() *decad.Body {
			return b.panel(-20*s, -80*s, 20*s, 25*s, post.width*s,
				shaftFrame(state.Flyer.X, post.back*s+state.HeadShift, state.Flyer.Z, 0),
				hole{0, 0, post.bearingRadius})
		})
	}

	b.part("bobbin", "bobbin", "bobbin", func() *decad.Body {
		return b.revolve([]point2{
			{11, 0}, {45 * s, 0}, {45 * s, 3.5 * s}, {42 * s, 5 * s},
			{45 * s, 6.5 * s}, {45 * s, 10 * s}, {17 * s, 10 * s},
			{17 * s, 130 * s}, {45 * s, 130 * s}, {45 * s, 140 * s},
			{11, 140 * s}, {11, 130 * s}, {6, 130 * s}, {6, 10 * s}, {11, 10 * s},
		}, shaftFrame(state.Flyer.X, -64*s+state.HeadShift, state.Flyer.Z, bobbinAngle))
	})
	b.part("bobbin-yarn-pack", "yarn-pack", "bobbin", func() *decad.Body {
		return b.annulus(17*s, 31*s, 112*s,
			shaftFrame(state.Flyer.X, -78*s+state.HeadShift, state.Flyer.Z, bobbinAngle))
	})
	b.part("bobbin-witness-bar", "marker", "bobbin", func() *decad.Body {
		return b.panel(29*s, -2*s, 33*s, 2*s, 8*s,
			shaftFrame(state.Flyer.X, -130*s+state.HeadShift, state.Flyer.Z, bobbinAngle))
	})

	for _, sleeve := range []struct {
		name, group, motion string
		inner, outer        float64
		front, back         float64
	}{
		{"front-maiden-nylon-sleeve", "bush", "fixed", 8.1, 13, -245*s - 10, -245*s + 10},
		{"rear-maiden-608", "bearing", "fixed", 4, 11, -48 * s, -48*s + 7},
		{"bobbin-front-608", "bearing", "fixed", 4, 11, -194*s - 7, -194 * s},
		{"bobbin-rear-608", "bearing", "fixed", 4, 11, -74 * s, -74*s + 7},
		{"bobbin-inner-race-sleeve", "spacer", "flyer", 4, 5.5, -194 * s, -74 * s},
		{"bobbin-front-thrust-sleeve", "spacer", "flyer", 4, 6, -205*s + 0.2, -194*s - 7 - 0.2},
		{"bobbin-rear-retaining-collar", "collar", "flyer", 4, 6, -74*s + 7 + 0.2, -74*s + 7 + 3.2},
		{"rear-flyer-front-collar", "collar", "flyer", 4, 6, -54*s - 3.2, -54*s - 0.2},
		{"rear-flyer-front-thrust-spacer", "spacer", "flyer", 4, 6, -54*s - 0.2, -48 * s},
		{"rear-flyer-rear-thrust-spacer", "spacer", "flyer", 4, 6, -48*s + 7, -28 * s},
		{"rear-flyer-whorl-retaining-collar", "collar", "flyer", 4, 6, 4*s + 0.1, 4*s + 3.1},
		{"front-sleeve-retainer", "collar", "flyer", 8, 10, -259*s - 2.2, -259*s - 0.2},
	} {
		b.part(sleeve.name, sleeve.group, sleeve.motion, func() *decad.Body {
			phase := angle
			if sleeve.motion == "fixed" {
				phase = 0
			}
			return b.annulus(sleeve.inner, sleeve.outer, sleeve.back-sleeve.front,
				shaftFrame(state.Flyer.X, sleeve.back+state.HeadShift, state.Flyer.Z, phase))
		})
	}
}
