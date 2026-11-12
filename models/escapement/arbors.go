package escapement

import (
	"fmt"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/sketch"
)

// Arbor dimensions in mm: the shaft is the 605ZZ bore, the collar is 7.4 across, the flat is 0.5 deep on +X.
const (
	ArborRadius  = 2.5
	CollarRadius = 3.7
	FlatX        = 2.0
	flatReach    = 3.5
)

// meridianPoints is a (radius, z) polygon with a point on the axis at each end.
type meridianPoints [][2]float64

// stepped revolves the staircase polygon pts about the vertical line through (x, y).
func (b *builder) stepped(what string, x, y float64, pts meridianPoints) (*decad.Body, error) {
	s, err := b.meridian(what, x, y)
	if err != nil {
		return nil, err
	}
	ps := make([]*sketch.Point, len(pts))
	for i, p := range pts {
		ps[i] = s.CreatePoint(p[0], p[1])
	}
	// The first point is fixed by both distances. Each later point shares one coordinate with its predecessor along
	// a horizontal or vertical line and takes the other by a distance; a slanted line fixes both. The last point
	// lies on the axis and the closing line down the axis follows from the two zero radii.
	s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[0], pts[0][0]),
		sketch.NewVerticalDistance(s.Origin(), ps[0], pts[0][1]))
	for i := 1; i < len(pts); i++ {
		line := s.CreateLine(ps[i-1], ps[i])
		switch {
		case pts[i-1][1] == pts[i][1]:
			s.AddConstraint(sketch.NewHorizontal(line), sketch.NewHorizontalDistance(s.Origin(), ps[i], pts[i][0]))
		case pts[i-1][0] == pts[i][0]:
			s.AddConstraint(sketch.NewVertical(line), sketch.NewVerticalDistance(s.Origin(), ps[i], pts[i][1]))
		default:
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), ps[i], pts[i][0]),
				sketch.NewVerticalDistance(s.Origin(), ps[i], pts[i][1]))
		}
	}
	s.CreateLine(ps[len(ps)-1], ps[0])
	return b.revolve(s, what)
}

// flatBox extrudes the box x in [FlatX, flatReach+0], y in [-3.5, 3.5] over z in [z0, z1] about the axis (x, y).
func (b *builder) flatBox(what string, x, y, z0, z1 float64) (*decad.Body, error) {
	s, err := b.flat(what, x, y, z0)
	if err != nil {
		return nil, err
	}
	rect := s.CreateRectangle(FlatX, -flatReach, flatReach, flatReach)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), rect.A, FlatX),
		sketch.NewVerticalDistance(s.Origin(), rect.A, -flatReach),
		sketch.NewHorizontalDistance(rect.A, rect.B, flatReach-FlatX),
		sketch.NewVerticalDistance(rect.A, rect.D, 2*flatReach),
	)
	return b.extrude(s, 0, z1-z0, what)
}

// arbor builds a stepped shaft and cuts its flats over the spans.
func (b *builder) arbor(what string, x, y float64, pts meridianPoints, spans [][2]float64) (*decad.Body, error) {
	body, err := b.stepped(what, x, y, pts)
	if err != nil {
		return nil, err
	}
	var boxes []*decad.Body
	for i, sp := range spans {
		box, err := b.flatBox(fmt.Sprintf("%s flat %d", what, i+1), x, y, sp[0], sp[1])
		if err != nil {
			return nil, err
		}
		boxes = append(boxes, box)
	}
	tool := boxes[0]
	if len(boxes) > 1 {
		if tool, err = b.union(what+" flats", boxes[0], boxes[1:]...); err != nil {
			return nil, err
		}
	}
	return b.cut(body, tool, what+" flats")
}

var (
	escapeArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 25.05}, {3.7, 25.05}, {3.7, 27.75},
		{2.5, 27.75}, {2.5, 33}, {0, 33}}
	palletArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 19.95}, {3.7, 19.95},
		{3.7, 20.95}, {2.5, 20.95}, {2.5, 37.2}, {0, 37.2}}
	greatArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {3.7, 0.25},
		{3.7, 1.25}, {2.5, 1.25}, {2.5, 37.2}, {0, 37.2}}
	pinPoints = meridianPoints{{0, 31.05}, {2.5, 31.05}, {2.5, 34}, {3.7, 34},
		{3.7, 38.75}, {2.5, 38.75}, {2.5, 45.70}, {0, 45.70}}
)

func (b *builder) buildEscapeArbor() (*decad.Body, error) {
	return b.arbor("escape arbor", 0, 0, escapeArborPoints, [][2]float64{{0.5, 25.0}})
}

func (b *builder) buildPalletArbor() (*decad.Body, error) {
	return b.arbor("pallet arbor", 0, b.g.P.Y, palletArborPoints, [][2]float64{{21.0, 27.75}, {34.25, 38.2}})
}

func (b *builder) buildGreatWheelArbor() (*decad.Body, error) {
	return b.arbor("great wheel arbor", b.g.G.X, b.g.G.Y, greatArborPoints, [][2]float64{{5.25, 27.75}, {34.25, 38.2}})
}

func (b *builder) buildSuspensionPin() (*decad.Body, error) {
	return b.arbor("suspension pin", 0, b.g.S.Y, pinPoints, [][2]float64{{30.05, 33.95}, {44.25, 46.70}})
}
