package bevelgear

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/r3"
)

// shaftFrame places a member's local frame, shaft along +X with the mate toward +Y, around its shaft in the XY
// plane. The driving member keeps the local frame. The pinion's shaft runs along the direction (cos, sin).
//
// Decad certifies a tooth join about a general axis only when that axis is an exact float vector that every
// on-axis meridian point and every cone rim lies on exactly (decad agent-docs/revolve-axis-design.md section 6).
// So the direction is rounded to 26 significant bits, and a section point (station, radius) is written as
// station*A + radius*N on a dyadic grid fine enough that each product and the sum are exact floats.
type shaftFrame struct{ cos, sin float64 }

var localShaft = shaftFrame{cos: 1}

// directionBits and gridBits split the 53 bits of a float: a coordinate on the grid carries gridBits bits and a
// component of the direction carries directionBits, so a product of the two is exact.
const (
	directionBits = 26
	gridBits      = 27
)

// newShaftFrame returns the frame whose shaft makes angle with +X. An angle of zero gives localShaft.
func newShaftFrame(angle float64) shaftFrame {
	sin, cos := math.Sincos(angle)
	f := shaftFrame{cos: roundBits(cos, directionBits), sin: roundBits(sin, directionBits)}
	if math.Abs(f.cos) < 1e-12 {
		f.cos = 0
	}
	if math.Abs(f.sin) < 1e-12 {
		f.sin = 0
	}
	return f
}

// roundBits rounds v to bits significant bits.
func roundBits(v float64, bits int) float64 {
	if v == 0 {
		return 0
	}
	mantissa, exponent := math.Frexp(v)
	return math.Ldexp(math.Round(math.Ldexp(mantissa, bits)), exponent-bits)
}

func (f shaftFrame) local() bool { return f == localShaft }

// axis is the exact float vector the member's Revolve takes as its axis. Decad reads it unnormalized.
func (f shaftFrame) axis() r3.Vec { return r3.NewVec(f.cos, f.sin, 0) }

// sketchAxis is the member's shaft as a Revolve axis from the sketch origin, the common apex.
func (f shaftFrame) sketchAxis() decad.SketchLine {
	return decad.SketchLine{Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: f.cos, V: f.sin}}
}

// unit is the direction of the shaft as a unit vector.
func (f shaftFrame) unit() (float64, float64) {
	norm := math.Hypot(f.cos, f.sin)
	return f.cos / norm, f.sin / norm
}

// vec maps a vector of the local frame to the world.
func (f shaftFrame) vec(v r3.Vec) r3.Vec {
	if f.local() {
		return v
	}
	cos, sin := f.unit()
	return r3.NewVec(v.X*cos-v.Y*sin, v.X*sin+v.Y*cos, v.Z)
}

// snap puts an axial section on the grid, so that station*A + radius*N is exact. The local frame keeps the
// section as it is.
func (f shaftFrame) snap(section []axialPoint) ([]axialPoint, error) {
	snapped := make([]axialPoint, len(section))
	if f.local() {
		copy(snapped, section)
		return snapped, nil
	}
	reach := 0.0
	for _, point := range section {
		reach = math.Max(reach, math.Max(math.Abs(point.station), math.Abs(point.radius)))
	}
	if !finite(reach) || reach == 0 {
		return nil, fmt.Errorf("section has no finite extent")
	}
	// Every coordinate stays below 2^top, so the grid step 2^(top-gridBits) keeps each one within gridBits bits.
	step := math.Ldexp(1, math.Ilogb(reach)+2-gridBits)
	for i, point := range section {
		snapped[i] = axialPoint{station: math.Round(point.station/step) * step, radius: math.Round(point.radius/step) * step}
	}
	return snapped, nil
}

// sectionXY maps an axial section to sketch coordinates in the XY plane. The local frame keeps the section as
// it is. Otherwise every point lands on the grid and is written as station*A + radius*N, which is exact.
func (f shaftFrame) sectionXY(section []axialPoint) ([][2]float64, error) {
	snapped, err := f.snap(section)
	if err != nil {
		return nil, err
	}
	xy := make([][2]float64, len(snapped))
	for i, point := range snapped {
		if f.local() {
			xy[i] = [2]float64{point.station, point.radius}
			continue
		}
		xy[i] = [2]float64{point.station*f.cos - point.radius*f.sin, point.station*f.sin + point.radius*f.cos}
	}
	return xy, nil
}
