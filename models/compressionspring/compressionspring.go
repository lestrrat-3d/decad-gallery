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
	// ActiveTurns is the number of active coils. Zero means the supplier does not publish it. A positive value must
	// not exceed TotalTurns; a negative or non-finite value is an error.
	ActiveTurns float64
	Rate        float64
	SolidHeight float64
	LoadAtSolid float64
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

// LeeLC022D03S is the stainless click-indexer return spring, the Lee Spring catalog values at
// https://www.leespring.com/product/compression-spring-lc022d03s-stainless-steel: 7.62 mm outside diameter, 0.56 mm
// wire, 15.88 mm free length, 5.25 total and 3.25 active coils, 0.736 N/mm, 3.1 mm solid height, 9.27 N at solid.
func LeeLC022D03S() Spec {
	return Spec{Name: "Lee Spring LC 022D 03 S", OutsideDiameter: 7.62, WireDiameter: 0.56,
		FreeLength: 15.88, TotalTurns: 5.25, ActiveTurns: 3.25, Rate: 0.736,
		SolidHeight: 3.10, LoadAtSolid: 9.27}
}

// LesjoforsStock2353 is the Lesjofors stock spring Cat.no 2353, stainless steel ISO 6931-1, from the Lesjofors Stock
// Spring Catalogue 2025, page 30: 0.60 mm wire, 6.85 mm mean diameter (6.00 mm minimum inside diameter), 15 mm free
// length, 6.3 total coils, 0.86 N/mm, 3/4 of the end coils ground. The catalogue does not publish the active coils, so
// ActiveTurns is zero. Three values are computed, not published: the outside diameter 7.45 mm (mean diameter plus
// wire), the solid height 3.78 mm (wire times total coils, the catalogue's own Lst = dt x nt) and the load at solid
// 9.65 N (0.86 x (15 - 3.78)). The catalogue's permitted loaded length is 4.5 mm, where it gives 8.9 N.
func LesjoforsStock2353() Spec {
	return Spec{Name: "Lesjofors Cat.no 2353", OutsideDiameter: 7.45, WireDiameter: 0.60,
		FreeLength: 15, TotalTurns: 6.3, ActiveTurns: 0, Rate: 0.86,
		SolidHeight: 3.78, LoadAtSolid: 9.65}
}

// KSSC1087 is the KSSC K series 1087 compression spring sold by MonotaRO (order code 01353223,
// https://www.monotaro.com/p/0135/3223/). The page's spec table gives: SUS304WPB stainless steel, 0.60 mm wire,
// 7.0 mm outside diameter, 14.0 mm free length, 6.5 total turns, both ends not ground, 0.9408 N/mm, 4.5 mm solid
// height and an allowable load of 7.15 N at 6.4 mm length. The page does not publish the inside diameter, the active
// turns or the load at solid, so ActiveTurns is zero and the load at solid 8.94 N is computed:
// 0.9408 x (14.0 - 4.5). Fatigue life is not published.
func KSSC1087() Spec {
	return Spec{Name: "KSSC 1087 (MonotaRO 01353223)", OutsideDiameter: 7.0, WireDiameter: 0.60,
		FreeLength: 14.0, TotalTurns: 6.5, ActiveTurns: 0, Rate: 0.9408,
		SolidHeight: 4.5, LoadAtSolid: 8.94}
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
		{"rate", s.Rate},
		{"solid height", s.SolidHeight}, {"load at solid", s.LoadAtSolid},
		{"installed length", p.InstalledLength},
	}
	for _, v := range values {
		if math.IsNaN(v.value) || math.IsInf(v.value, 0) || v.value <= 0 {
			return nil, fmt.Errorf("compressionspring: %s must be finite and positive", v.name)
		}
	}
	if math.IsNaN(s.ActiveTurns) || math.IsInf(s.ActiveTurns, 0) || s.ActiveTurns < 0 {
		return nil, fmt.Errorf("compressionspring: active turns must be finite and not negative")
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
