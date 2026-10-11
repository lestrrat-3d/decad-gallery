// Package helicalgear builds gears with involute teeth twisted between two axial sections.
package helicalgear

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spurgear"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

// Params adds a signed top-section twist to the spur gear dimensions.
// Angles are radians and lengths are millimeters.
type Params struct {
	spurgear.Params
	HelixAngle float64
}

// DefaultParams returns the spur defaults with the source's 14.5-degree twist.
func DefaultParams() Params {
	return Params{Params: spurgear.DefaultParams(), HelixAngle: 14.5 * math.Pi / 180}
}

// Build adds a helical gear to doc. ctx and doc must not be nil.
func Build(ctx context.Context, doc *decad.Document, p Params) (*decad.Body, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("context and document must not be nil")
	}
	if math.IsNaN(p.HelixAngle) || math.IsInf(p.HelixAngle, 0) {
		return nil, fmt.Errorf("helix angle must be finite")
	}
	world := sketch.NewWorld()
	tooth, err := buildTooth(ctx, doc, world, p)
	if err != nil {
		return nil, err
	}
	blank, err := buildRootDisc(ctx, doc, world, p)
	if err != nil {
		return nil, fmt.Errorf("build root disc: %w", err)
	}
	pattern := make([]*decad.Body, p.Teeth)
	pattern[0] = tooth
	for i := 1; i < p.Teeth; i++ {
		turn, err := r3.Rotation(r3.NewVec(0, 0, 1),
			units.Radians(2*math.Pi*float64(i)/float64(p.Teeth)))
		if err != nil {
			return nil, fmt.Errorf("rotate tooth %d: %w", i+1, err)
		}
		pattern[i], err = tooth.PlacedCopy(ctx, turn)
		if err != nil {
			return nil, fmt.Errorf("place tooth %d: %w", i+1, err)
		}
	}
	gear := blank
	for i, instance := range pattern {
		gear, err = decad.Union(ctx, gear, instance)
		if err != nil {
			return nil, fmt.Errorf("join tooth %d: %w", i+1, err)
		}
	}
	gear, err = filletRootEdges(ctx, gear, p)
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
	tooth, err := doc.Loft(ctx, lowerSketch, lowerProfile, upperSketch, upperProfile)
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
	center := s.CreatePoint(0, 0)
	s.Fix(center)
	s.CreateCircle(center, root)
	if _, err := s.Solve(ctx); err != nil {
		return nil, fmt.Errorf("solve root disc sketch: %w", err)
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("root disc sketch has %d profiles; want one valid region", len(profiles))
	}
	body, err := doc.Extrude(s, profiles[0], decad.Distance{D: units.Millimeters(p.Thickness), Dir: decad.Along})
	if err != nil {
		return nil, fmt.Errorf("extrude root disc: %w", err)
	}
	return body, nil
}
