package planetarygearbox_test

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch/param"
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

func TestDefaultGearbox(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	require.Len(t, doc.Bodies(), 34)
	require.Len(t, box.Stages, 1)
	require.Len(t, box.Stages[0].Planets, 3)
	require.Len(t, box.Stages[0].Bearings, 3)
	require.Len(t, box.Named(), 34)
	for _, nb := range box.Named() {
		require.True(t, nb.Body.IsSolid(), nb.Name)
		v, err := nb.Body.Volume()
		require.NoError(t, err, nb.Name)
		require.Positive(t, v.Value.Base(), nb.Name)
	}
	require.InDelta(t, 1+71.0/13, p.Ratio(), 1e-12)
	rootClearance := minimumGearRootClearance(p.Module)
	planetRoot, err := doc.Param("root_radius")
	require.NoError(t, err)
	require.InDelta(t, p.Module*float64(p.PlanetTeeth)/2-p.Module-rootClearance-gearRootMeasurementMargin,
		planetRoot.Base(), 1e-9)
	ringRoot, err := doc.Param("ring_root_radius")
	require.NoError(t, err)
	require.InDelta(t, p.Module*float64(p.RingTeeth)/2+p.Module+rootClearance, ringRoot.Base(), 1e-9)

	// The ring spans the gap below the first stage to one stage pitch, with its outer wall
	// at the ring root radius plus the wall.
	ring := bounds(t, box.Ring)
	require.InDelta(t, -p.StageGap, ring.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness+p.CarrierThickness+2*p.StageGap, ring.Max.Z, 1e-6)
	// The ring's outer wall is at the ring root radius plus the wall. The three mounting ears stand beyond it, the
	// first on +Y and the others 120 degrees on, so the ring reaches an ear's tip along +Y and the tip of the
	// ears at 330 degrees along +X.
	outer := 35.5 + p.Module + minimumGearRootClearance(p.Module) + p.RingWall
	earHalf := p.EarBoreDiameter/2 + 3
	earCenter := outer + earHalf
	require.InDelta(t, earCenter+earHalf, ring.Max.Y, 0.05)
	require.InDelta(t, earCenter*math.Cos(math.Pi/6)+earHalf, ring.Max.X, 0.05)
	require.InDelta(t, -outer, ring.Min.Y, 0.05)

	stage := box.Stages[0]
	sun := bounds(t, stage.Sun)
	require.InDelta(t, 0, sun.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness, sun.Max.Z, 1e-6)
	sunTipRadius := p.Module*float64(p.SunTeeth)/2 + p.Module - gearRootMeasurementMargin
	require.InDelta(t, sunTipRadius, sun.Max.X, 1e-2)
	for i, planet := range stage.Planets {
		b := bounds(t, planet)
		phi := planetarygearbox.PlanetOrbitAngle(p, i)
		require.InDelta(t, 21*math.Cos(phi), (b.Min.X+b.Max.X)/2, 0.05, "planet %d", i)
		require.InDelta(t, 21*math.Sin(phi), (b.Min.Y+b.Max.Y)/2, 0.05, "planet %d", i)
		require.InDelta(t, p.GearThickness, b.Max.Z-b.Min.Z, 1e-6)
	}
	// The carrier is the plate alone, above the gears. Its pins are separate bodies, and nothing stands above the
	// plate, so the carrier prints plate-down.
	carrier := bounds(t, stage.Carrier)
	require.InDelta(t, p.GearThickness+p.StageGap, carrier.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness+p.StageGap+p.CarrierThickness, carrier.Max.Z, 1e-6)
	// The plate reaches the far side of the pin bosses and no further: orbit radius plus boss radius.
	require.InDelta(t, 21+(5+2*p.CarrierWall)/2, carrier.Max.X, 5e-3)
	// The output shaft is a separate body. It starts at the carrier's lower face and ends OutputShaftLength
	// beyond the rear cover's bearing boss.
	output := bounds(t, stage.Shaft)
	require.InDelta(t, p.GearThickness+p.StageGap, output.Min.Z, 1e-6)
	require.InDelta(t, rearBossTop(p, 1)+p.OutputShaftLength, output.Max.Z, 1e-6)
}

// penetration counts the vertices of either body that lie inside the other, beyond a small
// tolerance. Vertices of a flank in true contact are on the boundary, not inside.
func penetration(t *testing.T, a, b *decad.Body) int {
	t.Helper()
	return inside(t, a, b) + inside(t, b, a)
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

func TestPartClearances(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
	stage := box.Stages[0]
	planetBearing := stage.Bearings[0]
	bearingBounds := bounds(t, planetBearing)
	bearingMid := (bearingBounds.Min.Z + bearingBounds.Max.Z) / 2
	const minimum = 0.2
	// A bearing seat is designed at 0.05 mm radial clearance (docs/model-pipeline.md "Part clearances"). The
	// section measurement's uncertainty reaches 0.024 mm on the meshed covers, so the proven bound is about 0.026 mm.
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
		frontFace+m2bolt.HeadHeight/2, minimumFastenerGapBound)
	requireSectionClearance(t, box.Bolts[0], box.Ring, 0, minimumFastenerGapBound)
	nut := bounds(t, box.Nuts[0])
	requireSectionClearance(t, box.Nuts[0], box.Ring, (nut.Min.Z+nut.Max.Z)/2, minimumFastenerGapBound)

	frontBearing := bounds(t, box.FrontBearing)
	requireSectionTouch(t, box.InputShaft, box.FrontBearing, (frontBearing.Min.Z+frontBearing.Max.Z)/2)
	requireSectionClearance(t, box.FrontBearing, box.FrontCover,
		(frontBearing.Min.Z+frontBearing.Max.Z)/2, seatMinimum)
	rearBearing := bounds(t, box.RearBearing)
	rearBearingMid := (rearBearing.Min.Z + rearBearing.Max.Z) / 2
	requireSectionTouch(t, stage.Shaft, box.RearBearing, rearBearingMid)
	requireSectionClearance(t, box.RearBearing, box.RearCover, rearBearingMid, seatMinimum)

	branchParams := p
	branchParams.FlatDepth = 0
	branchParams.StageGap = 0.75
	branchParams.BearingClearance = 0.5
	branchParams.SeatChamfer = 0
	branch, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), branchParams)
	require.NoError(t, err)
	branchBearing := branch.Stages[0].Bearings[0]
	branchBearingBounds := bounds(t, branchBearing)
	branchBearingMid := (branchBearingBounds.Min.Z + branchBearingBounds.Max.Z) / 2
	requireSectionClearance(t, branchBearing, branch.Stages[0].Planets[0], branchBearingMid, minimum)
	requireSectionTouch(t, branchBearing, branch.Stages[0].Pins[0], branchBearingMid)
	requireSectionClearance(t, branch.Stages[0].Carrier, branch.Stages[0].Shaft,
		p.GearThickness+branchParams.StageGap+p.CarrierThickness/2, seatMinimum)
	requireInnerLoopsSharp(t, "branch carrier", branch.Stages[0].Carrier, p.PlanetCount+1)
	branchCarrierZ := p.GearThickness + branchParams.StageGap
	requireSectionClearance(t, branch.Stages[0].Pins[0], branch.Stages[0].Carrier,
		branchCarrierZ+p.CarrierThickness/2, seatMinimum)
	require.InDelta(t, branchCarrierZ+p.CarrierThickness, bounds(t, branch.Stages[0].Pins[0]).Max.Z, 1e-6)
	require.InDelta(t, 0.5, p.GearThickness+branchParams.StageGap-branchBearingBounds.Max.Z,
		1e-6, "minimum stage gap above the planet bearing")
}

// inside counts the outline vertices of a, taken at mid-height of b, that lie inside b. The gears
// are straight prisms, so their meshes have vertices only on the end caps and the cap outline
// stands for every height.
func inside(t *testing.T, a, b *decad.Body) int {
	t.Helper()
	mesh, err := a.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	require.NoError(t, err)
	box := bounds(t, b)
	mid := (box.Min.Z + box.Max.Z) / 2
	count := 0
	for _, v := range mesh.Vertices() {
		if v.X < box.Min.X || v.X > box.Max.X || v.Y < box.Min.Y || v.Y > box.Max.Y {
			continue
		}
		loc, err := b.LocatePoint(t.Context(), r3.NewVec(v.X, v.Y, mid), units.Millimeters(0.02))
		require.NoError(t, err)
		if loc == decad.PointInside {
			count++
		}
	}
	return count
}

func TestPlanetsMeshWithSunAndRing(t *testing.T) {
	_, box := defaultGearbox(t)
	for i, planet := range box.Stages[0].Planets {
		require.Zero(t, penetration(t, planet, box.Stages[0].Sun), "planet %d inside the sun", i)
		require.Zero(t, penetration(t, planet, box.Ring), "planet %d inside the ring", i)
	}
}

func TestGearTipRootClearance(t *testing.T) {
	for _, module := range []float64{0.5, 1, 2, 4} {
		p := planetarygearbox.DefaultParams()
		p.Module = module
		minimum := minimumGearRootClearance(module)
		rootDepth := module + minimum
		orbit := module * float64(p.SunTeeth+p.PlanetTeeth) / 2
		ringRoot := module*float64(p.RingTeeth)/2 + rootDepth
		ringTip := module*float64(p.RingTeeth)/2 - module
		sunPitch := module * float64(p.SunTeeth) / 2
		planetPitch := module * float64(p.PlanetTeeth) / 2
		sunTip := sunPitch + module - gearRootMeasurementMargin
		planetTip := planetPitch + module - gearRootMeasurementMargin
		sunRoot := sunPitch - rootDepth
		planetRoot := planetPitch - rootDepth - gearRootMeasurementMargin

		require.GreaterOrEqual(t, orbit-sunRoot-planetTip, minimum, "sun root to planet tip")
		require.GreaterOrEqual(t, orbit-planetRoot-sunTip, minimum, "planet root to sun tip")
		require.GreaterOrEqual(t, ringRoot-orbit-planetTip, minimum, "ring root to planet tip")
		require.GreaterOrEqual(t, ringTip-orbit-planetRoot, minimum, "ring tip to planet root")
		planetCenterSpacing := 2 * orbit * math.Sin(math.Pi/float64(p.PlanetCount))
		require.GreaterOrEqual(t, planetCenterSpacing-2*planetTip, minimumLooseGearClearance,
			"non-meshing planet tips")
	}

	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
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

func TestWrongPhaseIsDetected(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
	planet := box.Stages[0].Planets[0]
	center := r3.NewVec(21, 0, 0)
	turned, err := decad.Copy(t.Context(), planet, decad.WithCopyRotation(center, r3.NewVec(0, 0, 1),
		units.Radians(math.Pi/float64(p.PlanetTeeth))))
	require.NoError(t, err)
	require.Positive(t, penetration(t, turned, box.Stages[0].Sun))
	require.Positive(t, penetration(t, turned, box.Ring))
}

func volume(t *testing.T, b *decad.Body) (float64, float64) {
	t.Helper()
	v, err := b.Volume()
	require.NoError(t, err)
	return v.Value.Base(), v.Bound.Base()
}

// TestBacklashRemovesMaterial checks that more backlash thins the sun and planet teeth and widens
// the ring's tooth spaces, so each of those bodies has less volume, and that the planets still
// mesh without penetration.
func TestBacklashRemovesMaterial(t *testing.T) {
	_, base := defaultGearbox(t)
	p := planetarygearbox.DefaultParams()
	p.Backlash = 0.2
	loose, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
	require.NoError(t, err)
	for name, pair := range map[string][2]*decad.Body{
		"sun":    {base.Stages[0].Sun, loose.Stages[0].Sun},
		"planet": {base.Stages[0].Planets[0], loose.Stages[0].Planets[0]},
		"ring":   {base.Ring, loose.Ring},
	} {
		tight, tightBound := volume(t, pair[0])
		looseVolume, looseBound := volume(t, pair[1])
		require.Greater(t, tight-looseVolume, tightBound+looseBound, name)
	}
	for i, planet := range loose.Stages[0].Planets {
		require.Zero(t, penetration(t, planet, loose.Stages[0].Sun), "planet %d inside the sun", i)
		require.Zero(t, penetration(t, planet, loose.Ring), "planet %d inside the ring", i)
	}
}

// TestBearingsSitInPlanetPockets checks that each planet has an empty pocket where its bearing sits, with
// the bearing's 605ZZ envelope in it, and that the lip under the pocket is the only material cut back:
// the planet weighs the plain gear with the lip bore minus the ring of pocket between the lip bore and
// the pocket wall. Points are probed 1.5 mm or more from any surface, because point location is
// undecided closer than that to a thin wall.
func TestBearingsSitInPlanetPockets(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	stage := box.Stages[0]
	bearingWidth, err := doc.Param("bearing_width")
	require.NoError(t, err)
	bearingLift, err := doc.Param("planet_bearing_lift")
	require.NoError(t, err)

	rootClearance := minimumGearRootClearance(p.Module)
	plain, err := (spurgear.Variant{Define: func(table *param.Table) error {
		if err := table.SetValue("gear_root_clearance", units.Millimeters(rootClearance)); err != nil {
			return err
		}
		if err := table.SetValue("gear_root_margin", units.Millimeters(gearRootMeasurementMargin)); err != nil {
			return err
		}
		if err := table.SetValue("gear_tip_measurement_margin", units.Millimeters(gearRootMeasurementMargin)); err != nil {
			return err
		}
		if err := table.SetExpr("root_radius", "pitch_radius - module - gear_root_clearance - gear_root_margin", units.Millimeter); err != nil {
			return err
		}
		return table.SetExpr("tip_radius", "pitch_radius + module - gear_tip_measurement_margin", units.Millimeter)
	}}).Build(t.Context(), decad.New(), spurgear.Params{
		Module: p.Module, Teeth: p.PlanetTeeth, PressureAngle: p.PressureAngle, Thickness: p.GearThickness,
		Samples: p.Samples, BoreDiameter: 10, Backlash: p.Backlash,
	})
	require.NoError(t, err)
	plainVolume, plainBound := volume(t, plain)
	pocketDiameter := 14 + 2*p.BearingClearance
	pocket := math.Pi / 4 * (pocketDiameter*pocketDiameter - 10*10) * 5

	for i, planet := range stage.Planets {
		phi := planetarygearbox.PlanetOrbitAngle(p, i)
		around := func(radius, angle, z float64) r3.Vec {
			return r3.NewVec(21*math.Cos(phi)+radius*math.Cos(angle), 21*math.Sin(phi)+radius*math.Sin(angle), z)
		}
		locate := func(b *decad.Body, point r3.Vec) decad.PointLocation {
			loc, err := b.LocatePoint(t.Context(), point, units.Millimeters(locateTolerance))
			require.NoError(t, err)
			return loc
		}
		for _, radius := range []float64{3.5, 5, 6.3} {
			for _, z := range []float64{4, 5.5, 7} {
				for _, angle := range []float64{0.7, 2.5, 4.4} {
					point := around(radius, angle, z)
					require.Equal(t, decad.PointOutside, locate(planet, point), "planet %d pocket at %v", i, point)
					require.Equal(t, decad.PointInside, locate(stage.Bearings[i], point), "bearing %d at %v", i, point)
				}
			}
		}

		planetVolume, planetBound := volume(t, planet)
		// The pocket's entrance is chamfered, which removes a ring of triangular section.
		entrance := math.Pi * (14 + 2*p.BearingClearance) * p.SeatChamfer * p.SeatChamfer / 2
		require.InDelta(t, plainVolume-pocket-entrance, planetVolume, plainBound+planetBound+1, "planet %d volume", i)

		// The bearing is the 605ZZ envelope, 5 mm bore, 14 mm outside and 5 mm wide, centered on the
		// planet, raised above the pocket floor by 0.25 mm.
		b := bounds(t, stage.Bearings[i])
		require.InDelta(t, 14, b.Max.X-b.Min.X, 0.05, "bearing %d outside diameter", i)
		require.InDelta(t, bearingWidth.Base(), b.Max.Z-b.Min.Z, 1e-6, "bearing %d width", i)
		require.InDelta(t, p.GearThickness-bearingWidth.Base()+bearingLift.Base(), b.Min.Z, 1e-6, "bearing %d bottom", i)
		require.InDelta(t, p.GearThickness+bearingLift.Base(), b.Max.Z, 1e-6, "bearing %d top", i)
	}
}

// circleSegment is the area a flat depth deep cuts off a circle of the given radius.
func circleSegment(radius, depth float64) float64 {
	floor := radius - depth
	return radius*radius*math.Acos(floor/radius) - floor*math.Sqrt(radius*radius-floor*floor)
}

// TestCarrierIsATriangle checks that the carrier's plate is the solid triangle between the pin bosses with
// the shaft's D hole in the middle and a socket through the plate for each pin. Points are probed
// 1.5 mm or more from a surface where the body is thick, because point location is undecided closer than
// about 0.15 mm.
func TestCarrierIsATriangle(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
	carrier := box.Stages[0].Carrier
	locate := func(point r3.Vec) decad.PointLocation {
		loc, err := carrier.LocatePoint(t.Context(), point, units.Millimeters(locateTolerance))
		require.NoError(t, err)
		return loc
	}
	plate := p.GearThickness + p.StageGap + p.CarrierThickness/2
	at := func(radius, angle, z float64) r3.Vec {
		return r3.NewVec(radius*math.Cos(angle), radius*math.Sin(angle), z)
	}
	// Between two pins, at 60 degrees, the plate is solid from the hole out to the tangent edge, 10.5 mm
	// (half the pin spacing's height) plus the 5.25 mm boss radius = 15.75 mm from the axis.
	require.Equal(t, decad.PointInside, locate(at(12, math.Pi/3, plate)), "solid between the pins")
	require.Equal(t, decad.PointInside, locate(at(14, math.Pi/3, plate)), "solid out to the edge")
	require.Equal(t, decad.PointOutside, locate(at(20, math.Pi/3, plate)), "nothing past the tangent edge")
	// 0.75 mm from the socket flat, too close to a surface for point location, so the section decides.
	require.Equal(t, decad.PointInside, sectionLocate(t, carrier, at(25, 0, plate)), "a pin boss")
	require.Equal(t, decad.PointOutside, locate(at(0, 0, plate)), "the D hole")
	require.Equal(t, decad.PointOutside, locate(r3.NewVec(-1.7, 0, plate)), "the hole is open on the round side")
	require.Equal(t, decad.PointOutside, sectionLocate(t, carrier, r3.NewVec(-3.4, 0, plate)), "the hole is wide, R 3.75")
	require.Equal(t, decad.PointInside, locate(r3.NewVec(6, 0, plate)), "the plate is solid just outside the hole")
	// Each pin's socket passes through the plate: void in the socket near both faces, plate beside it.
	carrierZ := p.GearThickness + p.StageGap
	for _, z := range []float64{carrierZ + p.CarrierThickness/4, carrierZ + 3*p.CarrierThickness/4} {
		require.Equal(t, decad.PointOutside, sectionLocate(t, carrier, at(21, 0, z)), "socket at %v", z)
	}
	require.Equal(t, decad.PointInside, sectionLocate(t, carrier, at(16.5, 0, carrierZ+p.CarrierThickness/4)),
		"plate beside the socket")
	// The carrier weighs more than the plain triangle's pins and less than a disc that reaches its pins.
	v, bound := volume(t, carrier)
	disc := math.Pi * 28 * 28 * p.CarrierThickness
	require.Less(t, v-bound, disc+3*math.Pi*16*(p.GearThickness+p.StageGap+p.CarrierThickness/2))
}

// TestCarrierPins checks that each planet has its own pin body: a thin rod down to the gears' lower face and a
// D-shaped thick part inside the plate's socket, level with the plate's upper face, with a 0.05 mm gap all round
// the thick part. The pin's volume is the rod's plus the D prism's. Points inside the thin pin are undecided, so
// the shape is checked by bounds, volume and sections.
func TestCarrierPins(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	stage := box.Stages[0]
	require.Len(t, stage.Pins, p.PlanetCount)
	pinDiameter, err := doc.Param("pin_diameter")
	require.NoError(t, err)
	collarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	radius, thickRadius := pinDiameter.Base()/2, collarDiameter.Base()/2
	carrierZ := p.GearThickness + p.StageGap
	carrierTop := bounds(t, stage.Carrier).Max.Z
	const lift, clearance = 0.25, 0.25
	thickBottom := -p.StageGap + lift + clearance
	want := math.Pi*radius*radius*(p.GearThickness+clearance+lift) +
		(math.Pi*thickRadius*thickRadius-circleSegment(thickRadius, p.FlatDepth))*(p.CarrierThickness-thickBottom)
	for i, pin := range stage.Pins {
		require.True(t, pin.IsSolid(), "pin %d", i)
		b := bounds(t, pin)
		require.InDelta(t, 0, b.Min.Z, 1e-6, "pin %d starts at the gears' lower face", i)
		require.InDelta(t, carrierZ+p.CarrierThickness, b.Max.Z, 1e-6, "pin %d ends at the plate's upper face", i)
		require.InDelta(t, carrierTop, b.Max.Z, 1e-6, "pin %d is level with the carrier's top", i)
		v, bound := volume(t, pin)
		require.InDelta(t, want, v, bound+0.5, "pin %d volume", i)

		phi := planetarygearbox.PlanetOrbitAngle(p, i)
		u := r3.NewVec(math.Cos(phi), math.Sin(phi), 0)
		center := r3.NewVec(21*u.X, 21*u.Y, 0)
		rod := sectionRadialSpan(t, pin, center, p.GearThickness/2, 0)
		require.InDelta(t, radius, rod.min, rod.bound+2e-3, "pin %d rod", i)
		require.InDelta(t, radius, rod.max, rod.bound+2e-3, "pin %d rod", i)
		plateZ := carrierZ + p.CarrierThickness/2
		thick := sectionRadialSpan(t, pin, center, plateZ, 0)
		require.InDelta(t, thickRadius, thick.max, thick.bound+2e-3, "pin %d thick part", i)
		require.InDelta(t, thickRadius-p.FlatDepth, thick.min, thick.bound+2e-3, "pin %d flat", i)
		// The flat faces outward: the half the flat cuts off is absent and the opposite side is present.
		at := func(along float64) r3.Vec {
			return r3.NewVec(center.X+along*u.X, center.Y+along*u.Y, plateZ)
		}
		require.Equal(t, decad.PointOutside, sectionLocate(t, pin, at(thickRadius-p.FlatDepth/2)), "pin %d flat side", i)
		require.Equal(t, decad.PointInside, sectionLocate(t, pin, at(-(thickRadius-p.FlatDepth/2))), "pin %d round side", i)
		// The thick part has 0.05 mm of radial clearance in its socket, proven to about 0.026 mm.
		requireSectionClearance(t, pin, stage.Carrier, plateZ, 0.02)

		bearingBounds := bounds(t, stage.Bearings[i])
		top := bearingBounds.Max.Z
		inRod := r3.NewVec(center.X-3*u.X, center.Y-3*u.Y, 0)
		inRod.Z = top + 0.2
		require.Equal(t, decad.PointOutside, sectionLocate(t, pin, inRod), "pin %d thick part starts above 0.2", i)
		inRod.Z = top + 0.3
		require.Equal(t, decad.PointInside, sectionLocate(t, pin, inRod), "pin %d thick part starts below 0.3", i)
		requireSectionTouch(t, pin, stage.Bearings[i], (bearingBounds.Min.Z+bearingBounds.Max.Z)/2)
	}
}

// rearBossTop is the height of the rear cover's bearing boss top for a gearbox of the given stages.
func rearBossTop(p planetarygearbox.Params, stages int) float64 {
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap
	return float64(stages)*pitch + p.CoverThickness + 5
}

// TestHousing checks the ring, the two covers, their bearings and the input shaft: where they sit along the
// axis, that the bolt holes of the ring and both covers line up, that the covers are frames with windows and
// not discs, and that the input shaft fills the first sun's bore. Points are probed 1.5 mm or more from any
// surface, because point location is undecided closer than about 0.15 mm.
func TestHousing(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	locate := func(b *decad.Body, point r3.Vec) decad.PointLocation {
		loc, err := b.LocatePoint(t.Context(), point, units.Millimeters(locateTolerance))
		require.NoError(t, err)
		if loc != decad.PointUndecided {
			return loc
		}
		// The chamfered covers are held as meshes and point location leaves their interior undecided, so the
		// section through the point decides it. Every probe is 1.5 mm or more from a surface, far outside the
		// section's bound.
		return sectionLocate(t, b, point)
	}
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap

	// Axial layout. Each cover's frame lies against a ring end, with its bearing boss pointing away.
	front, rear := bounds(t, box.FrontCover), bounds(t, box.RearCover)
	require.InDelta(t, -p.StageGap-p.CoverThickness-5, front.Min.Z, 1e-6)
	require.InDelta(t, -p.StageGap, front.Max.Z, 1e-6)
	require.InDelta(t, pitch, rear.Min.Z, 1e-6)
	require.InDelta(t, pitch+p.CoverThickness+5, rear.Max.Z, 1e-6)
	frontCoverOuterRadius := p.Module*float64(p.RingTeeth)/2 + p.Module +
		minimumGearRootClearance(p.Module) + p.RingWall
	require.InDelta(t, frontCoverOuterRadius, front.Max.X, 1e-6)
	fb, rb := bounds(t, box.FrontBearing), bounds(t, box.RearBearing)
	require.InDelta(t, front.Min.Z, fb.Min.Z, 1e-6)
	require.InDelta(t, front.Min.Z+5, fb.Max.Z, 1e-6)
	require.InDelta(t, rear.Max.Z-5, rb.Min.Z, 1e-6)
	require.InDelta(t, rear.Max.Z, rb.Max.Z, 1e-6)

	// The input shaft starts InputShaftLength outside the front boss and ends level with the first sun's top.
	shaft := bounds(t, box.InputShaft)
	require.InDelta(t, front.Min.Z-p.InputShaftLength, shaft.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness, shaft.Max.Z, 1e-6)
	// The collar next to the front bearing is wider than the shaft, which has its flat cut above the collar.
	shaftCollarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	require.InDelta(t, shaftCollarDiameter.Base(), shaft.Max.X-shaft.Min.X, 2e-3)
	require.InDelta(t, shaftCollarDiameter.Base(), shaft.Max.Y-shaft.Min.Y, 2e-3)
	require.Equal(t, decad.PointInside, locate(box.InputShaft, r3.NewVec(0, 0, 4)))
	require.Equal(t, decad.PointOutside, locate(box.Stages[0].Sun, r3.NewVec(0, 0, 4)), "the shaft's place is the sun's bore")

	// Between the four bolt holes the ring and both cover rims are solid.
	boltRadius := (35.5 + 1.25 + 35.5 + 1.25 + p.RingWall) / 2
	for i := range 4 {
		hole := math.Pi/4 + float64(i)*math.Pi/2
		between := float64(i) * math.Pi / 2
		for name, c := range map[string]struct {
			body *decad.Body
			z    float64
		}{
			"ring":  {box.Ring, 6},
			"front": {box.FrontCover, -p.StageGap - p.CoverThickness/2},
			"rear":  {box.RearCover, pitch + p.CoverThickness/2},
		} {
			at := func(angle float64) r3.Vec {
				return r3.NewVec(boltRadius*math.Cos(angle), boltRadius*math.Sin(angle), c.z)
			}
			// The hole itself is small, so the point locator cannot decide it. TestFastenersFitTheirPockets
			// checks the bolts clear every body instead.
			_ = hole
			require.Equal(t, decad.PointInside, locate(c.body, at(between)), "%s rim beside hole %d", name, i)
		}
	}

	// A cover is a frame: material on a spoke and none in the window between two spokes.
	for name, body := range map[string]*decad.Body{"front": box.FrontCover, "rear": box.RearCover} {
		z := -p.StageGap - p.CoverThickness/2
		if name == "rear" {
			z = pitch + p.CoverThickness/2
		}
		// The front cover is turned over, so its spokes sit at the mirror angles.
		spoke, window := math.Pi/6, math.Pi/2
		if name == "front" {
			spoke, window = -math.Pi/6, -math.Pi/2
		}
		require.Equal(t, decad.PointInside, locate(body, r3.NewVec(28*math.Cos(spoke), 28*math.Sin(spoke), z)), "%s spoke", name)
		require.Equal(t, decad.PointOutside, locate(body, r3.NewVec(28*math.Cos(window), 28*math.Sin(window), z)), "%s window", name)
		// The rim's inner radius is the bolt circle radius (the cover's outer radius less half the ring wall)
		// less the head pocket radius (the head's radius plus 0.1 mm) and 1.5 mm of material. Each of the
		// three windows reaches out to that radius and no further.
		coverOuter := bounds(t, body).Max.X
		rimInner := coverOuter - p.RingWall/2 - (m2bolt.HeadDiameter/2 + 0.3) - 1.5
		sectionZ := bounds(t, body).Min.Z + 0.5
		if name == "front" {
			sectionZ = bounds(t, body).Max.Z - 0.5
		}
		plane, err := r3.NewFrame(r3.NewVec(0, 0, sectionZ), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
		require.NoError(t, err)
		section := sectionOutline(t, body, plane)
		windows := 0
		for _, loop := range section.Loops {
			reach := 0.0
			for _, point := range loop {
				reach = math.Max(reach, math.Hypot(point.X, point.Y))
			}
			if math.Abs(reach-rimInner) < 0.05 {
				windows++
			}
		}
		require.Equal(t, 3, windows, "%s windows reaching the rim's inner radius %v", name, rimInner)
	}
	// Each cover weighs less than a solid disc of its outer radius and thickness.
	disc := math.Pi * 43.75 * 43.75 * p.CoverThickness
	for name, body := range map[string]*decad.Body{"front": box.FrontCover, "rear": box.RearCover} {
		v, bound := volume(t, body)
		require.Less(t, v+bound, disc, name)
	}
}

// TestShaftFlat checks that the input shaft has one flat on +X where the first sun sits and is a full round cylinder
// where the bearing sits, and that the sun's bore has the matching D: at the flat the shaft is absent and the sun
// has material, elsewhere in the bore the shaft is present and the sun is not.
func TestShaftFlat(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	sun := box.Stages[0].Sun
	locate := func(b *decad.Body, point r3.Vec) decad.PointLocation {
		loc, err := b.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		return loc
	}
	// The flat is 0.5 mm deep on the 5 mm shaft, so it lies 2 mm from the axis, and the sun's flat 2.05 mm from
	// it. The sun's bore is 5.1 mm across, so material at 2.5 mm on +X is the tooth and the bore is open at
	// the middle. The tooth is a sliver under 0.5 mm thick, too thin for point location, so the sun's section
	// decides those probes. The shaft itself is too thin for point location near its surface; the volume check
	// below covers the flat.
	z := p.GearThickness / 2
	require.Equal(t, decad.PointInside, sectionLocate(t, sun, r3.NewVec(2.5, 0.3, z)), "the sun's bore keeps a tooth at the flat")
	require.Equal(t, decad.PointOutside, sectionLocate(t, sun, r3.NewVec(-1, 0.3, z)), "the sun's bore is open opposite the flat")
	require.Equal(t, decad.PointInside, locate(box.InputShaft, r3.NewVec(1, 0.3, z)), "the shaft fills the bore's middle")
	// Under the front bearing the shaft is a full cylinder, flat side included.
	bearingZ := -(p.StageGap + p.CoverThickness + 2.5)
	require.Equal(t, decad.PointInside, locate(box.InputShaft, r3.NewVec(2, 0.3, bearingZ)), "round where the bearing sits")
	b := bounds(t, box.InputShaft)
	length := b.Max.Z - b.Min.Z
	v, bound := volume(t, box.InputShaft)
	// A round rod and a round collar 7.4 mm across in the space between the front bearing's inner race and the
	// sun, less the flat: the circle's part beyond the flat over the flat's length.
	shaftDiameter, err := doc.Param("shaft_diameter")
	require.NoError(t, err)
	shaftCollarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	radius := shaftDiameter.Base() / 2
	segment := circleSegment(radius, p.FlatDepth)
	collarRadius := shaftCollarDiameter.Base() / 2
	collar := (math.Pi*collarRadius*collarRadius - math.Pi*radius*radius) * (p.StageGap + p.CoverThickness - 0.5)
	require.InDelta(t, math.Pi*radius*radius*length+collar-segment*p.GearThickness, v, bound+0.5)
}

// TestOutputShaft checks the thin-thick output shaft: a D thick part level with the carrier's lower face, pressed
// through the carrier's hole and ending 0.25 mm short of the rear bearing, and a thin rod through the bearing.
func TestOutputShaft(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	output := box.Stages[0].Shaft
	kz := p.GearThickness + p.StageGap
	top := bounds(t, box.RearBearing).Min.Z
	require.True(t, output.IsSolid())
	b := bounds(t, output)
	require.InDelta(t, kz, b.Min.Z, 1e-6)
	require.InDelta(t, bounds(t, box.Stages[0].Carrier).Min.Z, b.Min.Z, 1e-6)
	require.InDelta(t, rearBossTop(p, 1)+p.OutputShaftLength, b.Max.Z, 1e-6)
	collarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	shaftDiameter, err := doc.Param("shaft_diameter")
	require.NoError(t, err)
	thickRadius, radius := collarDiameter.Base()/2, shaftDiameter.Base()/2
	want := (math.Pi*thickRadius*thickRadius-circleSegment(thickRadius, p.FlatDepth))*
		(p.CarrierThickness+p.StageGap+p.CoverThickness-0.25) +
		math.Pi*radius*radius*(5+p.OutputShaftLength+0.25)
	v, bound := volume(t, output)
	require.InDelta(t, want, v, bound+0.5)
	require.Equal(t, decad.PointInside, sectionLocate(t, output, r3.NewVec(-3, 0, kz+p.CarrierThickness/2)))
	require.Equal(t, decad.PointOutside, sectionLocate(t, output, r3.NewVec(3.45, 0, kz+p.CarrierThickness/2)), "flat")
	require.Equal(t, decad.PointInside, sectionLocate(t, output, r3.NewVec(-3, 0, top-0.5)), "thick part")
	require.Equal(t, decad.PointOutside, sectionLocate(t, output, r3.NewVec(-3, 0, top-0.125)), "thin rod")
}

func TestPhaseAngles(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	require.Zero(t, planetarygearbox.RingToothAngle(p))
	// An even planet tooth count turns the ring half a tooth.
	q := planetarygearbox.Params{SunTeeth: 14, PlanetTeeth: 30, RingTeeth: 74, PlanetCount: 3}
	require.InDelta(t, math.Pi/74, planetarygearbox.RingToothAngle(q), 1e-12)
	for i := range p.PlanetCount {
		require.InDelta(t, 2*math.Pi*float64(i)/3, planetarygearbox.PlanetOrbitAngle(p, i), 1e-12)
	}
}

func TestTwoStages(t *testing.T) {
	p := twoStageParams()
	doc, box := twoStageGearbox(t)
	require.Len(t, doc.Bodies(), 46)
	require.Len(t, box.Stages, 2)
	requireContainers(t, doc, box)
	require.InDelta(t, (1+71.0/13)*(1+71.0/13), p.Ratio(), 1e-9)
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap

	ring := bounds(t, box.Ring)
	require.InDelta(t, 2*pitch, ring.Max.Z, 1e-6)
	for k, stage := range box.Stages {
		for _, body := range append([]*decad.Body{stage.Sun, stage.Carrier}, stage.Planets...) {
			require.True(t, body.IsSolid())
		}
		require.InDelta(t, float64(k)*pitch, bounds(t, stage.Sun).Min.Z, 1e-6)
		require.InDelta(t, float64(k)*pitch, bounds(t, stage.Planets[0]).Min.Z, 1e-6)
	}
	// The link shaft starts at the first carrier's lower face and ends flush with the top of the second sun.
	link := bounds(t, box.Stages[0].Shaft)
	require.InDelta(t, p.GearThickness+p.StageGap, link.Min.Z, 1e-6)
	require.InDelta(t, pitch+p.GearThickness, link.Max.Z, 1e-6)
	for k, stage := range box.Stages {
		requireInnerLoopsSharp(t, planetarygearbox.CarrierName(k), stage.Carrier, p.PlanetCount+1)
	}
	// The link shaft is thin-thick: a D thick part pressed through the first carrier that ends 0.25 mm short of the
	// second sun, and a thin rod through the sun's bore.
	shaftCollarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	shaftDiameter, err := doc.Param("shaft_diameter")
	require.NoError(t, err)
	thickRadius, radius := shaftCollarDiameter.Base()/2, shaftDiameter.Base()/2
	kz := p.GearThickness + p.StageGap
	require.InDelta(t, shaftCollarDiameter.Base(), link.Max.Y-link.Min.Y, 2e-3)
	require.InDelta(t, shaftCollarDiameter.Base()-p.FlatDepth, link.Max.X-link.Min.X, 2e-3)
	linkShaft := box.Stages[0].Shaft
	require.Equal(t, decad.PointInside, sectionLocate(t, linkShaft, r3.NewVec(-3, 0, kz+p.CarrierThickness/2)))
	require.Equal(t, decad.PointInside, sectionLocate(t, linkShaft, r3.NewVec(-3, 0, pitch-0.5)), "thick part")
	require.Equal(t, decad.PointOutside, sectionLocate(t, linkShaft, r3.NewVec(-3, 0, pitch-0.125)), "thin rod")
	require.Equal(t, decad.PointOutside, sectionLocate(t, linkShaft, r3.NewVec(-3, 0, pitch+p.GearThickness/2)))
	requireSectionClearance(t, linkShaft, box.Stages[0].Carrier, kz+p.CarrierThickness/2, 0.02)
	// The second sun's bore is a press fit on the link shaft, like the first sun's on the input shaft.
	requireSectionClearance(t, linkShaft, box.Stages[1].Sun, pitch+p.GearThickness/2, 0.02)
	want := (math.Pi*thickRadius*thickRadius-circleSegment(thickRadius, p.FlatDepth))*
		(p.CarrierThickness+p.StageGap-0.25) +
		math.Pi*radius*radius*(p.GearThickness+0.25) - circleSegment(radius, p.FlatDepth)*p.GearThickness
	v, bound := volume(t, linkShaft)
	require.InDelta(t, want, v, bound+0.5)
	require.InDelta(t, rearBossTop(p, 2)+p.OutputShaftLength, bounds(t, box.Stages[1].Shaft).Max.Z, 1e-6)
	// The second sun's bore has the tooth for the link shaft's flat on +X. Point location is undecided inside
	// the thin shaft and the tooth, so the sun's section decides: material at the tooth, none on the round side.
	secondSun := box.Stages[1].Sun
	mid := pitch + p.GearThickness/2
	require.Equal(t, decad.PointInside, sectionLocate(t, secondSun, r3.NewVec(2.5, 0.3, mid)), "the second sun's bore keeps a tooth at the flat")
	require.Equal(t, decad.PointOutside, sectionLocate(t, secondSun, r3.NewVec(-1, 0.3, mid)), "the second sun's bore is open opposite the flat")
	// The second stage meshes like the first.
	require.Zero(t, penetration(t, box.Stages[1].Planets[1], box.Stages[1].Sun))
	require.Zero(t, penetration(t, box.Stages[1].Planets[2], box.Ring))

	// With no flat and the tightest gap the carrier's hole and sockets and the thick parts are round.
	t.Run("round sections", func(t *testing.T) {
		q := planetarygearbox.DefaultParams()
		q.Stages, q.FlatDepth, q.StageGap = 2, 0, 0.75
		round, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), q)
		require.NoError(t, err)
		for k, stage := range round.Stages {
			requireInnerLoopsSharp(t, planetarygearbox.CarrierName(k), stage.Carrier, q.PlanetCount+1)
		}
		pitch := q.GearThickness + q.CarrierThickness + 1.5
		kz := q.GearThickness + 0.75
		link := round.Stages[0].Shaft
		requireSectionClearance(t, link, round.Stages[0].Carrier, kz+q.CarrierThickness/2, 0.02)
		requireSectionClearance(t, round.Stages[1].Shaft, round.Stages[1].Carrier, pitch+kz+q.CarrierThickness/2, 0.02)
		require.InDelta(t, pitch+q.GearThickness, bounds(t, link).Max.Z, 1e-6)
		require.Equal(t, decad.PointOutside, sectionLocate(t, link, r3.NewVec(-3, 0, pitch-0.125)))
	})
}

func TestPlacedGearbox(t *testing.T) {
	p := placedParams(t)
	_, box := placedGearbox(t)
	_, world := defaultGearbox(t)
	// U=+Y, V=+Z, so the axis is +X: the reference plane is x=10.
	sun := bounds(t, box.Stages[0].Sun)
	require.InDelta(t, 10, sun.Min.X, 1e-6)
	require.InDelta(t, 10+p.GearThickness, sun.Max.X, 1e-6)
	require.InDelta(t, -5, (sun.Min.Y+sun.Max.Y)/2, 0.15)
	require.InDelta(t, 7, (sun.Min.Z+sun.Max.Z)/2, 0.15)
	for i, nb := range box.Named() {
		a, err := nb.Body.Volume()
		require.NoError(t, err)
		b, err := world.Named()[i].Body.Volume()
		require.NoError(t, err)
		require.InDelta(t, b.Value.Base(), a.Value.Base(), a.Bound.Base()+b.Bound.Base()+0.05, nb.Name)
	}
}

func TestInvalidParams(t *testing.T) {
	for name, tc := range map[string]struct {
		edit func(*planetarygearbox.Params)
		want string
	}{
		"ring not sun plus two planets": {func(p *planetarygearbox.Params) { p.RingTeeth = 72 }, "ring teeth"},
		"planets cannot be spaced":      {func(p *planetarygearbox.Params) { p.PlanetCount = 5 }, "planet count"},
		"planets have too little clearance": {func(p *planetarygearbox.Params) {
			p.SunTeeth, p.PlanetTeeth, p.RingTeeth, p.PlanetCount = 13, 29, 71, 3
			p.PlanetCount = 6
		}, "clearance"},
		"single planet":                    {func(p *planetarygearbox.Params) { p.PlanetCount = 1 }, "planet count"},
		"no stages":                        {func(p *planetarygearbox.Params) { p.Stages = 0 }, "stages"},
		"zero module":                      {func(p *planetarygearbox.Params) { p.Module = 0 }, "module"},
		"bolt holes too big":               {func(p *planetarygearbox.Params) { p.BoltHoleDiameter = 6 }, "bolt holes"},
		"thin ring wall":                   {func(p *planetarygearbox.Params) { p.RingWall = 3 }, "bolt holes"},
		"carrier wall too wide":            {func(p *planetarygearbox.Params) { p.CarrierWall = 12 }, "no room for spokes"},
		"thin gear":                        {func(p *planetarygearbox.Params) { p.GearThickness = 5.5 }, "gear thickness"},
		"stage gap too short":              {func(p *planetarygearbox.Params) { p.StageGap = 0.5 }, "stage gap"},
		"flat too deep":                    {func(p *planetarygearbox.Params) { p.FlatDepth = 3 }, "flat depth"},
		"flat too deep for the thick part": {func(p *planetarygearbox.Params) { p.FlatDepth = 1.2 }, "collar margin"},
		"carrier wall too thin for a socket": {func(p *planetarygearbox.Params) { p.CarrierWall = 2.2 },
			"around a pin socket"},
		"seat chamfer too big": {func(p *planetarygearbox.Params) { p.SeatChamfer = 2 }, "seat chamfer"},
		"negative clearance":   {func(p *planetarygearbox.Params) { p.BearingClearance = -0.1 }, "bearing clearance"},
		"bad placement":        {func(p *planetarygearbox.Params) { p.Placement = &r3.Frame{} }, "placement"},
		"nonfinite":            {func(p *planetarygearbox.Params) { p.GearThickness = math.NaN() }, "finite"},
		"few samples":          {func(p *planetarygearbox.Params) { p.Samples = 2 }, "samples"},
		"negative backlash":    {func(p *planetarygearbox.Params) { p.Backlash = -0.1 }, "backlash"},
		"large backlash":       {func(p *planetarygearbox.Params) { p.Backlash = 0.6 }, "backlash"},
		"pressure angle":       {func(p *planetarygearbox.Params) { p.PressureAngle = math.Pi / 2 }, "pressure angle"},
		"tiny sun":             {func(p *planetarygearbox.Params) { p.SunTeeth = 2 }, "sun teeth"},
		"negative ear count":   {func(p *planetarygearbox.Params) { p.EarCount = -1 }, "ear count"},
		"ears without a bore":  {func(p *planetarygearbox.Params) { p.EarBoreDiameter = 0 }, "ear bore diameter"},
		"negative ear bore":    {func(p *planetarygearbox.Params) { p.EarBoreDiameter = -4 }, "ear bore diameter"},
		"nonfinite ear bore":   {func(p *planetarygearbox.Params) { p.EarBoreDiameter = math.Inf(1) }, "finite"},
		"ears over the bolts":  {func(p *planetarygearbox.Params) { p.EarCount = 8 }, "bolt site"},
		"ears too close":       {func(p *planetarygearbox.Params) { p.EarCount, p.EarBoreDiameter = 20, 20 }, "neighbouring ear"},
	} {
		t.Run(name, func(t *testing.T) {
			p := planetarygearbox.DefaultParams()
			tc.edit(&p)
			_, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
			require.ErrorContains(t, err, tc.want)
		})
	}
	_, err := planetarygearbox.Build(nil, decad.New(), planetarygearbox.DefaultParams()) //nolint:staticcheck
	require.Error(t, err)
	_, err = planetarygearbox.Build(t.Context(), nil, planetarygearbox.DefaultParams())
	require.Error(t, err)
}

func TestClearanceLimits(t *testing.T) {
	t.Run("bolt holes allow exactly 1.5 mm of ring material", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.BoltHoleDiameter = 4.1
		require.NoError(t, p.Validate())
		p.BoltHoleDiameter += 0.01
		require.ErrorContains(t, p.Validate(), "bolt holes")
	})

	t.Run("bearing pocket allows exactly one module of root wall", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.BearingClearance = maximumBearingClearanceAtRootWall(p)
		require.NoError(t, p.Validate())
		p.BearingClearance += 0.01
		require.ErrorContains(t, p.Validate(), "bearing pocket")
	})

	t.Run("pin socket keeps 1 mm of plate", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.CarrierWall = 2.3
		require.NoError(t, p.Validate())
		p.CarrierWall = 2.2
		require.ErrorContains(t, p.Validate(), "around a pin socket")
	})

	t.Run("pin socket keeps 0.5 mm at the chamfered face", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.CarrierWall, p.EdgeChamfer = 2.9, 1
		require.NoError(t, p.Validate())
		p.CarrierWall = 2.3
		require.ErrorContains(t, p.Validate(), "at the plate's face")
	})

	t.Run("carrier must stay more than half a module inside the ring tip", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.BoltHoleDiameter = 3.4
		p.RingWall = 13
		p.CarrierWall = 10.49
		require.NoError(t, p.Validate())
		p.CarrierWall = 10.5
		require.ErrorContains(t, p.Validate(), "carrier radius")
	})
}

// TestSeatChamfer checks that the entrance of each bearing seat, the planet's pocket and the covers' boss pocket,
// is chamfered: the body loses about a ring of triangular section, the setback squared over two times the
// pocket's circumference, against the same gearbox with no chamfer.
func TestSeatChamfer(t *testing.T) {
	sharp := planetarygearbox.DefaultParams()
	sharp.SeatChamfer = 0
	_, plain := func() (*decad.Document, planetarygearbox.Gearbox) {
		doc := planetarygearbox.NewDocument()
		box, err := planetarygearbox.Build(t.Context(), doc, sharp)
		require.NoError(t, err)
		return doc, box
	}()
	_, chamfered := defaultGearbox(t)
	p := planetarygearbox.DefaultParams()
	ring := math.Pi * (14 + 2*p.BearingClearance) * p.SeatChamfer * p.SeatChamfer / 2
	for name, pair := range map[string][2]*decad.Body{
		"planet": {plain.Stages[0].Planets[0], chamfered.Stages[0].Planets[0]},
		"cover":  {plain.RearCover, chamfered.RearCover},
	} {
		before, bound := volume(t, pair[0])
		after, afterBound := volume(t, pair[1])
		require.InDelta(t, ring, before-after, bound+afterBound+1, name)
		require.True(t, pair[1].IsSolid(), name)
	}
}
