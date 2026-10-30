package pedalspindle_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/pedalspindle"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func TestPedalSpindleFootmanClearsInputShaft(t *testing.T) {
	doc := decad.New()
	p := pedalspindle.DefaultParams()
	p.CrankAngle = math.Pi / 2
	a, err := pedalspindle.Build(t.Context(), doc, p)
	require.NoError(t, err)
	shaftBounds, err := partByName(t, a, "input-shaft").Bounds()
	require.NoError(t, err)
	rodBounds, err := partByName(t, a, "footman").Bounds()
	require.NoError(t, err)
	// Disjoint conservative boxes prove separation of the entire real solids.
	// WithClearances additionally asks for a closest-distance measurement that
	// Decad cannot currently supply for this faceted shaft/analytic rod pair.
	gap := shaftBounds.Min.Y - rodBounds.Max.Y
	bound := shaftBounds.Bound.Base() + rodBounds.Bound.Base()
	require.InDelta(t, 5.0, gap, bound+1e-9, "nominal axial gap at the upper crank position")
	require.Greater(t, gap-bound, 4.9, "certified axial separation must exclude solid intersection")
}

// Sample real placed solids every five degrees. Shared axial faces are allowed;
// any pair with positive axial overlap is checked using its actual XZ sections.
func TestPedalSpindleLinkageSweep(t *testing.T) {
	for _, scale := range []float64{1, .8, 1.2} {
		doc := decad.New()
		p := pedalspindle.DefaultParams()
		p.Scale = scale
		a, err := pedalspindle.Build(t.Context(), doc, p)
		require.NoError(t, err)
		for degrees := 0; degrees < 360; degrees += 5 {
			pose := p
			pose.CrankAngle = float64(degrees) * math.Pi / 180
			state, err := pedalspindle.Motion(pose)
			require.NoError(t, err)
			parts := make([]pedalspindle.Part, len(a.Parts))
			moving := make([]bool, len(a.Parts))
			copied := make([]bool, len(a.Parts))
			for i, part := range a.Parts {
				parts[i] = part
				center, angle, offset, moves := linkagePlacement(part.Name, a.State, state, pose.CrankAngle-p.CrankAngle)
				moving[i] = moves
				// Pose the input D shaft with the crank. It is an obstacle for this
				// linkage sweep; gear-to-shaft relative motion is tested separately.
				if part.Name == "input-shaft" || part.Name == "flywheel-hub" {
					center, angle, offset = a.State.Input, pose.CrankAngle-p.CrankAngle, r3.Vec{}
					copied[i] = true
				}
				copied[i] = copied[i] || moves
				if !copied[i] {
					continue
				}
				rotation, err := r3.RotationAround(center, r3.NewVec(0, -1, 0), units.Radians(angle))
				require.NoError(t, err)
				translation, err := r3.Translation(offset)
				require.NoError(t, err)
				xf, err := rotation.Then(translation)
				require.NoError(t, err)
				parts[i].Body, err = decad.Copy(t.Context(), part.Body, decad.WithCopyPlacement(xf))
				require.NoError(t, err)
			}
			for i, first := range parts {
				for j := i + 1; j < len(parts); j++ {
					if !moving[i] && !moving[j] {
						continue
					}
					requireNoLinkageOverlap(t, first, parts[j], scale, degrees)
				}
			}
			for i, part := range parts {
				if copied[i] {
					require.NoError(t, decad.Remove(part.Body))
				}
			}
		}
	}
}

func linkagePlacement(name string, base, pose pedalspindle.State, delta float64) (r3.Vec, float64, r3.Vec, bool) {
	switch name {
	case "crank":
		return base.Input, delta, r3.Vec{}, true
	case "treadle":
		return base.Heel, pose.PedalAngle - base.PedalAngle, r3.Vec{}, true
	case "footman":
		before := math.Atan2(base.CrankPin.Z-base.Toe.Z, base.CrankPin.X-base.Toe.X)
		after := math.Atan2(pose.CrankPin.Z-pose.Toe.Z, pose.CrankPin.X-pose.Toe.X)
		return base.Toe, after - before, pose.Toe.Sub(base.Toe), true
	case "crank-pin", "footman-bearing-crank":
		return r3.Vec{}, 0, pose.CrankPin.Sub(base.CrankPin), true
	case "toe-pin", "footman-bearing-toe":
		return r3.Vec{}, 0, pose.Toe.Sub(base.Toe), true
	default:
		return r3.Vec{}, 0, r3.Vec{}, false
	}
}

func requireNoLinkageOverlap(t *testing.T, a, b pedalspindle.Part, scale float64, degrees int) {
	t.Helper()
	// Bearing IDs and seats intentionally share exact circular boundaries.
	// Independently sampled circles can cross by chord error; build_test.go
	// checks these nominal mating diameters and centers on the analytic bodies.
	for _, seat := range [][2]string{
		{"crank-pin", "footman-bearing-crank"},
		{"toe-pin", "footman-bearing-toe"},
		{"footman", "footman-bearing-crank"},
		{"footman", "footman-bearing-toe"},
		{"treadle", "heel-bearing-front"},
		{"treadle", "heel-bearing-rear"},
	} {
		if (a.Name == seat[0] && b.Name == seat[1]) || (b.Name == seat[0] && a.Name == seat[1]) {
			return
		}
	}
	ab, err := a.Body.Bounds()
	require.NoError(t, err)
	bb, err := b.Body.Bounds()
	require.NoError(t, err)
	// The tolerance excludes nominal shared faces, not finite solid intersections.
	const faceTolerance = 1e-6
	if math.Min(ab.Max.X, bb.Max.X)-math.Max(ab.Min.X, bb.Min.X) <= faceTolerance ||
		math.Min(ab.Max.Z, bb.Max.Z)-math.Max(ab.Min.Z, bb.Min.Z) <= faceTolerance {
		return
	}
	yMin, yMax := math.Max(ab.Min.Y, bb.Min.Y), math.Min(ab.Max.Y, bb.Max.Y)
	if yMax-yMin <= faceTolerance {
		return
	}
	plane, err := r3.NewFrame(r3.NewVec(0, (yMin+yMax)/2, 0), r3.NewVec(1, 0, 0), r3.NewVec(0, 0, 1))
	require.NoError(t, err)
	ao, err := a.Body.SectionOutline(t.Context(), plane, units.Millimeters(.01))
	require.NoError(t, err, "%s section", a.Name)
	bo, err := b.Body.SectionOutline(t.Context(), plane, units.Millimeters(.01))
	require.NoError(t, err, "%s section", b.Name)
	gap, err := ao.Clearance(bo)
	require.NoError(t, err, "%s / %s", a.Name, b.Name)
	require.False(t, gap.Overlaps, "%s / %s overlap at scale %g, crank %d degrees", a.Name, b.Name, scale, degrees)
}
