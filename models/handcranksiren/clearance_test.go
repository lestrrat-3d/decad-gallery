package handcranksiren_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/handcranksiren"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// The class minima of the model-pipeline rules and the fastener bound. The tests do not read the model's own nominal
// values: they prove the placed parts meet these.
const (
	tight       = 0.2
	regular     = 0.35
	loose       = 0.5
	fastenerGap = 0.05
	// seatMinimum is the proven lower bound on a bearing seat's radial clearance. The seat is designed at 0.05, and the
	// section measurement's uncertainty takes the rest (docs/model-pipeline.md "Part clearances").
	seatMinimum = 0.02
)

// fitTolerance is the tolerance of the fit sections.
const fitTolerance = 0.02

// roundFitTolerance is the section tolerance of the fits between round parts. At fitTolerance the intermediate shaft's
// bound against its bearing is 0.192, below the Tight minimum.
const roundFitTolerance = 0.01

// housingTolerance is the finest section tolerance the faceted plates accept (their mesh bound is 0.01615).
const housingTolerance = 0.0162

// frameAt returns the plane through origin spanned by u and v.
func frameAt(t *testing.T, origin, u, v r3.Vec) r3.Frame {
	t.Helper()
	f, err := r3.NewFrame(origin, u, v)
	require.NoError(t, err)
	return f
}

// atY is the plane across Y at y.
func atY(t *testing.T, y float64) r3.Frame {
	return frameAt(t, r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: 1})
}

// sectionGap returns the proven lower bound on the gap between a and b in the plane, including both sections' mesh
// bounds. The sections must not overlap.
func sectionGap(t *testing.T, a, b *decad.Body, plane r3.Frame) float64 {
	t.Helper()
	return sectionGapAt(t, a, b, plane, fitTolerance)
}

// sectionGapAt is sectionGap with the section tolerance named: shapes of round parts hold their mesh to 0.01.
func sectionGapAt(t *testing.T, a, b *decad.Body, plane r3.Frame, tolerance float64) float64 {
	t.Helper()
	return sectionGapMixed(t, a, tolerance, b, tolerance, plane)
}

// sectionGapMixed takes each body's section at its own tolerance, so a faceted housing piece (which cannot be sectioned
// below its mesh bound of 0.01615) does not widen the round part's bound.
func sectionGapMixed(t *testing.T, a *decad.Body, ta float64, b *decad.Body, tb float64, plane r3.Frame) float64 {
	t.Helper()
	first, err := a.SectionOutline(t.Context(), plane, units.Millimeters(ta))
	require.NoError(t, err)
	second, err := b.SectionOutline(t.Context(), plane, units.Millimeters(tb))
	require.NoError(t, err)
	gap, err := first.Clearance(second)
	require.NoError(t, err)
	require.False(t, gap.Overlaps)
	return gap.Lower().Base()
}

// axialGap returns the proven lower bound on the distance from the top of lo to the bottom of hi along axis 0 (X), 1
// (Y) or 2 (Z).
func axialGap(t *testing.T, lo, hi *decad.Body, axis int) float64 {
	t.Helper()
	a, err := lo.Bounds()
	require.NoError(t, err)
	b, err := hi.Bounds()
	require.NoError(t, err)
	component := func(v r3.Vec) float64 { return [3]float64{v.X, v.Y, v.Z}[axis] }
	return component(b.Min) - component(a.Max) - a.Bound.Base() - b.Bound.Base()
}

// fitCheck collects every fit that falls short of its class, so one run names all of them.
type fitCheck struct {
	t        *testing.T
	failures []string
}

// atLeast records one fit: got is the proven lower bound and want the class minimum.
func (c *fitCheck) atLeast(id, what string, got, want float64) {
	c.t.Helper()
	c.t.Logf("%s %s: lower bound %.4f, class minimum %.2f", id, what, got, want)
	if got < want {
		c.failures = append(c.failures, fmt.Sprintf("%s %s: lower bound %.4f is below %.2f", id, what, got, want))
	}
}

func (c *fitCheck) done() {
	c.t.Helper()
	require.Empty(c.t, c.failures)
}

// axial returns the plane through the axis point (posed by motion) spanned by Y and the posed X direction, which meets
// the D-shaped parts on the round side only: the flat faces +Z.
func axialPlane(t *testing.T, p handcranksiren.Params, motion string, axis r3.Vec) r3.Frame {
	t.Helper()
	origin, err := handcranksiren.PosePoint(p, motion, axis)
	require.NoError(t, err)
	tip, err := handcranksiren.PosePoint(p, motion, axis.Add(r3.Vec{X: 1}))
	require.NoError(t, err)
	return frameAt(t, origin, r3.Vec{Y: 1}, tip.Sub(origin))
}

// requireClearances measures every distinct fit of the default siren on the placed parts, and requires the proven lower
// bound to meet the class minimum. instructions.md lists the rows.
func requireClearances(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part) {
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	c := &fitCheck{t: t}
	body := func(name string) *decad.Body { return parts[name].Body }
	section := func(id, a, b string, plane r3.Frame, class float64) {
		t.Helper()
		c.atLeast(id, a+" / "+b, sectionGap(t, body(a), body(b), plane), class)
	}
	exact := func(id, a, b string, plane r3.Frame, class float64) {
		t.Helper()
		c.atLeast(id, a+" / "+b, sectionGapAt(t, body(a), body(b), plane, roundFitTolerance), class)
	}
	axial := func(id, lo, hi string, axis int, class float64) {
		t.Helper()
		c.atLeast(id, lo+" / "+hi+" axial", axialGap(t, body(lo), body(hi), axis), class)
	}

	// T4: each shaft in each of its bearings, at the bearing's mid-plane.
	shaftOf := map[string]string{"rotor": "rotor-shaft", "intermediate": "intermediate-shaft", "crank": "crank-shaft"}
	for _, site := range handcranksiren.BearingSites() {
		kind := site.Name[len("bearing-"):]
		for k := range shaftOf {
			if len(kind) > len(k) && kind[:len(k)] == k {
				exact("T4", shaftOf[k], site.Name, atY(t, site.Start+3.5), tight)
			}
		}
	}
	// T5: each gear's D-bore on its shaft, at the gear's mid-plane.
	for _, g := range [...]struct {
		gear, shaft string
		midY        float64
	}{
		{"crank-gear", "crank-shaft", 23.6}, {"intermediate-pinion", "intermediate-shaft", 23.6},
		{"intermediate-gear", "intermediate-shaft", 15.6}, {"rotor-pinion", "rotor-shaft", 15.6},
	} {
		exact("T5", g.gear, g.shaft, atY(t, g.midY), tight)
	}
	// T7: the rotor-shaft collar against the front bearing, across the axial plane through R.
	exact("T7", "rotor-shaft", "bearing-rotor-front", axialPlane(t, p, "rotor", state.Rotor), tight)
	// T9, T10: the rotor pinion against the rotor-shaft collar, the crank gear against the crank-shaft collar.
	exact("T9", "rotor-pinion", "rotor-shaft", axialPlane(t, p, "rotor", state.Rotor), tight)
	exact("T10", "crank-gear", "crank-shaft", axialPlane(t, p, "crank", state.Crank), tight)
	// L3: the gears of the two planes against each other, and the shafts against the gears they do not carry.
	axial("L3", "intermediate-gear", "intermediate-pinion", 1, loose)
	axial("L3", "rotor-pinion", "intermediate-pinion", 1, loose)
	axial("L3", "intermediate-gear", "crank-gear", 1, loose)
	axial("L3", "rotor-pinion", "crank-gear", 1, loose)
	section("L3", "crank-shaft", "intermediate-gear", atY(t, 15.6), loose)
	section("L3", "rotor-shaft", "crank-gear", atY(t, 23.6), loose)

	// T1: the plates and the frame across their seams.
	axial("T1", "gearbox-front", "gearbox-frame", 1, tight)
	axial("T1", "gearbox-frame", "gearbox-rear", 1, tight)
	// S1, S2: each bearing in its seat, at the bearing's mid-plane, and above its seat floor.
	for _, site := range handcranksiren.BearingSites() {
		c.atLeast("S1", site.Name+" / "+site.Holder, sectionGapMixed(t, body(site.Name), roundFitTolerance,
			body(site.Holder), housingTolerance, atY(t, site.Start+3.5)), seatMinimum)
		requireSeatFloor(t, c, site, body(site.Holder))
	}
	// T6, R1: the shaft ends against the blind relief floors, and the shafts in the plates' through holes.
	for _, row := range [...]struct{ id, shaft, plate string }{
		{"T6", "intermediate-shaft", "gearbox-front"}, {"T6", "crank-shaft", "gearbox-front"},
		{"T6", "intermediate-shaft", "gearbox-rear"}, {"T6", "rotor-shaft", "gearbox-rear"},
	} {
		motion := row.shaft[:len(row.shaft)-len("-shaft")]
		section(row.id, row.shaft, row.plate, axialPlane(t, p, motion, axisOf(state, motion)), tight)
	}
	section("R1", "rotor-shaft", "gearbox-front", axialPlane(t, p, "rotor", state.Rotor), regular)
	section("R1", "crank-shaft", "gearbox-rear", axialPlane(t, p, "crank", state.Crank), regular)
	// L2: the gears' faces against the plates.
	for _, gear := range []string{"intermediate-gear", "rotor-pinion"} {
		axial("L2", "gearbox-front", gear, 1, loose)
	}
	for _, gear := range []string{"crank-gear", "intermediate-pinion"} {
		axial("L2", gear, "gearbox-rear", 1, loose)
	}
	// L4: each gear against the nearest bearing face.
	for _, bearing := range []string{"bearing-rotor-front", "bearing-intermediate-front"} {
		for _, gear := range []string{"intermediate-gear", "rotor-pinion"} {
			if (bearing == "bearing-rotor-front") == (gear == "rotor-pinion") {
				axial("L4", bearing, gear, 1, loose)
			}
		}
	}
	axial("L4", "crank-gear", "bearing-crank-rear", 1, loose)
	axial("L4", "intermediate-pinion", "bearing-intermediate-rear", 1, loose)

	// T2, T3: the drum's posts against the front plate, the stator against the drum, the collector against the stator.
	axial("T2", "drum", "gearbox-front", 1, tight)
	axial("T3", "stator", "drum", 1, tight)
	axial("T3", "collector", "stator", 1, tight)
	// T11: the horn flange against the collector's -X wall, along X.
	axial("T11", "horn", "collector", 0, tight)
	// L6: the collector's end wall against the drum, and the horn against the stator and the drum in the plane Z 84.
	axial("L6", "collector", "drum", 1, loose)
	zPlane := frameAt(t, r3.Vec{Z: 84}, r3.Vec{X: 1}, r3.Vec{Y: 1})
	section("L6", "horn", "stator", zPlane, loose)
	section("L6", "horn", "drum", zPlane, loose)
	// T12: the grip's pad against gearbox-front and the head of the front bolt it covers, along Y.
	axial("T12", "grip", "gearbox-front", 1, tight)
	axial("T12", "grip", "gearbox-front-bolt-2", 1, tight)
	// L7: the collector, the stator and the drum against the grip, along X.
	for _, part := range []string{"collector", "stator", "drum"} {
		axial("L7", part, "grip", 0, loose)
	}
	// T5, T6: the rotor hub's D-bore on the rotor shaft, and the shaft's front end against the bore's floor.
	exact("T5", "rotor", "rotor-shaft", atY(t, -20.5), tight)
	exact("T6", "rotor", "rotor-shaft", axialPlane(t, p, "rotor", state.Rotor), tight)
	// R1: the rotor shaft through the drum's hole.
	section("R1", "rotor-shaft", "drum", axialPlane(t, p, "rotor", state.Rotor), regular)
	requireRotorFits(t, p, parts, c)
	// T5, T6, T8: the arm's D-bore on the crank shaft, the shaft's end against the bore's floor, the pin's stub in the
	// arm's hexagonal pocket.
	exact("T5", "crank-arm", "crank-shaft", atY(t, 41.0), tight)
	exact("T6", "crank-arm", "crank-shaft", axialPlane(t, p, "crank", state.Crank), tight)
	exact("T8", "crank-pin", "crank-arm", atY(t, 44.0), tight)
	// R3: the handle on the pin, in the shank's and the head's sections and along the pin.
	pinPlane := axialPlane(t, p, "crank", handcranksiren.PinAxis)
	exact("R3", "crank-handle", "crank-pin", atY(t, 70), regular)
	exact("R3", "crank-handle", "crank-pin", atY(t, 95), regular)
	exact("R3", "crank-handle", "crank-pin", pinPlane, regular)
	// R4: the arm against the handle's inner end.
	axial("R4", "crank-arm", "crank-handle", 1, regular)
	// L5: the arm against the rear plate.
	axial("L5", "gearbox-rear", "crank-arm", 1, loose)
	c.done()
}

// axisOf returns the axis point of the shaft with the given motion name.
func axisOf(s handcranksiren.State, motion string) r3.Vec {
	switch motion {
	case "rotor":
		return s.Rotor
	case "intermediate":
		return s.Intermediate
	}
	return s.Crank
}

// requireSeatFloor proves S2: the holder has no material inside the seat's pocket, at the bearing's floor-side face
// moved 0.2 toward the floor, so the floor is at least 0.2 from the bearing. It samples radius 10.25 on 12 angles.
func requireSeatFloor(t *testing.T, c *fitCheck, site handcranksiren.BearingSite, holder *decad.Body) {
	t.Helper()
	// The front plates' seat floors lie below the bearing's start face and the rear plates' above its end face.
	y := site.Start - 0.2
	if site.Holder == "gearbox-rear" {
		y = site.Start + 7 + 0.2
	}
	plane := frameAt(t, site.Axis.Add(r3.Vec{Y: y}), r3.Vec{X: 1}, r3.Vec{Z: 1})
	outline, err := holder.SectionOutline(t.Context(), plane, units.Millimeters(fitTolerance))
	require.NoError(t, err)
	section := sectionAt{outline: outline, origin: plane.Origin(), u: r3.Vec{X: 1}, v: r3.Vec{Z: 1}}
	solid := 0
	for k := range 12 {
		angle := float64(k) * math.Pi / 6
		if section.contains(r3.Vec{X: 10.25 * math.Cos(angle), Z: 10.25 * math.Sin(angle)}) {
			solid++
		}
	}
	got := 0.0
	if solid == 0 {
		got = 0.2
	}
	c.atLeast("S2", site.Name+" / seat floor", got, 0.2)
}

// requireRotorFits proves R2, the rotor's gap to the stator with the clamp budget of the stack, and L1, the rotor's gap
// to the drum. Both depend on Slots, so the variants run them too.
func requireRotorFits(t *testing.T, p handcranksiren.Params, parts map[string]handcranksiren.Part, c *fitCheck) {
	t.Helper()
	state, err := handcranksiren.Motion(p)
	require.NoError(t, err)
	body := func(name string) *decad.Body { return parts[name].Body }
	section := func(id, a, b string, plane r3.Frame, class float64) {
		t.Helper()
		c.atLeast(id, a+" / "+b, sectionGap(t, body(a), body(b), plane), class)
	}
	// R2: the stator against the rotor, with the clamp budget of the stack (instructions.md, Shafts and bearings).
	c.atLeast("R2", "stator / rotor axial", axialGap(t, body("stator"), body("rotor"), 1), 1.10)
	// L1: the rotor's rim against the drum wall, and its vanes and hub against the back plate.
	section("L1", "rotor", "drum", atY(t, -26.0), loose)
	section("L1", "rotor", "drum", axialPlane(t, p, "rotor", state.Rotor), loose)
	// The same plane turned to a vane's mid-line, so the vanes are measured too.
	vane := math.Pi / float64(p.Slots)
	vaneOrigin, err := handcranksiren.PosePoint(p, "rotor", state.Rotor)
	require.NoError(t, err)
	vaneTip, err := handcranksiren.PosePoint(p, "rotor", state.Rotor.Add(r3.Vec{X: math.Cos(vane), Z: math.Sin(vane)}))
	require.NoError(t, err)
	section("L1", "rotor", "drum", frameAt(t, vaneOrigin, r3.Vec{Y: 1}, vaneTip.Sub(vaneOrigin)), loose)
}
