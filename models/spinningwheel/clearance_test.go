package spinningwheel_test

import (
	"math"
	"strings"
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
			section, err := obstacle.Body.SectionOutline(t.Context(), plane, units.Millimeters(.05))
			require.NoError(t, err)
			require.Greater(t, len(section.Loops), 0, "%s needs a real section where its box meets the belt", obstacle.Name)
			gap, err := outline.Clearance(section)
			require.NoError(t, err)
			require.False(t, gap.Overlaps, "%s crosses %s in the belt plane", band.Name, obstacle.Name)
			require.Greater(t, gap.Lower().Base(), .5, "%s must clear %s", band.Name, obstacle.Name)
		}
	}
	require.Equal(t, 4, checked, "both exact wraps and both straight spans must be checked")
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
		case "wheel", "treadle", "gimbal", "footman":
			moving = append(moving, namedShape{part, nativeCollisionShape(part.Body, meshes[part.Name])})
		case "fixed":
			if part.Group == "frame" || part.Group == "adjustment" || part.Group == "hinge" {
				fixed = append(fixed, namedShape{part, nativeCollisionShape(part.Body, meshes[part.Name])})
			}
		}
	}
	for i := range 72 {
		posed := make(map[string]collisionShape, len(moving))
		pose := p
		pose.WheelAngle = float64(i) * math.Pi / 36
		state, err := spinningwheel.Motion(pose)
		require.NoError(t, err)
		for _, item := range moving {
			xf, err := spinningwheel.PoseTransform(a.State, state, item.part.Motion, pose.WheelAngle-p.WheelAngle)
			require.NoError(t, err)
			shape := item.shape.placed(xf)
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
	for _, part := range a.Parts {
		if part.Motion != "flyer" || (part.Group != "flyer" && part.Group != "whorl" && part.Group != "guide" && !strings.Contains(part.Name, "guide")) {
			continue
		}
		sweptMembers++
		body, err := decad.Copy(t.Context(), part.Body, decad.WithCopyRotation(a.State.Flyer, r3.Vec{Y: -1}, units.Radians(-a.State.FlyerAngle-a.State.HeadTilt)))
		require.NoError(t, err)
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
			require.True(t, boxesSeparate(lo, hi, other.Min, other.Max, box.Bound.Base()+other.Bound.Base()+1e-7),
				"swept bounds cannot prove %s clear of %s", part.Name, obstacle.Name)
		}
		require.NoError(t, decad.Remove(body))
	}
	require.Equal(t, 31, sweptMembers, "crossbar, two arms, whorl, thirteen hooks, thirteen stems and front stalk need swept checks")
}
