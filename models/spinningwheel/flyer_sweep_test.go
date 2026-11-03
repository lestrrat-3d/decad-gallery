package spinningwheel_test

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// radialSpan is a closed range of distances from the flyer axis.
type radialSpan struct{ lo, hi float64 }

// sweepSlab is the thickness along the flyer axis of one radial comparison.
const sweepSlab = 1.0

// tessellateForSweep returns a verified mesh of body for the exact swept-volume test.
func tessellateForSweep(t *testing.T, body *decad.Body, chord float64) surfaceTriangles {
	t.Helper()
	mesh, err := body.Tessellate(t.Context(), units.Millimeters(chord), decad.WithVerification(decad.VerifyAll))
	require.NoError(t, err)
	require.True(t, mesh.BoundaryVerified() && mesh.VolumeVerified())
	return surfaceTriangles{vertices: mesh.Vertices(), triangles: mesh.Triangles(), bound: mesh.Bound().Base()}
}

// clipSlab clips triangle t to the closed slab y0 <= Y <= y1 and returns the convex polygon.
func clipSlab(poly []r3.Vec, y0, y1 float64) []r3.Vec {
	clip := func(in []r3.Vec, keep func(r3.Vec) bool, at float64) []r3.Vec {
		var out []r3.Vec
		for i, p := range in {
			q := in[(i+1)%len(in)]
			if keep(p) {
				out = append(out, p)
			}
			if keep(p) != keep(q) {
				u := (at - p.Y) / (q.Y - p.Y)
				out = append(out, r3.Vec{X: p.X + u*(q.X-p.X), Y: at, Z: p.Z + u*(q.Z-p.Z)})
			}
		}
		return out
	}
	poly = clip(poly, func(p r3.Vec) bool { return p.Y >= y0 }, y0)
	if len(poly) == 0 {
		return nil
	}
	return clip(poly, func(p r3.Vec) bool { return p.Y <= y1 }, y1)
}

// polygonRadii returns the least and greatest distance from the axis through (cx, cz) to a convex polygon.
func polygonRadii(poly []r3.Vec, cx, cz float64) radialSpan {
	span := radialSpan{lo: math.Inf(1)}
	positive, negative, area := false, false, 0.0
	for i, p := range poly {
		q := poly[(i+1)%len(poly)]
		ax, az, bx, bz := p.X-cx, p.Z-cz, q.X-cx, q.Z-cz
		span.hi = math.Max(span.hi, math.Hypot(ax, az))
		dx, dz := bx-ax, bz-az
		length := dx*dx + dz*dz
		u := 0.0
		if length > 0 {
			u = math.Max(0, math.Min(1, -(ax*dx+az*dz)/length))
		}
		span.lo = math.Min(span.lo, math.Hypot(ax+u*dx, az+u*dz))
		cross := ax*bz - az*bx
		positive = positive || cross > 1e-12
		negative = negative || cross < -1e-12
		area += math.Abs(cross)
	}
	// A real polygon whose edges all turn the same way around the axis encloses it. A zero-area sliver cannot.
	if (!positive || !negative) && area > 1e-9 {
		span.lo = 0
	}
	return span
}

// slabSpans returns, per slab index, the merged radial spans that the mesh surface occupies around the axis.
func slabSpans(mesh surfaceTriangles, cx, cz float64, first, last int) map[int][]radialSpan {
	raw := map[int][]radialSpan{}
	for _, f := range mesh.triangles {
		tri := []r3.Vec{mesh.vertices[f[0]], mesh.vertices[f[1]], mesh.vertices[f[2]]}
		lo := math.Min(tri[0].Y, math.Min(tri[1].Y, tri[2].Y))
		hi := math.Max(tri[0].Y, math.Max(tri[1].Y, tri[2].Y))
		for k := max(first, int(math.Floor(lo/sweepSlab))); k <= min(last, int(math.Floor(hi/sweepSlab))); k++ {
			poly := clipSlab(tri, float64(k)*sweepSlab, float64(k+1)*sweepSlab)
			if len(poly) < 2 {
				continue
			}
			raw[k] = append(raw[k], polygonRadii(poly, cx, cz))
		}
	}
	merged := make(map[int][]radialSpan, len(raw))
	for k, spans := range raw {
		sort.Slice(spans, func(i, j int) bool { return spans[i].lo < spans[j].lo })
		out := []radialSpan{spans[0]}
		for _, s := range spans[1:] {
			if last := &out[len(out)-1]; s.lo <= last.hi {
				last.hi = math.Max(last.hi, s.hi)
				continue
			}
			out = append(out, s)
		}
		merged[k] = out
	}
	return merged
}

// meshHasPoint reports whether p lies inside the closed mesh.
func meshHasPoint(mesh surfaceTriangles, p r3.Vec) bool {
	return meshContains(collisionShape{points: mesh.vertices, triangles: mesh.triangles}, p)
}

// sweptMeshClearance proves that the solid of revolution swept by member about the Y axis through (cx, cz)
// stays clear of the fixed obstacle by more than both mesh bounds.
//
// Both solids are cut into 1 mm slabs along the axis. Within a slab a revolved point keeps its radius, so the
// radial spans of the member surface and the obstacle surface must be disjoint by the margin. Disjoint surface
// spans leave each span inside one connected region of the other solid, so one probe point per span then rules
// out a surface buried in the other solid. The result is a sound, non-empty-gap proof for every rotation angle.
// It returns the smallest radial gap found, and a description when the proof fails.
func sweptMeshClearance(t *testing.T, member, obstacle surfaceTriangles, cx, cz float64) (float64, string) {
	t.Helper()
	ylo, yhi := math.Inf(-1), math.Inf(1)
	for _, m := range []surfaceTriangles{member, obstacle} {
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, v := range m.vertices {
			lo, hi = math.Min(lo, v.Y), math.Max(hi, v.Y)
		}
		ylo, yhi = math.Max(ylo, lo), math.Min(yhi, hi)
	}
	margin := member.bound + obstacle.bound + 1e-7
	if ylo > yhi+margin {
		return math.Inf(1), ""
	}
	first, last := int(math.Floor((ylo-margin)/sweepSlab)), int(math.Floor((yhi+margin)/sweepSlab))
	memberSpans := slabSpans(member, cx, cz, first, last)
	obstacleSpans := slabSpans(obstacle, cx, cz, first, last)
	gap := math.Inf(1)
	for k, mine := range memberSpans {
		for _, theirs := range obstacleSpans[k] {
			for _, m := range mine {
				d := math.Max(m.lo-theirs.hi, theirs.lo-m.hi)
				gap = math.Min(gap, d)
				if d <= margin {
					return gap, fmt.Sprintf("slab Y %g: member radii %.3f..%.3f against obstacle radii %.3f..%.3f, gap %.4f, margin %.4f",
						float64(k)*sweepSlab, m.lo, m.hi, theirs.lo, theirs.hi, d, margin)
				}
			}
		}
	}
	for k, spans := range obstacleSpans {
		if len(memberSpans[k]) == 0 {
			continue
		}
		y := (float64(k) + .5) * sweepSlab
		for _, s := range spans {
			if meshHasPoint(member, r3.Vec{X: cx + (s.lo+s.hi)/2, Y: y, Z: cz}) {
				return gap, fmt.Sprintf("slab Y %g: obstacle surface at radii %.3f..%.3f lies inside the swept member", float64(k)*sweepSlab, s.lo, s.hi)
			}
		}
	}
	for k, spans := range memberSpans {
		if len(obstacleSpans[k]) == 0 {
			continue
		}
		y := (float64(k) + .5) * sweepSlab
		for _, s := range spans {
			if meshHasPoint(obstacle, r3.Vec{X: cx + (s.lo+s.hi)/2, Y: y, Z: cz}) {
				return gap, fmt.Sprintf("slab Y %g: member surface at radii %.3f..%.3f lies inside the obstacle", float64(k)*sweepSlab, s.lo, s.hi)
			}
		}
	}
	return gap, ""
}

// The swept-volume test must accept a ring that is radially apart and refuse one that touches or is buried.
func TestSpinningWheelSweptMeshClearance(t *testing.T) {
	ring := func(bore, outer, width, y float64) surfaceTriangles {
		f, err := r3.NewFrame(r3.Vec{Y: y}, r3.Vec{X: 1}, r3.Vec{Z: -1})
		require.NoError(t, err)
		m, err := ballbearing.Build(t.Context(), decad.New(), ballbearing.Params{BoreDiameter: bore, OuterDiameter: outer, Width: width, Placement: &f})
		require.NoError(t, err)
		return tessellateForSweep(t, m.Body, .05)
	}
	member := ring(8, 22, 7, 0)
	for _, c := range []struct {
		name                  string
		bore, outer, width, y float64
		clear                 bool
	}{
		{"radially_apart", 40, 50, 5, 1, true},
		{"axially_apart", 8, 22, 1, 20, true},
		{"crossing_wall", 16, 30, 5, 1, false},
		{"buried_in_wall", 14, 16, 2, 2, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, why := sweptMeshClearance(t, member, ring(c.bore, c.outer, c.width, c.y), 0, 0)
			require.Equal(t, c.clear, why == "", why)
		})
	}
}
