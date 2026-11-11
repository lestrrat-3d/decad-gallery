package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// earWall mirrors the material the ears keep around a bore.
const earWall = 3

// ringOuterRadius is the ring's outer wall for the default module: the ring root radius plus the wall.
func ringOuterRadius(p planetarygearbox.Params) float64 {
	return p.Module*float64(p.RingTeeth)/2 + p.Module + minimumGearRootClearance(p.Module) + p.RingWall
}

// insideAt returns a classifier of points in the plane z against the section of body, by even-odd crossings.
func insideAt(t *testing.T, body *decad.Body, z float64) func(x, y float64) bool {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	section := sectionOutline(t, body, plane)
	return func(x, y float64) bool {
		crossings := 0
		for _, loop := range section.Loops {
			for i, a := range loop {
				b := loop[(i+1)%len(loop)]
				if (a.Y > y) == (b.Y > y) {
					continue
				}
				if x < a.X+(y-a.Y)/(b.Y-a.Y)*(b.X-a.X) {
					crossings++
				}
			}
		}
		return crossings%2 == 1
	}
}

// requireEars checks the mounting ears of ring: each stands beyond the outer wall, runs the whole ring length, has
// its bore through it and nothing else, and clears the covers' outer wall by the ear's bore wall.
func requireEars(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	outer := ringOuterRadius(p)
	half := p.EarBoreDiameter/2 + earWall
	center := outer + half
	first, last := -p.StageGap, float64(p.Stages)*(p.GearThickness+p.CarrierThickness+2*p.StageGap)
	cover := bounds(t, box.FrontCover)
	require.InDelta(t, outer, cover.Max.X, 0.05)
	require.GreaterOrEqual(t, center-p.EarBoreDiameter/2-cover.Max.X, float64(earWall)-0.05,
		"the bore is closer to the covers' outer wall than the ear's wall")

	for _, z := range []float64{first + 1, (first + last) / 2, last - 1} {
		inside := insideAt(t, box.Ring, z)
		for i := range p.EarCount {
			angle := math.Pi/2 + 2*math.Pi*float64(i)/float64(p.EarCount)
			sin, cos := math.Sincos(angle)
			at := func(along, side float64) bool { return inside(along*cos-side*sin, along*sin+side*cos) }
			for name, want := range map[string]struct {
				along, side float64
				material    bool
			}{
				"bore center":                {center, 0, false},
				"bore edge toward the ring":  {center - p.EarBoreDiameter/2 + 0.3, 0, false},
				"bore edge toward the tip":   {center + p.EarBoreDiameter/2 - 0.3, 0, false},
				"bore edge sideways":         {center, p.EarBoreDiameter/2 - 0.3, false},
				"wall toward the ring":       {center - p.EarBoreDiameter/2 - 0.3, 0, true},
				"wall toward the tip":        {center + p.EarBoreDiameter/2 + 0.3, 0, true},
				"wall sideways":              {center, p.EarBoreDiameter/2 + 0.3, true},
				"between the wall and ring":  {outer + 0.5, 0, true},
				"inside the ring wall":       {outer - 0.5, 0, true},
				"beyond the tip":             {center + half + 0.3, 0, false},
				"beyond the side":            {center, half + 0.3, false},
				"beside the ear on the wall": {outer + 0.5, half + 0.3, false},
			} {
				require.Equal(t, want.material, at(want.along, want.side), "ear %d at z=%v: %s", i+1, z, name)
			}
		}
	}
}

func TestMountingEars(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	require.Equal(t, 3, p.EarCount)
	require.InDelta(t, 4, p.EarBoreDiameter, 0)
	_, box := defaultGearbox(t)
	requireEars(t, box, p)
}

// TestEarBoreAndCount builds a different bore and ear count: two ears on the Y axis, with a 6 mm bore.
func TestEarBoreAndCount(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	p.EarCount, p.EarBoreDiameter = 2, 6
	box, err := planetarygearbox.Build(t.Context(), planetarygearbox.NewDocument(), p)
	require.NoError(t, err)
	requireEars(t, box, p)

	outer := ringOuterRadius(p)
	reach := outer + 2*(p.EarBoreDiameter/2+earWall)
	ring := bounds(t, box.Ring)
	require.InDelta(t, reach, ring.Max.Y, 0.05)
	require.InDelta(t, -reach, ring.Min.Y, 0.05)
	// The ears stand on the Y axis, so the ring reaches no further along X than its outer wall.
	require.InDelta(t, outer, ring.Max.X, 0.05)
	requireRingBevel(t, box.Ring, p)
}

// TestRingWithoutEars builds the ring with no ears: a plain cylinder, which a document without a chain tolerance
// builds, with the same rim bevel as before, and lighter than the default ring by the ears.
func TestRingWithoutEars(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	p.EarCount = 0
	doc := decad.New()
	box, err := planetarygearbox.Build(t.Context(), doc, p)
	require.NoError(t, err)
	require.Len(t, doc.Bodies(), 34)
	outer := ringOuterRadius(p)
	ring := bounds(t, box.Ring)
	require.InDelta(t, outer, ring.Max.X, 1e-3)
	require.InDelta(t, outer, ring.Max.Y, 1e-3)
	require.InDelta(t, -outer, ring.Min.Y, 1e-3)
	requireOuterEdgesChamfered(t, "ring", box.Ring, 2)
	requireRimSetback(t, "ring", box.Ring, p.EdgeChamfer)

	// Each ear adds its section over the ring length: the strip between the outer wall and the bore's center line,
	// a half circle, less the bore. The chamfers take a small part of it, within the tolerance.
	half := p.EarBoreDiameter/2 + earWall
	wall := ringOuterRadius(p)
	strip := 2*half*(wall+half) - (half*math.Sqrt(wall*wall-half*half) + wall*wall*math.Asin(half/wall))
	section := strip + math.Pi*half*half/2 - math.Pi*p.EarBoreDiameter*p.EarBoreDiameter/4
	length := float64(p.Stages)*(p.GearThickness+p.CarrierThickness+2*p.StageGap) + p.StageGap
	want := float64(3) * section * length
	_, with := defaultGearbox(t)
	plain, err := box.Ring.Volume()
	require.NoError(t, err)
	earred, err := with.Ring.Volume()
	require.NoError(t, err)
	require.InDelta(t, want, earred.Value.Base()-plain.Value.Base(), 0.03*want+plain.Bound.Base()+earred.Bound.Base())
}
