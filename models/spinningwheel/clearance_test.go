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

func requireLinkageFrameClearances(t *testing.T, a *spinningwheel.Assembly, p spinningwheel.Params) {
	t.Helper()
	var fixed []spinningwheel.Part
	for _, part := range a.Parts {
		if part.Motion == "fixed" && (part.Group == "frame" || part.Group == "adjustment" || part.Group == "head-frame") {
			fixed = append(fixed, part)
		}
	}
	for i := range 72 {
		pose := p
		pose.WheelAngle = float64(i) * math.Pi / 36
		state, err := spinningwheel.Motion(pose)
		require.NoError(t, err)
		for _, part := range a.Parts {
			if part.Name != "wheel-crank" && part.Name != "crank-pin" && part.Name != "toe-pin" &&
				part.Name != "footman" && part.Name != "treadle" && !strings.HasPrefix(part.Name, "linkage-") {
				continue
			}
			var center, offset r3.Vec
			var angle float64
			switch part.Motion {
			case "wheel":
				center, angle = a.State.Wheel, pose.WheelAngle-p.WheelAngle
			case "treadle":
				center, angle = a.State.Heel, state.PedalAngle-a.State.PedalAngle
			case "footman":
				center, angle, offset = a.State.Toe, state.RodAngle-a.State.RodAngle, state.Toe.Sub(a.State.Toe)
			default:
				continue
			}
			xf, err := r3.RotationAround(center, r3.Vec{Y: -1}, units.Radians(angle))
			require.NoError(t, err)
			translation, err := r3.Translation(offset)
			require.NoError(t, err)
			xf, err = xf.Then(translation)
			require.NoError(t, err)
			body, err := decad.Copy(t.Context(), part.Body, decad.WithCopyPlacement(xf))
			require.NoError(t, err)
			box, err := body.Bounds()
			require.NoError(t, err)
			for _, obstacle := range fixed {
				other, err := obstacle.Body.Bounds()
				require.NoError(t, err)
				if boxesSeparate(box.Min, box.Max, other.Min, other.Max, box.Bound.Base()+other.Bound.Base()+1e-7) {
					continue
				}
				// These members have constant sections through Y. Check that fact before a section replaces the box screen.
				requireYPrism(t, body, part.Name)
				requireYPrism(t, obstacle.Body, obstacle.Name)
				yMin, yMax := math.Max(box.Min.Y, other.Min.Y), math.Min(box.Max.Y, other.Max.Y)
				require.Greater(t, yMax-yMin, 1e-6, "%s/%s needs an interior section", part.Name, obstacle.Name)
				plane, err := r3.NewFrame(r3.Vec{Y: (yMin + yMax) / 2}, r3.Vec{X: 1}, r3.Vec{Z: 1})
				require.NoError(t, err)
				movingSection, err := body.SectionOutline(t.Context(), plane, units.Millimeters(.025))
				require.NoError(t, err)
				fixedSection, err := obstacle.Body.SectionOutline(t.Context(), plane, units.Millimeters(.025))
				require.NoError(t, err)
				gap, err := movingSection.Clearance(fixedSection)
				require.NoError(t, err)
				require.False(t, gap.Overlaps, "%s overlaps %s at%d degrees", part.Name, obstacle.Name, i*5)
				require.Greater(t, gap.Lower().Base(), 1e-6, "%s must clear %s at%d degrees", part.Name, obstacle.Name, i*5)
			}
			require.NoError(t, decad.Remove(body))
		}
	}
}

// Side planes and cylinders run along Y; every transverse plane is an end cap.
// This excludes varying sections before using one plane to check the whole overlapping width.
func requireYPrism(t *testing.T, body *decad.Body, name string) {
	t.Helper()
	box, err := body.Bounds()
	require.NoError(t, err)
	for i, face := range body.Faces() {
		switch surface := face.Surface().(type) {
		case decad.Plane:
			n := surface.Frame.N()
			if math.Abs(n.Y) < 1e-10 {
				continue
			}
			require.InDelta(t, 1, math.Abs(n.Y), 1e-10, "%s face%d must be transverse", name, i)
			y := surface.Frame.Origin().Y
			require.Less(t, math.Min(math.Abs(y-box.Min.Y), math.Abs(y-box.Max.Y)), box.Bound.Base()+1e-6, "%s face%d must be an end cap", name, i)
		case decad.Cylinder:
			require.InDelta(t, 1, math.Abs(surface.Axis.Y), 1e-10, "%s face%d cylinder must run along Y", name, i)
		default:
			t.Fatalf("%s face%d has unsupported section-invariance kind%d", name, i, face.Surface().Kind())
		}
	}
}

// A cylinder enclosing the actual unrotated member bounds proves clearance
// continuously for a whole flyer turn, rather than just at animation frames.
func requireFlyerSweptClearances(t *testing.T, a *spinningwheel.Assembly) {
	t.Helper()
	sweptMembers := 0
	for _, part := range a.Parts {
		if part.Motion != "flyer" || (part.Group != "flyer" && part.Group != "whorl" && !strings.Contains(part.Name, "guide")) {
			continue
		}
		sweptMembers++
		body, err := decad.Copy(t.Context(), part.Body, decad.WithCopyRotation(a.State.Flyer, r3.Vec{Y: -1}, units.Radians(-a.State.FlyerAngle)))
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
			case "frame", "head-frame", "adjustment", "fastener", "brake-band", "spring", "brake", "knob", "thread-envelope":
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
	require.Equal(t, 23, sweptMembers, "crossbar, two arms, whorl, nine guides, nine stems and front stalk need swept checks")
}
