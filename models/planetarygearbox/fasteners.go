package planetarygearbox

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/m2bolt"
	"github.com/lestrrat-3d/decad-gallery/models/m2nut"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

const (
	// fastenerClearance is the radial gap between an M2 bolt head or nut and its printed pocket.
	fastenerClearance = 0.1
	// minBoltHoleDiameter is the smallest bolt hole: the M2 clearance hole.
	minBoltHoleDiameter = 2.4
	// coverFloor is the cover material kept under a bolt head's pocket.
	coverFloor = 1
	// pocketMargin is the material kept between a fastener pocket and the ring's tooth roots or outer wall.
	pocketMargin = 1
	// fastenerLength is the bolt length: the shortest standard M2 length. Each bolt holds one cover to the ring,
	// so its length does not depend on the number of stages.
	fastenerLength = 6.0
	// fastenerStock is the head pocket tool's reach beyond a face, so no tool face lies in a body face.
	fastenerStock = 1.0
	// nutRoof is the ring material between a ring end face and the cavity that holds a nut. The bolt hole runs
	// through it, and it keeps the nut from falling out of the ring.
	nutRoof = 1.0
)

// FrontBoltName is the name of front bolt i (0-based). Four bolts hold each cover to the ring, each head sunk flush
// in the cover's outer face and each screwed into a nut held in a closed cavity inside the ring's wall,
// nutRoof behind the ring's end face.
func FrontBoltName(i int) string { return fmt.Sprintf("Front Bolt %d", i+1) }

// RearBoltName is the name of rear bolt i (0-based).
func RearBoltName(i int) string { return fmt.Sprintf("Rear Bolt %d", i+1) }

// FrontNutName is the name of front nut i (0-based).
func FrontNutName(i int) string { return fmt.Sprintf("Front Nut %d", i+1) }

// RearNutName is the name of rear nut i (0-based).
func RearNutName(i int) string { return fmt.Sprintf("Rear Nut %d", i+1) }

// headPocketDiameter is the diameter of the pocket that sinks a bolt head in a cover.
func headPocketDiameter() float64 { return m2bolt.HeadDiameter + 2*fastenerClearance }

// nutPocketFlats is the across-flats width of the cavity that holds a nut in the ring.
func nutPocketFlats() float64 { return m2nut.AcrossFlats + 2*fastenerClearance }

// nutCavityHeight is the axial height of the cavity that holds a nut: the nut's thickness and the clearance at each face.
func nutCavityHeight() float64 { return m2nut.Thickness + 2*fastenerClearance }

// cutHeadPockets cuts the cover's four head pockets into the +Z (outer) face of the frame so the
// bolt head ends level with the face.
func (r *run) cutHeadPockets(ctx context.Context, cover *decad.Body) (*decad.Body, error) {
	p, d := r.p, r.d
	plane, err := r.planeAt(p.CoverThickness - m2bolt.HeadHeight)
	if err != nil {
		return nil, fmt.Errorf("head pocket: %w", err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("head pocket: sketch: %w", err)
	}
	circle := s.CreateCircle(s.Origin(), headPocketDiameter()/2)
	s.AddConstraint(sketch.NewDiameter(circle, headPocketDiameter()))
	profile, err := profileWithHoles(ctx, s, 0, "head pocket")
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(m2bolt.HeadHeight + fastenerStock), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("head pocket: extrude: %w", err)
	}
	var moves []r3.Transform
	for _, angle := range boltAngles() {
		m, err := pose(r3.Identity(), 0, d.boltCircle*math.Cos(angle), d.boltCircle*math.Sin(angle), 0)
		if err != nil {
			return nil, fmt.Errorf("head pocket pose: %w", err)
		}
		moves = append(moves, m)
	}
	tools, err := instances(ctx, tool, moves)
	if err != nil {
		return nil, fmt.Errorf("head pockets: %w", err)
	}
	for i, t := range tools {
		if cover, err = decad.Cut(ctx, cover, t); err != nil {
			return nil, fmt.Errorf("cut head pocket %d: %w", i, err)
		}
	}
	if !cover.IsSolid() {
		return nil, fmt.Errorf("cover with head pockets is not a solid")
	}
	return cover, nil
}

// buildNutTool builds the cutter for one nut cavity about the local origin: a hexagon of the cavity's
// across-flats width with its flats parallel to X, from the origin up the cavity height.
func (r *run) buildNutTool(ctx context.Context) (*decad.Body, error) {
	plane, err := r.planeAt(0)
	if err != nil {
		return nil, fmt.Errorf("nut pocket: %w", err)
	}
	s, err := r.world.CreateSketch(plane)
	if err != nil {
		return nil, fmt.Errorf("nut pocket: sketch: %w", err)
	}
	flats := nutPocketFlats()
	hex, err := s.CreatePolygon(0, 0, 6, flats/math.Sqrt(3))
	if err != nil {
		return nil, fmt.Errorf("nut pocket: hexagon: %w", err)
	}
	s.AddConstraint(
		sketch.NewCoincident(hex.Center, s.Origin()),
		sketch.NewHorizontal(hex.Sides[1]),
		sketch.NewDistancePointLine(hex.Center, hex.Sides[1], flats/2),
	)
	profile, err := profileWithHoles(ctx, s, 0, "nut pocket")
	if err != nil {
		return nil, err
	}
	tool, err := decad.Extrude(ctx, r.doc, s, profile,
		decad.Distance{D: units.Millimeters(nutCavityHeight()), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("nut pocket: extrude: %w", err)
	}
	return tool, nil
}

// fastenerSite is one bolt position at one end of the housing.
type fastenerSite struct {
	x, y, angle float64
}

func (r *run) fastenerSites() []fastenerSite {
	var sites []fastenerSite
	for _, angle := range boltAngles() {
		sites = append(sites, fastenerSite{r.d.boltCircle * math.Cos(angle), r.d.boltCircle * math.Sin(angle), angle})
	}
	return sites
}

// cutNutPockets cuts a closed hexagonal cavity for a nut inside each end of the ring blank, at the four bolt
// angles. The blank is the ring's outer cylinder in construction coordinates, from ringMargin up the ring length.
// A cavity starts nutRoof inside the end face, so the bolt hole is its only opening and the nut cannot fall out of
// the ring. The nut goes in while the ring prints, with a pause under the cavity's roof. A nut's flats face the bolt
// circle's radius, so the cavity keeps the most wall. The cavities are cut before the tooth spaces, because the
// cuts then run on a plain cylinder.
func (r *run) cutNutPockets(ctx context.Context, blank *decad.Body) (*decad.Body, error) {
	tool, err := r.buildNutTool(ctx)
	if err != nil {
		return nil, err
	}
	front := float64(ringMargin) + nutRoof
	rear := float64(ringMargin) + r.d.ringLength - nutRoof - nutCavityHeight()
	var moves []r3.Transform
	for _, z := range []float64{front, rear} {
		for _, s := range r.fastenerSites() {
			m, err := pose(r3.Identity(), nutAngle(s), s.x, s.y, z)
			if err != nil {
				return nil, fmt.Errorf("nut cavity pose: %w", err)
			}
			moves = append(moves, m)
		}
	}
	tools, err := instances(ctx, tool, moves)
	if err != nil {
		return nil, fmt.Errorf("nut cavities: %w", err)
	}
	for i, t := range tools {
		if blank, err = decad.Cut(ctx, blank, t); err != nil {
			return nil, fmt.Errorf("cut nut cavity %d: %w", i, err)
		}
	}
	if !blank.IsSolid() {
		return nil, fmt.Errorf("ring blank with nut cavities is not a solid")
	}
	return blank, nil
}

// nutAngle is the turn that puts a nut's flats toward the bolt circle's radius; its corners lie on X at zero turn.
func nutAngle(s fastenerSite) float64 { return s.angle - math.Pi/2 }

// buildFasteners builds the eight M2 bolts and nuts. Each bolt's head ends level with its cover's outer face, so
// nothing stands proud, and each nut sits in its closed cavity in the ring. The bolt tip stays inside the ring's
// bolt hole beyond the nut.
func (r *run) buildFasteners(ctx context.Context, box *Gearbox) error {
	p, d := r.p, r.d
	ringEnd := d.ringLength - p.StageGap
	sites := r.fastenerSites()
	// A nut rests in its cavity with the clearance below it.
	frontNutZ := -p.StageGap + nutRoof + fastenerClearance
	rearNutZ := ringEnd - nutRoof - fastenerClearance - m2nut.Thickness
	var err error

	bolt, err := m2bolt.Build(ctx, r.doc, m2bolt.Params{Length: fastenerLength})
	if err != nil {
		return fmt.Errorf("M2 bolt: %w", err)
	}
	nut, err := m2nut.Build(ctx, r.doc, m2nut.DefaultParams())
	if err != nil {
		return fmt.Errorf("M2 nut: %w", err)
	}

	// Front bolts are turned over so the shank points into the ring; the head's underside rests on the pocket floor.
	frontFloor := -(p.StageGap + p.CoverThickness) + m2bolt.HeadHeight
	rearFloor := ringEnd + p.CoverThickness - m2bolt.HeadHeight
	var boltMoves, nutMoves []r3.Transform
	for _, s := range sites {
		m, err := poseFlippedAt(r.frame, s.x, s.y, frontFloor)
		if err != nil {
			return fmt.Errorf("front bolt pose: %w", err)
		}
		boltMoves = append(boltMoves, m)
	}
	for _, s := range sites {
		m, err := pose(r.frame, 0, s.x, s.y, rearFloor)
		if err != nil {
			return fmt.Errorf("rear bolt pose: %w", err)
		}
		boltMoves = append(boltMoves, m)
	}
	for _, z := range []float64{frontNutZ, rearNutZ} {
		for _, s := range sites {
			m, err := pose(r.frame, nutAngle(s), s.x, s.y, z)
			if err != nil {
				return fmt.Errorf("nut pose: %w", err)
			}
			nutMoves = append(nutMoves, m)
		}
	}
	if box.Bolts, err = instances(ctx, bolt.Body, boltMoves); err != nil {
		return fmt.Errorf("bolts: %w", err)
	}
	if box.Nuts, err = instances(ctx, nut.Body, nutMoves); err != nil {
		return fmt.Errorf("nuts: %w", err)
	}
	return nil
}
