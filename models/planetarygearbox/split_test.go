package planetarygearbox

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/cornercheck"
	"github.com/lestrrat-3d/decad-gallery/internal/splitcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// splitTolerance is the chord tolerance of the split check's mesh.
const splitTolerance = 0.01

// splitPart is one printed shaft or pin, built and posed, with what its split check needs.
type splitPart struct {
	name  string
	body  *decad.Body
	point r3.Vec // a point on the part's axis, at height 0
	axis  r3.Vec
	// flat is the direction the part's flat faces, and the reference that sets the split plane when FlatDepth is zero.
	flat r3.Vec
	// rMin is the smallest radius that meets a step or a cone on the part, which sets the section heights.
	rMin   float64
	exempt []cornercheck.Exempt
}

// at returns a test for points within 0.05 mm of height z along axis whose offset from the axis passes in. The
// offset has no component along the axis.
func at(point, axis r3.Vec, z float64, in func(offset r3.Vec) bool) func(r3.Vec) bool {
	return func(p r3.Vec) bool {
		d := p.Sub(point)
		along := d.Dot(axis)
		return math.Abs(along-z) < 0.05 && in(d.Sub(axis.Scale(along)))
	}
}

// bearingStep is the exemption for a step on a part of rod radius that faces a purchased bearing's inner ring across
// a gap, at height z. Its inside corner lies on the rod's surface, where the ring's corner sits, so a cone or a round
// there would reach the ring: a mating corner with the bearing, listed under Retained sharp corners.
func bearingStep(point, axis r3.Vec, z, rod float64) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "step facing a bearing's inner ring",
		Near: at(point, axis, z, func(off r3.Vec) bool {
			return math.Abs(off.Len()-rod) < 0.06
		}),
	}
}

// flatStart is the exemption for the square start of a flat at height z, the flat plane chord from the axis, where a
// sun's D bore stops on the end of the flat: a mating corner pair, listed under Retained sharp corners.
func flatStart(point, axis, flat r3.Vec, z, chord float64) cornercheck.Exempt {
	return cornercheck.Exempt{
		Reason: "flat ending where the sun's D bore stops",
		Near: at(point, axis, z, func(off r3.Vec) bool {
			return math.Abs(off.Dot(flat)-chord) < 0.06
		}),
	}
}

// splitParts lists every printed shaft and pin of box, built from p with a nil Placement. The Gearbox's names come
// from Gearbox.Named.
func splitParts(box Gearbox, p Params) []splitPart {
	d := derive(p)
	axis := r3.Vec{Z: 1}
	bodies := map[string]*decad.Body{}
	for _, nb := range box.Named() {
		bodies[nb.Name] = nb.Body
	}
	rod, sun := d.shaft/2, d.sunShaft/2
	var parts []splitPart
	input := splitPart{
		name: InputShaftName, body: bodies[InputShaftName], axis: axis, flat: r3.Vec{X: 1}, rMin: sun,
		exempt: []cornercheck.Exempt{
			bearingStep(r3.Vec{}, axis, -(p.StageGap+p.CoverThickness)+collarBearingGap, rod),
		},
	}
	if p.FlatDepth > 0 {
		input.exempt = append(input.exempt, flatStart(r3.Vec{}, axis, input.flat, 0, sun-p.FlatDepth))
	}
	parts = append(parts, input)

	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap
	for k := range box.Stages {
		name := ShaftName(k, len(box.Stages))
		shaft := splitPart{name: name, body: bodies[name], axis: axis, flat: r3.Vec{X: 1}, rMin: sun}
		if k == len(box.Stages)-1 {
			shaft.rMin = rod
			// The output shaft's thick part ends in a square step that faces the rear bearing's inner ring.
			step := float64(k)*pitch + p.GearThickness + p.StageGap + p.CarrierThickness + p.StageGap +
				p.CoverThickness - fitClearance
			shaft.exempt = []cornercheck.Exempt{bearingStep(r3.Vec{}, axis, step, rod)}
		}
		parts = append(parts, shaft)
		for i := range p.PlanetCount {
			phi := PlanetOrbitAngle(p, i)
			point := r3.Vec{X: d.orbit * math.Cos(phi), Y: d.orbit * math.Sin(phi)}
			// The pin's thick part ends in a square step that faces the planet bearing's inner ring.
			step := float64(k)*pitch + p.GearThickness + planetBearingLift + pinShoulderClearance
			pin := PinName(k, i)
			parts = append(parts, splitPart{
				name: pin, body: bodies[pin], point: point, axis: axis, flat: r3.Vec{X: math.Cos(phi), Y: math.Sin(phi)},
				rMin: d.pin / 2, exempt: []cornercheck.Exempt{bearingStep(point, axis, step, d.pin/2)},
			})
		}
	}
	return parts
}

// requireSplit checks every printed shaft and pin of box against "Split shafts" in docs/model-pipeline.md. The split
// plane holds the part's axis and the direction its flat faces, so its normal is the axis crossed with that direction.
// Each part passes the split check, and its halves read on their cut faces with cornercheck.CheckAt at the heights
// cornercheck.SplitHeights returns: Check would read nothing, because its MidHeights come from planar faces parallel to
// the bed and a round shaft whose flats stand normal to the cut face has none. Only the listed bearing-facing steps and
// the square flat starts are exempt.
func requireSplit(t *testing.T, box Gearbox, p Params) {
	t.Helper()
	for _, part := range splitParts(box, p) {
		t.Run(part.name, func(t *testing.T) {
			require.NotNil(t, part.body)
			report, err := splitcheck.Check(t.Context(), splitcheck.Shaft{
				Body: part.body, Point: part.point, Axis: part.axis, Reference: part.flat,
			}, splitTolerance)
			require.NoError(t, err)
			require.Empty(t, report.Violations)
			want, ok := part.axis.Cross(part.flat).Normalize()
			require.True(t, ok)
			require.InDelta(t, 1, math.Abs(report.Normal.Dot(want)), 1e-6, "split normal %v", report.Normal)
			heights, err := cornercheck.SplitHeights(part.point, report.Normal, part.rMin)
			require.NoError(t, err)
			fs, err := cornercheck.CheckAt(t.Context(), part.body, report.Normal, heights, part.exempt)
			require.NoError(t, err)
			require.Empty(t, fs, cornercheck.Report(fs, 5))
		})
	}
}

// TestSplitShafts applies "Split shafts" to every shaft and pin of the default gearbox.
func TestSplitShafts(t *testing.T) {
	cornersDefault.once.Do(func() {
		cornersDefault.box, cornersDefault.err = Build(t.Context(), NewDocument(), DefaultParams())
	})
	require.NoError(t, cornersDefault.err)
	requireSplit(t, cornersDefault.box, DefaultParams())
}

// TestSplitShaftsVariants applies "Split shafts" to every parameter case that changes a shaft or a pin: the other
// kits, two stages and no flat. It takes several minutes, so it runs in the scheduled full suite and not in pull
// request CI.
func TestSplitShaftsVariants(t *testing.T) {
	kit := func(name string) func(t *testing.T) Params {
		return func(t *testing.T) Params {
			p, err := DefaultParamsFor(name)
			require.NoError(t, err)
			return p
		}
	}
	variants := []struct {
		name  string
		build func(t *testing.T) Params
	}{
		{"planet605-shaft608", kit("planet605-shaft608")},
		{"planet608-shaft605", kit("planet608-shaft605")},
		{"planet608-shaft608", kit("planet608-shaft608")},
		{"Stages 2", func(*testing.T) Params {
			p := DefaultParams()
			p.Stages = 2
			return p
		}},
		{"Stages 2, 608ZZ shaft", func(t *testing.T) Params {
			p := kit("planet605-shaft608")(t)
			p.Stages = 2
			return p
		}},
		{"FlatDepth 0", func(*testing.T) Params {
			p := DefaultParams()
			p.FlatDepth = 0
			return p
		}},
	}
	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			p := v.build(t)
			box, err := Build(t.Context(), NewDocument(), p)
			require.NoError(t, err)
			requireSplit(t, box, p)
		})
	}
}
