package spinningwheel

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

func (b *builder) frame() {
	s := b.p.Scale
	for _, timber := range []struct {
		name                   string
		x0, x1, y0, y1, z0, z1 float64
	}{
		{"bench", -360, 420, 60, 140, 500, 530},
		{"left post", -340, -300, 80, 120, 30, 500},
		{"left foot", -370, -270, -140, 200, 0, 30},
		{"head crossbeam", -300, -220, -100, 100, 500, 530},
		{"wheel foot", 150, 290, -100, 200, 0, 30},
		{"bench support", 200, 240, 100, 140, 30, 500},
		{"heel foot", 35, 125, -270, -110, 0, 30},
		{"connecting rail", 80, 220, -140, -110, 0, 25},
	} {
		b.part(timber.name, "frame", "fixed", func() *decad.Body {
			return b.box(timber.x0*s, timber.x1*s, timber.y0*s, timber.y1*s, timber.z0*s, timber.z1*s)
		})
	}
	for _, post := range []struct {
		name  string
		yBack float64
	}{
		{"front wheel upright", -25 * s},
		{"rear wheel upright", 55 * s},
	} {
		b.part(post.name, "frame", "fixed", func() *decad.Body {
			return b.panel(200*s, 30*s, 240*s, 455*s, 30*s,
				shaftFrame(0, post.yBack, 0, 0), hole{220 * s, 430 * s, 11})
		})
	}
	for _, bearing := range []struct {
		name  string
		yBack float64
	}{
		{"front wheel 608 bearing", -45*s + 7},
		{"rear wheel 608 bearing", 45 * s},
	} {
		b.part(bearing.name, "bearing", "fixed", func() *decad.Body {
			return b.annulus(4, 11, 7, shaftFrame(220*s, bearing.yBack, 430*s, 0))
		})
	}
	for _, ring := range []struct {
		name          string
		yBack, length float64
	}{
		{"front wheel shaft collar", -55 * s, 3},
		{"rear wheel shaft collar", 55*s + 3, 3},
		{"front wheel thrust spacer", -45 * s, 10 * s},
		{"rear wheel thrust spacer", 55 * s, 10 * s},
	} {
		b.part(ring.name, "retainer", "fixed", func() *decad.Body {
			return b.annulus(4, 6, ring.length, shaftFrame(220*s, ring.yBack, 430*s, 0))
		})
	}
}

func (b *builder) adjustment() {
	s := b.p.Scale
	tension := b.p.DriveTension
	headX := b.a.State.Flyer.X
	shift := b.a.State.HeadShift
	brakeX := headX + 42*s + 0.6*math.Sqrt(5)
	brakeY := -69*s + shift
	positiveX := func(x float64) r3.Frame {
		return frame(r3.Vec{X: x}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	}
	positiveZ := func(z float64) r3.Frame {
		return frame(r3.Vec{Z: z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	}
	b.part("fixed head support", "adjustment", "fixed", func() *decad.Body {
		return b.box(-265*s, -215*s, -90*s, -50*s, 530*s, 560*s)
	})
	b.part("head adjustment base", "adjustment", "fixed", func() *decad.Body {
		return b.box(-340*s, -140*s, -290*s, -2*s, 560*s, 575*s)
	})
	b.part("tension carriage", "adjustment", "fixed", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -280*s, 575*s, -12*s, 595*s)
		circle(section, -260*s, 585*s, 4.3)
		body := b.extrude(section, 1, 170*s, positiveX((-325-tension)*s))
		nutX := (-250 - tension) * s
		pocketFrame := axisFrame(r3.Vec{X: nutX - 4, Y: -260 * s, Z: 585 * s}, r3.Vec{X: 1})
		body = b.cut(body, b.dTool(8.1, 7.1, 8, pocketFrame))
		clearanceFrame := axisFrame(r3.Vec{X: brakeX, Y: brakeY, Z: 575*s - 1}, r3.Vec{Z: 1})
		return b.cut(body, b.cylinder(2.1, 20*s+2, clearanceFrame))
	})
	for _, guide := range []struct {
		name   string
		y0, y1 float64
	}{
		{"front carriage guide", -290 * s, -280*s - 0.2},
		{"rear carriage guide", -12*s + 0.2, -2 * s},
	} {
		b.part(guide.name, "adjustment", "fixed", func() *decad.Body {
			return b.box(-340*s, -140*s, guide.y0, guide.y1, 575*s, 595*s)
		})
	}
	b.part("leadscrew bearing block", "adjustment", "fixed", func() *decad.Body {
		section := b.section(false)
		rectangle(section, -270*s, 560*s, -250*s, 605*s)
		circle(section, -260*s, 585*s, 4.3)
		return b.extrude(section, 1, 15*s, positiveX(-355*s))
	})
	phase := math.Remainder(2*math.Pi*tension*s/1.25, 2*math.Pi)
	screwFrame := func(x float64) r3.Frame {
		return frame(r3.Vec{X: x, Y: -260 * s, Z: 585 * s},
			r3.Vec{Y: math.Cos(phase), Z: math.Sin(phase)},
			r3.Vec{Y: -math.Sin(phase), Z: math.Cos(phase)})
	}
	b.part("M8 tension leadscrew", "thread envelope", "fixed", func() *decad.Body {
		body := b.cylinder(4, 180*s, screwFrame(-380*s))
		cap := b.panel(3, -5, 5, 5, 20*s, screwFrame(-380*s))
		return b.flatShaft(body, cap)
	})
	b.part("tension knob", "adjustment", "fixed", func() *decad.Body {
		return b.dAnnulus(4.1, 3.1, 18*s, 20*s, screwFrame(-380*s))
	})
	b.part("captive M8 tension nut", "thread envelope", "fixed", func() *decad.Body {
		nutX := (-250 - tension) * s
		nutFrame := axisFrame(r3.Vec{X: nutX - 4, Y: -260 * s, Z: 585 * s}, r3.Vec{X: 1})
		body := b.dTool(8, 7, 8, nutFrame)
		boreFrame := axisFrame(r3.Vec{X: nutX - 5, Y: -260 * s, Z: 585 * s}, r3.Vec{X: 1})
		return b.cut(body, b.cylinder(4, 10, boreFrame))
	})
	for _, collar := range []struct {
		name string
		x    float64
	}{
		{"left leadscrew collar", -355*s - 3},
		{"right leadscrew collar", -340 * s},
	} {
		b.part(collar.name, "retainer", "fixed", func() *decad.Body {
			return b.annulus(4, 6, 3,
				axisFrame(r3.Vec{X: collar.x, Y: -260 * s, Z: 585 * s}, r3.Vec{X: 1}))
		})
	}
	b.part("maiden bar", "adjustment", "fixed", func() *decad.Body {
		section := b.section(false)
		rectangle(section, headX-70*s, -280*s+shift, headX+70*s, -30*s+shift)
		for _, offset := range []float64{-50 * s, 50 * s} {
			rectangle(section, headX+offset-3.3, -140*s+shift, headX+offset+3.3, -90*s+shift)
		}
		for _, y := range []float64{-245*s + shift, -42*s + shift} {
			for _, offset := range []float64{-30 * s, 30 * s} {
				circle(section, headX+offset, y, 3.3)
			}
		}
		circle(section, brakeX, brakeY, 2.1)
		body := b.extrude(section, 7, 15*s, positiveZ(595*s))
		pocket := b.box(brakeX-4.1, brakeX+4.1, brakeY-4.1, brakeY+4.1, 595*s-1, 595*s+4)
		return b.cut(body, pocket)
	})
	for _, clamp := range []struct {
		name string
		x    float64
	}{
		{"left maiden clamp", headX - 50*s},
		{"right maiden clamp", headX + 50*s},
	} {
		b.part(clamp.name+" M6 bolt", "thread envelope", "fixed", func() *decad.Body {
			return b.cylinder(3, 23*s,
				axisFrame(r3.Vec{X: clamp.x, Y: -100 * s, Z: 587 * s}, r3.Vec{Z: 1}))
		})
		b.part(clamp.name+" low head", "fastener", "fixed", func() *decad.Body {
			return b.cylinder(6, 3,
				axisFrame(r3.Vec{X: clamp.x, Y: -100 * s, Z: 610 * s}, r3.Vec{Z: 1}))
		})
	}
	for _, foot := range []struct {
		name   string
		y0, y1 float64
	}{
		{"front maiden foot", -259*s + shift, -231*s + shift},
		{"rear maiden foot", -54*s + shift, -30*s + shift},
	} {
		y := (foot.y0 + foot.y1) / 2
		b.part(foot.name, "adjustment", "fixed", func() *decad.Body {
			return b.panel(headX-40*s, foot.y0, headX+40*s, foot.y1, 15*s, positiveZ(610*s),
				hole{headX - 30*s, y, 3.3}, hole{headX + 30*s, y, 3.3})
		})
		for _, bolt := range []struct {
			name string
			x    float64
		}{
			{"left", headX - 30*s},
			{"right", headX + 30*s},
		} {
			b.part(foot.name+" "+bolt.name+" M6 bolt", "thread envelope", "fixed", func() *decad.Body {
				return b.cylinder(3, 25*s,
					axisFrame(r3.Vec{X: bolt.x, Y: y, Z: 600 * s}, r3.Vec{Z: 1}))
			})
			b.part(foot.name+" "+bolt.name+" bolt head", "fastener", "fixed", func() *decad.Body {
				return b.cylinder(6, 4.2,
					axisFrame(r3.Vec{X: bolt.x, Y: y, Z: 625 * s}, r3.Vec{Z: 1}))
			})
		}
	}
}
