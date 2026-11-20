package planetarygearbox_test

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// locateTolerance is the tolerance point queries ask of the chamfered ring, covers and carriers. They are held as
// meshes whose bound is up to about 0.05 mm, and a query cannot ask for less than the body's bound.
const locateTolerance = 0.06

// sectionOutline returns the outline of body in the plane. It asks for 0.01 mm and, for a body held as a mesh
// whose own bound is larger, for that bound, so the clearances computed from it carry the real uncertainty.
func sectionOutline(t *testing.T, body *decad.Body, plane r3.Frame) decad.SectionOutline {
	t.Helper()
	tolerance := 0.01
	for range 2 {
		outline, err := body.SectionOutline(t.Context(), plane, units.Millimeters(tolerance))
		if err == nil {
			return outline
		}
		const marker = "minimum mesh bound "
		message := err.Error()
		at := strings.Index(message, marker)
		require.GreaterOrEqual(t, at, 0, "section outline: %v", err)
		var bound float64
		_, scanErr := fmt.Sscanf(message[at+len(marker):], "%f", &bound)
		require.NoError(t, scanErr)
		tolerance = math.Ceil(bound*1e4) / 1e4
	}
	panic("section outline: tolerance retry failed")
}

const (
	gearRootModuleClearanceFactor = 0.25
	minimumLooseGearClearance     = 0.5
	gearRootMeasurementMargin     = 0.02
	minimumFastenerGapBound       = 0.05
	gearRotationSamples           = 4
)

func minimumGearRootClearance(module float64) float64 {
	return math.Max(gearRootModuleClearanceFactor*module, minimumLooseGearClearance)
}

func maximumBearingClearanceAtRootWall(p planetarygearbox.Params) float64 {
	rootDepth := p.Module + minimumGearRootClearance(p.Module) + gearRootMeasurementMargin
	planetRoot := p.Module*float64(p.PlanetTeeth)/2 - rootDepth
	return planetRoot - p.Module - ballbearing.Params605ZZ().OuterDiameter/2
}

// sharedBuild holds one gearbox build that several tests read. The tests must not change it.
type sharedBuild struct {
	once sync.Once
	doc  *decad.Document
	box  planetarygearbox.Gearbox
	err  error
}

// get builds the gearbox for p on the first call and returns that build on every call.
func (s *sharedBuild) get(t *testing.T, p planetarygearbox.Params) (*decad.Document, planetarygearbox.Gearbox) {
	t.Helper()
	s.once.Do(func() {
		s.doc = planetarygearbox.NewDocument()
		s.box, s.err = planetarygearbox.Build(t.Context(), s.doc, p)
	})
	require.NoError(t, s.err)
	return s.doc, s.box
}

var defaultBuild, twoStageBuild, placedBuild sharedBuild

// defaultGearbox builds the default gearbox once for the tests that read it.
func defaultGearbox(t *testing.T) (*decad.Document, planetarygearbox.Gearbox) {
	t.Helper()
	return defaultBuild.get(t, planetarygearbox.DefaultParams())
}

// twoStageParams are the default parameters with two stages.
func twoStageParams() planetarygearbox.Params {
	p := planetarygearbox.DefaultParams()
	p.Stages = 2
	return p
}

// twoStageGearbox builds the two-stage gearbox once for the tests that read it.
func twoStageGearbox(t *testing.T) (*decad.Document, planetarygearbox.Gearbox) {
	t.Helper()
	return twoStageBuild.get(t, twoStageParams())
}

// placedParams are the default parameters placed on a frame whose axis is +X, through (10, -5, 7).
func placedParams(t *testing.T) planetarygearbox.Params {
	t.Helper()
	p := planetarygearbox.DefaultParams()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p.Placement = &f
	return p
}

// placedGearbox builds the placed gearbox once for the tests that read it.
func placedGearbox(t *testing.T) (*decad.Document, planetarygearbox.Gearbox) {
	t.Helper()
	return placedBuild.get(t, placedParams(t))
}

func bounds(t *testing.T, b *decad.Body) decad.Box {
	t.Helper()
	box, err := b.Bounds()
	require.NoError(t, err)
	return box
}

// penetration counts the vertices of either body that lie inside the other, beyond a small
// tolerance. Vertices of a flank in true contact are on the boundary, not inside.
func penetration(t *testing.T, a, b *decad.Body) int {
	t.Helper()
	return inside(t, a, b) + inside(t, b, a)
}

func hasPenetration(t *testing.T, a, b *decad.Body) bool {
	t.Helper()
	return countInside(t, a, b, true) > 0 || countInside(t, b, a, true) > 0
}

func requireDefaultPlanetMesh(t *testing.T, index int) {
	t.Helper()
	_, box := defaultGearbox(t)
	requirePlanetMesh(t, box, index)
}

func requireBacklashPlanetMesh(t *testing.T, index int) {
	t.Helper()
	p := planetarygearbox.DefaultParams()
	p.Backlash = 0.2
	box, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
	require.NoError(t, err)
	requirePlanetMesh(t, box, index)
}

func requirePlanetMesh(t *testing.T, box planetarygearbox.Gearbox, index int) {
	t.Helper()
	planets := box.Stages[0].Planets
	require.Len(t, planets, 3, "update the separate planet mesh tests when the default planet count changes")
	planet := planets[index]
	require.Zero(t, penetration(t, planet, box.Stages[0].Sun), "planet %d inside the sun", index)
	require.Zero(t, penetration(t, planet, box.Ring), "planet %d inside the ring", index)
}

func requireSectionClearance(t *testing.T, a, b *decad.Body, z, minimum float64) {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	outlineA := sectionOutline(t, a, plane)
	outlineB := sectionOutline(t, b, plane)
	clearance, err := outlineA.Clearance(outlineB)
	require.NoError(t, err)
	require.False(t, clearance.Overlaps)
	require.GreaterOrEqual(t, clearance.Lower().Base(), minimum,
		"section gap %v ± %v mm, closest points %v and %v", clearance.Distance.Value.Base(),
		clearance.Distance.Bound.Base(), clearance.A, clearance.B)
}

// requireSectionTouch checks that the sections of a and b at z meet with no gap: the closest distance is zero
// within its measurement bound. Outlines that share an edge report as overlapping, so overlap is not tested here;
// the shaft and the bore have the same diameter by construction. It is the check for a shaft or pin in a purchased
// bearing's bore (docs/model-pipeline.md "Part clearances").
func requireSectionTouch(t *testing.T, a, b *decad.Body, z float64) {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	clearance, err := sectionOutline(t, a, plane).Clearance(sectionOutline(t, b, plane))
	require.NoError(t, err)
	require.LessOrEqual(t, clearance.Distance.Value.Base(), clearance.Distance.Bound.Base()+1e-3,
		"section gap %v ± %v mm, closest points %v and %v", clearance.Distance.Value.Base(),
		clearance.Distance.Bound.Base(), clearance.A, clearance.B)
}

// sectionLocate classifies point against the outline of body in the plane through the point, by counting the
// outline loops that surround it.
func sectionLocate(t *testing.T, body *decad.Body, point r3.Vec) decad.PointLocation {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, point.Z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	section := sectionOutline(t, body, plane)
	crossings := 0
	for _, loop := range section.Loops {
		for i, a := range loop {
			b := loop[(i+1)%len(loop)]
			if (a.Y > point.Y) == (b.Y > point.Y) {
				continue
			}
			if point.X < a.X+(point.Y-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
				crossings++
			}
		}
	}
	if crossings%2 == 1 {
		return decad.PointInside
	}
	return decad.PointOutside
}

type radialSpan struct {
	min, max, bound float64
}

func sectionRadialSpan(t *testing.T, body *decad.Body, center r3.Vec, z, targetRadius float64) radialSpan {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	section := sectionOutline(t, body, plane)
	require.NotEmpty(t, section.Loops)

	var selected radialSpan
	bestScore := math.Inf(1)
	for _, loop := range section.Loops {
		current := radialSpan{min: math.Inf(1), bound: section.OffSurface.Base()}
		for i, point := range loop {
			next := loop[(i+1)%len(loop)]
			x0, y0 := point.X-center.X, point.Y-center.Y
			x1, y1 := next.X-center.X, next.Y-center.Y
			dx, dy := x1-x0, y1-y0
			lengthSquared := dx*dx + dy*dy
			fraction := 0.0
			if lengthSquared > 0 {
				fraction = math.Max(0, math.Min(1, -(x0*dx+y0*dy)/lengthSquared))
			}
			closest := math.Hypot(x0+fraction*dx, y0+fraction*dy)
			current.min = math.Min(current.min, closest)
			current.max = math.Max(current.max, math.Hypot(x0, y0))
		}
		score := math.Abs((current.min+current.max)/2 - targetRadius)
		if targetRadius == 0 {
			score = -current.max
		}
		if score < bestScore {
			selected, bestScore = current, score
		}
	}
	return selected
}

// requireFitClearances checks the part fits of the built gearbox box against their fit classes.
func requireFitClearances(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	stage := box.Stages[0]
	planetBearing := stage.Bearings[0]
	bearingBounds := bounds(t, planetBearing)
	bearingMid := (bearingBounds.Min.Z + bearingBounds.Max.Z) / 2
	// A bearing seat is designed at 0.05 mm radial clearance (docs/model-pipeline.md "Part clearances"). The
	// section measurement's uncertainty reaches 0.019 mm on the meshed covers, so the proven bound is about 0.031 mm.
	const seatMinimum = 0.02

	// The sun's bore is a press fit on the shaft: the sun turns with it.
	requireSectionClearance(t, box.InputShaft, stage.Sun, p.GearThickness/2, seatMinimum)
	// The shaft's thick part is a press fit in the carrier's hole, tighter than the Tight minimum.
	requireSectionClearance(t, stage.Carrier, stage.Shaft,
		p.GearThickness+p.StageGap+p.CarrierThickness/2, seatMinimum)
	requireSectionClearance(t, planetBearing, stage.Planets[0], bearingMid, seatMinimum)
	requireSectionTouch(t, planetBearing, stage.Pins[0], bearingMid)
	frontFace := -p.StageGap - p.CoverThickness
	requireSectionClearance(t, box.Bolts[0], box.FrontCover,
		frontFace+m2fastener.BoltSize.HeadHeight/2, minimumFastenerGapBound)
	requireSectionClearance(t, box.Bolts[0], box.Ring, 0, minimumFastenerGapBound)
	nut := bounds(t, box.Nuts[0])
	requireSectionClearance(t, box.Nuts[0], box.Ring, (nut.Min.Z+nut.Max.Z)/2, minimumFastenerGapBound)
	// An ear's bolt has the clearance hole's gap in the ear's bore, and its nut the cavity's gap.
	earNut := bounds(t, box.EarNuts[0])
	requireSectionClearance(t, box.EarBolts[0], box.Ring, earNut.Max.Z+2, minimumFastenerGapBound)
	requireSectionClearance(t, box.EarNuts[0], box.Ring, (earNut.Min.Z+earNut.Max.Z)/2, minimumFastenerGapBound)

	frontBearing := bounds(t, box.FrontBearing)
	requireSectionTouch(t, box.InputShaft, box.FrontBearing, (frontBearing.Min.Z+frontBearing.Max.Z)/2)
	requireSectionClearance(t, box.FrontBearing, box.FrontCover,
		(frontBearing.Min.Z+frontBearing.Max.Z)/2, seatMinimum)
	rearBearing := bounds(t, box.RearBearing)
	rearBearingMid := (rearBearing.Min.Z + rearBearing.Max.Z) / 2
	requireSectionTouch(t, stage.Shaft, box.RearBearing, rearBearingMid)
	requireSectionClearance(t, box.RearBearing, box.RearCover, rearBearingMid, seatMinimum)
}

// inside counts the outline vertices of a, taken at mid-height of b, that lie inside b. The gears
// are straight prisms, so their meshes have vertices only on the end caps and the cap outline
// stands for every height.
func inside(t *testing.T, a, b *decad.Body) int {
	t.Helper()
	return countInside(t, a, b, false)
}

func countInside(t *testing.T, a, b *decad.Body, stopAfterFirst bool) int {
	t.Helper()
	mesh, err := a.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	require.NoError(t, err)
	box := bounds(t, b)
	mid := (box.Min.Z + box.Max.Z) / 2
	count := 0
	locations := make(map[[2]float64]decad.PointLocation)
	for _, v := range mesh.Vertices() {
		if v.X < box.Min.X || v.X > box.Max.X || v.Y < box.Min.Y || v.Y > box.Max.Y {
			continue
		}
		key := [2]float64{v.X, v.Y}
		loc, ok := locations[key]
		if !ok {
			loc, err = b.LocatePoint(t.Context(), r3.NewVec(v.X, v.Y, mid), units.Millimeters(0.02))
			require.NoError(t, err)
			locations[key] = loc
		}
		if loc == decad.PointInside {
			count++
			if stopAfterFirst {
				return count
			}
		}
	}
	return count
}

// requireTipRootClearance checks the tip-to-root clearance of the built gearbox box at several rotation angles across
// one sun tooth pitch.
func requireTipRootClearance(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	z := p.GearThickness / 2
	orbit := p.Module * float64(p.SunTeeth+p.PlanetTeeth) / 2
	minimum := minimumGearRootClearance(p.Module)
	ring := sectionRadialSpan(t, box.Ring, r3.Vec{}, z, p.Module*float64(p.RingTeeth)/2)
	toothPitch := 2 * math.Pi / float64(p.SunTeeth)

	for sample := range gearRotationSamples + 1 {
		sunAngle := toothPitch * float64(sample) / gearRotationSamples
		carrierAngle := float64(p.SunTeeth) / float64(p.SunTeeth+p.RingTeeth) * sunAngle
		planetRelativeAngle := -float64(p.RingTeeth) / float64(p.PlanetTeeth) * carrierAngle
		sun, err := decad.Copy(t.Context(), box.Stages[0].Sun, decad.WithCopyRotation(
			r3.Vec{}, r3.NewVec(0, 0, 1), units.Radians(sunAngle)))
		require.NoError(t, err)
		planet, err := decad.Copy(t.Context(), box.Stages[0].Planets[0], decad.WithCopyRotation(
			r3.NewVec(orbit, 0, 0), r3.NewVec(0, 0, 1), units.Radians(planetRelativeAngle)))
		require.NoError(t, err)
		carrierTurn, err := r3.Rotation(r3.NewVec(0, 0, 1), units.Radians(carrierAngle))
		require.NoError(t, err)
		planet, err = decad.Transform(t.Context(), planet, carrierTurn)
		require.NoError(t, err)
		planetCenter := r3.NewVec(orbit*math.Cos(carrierAngle), orbit*math.Sin(carrierAngle), 0)
		sunSpan := sectionRadialSpan(t, sun, r3.Vec{}, z, 0)
		planetSpan := sectionRadialSpan(t, planet, planetCenter, z, 0)

		for name, measured := range map[string]float64{
			"sun root to planet tip":  orbit - (sunSpan.min + sunSpan.bound) - (planetSpan.max + planetSpan.bound),
			"planet root to sun tip":  orbit - (planetSpan.min + planetSpan.bound) - (sunSpan.max + sunSpan.bound),
			"ring root to planet tip": (ring.max - ring.bound) - orbit - (planetSpan.max + planetSpan.bound),
			"ring tip to planet root": (ring.min - ring.bound) - orbit + (planetSpan.min - planetSpan.bound),
		} {
			require.GreaterOrEqual(t, measured, minimum,
				"sample %d at %.6g radians, %s measured tip-to-root clearance", sample, sunAngle, name)
		}
	}
}

func volume(t *testing.T, b *decad.Body) (float64, float64) {
	t.Helper()
	v, err := b.Volume()
	require.NoError(t, err)
	return v.Value.Base(), v.Bound.Base()
}

// circleSegment is the area a flat depth deep cuts off a circle of the given radius.
func circleSegment(radius, depth float64) float64 {
	floor := radius - depth
	return radius*radius*math.Acos(floor/radius) - floor*math.Sqrt(radius*radius-floor*floor)
}

// integrate sums f over [0, length] by the midpoint rule.
func integrate(length float64, f func(z float64) float64) float64 {
	const steps = 4000
	h := length / steps
	sum := 0.0
	for i := range steps {
		sum += f((float64(i) + 0.5) * h)
	}
	return sum * h
}

// coneRun is the axial length of a 30° half-angle cone per unit of radius change.
var coneRun = math.Sqrt(3)

// coneVolume is the volume of the cone from radius from down to radius to over to-from times coneRun of length, less
// the circle segment a flat depth deep cuts off every section, as the link shaft's flat follows the cone.
func coneVolume(from, to, depth float64) float64 {
	return integrate((from-to)*coneRun, func(z float64) float64 {
		radius := from - z/coneRun
		return math.Pi*radius*radius - circleSegment(radius, depth)
	})
}

// rearBossTop is the height of the rear cover's bearing boss top for a gearbox of the given stages.
func rearBossTop(p planetarygearbox.Params, stages int) float64 {
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap
	return float64(stages)*pitch + p.CoverThickness + 5
}

// requireLinkCountersink checks the gap between a link shaft's 30° cone and the countersink in the next sun's rear bore.
// The gap is the 0.25 mm the shaft's old square step had to the sun, measured normal to the cone, and the Tight minimum
// of Part clearances is 0.2 mm. A horizontal section sees the gap along the radius, which is the normal gap over cos 30°,
// so the sections up the countersink must show 0.2/cos 30° or more. sunZ is the height of the sun's rear face.
func requireLinkCountersink(t *testing.T, link, sun *decad.Body, sunZ float64) {
	t.Helper()
	for _, up := range []float64{0.3, 0.6, 0.9} {
		requireSectionClearance(t, link, sun, sunZ+up, 0.2/math.Cos(math.Pi/6))
	}
}
