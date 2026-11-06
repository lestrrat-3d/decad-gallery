package planetarygearbox_test

import (
	"math"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

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
	require.Len(t, doc.Bodies(), 9)
	require.Len(t, box.Stages, 1)
	require.Len(t, box.Stages[0].Planets, 3)
	require.Len(t, box.Stages[0].Bearings, 3)
	require.Len(t, box.Named(), 9)
	for _, nb := range box.Named() {
		require.True(t, nb.Body.IsSolid(), nb.Name)
		v, err := nb.Body.Volume()
		require.NoError(t, err, nb.Name)
		require.Positive(t, v.Value.Base(), nb.Name)
	}
	require.InDelta(t, 1+71.0/13, p.Ratio(), 1e-12)

	// The ring spans the gap below the first stage to one stage pitch, with its outer wall
	// at the ring root radius plus the wall.
	ring := bounds(t, box.Ring)
	require.InDelta(t, -p.StageGap, ring.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness+p.CarrierThickness+2*p.StageGap, ring.Max.Z, 1e-6)
	require.InDelta(t, 35.5+1.25+p.RingWall, ring.Max.X, 1e-3)

	stage := box.Stages[0]
	sun := bounds(t, stage.Sun)
	require.InDelta(t, 0, sun.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness, sun.Max.Z, 1e-6)
	require.InDelta(t, 7.5, sun.Max.X, 1e-2)
	for i, planet := range stage.Planets {
		b := bounds(t, planet)
		phi := planetarygearbox.PlanetOrbitAngle(p, i)
		require.InDelta(t, 21*math.Cos(phi), (b.Min.X+b.Max.X)/2, 0.05, "planet %d", i)
		require.InDelta(t, 21*math.Sin(phi), (b.Min.Y+b.Max.Y)/2, 0.05, "planet %d", i)
		require.InDelta(t, p.GearThickness, b.Max.Z-b.Min.Z, 1e-6)
	}

	// The carrier's arms sit above the gears; its pins reach down to the gears' lower face and
	// its stub is the output shaft.
	carrier := bounds(t, stage.Carrier)
	require.InDelta(t, 0, carrier.Min.Z, 1e-6)
	require.InDelta(t, p.GearThickness+p.StageGap+p.CarrierThickness+p.OutputShaftLength, carrier.Max.Z, 1e-6)
	// The arms reach the far side of the pin bosses and no further: orbit radius plus boss radius.
	require.InDelta(t, 21+7, carrier.Max.X, 1e-6)
}

// penetration counts the vertices of either body that lie inside the other, beyond a small
// tolerance. Vertices of a flank in true contact are on the boundary, not inside.
func penetration(t *testing.T, a, b *decad.Body) int {
	t.Helper()
	return inside(t, a, b) + inside(t, b, a)
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
	_, box := defaultGearbox(t)
	stage := box.Stages[0]

	plain, err := spurgear.Build(t.Context(), decad.New(), spurgear.Params{
		Module: p.Module, Teeth: p.PlanetTeeth, PressureAngle: p.PressureAngle, Thickness: p.GearThickness,
		Samples: p.Samples, BoreDiameter: 18, Backlash: p.Backlash,
	})
	require.NoError(t, err)
	plainVolume, plainBound := volume(t, plain)
	pocket := math.Pi / 4 * (22.2*22.2 - 18*18) * 7

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
		require.InDelta(t, plainVolume-pocket, planetVolume, plainBound+planetBound+1, "planet %d volume", i)

		// The bearing is the 608ZZ envelope, 8 mm bore, 22 mm outside and 7 mm wide, centered on the
		// planet, with its upper face level with the planet's.
		b := bounds(t, stage.Bearings[i])
		require.InDelta(t, 22, b.Max.X-b.Min.X, 0.05, "bearing %d outside diameter", i)
		require.InDelta(t, 7, b.Max.Z-b.Min.Z, 1e-6, "bearing %d width", i)
		require.InDelta(t, p.GearThickness, b.Max.Z, 1e-6, "bearing %d top", i)
	}
}

// TestCarrierHasNoDisc checks that material is only along the arms: it is present on an arm's axis and
// absent between two arms at the same radius, and the carrier weighs less than a plate that reaches
// as far as the arms.
func TestCarrierHasNoDisc(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
	carrier := box.Stages[0].Carrier
	z := p.GearThickness + p.StageGap + p.CarrierThickness/2
	for _, c := range []struct {
		angle float64
		want  decad.PointLocation
	}{{0, decad.PointInside}, {2 * math.Pi / 3, decad.PointInside}, {math.Pi / 3, decad.PointOutside}} {
		point := r3.NewVec(15*math.Cos(c.angle), 15*math.Sin(c.angle), z)
		loc, err := carrier.LocatePoint(t.Context(), point, units.Millimeters(0.02))
		require.NoError(t, err)
		require.Equal(t, c.want, loc, "angle %v", c.angle)
	}
	volume, bound := volume(t, carrier)
	plate := math.Pi * 28 * 28 * p.CarrierThickness
	require.Less(t, volume+bound, plate)
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
	require.Len(t, doc.Bodies(), 17)
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
	// The first carrier's stub ends flush with the top of the second sun.
	require.InDelta(t, pitch+p.GearThickness, bounds(t, box.Stages[0].Carrier).Max.Z, 1e-6)
	require.InDelta(t, pitch+p.GearThickness+p.StageGap+p.CarrierThickness+p.OutputShaftLength,
		bounds(t, box.Stages[1].Carrier).Max.Z, 1e-6)
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
		"planets collide": {func(p *planetarygearbox.Params) {
			p.SunTeeth, p.PlanetTeeth, p.RingTeeth, p.PlanetCount = 13, 29, 71, 3
			p.PlanetCount = 6
		}, "collide"},
		"single planet":      {func(p *planetarygearbox.Params) { p.PlanetCount = 1 }, "planet count"},
		"no stages":          {func(p *planetarygearbox.Params) { p.Stages = 0 }, "stages"},
		"zero module":        {func(p *planetarygearbox.Params) { p.Module = 0 }, "module"},
		"shaft too wide":     {func(p *planetarygearbox.Params) { p.ShaftDiameter = 9 }, "shaft diameter"},
		"carrier too wide":   {func(p *planetarygearbox.Params) { p.CarrierWall = 12 }, "carrier arms"},
		"thin gear":          {func(p *planetarygearbox.Params) { p.GearThickness = 7.5 }, "gear thickness"},
		"negative clearance": {func(p *planetarygearbox.Params) { p.BearingClearance = -0.1 }, "bearing clearance"},
		"bad placement":      {func(p *planetarygearbox.Params) { p.Placement = &r3.Frame{} }, "placement"},
		"nonfinite":          {func(p *planetarygearbox.Params) { p.GearThickness = math.NaN() }, "finite"},
		"few samples":        {func(p *planetarygearbox.Params) { p.Samples = 2 }, "samples"},
		"negative backlash":  {func(p *planetarygearbox.Params) { p.Backlash = -0.1 }, "backlash"},
		"large backlash":     {func(p *planetarygearbox.Params) { p.Backlash = 0.6 }, "backlash"},
		"pressure angle":     {func(p *planetarygearbox.Params) { p.PressureAngle = math.Pi / 2 }, "pressure angle"},
		"tiny sun":           {func(p *planetarygearbox.Params) { p.SunTeeth = 2 }, "sun teeth"},
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
