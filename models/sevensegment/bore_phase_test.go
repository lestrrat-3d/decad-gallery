package sevensegment_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/stretchr/testify/require"
)

// borePlay is the largest turn, in degrees, of a double-D shaft section (radius sr, flats at +-sf) inside its double-D
// bore (radius br, flats at +-bf) when the bore may also shift across its clearance: the angle a part can turn on the
// shaft to either side of its modeled angle. A part that carries torque shifts until it bears on two opposite points, so
// the shift is part of the play.
func borePlay(sr, sf, br, bf float64) float64 {
	var pts [][2]float64
	for i := range 1440 {
		x, y := sr*math.Cos(2*math.Pi*float64(i)/1440), sr*math.Sin(2*math.Pi*float64(i)/1440)
		if math.Abs(x) <= sf {
			pts = append(pts, [2]float64{x, y})
		}
	}
	h := math.Sqrt(sr*sr - sf*sf)
	for _, x := range []float64{sf, -sf} {
		for j := 0; j <= 200; j++ {
			pts = append(pts, [2]float64{x, -h + 2*h*float64(j)/200})
		}
	}
	fits := func(al, dx, dy float64) bool {
		s, c := math.Sincos(al)
		for _, p := range pts {
			x, y := c*p[0]-s*p[1]+dx, s*p[0]+c*p[1]+dy
			if x*x+y*y > br*br+1e-12 || math.Abs(x) > bf+1e-12 {
				return false
			}
		}
		return true
	}
	clear, best := br-sr, 0.0
	for ix := -10; ix <= 10; ix++ {
		for iy := -10; iy <= 10; iy++ {
			dx, dy := clear*float64(ix)/10, clear*float64(iy)/10
			if !fits(0, dx, dy) {
				continue
			}
			lo, hi := 0.0, 0.1
			for range 40 {
				if m := (lo + hi) / 2; fits(m, dx, dy) {
					lo = m
				} else {
					hi = m
				}
			}
			best = math.Max(best, lo)
		}
	}
	return best * 180 / math.Pi
}

// phaseSwing is how far, in degrees, a cam can lead or lag the ratchet wheel: each turns on the shaft by its bore play,
// and a part a torque drives lags its driver, so the two plays add when the pawls drive the shaft.
func phaseSwing() float64 {
	cam := borePlay(sevensegment.ShaftCamRadius, sevensegment.ShaftCamFlat, sevensegment.CamBoreRadius, sevensegment.CamBoreFlat)
	ratchet := borePlay(sevensegment.ShaftCamRadius, sevensegment.ShaftCamFlat, sevensegment.RatchetBoreRadius,
		sevensegment.RatchetBoreFlat)
	return cam + ratchet
}

// TestSevenSegmentBorePlay pins the D-bore play the determinacy tests use: each double-D bore lets its part turn 1.71
// degrees either way (a single-D bore of the same fit turns 3.51, because the part shifts away from its flat), so a cam
// can lead or lag the ratchet by 3.42.
func TestSevenSegmentBorePlay(t *testing.T) {
	cam := borePlay(sevensegment.ShaftCamRadius, sevensegment.ShaftCamFlat, sevensegment.CamBoreRadius, sevensegment.CamBoreFlat)
	t.Logf("cam bore play %.3f deg, phase swing %.3f deg", cam, phaseSwing())
	require.InDelta(t, 1.71, cam, 0.02)
	require.InDelta(t, 3.42, phaseSwing(), 0.04)
}

// phaseRest walks the ratchet through one step, from the lock face stop to past the drop into the next notch, with the
// cams offset from it by the drive: while the pawls drive the shaft back the cams lead the ratchet by up to swing; where
// friction locks the pawls (release torque <= 0 before the crest) the knob left them in phase; past the crest, where the
// pawls drive the shaft on, the cams lag by up to swing. At every angle where a cam (over the slider's timing allowance)
// is more than 0.05 off its dwell it returns the worst such offset at an angle the shaft can rest (the detent does not
// beat the opposing torque), and the least ratio of release torque to opposing torque while the pawls drive back.
func phaseRest(t *testing.T, dm sevensegment.Dimensions, mu, swing float64) (worstOff, minRelease float64) {
	t.Helper()
	l := builtLoads(t)
	fx := budgetFx(t, mu)
	c := pawlCompliance(dm, pawlE)
	const step, lo = 0.05, -12.0
	type camPoint struct {
		off, load float64
		done      bool
	}
	cams := make([][]camPoint, 10)
	for k := range cams {
		cams[k] = make([]camPoint, int(60/step))
	}
	at := func(k int, th float64) camPoint {
		p := &cams[k][int(math.Round((th-lo)/step))]
		if p.done {
			return *p
		}
		p.done = true
		for _, sa := range []float64{-sevensegment.TimingAllowance, 0, sevensegment.TimingAllowance} {
			a := th + sa
			digit := int(math.Round(a / 36))
			for _, s := range sevensegment.Segments() {
				lit, _ := sevensegment.Lit((k+digit+10)%10, s)
				want := 0.0
				if lit {
					want = sevensegment.Travel
				}
				p.off = math.Max(p.off, math.Abs(dm.Lift(s, 36*float64(k)+a)-want))
			}
		}
		if p.off > 0.05 {
			p.load, _ = flagLoad(dm, k, th, mu, fx, l.ws)
		}
		return *p
	}
	relZero := dm.CrestAngle
	for psi := 20.0; psi <= dm.CrestAngle; psi += step {
		if detentTorque(dm, c, psi, mu, true) <= 0 {
			relZero = psi
			break
		}
	}
	end := 37.0
	for psi := dm.CrestAngle; psi <= 37; psi += step {
		if d, _ := dm.PawlDeflection(psi); d <= 1e-6 {
			end = psi
			break
		}
	}
	minRelease = math.Inf(1)
	for i := 0; ; i++ {
		psi := -1.4 + float64(i)*step
		if psi > end+0.5 {
			break
		}
		td := detentTorque(dm, c, psi, mu, true)
		elo, ehi, back := 0.0, swing, true
		switch {
		case psi > dm.CrestAngle:
			elo, ehi, back, td = -swing, 0, false, -td
		case psi >= relZero:
			elo, ehi, back, td = 0, 0, false, 0
		}
		for e := elo; e <= ehi+1e-9; e += step {
			th := math.Round((psi+e)/step) * step
			for k := range 10 {
				p := at(k, th)
				if p.off <= 0.05 {
					continue
				}
				if back {
					minRelease = math.Min(minRelease, td/p.load)
				}
				if td < p.load {
					worstOff = math.Max(worstOff, p.off)
				}
			}
		}
	}
	return worstOff, minRelease
}
