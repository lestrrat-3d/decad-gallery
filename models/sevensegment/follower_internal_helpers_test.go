package sevensegment

import "math"

// The cam follower drive model of instructions.md "Cam follower drive": a rigid slider with Coulomb friction.
const (
	guideFloorZ      = 54.0                                                      // the guide plate's underside
	followerOffsetY  = (2.35 + StemThickness/2 + PinLength - sliderPinChamf) / 2 // 2.95: groove face to pin chamfer
	followerStemFace = StemThickness/2 - stemChamfer                             // 1.5: the stem's flat x face, half length
	followerYawLever = 7.0 - 2*stemChamfer                                       // 6.4: the stem (x +-3.5) less its chamfers
	followerLoad     = 0.067                                                     // N per changing slider
)

// followerBladeY returns how far toward -y the blade's x contact can sit, from the stem's centre plane: -7.8 for a, g
// and d (the blade's far face), -1.8 for b to f (the print face).
func followerBladeY(s Segment) float64 {
	switch s {
	case SegA, SegG, SegD:
		return -7.8
	}
	return -1.8
}

// followerKind returns 0 for a, g and d and 1 for b to f.
func followerKind(s Segment) int {
	if followerBladeY(s) == -7.8 {
		return 0
	}
	return 1
}

// followerDrive returns the drive ratio D and the net drive per unit groove-wall load at pressure angle alpha
// (degrees), lift (mm), friction mu and blade offset bladeY. It reads sliderPinZ, bladeRidgeZ and guideFloorZ for a and L.
func followerDrive(alpha, lift, mu, bladeY float64) (float64, float64) {
	rad := alpha * math.Pi / 180
	fv := math.Cos(rad) - mu*math.Sin(rad)
	h := math.Sin(rad) + mu*math.Cos(rad)
	a := guideFloorZ - (sliderPinZ + lift)
	l := bladeRidgeZ + lift - guideFloorZ
	e := followerOffsetY
	r1 := h * (1 + a/l)
	r2 := h * a / l
	m := math.Max(0, e*h+bladeY*r2-followerStemFace*r1)
	drive := fv * (1 - 2*mu*e/l)
	drag := mu * (h*(1+2*a/l) + 2*m/followerYawLever)
	return drive / drag, drive - drag
}

// followerResult is followerSweep's result. Index 0 is a/g/d, 1 is b-f.
type followerResult struct {
	minRatio  [2]float64
	maxWall   [2]float64 // N; +Inf where a sample locks
	maxTorque float64    // N.mm: the largest sum over a (step, off) sample of load*H*R over that step's sliders
	step      int        // From of the step with that sum
}

// followerSweep samples every ramp of every segment at friction mu: for each step k and off from RampStart to RampEnd
// in 0.05 steps, psi = psiOf(k) + off, for every segment with a ramp whose From == k.
func followerSweep(dm Dimensions, mu float64) followerResult {
	res := followerResult{minRatio: [2]float64{math.Inf(1), math.Inf(1)}}
	for k := range 10 {
		for i := 0; ; i++ {
			off := RampStart + 0.05*float64(i)
			if off > RampEnd+1e-9 {
				break
			}
			psi := psiOf(k) + off
			sum := 0.0
			for _, s := range Segments() {
				if !hasRampFrom(dm, s, k) {
					continue
				}
				alpha := dm.PressureAngle(s, psi)
				radius := dm.GrooveRadius(s, psi)
				ratio, net := followerDrive(alpha, radius-OffRadius, mu, followerBladeY(s))
				kind := followerKind(s)
				res.minRatio[kind] = math.Min(res.minRatio[kind], ratio)
				load := math.Inf(1)
				if net > 0 {
					load = followerLoad / net
				}
				res.maxWall[kind] = math.Max(res.maxWall[kind], load)
				rad := alpha * math.Pi / 180
				sum += load * (math.Sin(rad) + mu*math.Cos(rad)) * radius
			}
			if sum > res.maxTorque {
				res.maxTorque, res.step = sum, k
			}
		}
	}
	return res
}

func hasRampFrom(dm Dimensions, s Segment, k int) bool {
	for _, r := range dm.Ramps[s] {
		if r.From == k {
			return true
		}
	}
	return false
}

// lockFriction bisects mu over [0.2, 0.6], 40 halvings, for followerSweep's least ratio of kind (0 a/g/d, 1 b-f)
// reaching 1.
func lockFriction(dm Dimensions, kind int) float64 {
	lo, hi := 0.2, 0.6
	for range 40 {
		mid := (lo + hi) / 2
		if followerSweep(dm, mid).minRatio[kind] <= 1 {
			hi = mid
			continue
		}
		lo = mid
	}
	return (lo + hi) / 2
}
