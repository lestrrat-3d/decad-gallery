// Package splitcheck checks that a printed shaft can be printed as two halves split lengthwise, as
// "Split shafts" in docs/model-pipeline.md sets out. The gallery does not model the split. The export to Fusion adds
// it, and this package only checks that a built shaft's shape allows it.
package splitcheck

import (
	"context"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

const (
	// MinHalfHeight is the least height in mm of each half above its split face: five 0.4 mm lines.
	MinHalfHeight = 2.0
	// MaxOverhang is the steepest a downward-facing surface of a half may lean from vertical, in degrees, with the
	// half's split face on the bed, unless it is a bridge.
	MaxOverhang = 45.0
	// MaxBridge is the widest in mm a connected downward face steeper than MaxOverhang may be, measured across its
	// narrowest direction in the bed's plane, for the printer to bridge it without support: the roof of a nut cavity
	// (5.02 mm) or of a bolt's half groove on the cut face.
	MaxBridge = 6.0

	// parallel is the tolerance on a unit-vector cross or dot product that counts two directions as parallel or
	// perpendicular. Mesh normals of planar faces are exact to rounding.
	parallel = 1e-6
	// margin is the arithmetic margin in mm added to every distance bound.
	margin = 1e-6
	// maxViolations caps the violations Check lists, so a failing report stays short.
	maxViolations = 8
)

// Shaft is a built shaft body and its axis.
type Shaft struct {
	// Body is the shaft as built, in any pose. It MUST NOT be nil.
	Body *decad.Body
	// Point is any point on the axis.
	Point r3.Vec
	// Axis is the axis direction. It need not be unit length, but MUST NOT be zero.
	Axis r3.Vec
	// Reference sets the split plane of a shaft with no flat: the plane holds Axis and Reference. A flatted shaft
	// ignores it, because the flats' normal sets the plane.
	Reference r3.Vec
	// Normal, when it is not zero, is the split plane's normal and overrides the flats and Reference. A part whose
	// faces do not name one plane, such as a pin with a hexagonal stub and a double flat parallel to its seam, names it
	// here. It MUST be normal to Axis. With Normal set, flats facing several ways are not a violation.
	Normal r3.Vec
	// Exempt lists the lines of the bolts that cross the seam to clamp the halves. A head pocket on one side and a nut
	// cavity on the other make the halves differ round each such line, so the symmetry test skips every sample within
	// Radius of one, and every sample whose mirror image lies there.
	Exempt []Line
}

// Line is a straight line through Point along Dir, with the radius Shaft.Exempt reads round it.
type Line struct {
	Point, Dir r3.Vec
	Radius     float64
}

// Report is what Check measured on one shaft. A shaft meets the rule when Violations is empty.
type Report struct {
	// Normal is the unit normal of the split plane, which holds the axis.
	Normal r3.Vec
	// Flats is the number of distinct planes of the planar faces parallel to the axis.
	Flats int
	// Asymmetry is the largest distance in mm from the mirror image of a boundary sample to the boundary.
	Asymmetry float64
	// AsymmetryLimit is the largest Asymmetry a symmetric body can read: twice the mesh bound plus a margin.
	AsymmetryLimit float64
	// Overhang is the steepest lean from vertical, in degrees, of a downward-facing surface of either half printed
	// with its split face on the bed. Zero means no surface faces downward.
	Overhang float64
	// Bridges is the number of connected downward faces steeper than MaxOverhang that are narrow enough to bridge,
	// counted per half, and Bridge is the widest of them in mm.
	Bridges int
	Bridge  float64
	// Exempted is the number of symmetry samples Shaft.Exempt skipped.
	Exempted int
	// HalfHeights is the height in mm of each half above its split face, on the +Normal side first.
	HalfHeights [2]float64
	// Violations lists each way the shaft breaks the rule, at most a few per kind.
	Violations []string
}

// Check tessellates s.Body with chord tolerance tol (mm) and checks it against the split shaft rule:
//
//   - the split plane holds the axis. Shaft.Normal sets it; otherwise every planar face parallel to the axis (a flat)
//     faces along one line, and the plane holds the axis and that line, so it cuts each flat in half;
//   - the body is mirror-symmetric about the split plane, apart from the samples Shaft.Exempt skips;
//   - with either half's split face on the bed, no surface faces downward steeper than MaxOverhang from vertical,
//     except a bridge no wider than MaxBridge;
//   - each half stands at least MinHalfHeight above its split face.
//
// It returns an error only when it cannot measure; a rule failure is a Violation.
func Check(ctx context.Context, s Shaft, tol float64) (Report, error) {
	var r Report
	if s.Body == nil {
		return r, fmt.Errorf("splitcheck: nil body")
	}
	axis, ok := s.Axis.Normalize()
	if !ok || !finiteVec(axis) {
		return r, fmt.Errorf("splitcheck: zero axis")
	}
	mesh, err := s.Body.Tessellate(ctx, units.Millimeters(tol), decad.WithVerification(decad.VerifyBoundary))
	if err != nil {
		return r, fmt.Errorf("splitcheck: tessellate: %w", err)
	}
	if !mesh.BoundaryVerified() {
		return r, fmt.Errorf("splitcheck: tessellation boundary not verified")
	}
	vertices, triangles, sources := mesh.Vertices(), mesh.Triangles(), mesh.SourceFaces()
	if len(sources) != len(triangles) {
		return r, fmt.Errorf("splitcheck: %d source faces for %d triangles", len(sources), len(triangles))
	}
	bound := mesh.Bound().Base()
	normals := make([]r3.Vec, len(triangles))
	for i, t := range triangles {
		normals[i] = unit(vertices[t[1]].Sub(vertices[t[0]]).Cross(vertices[t[2]].Sub(vertices[t[0]])))
	}

	if s.Normal != (r3.Vec{}) {
		n, ok := s.Normal.Normalize()
		if !ok || !finiteVec(n) || math.Abs(n.Dot(axis)) > parallel {
			return r, fmt.Errorf("splitcheck: Normal must be a direction normal to the axis")
		}
		r.Normal = n
		r.flats(s.Point, axis, vertices, triangles, sources, normals, false)
	} else {
		// The flats set the split plane. When they face along more than one line, the first sets it, so the report
		// still shows the other checks.
		flat := r.flats(s.Point, axis, vertices, triangles, sources, normals, true)
		if r.Flats == 0 {
			flat = s.Reference.Sub(axis.Scale(s.Reference.Dot(axis)))
			if flat.Len() < parallel {
				r.violate("the shaft has no flat and no Reference off its axis to set the split plane")
				return r, nil
			}
		}
		r.Normal = unit(axis.Cross(flat))
	}
	exempt := make([]Line, 0, len(s.Exempt))
	for _, l := range s.Exempt {
		d, ok := l.Dir.Normalize()
		if !ok || !finiteVec(d) || !(l.Radius > 0) {
			return r, fmt.Errorf("splitcheck: an exempt line needs a direction and a positive radius")
		}
		exempt = append(exempt, Line{Point: l.Point, Dir: d, Radius: l.Radius})
	}

	if err := r.symmetry(s.Point, vertices, triangles, bound, exempt); err != nil {
		return r, err
	}
	r.overhangs(s.Point, vertices, triangles, normals)
	r.heights(s.Point, vertices, bound)
	return r, nil
}

// flats finds the planar faces parallel to the axis and returns the first one's normal. It sets r.Flats to the number
// of distinct planes among those along that line. With strict set, flats facing along another line are violations.
func (r *Report) flats(point, axis r3.Vec, vertices []r3.Vec, triangles [][3]int, sources []*decad.Face,
	normals []r3.Vec, strict bool) r3.Vec {
	type plane struct {
		normal r3.Vec
		offset float64
	}
	var line r3.Vec
	var planes []plane
	var astray []r3.Vec // the other directions found, each reported once
	for i, t := range triangles {
		n := normals[i]
		if sources[i] == nil || sources[i].Surface() == nil || sources[i].Surface().Kind() != decad.KindPlane ||
			math.Abs(n.Dot(axis)) > parallel {
			continue
		}
		if planes == nil {
			line = n
		}
		if line.Cross(n).Len() > parallel {
			known := false
			for _, a := range astray {
				if a.Cross(n).Len() <= parallel {
					known = true
					break
				}
			}
			if !known {
				astray = append(astray, n)
			}
			continue
		}
		offset := vertices[t[0]].Sub(point).Dot(n)
		seen := false
		for _, p := range planes {
			if p.normal.Dot(n) > 0 && math.Abs(p.offset-offset) < 1e-4 {
				seen = true
				break
			}
		}
		if !seen {
			planes = append(planes, plane{n, offset})
		}
	}
	for i, a := range astray {
		if !strict {
			break
		}
		if i == 3 {
			r.violate(fmt.Sprintf("%d flat directions in all are not parallel to flat normal %v", len(astray), round(line)))
			break
		}
		r.violate(fmt.Sprintf("flat normal %v is not parallel to flat normal %v", round(a), round(line)))
	}
	r.Flats = len(planes)
	return line
}

// symmetry mirrors every vertex and triangle centroid across the split plane and measures how far the image lies
// from the boundary. A vertex lies on the held boundary and a centroid within bound of it; the image of either lies
// within bound of the true boundary when the body is symmetric, and the mesh within bound of that.
func (r *Report) symmetry(point r3.Vec, vertices []r3.Vec, triangles [][3]int, bound float64, exempt []Line) error {
	m, err := meshgap.New(vertices, triangles, bound)
	if err != nil {
		return fmt.Errorf("splitcheck: %w", err)
	}
	r.AsymmetryLimit = 2*bound + margin
	samples := append([]r3.Vec(nil), vertices...)
	for _, t := range triangles {
		samples = append(samples, vertices[t[0]].Add(vertices[t[1]]).Add(vertices[t[2]]).Scale(1.0/3))
	}
	asymmetric := 0
	near := func(q r3.Vec) bool {
		for _, l := range exempt {
			rel := q.Sub(l.Point)
			if rel.Sub(l.Dir.Scale(rel.Dot(l.Dir))).Len() <= l.Radius {
				return true
			}
		}
		return false
	}
	for _, p := range samples {
		image := p.Sub(r.Normal.Scale(2 * p.Sub(point).Dot(r.Normal)))
		if near(p) || near(image) {
			r.Exempted++
			continue
		}
		d, err := m.PointDistance(image)
		if err != nil {
			return fmt.Errorf("splitcheck: %w", err)
		}
		r.Asymmetry = math.Max(r.Asymmetry, d)
		if d <= r.AsymmetryLimit {
			continue
		}
		asymmetric++
		if asymmetric <= 3 {
			r.violate(fmt.Sprintf("the mirror image of %v lies %.4f mm off the boundary (limit %.4f)",
				round(p), d, r.AsymmetryLimit))
		}
	}
	if asymmetric > 3 {
		r.violate(fmt.Sprintf("%d boundary samples in all have no mirror image on the boundary", asymmetric))
	}
	return nil
}

// overhangs checks each triangle against the half or halves it lies in. The half on the +Normal side prints with
// +Normal up, the other with −Normal up. A surface faces downward too steeply when its outward normal points below
// horizontal by more than MaxOverhang, its lean from vertical. Such triangles that share vertices form one downward
// face; the face is a bridge when its narrowest width in the bed's plane is at most MaxBridge.
func (r *Report) overhangs(point r3.Vec, vertices []r3.Vec, triangles [][3]int, normals []r3.Vec) {
	limit := math.Sin(MaxOverhang * math.Pi / 180)
	wide := 0
	for _, up := range []float64{1, -1} {
		var steep []int
		for i, t := range triangles {
			lo, hi := math.Inf(1), math.Inf(-1)
			for _, v := range t {
				d := vertices[v].Sub(point).Dot(r.Normal)
				lo, hi = math.Min(lo, d), math.Max(hi, d)
			}
			if (up > 0 && hi <= margin) || (up < 0 && lo >= -margin) {
				continue
			}
			down := -up * normals[i].Dot(r.Normal)
			if down <= 0 {
				continue
			}
			r.Overhang = math.Max(r.Overhang, math.Asin(math.Min(1, down))*180/math.Pi)
			if down > limit+parallel {
				steep = append(steep, i)
			}
		}
		for _, face := range downwardFaces(vertices, triangles, steep) {
			span, at := r.span(vertices, triangles, face)
			if span <= MaxBridge+margin {
				r.Bridges++
				r.Bridge = math.Max(r.Bridge, span)
				continue
			}
			wide++
			if wide <= 3 {
				r.violate(fmt.Sprintf("a downward face near %v is %.2f mm across, wider than a %.0f mm bridge",
					round(at), span, MaxBridge))
			}
		}
	}
	if wide > 3 {
		r.violate(fmt.Sprintf("%d downward faces in all are wider than a %.0f mm bridge", wide, MaxBridge))
	}
}

// downwardFaces groups the triangles steep into faces of triangles that share a vertex position.
func downwardFaces(vertices []r3.Vec, triangles [][3]int, steep []int) [][]int {
	parent := make([]int, len(steep))
	for i := range parent {
		parent[i] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	type key [3]int64
	q := func(v r3.Vec) key {
		return key{int64(math.Round(v.X * 1e6)), int64(math.Round(v.Y * 1e6)), int64(math.Round(v.Z * 1e6))}
	}
	owner := map[key]int{}
	for i, t := range steep {
		for _, v := range triangles[t] {
			k := q(vertices[v])
			if j, ok := owner[k]; ok {
				parent[find(i)] = find(j)
				continue
			}
			owner[k] = i
		}
	}
	groups := map[int][]int{}
	for i, t := range steep {
		root := find(i)
		groups[root] = append(groups[root], t)
	}
	out := make([][]int, 0, len(groups))
	for _, g := range groups {
		out = append(out, g)
	}
	return out
}

// span returns a face's narrowest width in the bed's plane, read over directions 1° apart, and its first vertex.
func (r *Report) span(vertices []r3.Vec, triangles [][3]int, face []int) (float64, r3.Vec) {
	// Two directions in the split plane: any unit vector normal to Normal, and Normal × it.
	e1 := unit(r.Normal.Cross(r3.Vec{X: 1}))
	if e1.Len() == 0 || math.Abs(r.Normal.X) > 0.9 {
		e1 = unit(r.Normal.Cross(r3.Vec{Y: 1}))
	}
	e2 := r.Normal.Cross(e1)
	best := math.Inf(1)
	for deg := 0; deg < 180; deg++ {
		a := float64(deg) * math.Pi / 180
		d := e1.Scale(math.Cos(a)).Add(e2.Scale(math.Sin(a)))
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, t := range face {
			for _, v := range triangles[t] {
				x := vertices[v].Dot(d)
				lo, hi = math.Min(lo, x), math.Max(hi, x)
			}
		}
		best = math.Min(best, hi-lo)
	}
	return best, vertices[triangles[face[0]][0]]
}

// heights measures how far each half stands above its split face. The vertices lie on the held boundary, so the
// height reads at most bound low.
func (r *Report) heights(point r3.Vec, vertices []r3.Vec, bound float64) {
	for _, v := range vertices {
		d := v.Sub(point).Dot(r.Normal)
		r.HalfHeights[0] = math.Max(r.HalfHeights[0], d)
		r.HalfHeights[1] = math.Max(r.HalfHeights[1], -d)
	}
	for i, h := range r.HalfHeights {
		if h+bound+margin < MinHalfHeight {
			r.violate(fmt.Sprintf("half %d stands %.3f mm above its split face (minimum %.1f)", i+1, h, MinHalfHeight))
		}
	}
}

func (r *Report) violate(v string) {
	if len(r.Violations) < maxViolations {
		r.Violations = append(r.Violations, v)
	}
}

// round keeps reported coordinates short.
func round(v r3.Vec) r3.Vec {
	f := func(x float64) float64 { return math.Round(x*1000) / 1000 }
	return r3.Vec{X: f(v.X), Y: f(v.Y), Z: f(v.Z)}
}

// unit returns v scaled to unit length, or the zero vector when v has no direction.
func unit(v r3.Vec) r3.Vec {
	u, ok := v.Normalize()
	if !ok {
		return r3.Vec{}
	}
	return u
}

func finiteVec(v r3.Vec) bool {
	return !math.IsNaN(v.X+v.Y+v.Z) && !math.IsInf(v.X+v.Y+v.Z, 0)
}
