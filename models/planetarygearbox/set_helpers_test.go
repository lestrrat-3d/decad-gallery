package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// axialSpan is the interval a body occupies along axis once moved by offset.
type axialSpan struct{ lo, hi float64 }

func spanOf(t *testing.T, body *decad.Body, offset, axis r3.Vec) axialSpan {
	t.Helper()
	box := bounds(t, body)
	s := axialSpan{lo: math.Inf(1), hi: math.Inf(-1)}
	for _, x := range []float64{box.Min.X, box.Max.X} {
		for _, y := range []float64{box.Min.Y, box.Max.Y} {
			for _, z := range []float64{box.Min.Z, box.Max.Z} {
				v := r3.Vec{X: x, Y: y, Z: z}.Add(offset).Dot(axis)
				s.lo, s.hi = math.Min(s.lo, v), math.Max(s.hi, v)
			}
		}
	}
	return s
}

// above reports whether a lies wholly beyond b along the axis.
func (a axialSpan) above(b axialSpan) bool { return a.lo > b.hi }

// explodeGap is the gap Gearbox.Explode leaves between a part and the parts it clears.
const explodeGap = 6.0

// sharingPairs lists the part pairs that may still share space in the exploded view, each name pair in both orders:
// a planet in mesh with its sun, and a nut in its closed cavity in the ring or in a mounting ear.
func sharingPairs(box planetarygearbox.Gearbox) map[[2]string]struct{} {
	pairs := map[[2]string]struct{}{}
	add := func(a, b string) {
		pairs[[2]string{a, b}] = struct{}{}
		pairs[[2]string{b, a}] = struct{}{}
	}
	for k, s := range box.Stages {
		for i := range s.Planets {
			add(planetarygearbox.SunName(k), planetarygearbox.PlanetName(k, i))
		}
	}
	half := len(box.Nuts) / 2
	for i := range half {
		add(planetarygearbox.RingName, planetarygearbox.FrontNutName(i))
		add(planetarygearbox.RingName, planetarygearbox.RearNutName(i))
	}
	for i := range box.EarNuts {
		add(planetarygearbox.RingName, planetarygearbox.EarNutName(i))
	}
	return pairs
}

// collarSpan returns the axial span, in the assembled pose, of the part of body that reaches more than radius from
// the joint axis: the collar of a shaft, or the thick part of a stepped shaft or a carrier pin. It reads the vertices
// of a 0.05 mm tessellation, and fails when no vertex reaches that far. Called with a radius just under the collar's,
// it spans the collar's cylinder and stops where its cone begins, because a cone's mesh has vertices only on its two
// end circles (and on a flat's cut across it, which stays under the radius above the cone's start).
func collarSpan(t *testing.T, body *decad.Body, axis parts.Axis, radius float64) axialSpan {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(0.05), decad.WithVerification(decad.VerifyNone))
	require.NoError(t, err)
	dir, ok := axis.Dir.Normalize()
	require.True(t, ok)
	s := axialSpan{lo: math.Inf(1), hi: math.Inf(-1)}
	for _, v := range mesh.Vertices() {
		rel := v.Sub(axis.Point)
		along := rel.Dot(dir)
		if rel.Sub(dir.Scale(along)).Len() <= radius {
			continue
		}
		s.lo, s.hi = math.Min(s.lo, v.Dot(dir)), math.Max(s.hi, v.Dot(dir))
	}
	require.False(t, math.IsInf(s.lo, 1), "the part reaches past %g mm from its axis", radius)
	return s
}

// shift moves the span by d along the axis.
func (a axialSpan) shift(d float64) axialSpan { return axialSpan{lo: a.lo + d, hi: a.hi + d} }

func requireExplodedGearbox(t *testing.T, doc *decad.Document, box planetarygearbox.Gearbox,
	p planetarygearbox.Params) {
	t.Helper()
	set, err := box.Set(doc, p)
	require.NoError(t, err)
	layout, err := box.Explode(t.Context(), set)
	require.NoError(t, err)
	require.Equal(t, planetarygearbox.FrontCoverName, layout.Ground)
	require.True(t, layout.Derived, "Decad derives the removal order")

	axis := r3.Vec{Z: 1}
	if p.Placement != nil {
		frame, err := r3.FromFrame(*p.Placement)
		require.NoError(t, err)
		axis = frame.ApplyDir(axis)
	}

	// Every part has a body, a joint path to the front cover and a place in the derived removal order, and every
	// part but the front cover moves along the axis.
	named := box.Named()
	require.Len(t, layout.Parts, len(named))
	bodies := map[string]*decad.Body{}
	for _, nb := range named {
		bodies[nb.Name] = nb.Body
	}
	offsets, gaps := map[string]r3.Vec{}, map[string]float64{}
	for _, part := range layout.Parts {
		offsets[part.Name], gaps[part.Name] = part.Offset, part.Gap.Base()
		require.GreaterOrEqual(t, part.Depth, 0, "%s is joined to the front cover", part.Name)
		if part.Name == planetarygearbox.FrontCoverName {
			continue
		}
		require.Positive(t, part.Order, "%s has a place in the removal order", part.Name)
		require.NotEmpty(t, part.Stage, "%s moves in a removal round", part.Name)
		require.InDelta(t, part.Offset.Len(), math.Abs(part.Offset.Dot(axis)), 1e-9, "%s moves along the axis",
			part.Name)
	}
	require.Zero(t, offsets[planetarygearbox.FrontCoverName].Len(), "the front cover stays put")
	span := func(name string) axialSpan {
		t.Helper()
		body, ok := bodies[name]
		require.True(t, ok, name)
		return spanOf(t, body, offsets[name], axis)
	}
	for _, part := range layout.Parts {
		s := span(part.Name)
		t.Logf("%-12s %-20s span [%.1f, %.1f]", part.Stage, part.Name, s.lo, s.hi)
	}

	// Only a planet and its sun, or a nut and the ring, may still share space, and only when they move by the same
	// offset, which leaves them as assembled.
	sharing := sharingPairs(box)
	for _, pair := range layout.Overlaps {
		_, ok := sharing[pair]
		require.True(t, ok, "%s and %s share space", pair[0], pair[1])
		require.InDelta(t, 0, offsets[pair[0]].Sub(offsets[pair[1]]).Len(), 1e-9, "%s and %s share space",
			pair[0], pair[1])
	}

	// Identical parts share one offset: the planets, planet bearings and carrier pins of a stage, the bolts and the
	// nuts of each cover. Each planet is turned to its own mesh phase, so the planets' boxes, and the distances Decad
	// moves them to clear the parts below, differ by a few thousandths of a millimetre.
	requireSameOffset := func(what string, n int, name func(int) string) {
		t.Helper()
		for i := 1; i < n; i++ {
			require.InDelta(t, 0, offsets[name(i)].Sub(offsets[name(0)]).Len(), 0.01, "%s %s and %s", what, name(i),
				name(0))
		}
	}
	half := len(box.Bolts) / 2
	requireSameOffset("front bolts", half, planetarygearbox.FrontBoltName)
	requireSameOffset("rear bolts", half, planetarygearbox.RearBoltName)
	requireSameOffset("front nuts", half, planetarygearbox.FrontNutName)
	requireSameOffset("rear nuts", half, planetarygearbox.RearNutName)
	requireSameOffset("ear bolts", len(box.EarBolts), planetarygearbox.EarBoltName)
	requireSameOffset("ear nuts", len(box.EarNuts), planetarygearbox.EarNutName)
	for k, s := range box.Stages {
		n := len(s.Planets)
		requireSameOffset("planets", n, func(i int) string { return planetarygearbox.PlanetName(k, i) })
		requireSameOffset("planet bearings", n, func(i int) string { return planetarygearbox.BearingName(k, i) })
		requireSameOffset("carrier pins", n, func(i int) string { return planetarygearbox.PinName(k, i) })
	}

	// Each end reads, from the outside in, bearing, cover, shaft: each shaft sits on the inner side of its cover and
	// each cover bearing outside it, each one of its own gaps from the cover, so the bearing is beyond the shaft's free
	// end. The input shaft's free end is at the front and the output shaft's at the rear. The rear cover sits beyond the
	// ring. Decad lays the parts out from its own boxes, so a gap is read to within 0.1 mm.
	requireNextTo := func(cover, part string) {
		t.Helper()
		a, b := span(cover), span(part)
		require.LessOrEqual(t, math.Max(a.lo-b.hi, b.lo-a.hi), gaps[part]+0.1, "%s sits next to %s", part, cover)
	}
	input, front, frontBearing := span(planetarygearbox.InputShaftName), span(planetarygearbox.FrontCoverName),
		span(planetarygearbox.FrontBearingName)
	require.True(t, input.above(front), "the input shaft is on the inner side of the front cover")
	require.True(t, front.above(frontBearing), "the front bearing is outside the front cover")
	require.True(t, input.above(frontBearing), "the front bearing is beyond the input shaft's free end")
	requireNextTo(planetarygearbox.FrontCoverName, planetarygearbox.InputShaftName)
	requireNextTo(planetarygearbox.FrontCoverName, planetarygearbox.FrontBearingName)
	output, ring, rear, rearBearing := span(planetarygearbox.OutputShaftName), span(planetarygearbox.RingName),
		span(planetarygearbox.RearCoverName), span(planetarygearbox.RearBearingName)
	require.True(t, rear.above(output), "the output shaft is on the inner side of the rear cover")
	require.True(t, rearBearing.above(rear), "the rear bearing is outside the rear cover")
	require.True(t, rearBearing.above(output), "the rear bearing is beyond the output shaft's free end")
	requireNextTo(planetarygearbox.RearCoverName, planetarygearbox.OutputShaftName)
	requireNextTo(planetarygearbox.RearCoverName, planetarygearbox.RearBearingName)
	require.True(t, rear.above(ring), "the rear cover is outside the ring")
	// Each collar ring comes off its shaft's free end, outside its cover's bearing.
	require.True(t, frontBearing.above(span(planetarygearbox.FrontCollarName)), "the front collar is outside the front bearing")
	require.True(t, span(planetarygearbox.RearCollarName).above(rearBearing), "the rear collar is outside the rear bearing")

	// No collar passes a bore narrower than itself. A shaft's collar or thick part, and a carrier pin's thick part,
	// are the bearing bore plus 2.4 mm across (7.4 mm for a 605ZZ). A bearing bore and a sun's D bore are narrower,
	// so each bearing or sun that a shaft or pin passes stays on the side of its collar where it was assembled. A
	// cover's lip bore and the carrier socket that holds the thick part are wide enough for it. A pin turns on the
	// planet bearing's bore and any other shaft on the shaft bearing's.
	planetBore, ok := ballbearing.Lookup(p.PlanetBearing)
	require.True(t, ok, "planet bearing %q is in the catalog", p.PlanetBearing)
	shaftBore, ok := ballbearing.Lookup(p.ShaftBearing)
	require.True(t, ok, "shaft bearing %q is in the catalog", p.ShaftBearing)
	pins := map[string]struct{}{}
	narrow := map[string]struct{}{planetarygearbox.FrontBearingName: {}, planetarygearbox.RearBearingName: {}}
	shafts := map[string]struct{}{planetarygearbox.InputShaftName: {}}
	for k, s := range box.Stages {
		narrow[planetarygearbox.SunName(k)] = struct{}{}
		shafts[planetarygearbox.ShaftName(k, len(box.Stages))] = struct{}{}
		for i := range s.Planets {
			narrow[planetarygearbox.BearingName(k, i)] = struct{}{}
			shafts[planetarygearbox.PinName(k, i)] = struct{}{}
			pins[planetarygearbox.PinName(k, i)] = struct{}{}
		}
	}
	const touching = 1e-6
	checked := 0
	for _, j := range set.Joints() {
		_, isShaft := shafts[j.A]
		_, isNarrow := narrow[j.B]
		if j.Kind != parts.PinInBore || !isShaft || !isNarrow {
			continue
		}
		// A thick part is a collar margin (1.2 mm) wider than the rod; read it just under that. The input shaft's collar
		// is a cone on the sun's side, down to a sun section that a 608ZZ shaft narrows to 6 mm, so its cone has no
		// vertex between its two end circles and it is read from 0.2 mm over the bore, under the 8.8 mm collar of a 608ZZ shaft.
		radius := shaftBore.BoreDiameter/2 + 1.15
		if _, isPin := pins[j.A]; isPin {
			radius = planetBore.BoreDiameter/2 + 1.15
		}
		if j.A == planetarygearbox.InputShaftName {
			radius = shaftBore.BoreDiameter/2 + 0.2
		}
		collar := collarSpan(t, bodies[j.A], *j.Axis, radius)
		bore := spanOf(t, bodies[j.B], r3.Vec{}, axis)
		beyond := bore.lo >= collar.hi-touching
		require.True(t, beyond || bore.hi <= collar.lo+touching, "%s lies to one side of the collar of %s", j.B, j.A)
		collar, bore = collar.shift(offsets[j.A].Dot(axis)), span(j.B)
		if beyond {
			require.Greater(t, bore.lo, collar.hi-touching, "the collar of %s stays below %s", j.A, j.B)
		} else {
			require.Less(t, bore.hi, collar.lo+touching, "the collar of %s stays above %s", j.A, j.B)
		}
		checked++
	}
	// The input shaft passes the front bearing and the first sun, each later shaft the next sun or the rear bearing,
	// and each pin its planet's bearing.
	require.Equal(t, 2+len(box.Stages)+len(box.Stages)*p.PlanetCount, checked)

	// Each bolt comes out of its cover, on the cover's outer side.
	for i := range half {
		require.True(t, front.above(span(planetarygearbox.FrontBoltName(i))),
			"front bolt %d is outside the front cover", i+1)
		require.True(t, span(planetarygearbox.RearBoltName(i)).above(rear), "rear bolt %d is outside the rear cover", i+1)
	}
	// Each ear bolt comes out of its ear's rear face, beyond the ring.
	for i := range box.EarBolts {
		require.True(t, span(planetarygearbox.EarBoltName(i)).above(ring), "ear bolt %d is beyond the ring", i+1)
	}

	// In each stage a planet bearing is off the free end of its pin, the pin sits next to its carrier, one gap
	// below or above it, and the shaft pressed into the carrier is out of it. A pin's thick part ends level with the
	// carrier's upper face and its socket runs through the plate, so the pin can leave it either way. Decad lays the
	// parts out from its own boxes, which differ from Body.Bounds by up to about 0.01 mm, so the gap is read to
	// within 0.1 mm.
	for k, s := range box.Stages {
		carrier := span(planetarygearbox.CarrierName(k))
		shaft := span(planetarygearbox.ShaftName(k, len(box.Stages)))
		for i := range s.Planets {
			bearing := span(planetarygearbox.BearingName(k, i))
			pin := span(planetarygearbox.PinName(k, i))
			require.True(t, pin.above(bearing), "bearing %d.%d is off the pin's free end", k+1, i+1)
			require.True(t, carrier.above(pin) || pin.above(carrier), "pin %d.%d is out of its carrier", k+1, i+1)
			require.LessOrEqual(t, math.Max(carrier.lo-pin.hi, pin.lo-carrier.hi), explodeGap+0.1,
				"pin %d.%d sits next to its carrier", k+1, i+1)
		}
		require.True(t, shaft.above(carrier), "shaft %d is out of its carrier", k+1)
	}
}
