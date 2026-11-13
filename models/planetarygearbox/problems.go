package planetarygearbox

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
)

// Problems returns every way p is invalid, in check order. Each names the failing rule; a bearing-forced minimum
// also names the minimum and a fix. The result is empty for a buildable p.
//
// The input checks run first. When one fails, Problems returns the input problems alone, because the fit checks
// read derived dimensions that need valid inputs. A check runs only when the inputs it reads passed their own checks.
func Problems(p Params) []string {
	if probs := inputProblems(p); len(probs) > 0 {
		return probs
	}
	d := derive(p)
	probs := fitProblems(p, d)
	probs = append(probs, bearingFitProblems(p, d)...)
	return append(probs, earProblems(p, d)...)
}

// inputProblems checks the inputs one at a time, in the order of the construction contract.
func inputProblems(p Params) []string {
	var probs []string
	add := func(format string, args ...any) { probs = append(probs, fmt.Sprintf(format, args...)) }

	namesKnown := true
	for _, b := range []struct{ role, name string }{{"planet", p.PlanetBearing}, {"shaft", p.ShaftBearing}} {
		if err := bearingNameError(b.role, b.name); err != nil {
			probs = append(probs, err.Error())
			namesKnown = false
		}
	}

	lengths := []struct {
		name string
		v    float64
	}{
		{"Module", p.Module}, {"PressureAngle", p.PressureAngle}, {"GearThickness", p.GearThickness},
		{"CarrierThickness", p.CarrierThickness}, {"StageGap", p.StageGap}, {"CoverThickness", p.CoverThickness},
		{"BoltHoleDiameter", p.BoltHoleDiameter}, {"InputShaftLength", p.InputShaftLength},
		{"FlatDepth", p.FlatDepth}, {"SeatChamfer", p.SeatChamfer}, {"EdgeChamfer", p.EdgeChamfer},
		{"BearingClearance", p.BearingClearance}, {"CarrierWall", p.CarrierWall}, {"RingWall", p.RingWall},
		{"Backlash", p.Backlash}, {"OutputShaftLength", p.OutputShaftLength}, {"EarBoreDiameter", p.EarBoreDiameter},
	}
	allFinite := true
	for _, l := range lengths {
		if !finite(l.v) {
			add("%s must be finite, got %v", l.name, l.v)
			allFinite = false
		}
	}
	if !allFinite {
		return probs
	}

	flatOK := p.FlatDepth >= 0
	if namesKnown {
		flatOK = flatOK && p.FlatDepth <= math.Min(planetBearing(p).BoreDiameter, shaftBearing(p).BoreDiameter)/4
	}
	for _, c := range []struct {
		name string
		v    float64
		ok   bool
	}{
		{"module", p.Module, p.Module > 0},
		{"gear thickness", p.GearThickness, p.GearThickness > 0},
		{"carrier thickness", p.CarrierThickness, p.CarrierThickness >= 2},
		{"stage gap", p.StageGap, p.StageGap >= planetBearingLift+pinShoulderClearance+pinShoulderReach},
		{"cover thickness", p.CoverThickness, p.CoverThickness > 0},
		{"bolt hole diameter", p.BoltHoleDiameter, p.BoltHoleDiameter > 0},
		{"input shaft length", p.InputShaftLength, p.InputShaftLength > 0},
		{"flat depth", p.FlatDepth, flatOK},
		{"seat chamfer", p.SeatChamfer, p.SeatChamfer >= 0 && p.SeatChamfer <= 1},
		{"edge chamfer", p.EdgeChamfer, p.EdgeChamfer >= MinEdgeChamfer && p.EdgeChamfer <= 1},
		{"bearing clearance", p.BearingClearance, p.BearingClearance >= seatClearance},
		{"carrier wall", p.CarrierWall, p.CarrierWall > 0},
		{"ring wall", p.RingWall, p.RingWall > 0},
		{"backlash", p.Backlash, p.Backlash >= 0 && p.Backlash <= p.Module/2},
		{"output shaft length", p.OutputShaftLength, p.OutputShaftLength > 0},
		{"ear count", float64(p.EarCount), p.EarCount >= 0},
		{"ear bore diameter", p.EarBoreDiameter, p.EarBoreDiameter >= 0 && (p.EarCount == 0 || p.EarBoreDiameter > 0)},
	} {
		if !c.ok {
			add("%s %v is out of range", c.name, c.v)
		}
	}
	if p.PressureAngle <= 0 || p.PressureAngle >= math.Pi/2 {
		add("pressure angle %v must be in (0, pi/2)", p.PressureAngle)
	}
	if p.Samples < 3 {
		add("samples %d must be at least 3", p.Samples)
	}
	countOK := p.PlanetCount >= 2
	if !countOK {
		add("planet count %d must be at least 2", p.PlanetCount)
	}
	if p.Stages < 1 {
		add("stages %d must be at least 1", p.Stages)
	}
	teethOK := true
	for _, t := range []struct {
		name  string
		teeth int
	}{{"sun", p.SunTeeth}, {"planet", p.PlanetTeeth}, {"ring", p.RingTeeth}} {
		if t.teeth < 3 {
			add("%s teeth %d must be at least 3", t.name, t.teeth)
			teethOK = false
		}
	}
	if !teethOK {
		return append(probs, placementProblems(p)...)
	}
	if p.RingTeeth != p.SunTeeth+2*p.PlanetTeeth {
		add("ring teeth %d must equal sun teeth + 2 * planet teeth = %d", p.RingTeeth, p.SunTeeth+2*p.PlanetTeeth)
	}
	if countOK && (p.SunTeeth+p.RingTeeth)%p.PlanetCount != 0 {
		add("sun teeth + ring teeth = %d must divide by planet count %d so the planets mesh at equal angles",
			p.SunTeeth+p.RingTeeth, p.PlanetCount)
	}
	return append(probs, placementProblems(p)...)
}

func placementProblems(p Params) []string {
	if p.Placement != nil && !p.Placement.IsValid() {
		return []string{"invalid placement frame"}
	}
	return nil
}

// suggestion names a kit tooth set for p's bearings when p is at module 1 and uses other teeth, or returns "".
func suggestion(p Params) string {
	if p.Module != 1 {
		return ""
	}
	for _, k := range kits {
		if k.PlanetBearing != p.PlanetBearing || k.ShaftBearing != p.ShaftBearing {
			continue
		}
		if k.sun == p.SunTeeth && k.planet == p.PlanetTeeth && k.ring == p.RingTeeth {
			return ""
		}
		return fmt.Sprintf("; try %d/%d/%d", k.sun, k.planet, k.ring)
	}
	return ""
}

// fitProblems checks that the parts fit around each other. It returns every failing check.
func fitProblems(p Params, d dims) []string {
	var probs []string
	add := func(format string, args ...any) { probs = append(probs, fmt.Sprintf(format, args...)) }
	m := p.Module
	if chord := 2 * d.orbit * math.Sin(math.Pi/float64(p.PlanetCount)); chord < 2*d.planetTip+looseGearClearance {
		add("neighbouring planet centers are %v apart; tips need %v for at least %v clearance",
			chord, 2*d.planetTip+looseGearClearance, looseGearClearance)
	}
	sunRoot := m*float64(p.SunTeeth)/2 - gearRootDepth(m)
	if d.shaftBore/2 > sunRoot-m {
		minSun := int(math.Ceil((d.shaftBore+2*gearRootDepth(m)+2*m)/m - 1e-9))
		add("the %v mm shaft bore leaves less than one module of wall under the sun root; "+
			"SunTeeth must be at least %d for the %g mm shaft at module %g%s",
			d.shaftBore, minSun, d.shaftBore, m, suggestion(p))
	}
	planetW := planetBearing(p).Width
	if p.GearThickness-planetW < lipThickness {
		add("gear thickness %v leaves less than %v for the lip behind a %v wide bearing; "+
			"a %s planet bearing needs GearThickness >= %g",
			p.GearThickness, float64(lipThickness), planetW, p.PlanetBearing, planetW+lipThickness)
	}
	if d.planetPocket/2 > d.planetRoot-m {
		minPlanet := int(math.Ceil((d.planetPocket+2*gearRootDepth(m)+2*gearRootMeasurementMargin+2*m)/m - 1e-9))
		add("bearing pocket diameter %v leaves less than one module of wall under the planet root; "+
			"PlanetTeeth must be at least %d for the %s planet bearing at module %g%s",
			d.planetPocket, minPlanet, p.PlanetBearing, m, suggestion(p))
	}
	if d.shaft/2+d.boss/2 >= d.orbit {
		add("shaft diameter %v and planet pin boss %v do not fit within orbit radius %v", d.shaft, d.boss, d.orbit)
	}
	if d.boltCircle-p.BoltHoleDiameter/2 < d.ringRoot+boltRimMargin ||
		d.boltCircle+p.BoltHoleDiameter/2 > d.ringOuter-boltRimMargin {
		add("bolt holes of diameter %v do not fit the ring wall %v with %v of material either side",
			p.BoltHoleDiameter, p.RingWall, boltRimMargin)
	}
	if p.BoltHoleDiameter < minBoltHoleDiameter {
		add("bolt hole diameter %v is under the %v clearance hole for the M2 bolt shank %v",
			p.BoltHoleDiameter, minBoltHoleDiameter, m2bolt.ShankDiameter)
	}
	if p.CoverThickness < m2bolt.HeadHeight+coverFloor {
		add("cover thickness %v leaves less than %v under a %v tall bolt head",
			p.CoverThickness, float64(coverFloor), m2bolt.HeadHeight)
	}
	if reach := math.Max(nutPocketDiameter(), headPocketDiameter()) / 2; d.boltCircle-reach < d.ringRoot+pocketMargin ||
		d.boltCircle+reach > d.ringOuter-pocketMargin {
		add("bolt head and nut pockets of radius %v do not fit the ring wall %v with %v of material either side",
			reach, p.RingWall, float64(pocketMargin))
	}
	if d.ringLength < 2*(nutRoof+nutCavityHeight())+2*pocketMargin {
		add("ring length %v is too short for a nut cavity at each end", d.ringLength)
	}
	if d.coverHub/2 >= d.rimInner-d.arm {
		minWall := 2 * (d.coverHub/2 + d.arm + headPocketDiameter()/2 + boltRimMargin - d.ringRoot)
		add("the cover's bearing hub of diameter %v leaves no room for spokes inside the rim at radius %v; "+
			"the %s shaft bearing needs RingWall above %.2f or more RingTeeth",
			d.coverHub, d.rimInner, p.ShaftBearing, minWall)
	}
	if d.orbit+d.boss/2 >= d.ringTip-looseGearClearance {
		add("carrier radius %v does not leave more than %v clearance from ring tip radius %v",
			d.orbit+d.boss/2, looseGearClearance, d.ringTip)
	}
	if p.FlatDepth >= collarMargin {
		add("flat depth %v must stay under the %v collar margin, so a thick part's flat stays outside its rod",
			p.FlatDepth, float64(collarMargin))
	}
	socket := d.pinCollar/2 + pressFitClearance
	if wall := d.boss/2 - socket; wall < minSocketWall {
		add("carrier wall %v leaves %v of plate around a pin socket, less than %v",
			p.CarrierWall, wall, float64(minSocketWall))
	}
	if face := d.boss/2 - socket - p.EdgeChamfer; face < minSocketFaceWall {
		add("carrier wall %v and edge chamfer %v leave %v of plate around a pin socket at the plate's face, less than %v",
			p.CarrierWall, p.EdgeChamfer, face, float64(minSocketFaceWall))
	}
	return probs
}

// bearingFitProblems runs the checks that a bearing's size forces on the shafts, pins and carrier.
func bearingFitProblems(p Params, d dims) []string {
	var probs []string
	for _, s := range []string{
		collarRaceProblem(d.collar, shaftBearing(p).InnerRaceDiameter, p.ShaftBearing),
		pinRaceProblem(d.pinCollar, planetBearing(p).InnerRaceDiameter, p.PlanetBearing),
		collarLipProblem(d.collar, d.coverLip),
		centreHoleProblem(d.orbit, d.collar, d.pinCollar),
	} {
		if s != "" {
			probs = append(probs, s)
		}
	}
	return probs
}

// collarRaceProblem returns a problem when the shaft collar is not narrower than the shaft bearing's inner race.
func collarRaceProblem(collar, race float64, bearing string) string {
	if collar < race {
		return ""
	}
	return fmt.Sprintf("the %g mm shaft collar is not narrower than the %g mm inner race of the %s shaft bearing",
		collar, race, bearing)
}

// pinRaceProblem returns a problem when the pin's thick part is not narrower than the planet bearing's inner race.
func pinRaceProblem(pinCollar, race float64, bearing string) string {
	if pinCollar < race {
		return ""
	}
	return fmt.Sprintf("the %g mm pin thick part is not narrower than the %g mm inner race of the %s planet bearing",
		pinCollar, race, bearing)
}

// collarLipProblem returns a problem when the shaft collar does not pass the cover's lip bore with fitClearance on
// each side.
func collarLipProblem(collar, lip float64) string {
	if collar+2*fitClearance <= lip {
		return ""
	}
	return fmt.Sprintf("the %g mm shaft collar does not pass the %g mm cover lip bore with %g on each side",
		collar, lip, fitClearance)
}

// centreHoleProblem returns a problem when the carrier's centre hole and a pin socket leave less than minSocketWall
// of plate between them.
func centreHoleProblem(orbit, collar, pinCollar float64) string {
	plate := orbit - (collar/2 + pressFitClearance) - (pinCollar/2 + pressFitClearance)
	if plate >= minSocketWall {
		return ""
	}
	return fmt.Sprintf("the carrier's centre hole and pin sockets leave %g of plate between them, less than %g",
		plate, float64(minSocketWall))
}

// earProblems checks that the ears clear each other and the ring's bolt sites.
func earProblems(p Params, d dims) []string {
	if p.EarCount == 0 {
		return nil
	}
	var probs []string
	half := earHalfWidth(p.EarBoreDiameter)
	radius := earCenterRadius(d.ringOuter, p.EarBoreDiameter)
	if p.EarCount > 1 {
		// Neighbouring ears are closest at their half circles, whose centers are a chord apart.
		if chord := 2 * radius * math.Sin(math.Pi/float64(p.EarCount)); chord < 2*half+earGap {
			probs = append(probs, fmt.Sprintf(
				"neighbouring ear bores are %v apart; the ears need %v for at least %v between them",
				chord, 2*half+earGap, float64(earGap)))
		}
	}
	// A bolt site is clear of an ear when its pockets stay beyond the ear's side edge, or the ear lies on the
	// far side of the axis.
	reach := math.Max(nutPocketDiameter(), headPocketDiameter())/2 + pocketMargin
	for i := range p.EarCount {
		for _, b := range boltAngles() {
			delta := earAngle(i, p.EarCount) - b
			if math.Cos(delta) > 0 && d.boltCircle*math.Abs(math.Sin(delta)) < half+reach {
				probs = append(probs, fmt.Sprintf(
					"ear %d lies over a bolt site: turn the ears off the bolt angles by using another ear count", i+1))
				break
			}
		}
	}
	return probs
}
