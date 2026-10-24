package cycloidal

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// dimensions are the sizes derived from Params after the auto-or-override choices, in millimeters.
type dimensions struct {
	pinRadius, effectiveRadius    float64 // Rr, Rr + c
	rootRadius                    float64 // Rv = R - Rr_eff - E
	pinCircleRadius, outputRadius float64 // R, Rop
	outputPinDiameter             float64 // D_pin
	outputHoleDiameter            float64 // D_pin + 2E
	housingInner, housingOuter    float64
	outputPlateDiameter           float64
	lobes                         int
}

// resolve derives the dimensions of p. It does not check them.
func resolve(p Params) dimensions {
	var d dimensions
	n, m := float64(p.PinCount), float64(p.OutputPinCount)
	d.lobes = p.PinCount - 1
	d.pinCircleRadius = p.PinCircleDiameter / 2
	d.outputRadius = p.OutputPinCircleDiameter / 2
	if p.PinDiameter > 0 {
		d.pinRadius = p.PinDiameter / 2
	} else {
		d.pinRadius = (p.Eccentricity + d.pinCircleRadius*math.Sin(math.Pi/n)) / 2
	}
	d.effectiveRadius = d.pinRadius + p.DiskClearance
	d.rootRadius = d.pinCircleRadius - d.effectiveRadius - p.Eccentricity
	if p.OutputPinDiameter > 0 {
		d.outputPinDiameter = p.OutputPinDiameter
	} else {
		d.outputPinDiameter = d.outputRadius*math.Sin(math.Pi/m) - p.Eccentricity
	}
	d.outputHoleDiameter = d.outputPinDiameter + 2*p.Eccentricity
	d.housingInner = 2 * (d.pinCircleRadius - d.pinRadius - p.HousingWall)
	d.housingOuter = 2 * (d.pinCircleRadius - d.pinRadius + 2*p.Eccentricity + p.HousingWall)
	d.outputPlateDiameter = p.OutputPinCircleDiameter + d.outputPinDiameter + 2*p.HousingWall
	return d
}

// undercutScanPoints is the number of curve parameters the curvature scan samples.
const undercutScanPoints = 2000

// minInwardCurvatureRadius returns the smallest radius of curvature of the base trochoid among the points
// whose center of curvature lies toward the drive axis. A rotor profile offset by a roller larger than this
// undercuts. It is +Inf when no point qualifies.
func minInwardCurvatureRadius(r, e float64, n int) float64 {
	nf := float64(n)
	least := math.Inf(1)
	for i := range undercutScanPoints {
		t := 2 * math.Pi * float64(i) / undercutScanPoints
		sn, cn := math.Sincos(nf * t)
		st, ct := math.Sincos(t)
		bx, by := r*ct-e*cn, -r*st+e*sn
		xp, yp := -r*st+e*nf*sn, -r*ct+e*nf*cn
		xpp, ypp := -r*ct+e*nf*nf*cn, r*st-e*nf*nf*sn
		k := xp*ypp - yp*xpp
		if math.Abs(k) < 1e-12 {
			continue
		}
		speed := math.Hypot(xp, yp)
		rho := speed * speed * speed / k
		cx, cy := bx+rho*(-yp/speed), by+rho*(xp/speed)
		if cx*cx+cy*cy < bx*bx+by*by {
			least = math.Min(least, math.Abs(rho))
		}
	}
	return least
}

// undercuts reports whether the profile of p with eccentricity e undercuts.
func undercuts(p Params, e float64) bool {
	q := p
	q.Eccentricity = e
	d := resolve(q)
	return d.effectiveRadius >= minInwardCurvatureRadius(d.pinCircleRadius, e, p.PinCount)
}

// undercutBound returns the largest eccentricity not above p's that does not undercut, found by 40 bisection
// steps. It is zero when none does.
func undercutBound(p Params) float64 {
	lo, hi := 0.0, p.Eccentricity
	for range 40 {
		mid := (lo + hi) / 2
		if undercuts(p, mid) {
			hi = mid
		} else {
			lo = mid
		}
	}
	return lo
}

// ErrInvalid wraps every parameter problem Validate reports.
var ErrInvalid = errors.New("cycloidal: invalid parameters")

// Problems returns every way p is invalid, in the order of the add-in's validity table, each naming the
// offending input and a numeric bound. The result is empty for a buildable p.
func Problems(p Params) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, fmt.Sprintf(format, args...)) }
	switch {
	case p.PinCount < 4:
		add("Pin Count must be at least 4 (currently %d).", p.PinCount)
	case p.OutputPinCount < 3:
		add("Output Pin Count must be at least 3 (currently %d).", p.OutputPinCount)
	case p.DiscCount != 1 && p.DiscCount != 2:
		add("Disc Count must be 1 or 2 (currently %d).", p.DiscCount)
	}
	for _, f := range []struct {
		name  string
		value float64
		zero  bool
	}{
		{"Pin Circle Diameter", p.PinCircleDiameter, false},
		{"Eccentricity", p.Eccentricity, false},
		{"Disc Thickness", p.DiscThickness, false},
		{"Center Bearing Diameter", p.CenterBearingDiameter, false},
		{"Output Pin Circle Diameter", p.OutputPinCircleDiameter, false},
		{"Housing Wall", p.HousingWall, false},
		{"Base Thickness", p.BaseThickness, false},
		{"Output Plate Thickness", p.OutputPlateThickness, false},
		{"Pin Diameter", p.PinDiameter, true},
		{"Disk Clearance", p.DiskClearance, true},
		{"Disc Gap", p.DiscGap, true},
		{"Input Shaft Diameter", p.InputShaftDiameter, true},
		{"Bearing Clearance", p.BearingClearance, true},
		{"Output Pin Diameter", p.OutputPinDiameter, true},
		{"Chamfer Size", p.ChamferSize, true},
	} {
		switch {
		case math.IsNaN(f.value) || math.IsInf(f.value, 0):
			add("%s must be finite.", f.name)
		case f.zero && f.value < 0:
			add("%s must not be negative (currently %.2f mm).", f.name, f.value)
		case !f.zero && f.value <= 0:
			add("%s must be greater than 0 (currently %.2f mm).", f.name, f.value)
		}
	}
	if len(out) > 0 {
		return out
	}
	n, m := float64(p.PinCount), float64(p.OutputPinCount)
	d := resolve(p)
	r, rop, e := d.pinCircleRadius, d.outputRadius, p.Eccentricity
	dhole, dpin := d.outputHoleDiameter, d.outputPinDiameter
	cbd, isd, clr := p.CenterBearingDiameter, p.InputShaftDiameter, p.BearingClearance
	if p.DiscCount == 2 && (p.PinCount%2 != 0 || p.OutputPinCount%2 != 0) {
		add("Two discs require an even Pin Count and an even Output Pin Count (currently N=%d, M=%d).",
			p.PinCount, p.OutputPinCount)
	}
	if p.DiscCount == 2 && p.DiscGap == 0 {
		add("Two discs require a Disc Gap greater than 0 (the cam sections join through the gap).")
	}
	if !(e < d.pinRadius && d.pinRadius < r*math.Sin(math.Pi/n)) {
		if p.PinDiameter == 0 {
			add("Pin geometry out of range — increase Pin Circle Diameter above %.2f mm or reduce Eccentricity below %.2f mm.",
				2*e/math.Sin(math.Pi/n), r*math.Sin(math.Pi/n))
		} else {
			add("Pin Diameter must be between %.2f mm and %.2f mm (currently %.2f).",
				2*e, 2*r*math.Sin(math.Pi/n), 2*d.pinRadius)
		}
	}
	if dpin <= 0 {
		if p.OutputPinDiameter == 0 {
			add("Output pins vanish (resolved diameter ≤ 0) — increase Output Pin Circle Diameter above %.2f mm, increase Output Pin Count, or reduce Eccentricity.",
				2*e/math.Sin(math.Pi/m))
		} else {
			add("Output Pin Diameter must be greater than 0.")
		}
	}
	if !(dhole < 2*rop*math.Sin(math.Pi/m)) {
		add("Output holes overlap — increase Output Pin Circle Diameter above %.2f mm, increase Output Pin Count, or reduce Output Pin Diameter / Eccentricity.",
			(dpin+2*e)/math.Sin(math.Pi/m))
	}
	if !(e < r/n) {
		add("Eccentricity too large — reduce it below %.2f mm (or increase Pin Circle Diameter / reduce Pin Count).", r/n)
	}
	if !(rop < d.rootRadius) {
		add("Output Pin Circle too large — set Output Pin Circle Diameter below %.2f mm (currently %.2f).",
			2*d.rootRadius, 2*rop)
	}
	if d.rootRadius > 0 && undercuts(p, e) {
		if bound := undercutBound(p); bound > 0 {
			add("Eccentricity too large — the rotor profile undercuts/self-intersects. Reduce Eccentricity below %.2f mm.", bound)
		} else {
			add("Eccentricity too large — the rotor profile undercuts/self-intersects. Reduce Eccentricity.")
		}
	}
	if !(isd < cbd) {
		add("Input Shaft Diameter must be less than Center Bearing Diameter (%.2f mm).", cbd)
	}
	if !(e+isd/2 < cbd/2) {
		add("Input bore doesn't fit inside the cam — set Input Shaft Diameter below %.2f mm, or reduce Eccentricity / increase Center Bearing Diameter.",
			cbd-2*e)
	}
	if !((cbd+clr)/2 < rop-dhole/2) {
		add("Disk center bore overlaps the output holes — increase Output Pin Circle Diameter above %.2f mm, or reduce Center Bearing Diameter / Bearing Clearance / output pin size.",
			cbd+clr+dhole)
	}
	return out
}

// Validate returns nil for a buildable p, else an error wrapping ErrInvalid that lists every problem.
func Validate(p Params) error {
	if problems := Problems(p); len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalid, strings.Join(problems, "\n"))
	}
	return nil
}
