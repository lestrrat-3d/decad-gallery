package planetarygearbox_test

import (
	"math"
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
	return planetRoot - p.Module - ballbearing.Params608ZZ().OuterDiameter/2
}

var (
	defaultOnce sync.Once
	defaultDoc  *decad.Document
	defaultBox  planetarygearbox.Gearbox
	defaultErr  error
)

// defaultGearbox builds the default gearbox once for the tests that read it.
func defaultGearbox(t *testing.T) (*decad.Document, planetarygearbox.Gearbox) {
	t.Helper()
	defaultOnce.Do(func() {
		defaultDoc = decad.New()
		defaultBox, defaultErr = planetarygearbox.Build(t.Context(), defaultDoc, planetarygearbox.DefaultParams())
	})
	require.NoError(t, defaultErr)
	return defaultDoc, defaultBox
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
	require.Len(t, doc.Bodies(), 31)
	require.Len(t, box.Stages, 1)
	require.Len(t, box.Stages[0].Planets, 3)
	require.Len(t, box.Stages[0].Bearings, 3)
	require.Len(t, box.Named(), 31)
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
	require.InDelta(t, 35.5+p.Module+minimumGearRootClearance(p.Module)+
		p.RingWall, ring.Max.X, 1e-3)

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

	// The carrier's plate sits above the gears and its pins reach down to the gears' lower face. Nothing
	// stands above the plate, so the carrier prints plate-down.
	carrier := bounds(t, stage.Carrier)
	require.InDelta(t, 0, carrier.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness+p.StageGap+p.CarrierThickness, carrier.Max.Z, 1e-6)
	// The plate reaches the far side of the pin bosses and no further: orbit radius plus boss radius.
	require.InDelta(t, 21+(7.5+2*p.CarrierWall)/2, carrier.Max.X, 5e-3)
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
	outlineA, err := a.SectionOutline(t.Context(), plane, units.Millimeters(0.01))
	require.NoError(t, err)
	outlineB, err := b.SectionOutline(t.Context(), plane, units.Millimeters(0.01))
	require.NoError(t, err)
	clearance, err := outlineA.Clearance(outlineB)
	require.NoError(t, err)
	require.False(t, clearance.Overlaps)
	require.GreaterOrEqual(t, clearance.Lower().Base(), minimum,
		"section gap %v ± %v mm, closest points %v and %v", clearance.Distance.Value.Base(),
		clearance.Distance.Bound.Base(), clearance.A, clearance.B)
}

type radialSpan struct {
	min, max, bound float64
}

func sectionRadialSpan(t *testing.T, body *decad.Body, center r3.Vec, z, targetRadius float64) radialSpan {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	section, err := body.SectionOutline(t.Context(), plane, units.Millimeters(0.01))
	require.NoError(t, err)
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

	requireSectionClearance(t, box.InputShaft, stage.Sun, p.GearThickness/2, minimum)
	requireSectionClearance(t, stage.Carrier, stage.Shaft,
		p.GearThickness+p.StageGap+p.CarrierThickness/2, minimum)
	requireSectionClearance(t, planetBearing, stage.Planets[0], bearingMid, minimum)
	requireSectionClearance(t, planetBearing, stage.Carrier, bearingMid, minimum)
	frontFace := -p.StageGap - p.CoverThickness
	requireSectionClearance(t, box.Bolts[0], box.FrontCover,
		frontFace+m2bolt.HeadHeight/2, minimumFastenerGapBound)
	requireSectionClearance(t, box.Bolts[0], box.Ring, 0, minimumFastenerGapBound)
	nut := bounds(t, box.Nuts[0])
	requireSectionClearance(t, box.Nuts[0], box.Ring, (nut.Min.Z+nut.Max.Z)/2, minimumFastenerGapBound)

	frontBearing := bounds(t, box.FrontBearing)
	requireSectionClearance(t, box.InputShaft, box.FrontBearing,
		(frontBearing.Min.Z+frontBearing.Max.Z)/2, minimum)
	requireSectionClearance(t, box.FrontBearing, box.FrontCover,
		(frontBearing.Min.Z+frontBearing.Max.Z)/2, minimum)
	rearBearing := bounds(t, box.RearBearing)
	rearBearingMid := (rearBearing.Min.Z + rearBearing.Max.Z) / 2
	requireSectionClearance(t, stage.Shaft, box.RearBearing, rearBearingMid, minimum)
	requireSectionClearance(t, box.RearBearing, box.RearCover, rearBearingMid, minimum)

	branchParams := p
	branchParams.KeyDepth = 0
	branchParams.StageGap = 0.75
	branchParams.BearingClearance = 0.5
	branchParams.SeatChamfer = 0
	branch, err := planetarygearbox.Build(t.Context(), decad.New(), branchParams)
	require.NoError(t, err)
	branchBearing := branch.Stages[0].Bearings[0]
	branchBearingBounds := bounds(t, branchBearing)
	branchBearingMid := (branchBearingBounds.Min.Z + branchBearingBounds.Max.Z) / 2
	requireSectionClearance(t, branchBearing, branch.Stages[0].Planets[0], branchBearingMid, minimum)
	requireSectionClearance(t, branchBearing, branch.Stages[0].Carrier, branchBearingMid, minimum)
	requireSectionClearance(t, branch.Stages[0].Carrier, branch.Stages[0].Shaft,
		p.GearThickness+branchParams.StageGap+p.CarrierThickness/2, minimum)
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
	loose, err := planetarygearbox.Build(t.Context(), decad.New(), p)
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
// the bearing's 608ZZ envelope in it, and that the lip under the pocket is the only material cut back:
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
		Samples: p.Samples, BoreDiameter: 18, Backlash: p.Backlash,
	})
	require.NoError(t, err)
	plainVolume, plainBound := volume(t, plain)
	pocketDiameter := 22 + 2*p.BearingClearance
	pocket := math.Pi / 4 * (pocketDiameter*pocketDiameter - 18*18) * 7

	for i, planet := range stage.Planets {
		phi := planetarygearbox.PlanetOrbitAngle(p, i)
		around := func(radius, angle, z float64) r3.Vec {
			return r3.NewVec(21*math.Cos(phi)+radius*math.Cos(angle), 21*math.Sin(phi)+radius*math.Sin(angle), z)
		}
		locate := func(b *decad.Body, point r3.Vec) decad.PointLocation {
			loc, err := b.LocatePoint(t.Context(), point, units.Millimeters(0.02))
			require.NoError(t, err)
			return loc
		}
		for _, radius := range []float64{5, 8, 9.5} {
			for _, z := range []float64{3.5, 5, 6.5} {
				for _, angle := range []float64{0.7, 2.5, 4.4} {
					point := around(radius, angle, z)
					require.Equal(t, decad.PointOutside, locate(planet, point), "planet %d pocket at %v", i, point)
					require.Equal(t, decad.PointInside, locate(stage.Bearings[i], point), "bearing %d at %v", i, point)
				}
			}
		}

		planetVolume, planetBound := volume(t, planet)
		// The pocket's entrance is chamfered, which removes a ring of triangular section.
		entrance := math.Pi * (22 + 2*p.BearingClearance) * p.SeatChamfer * p.SeatChamfer / 2
		require.InDelta(t, plainVolume-pocket-entrance, planetVolume, plainBound+planetBound+1, "planet %d volume", i)

		// The bearing is the 608ZZ envelope, 8 mm bore, 22 mm outside and 7 mm wide, centered on the
		// planet, raised above the pocket floor by 0.25 mm.
		b := bounds(t, stage.Bearings[i])
		require.InDelta(t, 22, b.Max.X-b.Min.X, 0.05, "bearing %d outside diameter", i)
		require.InDelta(t, bearingWidth.Base(), b.Max.Z-b.Min.Z, 1e-6, "bearing %d width", i)
		require.InDelta(t, p.GearThickness-bearingWidth.Base()+bearingLift.Base(), b.Min.Z, 1e-6, "bearing %d bottom", i)
		require.InDelta(t, p.GearThickness+bearingLift.Base(), b.Max.Z, 1e-6, "bearing %d top", i)
	}
}

// TestCarrierIsATriangle checks that the carrier's plate is the solid triangle between the pin bosses with
// the shaft's D hole in the middle, and that its pins and their collars stand on one face. Points are probed
// 1.5 mm or more from a surface where the body is thick, because point location is undecided closer than
// about 0.15 mm.
func TestCarrierIsATriangle(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
	carrier := box.Stages[0].Carrier
	locate := func(point r3.Vec) decad.PointLocation {
		loc, err := carrier.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		return loc
	}
	plate := p.GearThickness + p.StageGap + p.CarrierThickness/2
	at := func(radius, angle, z float64) r3.Vec {
		return r3.NewVec(radius*math.Cos(angle), radius*math.Sin(angle), z)
	}
	// Between two pins, at 60 degrees, the plate is solid from the hole out to the tangent edge, 10.5 mm
	// (half the pin spacing's height) plus the 7 mm boss radius = 17.5 mm from the axis.
	require.Equal(t, decad.PointInside, locate(at(12, math.Pi/3, plate)), "solid between the pins")
	require.Equal(t, decad.PointInside, locate(at(15, math.Pi/3, plate)), "solid out to the edge")
	require.Equal(t, decad.PointOutside, locate(at(20, math.Pi/3, plate)), "nothing past the tangent edge")
	require.Equal(t, decad.PointInside, locate(at(21, 0, plate)), "a pin boss")
	require.Equal(t, decad.PointOutside, locate(at(0, 0, plate)), "the D hole")
	require.Equal(t, decad.PointOutside, locate(r3.NewVec(-2.5, 0, plate)), "the hole is open on the round side")
	require.Equal(t, decad.PointInside, locate(r3.NewVec(6, 0, plate)), "the plate is solid just outside the hole")
	// A pin stands below the plate.
	require.Equal(t, decad.PointInside, locate(at(21, 0, 4)), "pin")
	require.Equal(t, decad.PointOutside, locate(at(21+6, 0, 4)), "beside the pin")
	// The carrier weighs more than the plain triangle's pins and less than a disc that reaches its pins.
	v, bound := volume(t, carrier)
	disc := math.Pi * 28 * 28 * p.CarrierThickness
	require.Less(t, v-bound, disc+3*math.Pi*16*(p.GearThickness+p.StageGap+p.CarrierThickness/2))
}

// rearBossTop is the height of the rear cover's bearing boss top for a gearbox of the given stages.
func rearBossTop(p planetarygearbox.Params, stages int) float64 {
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap
	return float64(stages)*pitch + p.CoverThickness + 7
}

// TestHousing checks the ring, the two covers, their bearings and the input shaft: where they sit along the
// axis, that the bolt holes of the ring and both covers line up, that the covers are frames with windows and
// not discs, and that the input shaft fills the first sun's bore. Points are probed 1.5 mm or more from any
// surface, because point location is undecided closer than about 0.15 mm.
func TestHousing(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	locate := func(b *decad.Body, point r3.Vec) decad.PointLocation {
		loc, err := b.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		return loc
	}
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap

	// Axial layout. Each cover's frame lies against a ring end, with its bearing boss pointing away.
	front, rear := bounds(t, box.FrontCover), bounds(t, box.RearCover)
	require.InDelta(t, -p.StageGap-p.CoverThickness-7, front.Min.Z, 1e-6)
	require.InDelta(t, -p.StageGap, front.Max.Z, 1e-6)
	require.InDelta(t, pitch, rear.Min.Z, 1e-6)
	require.InDelta(t, pitch+p.CoverThickness+7, rear.Max.Z, 1e-6)
	frontCoverOuterRadius := p.Module*float64(p.RingTeeth)/2 + p.Module +
		minimumGearRootClearance(p.Module) + p.RingWall
	require.InDelta(t, frontCoverOuterRadius, front.Max.X, 1e-6)
	fb, rb := bounds(t, box.FrontBearing), bounds(t, box.RearBearing)
	require.InDelta(t, front.Min.Z, fb.Min.Z, 1e-6)
	require.InDelta(t, front.Min.Z+7, fb.Max.Z, 1e-6)
	require.InDelta(t, rear.Max.Z-7, rb.Min.Z, 1e-6)
	require.InDelta(t, rear.Max.Z, rb.Max.Z, 1e-6)

	// The input shaft starts InputShaftLength outside the front boss and ends level with the first sun's top.
	shaft := bounds(t, box.InputShaft)
	require.InDelta(t, front.Min.Z-p.InputShaftLength, shaft.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness, shaft.Max.Z, 1e-6)
	// The collar next to the front bearing is wider than the shaft, which has its keyway cut above the collar.
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
		rimInner := coverOuter - p.RingWall/2 - (m2bolt.HeadDiameter/2 + 0.1) - 1.5
		sectionZ := bounds(t, body).Min.Z + 0.5
		if name == "front" {
			sectionZ = bounds(t, body).Max.Z - 0.5
		}
		plane, err := r3.NewFrame(r3.NewVec(0, 0, sectionZ), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
		require.NoError(t, err)
		section, err := body.SectionOutline(t.Context(), plane, units.Millimeters(0.01))
		require.NoError(t, err)
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

// TestShaftKeys checks that the input shaft has three keyway grooves, 120 degrees apart, where the first sun sits and is a
// full round cylinder where the bearing sits, and that the sun's bore has the matching tooth: in the groove the
// shaft is absent and the sun has material, elsewhere in the bore the shaft is present and the sun is not.
func TestShaftKeys(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	sun := box.Stages[0].Sun
	locate := func(b *decad.Body, point r3.Vec) decad.PointLocation {
		loc, err := b.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		return loc
	}
	// Each groove is 1.1 mm deep and 2 mm wide on the 7.5 mm shaft. The point at radius 3 lies in the groove
	// and inside the round section, far enough from both surfaces to be decided.
	// Each groove is centered on an angle 120 degrees from the next, and the shaft is round between them.
	at := func(radius, degrees, z float64) r3.Vec {
		sin, cos := math.Sincos(degrees * math.Pi / 180)
		return r3.NewVec(radius*cos, radius*sin, z)
	}
	for _, degrees := range []float64{0, 120, 240} {
		key := at(3.5, degrees, p.GearThickness/2)
		require.Equal(t, decad.PointOutside, locate(box.InputShaft, key), "the shaft has a groove at %v degrees", degrees)
		require.Equal(t, decad.PointInside, locate(sun, key), "the sun's bore keeps a tooth at %v degrees", degrees)
	}
	for _, degrees := range []float64{60, 180, 300} {
		round := at(2, degrees, 4)
		require.Equal(t, decad.PointInside, locate(box.InputShaft, round), "shaft at %v degrees", degrees)
		require.Equal(t, decad.PointOutside, locate(sun, round), "sun bore at %v degrees", degrees)
	}
	// Under the front bearing the shaft is a full cylinder, key side included.
	bearingZ := -(p.StageGap + p.CoverThickness + 3.5)
	require.Equal(t, decad.PointInside, locate(box.InputShaft, r3.NewVec(2, 0, bearingZ)), "round where the bearing sits")
	const keyCount = 3
	b := bounds(t, box.InputShaft)
	length := b.Max.Z - b.Min.Z
	v, bound := volume(t, box.InputShaft)
	// A round rod and a round collar 10.4 mm across in the space between the front bearing's inner race and the
	// sun, less the groove: the circle's part inside the groove's width, from the floor out, over the groove's
	// length.
	shaftDiameter, err := doc.Param("shaft_diameter")
	require.NoError(t, err)
	shaftCollarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	radius, depth, half := shaftDiameter.Base()/2, p.KeyDepth, p.KeyWidth/2
	floor := radius - depth
	groove := 2 * (half*math.Sqrt(radius*radius-half*half)/2 + radius*radius/2*math.Asin(half/radius) - floor*half)
	collarRadius := shaftCollarDiameter.Base() / 2
	collar := (math.Pi*collarRadius*collarRadius - math.Pi*radius*radius) * (p.StageGap + p.CoverThickness - 0.5)
	grooveLength := p.GearThickness
	require.InDelta(t, math.Pi*radius*radius*length+collar-keyCount*groove*grooveLength, v, bound+0.5)
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
	p := planetarygearbox.DefaultParams()
	p.Stages = 2
	doc := decad.New()
	box, err := planetarygearbox.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 40)
	require.Len(t, box.Stages, 2)
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
	// A collar in the gap between the first carrier and the second sun stops 0.25 mm short
	// of both, and the grooves are cut away above and below it, so it is whole where a groove would run.
	shaftCollarDiameter, err := doc.Param("shaft_collar_diameter")
	require.NoError(t, err)
	require.InDelta(t, shaftCollarDiameter.Base(), link.Max.X-link.Min.X, 2e-3)
	linkShaft := box.Stages[0].Shaft
	collarMid := p.GearThickness + p.StageGap + p.CarrierThickness + p.StageGap/2
	locateLink := func(point r3.Vec) decad.PointLocation {
		loc, err := linkShaft.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		return loc
	}
	require.Equal(t, decad.PointInside, locateLink(r3.NewVec(3.5, 0, collarMid)), "the collar is whole over the groove")
	require.Equal(t, decad.PointOutside, locateLink(r3.NewVec(3.5, 0, collarMid+p.StageGap)), "no collar above it")
	require.InDelta(t, rearBossTop(p, 2)+p.OutputShaftLength, bounds(t, box.Stages[1].Shaft).Max.Z, 1e-6)
	// The second sun's bore has the tooth for the link shaft's groove on +X. Point location is undecided inside
	// the thin shaft, so only the sun is probed: material at the tooth, none on the round side.
	secondSun := box.Stages[1].Sun
	locate := func(point r3.Vec) decad.PointLocation {
		loc, err := secondSun.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		return loc
	}
	mid := pitch + p.GearThickness/2
	require.Equal(t, decad.PointInside, locate(r3.NewVec(3.5, 0, mid)), "the second sun's bore keeps a tooth in the groove")
	require.Equal(t, decad.PointOutside, locate(r3.NewVec(-3.5, 0, mid)), "the second sun's bore is round opposite the groove")
	// The second stage meshes like the first.
	require.Zero(t, penetration(t, box.Stages[1].Planets[1], box.Stages[1].Sun))
	require.Zero(t, penetration(t, box.Stages[1].Planets[2], box.Ring))
}

func TestPlacedGearbox(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p.Placement = &f
	box, err := planetarygearbox.Build(t.Context(), decad.New(), p)
	require.NoError(t, err)
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
		"single planet":         {func(p *planetarygearbox.Params) { p.PlanetCount = 1 }, "planet count"},
		"no stages":             {func(p *planetarygearbox.Params) { p.Stages = 0 }, "stages"},
		"zero module":           {func(p *planetarygearbox.Params) { p.Module = 0 }, "module"},
		"bolt holes too big":    {func(p *planetarygearbox.Params) { p.BoltHoleDiameter = 6 }, "bolt holes"},
		"thin ring wall":        {func(p *planetarygearbox.Params) { p.RingWall = 3 }, "bolt holes"},
		"carrier wall too wide": {func(p *planetarygearbox.Params) { p.CarrierWall = 12 }, "no room for spokes"},
		"thin gear":             {func(p *planetarygearbox.Params) { p.GearThickness = 7.5 }, "gear thickness"},
		"stage gap too short":   {func(p *planetarygearbox.Params) { p.StageGap = 0.5 }, "stage gap"},
		"key too deep":          {func(p *planetarygearbox.Params) { p.KeyDepth = 3 }, "key depth"},
		"key too wide":          {func(p *planetarygearbox.Params) { p.KeyWidth = 5 }, "key width"},
		"seat chamfer too big":  {func(p *planetarygearbox.Params) { p.SeatChamfer = 2 }, "seat chamfer"},
		"negative clearance":    {func(p *planetarygearbox.Params) { p.BearingClearance = -0.1 }, "bearing clearance"},
		"bad placement":         {func(p *planetarygearbox.Params) { p.Placement = &r3.Frame{} }, "placement"},
		"nonfinite":             {func(p *planetarygearbox.Params) { p.GearThickness = math.NaN() }, "finite"},
		"few samples":           {func(p *planetarygearbox.Params) { p.Samples = 2 }, "samples"},
		"negative backlash":     {func(p *planetarygearbox.Params) { p.Backlash = -0.1 }, "backlash"},
		"large backlash":        {func(p *planetarygearbox.Params) { p.Backlash = 0.6 }, "backlash"},
		"pressure angle":        {func(p *planetarygearbox.Params) { p.PressureAngle = math.Pi / 2 }, "pressure angle"},
		"tiny sun":              {func(p *planetarygearbox.Params) { p.SunTeeth = 2 }, "sun teeth"},
	} {
		t.Run(name, func(t *testing.T) {
			p := planetarygearbox.DefaultParams()
			tc.edit(&p)
			_, err := planetarygearbox.Build(t.Context(), decad.New(), p)
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
		p.BoltHoleDiameter = 4
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

	t.Run("carrier must stay more than half a module inside the ring tip", func(t *testing.T) {
		p := planetarygearbox.DefaultParams()
		p.BoltHoleDiameter = 3.4
		p.RingWall = 13
		p.CarrierWall = 9.24
		require.NoError(t, p.Validate())
		p.CarrierWall = 9.25
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
		doc := decad.New()
		box, err := planetarygearbox.Build(t.Context(), doc, sharp)
		require.NoError(t, err)
		return doc, box
	}()
	_, chamfered := defaultGearbox(t)
	p := planetarygearbox.DefaultParams()
	ring := math.Pi * (22 + 2*p.BearingClearance) * p.SeatChamfer * p.SeatChamfer / 2
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
