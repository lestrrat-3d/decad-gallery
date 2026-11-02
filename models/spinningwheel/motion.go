package spinningwheel

import (
	"fmt"
	"math"

	"github.com/lestrrat-3d/r3"
	"github.com/lestrrat-3d/units"
)

// Params sets the wheel pose, head alignment, brake travel and illustrative yarn feed.
type Params struct {
	Scale           float64
	WheelAngle      float64
	DriveTension    float64
	BrakeTravel     float64
	YarnFeedPerTurn float64
	Ratio           int
}

// State records joint positions, orthonormal linkage frames and accumulated phases.
type State struct {
	Wheel        r3.Vec
	Flyer        r3.Vec
	Heel         r3.Vec
	CrankPin     r3.Vec
	Toe          r3.Vec
	RodHead      r3.Vec
	HeadPivot    r3.Vec
	PedalAngle   float64
	RodAngle     float64
	FlyerAngle   float64
	BobbinAngle  float64
	HeadTilt     float64
	WheelPitch   float64
	FlyerPitch   float64
	BeltLength   float64
	BrakeLength  float64
	SpringLength float64
	HeadShift    float64
	PedalFrame   r3.Frame
	RodFrame     r3.Frame
	GimbalFrame  r3.Frame
}

// DefaultParams returns the nominal scale, ratio and pose.
func DefaultParams() Params {
	return Params{
		Scale: 1, WheelAngle: math.Pi / 3, DriveTension: 6,
		BrakeTravel: 3, YarnFeedPerTurn: 100, Ratio: 7,
	}
}

// Validate rejects nonfinite values, unsupported ratios and dimensions outside their ranges.
func (p Params) Validate() error {
	if !finiteMotionValue(p.Scale) || p.Scale < 0.8 || p.Scale > 1.2 {
		return fmt.Errorf("scale must be finite and within [0.8, 1.2]")
	}
	if !finiteMotionValue(p.WheelAngle) {
		return fmt.Errorf("wheel angle must be finite")
	}
	if !finiteMotionValue(p.DriveTension) || p.DriveTension < 0 || p.DriveTension > 12 {
		return fmt.Errorf("drive tension must be finite and within [0, 12]")
	}
	if !finiteMotionValue(p.BrakeTravel) || p.BrakeTravel < 0 || p.BrakeTravel > 6 {
		return fmt.Errorf("brake travel must be finite and within [0, 6]")
	}
	if !finiteMotionValue(p.YarnFeedPerTurn) || p.YarnFeedPerTurn < 0 || p.YarnFeedPerTurn > 200 {
		return fmt.Errorf("yarn feed per turn must be finite and within [0, 200]")
	}
	switch p.Ratio {
	case 7, 9, 12, 17:
	default:
		return fmt.Errorf("ratio must be one of 7, 9, 12 or 17")
	}
	if !finiteMotionValue(float64(p.Ratio) * p.WheelAngle) {
		return fmt.Errorf("flyer phase overflows for ratio %d and wheel angle %g", p.Ratio, p.WheelAngle)
	}
	return nil
}

func finiteMotionValue(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

type linkagePose struct {
	pedal                    r3.Frame
	toe, head, radial, cross r3.Vec
	residual                 float64
}

func linkageAt(scale float64, crank r3.Vec, angle float64) (linkagePose, error) {
	pedal, err := r3.NewFrame(
		r3.Vec{Y: -190 * scale, Z: 60 * scale},
		r3.Vec{X: 1}, r3.Vec{Y: math.Cos(angle), Z: math.Sin(angle)},
	)
	if err != nil {
		return linkagePose{}, fmt.Errorf("create pedal frame: %w", err)
	}
	toe := pedal.ToWorld(r3.Vec{X: 220 * scale, Y: 450 * scale, Z: 20 * scale})
	radial, ok := (r3.Vec{X: toe.X - crank.X, Z: toe.Z - crank.Z}).Normalize()
	if !ok {
		return linkagePose{}, fmt.Errorf("gimbal radial direction has no finite length")
	}
	head := crank.Add(radial.Scale(32 * scale))
	residual := toe.Sub(head).Len() - 486*scale
	if !finiteMotionValue(residual) {
		return linkagePose{}, fmt.Errorf("linkage closure residual is not finite")
	}
	return linkagePose{
		pedal: pedal, toe: toe, head: head, radial: radial,
		cross: r3.Vec{X: -radial.Z, Z: radial.X}, residual: residual,
	}, nil
}

func solveLinkage(scale float64, crank r3.Vec) (float64, linkagePose, error) {
	lo, hi := -0.12, 0.20
	lower, err := linkageAt(scale, crank, lo)
	if err != nil {
		return 0, linkagePose{}, err
	}
	upper, err := linkageAt(scale, crank, hi)
	if err != nil {
		return 0, linkagePose{}, err
	}
	if lower.residual == 0 || upper.residual == 0 || (lower.residual > 0) == (upper.residual > 0) {
		return 0, linkagePose{}, fmt.Errorf(
			"linkage closure has no opposite endpoint signs: %g and %g", lower.residual, upper.residual,
		)
	}
	var pose linkagePose
	var angle float64
	for range 128 {
		angle = (lo + hi) / 2
		pose, err = linkageAt(scale, crank, angle)
		if err != nil {
			return 0, linkagePose{}, err
		}
		if math.Abs(pose.residual) < 1e-10*scale {
			break
		}
		if (pose.residual > 0) == (lower.residual > 0) {
			lo, lower = angle, pose
		} else {
			hi = angle
		}
	}
	if math.Abs(pose.residual) >= 1e-8*scale {
		return 0, linkagePose{}, fmt.Errorf("linkage closure residual %g exceeds tolerance", pose.residual)
	}
	return angle, pose, nil
}

// Motion solves the skew treadle linkage and computes head, belt and bobbin motion.
func Motion(p Params) (State, error) {
	if err := p.Validate(); err != nil {
		return State{}, fmt.Errorf("spinning wheel motion: %w", err)
	}
	scale := p.Scale
	angle := math.Remainder(p.WheelAngle, 2*math.Pi)
	state := State{
		Wheel:     r3.Vec{X: 220 * scale, Z: 580 * scale},
		Heel:      r3.Vec{X: 60 * scale, Y: -190 * scale, Z: 60 * scale},
		HeadPivot: r3.Vec{X: -172 * scale, Z: 555 * scale},
		CrankPin: r3.Vec{
			X: (220 + 25*math.Cos(angle)) * scale,
			Y: 105 * scale, Z: (580 + 25*math.Sin(angle)) * scale,
		},
		HeadTilt:     math.Asin((p.DriveTension - 6) / 122),
		FlyerAngle:   float64(p.Ratio) * p.WheelAngle,
		BobbinAngle:  (float64(p.Ratio) - p.YarnFeedPerTurn/(2*math.Pi*31*scale)) * p.WheelAngle,
		WheelPitch:   274*scale + math.Sqrt(5),
		SpringLength: 25*scale + p.BrakeTravel/2,
	}
	switch p.Ratio {
	case 7:
		state.HeadShift = 278 * scale
	case 9:
		state.HeadShift = 270 * scale
	case 12:
		state.HeadShift = 262 * scale
	case 17:
		state.HeadShift = 254 * scale
	}
	tilt, err := r3.RotationAround(state.HeadPivot, r3.Vec{Y: -1}, units.Radians(state.HeadTilt))
	if err != nil {
		return State{}, fmt.Errorf("create head tilt: %w", err)
	}
	state.Flyer = tilt.Apply(r3.Vec{X: -140 * scale, Z: 690 * scale})
	pedalAngle, pose, err := solveLinkage(scale, state.CrankPin)
	if err != nil {
		return State{}, fmt.Errorf("solve treadle linkage: %w", err)
	}
	state.PedalAngle = pedalAngle
	state.PedalFrame, state.Toe, state.RodHead = pose.pedal, pose.toe, pose.head
	state.GimbalFrame, err = r3.NewFrame(state.CrankPin, pose.radial, pose.cross)
	if err != nil {
		return State{}, fmt.Errorf("create gimbal frame: %w", err)
	}
	rod, ok := state.RodHead.Sub(state.Toe).Normalize()
	if !ok {
		return State{}, fmt.Errorf("footman direction has no finite length")
	}
	state.RodFrame, err = r3.NewFrame(state.Toe, rod, pose.cross)
	if err != nil {
		return State{}, fmt.Errorf("create footman frame: %w", err)
	}
	state.RodAngle = math.Atan2(rod.Z, rod.X)
	state.FlyerPitch = state.WheelPitch / float64(p.Ratio)
	distance := state.Flyer.Sub(state.Wheel).Len()
	difference := state.WheelPitch - state.FlyerPitch
	if !finiteMotionValue(distance) || distance <= difference {
		return State{}, fmt.Errorf("drive pitch circles have no open common tangent")
	}
	alpha := math.Acos(difference / distance)
	state.BeltLength = state.WheelPitch*(2*math.Pi-2*alpha) + state.FlyerPitch*2*alpha +
		2*math.Sqrt(distance*distance-difference*difference)
	brakePitch := 42*scale + 0.6*math.Sqrt(5)
	state.BrakeLength = math.Pi*brakePitch + 612*scale - 36.8 + 14.6*math.Pi
	return state, nil
}

// PoseTransform maps a rigid part from base to state. Compliant parts must be rebuilt at the target pose.
func PoseTransform(base, state State, motion string, wheelDelta float64) (r3.Transform, error) {
	switch motion {
	case "fixed":
		return r3.Identity(), nil
	case "compliant":
		return r3.Transform{}, fmt.Errorf("compliant pose transform requires rebuilding the flexible connector")
	case "wheel":
		return poseRotation(base.Wheel, wheelDelta, motion)
	case "flyer":
		return posePhaseRotation(base.Flyer, base.FlyerAngle, state.FlyerAngle, motion)
	case "bobbin":
		return posePhaseRotation(base.Flyer, base.BobbinAngle, state.BobbinAngle, motion)
	case "treadle":
		return poseFrameTransform(base.PedalFrame, state.PedalFrame, motion)
	case "gimbal":
		return poseFrameTransform(base.GimbalFrame, state.GimbalFrame, motion)
	case "footman":
		return poseFrameTransform(base.RodFrame, state.RodFrame, motion)
	default:
		return r3.Transform{}, fmt.Errorf("unknown part motion %q", motion)
	}
}

func poseRotation(pivot r3.Vec, angle float64, motion string) (r3.Transform, error) {
	if !finiteMotionValue(angle) {
		return r3.Transform{}, fmt.Errorf("%s pose rotation angle must be finite", motion)
	}
	transform, err := r3.RotationAround(
		pivot, r3.Vec{Y: -1}, units.Radians(math.Remainder(angle, 2*math.Pi)),
	)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("create %s pose rotation: %w", motion, err)
	}
	return transform, nil
}

func posePhaseRotation(pivot r3.Vec, base, target float64, motion string) (r3.Transform, error) {
	if !finiteMotionValue(base) || !finiteMotionValue(target) {
		return r3.Transform{}, fmt.Errorf("%s pose phases must be finite", motion)
	}
	// Reduce before subtracting so two finite phases cannot overflow their difference.
	return poseRotation(pivot, math.Remainder(target, 2*math.Pi)-math.Remainder(base, 2*math.Pi), motion)
}

func poseFrameTransform(base, target r3.Frame, motion string) (r3.Transform, error) {
	initial, err := r3.FromFrame(base)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("create base %s pose transform: %w", motion, err)
	}
	inverse, err := initial.Inverse()
	if err != nil {
		return r3.Transform{}, fmt.Errorf("invert base %s pose transform: %w", motion, err)
	}
	final, err := r3.FromFrame(target)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("create target %s pose transform: %w", motion, err)
	}
	transform, err := inverse.Then(final)
	if err != nil {
		return r3.Transform{}, fmt.Errorf("compose %s pose transform: %w", motion, err)
	}
	return transform, nil
}
