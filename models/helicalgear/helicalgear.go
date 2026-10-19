// Package helicalgear builds gears with involute teeth twisted between two axial sections.
package helicalgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/placement"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params adds a top-section twist to the spur gear dimensions.
// Angles are radians and lengths are millimeters.
type Params struct {
	spurgear.Params
	HelixAngle float64
}

// DefaultParams returns the spur defaults with the source's 14.5-degree twist.
func DefaultParams() Params {
	return Params{Params: spurgear.DefaultParams(), HelixAngle: 14.5 * math.Pi / 180}
}

// ValidateHelixAngle accepts a positive twist from 1 through 45 degrees.
// The angle is in radians.
func ValidateHelixAngle(angle float64) error {
	if math.IsNaN(angle) || math.IsInf(angle, 0) {
		return fmt.Errorf("helix angle must be finite")
	}
	if angle == 0 {
		return fmt.Errorf("zero helix angle is a spur gear; use spurgear.Build")
	}
	if angle < math.Pi/180 || angle > math.Pi/4 {
		return fmt.Errorf("helix angle must be from 1 to 45 degrees")
	}
	return nil
}

// Build adds a helical gear to doc. ctx and doc must not be nil.
// It builds the gear about the local origin and +Z, then moves it to Params.Placement,
// the reference point and plane. A nil Placement leaves it on the world XY plane.
// Assemble, PreparePattern, and FinishAssembly work in the local frame and ignore Placement.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if err := ValidateHelixAngle(p.HelixAngle); err != nil {
		return nil, err
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return nil, fmt.Errorf("placement frame is not valid")
	}
	local := p
	local.Placement = nil
	world := sketch.NewWorld()
	tooth, err := buildTooth(ctx, doc, world, local)
	if err != nil {
		return nil, err
	}
	gear, err := Assemble(ctx, doc, local, tooth)
	if err != nil {
		return nil, err
	}
	return placement.Place(ctx, gear, p.Placement)
}

// Assemble joins a completed tooth around a full-thickness root disc and applies
// the inherited finishing operations. ctx, doc, and tooth must not be nil.
func Assemble(ctx context.Context, doc *decad.Document, p Params, tooth *decad.Body) (*decad.Body, error) {
	blank, pattern, err := PreparePattern(ctx, doc, p, tooth)
	if err != nil {
		return nil, err
	}
	gear := blank
	for i, instance := range pattern {
		gear, err = decad.Union(ctx, gear, instance)
		if err != nil {
			return nil, fmt.Errorf("join tooth %d: %w", i+1, err)
		}
	}
	return FinishAssembly(ctx, doc, p, gear)
}

// PreparePattern makes the full-thickness root disc and circular tooth pattern.
// ctx, doc, and tooth must not be nil. The caller joins the pattern to the disc.
func PreparePattern(ctx context.Context, doc *decad.Document, p Params, tooth *decad.Body) (*decad.Body, []*decad.Body, error) {
	if ctx == nil || doc == nil || tooth == nil {
		return nil, nil, fmt.Errorf("context, document, and tooth must not be nil")
	}
	if p.Teeth < 1 {
		return nil, nil, fmt.Errorf("tooth count must be positive")
	}
	world := sketch.NewWorld()
	blank, err := buildRootDisc(ctx, doc, world, p)
	if err != nil {
		return nil, nil, fmt.Errorf("build root disc: %w", err)
	}
	pattern := make([]*decad.Body, p.Teeth)
	pattern[0] = tooth
	for i := 1; i < p.Teeth; i++ {
		turn, err := r3.Rotation(r3.NewVec(0, 0, 1),
			units.Radians(2*math.Pi*float64(i)/float64(p.Teeth)))
		if err != nil {
			return nil, nil, fmt.Errorf("rotate tooth %d: %w", i+1, err)
		}
		pattern[i], err = decad.Copy(ctx, tooth, decad.WithCopyPlacement(turn))
		if err != nil {
			return nil, nil, fmt.Errorf("place tooth %d: %w", i+1, err)
		}
	}
	return blank, pattern, nil
}

// FinishAssembly applies the inherited root fillet, optional bore, and cap chamfer
// to a joined gear. ctx, doc, and gear must not be nil.
func FinishAssembly(ctx context.Context, doc *decad.Document, p Params, gear *decad.Body) (*decad.Body, error) {
	if ctx == nil || doc == nil || gear == nil {
		return nil, fmt.Errorf("context, document, and gear must not be nil")
	}
	gear, err := filletRootEdges(ctx, gear, p)
	if err != nil {
		return nil, err
	}
	if p.BoreDiameter > 0 {
		gear, err = spurgear.CutThroughBore(ctx, doc, gear, p.Params)
		if err != nil {
			return nil, err
		}
	}
	if p.CapChamfer > 0 {
		gear, err = spurgear.ChamferOuterCaps(ctx, gear, p.CapChamfer)
		if err != nil {
			return nil, err
		}
	}
	return gear, nil
}

func buildTooth(ctx context.Context, doc *decad.Document, world *sketch.World,
	p Params) (*decad.Body, error) {
	frame, err := r3.NewFrame(r3.NewVec(0, 0, p.Thickness), r3.NewVec(1, 0, 0), r3.NewVec(0, 1, 0))
	if err != nil {
		return nil, fmt.Errorf("create twisted section frame: %w", err)
	}
	upper, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("create twisted section plane: %w", err)
	}
	lowerSketch, lowerProfile, err := spurgear.BuildToothSection(ctx, world, world.XY(), p.Params, 0)
	if err != nil {
		return nil, fmt.Errorf("build lower tooth section: %w", err)
	}
	upperSketch, upperProfile, err := spurgear.BuildToothSection(ctx, world, upper, p.Params, p.HelixAngle)
	if err != nil {
		return nil, fmt.Errorf("build twisted tooth section: %w", err)
	}
	tooth, err := decad.Loft(ctx, doc, lowerSketch, lowerProfile, upperSketch, upperProfile)
	if err != nil {
		return nil, fmt.Errorf("loft helical tooth: %w", err)
	}
	return tooth, nil
}

func buildRootDisc(ctx context.Context, doc *decad.Document, world *sketch.World,
	p Params) (*decad.Body, error) {
	root := p.Module * (float64(p.Teeth) - 2.5) / 2
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return nil, err
	}
	circle := s.CreateCircle(s.Origin(), root)
	s.AddConstraint(sketch.NewRadius(circle, root))
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, fmt.Errorf("verify root disc sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("root disc sketch has %d profiles; want one valid region", len(profiles))
	}
	body, err := decad.Extrude(ctx, doc, s, profiles[0], decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	return body, nil
}
