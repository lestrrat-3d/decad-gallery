package planetarygearbox_test

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/decad/linkage"
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
// a planet in mesh with its sun, and a nut in its cavity in the ring.
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
	return pairs
}

// collarSpan returns the axial span, in the assembled pose, of the part of body that reaches more than radius from
// the joint axis: the collar of a shaft, or the thick part of a stepped shaft or a carrier pin. It reads the vertices
// of a 0.05 mm tessellation, and fails when no vertex reaches that far.
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

	// No collar passes a bore narrower than itself. A shaft's collar or thick part, and a carrier pin's thick part,
	// are 7.4 mm across. The 605ZZ bore is 5 mm and a sun's D bore 5.1 mm, so each bearing or sun that a shaft or
	// pin passes stays on the side of its collar where it was assembled. A cover's 10 mm lip bore and the carrier
	// socket that holds the thick part are wide enough for it.
	radius := ballbearing.Params605ZZ().BoreDiameter/2 + 0.5
	narrow := map[string]struct{}{planetarygearbox.FrontBearingName: {}, planetarygearbox.RearBearingName: {}}
	shafts := map[string]struct{}{planetarygearbox.InputShaftName: {}}
	for k, s := range box.Stages {
		narrow[planetarygearbox.SunName(k)] = struct{}{}
		shafts[planetarygearbox.ShaftName(k, len(box.Stages))] = struct{}{}
		for i := range s.Planets {
			narrow[planetarygearbox.BearingName(k, i)] = struct{}{}
			shafts[planetarygearbox.PinName(k, i)] = struct{}{}
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

func TestExplodedGearbox(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	require.Len(t, box.Stages, 1)
	requireExplodedGearbox(t, doc, box, p)

	// With the ring held, the input shaft and its sun turn in the front bearing, the carrier and the output shaft turn
	// in the rear bearing, and each planet turns on its carrier pin.
	t.Run("linkage", func(t *testing.T) {
		set, err := box.Set(doc, p)
		require.NoError(t, err)
		l, unplaced, err := set.Linkage(planetarygearbox.RingName, nil)
		require.NoError(t, err)
		names := map[*decad.Body]string{}
		for _, nb := range box.Named() {
			names[nb.Body] = nb.Name
		}
		var links []string
		for _, link := range l.Links() {
			_, ok := link.Joint().(linkage.RevoluteJoint)
			require.True(t, ok, "every link turns")
			var members []string
			for _, b := range link.Bodies() {
				members = append(members, names[b])
			}
			slices.Sort(members)
			links = append(links, strings.Join(members, ", "))
		}
		want := []string{
			planetarygearbox.InputShaftName + ", " + planetarygearbox.SunName(0),
			strings.Join([]string{planetarygearbox.CarrierName(0), planetarygearbox.PinName(0, 0),
				planetarygearbox.PinName(0, 1), planetarygearbox.PinName(0, 2), planetarygearbox.OutputShaftName}, ", "),
		}
		for i := range p.PlanetCount {
			want = append(want, planetarygearbox.PlanetName(0, i)+", "+planetarygearbox.BearingName(0, i))
		}
		for i := range want {
			members := strings.Split(want[i], ", ")
			slices.Sort(members)
			want[i] = strings.Join(members, ", ")
		}
		require.ElementsMatch(t, want, links)

		// The second joint between two links that already turn about the same axis closes a loop.
		var loops []string
		for _, j := range unplaced {
			loops = append(loops, j.Name)
		}
		wantLoops := []string{"input shaft in the front cover", "output shaft in the rear cover"}
		for i := range p.PlanetCount {
			wantLoops = append(wantLoops, fmt.Sprintf("pin 1.%d through its planet's lip", i+1))
		}
		require.ElementsMatch(t, wantLoops, loops)
	})
}

func TestExplodedTwoStageGearbox(t *testing.T) {
	p := twoStageParams()
	doc, box := twoStageGearbox(t)
	requireExplodedGearbox(t, doc, box, p)

	set, err := box.Set(doc, p)
	require.NoError(t, err)
	t.Run("joints", func(t *testing.T) {
		joints := map[string]parts.Joint{}
		for _, j := range set.Joints() {
			joints[j.Name] = j
		}
		for _, want := range []string{
			"input shaft in the front bearing", "input shaft in the front cover", "input shaft in the first sun",
			"Link Shaft 1 in Carrier 1", "Link Shaft 1 in Sun Gear 2", "Output Shaft in Carrier 2",
			"output shaft in the rear bearing", "output shaft in the rear cover", "pin 2.3 in its bearing",
			"pin 2.3 in the carrier", "pin 2.3 through its planet's lip", "bearing 2.3 in its planet",
			"planet 2.3 meshes the ring", "Rear Bolt 4 in the cover", "Rear Bolt 4 in the ring",
		} {
			require.Contains(t, joints, want)
		}
		// A shaft joins its bearing as a pin in a bore: the bearing's bore is the narrowest hole it passes.
		for _, j := range joints {
			if strings.Contains(j.Name, "in the front bearing") || strings.Contains(j.Name, "in the rear bearing") {
				require.Equal(t, parts.PinInBore, j.Kind, j.Name)
			}
		}
		// The shafts turn in their covers and bearings, and each planet turns on its pin; the rest is held fixed.
		turns := map[string]struct{}{
			"input shaft in the front bearing": {}, "input shaft in the front cover": {},
			"output shaft in the rear bearing": {}, "output shaft in the rear cover": {},
		}
		for k := range p.Stages {
			for i := range p.PlanetCount {
				turns[fmt.Sprintf("pin %d.%d in its bearing", k+1, i+1)] = struct{}{}
				turns[fmt.Sprintf("pin %d.%d through its planet's lip", k+1, i+1)] = struct{}{}
			}
		}
		for name, j := range joints {
			want := parts.Fixed
			if _, ok := turns[name]; ok {
				want = parts.Turns
			}
			require.Equal(t, want, j.Motion, name)
		}
		// A shaft or pin in a bearing bore has no clearance, and a bearing sits in its seat with the seat clearance.
		requireClearance := func(name string, want float64) {
			t.Helper()
			c := joints[name].Clearance
			require.NotEqual(t, units.Value{}, c, "%s declares a clearance", name)
			require.InDelta(t, want, c.Base(), 1e-12, name)
		}
		requireClearance("input shaft in the front bearing", 0)
		requireClearance("pin 2.3 in its bearing", 0)
		requireClearance("bearing 2.3 in its planet", p.BearingClearance)
		requireClearance("front bearing in the front cover", p.BearingClearance)
	})
	t.Run("bill of materials", func(t *testing.T) {
		bom, err := set.BOM(t.Context())
		require.NoError(t, err)
		count := map[string]int{}
		for _, row := range bom.Rows {
			if row.Kind == parts.Purchased {
				count[row.Spec] += row.Quantity
			}
		}
		require.Equal(t, 2+p.Stages*p.PlanetCount, count["605ZZ ball bearing"],
			"two cover bearings and one for each planet")
		require.Equal(t, 8, count["M2 hex nut"])
		require.Equal(t, 8, count["M2 x 6 bolt"])
	})
}

func TestExplodedPlacedGearbox(t *testing.T) {
	p := placedParams(t)
	doc, box := placedGearbox(t)
	requireExplodedGearbox(t, doc, box, p)
}
