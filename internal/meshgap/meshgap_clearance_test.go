package meshgap_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/r3"
	"github.com/stretchr/testify/require"
)

func TestExceedsClearanceMatchesExceeds(t *testing.T) {
	pointMesh := func(x, bound float64) meshgap.Mesh {
		t.Helper()
		m, err := meshgap.New([]r3.Vec{{X: x}}, [][3]int{{0, 0, 0}}, bound)
		require.NoError(t, err)
		return m
	}
	multiTrianglePair := func(targetX float64) (meshgap.Mesh, meshgap.Mesh) {
		t.Helper()
		left := []r3.Vec{{}}
		right := []r3.Vec{{X: targetX}}
		for i := range 8 {
			x := 100 + 2*float64(i)
			left = append(left, r3.Vec{X: x, Y: 100, Z: 100})
			right = append(right, r3.Vec{X: x, Y: 110, Z: 100})
		}
		build := func(points []r3.Vec) meshgap.Mesh {
			vertices := points
			faces := make([][3]int, len(vertices))
			for i := range faces {
				faces[i] = [3]int{i, i, i}
			}
			m, err := meshgap.New(vertices, faces, 0)
			require.NoError(t, err)
			return m
		}
		return build(left), build(right)
	}
	compare := func(a, b meshgap.Mesh, clearance float64) bool {
		t.Helper()
		var first bool
		for i, pair := range [][2]meshgap.Mesh{{a, b}, {b, a}} {
			_, want, wantErr := pair[0].Exceeds(pair[1], clearance)
			got, gotErr := pair[0].ExceedsClearance(pair[1], clearance)
			require.Equal(t, want, got)
			if wantErr == nil {
				require.NoError(t, gotErr)
			} else {
				require.EqualError(t, gotErr, wantErr.Error())
			}
			if i == 0 {
				first = want
			}
		}
		return first
	}
	t.Run("touching", func(t *testing.T) { compare(pointMesh(0, 0), pointMesh(0, 0), 0) })
	t.Run("charged margin equality and adjacent floats", func(t *testing.T) {
		a := pointMesh(0, 0.125)
		clearance := 0.5
		limit := clearance + (0.125 + 0.25 + 1e-6)
		require.False(t, compare(a, pointMesh(limit, 0.25), clearance))
		require.False(t, compare(a, pointMesh(math.Nextafter(limit, math.Inf(-1)), 0.25), clearance))
		require.True(t, compare(a, pointMesh(math.Nextafter(limit, math.Inf(1)), 0.25), clearance))
	})
	t.Run("multi-triangle pruning and adjacent outcomes", func(t *testing.T) {
		limit := 1e-6
		clearA, clearB := multiTrianglePair(2e-6)
		require.True(t, compare(clearA, clearB, 0))
		equalA, equalB := multiTrianglePair(limit)
		require.False(t, compare(equalA, equalB, 0))
		belowA, belowB := multiTrianglePair(math.Nextafter(limit, math.Inf(-1)))
		require.False(t, compare(belowA, belowB, 0))
		aboveA, aboveB := multiTrianglePair(math.Nextafter(limit, math.Inf(1)))
		require.True(t, compare(aboveA, aboveB, 0))
	})
	t.Run("sliver and degenerate triangles", func(t *testing.T) {
		sliver, err := meshgap.New([]r3.Vec{{}, {X: 1}}, [][3]int{{0, 0, 1}}, 0)
		require.NoError(t, err)
		other, err := meshgap.New(
			[]r3.Vec{{X: 0.5, Z: 2}, {X: 3, Z: 2}, {X: 0.5, Y: 3, Z: 2}},
			[][3]int{{0, 1, 2}}, 0,
		)
		require.NoError(t, err)
		compare(sliver, other, 3)
		compare(pointMesh(4, 0), pointMesh(4, 0), 0)
	})
	t.Run("invalid and empty inputs", func(t *testing.T) {
		valid := pointMesh(0, 0)
		var empty meshgap.Mesh
		compare(valid, empty, 0)
		compare(valid, valid, -1)
		compare(valid, valid, math.NaN())
		compare(valid, valid, math.Inf(1))
		_, err := meshgap.New([]r3.Vec{{X: math.NaN()}}, [][3]int{{0, 0, 0}}, 0)
		require.ErrorContains(t, err, "nonfinite vertex")
		_, err = meshgap.New([]r3.Vec{{}}, [][3]int{{0, 0, 0}}, math.Inf(1))
		require.ErrorContains(t, err, "displacement bound")
	})
	t.Run("fast path cap boundaries and fallbacks", func(t *testing.T) {
		atCap := pointMesh(1<<16, 1<<16)
		origin := pointMesh(0, 0)
		compare(atCap, origin, 1<<16)
		compare(pointMesh((1<<16)+1, 0), origin, 0)
		compare(pointMesh(70000, 0), origin, 70000)
		compare(origin, pointMesh(1e200, 0), 0)
	})
}
