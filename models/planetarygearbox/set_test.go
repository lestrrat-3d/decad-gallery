package planetarygearbox_test

import (
	"math"
	"strings"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/decad/parts"
	"github.com/lestrrat-3d/r3"
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

func requireExplodedGearbox(t *testing.T, doc *decad.Document, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	set, err := box.Set(doc, p)
	require.NoError(t, err)
	layout, err := box.Explode(t.Context(), set, p)
	require.NoError(t, err)
	require.Equal(t, planetarygearbox.InputShaftName, layout.Ground)

	axis := r3.Vec{Z: 1}
	if p.Placement != nil {
		frame, err := r3.FromFrame(*p.Placement)
		require.NoError(t, err)
		axis = frame.ApplyDir(axis)
	}

	// Every part has a body and a joint path to the input shaft.
	named := box.Named()
	require.Len(t, layout.Parts, len(named))
	bodies := map[string]*decad.Body{}
	for _, nb := range named {
		bodies[nb.Name] = nb.Body
	}
	offsets := map[string]r3.Vec{}
	for _, part := range layout.Parts {
		offsets[part.Name] = part.Offset
		require.GreaterOrEqual(t, part.Depth, 0, "%s is joined to the input shaft", part.Name)
		if part.Name != planetarygearbox.InputShaftName {
			require.InDelta(t, part.Offset.Len(), math.Abs(part.Offset.Dot(axis)), 1e-9, "%s moves along the axis", part.Name)
		}
	}
	require.Zero(t, offsets[planetarygearbox.InputShaftName].Len(), "the input shaft stays put")
	span := func(name string) axialSpan {
		t.Helper()
		body, ok := bodies[name]
		require.True(t, ok, name)
		return spanOf(t, body, offsets[name], axis)
	}

	// Two parts may share space only when they move by the same offset, which leaves them as assembled: a captive nut
	// in the ring and a planet in mesh with its sun.
	for _, pair := range layout.Overlaps {
		require.InDelta(t, 0, offsets[pair[0]].Sub(offsets[pair[1]]).Len(), 1e-9, "%s and %s share space", pair[0], pair[1])
	}

	last := len(box.Stages) - 1
	bore := ballbearing.Params605ZZ().BoreDiameter
	// A shaft's collar is wider than the 605ZZ bore and cannot pass the bearing, so the bearing and the cover slide off
	// the shaft's free end and the shaft stays on the carrier side of both. The input shaft's free end is at the front,
	// and the output shaft's is at the rear.
	for _, name := range []string{planetarygearbox.InputShaftName, planetarygearbox.OutputShaftName} {
		box := bounds(t, bodies[name])
		require.Greater(t, math.Max(box.Max.X-box.Min.X, box.Max.Y-box.Min.Y), bore, "%s is wider than the bearing bore", name)
	}
	input, front, frontBearing := span(planetarygearbox.InputShaftName), span(planetarygearbox.FrontCoverName),
		span(planetarygearbox.FrontBearingName)
	require.True(t, input.above(front), "the input shaft is on the carrier side of the front cover")
	require.True(t, front.above(frontBearing), "the front bearing is outside the front cover")
	require.True(t, input.above(frontBearing), "the input shaft is on the carrier side of the front bearing")
	output := span(planetarygearbox.OutputShaftName)
	ring, rear, rearBearing := span(planetarygearbox.RingName), span(planetarygearbox.RearCoverName),
		span(planetarygearbox.RearBearingName)
	require.True(t, ring.above(output), "the ring slides off the output shaft's free end")
	require.True(t, rear.above(ring), "the rear cover is outside the ring")
	require.True(t, rear.above(output), "the output shaft is on the carrier side of the rear cover")
	require.True(t, rearBearing.above(rear), "the rear bearing is outside the rear cover")
	require.True(t, rearBearing.above(output), "the output shaft is on the carrier side of the rear bearing")
	require.True(t, output.lo >= span(planetarygearbox.CarrierName(last)).lo, "the output shaft leaves its carrier")

	// Each bolt comes out of its cover, on the cover's outer side.
	for i := range box.Bolts {
		half := len(box.Bolts) / 2
		if i < half {
			require.True(t, front.above(span(planetarygearbox.FrontBoltName(i))), "front bolt %d is outside the front cover", i+1)
			continue
		}
		require.True(t, span(planetarygearbox.RearBoltName(i-half)).above(rear), "rear bolt %d is outside the rear cover", i-half+1)
	}

	// Up each stage, the parts come apart in the order they were assembled: planets and sun, planet bearings, carrier pins,
	// carrier, shaft. A planet bearing slides off the free end of its pin, and a pin leaves its carrier from below.
	prev := span(planetarygearbox.SunName(0))
	require.True(t, prev.above(input), "the first sun is off the input shaft's end")
	for k, s := range box.Stages {
		sun := span(planetarygearbox.SunName(k))
		if k > 0 {
			require.True(t, sun.above(prev), "sun %d is above the shaft before it", k+1)
		}
		carrier := span(planetarygearbox.CarrierName(k))
		shaft := span(planetarygearbox.ShaftName(k, len(box.Stages)))
		for i := range s.Planets {
			planet := span(planetarygearbox.PlanetName(k, i))
			bearing := span(planetarygearbox.BearingName(k, i))
			pin := span(planetarygearbox.PinName(k, i))
			require.True(t, bearing.above(planet), "bearing %d.%d is out of its planet", k+1, i+1)
			require.True(t, pin.above(bearing), "bearing %d.%d is off the pin's free end", k+1, i+1)
			require.True(t, carrier.above(pin), "pin %d.%d is out of its carrier", k+1, i+1)
		}
		require.True(t, shaft.lo >= carrier.lo, "shaft %d leaves its carrier", k+1)
		prev = shaft
	}

	require.NotContains(t, layout.Stages, "", "every part is in a named stage")
}

func TestExplodedGearbox(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	doc, box := defaultGearbox(t)
	require.Len(t, box.Stages, 1)
	requireExplodedGearbox(t, doc, box, p)
}

func TestExplodedTwoStageGearbox(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	p.Stages = 2
	doc := planetarygearbox.NewDocument()
	box, err := planetarygearbox.Build(t.Context(), doc, p)
	require.NoError(t, err)
	requireExplodedGearbox(t, doc, box, p)

	set, err := box.Set(doc, p)
	require.NoError(t, err)
	t.Run("joints", func(t *testing.T) {
		joints := set.Joints()
		names := map[string]struct{}{}
		for _, j := range joints {
			names[j.Name] = struct{}{}
		}
		for _, want := range []string{
			"input shaft in the front bearing", "input shaft in the front cover", "input shaft in the first sun",
			"Link Shaft 1 in Carrier 1", "Link Shaft 1 in Sun Gear 2", "Output Shaft in Carrier 2",
			"output shaft in the rear bearing", "output shaft in the rear cover", "the ring clears the output shaft",
			"pin 2.3 in its bearing", "pin 2.3 in the carrier", "bearing 2.3 in its planet", "planet 2.3 meshes the ring",
		} {
			require.Contains(t, names, want)
		}
		// A shaft joins its bearing as a pin in a bore: the bearing's bore is the narrowest hole it passes.
		for _, j := range joints {
			if strings.Contains(j.Name, "in the front bearing") || strings.Contains(j.Name, "in the rear bearing") {
				require.Equal(t, parts.PinInBore, j.Kind, j.Name)
			}
		}
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
		require.Equal(t, 2+2*p.PlanetCount, count["605ZZ ball bearing"], "two cover bearings and one for each planet")
		require.Equal(t, 8, count["M2 hex nut"])
		require.Equal(t, 8, count["M2 x 6 bolt"])
	})
}

func TestExplodedPlacedGearbox(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	f, err := r3.NewFrame(r3.NewVec(10, -5, 7), r3.NewVec(0, 1, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	p.Placement = &f
	doc := planetarygearbox.NewDocument()
	box, err := planetarygearbox.Build(t.Context(), doc, p)
	require.NoError(t, err)
	requireExplodedGearbox(t, doc, box, p)
}
