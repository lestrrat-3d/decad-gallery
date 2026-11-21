package turbinefan

import (
	"math"

	"github.com/lestrrat-3d/r3"
)

// ThicknessRatio is the table's thickness ratio at radius r for the rotor sz, interpolated linearly between stations.
func ThicknessRatio(r float64, sz Size) float64 {
	stations := stationsFor(sz)
	for i := 1; i < len(stations); i++ {
		a, b := stations[i-1], stations[i]
		if r <= b.r {
			return a.ratio + (b.ratio-a.ratio)*(r-a.r)/(b.r-a.r)
		}
	}
	return stations[len(stations)-1].ratio
}

// StationRadiusChord returns the radius and chord of station k (1 to 10) of sz's blade table.
func StationRadiusChord(sz Size, k int) (float64, float64) {
	st := stationsFor(sz)[k-1]
	return st.r, st.chord
}

// SectionSurfacePoints returns station k's (1 to 10) pressure and suction surface points at chord fraction 0.4, in the
// right-hand blade's local section plane (X negated) at the reference pitch.
func SectionSurfacePoints(sz Size, k int) ([2]float64, [2]float64) {
	st := stationsFor(sz)[k-1]
	beta := sz.Beta(st.r, refPitch)
	mean := st.chord * camberLine(0.4, st.camber)
	half := halfThickness(0.4, st)
	pressure := sectionLocal(st, beta, 0.4, mean-half)
	suction := sectionLocal(st, beta, 0.4, mean+half)
	pressure[0], suction[0] = -pressure[0], -suction[0]
	return pressure, suction
}

// PosedEdgePoints returns the posed stud end corner on the in-section down side, which the stud flat cuts away
// (z = -StudFlatDepth at the rest tilt), and the posed leading-edge points of stations 4 and 5, for hand at tilt (pose
// of DefaultBladePitch).
func PosedEdgePoints(sz Size, hand Hand, tilt float64) (r3.Vec, r3.Vec, r3.Vec) {
	return PosedEdgePointsAt(sz, DefaultBladePitch, hand, tilt)
}

// PosedEdgePointsAt is PosedEdgePoints for a blade at pitch: the stud corner turns with the pose's up direction, and
// the leading-edge points with the blade's pitch turn.
func PosedEdgePointsAt(sz Size, pitch float64, hand Hand, tilt float64) (r3.Vec, r3.Vec, r3.Vec) {
	f, err := PrintPoseFor(sz, pitch, hand, tilt)
	if err != nil {
		panic(err)
	}
	xf, err := r3.FromFrame(f)
	if err != nil {
		panic(err)
	}
	u := poseUp(sz, pitch, hand)
	corner := xf.Apply(u.Scale(-sz.BladeStudRadius).Add(r3.Vec{Z: sz.BladeStudStart + studChamfer}))
	var le [2]r3.Vec
	for i, k := range []int{4, 5} {
		st := stationsFor(sz)[k-1]
		p := sectionLocal(st, sz.Beta(st.r, refPitch), 0, 0)
		turn := pitch - refPitch
		if hand == RightHand {
			p[0] = -p[0]
		} else {
			turn = -turn
		}
		x, y := p[0]*math.Cos(turn)-p[1]*math.Sin(turn), p[0]*math.Sin(turn)+p[1]*math.Cos(turn)
		le[i] = xf.Apply(r3.Vec{X: x, Y: y, Z: st.r})
	}
	return corner, le[0], le[1]
}

// RestTiltAtDepth is restTilt at a flat depth other than StudFlatDepth, for the root clearance check.
func RestTiltAtDepth(vertices []r3.Vec, sz Size, pitch float64, hand Hand, depth float64) (float64, error) {
	tilt, _, err := restTilt(vertices, sz, pitch, hand, depth)
	return tilt, err
}
