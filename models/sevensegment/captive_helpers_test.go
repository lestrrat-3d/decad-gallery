package sevensegment_test

import (
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/sevensegment"
	"github.com/lestrrat-3d/r3"
)

// slideraise is the droop a b-f slider (0 for a, g, d, which sag instead: see groovePinTurnover) settles to about its groove pin: the model's built-in raise plus RaiseBelowDroop (the
// pin is modeled that much below the droop). It is 0.25 for b and c and 0.30 for e and f.
func slideraise(s sevensegment.Segment) float64 {
	if _, _, horizontal := segmentCentreOf(s); horizontal {
		return 0 // a, g, d hang on a separate pin and sag straight down: they do not tilt
	}
	return sevensegment.RaiseOf(s) + sevensegment.RaiseBelowDroop
}

// meshTolerance is the depth to which a vertex of one mesh may sit inside another before the test calls it penetration: the
// tessellation bound of the two bodies (0.05 each) leaves a vertex a few hundredths inside a surface it touches by design.
const meshTolerance = 0.07

// penetration returns the deepest vertex of verts inside the solid mesh, as the distance from the vertex to the solid's
// surface (gap, posed like the solid). toSolid maps a point to the solid's own frame, where the inside test (pointIn, on
// the solid's crossing grid) runs.
func penetration(verts []r3.Vec, solid *crossingGrid, gap meshgap.Mesh, toSolid func(r3.Vec) r3.Vec) float64 {
	deepest := 0.0
	for _, v := range verts {
		if !solid.pointIn(toSolid(v)) {
			continue
		}
		d, err := gap.PointDistance(v)
		if err == nil && d > deepest {
			deepest = d
		}
	}
	return deepest
}

// driveGeometry returns whether segment s's slider runs along X (a, g, d: u is world X) or Y, the u of the tongue's cone face, and
// the station and height of its drive axis, at the build pose of the slider with station yi.
func driveGeometry(s sevensegment.Segment, yi float64) (uIsX bool, ue, sc, zp float64) {
	switch s {
	case sevensegment.SegA, sevensegment.SegG, sevensegment.SegD:
		return true, -2.85, yi + sevensegment.PinStationAGD, sevensegment.PinAxisZ + sevensegment.RaiseOf(s) // the whole body is built RaiseOf higher
	case sevensegment.SegE, sevensegment.SegF:
		return false, yi - 0.3, -sevensegment.PinStationEF, sevensegment.PinAxisZ + sevensegment.RaiseOf(s)
	}
	return false, yi - 0.3, sevensegment.PinStationBC, sevensegment.PinAxisZ + sevensegment.RaiseOf(s)
}

// driveSamples returns points on the surface of segment s's drive cone: the frustum from the base circle (radius coneBaseR about
// the base centre, on the cone face) to the junction circle, and the right cone from the junction to the tip with its chamfer.
// A mesh holds a cone only at its end rings, so a distance from the mesh vertices misses the section at the pocket mouth.
func driveSamples(s sevensegment.Segment, yi float64) []r3.Vec {
	uIsX, ue, sc, zp := driveGeometry(s, yi)
	dc, zc := sc+sevensegment.ToAxis(uIsX)*sevensegment.ConeOffsetW, zp+sevensegment.ConeOffsetZ
	uj := ue + sevensegment.ConeSpan - sevensegment.ConeJoin
	slope := sevensegment.ConeSlope
	pt := func(u, w, z float64) r3.Vec {
		if uIsX {
			return r3.Vec{X: u, Y: w, Z: z}
		}
		return r3.Vec{X: w, Y: u, Z: z}
	}
	var out []r3.Vec
	for i := 0; i <= 40; i++ {
		f := float64(i) / 40
		cw, cz := dc+f*(sc-dc), zc+f*(zp-zc)
		rad := sevensegment.ConeBaseR + f*(sevensegment.ConeJoinR-sevensegment.ConeBaseR)
		for a := 0; a < 360; a += 4 {
			ar := float64(a) * math.Pi / 180
			out = append(out, pt(ue+f*(uj-ue), cw+rad*math.Cos(ar), cz+rad*math.Sin(ar)))
		}
	}
	for i := 0; i <= 80; i++ {
		x := sevensegment.ConeLength * float64(i) / 80
		rad := sevensegment.ConeJoinR - x*slope
		if edge := sevensegment.ConeLength - sevensegment.ConeChamfer; x > edge {
			rad -= (x - edge) * (1 - slope)
		}
		for a := 0; a < 360; a += 4 {
			ar := float64(a) * math.Pi / 180
			out = append(out, pt(uj+x, sc+rad*math.Cos(ar), zp+rad*math.Sin(ar)))
		}
	}
	return out
}

// tongueVertices returns the vertices of segment s's slider that lie on its tongue: the part from the print face ua to the cone
// face ue above z 72.9, away from the blade.
func tongueVertices(s sevensegment.Segment, yi float64, m *decad.Mesh) []r3.Vec {
	_, _, horizontal := segmentCentreOf(s)
	var tongue []r3.Vec
	for _, v := range m.Vertices() {
		if horizontal {
			if v.Z < sevensegment.SliderHeadTop+0.1 || v.Y < yi-1.5 {
				continue
			}
			if v.X <= -2.85+0.05 && v.X >= -5.25 {
				tongue = append(tongue, v)
			}
			continue
		}
		if v.Z < sevensegment.SliderHeadTop+0.1 || math.Abs(v.X) < 18.0 {
			continue
		}
		if v.Y <= yi-0.3+0.05 && v.Y >= yi-sevensegment.StemThickness/2 {
			tongue = append(tongue, v)
		}
	}
	return tongue
}

// segmentCentreOf reports whether segment s lies along X (a, g, d).
func segmentCentreOf(s sevensegment.Segment) (float64, float64, bool) {
	switch s {
	case sevensegment.SegA, sevensegment.SegG, sevensegment.SegD:
		return 0, 0, true
	}
	return 0, 0, false
}

// pinDistance is the drive axis's distance from the stem along x for a b-f slider: 22.5 for b and c, 27 for e and f.
func pinDistance(s sevensegment.Segment) float64 {
	switch s {
	case sevensegment.SegB, sevensegment.SegC:
		return sevensegment.PinStationBC
	case sevensegment.SegE, sevensegment.SegF:
		return sevensegment.PinStationEF
	}
	return 0
}

// groovePinTurnover is how far a, g and d rise further when the unit is turned over: their separate pin bears on the hole at
// its mouth and root, so the slider sags 0.264 and the sag reverses (0.528 more travel). The built mesh is already RaiseOf
// higher than the pose the design's 0.528 is measured from, so the pose takes the difference. It is 0 for b to f.
func groovePinTurnover(s sevensegment.Segment) float64 {
	if _, _, horizontal := segmentCentreOf(s); !horizontal {
		return 0
	}
	return 2*0.264 - sevensegment.RaiseOf(s)
}
