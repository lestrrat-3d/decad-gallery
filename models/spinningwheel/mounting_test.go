package spinningwheel_test

import (
	"math"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/internal/meshgap"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

func meshComponents(vertices int, triangles [][3]int) int {
	parent := make([]int, vertices)
	for i := range parent {
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	for _, f := range triangles {
		parent[root(f[1])] = root(f[0])
		parent[root(f[2])] = root(f[0])
	}
	components := make(map[int]struct{})
	for _, f := range triangles {
		components[root(f[0])] = struct{}{}
	}
	return len(components)
}

func meshEulerCharacteristic(triangles [][3]int) int {
	vertices := make(map[int]struct{})
	edges := make(map[[2]int]struct{})
	for _, face := range triangles {
		for i, vertex := range face {
			vertices[vertex] = struct{}{}
			other := face[(i+1)%3]
			edges[[2]int{min(vertex, other), max(vertex, other)}] = struct{}{}
		}
	}
	return len(vertices) - len(edges) + len(triangles)
}

// A real hollow bearing is connected; combining two real, separated meshes must not hide a loose component.
func TestSpinningWheelMeshComponents(t *testing.T) {
	var combinedVertices []r3.Vec
	var combinedTriangles [][3]int
	for _, x := range []float64{0, 40} {
		p := ballbearing.DefaultParams()
		placement, err := r3.NewFrame(r3.Vec{X: x}, r3.Vec{X: 1}, r3.Vec{Y: 1})
		require.NoError(t, err)
		p.Placement = &placement
		model, err := ballbearing.Build(t.Context(), decad.New(), p)
		require.NoError(t, err)
		mesh, err := model.Body.Tessellate(t.Context(), units.Millimeters(.12), decad.WithVerification(decad.VerifyAll))
		require.NoError(t, err)
		vertices, triangles := mesh.Vertices(), mesh.Triangles()
		require.Equal(t, 1, meshComponents(len(vertices), triangles))
		require.Zero(t, meshEulerCharacteristic(triangles), "a closed bearing has one through-hole")
		require.Equal(t, -1, meshEulerCharacteristic(triangles[1:]), "removing a face must change the topology check")
		offset := len(combinedVertices)
		for _, f := range triangles {
			combinedTriangles = append(combinedTriangles, [3]int{f[0] + offset, f[1] + offset, f[2] + offset})
		}
		combinedVertices = append(combinedVertices, vertices...)
	}
	require.Equal(t, 2, meshComponents(len(combinedVertices), combinedTriangles))
}

func requirePartHomes(t *testing.T, a *spinningwheel.Assembly) {
	t.Helper()
	parts := make(map[string]spinningwheel.Part, len(a.Parts))
	for _, part := range a.Parts {
		parts[part.Name] = part
	}
	mounts := make(map[string]spinningwheel.Mount, len(a.Mounts))
	grounded := make(map[string]struct{})
	for _, mount := range a.Mounts {
		_, exists := parts[mount.Part]
		require.True(t, exists, "mount names an absent part %s", mount.Part)
		_, duplicate := mounts[mount.Part]
		require.False(t, duplicate, "duplicate mount %s", mount.Part)
		mounts[mount.Part] = mount
		require.NotEmpty(t, mount.Kind, "%s needs a physical mount type", mount.Part)
		require.True(t, !math.IsNaN(mount.Clearance) && !math.IsInf(mount.Clearance, 0) && mount.Clearance >= 0,
			"%s clearance must be a finite nonnegative length", mount.Part)
		if mount.Kind == "ground" {
			require.Contains(t, []string{"head-leg", "front-wheel-leg", "rear-wheel-leg"}, mount.Part)
			require.Empty(t, mount.Supports, "%s is a ground mount", mount.Part)
			box, err := parts[mount.Part].Body.Bounds()
			require.NoError(t, err)
			require.InDelta(t, 0, box.Min.Z, box.Bound.Base()+1e-6, "%s must meet the floor", mount.Part)
			grounded[mount.Part] = struct{}{}
			continue
		}
		require.Greater(t, len(mount.Supports), 0, "%s has no support", mount.Part)
		for _, name := range append(append([]string{}, mount.Supports...), mount.Fasteners...) {
			_, present := parts[name]
			require.True(t, present, "%s references absent support/fastener %s", mount.Part, name)
			require.NotEqual(t, mount.Part, name, "%s cannot mount itself", mount.Part)
		}
	}
	require.Len(t, grounded, 3, "all three legs must meet the floor independently")
	for name := range parts {
		_, present := mounts[name]
		require.True(t, present, "part %s has no declared home", name)
	}
	// A cycle of loose components does not acquire a home by naming one another.
	for changed := true; changed; {
		changed = false
		for name, mount := range mounts {
			if _, ok := grounded[name]; ok {
				continue
			}
			for _, support := range mount.Supports {
				if _, ok := grounded[support]; ok {
					grounded[name] = struct{}{}
					changed = true
					break
				}
			}
		}
	}
	for name := range parts {
		_, ok := grounded[name]
		require.True(t, ok, "%s belongs to an unsupported group", name)
	}
}

func requireGeometricPartHomes(t *testing.T, a *spinningwheel.Assembly, meshes map[string]surfaceTriangles) {
	t.Helper()
	geometry := make(map[string]meshgap.Mesh, len(a.Parts))
	for _, part := range a.Parts {
		mesh, present := meshes[part.Name]
		require.True(t, present, "missing verified mesh for %s", part.Name)
		prepared, err := meshgap.New(mesh.vertices, mesh.triangles, mesh.bound)
		require.NoError(t, err, "prepare mount geometry for %s", part.Name)
		geometry[part.Name] = prepared
	}
	mounts := make([]meshgap.Mount, 0, len(a.Mounts))
	for _, mount := range a.Mounts {
		references := append([]string(nil), mount.Supports...)
		references = append(references, mount.Fasteners...)
		mounts = append(mounts, meshgap.Mount{
			Part: mount.Part, Ground: mount.Kind == "ground",
			References: references, Clearance: mount.Clearance,
		})
	}
	reached, nearest, err := meshgap.Reachable(mounts, geometry)
	require.NoError(t, err)
	unsupported := 0
	for _, part := range a.Parts {
		if _, present := reached[part.Name]; present {
			continue
		}
		if unsupported < 10 {
			closest := nearest[part.Name]
			t.Logf("%s has no geometric support path; nearest declared candidate %s has lower gap %g mm",
				part.Name, closest.Part, closest.LowerGap)
		}
		unsupported++
	}
	// Nearby surfaces alone do not prove retention. The declared joint types and
	// actual material probes are checked separately, as is every part's overlap.
	require.Zero(t, unsupported, "parts must have a support path after rejecting provably distant connections")
}

func requireNoMaterialOverlap(t *testing.T, doc *decad.Document, a *spinningwheel.Assembly) {
	t.Helper()
	names := make(map[*decad.Body]string, len(a.Parts))
	for _, part := range a.Parts {
		names[part.Body] = part.Name
	}
	report, err := doc.Verify(t.Context())
	require.NoError(t, err)
	for i, pair := range report.Interferences {
		if i == 10 {
			break
		}
		t.Logf("overlap %s / %s: volume %g +/- %g mm^3",
			names[pair.A], names[pair.B], pair.Volume.Value.Base(), pair.Volume.Bound.Base())
	}
	undecided := 0
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Pair == nil {
			continue
		}
		if undecided < 10 {
			t.Logf("pair %s / %s: %s (%s)", names[diagnostic.Pair.A], names[diagnostic.Pair.B], diagnostic.Code, diagnostic.Status)
		}
		undecided++
	}
	// Every solid's boundary and volume are checked separately with VerifyAll. Body measurement
	// bounds may honestly exceed Verify's default mass tolerance; pair uncertainty must not be ignored.
	require.Zero(t, len(report.Interferences), "finished physical parts must not share material")
	require.Zero(t, undecided, "all final part pairs need conclusive non-overlap results")
}
