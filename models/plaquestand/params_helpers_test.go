package plaquestand_test

import (
	"github.com/lestrrat-3d/decad-gallery/models/plaquestand"
)

// expected holds one case's offline values from the design (docs: instructions.md "Representative cases").
type expected struct {
	name                       string
	params                     plaquestand.Params
	span, backHeight           float64
	seatY, depth, height       float64
	tipFront, tipBack, tipSide float64
	area, length               float64
}

func cases() []expected {
	p := func(w, h, t, theta, lip, c float64) plaquestand.Params {
		return plaquestand.Params{
			PlaqueWidth: w, PlaqueHeight: h, PlaqueThickness: t, Recline: theta, LipHeight: lip, SlotClearance: c,
		}
	}
	return []expected{
		{"default", p(250, 305, 12.8, 15, 10, 0.55), 200, 60, 24.303105, 112, 70.432523,
			19.875977, 15.042312, 31.403024, 1361.179509, 394.964986},
		{"min", p(150, 120, 2, 10, 6, 0.55), 120, 30, 17.280750, 55, 38.349710,
			18.644768, 18.572439, 40.766229, 621.769165, 196.627946},
		{"max", p(400, 350, 15, 25, 20, 1.5), 246, 60, 29.192341, 160, 66.588125,
			27.346216, 15.206476, 34.345575, 1851.394079, 515.368145},
		{"forward", p(300, 350, 2, 10, 10, 0.55), 240, 60, 23.828646, 109, 67.893943,
			15.000000, 15.262544, 33.224387, 1253.769166, 372.627946},
		{"narrow-tall", p(150, 350, 15, 10, 6, 1.5), 120, 60, 31.809631, 124, 74.050264,
			15.000000, 15.154925, 17.328331, 1471.829062, 418.032811},
		{"short-steep", p(150, 120, 15, 25, 20, 1.5), 120, 30, 29.192341, 81, 40.220496,
			32.136555, 15.245825, 38.859428, 1039.394079, 297.368145},
	}
}
