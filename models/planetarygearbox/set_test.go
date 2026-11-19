package planetarygearbox_test

// CI: isolated-tests

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
			strings.Join([]string{planetarygearbox.InputShaftName, planetarygearbox.SunName(0),
				planetarygearbox.FrontCollarName}, ", "),
			strings.Join([]string{planetarygearbox.CarrierName(0), planetarygearbox.PinName(0, 0),
				planetarygearbox.PinName(0, 1), planetarygearbox.PinName(0, 2), planetarygearbox.OutputShaftName,
				planetarygearbox.RearCollarName}, ", "),
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
		// Eight housing nuts and one in each of the three ears; the two-stage ring is 27 mm long, so each ear takes
		// an M2 x 25 bolt.
		require.Equal(t, 11, count["M2 hex nut"])
		require.Equal(t, 8, count["M2 x 6 bolt"])
		require.Equal(t, 3, count["M2 x 25 bolt"])
	})
}

func TestExplodedPlacedGearbox(t *testing.T) {
	p := placedParams(t)
	doc, box := placedGearbox(t)
	requireExplodedGearbox(t, doc, box, p)
}

// TestInputShaftCollar checks the input shaft's collar on the built body against Arbor collars: it is at least
// 3.0 mm long, and it keeps the Tight 0.2 mm gap to the front bearing's inner face and the collar clearance to the sun.
func TestInputShaftCollar(t *testing.T) {
	p := planetarygearbox.DefaultParams()
	_, box := defaultGearbox(t)
	axis := parts.Axis{Point: r3.Vec{}, Dir: r3.Vec{Z: 1}}
	rod := ballbearing.Params605ZZ().BoreDiameter / 2
	collar := collarSpan(t, box.InputShaft, axis, rod+1.15)
	// The collar ends in a 30° cone on the sun's side, measured to where the cone meets the rod: its length is the
	// 1.2 mm of radius times the cone run.
	collar.hi += 1.2 * math.Sqrt(3)
	require.GreaterOrEqual(t, collar.hi-collar.lo, 3.0, "collar length")
	require.InDelta(t, p.StageGap+p.CoverThickness-0.75, collar.hi-collar.lo, 2e-3)
	bearingFace := bounds(t, box.FrontBearing).Max.Z
	require.GreaterOrEqual(t, collar.lo-bearingFace, 0.2, "gap to the front bearing")
	require.InDelta(t, 0.25, collar.lo-bearingFace, 2e-3)
	sunFace := bounds(t, box.Stages[0].Sun).Min.Z
	require.InDelta(t, 0.5, sunFace-collar.hi, 2e-3, "collar clearance to the sun")
}
