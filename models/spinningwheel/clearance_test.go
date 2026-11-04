package spinningwheel_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/lestrrat-3d/decad"
	"github.com/lestrrat-3d/decad-gallery/models/spinningwheel"
	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
	"github.com/stretchr/testify/require"
)

// Disjoint conservative boxes prove separation of the complete solids.
// Overlapping boxes fail this screen; they are not reported as a proven collision.
func boxesSeparate(amin, amax, bmin, bmax r3.Vec, margin float64) bool {
	return amin.X-bmax.X > margin || bmin.X-amax.X > margin ||
		amin.Y-bmax.Y > margin || bmin.Y-amax.Y > margin ||
		amin.Z-bmax.Z > margin || bmin.Z-amax.Z > margin
}

func requireDriveBandFrameClearances(t *testing.T, a *spinningwheel.Assembly) {
	t.Helper()
	plane, err := r3.NewFrame(r3.Vec{}, r3.Vec{X: 1}, r3.Vec{Z: 1})
	require.NoError(t, err)
	checked := 0
	for _, band := range a.Parts {
		if band.Group != "drive-band" {
			continue
		}
		checked++
		box, err := band.Body.Bounds()
		require.NoError(t, err)
		var outline *decad.SectionOutline
		for _, obstacle := range a.Parts {
			if obstacle.Motion != "fixed" || obstacle.Group == "drive-band" {
				continue
			}
			other, err := obstacle.Body.Bounds()
			require.NoError(t, err)
			if boxesSeparate(box.Min, box.Max, other.Min, other.Max, .5+box.Bound.Base()+other.Bound.Base()) {
				continue
			}
			if outline == nil {
				read, err := band.Body.SectionOutline(t.Context(), plane, units.Millimeters(.05))
				require.NoError(t, err)
				outline = &read
			}
			if obstacle.Group == "spring" {
				// A brake spring is a faceted Coil body: its evaluator box bound is several millimetres and the
				// evaluator refuses a belt-plane section finer than its 0.0889 mm mesh bound. Do not coarsen the
				// section tolerance. Test the verified mesh's own tight box against the belt body instead, which
				// charges only the mesh bound. A spring the mesh box cannot separate is decided by the mesh boundary gap.
				if gap := meshBoxGap(t, box, obstacle.Body, .09); gap > 0 {
					t.Logf("%s clears %s by %.3f mm beyond both bounds (tight mesh box)", band.Name, obstacle.Name, gap)
					continue
				}
				// The spring overlaps the belt box, so compare the full verified meshes instead of a section.
				beltMesh := tessellateForSweep(t, band.Body, .05)
				springMesh := tessellateForSweep(t, obstacle.Body, .09)
				meshGap, clear := nativeCollisionShape(band.Body, beltMesh).meshSeparationBy(nativeCollisionShape(obstacle.Body, springMesh), .5)
				require.True(t, clear, "%s must clear %s by 0.5 mm; mesh boundary gap lower bound %g", band.Name, obstacle.Name, meshGap)
				t.Logf("%s clears %s: mesh boundary gap lower bound %.3f mm beyond both bounds", band.Name, obstacle.Name, meshGap)
				continue
			}
			section, err := obstacle.Body.SectionOutline(t.Context(), plane, units.Millimeters(.05))
			require.NoError(t, err, "section of %s (group %s) in the belt plane", obstacle.Name, obstacle.Group)
			if len(section.Loops) == 0 {
				// The belt is a strap of finite width, so a solid can meet its box without crossing the Y = 0 plane.
				// A tight verified-mesh box that clears the belt box proves it clear outright.
				if gap := meshBoxGap(t, box, obstacle.Body, .05); gap > 0 {
					t.Logf("%s clears %s by %.3f mm beyond both bounds (tight mesh box)", band.Name, obstacle.Name, gap)
					continue
				}
				// Otherwise the two overlap in Y. Compare the full verified meshes: their boundary gap must exceed
				// 0.5 mm after both mesh bounds, and neither solid may lie inside the other.
				beltMesh := tessellateForSweep(t, band.Body, .05)
				obstacleMesh := tessellateForSweep(t, obstacle.Body, .05)
				belt := nativeCollisionShape(band.Body, beltMesh)
				solid := nativeCollisionShape(obstacle.Body, obstacleMesh)
				meshGap, clear := belt.meshSeparationBy(solid, .5)
				require.True(t, clear, "%s must clear %s by 0.5 mm; mesh boundary gap lower bound %g", band.Name, obstacle.Name, meshGap)
				t.Logf("%s clears %s: mesh boundary gap lower bound %.3f mm beyond both bounds (no belt-plane section)", band.Name, obstacle.Name, meshGap)
				continue
			}
			gap, err := outline.Clearance(section)
			require.NoError(t, err)
			require.False(t, gap.Overlaps, "%s crosses %s in the belt plane", band.Name, obstacle.Name)
			require.Greater(t, gap.Lower().Base(), .5, "%s must clear %s", band.Name, obstacle.Name)
		}
	}
	require.Equal(t, 4, checked, "both exact wraps and both straight spans must be checked")
}

// connectorShapesByPose rebuilds the connector at each of the count wheel angles on four goroutines.
// Each pose builds in its own document and nothing is shared between goroutines, so the work is race free.
// Only four builds are live at once, which bounds peak memory.
func connectorShapesByPose(t *testing.T, p spinningwheel.Params, count int) []collisionShape {
	t.Helper()
	shapes := make([]collisionShape, count)
	errs := make([]error, count)
	next := make(chan int)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				shapes[i], errs[i] = connectorShapeAt(t.Context(), p, i)
			}
		}()
	}
	for i := range count {
		next <- i
	}
	close(next)
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "connector at %d degrees", i*5)
	}
	return shapes
}

func connectorShapeAt(ctx context.Context, p spinningwheel.Params, i int) (collisionShape, error) {
	pose := p
	pose.WheelAngle = float64(i) * math.Pi / 36
	connector, err := spinningwheel.BuildConnector(ctx, spinningwheel.NewDocument(), pose)
	if err != nil {
		return collisionShape{}, err
	}
	mesh, err := connector.Body.Tessellate(ctx, units.Millimeters(.01), decad.WithVerification(decad.VerifyAll))
	if err != nil {
		return collisionShape{}, err
	}
	if !mesh.BoundaryVerified() || !mesh.VolumeVerified() {
		return collisionShape{}, errors.New("connector mesh boundary or volume not verified")
	}
	return nativeCollisionShape(connector.Body, surfaceTriangles{vertices: mesh.Vertices(), triangles: mesh.Triangles(), bound: mesh.Bound().Base()}), nil
}

func requireLinkageFrameClearances(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params, meshes map[string]surfaceTriangles) {
	t.Helper()
	type namedShape struct {
		part  spinningwheel.Part
		shape collisionShape
	}
	var fixed, moving []namedShape
	type bandKey struct {
		name   string
		lo, hi float64
	}
	bands := map[bandKey]collisionShape{}
	for _, part := range a.Parts {
		switch part.Motion {
		case "wheel", "treadle", "gimbal", "footman", "compliant":
			moving = append(moving, namedShape{part, nativeCollisionShape(part.Body, meshes[part.Name])})
		case "fixed":
			if part.Group == "frame" || part.Group == "adjustment" || part.Group == "hinge" {
				fixed = append(fixed, namedShape{part, nativeCollisionShape(part.Body, meshes[part.Name])})
			}
		}
	}
	hasCompliant := false
	for _, item := range moving {
		hasCompliant = hasCompliant || item.part.Motion == "compliant"
	}
	var connectors []collisionShape
	if hasCompliant {
		connectors = connectorShapesByPose(t, p, 72)
	}
	for i := range 72 {
		posed := make(map[string]collisionShape, len(moving))
		pose := p
		pose.WheelAngle = float64(i) * math.Pi / 36
		state, err := spinningwheel.Motion(pose)
		require.NoError(t, err)
		for _, item := range moving {
			var shape collisionShape
			if item.part.Motion == "compliant" {
				shape = connectors[i]
			} else {
				xf, err := spinningwheel.PoseTransform(a.State, state, item.part.Motion, pose.WheelAngle-p.WheelAngle)
				require.NoError(t, err)
				shape = item.shape.placed(xf)
			}
			posed[item.part.Name] = shape
			for _, obstacle := range fixed {
				if shape.separates(obstacle.shape) {
					continue
				}
				// An L-shaped base has empty space inside its convex enclosure. Read its actual constant section in this Y band.
				lo := math.Max(shape.lo.Y-shape.bound, obstacle.shape.lo.Y-obstacle.shape.bound)
				hi := math.Min(shape.hi.Y+shape.bound, obstacle.shape.hi.Y+obstacle.shape.bound)
				key := bandKey{obstacle.part.Name, lo, hi}
				band, known := bands[key]
				if !known {
					if sliced, ok := nativeInvariantYBand(t, obstacle.part.Body, obstacle.shape, lo, hi); ok {
						band = sliced
						bands[key] = band
						known = true
					}
				}
				if known && shape.separates(band) {
					continue
				}
				// A convex enclosure fills a bearing hole. Validate the actual native through-bore before exempting its shaft.
				if nativeShaftPassage(obstacle.part, shape, p.Scale) {
					continue
				}
				gap, clear := shape.meshSeparation(obstacle.shape)
				if clear {
					continue
				}
				t.Fatalf("%s cannot be proved clear of %s at %d degrees; bounded mesh gap %g mm", item.part.Name, obstacle.part.Name, i*5, gap)
			}
		}
		for _, rod := range moving {
			if rod.part.Motion != "footman" {
				continue
			}
			for _, fork := range moving {
				if fork.part.Motion != "gimbal" || fork.part.Group != "gimbal" {
					continue
				}
				a, b := posed[rod.part.Name], posed[fork.part.Name]
				if a.separates(b) {
					continue
				}
				gap, clear := a.meshSeparation(b)
				require.True(t, clear, "%s must clear %s at %d degrees; mesh gap %g", rod.part.Name, fork.part.Name, i*5, gap)
			}
		}
	}
}

// A cylinder enclosing the actual unrotated member bounds proves clearance
// continuously for a whole flyer turn, rather than just at animation frames.
func requireFlyerSweptClearances(t *testing.T, a *spinningwheel.Assembly) {
	t.Helper()
	sweptMembers := 0
	sweepMeshes := map[string]surfaceTriangles{}
	for _, part := range a.Parts {
		if part.Motion != "flyer" || (part.Group != "flyer" && part.Group != "whorl" && part.Group != "guide" && !strings.Contains(part.Name, "guide")) {
			continue
		}
		sweptMembers++
		// The member sweeps a solid of revolution about the vertical axis through State.Flyer, whatever the head
		// tilt and flyer phase it is posed at. Rotating the posed body about that axis cannot change the swept
		// volume, so the posed body is used as built; re-posing it with Copy would break its exact bore axes.
		body := part.Body
		box, err := body.Bounds()
		require.NoError(t, err)
		radial := math.Hypot(math.Max(math.Abs(box.Min.X-a.State.Flyer.X), math.Abs(box.Max.X-a.State.Flyer.X)),
			math.Max(math.Abs(box.Min.Z-a.State.Flyer.Z), math.Abs(box.Max.Z-a.State.Flyer.Z))) + 2*box.Bound.Base()
		inner := math.Hypot(math.Max(0, math.Max(box.Min.X-a.State.Flyer.X, a.State.Flyer.X-box.Max.X)),
			math.Max(0, math.Max(box.Min.Z-a.State.Flyer.Z, a.State.Flyer.Z-box.Max.Z))) - 2*box.Bound.Base()
		lo := r3.Vec{X: a.State.Flyer.X - radial, Y: box.Min.Y, Z: a.State.Flyer.Z - radial}
		hi := r3.Vec{X: a.State.Flyer.X + radial, Y: box.Max.Y, Z: a.State.Flyer.Z + radial}
		for _, obstacle := range a.Parts {
			if obstacle.Motion == "bobbin" {
				other, err := obstacle.Body.Bounds()
				require.NoError(t, err)
				if box.Min.Y > other.Max.Y+1e-6 || other.Min.Y > box.Max.Y+1e-6 {
					continue
				}
				otherRadius := 0.0
				if obstacle.Name == "bobbin" {
					// Full-revolve line-segment sections attain their radial extrema on these exact circles.
					for _, face := range obstacle.Body.Faces() {
						for _, loop := range face.Loops() {
							for _, edge := range loop.Edges() {
								if c, ok := edge.Curve().(decad.Circle3); ok {
									otherRadius = math.Max(otherRadius, c.Radius.Base())
								}
							}
						}
					}
					require.Greater(t, otherRadius, 0.0, "bobbin must expose native radial circles")
				} else {
					otherRadius = math.Hypot(math.Max(math.Abs(other.Min.X-a.State.Flyer.X), math.Abs(other.Max.X-a.State.Flyer.X)),
						math.Max(math.Abs(other.Min.Z-a.State.Flyer.Z), math.Abs(other.Max.Z-a.State.Flyer.Z)))
				}
				require.Greater(t, inner-otherRadius, other.Bound.Base()+1e-6, "%s must clear independent %s through all relative angles", part.Name, obstacle.Name)
				continue
			}
			if obstacle.Motion != "fixed" {
				continue
			}
			switch obstacle.Group {
			case "frame", "head-frame", "adjustment", "fastener", "brake-band", "spring", "brake", "brake-guide", "hinge", "knob", "thread-envelope":
			default:
				continue
			}
			other, err := obstacle.Body.Bounds()
			require.NoError(t, err)
			if boxesSeparate(lo, hi, other.Min, other.Max, box.Bound.Base()+other.Bound.Base()+1e-7) {
				continue
			}
			// The swept box is a coarse screen: an L-shaped or tall obstacle overlaps it without touching the
			// member. Decide such pairs on the real meshes, per slab along the flyer axis, for every angle.
			for _, body := range []*decad.Body{part.Body, obstacle.Body} {
				name := part.Name
				if body == obstacle.Body {
					name = obstacle.Name
				}
				if _, ok := sweepMeshes[name]; !ok {
					sweepMeshes[name] = tessellateForSweep(t, body, .05)
				}
			}
			gap, why := sweptMeshClearance(t, sweepMeshes[part.Name], sweepMeshes[obstacle.Name], a.State.Flyer.X, a.State.Flyer.Z)
			require.Empty(t, why, "swept volume of %s cannot be proved clear of %s", part.Name, obstacle.Name)
			t.Logf("%s sweeps clear of %s: smallest radial slab gap %.3f mm beyond both mesh bounds", part.Name, obstacle.Name, gap)
		}
	}
	require.Equal(t, 17, sweptMembers, "crossbar, two arms, whorl and thirteen formed guides need swept checks")
}

// meshBoxGap returns how far the tight box of the body's verified mesh lies from the belt box beyond the 0.5 mm
// limit and both bounds. A value above zero proves separation; it is not positive otherwise.
func meshBoxGap(t *testing.T, belt decad.Box, body *decad.Body, chord float64) float64 {
	t.Helper()
	mesh := tessellateForSweep(t, body, chord)
	lo, hi := mesh.vertices[0], mesh.vertices[0]
	for _, v := range mesh.vertices {
		lo = r3.Vec{X: math.Min(lo.X, v.X), Y: math.Min(lo.Y, v.Y), Z: math.Min(lo.Z, v.Z)}
		hi = r3.Vec{X: math.Max(hi.X, v.X), Y: math.Max(hi.Y, v.Y), Z: math.Max(hi.Z, v.Z)}
	}
	need := .5 + belt.Bound.Base() + mesh.bound
	gap := math.Max(math.Max(lo.X-belt.Max.X, belt.Min.X-hi.X), math.Max(math.Max(lo.Y-belt.Max.Y, belt.Min.Y-hi.Y), math.Max(lo.Z-belt.Max.Z, belt.Min.Z-hi.Z)))
	return gap - need
}
