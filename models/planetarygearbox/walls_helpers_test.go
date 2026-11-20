package planetarygearbox_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/planetarygearbox"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

// minPressWall is the press-fit wall of "Corners and roots": four 0.4 mm lines.
const minPressWall = 1.6

// pointSegment is the distance from p to the segment ab, in the XY plane.
func pointSegment(p, a, b r3.Vec) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	length := dx*dx + dy*dy
	f := 0.0
	if length > 0 {
		f = math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/length))
	}
	return math.Hypot(p.X-a.X-f*dx, p.Y-a.Y-f*dy)
}

// loopSpan is the area of the bounding box of a loop, which tells the outline from the holes inside it.
func loopSpan(loop []r3.Vec) float64 {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, v := range loop {
		minX, maxX = math.Min(minX, v.X), math.Max(maxX, v.X)
		minY, maxY = math.Min(minY, v.Y), math.Max(maxY, v.Y)
	}
	return (maxX - minX) * (maxY - minY)
}

// loopGap is the least distance between the vertices of each loop and the segments of the other.
func loopGap(a, b []r3.Vec) float64 {
	least := math.Inf(1)
	between := func(a, b []r3.Vec) {
		for _, v := range a {
			for j := range b {
				least = math.Min(least, pointSegment(v, b[j], b[(j+1)%len(b)]))
			}
		}
	}
	between(a, b)
	between(b, a)
	return least
}

// requireBoreWalls requires every hole in the section of body at height z to keep at least 1.6 mm of wall to every
// other loop of the section, the outline and the other holes, less the section's measurement bound. It requires holes
// holes.
func requireBoreWalls(t *testing.T, name string, body *decad.Body, z float64, holes int) {
	t.Helper()
	plane, err := r3.NewFrame(r3.NewVec(0, 0, z), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	require.NoError(t, err)
	section := sectionOutline(t, body, plane)
	outline := 0
	for i, l := range section.Loops {
		if loopSpan(l) > loopSpan(section.Loops[outline]) {
			outline = i
		}
	}
	require.Len(t, section.Loops, holes+1, "%s: loops of the section at Z %v", name, z)
	bound := section.OffSurface.Base()
	for i, hole := range section.Loops {
		if i == outline {
			continue
		}
		for j, other := range section.Loops {
			if j == i {
				continue
			}
			wall := loopGap(hole, other)
			require.GreaterOrEqual(t, wall-bound, minPressWall, "%s: wall round hole %d to loop %d at Z %v", name, i, j, z)
		}
	}
}

// requirePressFitWalls applies the press-fit clause of "Corners and roots" to the planetary parts: the sun's D bore
// of every stage keeps 1.6 mm of wall to the tooth roots, and each carrier keeps it round its pin sockets and its
// shaft hole. The sections are taken at mid-height of the part, where no chamfer or step reaches.
func requirePressFitWalls(t *testing.T, box planetarygearbox.Gearbox, p planetarygearbox.Params) {
	t.Helper()
	pitch := p.GearThickness + p.CarrierThickness + 2*p.StageGap
	for k, stage := range box.Stages {
		z := float64(k) * pitch
		t.Run(planetarygearbox.SunName(k), func(t *testing.T) {
			requireBoreWalls(t, planetarygearbox.SunName(k), stage.Sun, z+p.GearThickness/2, 1)
		})
		t.Run(planetarygearbox.CarrierName(k), func(t *testing.T) {
			requireBoreWalls(t, planetarygearbox.CarrierName(k), stage.Carrier,
				z+p.GearThickness+p.StageGap+p.CarrierThickness/2, p.PlanetCount+1)
		})
	}
}
