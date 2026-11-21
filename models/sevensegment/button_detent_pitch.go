package sevensegment

import "math"

// buttonDetentPitch returns the roller-centre radius and its radial derivative.
func buttonDetentPitch(degrees float64) (float64, float64) {
	const t = math.Pi / 180
	const slope = 2.8
	theta := math.Mod(degrees, 360)
	if theta < 0 {
		theta += 360
	}
	if theta >= 354 || theta < 20 {
		u := math.Mod(theta+6, 360) * t
		l := 4 * t
		if u < l {
			return 22 + slope*u*u/(2*l), slope * u / l
		}
		return 22 + slope*(u-l/2), slope
	}
	if theta < 260 {
		h := slope * (theta + 4) * t
		d := slope
		for _, dip := range [][3]float64{{67, 129, 0.4}, {205, 259, slope*14*t - 0.4}} {
			left, right, amplitude := dip[0], dip[1], dip[2]
			if theta >= right {
				h -= amplitude
			} else if theta > left {
				u, l := (theta-left)*t, (right-left)*t
				k := 2 * amplitude / l
				h -= k * (u/2 - l*math.Sin(2*math.Pi*u/l)/(4*math.Pi))
				d -= k * math.Pow(math.Sin(math.Pi*u/l), 2)
			}
		}
		return 22 + h, d
	}
	if theta < 280 {
		u, l := (theta-260)*t, 20*t
		return 22 + slope*(250*t+u-u*u/(2*l)), slope * (1 - u/l)
	}
	u := (theta - 280) * math.Pi / 74
	h := slope * 260 * t
	return 22 + h*(1+math.Cos(u))/2, -h * (90.0 / 74.0) * math.Sin(u)
}
