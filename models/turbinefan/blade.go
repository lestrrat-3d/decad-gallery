package turbinefan

import (
	"context"
	"fmt"
	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/sketchcheck"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/sketch"
	"github.com/lestrrat-3d/sketch/geom"
	"github.com/lestrrat-3d/units"
	"math"
	"math/big"
	"slices"
	"strings"
)

// bladeSplineControls converts the natural chord-length interpolant to the
// clamped, uniform cubic representation required by the loft.
func bladeSplineControls(points [21][2]float64) ([23][2]float64, error) {
	var out [23][2]float64
	var lengths [20]float64
	for i := range lengths {
		lengths[i] = math.Hypot(points[i+1][0]-points[i][0], points[i+1][1]-points[i][1])
		if lengths[i] == 0 || math.IsNaN(lengths[i]) || math.IsInf(lengths[i], 0) {
			return out, fmt.Errorf("turbinefan: blade spline: invalid chord %d", i)
		}
	}
	out[0], out[22] = points[0], points[20]
	for axis := 0; axis < 2; axis++ {
		// Natural cubic second derivatives, solved by tridiagonal elimination.
		var lower, diagonal, upper, rhs, second [21]float64
		diagonal[0], diagonal[20] = 1, 1
		for i := 1; i < 20; i++ {
			lower[i], diagonal[i], upper[i] = lengths[i-1], 2*(lengths[i-1]+lengths[i]), lengths[i]
			rhs[i] = 6 * ((points[i+1][axis]-points[i][axis])/lengths[i] - (points[i][axis]-points[i-1][axis])/lengths[i-1])
		}
		for i := 1; i <= 20; i++ {
			f := lower[i] / diagonal[i-1]
			diagonal[i] -= f * upper[i-1]
			rhs[i] -= f * rhs[i-1]
		}
		second[20] = rhs[20] / diagonal[20]
		for i := 19; i >= 0; i-- {
			second[i] = (rhs[i] - upper[i]*second[i+1]) / diagonal[i]
		}
		startDerivative := points[1][axis] - points[0][axis] - lengths[0]*lengths[0]*(2*second[0]+second[1])/6
		endDerivative := points[20][axis] - points[19][axis] + lengths[19]*lengths[19]*(second[19]+2*second[20])/6
		out[1][axis] = points[0][axis] + startDerivative/3
		out[21][axis] = points[20][axis] - endDerivative/3
	}
	// The remaining controls are solved against the actual binary64 knots.
	knots := make([]*big.Rat, 27)
	for i := range knots {
		value := 0.0
		if i >= 23 {
			value = 1
		} else if i >= 4 {
			value = float64(i-3) / 20
		}
		knots[i] = new(big.Rat).SetFloat64(value)
	}
	for axis := 0; axis < 2; axis++ {
		matrix := make([][]*big.Rat, 19)
		for row := range matrix {
			matrix[row] = make([]*big.Rat, 20)
			u := knots[row+4]
			basis := bladeSplineBasis(knots, u)
			for col := 0; col < 19; col++ {
				matrix[row][col] = new(big.Rat).Set(basis[col+2])
			}
			value := new(big.Rat).SetFloat64(points[row+1][axis])
			for _, i := range []int{0, 1, 21, 22} {
				value.Sub(value, new(big.Rat).Mul(basis[i], new(big.Rat).SetFloat64(out[i][axis])))
			}
			matrix[row][19] = value
		}
		for col := 0; col < 19; col++ {
			pivot := col
			for pivot < 19 && matrix[pivot][col].Sign() == 0 {
				pivot++
			}
			if pivot == 19 {
				return out, fmt.Errorf("turbinefan: blade spline: singular collocation")
			}
			matrix[col], matrix[pivot] = matrix[pivot], matrix[col]
			p := new(big.Rat).Set(matrix[col][col])
			for j := col; j < 20; j++ {
				matrix[col][j].Quo(matrix[col][j], p)
			}
			for row := col + 1; row < 19; row++ {
				f := new(big.Rat).Set(matrix[row][col])
				for j := col; j < 20; j++ {
					matrix[row][j].Sub(matrix[row][j], new(big.Rat).Mul(f, matrix[col][j]))
				}
			}
		}
		for row := 18; row >= 0; row-- {
			value := new(big.Rat).Set(matrix[row][19])
			for col := row + 1; col < 19; col++ {
				value.Sub(value, new(big.Rat).Mul(matrix[row][col], matrix[col][19]))
			}
			matrix[row][19] = value
			out[row+2][axis], _ = value.Float64()
		}
	}
	return out, nil
}

func bladeSplineBasis(knots []*big.Rat, u *big.Rat) []*big.Rat {
	basis := make([]*big.Rat, len(knots)-1)
	for i := range basis {
		basis[i] = new(big.Rat)
		if knots[i].Cmp(u) <= 0 && u.Cmp(knots[i+1]) < 0 {
			basis[i].SetInt64(1)
		}
	}
	for degree := 1; degree <= 3; degree++ {
		for i := 0; i < len(basis)-degree; i++ {
			v := new(big.Rat)
			if d := new(big.Rat).Sub(knots[i+degree], knots[i]); d.Sign() != 0 {
				v.Add(v, new(big.Rat).Mul(new(big.Rat).Quo(new(big.Rat).Sub(u, knots[i]), d), basis[i]))
			}
			if d := new(big.Rat).Sub(knots[i+degree+1], knots[i+1]); d.Sign() != 0 {
				v.Add(v, new(big.Rat).Mul(new(big.Rat).Quo(new(big.Rat).Sub(knots[i+degree+1], u), d), basis[i+1]))
			}
			basis[i] = v
		}
	}
	return basis[:23]
}

type bladeCircularCoordinates struct{ x, y, r0, z1 float64 }

func bladeRat(x float64) *big.Rat { return new(big.Rat).SetFloat64(x) }

func bladeRepresentable(x *big.Rat) (float64, bool) {
	f, _ := x.Float64()
	return f, !math.IsInf(f, 0) && bladeRat(f).Cmp(x) == 0
}

func bladeNeighbors(target *big.Rat, count int) []float64 {
	nearest, _ := target.Float64()
	values := []float64{nearest}
	lo, hi := nearest, nearest
	for i := 0; i < count; i++ {
		lo, hi = math.Nextafter(lo, math.Inf(-1)), math.Nextafter(hi, math.Inf(1))
		values = append(values, lo, hi)
	}
	slices.SortFunc(values, func(a, b float64) int {
		da := new(big.Rat).Abs(new(big.Rat).Sub(bladeRat(a), target))
		db := new(big.Rat).Abs(new(big.Rat).Sub(bladeRat(b), target))
		if c := da.Cmp(db); c != 0 {
			return c
		}
		return bladeRat(a).Cmp(bladeRat(b))
	})
	return values
}

func bladeCoordinates(radius, start, scale float64, seed int, pole [2]float64) (bladeCircularCoordinates, error) {
	r, s, j := bladeRat(radius), bladeRat(start), bladeRat(scale)
	target := new(big.Rat).Add(s, big.NewRat(3, 10))
	for _, z := range bladeNeighbors(target, 4) {
		c := new(big.Rat).Sub(bladeRat(z), s)
		if c.Sign() <= 0 || c.Cmp(r) >= 0 {
			continue
		}
		r0, ok := bladeRepresentable(new(big.Rat).Sub(r, c))
		if !ok {
			continue
		}
		var selected [2]float64
		valid := true
		factors := []*big.Rat{big.NewRat(15, 8), big.NewRat(7, 4)}
		if seed == 1 {
			factors = []*big.Rat{big.NewRat(53, 32), big.NewRat(61, 32)}
		}
		for i, factor := range factors {
			found := false
			target := new(big.Rat).Mul(j, factor)
			if seed == 2 {
				target = bladeRat(pole[i])
			}
			for _, v := range bladeNeighbors(target, 8) {
				vr := bladeRat(v)
				_, a := bladeRepresentable(new(big.Rat).Add(vr, r))
				_, b := bladeRepresentable(new(big.Rat).Sub(vr, r))
				_, d := bladeRepresentable(new(big.Rat).Add(vr, bladeRat(r0)))
				_, e := bladeRepresentable(new(big.Rat).Sub(vr, bladeRat(r0)))
				if a && b && d && e {
					selected[i], found = v, true
					break
				}
			}
			if !found {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		x, y := bladeRat(selected[0]), bladeRat(selected[1])
		distance := new(big.Rat).Add(new(big.Rat).Mul(x, x), new(big.Rat).Mul(y, y))
		if distance.Cmp(new(big.Rat).Mul(bladeRat(r0), bladeRat(r0))) < 0 {
			return bladeCircularCoordinates{selected[0], selected[1], r0, z}, nil
		}
	}
	return bladeCircularCoordinates{}, fmt.Errorf("turbinefan: blade: no representable circular coordinates for S=%g R=%g", start, radius)
}

type bladeSection struct{ z, chord, thickness, camber, ellipse float64 }

type bladeStation struct{ r, chord, ratio, camber, ellipse float64 }

func stationsFor(size Size) []bladeStation {
	stations := []bladeStation{{size.RootRadius, 8 * size.HubScale, 1, 0, 1}, {size.RootRadius + 1.9*size.HubScale, 8 * size.HubScale, 1, 0, 1}}
	q := []float64{19, 21.5, 25, 30, 38, 46, 55, 64}
	chord := []float64{11.5, 15.5, 17.8, 16.5, 14.5, 12.5, 10.5, 8}
	ratio := []float64{.6, .38, .26, .22, .19, .18, .17, .22}
	camber := []float64{.01, .025, .035, .04, .04, .04, .035, .03}
	ellipse := []float64{.6, .25, 0, 0, 0, 0, 0, 0}
	for i := range q {
		stations = append(stations, bladeStation{q[i] * size.Scale, chord[i] * size.Scale, ratio[i], camber[i], ellipse[i]})
	}
	return stations
}

func bladeSections(scale float64) []bladeSection {
	q := []float64{19, 21.5, 25, 30, 38, 46, 55, 64}
	chord := []float64{11.5, 15.5, 17.8, 16.5, 14.5, 12.5, 10.5, 8}
	thickness := []float64{.60, .38, .26, .22, .19, .18, .17, .22}
	camber := []float64{.010, .025, .035, .040, .040, .040, .035, .030}
	ellipse := []float64{.60, .25, 0, 0, 0, 0, 0, 0}
	sections := make([]bladeSection, 0, 9)
	for i := range q {
		sections = append(sections, bladeSection{scale * (26 + 38*(q[i]-19)/45), chord[i] * scale, thickness[i], camber[i], ellipse[i]})
		if i == 4 {
			at := 19 + 22.0*45/38
			t := (at - q[i]) / (q[i+1] - q[i])
			lerp := func(a, b float64) float64 { return a + t*(b-a) }
			sections = append(sections, bladeSection{48 * scale, scale * lerp(chord[i], chord[i+1]), lerp(thickness[i], thickness[i+1]), lerp(camber[i], camber[i+1]), lerp(ellipse[i], ellipse[i+1])})
		}
	}
	return sections
}

func bladeReferencePoints(section bladeSection, rootRadius, hubScale, scale, pitch float64, first bool) [21][2]float64 {
	var points [21][2]float64
	radius := 3 * hubScale
	beta := func(z float64) float64 {
		return 5*math.Pi/180 + math.Atan(math.Tan(20*math.Pi/180)*48*scale/math.Max(z, 21.5*scale)) + pitch - 25*math.Pi/180
	}
	yt := 4 * hubScale * math.Sin(beta(rootRadius))
	mean := func(x float64) float64 {
		const p = .4
		if x < p {
			return section.camber * (2*p*x - x*x) / (p * p)
		}
		return section.camber * (1 - 2*p + 2*p*x - x*x) / ((1 - p) * (1 - p))
	}
	for i := 0; i <= 20; i++ {
		var k int
		if i <= 10 {
			k = 10 - i
		} else {
			k = i - 10
		}
		x := (1 - math.Cos(float64(k)*math.Pi/10)) / 2
		if x == 1 {
			x = 1 - 1e-8
		}
		n := .2969*math.Sqrt(x) - .1260*x - .3516*x*x + .2843*x*x*x - .1036*x*x*x*x
		t := section.thickness * section.chord
		naca := (t/2-.15)*n/.10002 + .5*x
		oval := t*math.Sqrt(x*(1-x)) + .5*x
		half := (1-section.ellipse)*naca + section.ellipse*oval
		if i > 10 {
			half = -half
		}
		e := .4 + .1*section.ellipse
		a := section.chord * (e - x)
		b := section.chord*(mean(x)-mean(e)) + half
		ang := beta(section.z)
		points[i] = [2]float64{-a*math.Cos(ang) + b*math.Sin(ang), -a*math.Sin(ang) - b*math.Cos(ang)}
	}
	mx := (points[0][0] + points[20][0]) / 2
	my := (points[0][1] + points[20][1]) / 2
	for i := range points {
		points[i][0] += radius - mx
		points[i][1] += yt - my
	}
	for k := 1; k <= 10; k++ {
		x := (1 - math.Cos(float64(k)*math.Pi/10)) / 2
		if x <= .8 {
			continue
		}
		i, j := 10-k, 10+k
		m := [2]float64{(points[i][0] + points[j][0]) / 2, (points[i][1] + points[j][1]) / 2}
		d := [2]float64{(points[i][0] - points[j][0]) / 2, (points[i][1] - points[j][1]) / 2}
		u := (x - .8) / .2
		h := 3*u*u - 2*u*u*u
		d[0] *= 1 - h
		d[1] = (1-h)*d[1] - h
		points[i] = [2]float64{m[0] + d[0], m[1] + d[1]}
		points[j] = [2]float64{m[0] - d[0], m[1] - d[1]}
	}
	points[0], points[20] = [2]float64{radius, yt - 1}, [2]float64{radius, yt + 1}
	if first {
		for _, side := range [][3]int{{0, 7, 8}, {20, 16, 15}} {
			start, join, next := side[0], side[1], side[2]
			control := [2]float64{points[join][0] + 2*(points[join][0]-points[next][0]), points[join][1] + 2*(points[join][1]-points[next][1])}
			control[0] = math.Min(control[0], radius-1.2)
			step := 1
			if start > join {
				step = -1
			}
			for i := start + step; i != join; i += step {
				u := float64(i-start) / float64(join-start)
				for axis := 0; axis < 2; axis++ {
					points[i][axis] = (1-u)*(1-u)*points[start][axis] + 2*(1-u)*u*control[axis] + u*u*points[join][axis]
				}
			}
		}
		for i := 16; i <= 19; i++ {
			points[i][1] += .45 * math.Pow(math.Sin(math.Pi*float64(i-15)/5), 2)
		}
	}
	return points
}

func bladeLoft(ctx context.Context, doc *decad.Document, scale, hubScale, rootRadius, start, pitch float64, seed int, pole [2]float64, inspect func([]decad.LoftSection, decad.CircularLoftCorrespondence) error) (*decad.Body, []*sketch.Sketch, error) {
	radius := 3 * hubScale
	coordinates, err := bladeCoordinates(radius, start, hubScale, seed, pole)
	if err != nil {
		return nil, nil, err
	}
	type section struct {
		z, radius float64
		foil      *bladeSection
	}
	sections := []section{{start, coordinates.r0, nil}, {coordinates.z1, radius, nil}, {14 * hubScale, radius, nil}, {14.6 * hubScale, radius, nil}}
	for _, foil := range bladeSections(scale) {
		f := foil
		sections = append(sections, section{z: f.z, foil: &f})
	}
	var sketches []*sketch.Sketch
	var loft []decad.LoftSection
	for index, section := range sections {
		frame, err := r3.NewFrame(r3.Vec{X: coordinates.x, Y: coordinates.y, Z: section.z}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		if err != nil {
			return nil, nil, fmt.Errorf("turbinefan: blade z=%g frame: %w", section.z, err)
		}
		world := sketch.NewWorld()
		plane, err := world.CreatePlaneFromFrame(frame)
		if err != nil {
			return nil, nil, err
		}
		s, err := world.CreateSketch(plane)
		if err != nil {
			return nil, nil, err
		}
		if section.foil == nil {
			cx, cy := -coordinates.x, -coordinates.y
			center := sketchcheck.DimensionPoint(s, cx, cy)
			east := sketchcheck.DimensionPoint(s, cx+section.radius, cy)
			north := s.CreatePoint(cx, cy+section.radius)
			west := s.CreatePoint(cx-section.radius, cy)
			south := s.CreatePoint(cx, cy-section.radius)
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), north, cx), sketch.NewVerticalDistance(s.Origin(), west, cy), sketch.NewHorizontalDistance(s.Origin(), south, cx))
			other := s.CreatePoint(cx, cy)
			s.AddConstraint(sketch.NewHorizontalDistance(s.Origin(), other, cx))
			s.CreateArc(center, east, north)
			s.CreateArc(center, north, west)
			s.CreateArc(center, west, south)
			s.CreateArc(other, south, east)
		} else {
			reference := bladeReferencePoints(*section.foil, rootRadius, hubScale, scale, pitch, index == 4)
			controls, err := bladeSplineControls(reference)
			if err != nil {
				return nil, nil, fmt.Errorf("turbinefan: blade z=%g controls: %w", section.z, err)
			}
			points := make([]*sketch.Point, len(controls))
			for i, p := range controls {
				points[i] = sketchcheck.DimensionPoint(s, p[0]-coordinates.x, p[1]-coordinates.y)
			}
			if _, err := s.CreateSpline(points...); err != nil {
				return nil, nil, fmt.Errorf("turbinefan: blade z=%g spline: %w", section.z, err)
			}
			lo, hi := reference[0], reference[20]
			a := sketchcheck.DimensionPoint(s, radius-coordinates.x, (2*hi[1]+lo[1])/3-coordinates.y)
			b := sketchcheck.DimensionPoint(s, radius-coordinates.x, (hi[1]+2*lo[1])/3-coordinates.y)
			s.CreateLine(points[22], a)
			s.CreateLine(a, b)
			s.CreateLine(b, points[0])
		}
		if _, err := s.Solve(ctx); err != nil {
			return nil, nil, fmt.Errorf("turbinefan: blade z=%g solve: %w", section.z, err)
		}
		if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
			return nil, nil, fmt.Errorf("turbinefan: blade z=%g verify: %w", section.z, err)
		}
		var found *sketch.Profile
		for _, p := range s.Profiles() {
			if p.Valid && len(p.Holes) == 0 && len(p.Outer) == 4 {
				found = p
				break
			}
		}
		if found == nil {
			return nil, nil, fmt.Errorf("turbinefan: blade z=%g: no four-edge profile", section.z)
		}
		sketches = append(sketches, s)
		loft = append(loft, decad.LoftSection{Sketch: s, Profile: found})
	}
	var correspondence decad.CircularLoftCorrespondence
	for i := range correspondence.Numerators {
		correspondence.Numerators[i] = []int64{0, 12, 24, 41, 56, 80}
		correspondence.Denominators[i] = 80
	}
	correspondence.EmergingFlat = true
	correspondence.SmoothCircularBands = true
	if inspect != nil {
		if err := inspect(loft, correspondence); err != nil {
			return nil, nil, err
		}
	}
	body, err := decad.LoftSectionsWithCircularCorrespondence(ctx, doc, correspondence, loft...)
	if err != nil {
		return nil, nil, fmt.Errorf("turbinefan: blade loft: %w", err)
	}
	frame, err := r3.NewFrame(r3.Vec{Y: -(radius + 2), Z: 9.5 * hubScale}, r3.Vec{Z: 1}, r3.Vec{X: 1})
	if err != nil {
		return nil, nil, err
	}
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	circle := s.CreateCircle(s.Origin(), 1.3)
	s.AddConstraint(sketch.NewRadius(circle, 1.3))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("turbinefan: blade bolt: invalid profile")
	}
	tool, err := decad.Extrude(ctx, doc, s, profiles[0], decad.Distance{D: units.Millimeters(2 * (radius + 2)), Dir: decad.Along})
	if err != nil {
		return nil, nil, fmt.Errorf("turbinefan: blade bolt extrusion: %w", err)
	}
	body, err = decad.Cut(ctx, body, tool)
	if err != nil {
		return nil, nil, fmt.Errorf("turbinefan: blade bolt cut: %w", err)
	}
	return body, append(sketches, s), nil
}

const (
	designAttack          = 5 * math.Pi / 180
	refPitch              = 25 * math.Pi / 180
	DefaultBladePitch     = refPitch
	MinBladePitch         = 10 * math.Pi / 180
	MaxBladePitch         = 40 * math.Pi / 180
	BladeSocketFloorRound = .5
)

type Blade struct {
	Body     *decad.Body
	Sketches []*sketch.Sketch
}

func BuildBlade(ctx context.Context, doc *decad.Document, pitch float64, size Size) (*Blade, error) {
	return buildBlade(ctx, doc, pitch, size, nil)
}

func buildBlade(ctx context.Context, doc *decad.Document, pitch float64, size Size, finalInspect func([]decad.LoftSection, decad.CircularLoftCorrespondence) error) (*Blade, error) {
	if ctx == nil || doc == nil {
		return nil, fmt.Errorf("turbinefan: blade: nil context or document")
	}
	if pitch < MinBladePitch || pitch > MaxBladePitch || math.IsNaN(pitch) {
		return nil, fmt.Errorf("turbinefan: blade: pitch outside range")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !size.valid() {
		return nil, fmt.Errorf("turbinefan: invalid blade size")
	}
	var err error
	var last error
	var sections []decad.LoftSection
	var correspondence decad.CircularLoftCorrespondence
	inspect := func(s []decad.LoftSection, c decad.CircularLoftCorrespondence) error {
		sections, correspondence = s, c
		return nil
	}
	var pole [2]float64
	for seed := 0; seed < 2; seed++ {
		body, sketches, err := bladeLoft(ctx, doc, size.Scale, size.HubScale, size.RootRadius, size.BladeStudStart, pitch, seed, pole, inspect)
		if err == nil {
			if finalInspect != nil {
				if err := finalInspect(sections, correspondence); err != nil {
					return nil, err
				}
			}
			return &Blade{Body: body, Sketches: sketches}, nil
		}
		if !strings.Contains(err.Error(), "blade loft:") || !strings.Contains(err.Error(), "polar") {
			return nil, err
		}
		last = err
	}
	pole, err = bladePoleSeed(ctx, sections, correspondence)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: blade pole after %v: %w", last, err)
	}
	body, sketches, err := bladeLoft(ctx, doc, size.Scale, size.HubScale, size.RootRadius, size.BladeStudStart, pitch, 2, pole, inspect)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: blade candidate pole: %w", err)
	}
	if finalInspect != nil {
		if err := finalInspect(sections, correspondence); err != nil {
			return nil, err
		}
	}
	return &Blade{Body: body, Sketches: sketches}, nil
}

func BuildBladeMirrored(ctx context.Context, doc *decad.Document, pitch float64, size Size) (*Blade, error) {
	blade, err := BuildBlade(ctx, doc, pitch, size)
	if err != nil {
		return nil, err
	}
	frame, err := r3.NewFrame(r3.Vec{}, r3.Vec{Y: 1}, r3.Vec{Z: 1})
	if err != nil {
		return nil, err
	}
	reflection, err := r3.Reflection(frame)
	if err != nil {
		return nil, err
	}
	body, err := decad.Transform(ctx, blade.Body, reflection)
	if err != nil {
		return nil, fmt.Errorf("turbinefan: blade mirror: %w", err)
	}
	return &Blade{Body: body, Sketches: blade.Sketches}, nil
}

func BladeSocketTool(ctx context.Context, doc *decad.Document, frame r3.Frame, size Size) (*decad.Body, *sketch.Sketch, error) {
	if !size.valid() {
		return nil, nil, fmt.Errorf("turbinefan: invalid blade size")
	}
	world := sketch.NewWorld()
	plane, err := world.CreatePlaneFromFrame(frame)
	if err != nil {
		return nil, nil, err
	}
	s, err := world.CreateSketch(plane)
	if err != nil {
		return nil, nil, err
	}
	circle := s.CreateCircle(s.Origin(), size.BladeSocketRadius)
	s.AddConstraint(sketch.NewRadius(circle, size.BladeSocketRadius))
	if _, err := s.Solve(ctx); err != nil {
		return nil, nil, err
	}
	if err := sketchcheck.VerifyFullyConstrained(ctx, s); err != nil {
		return nil, nil, err
	}
	profiles := s.Profiles()
	if len(profiles) != 1 || !profiles[0].Valid {
		return nil, nil, fmt.Errorf("turbinefan: blade socket: invalid profile")
	}
	body, err := decad.Extrude(ctx, doc, s, profiles[0], decad.Distance{D: units.Millimeters(size.HubRadius + 1 - size.BladeSocketStart), Dir: decad.Along})
	if err != nil {
		return nil, nil, err
	}
	edges := decad.Edges(decad.OuterLoopOf(decad.CapStart(body))).Exactly(1)
	body, err = decad.Fillet(ctx, body, edges, units.Millimeters(BladeSocketFloorRound))
	if err != nil {
		return nil, nil, err
	}
	return body, s, nil
}

type bladePolePoint struct{ x, y float64 }

func (p bladePolePoint) plus(q bladePolePoint) bladePolePoint {
	return bladePolePoint{p.x + q.x, p.y + q.y}
}
func (p bladePolePoint) minus(q bladePolePoint) bladePolePoint {
	return bladePolePoint{p.x - q.x, p.y - q.y}
}
func (p bladePolePoint) times(s float64) bladePolePoint { return bladePolePoint{p.x * s, p.y * s} }
func (p bladePolePoint) dot(q bladePolePoint) float64   { return p.x*q.x + p.y*q.y }

type bladePoleSample struct{ p, d bladePolePoint }
type bladePoleCircle struct {
	center bladePolePoint
	radius float64
}
type bladePoleHalfPlane struct {
	normal     bladePolePoint
	offset     float64
	band, span int
}

const bladePoleSpanCount = 23 // Three trailing lines and twenty recorded cubic spans.

func bladePoleMap(controls []int64, denominator int64, v float64) (float64, float64) {
	var values, derivatives [6]float64
	degree := len(controls) - 1
	for i, x := range controls {
		values[i] = float64(x) / float64(denominator)
		if i > 0 {
			derivatives[i-1] = float64(degree) * float64(x-controls[i-1]) / float64(denominator)
		}
	}
	eval := func(p [6]float64, n int) float64 {
		for n > 0 {
			for i := 0; i < n; i++ {
				p[i] = (1-v)*p[i] + v*p[i+1]
			}
			n--
		}
		return p[0]
	}
	return eval(values, degree), eval(derivatives, degree-1)
}

// Evaluate the same chart as the Decad circular-to-spline correspondence.
// v is the normalized parameter within a whole quarter, not a circle angle.
func (c bladePoleCircle) at(quarter int, v, dv float64, m decad.CircularLoftCorrespondence) bladePoleSample {
	w, dw := bladePoleMap(m.Numerators[quarter], m.Denominators[quarter], v)
	den := 1 + w*w
	p := bladePolePoint{(1 - w*w) / den, 2 * w / den}
	d := bladePolePoint{-4 * w / (den * den), 2 * (1 - w*w) / (den * den)}
	for i := 0; i < quarter; i++ {
		p, d = bladePolePoint{-p.y, p.x}, bladePolePoint{-d.y, d.x}
	}
	return bladePoleSample{c.center.plus(p.times(c.radius)), d.times(c.radius * dw * dv)}
}

type bladePoleGrid [bladePoleSpanCount][9]bladePoleSample

// The physical coordinates and positive traversal come from the solved
// profile. In particular, a reversed spline has the reversed actual knot
// intervals, not a newly generated spline with reversed controls.
func bladePoleSection(s decad.LoftSection) (bladePoleGrid, *bladePoleCircle, [21]float64, error) {
	var grid bladePoleGrid
	var walked [21]float64
	if s.Sketch == nil || s.Profile == nil || len(s.Profile.Outer) != 4 {
		return grid, nil, walked, fmt.Errorf("blade pole: expected four recorded edges")
	}
	frame, err := s.Sketch.Plane().Frame()
	if err != nil {
		return grid, nil, walked, err
	}
	origin := bladePolePoint{frame.Origin().X, frame.Origin().Y}
	world := func(p *sketch.Point) bladePolePoint { return origin.plus(bladePolePoint{p.X(), p.Y()}) }
	edges := s.Profile.Outer
	if arc, ok := edges[0].Entity.(*sketch.Arc); ok {
		center := world(arc.Center)
		r := math.Hypot(arc.Start.X()-arc.Center.X(), arc.Start.Y()-arc.Center.Y())
		if !(r > 0) || math.IsInf(r, 0) {
			return grid, nil, walked, fmt.Errorf("blade pole: invalid circular radius")
		}
		return grid, &bladePoleCircle{center, r}, walked, nil
	}
	fit := -1
	var curve *sketch.Spline
	for i, e := range edges {
		if sp, ok := e.Entity.(*sketch.Spline); ok {
			fit, curve = i, sp
		}
	}
	if fit < 0 || len(curve.Control) != 23 {
		return grid, nil, walked, fmt.Errorf("blade pole: expected 23 recorded control points")
	}
	for j := 0; j < 3; j++ {
		e := edges[(fit+j+1)%4]
		l, ok := e.Entity.(*sketch.Line)
		if !ok {
			return grid, nil, walked, fmt.Errorf("blade pole: expected trailing line %d", j)
		}
		a, b := world(l.Start), world(l.End)
		if e.Reversed {
			a, b = b, a
		}
		d := b.minus(a)
		for k := 0; k <= 8; k++ {
			grid[j][k] = bladePoleSample{a.plus(d.times(float64(k) / 8)), d}
		}
	}
	ctrl := make([][2]float64, len(curve.Control))
	for i, p := range curve.Control {
		ctrl[i] = [2]float64{p.X(), p.Y()}
	}
	knots := geom.ClampedKnots(len(ctrl))[3:24]
	reverse := edges[fit].Reversed
	for i := range walked {
		walked[i] = knots[i]
		if reverse {
			walked[i] = 1 - knots[20-i]
		}
	}
	for i := 0; i < 20; i++ {
		a, b := knots[i], knots[i+1]
		if reverse {
			a, b = knots[20-i], knots[19-i]
		}
		for k := 0; k <= 8; k++ {
			u := float64(k) / 8
			t := (1-u)*a + u*b
			x, y, err := geom.EvalCubicBSpline(ctrl, t)
			if err != nil {
				return grid, nil, walked, err
			}
			dx, dy, err := geom.EvalCubicBSplineDeriv(ctrl, t)
			if err != nil {
				return grid, nil, walked, err
			}
			grid[3+i][k] = bladePoleSample{origin.plus(bladePolePoint{x, y}), bladePolePoint{dx, dy}.times(b - a)}
		}
	}
	return grid, nil, walked, nil
}

// bladePoleSeed proposes a physical XY origin. The caller must still quantize
// it and have Decad certify the exact, rebuilt section records.
func bladePoleSeed(ctx context.Context, sections []decad.LoftSection, m decad.CircularLoftCorrespondence) ([2]float64, error) {
	var zero [2]float64
	if !m.EmergingFlat || len(sections) < 3 {
		return zero, fmt.Errorf("blade pole: expected emerging-flat sections")
	}
	for i, row := range m.Numerators {
		if len(row) < 2 || len(row) > 6 || m.Denominators[i] <= 0 {
			return zero, fmt.Errorf("blade pole: invalid quarter map %d", i)
		}
	}
	grids := make([]bladePoleGrid, len(sections))
	var circles []*bladePoleCircle
	var knots [21]float64
	for i, s := range sections {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		g, c, w, err := bladePoleSection(s)
		if err != nil {
			return zero, err
		}
		grids[i] = g
		if c != nil {
			if len(circles) != i {
				return zero, fmt.Errorf("blade pole: circle after spline")
			}
			circles = append(circles, c)
		} else if i == len(circles) {
			knots = w
		} else if w != knots {
			return zero, fmt.Errorf("blade pole: spline knot walks differ")
		}
	}
	if len(circles) == 0 || len(circles) == len(sections) {
		return zero, fmt.Errorf("blade pole: missing circle or fitted section")
	}
	for i, c := range circles {
		for j := 0; j < 3; j++ {
			for k := 0; k <= 8; k++ {
				grids[i][j][k] = bladePoleSample{p: c.center.plus(bladePolePoint{c.radius, 0})}
			}
		}
		for j := 0; j < 20; j++ {
			q := j / 5
			lo, hi := knots[5*q], knots[5*q+5]
			a, b := (knots[j]-lo)/(hi-lo), (knots[j+1]-lo)/(hi-lo)
			for k := 0; k <= 8; k++ {
				u := float64(k) / 8
				grids[i][3+j][k] = c.at(q, (1-u)*a+u*b, b-a, m)
			}
		}
	}
	hs := [...]float64{0, .125, .25, .5, .75, .875, 1}
	planes := make([]bladePoleHalfPlane, 0, (len(grids)-1)*bladePoleSpanCount*9*len(hs))
	for band := 1; band < len(grids); band++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}
		for j, row := range grids[band-1] {
			for k, a := range row {
				b := grids[band][j][k]
				for _, h := range hs {
					p := a.p.times(1 - h).plus(b.p.times(h))
					d := a.d.times(1 - h).plus(b.d.times(h))
					length := math.Hypot(d.x, d.y)
					if length == 0 {
						continue
					}
					n := bladePolePoint{-d.y / length, d.x / length}
					planes = append(planes, bladePoleHalfPlane{n, n.dot(p), band - 1, j})
				}
			}
		}
	}
	clip := func(margin float64) ([]bladePolePoint, int, error) {
		r := circles[0].radius - margin
		if r <= 0 {
			return nil, -1, nil
		}
		poly, scratch := make([]bladePolePoint, 96, 128), make([]bladePolePoint, 0, 128)
		for i := range poly {
			a := 2 * math.Pi * float64(i) / 96
			poly[i] = circles[0].center.plus(bladePolePoint{r * math.Cos(a), r * math.Sin(a)})
		}
		for i, plane := range planes {
			if i%256 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, i, err
				}
			}
			last := poly[len(poly)-1]
			signed := func(p bladePolePoint) float64 { return plane.normal.dot(p) - plane.offset - margin }
			l := signed(last)
			for _, p := range poly {
				d := signed(p)
				if (l < 0 && d > 0) || (l > 0 && d < 0) {
					scratch = append(scratch, last.plus(p.minus(last).times(l/(l-d))))
				}
				if d >= 0 {
					scratch = append(scratch, p)
				}
				last, l = p, d
			}
			poly, scratch = scratch, poly[:0]
			if len(poly) == 0 {
				return nil, i, nil
			}
		}
		return poly, -1, nil
	}
	base, at, err := clip(0)
	if err != nil {
		return zero, err
	}
	if len(base) == 0 {
		return zero, fmt.Errorf("blade pole: sampled feasible region empty at band %d span %d", planes[at].band, planes[at].span)
	}
	lo, hi := 0., 1.
	for i := 0; i < 14; i++ {
		mid := (lo + hi) / 2
		poly, _, err := clip(mid)
		if err != nil {
			return zero, err
		}
		if len(poly) > 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	poly, _, err := clip(.9 * lo)
	if err != nil {
		return zero, err
	}
	if len(poly) == 0 {
		return zero, fmt.Errorf("blade pole: lost feasible polygon")
	}
	var sum bladePolePoint
	for _, p := range poly {
		sum = sum.plus(p)
	}
	seed := sum.times(1 / float64(len(poly)))
	return [2]float64{seed.x, seed.y}, nil
}
