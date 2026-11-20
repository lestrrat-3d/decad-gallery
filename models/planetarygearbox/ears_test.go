package planetarygearbox_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/stretchr/testify/require"
)

func TestMountingEars(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	require.Equal(t, 3, p.EarCount)
	require.InDelta(t, 2.6, p.EarBoreDiameter, 0)
	_, box := defaultGearbox(t)
	requireEars(t, box, p)
}

// TestEarBoreAndCount builds a different bore and ear count: two ears on the Y axis, with a 3.5 mm bore, the widest
// M2 clearance hole under the nut's 4 mm across its flats.
func TestEarBoreAndCount(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	p.EarCount, p.EarBoreDiameter = 2, 3.5
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
	// 34 bodies and the two collar rings; with no ears there are no ear bolts or nuts.
	require.Len(t, doc.Bodies(), 36)
	require.Empty(t, box.EarBolts)
	require.Empty(t, box.EarNuts)
	outer := ringOuterRadius(p)
	ring := bounds(t, box.Ring)
	require.InDelta(t, outer, ring.Max.X, 1e-3)
	require.InDelta(t, outer, ring.Max.Y, 1e-3)
	require.InDelta(t, -outer, ring.Min.Y, 1e-3)
	requireOuterEdgesChamfered(t, "ring", box.Ring, 2)
	requireRimSetback(t, "ring", box.Ring, p.EdgeChamfer)

	// Each ear adds its section over the ring length: the strip between the outer wall and the bore's center line,
	// a half circle, less the bore, and the two root rounds. The chamfers take a small part of it, within the tolerance.
	half := p.EarBoreDiameter/2 + earWall
	wall := ringOuterRadius(p)
	strip := 2*half*(wall+half) - (half*math.Sqrt(wall*wall-half*half) + wall*wall*math.Asin(half/wall))
	section := strip + math.Pi*half*half/2 - math.Pi*p.EarBoreDiameter*p.EarBoreDiameter/4
	// Each ear also meets the ring in two root rounds.
	section += 2 * rootRoundArea(wall, half, earRootRound(p))
	length := float64(p.Stages)*(p.GearThickness+p.CarrierThickness+2*p.StageGap) + p.StageGap
	// Each ear loses its head pocket and its nut cavity, both 2 mm high, beyond the bore they surround.
	bore := p.EarBoreDiameter / 2
	pockets := 2 * math.Pi * (earHeadPocket*earHeadPocket + earNutCavity*earNutCavity - 2*bore*bore)
	want := float64(3) * (section*length - pockets)
	_, with := defaultGearbox(t)
	plain, err := box.Ring.Volume()
	require.NoError(t, err)
	earred, err := with.Ring.Volume()
	require.NoError(t, err)
	require.InDelta(t, want, earred.Value.Base()-plain.Value.Base(), 0.03*want+plain.Bound.Base()+earred.Bound.Base())
}

// TestRootRoundArea checks the area formula against the default ear: 1.919 mm² for each of the 3.55 mm root rounds.
func TestRootRoundArea(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	require.InDelta(t, 3.55, earRootRound(p), 1e-9)
	require.InDelta(t, 1.919, rootRoundArea(ringOuterRadius(p), p.EarBoreDiameter/2+earWall, earRootRound(p)), 0.01)
}
