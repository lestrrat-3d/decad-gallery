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

// The body between the bearing seats is larger than the seat, so a part bored with a D chord at bodyPressFlat
// slides over a seat on its way to the body. Every body section has its flat at bodyFlatX.
const (
	bodyRadius       = 2.8  // arbor body between the seats
	driveBodyRadius  = 3.05 // great wheel arbor under the great wheel, the pawl carrier and the mainspring
	bodyFlatX        = 2.55 // flat plane of every body section; above 2.5, so the cutting box never touches a seat
	bodyPressRadius  = 2.85 // D bore arc on bodyRadius, 0.05 press
	drivePressRadius = 3.10 // D bore arc on driveBodyRadius, 0.05 press
	bodyPressFlat    = 2.60 // D bore chord on either body section, 0.05 press; above 2.5, so it passes a seat
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

// flatSpan is one flat cut along an arbor: the flat plane x = X over z in [Z0, Z1].
type flatSpan struct{ X, Z0, Z1 float64 }

// flatBox extrudes the box x in [flatX, flatReach], y in [-3.5, 3.5] over z in [z0, z1] about the axis (x, y).
func (b *builder) flatBox(what string, x, y, flatX, z0, z1 float64) (*decad.Body, error) {
	s, err := b.flat(what, x, y, z0)
	if err != nil {
		return nil, err
	}
	rect := s.CreateRectangle(flatX, -flatReach, flatReach, flatReach)
	s.AddConstraint(
		sketch.NewHorizontalDistance(s.Origin(), rect.A, flatX),
		sketch.NewVerticalDistance(s.Origin(), rect.A, -flatReach),
		sketch.NewHorizontalDistance(rect.A, rect.B, flatReach-flatX),
		sketch.NewVerticalDistance(rect.A, rect.D, 2*flatReach),
	)
	return b.extrude(s, 0, z1-z0, what)
}

// arbor builds a stepped shaft and cuts its flats over the spans.
func (b *builder) arbor(what string, x, y float64, pts meridianPoints, spans []flatSpan) (*decad.Body, error) {
	body, err := b.stepped(what, x, y, pts)
	if err != nil {
		return nil, err
	}
	var boxes []*decad.Body
	for i, sp := range spans {
		box, err := b.flatBox(fmt.Sprintf("%s flat %d", what, i+1), x, y, sp.X, sp.Z0, sp.Z1)
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
	escapeArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {bodyRadius, 0.25}, {bodyRadius, 24.0},
		{3.7, 24.0}, {3.7, 27.75}, {2.5, 27.75}, {2.5, 33}, {0, 33}}
	palletArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {3.7, 0.25},
		{3.7, 20.95}, {bodyRadius, 20.95}, {bodyRadius, 27.75}, {2.5, 27.75}, {2.5, 38.25}, {0, 38.25}}
	greatArborPoints = meridianPoints{{0, -5}, {2.5, -5}, {2.5, 0.25}, {3.7, 0.25},
		{3.7, 3.25}, {driveBodyRadius, 3.25}, {driveBodyRadius, 17.45}, {bodyRadius, 17.45}, {bodyRadius, 27.75},
		{2.5, 27.75}, {2.5, 38.25}, {0, 38.25}}
	pinPoints = meridianPoints{{0, 31.05}, {2.5, 31.05}, {2.5, 34}, {3.7, 34},
		{3.7, 39.80}, {2.5, 39.80}, {2.5, 46.75}, {0, 46.75}}

	escapeArborFlats = []flatSpan{{bodyFlatX, 0.20, 23.95}}
	palletArborFlats = []flatSpan{{bodyFlatX, 21.0, 27.80}, {FlatX, 34.25, 39.25}}
	greatArborFlats  = []flatSpan{{bodyFlatX, 5.25, 27.80}, {FlatX, 34.25, 39.25}}
	pinFlats         = []flatSpan{{FlatX, 30.05, 33.95}, {FlatX, 45.30, 47.75}}
)

func (b *builder) buildEscapeArbor() (*decad.Body, error) {
	return b.arbor("escape arbor", 0, 0, escapeArborPoints, escapeArborFlats)
}

func (b *builder) buildPalletArbor() (*decad.Body, error) {
	return b.arbor("pallet arbor", 0, b.g.P.Y, palletArborPoints, palletArborFlats)
}

func (b *builder) buildGreatWheelArbor() (*decad.Body, error) {
	return b.arbor("great wheel arbor", b.g.G.X, b.g.G.Y, greatArborPoints, greatArborFlats)
}

func (b *builder) buildSuspensionPin() (*decad.Body, error) {
	return b.arbor("suspension pin", 0, b.g.S.Y, pinPoints, pinFlats)
}
