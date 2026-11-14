package spiralspring

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
)

// buildProfile sketches the spring and returns the sketch, the profile with two holes, and the corners C1, C3,
// C4, P_i, P_o that the 3D fillets round.
func buildProfile(ctx context.Context, p Params, dm Dimensions) (*sketch.Sketch, *sketch.Profile, [5]r3.Vec, error) {
	var corners [5]r3.Vec
	fail := func(step string, err error) (*sketch.Sketch, *sketch.Profile, [5]r3.Vec, error) {
		return nil, nil, corners, fmt.Errorf("spiralspring: %s: %w", step, err)
	}
	t := p.Thickness
	R, g, d, w, rho0 := dm.HubRadius, dm.Gap, dm.CenterOffset, dm.SpokeWidth, dm.FirstInnerRadius
	e := dm.TaperOffset
	n := dm.Turns

	world := sketch.NewWorld()
	s, err := world.CreateSketch(world.XY())
	if err != nil {
		return fail("create sketch", err)
	}
	A := s.Origin()

	// 1. Second arc center.
	B := s.CreatePoint(d, 0)
	s.AddConstraint(sketch.NewHorizontalPoints(A, B), sketch.NewHorizontalDistance(A, B, d))
	// Half turn 0's inner edge is centred at B0 = (d + e, 0), so the strip thickens from t at J1 to t + 2e at the spoke.
	B0 := s.CreatePoint(d+e, 0)
	s.AddConstraint(sketch.NewHorizontalPoints(A, B0), sketch.NewHorizontalDistance(A, B0, d+e))

	// 2. Root.
	C1 := s.CreatePoint(-R, 0)
	C2 := s.CreatePoint(-(R + g + t), 0)
	C3 := s.CreatePoint(d+e-math.Sqrt((rho0-e)*(rho0-e)-w*w), -w)
	C4 := s.CreatePoint(-math.Sqrt(R*R-w*w), -w)
	hub := s.CreateArc(A, C4, C1)
	L1 := s.CreateLine(C1, C2)
	L2 := s.CreateLine(C3, C4)
	s.AddConstraint(
		sketch.NewRadius(hub, R),
		sketch.NewHorizontal(L1),
		sketch.NewHorizontalPoints(A, C2),
		sketch.NewHorizontal(L2),
		sketch.NewDistancePointLine(A, L2, w),
	)

	// 3. Coil.
	rk := func(k int) float64 { return rho0 + float64(k)*d }
	inner := make([]*sketch.Arc, 2*n)
	outer := make([]*sketch.Arc, 2*n)
	J := C3
	O := C2
	for k := 0; k < 2*n; k++ {
		center, innerCenter := A, A
		jx, ox := -rk(k), -(rk(k) + t)
		if k%2 == 0 {
			center, innerCenter = B, B
			if k == 0 {
				innerCenter = B0
			}
			jx, ox = d+rk(k), d+rk(k)+t
		}
		Jn := s.CreatePoint(jx, 0)
		On := s.CreatePoint(ox, 0)
		inner[k] = s.CreateArc(innerCenter, J, Jn)
		outer[k] = s.CreateArc(center, O, On)
		s.AddConstraint(sketch.NewHorizontalPoints(A, Jn), sketch.NewHorizontalPoints(A, On))
		J, O = Jn, On
	}
	s.AddConstraint(sketch.NewRadius(inner[0], rho0-e), sketch.NewRadius(outer[0], rho0+t))

	// 4. Tail and eye.
	ex, l, re := dm.EyeCenterX, -dm.EyeCenterY, dm.EyeRadius
	yc := -l + math.Sqrt(re*re-t*t/4)
	E := s.CreatePoint(ex, -l)
	Pi := s.CreatePoint(ex+t/2, yc)
	Po := s.CreatePoint(ex-t/2, yc)
	tailIn := s.CreateLine(J, Pi)
	tailOut := s.CreateLine(O, Po)
	eye := s.CreateArc(E, Po, Pi)
	hole := s.CreateCircle(E, dm.EyeHoleDiameter/2)
	s.AddConstraint(
		sketch.NewVertical(tailIn),
		sketch.NewVertical(tailOut),
		sketch.NewRadius(eye, re),
		sketch.NewVerticalDistance(J, E, -l),
		sketch.NewHorizontalDistance(O, E, t/2),
		sketch.NewDiameter(hole, dm.EyeHoleDiameter),
	)

	// 5. Bore.
	rb, f := dm.BoreRadius, dm.BoreFlatDistance
	if p.ArborFlatDepth > 0 {
		h := math.Sqrt(rb*rb - f*f)
		Q1 := s.CreatePoint(h, f)
		Q2 := s.CreatePoint(-h, f)
		bore := s.CreateArc(A, Q2, Q1)
		flat := s.CreateLine(Q1, Q2)
		s.AddConstraint(
			sketch.NewRadius(bore, rb),
			sketch.NewHorizontal(flat),
			sketch.NewDistancePointLine(A, flat, f),
		)
	} else {
		bore := s.CreateCircle(A, rb)
		s.AddConstraint(sketch.NewDiameter(bore, 2*rb))
	}

	// 6. Solve and verify.
	if _, err := s.Solve(ctx); err != nil {
		return fail("solve profile", err)
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return fail("verify profile", err)
	}

	// 7. Select the spring profile.
	profiles := s.Profiles()
	if len(profiles) != 3 {
		return fail("select profile", fmt.Errorf("got %d profiles, want 3", len(profiles)))
	}
	var spring *sketch.Profile
	for _, pr := range profiles {
		if !pr.Valid {
			return fail("select profile", fmt.Errorf("a profile is not valid"))
		}
		if len(pr.Holes) == 2 {
			if spring != nil {
				return fail("select profile", fmt.Errorf("more than one profile has two holes"))
			}
			spring = pr
		}
	}
	if spring == nil {
		return fail("select profile", fmt.Errorf("no profile has two holes"))
	}

	at := func(pt *sketch.Point) r3.Vec { return r3.Vec{X: pt.X(), Y: pt.Y()} }
	corners = [5]r3.Vec{at(C1), at(C3), at(C4), at(Pi), at(Po)}
	return s, spring, corners, nil
}
