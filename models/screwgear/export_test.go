package screwgear

import (
	"github.com/lestrrat-3d/r3"
)

// GearForTest is one gear of the pair frame, exposed for the frame tests.
type GearForTest struct {
	Origin, Ex, Ey, Ez r3.Vec
	Mount, Phase       float64
}

// PairForTest returns both gears in the pair frame, with the function that maps a pair-frame
// point onto the placement.
func PairForTest(p Params) ([2]GearForTest, func(r3.Vec) r3.Vec, error) {
	pf, err := newPairFrame(p)
	if err != nil {
		return [2]GearForTest{}, nil, err
	}
	var out [2]GearForTest
	for i, g := range pf.gears {
		out[i] = GearForTest{Origin: g.origin, Ex: g.ex, Ey: g.ey, Ez: g.ez, Mount: g.mount, Phase: g.phase}
	}
	return out, pf.world, nil
}

// GearAngleForTest is gear gi's cross-section angle at station s.
func GearAngleForTest(p Params, gi int, s float64) float64 {
	pf, _ := newPairFrame(p)
	return pf.gears[gi].angle(s)
}

// GearWorldForTest maps section coordinates (u, v) at station s of gear gi to the pair frame.
func GearWorldForTest(p Params, gi int, u, v, s float64) r3.Vec {
	pf, _ := newPairFrame(p)
	return pf.gears[gi].world(u, v, s)
}

// GearLocalForTest is GearWorldForTest's inverse.
func GearLocalForTest(p Params, gi int, pt r3.Vec) (float64, float64, float64) {
	pf, _ := newPairFrame(p)
	return pf.gears[gi].local(pt)
}

// BoreOpeningForTest is the opening of gear gi's bore on the side sign (-1 or +1).
func BoreOpeningForTest(p Params, gi int, sign float64) (float64, float64, float64) {
	pf, _ := newPairFrame(p)
	return boreOpening(pf.gears[gi], sign)
}

// RoofSideForTest is which long face of gear gi's bore on the side sign is its roof.
func RoofSideForTest(p Params, gi int, sign float64) float64 {
	pf, _ := newPairFrame(p)
	return roofSide(pf.gears[gi], sign)
}

// ChannelSeparationForTest is the separation between neighbouring bores and the gap it is least across.
func ChannelSeparationForTest(p Params) (float64, string) {
	pf, err := newPairFrame(p)
	if err != nil {
		return 0, err.Error()
	}
	return channelSeparation(newPlainSleeve(pf))
}

// WindowForTest is one window of the search.
type WindowForTest struct {
	Facing                           r3.Vec
	Lean                             float64
	Lo, Hi, Bottom, Top, Left, Right float64
	ZLimit                           float64
	Corners                          [][2]float64
	Room                             string
	Area                             float64
}

// WindowsForTest runs the window search at p and returns the window for each facing.
func WindowsForTest(p Params) ([2]WindowForTest, error) {
	pf, err := newPairFrame(p)
	if err != nil {
		return [2]WindowForTest{}, err
	}
	f := newSleeve(pf)
	var out [2]WindowForTest
	for i, w := range f.windows {
		out[i] = WindowForTest{
			Facing: w.facing, Lean: w.lean, Lo: w.lo, Hi: w.hi, Bottom: w.bottom, Top: w.top,
			Left: w.left, Right: w.right, ZLimit: w.zLimit, Corners: w.corners, Room: w.room(), Area: w.area(),
		}
	}
	return out, nil
}

// WindowFacingsForTest is the two directions the windows face.
func WindowFacingsForTest(p Params) [2]r3.Vec { return windowFacings(p) }
