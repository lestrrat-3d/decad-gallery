package sevensegment_test

// CI: isolated-tests

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// TestSevenSegmentCaptive turns the unit over at every digit. Against the settled rest the flags rise 0.5, the sliders 0.8 and
// the b-f sliders a further 2 raise at the pin (the tilt reverses); settled, the flags sit 0.5 and the sliders 0.4 below
// the modeled pose, so the flags are at their modeled height and the sliders 0.4 above it. Every rotor stays on its pivot pins (the flag touches neither the plates nor the pivot blocks), every pin
// in its pocket and every blade in its tube (the slider does not touch the black body or the plates).
func TestSevenSegmentCaptive(t *testing.T) {
	startDigit0()
	t.Parallel()
	var worstTop, worstBlack float64
	parts := builtDigit0(t)
	dm := parts.Dimensions
	var sliders, yellows, blacks [7]*decad.Mesh
	var blackGrids [7]*crossingGrid
	var plates [3]meshgap.Mesh
	var blocks [7][2]meshgap.Mesh
	for _, s := range sevensegment.Segments() {
		sliders[s], _ = meshOf(t, parts.Sliders[s], 0.05)
		yellows[s], _ = meshOf(t, parts.Flags[s].Yellow, 0.05)
		blacks[s], _ = meshOf(t, parts.Flags[s].Black, 0.05)
		blackGrids[s] = newCrossingGrid(blacks[s])
	}
	// The physical (clamped) geometry: the top plate 0.50 and the cradle 0.25 lower than modeled; the guide plate stays.
	shaft := r3.Vec{Z: sevensegment.ShaftZ}
	topMesh, _ := meshOf(t, parts.TopPlate, 0.05)
	topGrid := newCrossingGrid(topMesh)
	cradleMesh, _ := meshOf(t, parts.CradlePlate, 0.05)
	_, plates[2] = meshOf(t, parts.GuidePlate, 0.05)
	plates[0] = posed(t, topMesh, shaft, r3.Vec{Y: 1}, 0, -0.5)
	plates[1] = posed(t, cradleMesh, shaft, r3.Vec{Y: 1}, 0, -0.25)
	// The blocks are pressed in the cradle's and the top plate's pockets, which close on the seam: they sit with the cradle.
	for _, s := range sevensegment.Segments() {
		for end := range 2 {
			blockMesh, _ := meshOf(t, parts.PivotBlocks[s][end], 0.05)
			blocks[s][end] = posed(t, blockMesh, shaft, r3.Vec{Y: 1}, 0, -0.25)
		}
	}
	for k := range 10 {
		for _, s := range sevensegment.Segments() {
			was, _ := sevensegment.Lit(0, s)
			now, _ := sevensegment.Lit(k, s)
			lift := sevensegment.Travel * (b2f(now) - b2f(was))
			// The b-f slider tilts the other way when the unit is turned over: a turn about its groove pin that raises the rotor
			// pin by raise more than the modeled level pose (the droop that the model builds in reverses, so the pin is 2 raise
			// above its settled pose) (the pin is 15 or 18 from the stem). It rises a further 0.4 over the modeled pose, 0.2 short of
			// its tube's roof (the roof is 0.07 away at that pose; the 0.05 mesh bounds of the two bodies hide a smaller gap), which it meets by design at the end of the groove's play (a mesh cannot prove separation at
			// contact).
			tilt := 0.0
			if slideraise(s) > 0 {
				tilt = math.Atan(slideraise(s)/pinDistance(s)) * 180 / math.Pi
				if s == sevensegment.SegB || s == sevensegment.SegC {
					tilt = -tilt // the pin is at +x: a turn about +Y lowers it, so raising it turns the other way
				}
			}
			// The tilt also lifts the blade, which is 9 from the stem, by tan(tilt) x 9: the tube's roof, 0.8 above the settled
			// lit blade, then stops a b-f slider that much short of the groove's full 0.8 rise (the design's note that the
			// roof is 0.90 away leaves the tilt out). The roof stops it; the pin follows from the slider's height.
			bladeLift := math.Abs(math.Tan(tilt*math.Pi/180)) * Scale * 9
			// The tilt is about the stem in its guide slot (z 54), where the blade and the stem have their 0.25 play: at the
			// limit the blade meets its tube wall by design, so the turn is taken at half of it (the meshes' 0.1 mm bound hides a smaller gap; the limit itself is covered by TestSevenSegmentTongueSweep).
			pivotX := 0.25 // the stem rests on the slot wall that the blade's tilt leaves it against
			if s == sevensegment.SegE || s == sevensegment.SegF {
				pivotX = -0.25
			}
			groove := r3.Vec{X: pivotX, Y: dm.Station(s), Z: sevensegment.GuideFrom}
			tilt *= 0.5
			slider := posed(t, sliders[s], groove, r3.Vec{Y: 1}, tilt, lift+0.4-0.2-bladeLift)
			sliderPin := posed(t, sliders[s], groove, r3.Vec{Y: 1}, tilt, lift+0.4-0.2-bladeLift)
			point, axis := segmentAxis(s)
			// The rotor follows its pin: the pin line is 0.4 higher (and 2 raise more on b-f), which turns the rotor.
			zrel := dm.Lift(s, 36*float64(k)) - Scale*1.75 + sevensegment.RaiseOf(s) + 0.4 - 0.2 - bladeLift + slideraise(s)/2
			dphi := math.Atan2(zrel, sevensegment.PinOffset)*180/math.Pi - dm.FlagAngle(s, 0)
			// At the full rise the bore's roof meets the pin by design (gap 0), which a mesh cannot prove separated: the flags
			// are posed 0.11 low, the mesh bound. The pin axis is 0.25 below the modeled axis once clamped and the bore leaves
			// 0.2 round the pin, so the roof stands 0.05 under the modeled axis, 0.06 above this pose.
			black := posed(t, blacks[s], point, axis, dphi, -0.11)
			yellow := posed(t, yellows[s], point, axis, dphi, -0.11)
			for _, block := range blocks[s] {
				for _, flag := range []meshgap.Mesh{black, yellow} {
					ok, err := block.ExceedsClearance(flag, 0)
					require.NoError(t, err)
					require.True(t, ok, "digit %d: turned over, flag %s touches a pivot block (its pin left its bore)", k, s)
				}
			}
			for _, plate := range plates[:2] {
				for _, flag := range []meshgap.Mesh{black, yellow} {
					ok, err := plate.ExceedsClearance(flag, 0)
					require.NoError(t, err)
					require.True(t, ok, "digit %d: turned over, flag %s touches a plate (its journal left its hole)", k, s)
				}
			}
			ok, err := plates[0].ExceedsClearance(slider, 0)
			require.NoError(t, err)
			if slideraise(s) == 0 {
				require.True(t, ok, "digit %d: turned over, slider %s touches the top plate (its blade left its tube)", k, s)
			}
			ok, err = black.ExceedsClearance(sliderPin, 0)
			require.NoError(t, err)
			if slideraise(s) == 0 {
				require.True(t, ok, "digit %d: turned over, slider %s and its black body overlap", k, s)
			}
			// The b-f sliders meet their tube roof and their black body at about zero clearance by design, which a mesh cannot
			// prove separated. They must not go deeper than the mesh tolerance into either.
			turn, err := r3.RotationAround(groove, r3.Vec{Y: 1}, units.Degrees(tilt))
			require.NoError(t, err)
			verts := make([]r3.Vec, 0, len(sliders[s].Vertices()))
			for _, v := range sliders[s].Vertices() {
				p := turn.Apply(v)
				p.Z += lift + 0.4 - 0.2 - bladeLift
				verts = append(verts, p)
			}
			depthTop := penetration(verts, topGrid, plates[0], func(p r3.Vec) r3.Vec { p.Z += 0.5; return p })
			back, err := r3.RotationAround(point, axis, units.Degrees(-dphi))
			require.NoError(t, err)
			depthBlack := penetration(verts, blackGrids[s], black, func(p r3.Vec) r3.Vec { p.Z += 0.11; return back.Apply(p) })
			require.LessOrEqual(t, depthTop, meshTolerance, "digit %d: slider %s goes into the top plate", k, s)
			require.LessOrEqual(t, depthBlack, meshTolerance, "digit %d: slider %s goes into its black body", k, s)
			if slideraise(s) > 0 {
				worstTop, worstBlack = math.Max(worstTop, depthTop), math.Max(worstBlack, depthBlack)
			}
		}
	}
	t.Logf("b-f sliders turned over: deepest vertex %.3f into the top plate, %.3f into the black body (tolerance %.2f)", worstTop, worstBlack, meshTolerance)
}

// TestSevenSegmentSweep poses every flag and slider at each digit and at every ramp's middle, from the modeled pose, and
// measures the tongue's gap to the black body (T8, at least 0.7) and the pin's gap to the pocket walls (T7, at least 0.2)
// as point distances from the slider's pin and tongue vertices to the black mesh.
func TestSevenSegmentSweep(t *testing.T) {
	startDigit0()
	t.Parallel()
	parts := builtDigit0(t)
	dm := parts.Dimensions
	for _, s := range sevensegment.Segments() {
		slider, _ := meshOf(t, parts.Sliders[s], 0.05)
		black, _ := meshOf(t, parts.Flags[s].Black, 0.05)
		pin, tongue := pinAndTongue(s, dm.Station(s), slider)
		require.NotEmpty(t, pin, "segment %s pin", s)
		require.NotEmpty(t, tongue, "segment %s tongue", s)
		point, axis := segmentAxis(s)
		for k := range 10 {
			for _, theta := range []float64{36 * float64(k), 36*float64(k) + 18.75} {
				dz := dm.Lift(s, theta) - dm.Lift(s, 0)
				dphi := dm.FlagAngle(s, theta) - dm.FlagAngle(s, 0)
				turn, err := r3.RotationAround(point, axis, units.Degrees(dphi))
				require.NoError(t, err)
				moved := make([]r3.Vec, len(black.Vertices()))
				for i, v := range black.Vertices() {
					moved[i] = turn.Apply(v)
				}
				bm, err := meshgap.New(moved, black.Triangles(), black.Bound().Base())
				require.NoError(t, err)
				least := func(vs []r3.Vec) float64 {
					best := math.Inf(1)
					for _, v := range vs {
						d, err := bm.PointDistance(v.Add(r3.Vec{Z: dz}))
						require.NoError(t, err)
						best = math.Min(best, d)
					}
					return best
				}
				require.GreaterOrEqual(t, least(tongue), 0.7-0.04, "segment %s theta %.2f: tongue (T8)", s, theta)
				require.GreaterOrEqual(t, least(pin), 0.2-0.04, "segment %s theta %.2f: pin (T7)", s, theta)
			}
		}
	}
}
