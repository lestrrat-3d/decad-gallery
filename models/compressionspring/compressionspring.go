package compressionspring

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/units"
)

type Spec struct {
	Name            string
	OutsideDiameter float64
	WireDiameter    float64
	FreeLength      float64
	TotalTurns      float64
	ActiveTurns     float64
	Rate            float64
	SolidHeight     float64
	LoadAtSolid     float64
}

func LeeLC024C14M() Spec {
	return Spec{Name: "Lee Spring LC 024C 14 M", OutsideDiameter: 6.10, WireDiameter: 0.61,
		FreeLength: 50.80, TotalTurns: 19.64, ActiveTurns: 17.64, Rate: 0.49,
		SolidHeight: 12.32, LoadAtSolid: 19.13}
}

func LeeLC026D16S() Spec {
	return Spec{Name: "Lee Spring LC 026D 16 S", OutsideDiameter: 7.62, WireDiameter: 0.66,
		FreeLength: 63.50, TotalTurns: 18.33, ActiveTurns: 16.33, Rate: 0.30,
		SolidHeight: 12.17, LoadAtSolid: 15.93}
}

type Params struct {
	Spec            Spec
	InstalledLength float64
}

func DefaultParams() Params {
	return Params{Spec: LeeLC024C14M(), InstalledLength: 43.80}
}

type Model struct {
	Body    *decad.Body
	Profile *sketch.Sketch
	Pitch   float64
	Force   float64
}

func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("compressionspring: context is nil")
	}
	if doc == nil {
		return nil, fmt.Errorf("compressionspring: document is nil")
	}
	s := p.Spec
	values := []struct {
		name  string
		value float64
	}{
		{"outside diameter", s.OutsideDiameter}, {"wire diameter", s.WireDiameter},
		{"free length", s.FreeLength}, {"total turns", s.TotalTurns},
		{"active turns", s.ActiveTurns}, {"rate", s.Rate},
		{"solid height", s.SolidHeight}, {"load at solid", s.LoadAtSolid},
		{"installed length", p.InstalledLength},
	}
	for _, v := range values {
		if math.IsNaN(v.value) || math.IsInf(v.value, 0) || v.value <= 0 {
			return nil, fmt.Errorf("compressionspring: %s must be finite and positive", v.name)
		}
	}
	if s.ActiveTurns > s.TotalTurns {
		return nil, fmt.Errorf("compressionspring: active turns exceed total turns")
	}
	if s.OutsideDiameter <= 2*s.WireDiameter {
		return nil, fmt.Errorf("compressionspring: outside diameter must exceed twice the wire diameter")
	}
	if p.InstalledLength > s.FreeLength {
		return nil, fmt.Errorf("compressionspring: installed length exceeds free length")
	}
	if p.InstalledLength <= s.SolidHeight {
		return nil, fmt.Errorf("compressionspring: installed length must exceed solid height")
	}
	pitch := (p.InstalledLength - s.WireDiameter) / s.TotalTurns
	if !isFinite(pitch) || pitch <= s.WireDiameter {
		return nil, fmt.Errorf("compressionspring: pitch must exceed wire diameter")
	}
	force := s.Rate * (s.FreeLength - p.InstalledLength)
	if !isFinite(force) {
		return nil, fmt.Errorf("compressionspring: nominal force is not finite")
	}

	w := sketch.NewWorld()
	sk, err := w.CreateSketch(w.XY())
	if err != nil {
		return nil, fmt.Errorf("compressionspring: create sketch: %w", err)
	}
	centerX := (s.OutsideDiameter - s.WireDiameter) / 2
	center := sk.CreatePoint(centerX, 0)
	circle := sk.CreateCircle(center, s.WireDiameter/2)
	sk.AddConstraint(
		sketch.NewHorizontalPoints(sk.Origin(), center),
		sketch.NewHorizontalDistance(sk.Origin(), center, centerX),
		sketch.NewDiameter(circle, s.WireDiameter),
	)
	if _, err := sk.Solve(ctx); err != nil {
		return nil, fmt.Errorf("compressionspring: solve section sketch: %w", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, sk); err != nil {
		return nil, fmt.Errorf("compressionspring: verify section sketch: %w", err)
	}
	profiles := sk.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, fmt.Errorf("compressionspring: expected one wire profile, got %d", len(profiles))
	}
	body, err := decad.Coil(ctx, doc, sk, profiles[0], decad.SketchLine{
		Start: decad.Point2{U: 0, V: 0}, End: decad.Point2{U: 0, V: 1},
	}, units.Millimeters(pitch), units.Scalar(s.TotalTurns))
	if err != nil {
		return nil, fmt.Errorf("compressionspring: build coil: %w", err)
	}
	if body == nil || !body.IsSolid() {
		return nil, fmt.Errorf("compressionspring: coil did not produce a solid")
	}
	return &Model{Body: body, Profile: sk, Pitch: pitch, Force: force}, nil
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
