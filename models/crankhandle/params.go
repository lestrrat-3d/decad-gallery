package crankhandle

import (
	"errors"
	"fmt"
	"math"

	"github.com/lestrrat-3d/decad-gallery/internal/m2fastener"
	"github.com/lestrrat-3d/decad-gallery/models/ballbearing"
	"github.com/lestrrat-3d/r3"
)

// Fixed dimensions in mm, in the local frame (see Build). They are the hand-crank siren's crank, which imports this
// package. Each fit gap is its class minimum from docs/model-pipeline.md "Part clearances" plus FitMargin.
const (
	FitMargin  = 0.05
	TightFit   = 0.2 + FitMargin  // 0.25: the pin's stub in the arm's hexagonal pocket
	RegularFit = 0.35 + FitMargin // 0.40: the handle on the pin's shank, and its gap to the arm

	ArmRadius    = 10.0 // the arm is 20 wide: its sides run at x = ±ArmRadius
	ArmThickness = 8.0
	BoreFloor    = 1.0 // the D bore is blind: 1 mm of arm closes it at the outer face
	PinHoleR     = 1.3 // the pin bolt's clearance hole through the arm
	PocketFlats  = StubFlats + 2*TightFit
	PocketFrom   = 3.25 // the hexagonal pocket runs from here through the outer face
	PocketLength = 5.75
	StubFlats    = 7.03 // keeps 1 mm of stub round the nut cavity (radius 2.51)
	StubFrom     = 3.5
	StubLength   = 5.8 // to z 9.3, where the shank's root cone has grown past the stub's corners
	ShankR       = 5.15
	ShankFrom    = 8.25 // the root cone starts here at radius RootR, inside the stub
	RootR        = 3.5  // inside the stub's flats (3.515), so the cone starts inside the stub
	ShankTo      = 52.75
	HeadR        = 6.45
	HeadFrom     = 55.0 // the head cone runs from ShankTo up to here
	HeadLength   = 3.0
	HandleFrom   = 7.4 // the handle sketch's origin, 1 behind the handle's inner end
	PinBolt      = 10.0
	PinCavity    = 4.75 // from the bolt head's top face, in the arm's back face, to the nut cavity's near face
	PressWall    = 1.6  // the least wall round a press-fit bore (AGENTS.md "Verification")
	ChamferSize  = 0.5  // the arm's face chamfer

	// The pin prints as two halves split by the plane through its axis and local X, the direction of the hand's
	// driving force ("Split shafts" in docs/model-pipeline.md). Two M2 x 6 cross bolts clamp the halves: each runs
	// along -Y, normal to the seam, from a head pocket in the +Y half to a hexagonal nut pocket open to the -Y half's
	// flat, so the bolt's tip ends flush with that flat. The bolt holds the nut in its pocket and the pocket's flats stop
	// it turning. A double flat CrossFlat from the axis, parallel to the seam, seats the head flush and leaves the
	// halves mirror images.
	CrossBolt     = 6.0
	CrossFlat     = 4.0  // the flats' distance from the pin axis, on both sides
	CrossFlatHalf = 3.25 // half the flats' length along the pin: 1 mm of flat round the head pocket (radius 2.2)
	CrossCavity   = 6.0  // head top face to the nut pocket's floor: 2*CrossFlat - NutCavityHeight
	// ConeAngle is the half angle of the cones from the stub to the shank and from the shank to the head. A cut through
	// the stub's hexagon reads the cone up to 1.8 times steeper, and 30 degrees keeps that turn under 50 degrees.
	ConeAngle = 30.0

	// The arm pinches the shaft. It keeps its full width past the shaft: its sides run straight on to a point
	// ClampSideRun below the clamp bolt's axis, turn into ClampCornerR corners and close with one ClampEndR end arc, every
	// join tangent. A slit SlitWidth wide runs along x = 0 from the end arc into the bore and ends in a full round in the
	// bore's floor. The M2 x 18 clamp bolt crosses the whole width along -X: its head lies flush in the +X side face,
	// and its nut sits in a hexagonal pocket open to the -X side face, m2fastener.NutCavityHeight (2) deep, so the
	// bolt's tip ends flush with that face. The tightened bolt holds the nut in the pocket, and the pocket's flats stop
	// it turning.
	ClampBolt     = 18.0
	ClampHoleWall = 1.05 // between the clamp bolt's hole and the bore, where the hole crosses the slit
	ClampSideRun  = 3.6  // 1 mm past the head pocket's ring (2.2) and the nut pocket's corners (2.51), with margin
	ClampCornerR  = 4.5
	ClampEndR     = 22.0
	SlitWidth     = 1.0
	SlitInBore    = 1.0 // the slit runs this far past the bore's wall, so its round end lies in the bore's floor
	// A nut pocket's hexagon, the clamp's or a cross bolt's, has its sharp corners on the circle of radius
	// m2fastener.NutCavityRadius, 0.2 mm past the nut's corners. Its floor's outline is rounded by NutPocketRound on the
	// tool, so the part's inside corners there pass "Corners and roots" in a section that cuts the round at a slant.
	NutPocketRound = 0.6
)

// CrossZ are the cross bolts' distances along the pin from the arm's back face: one past the pin bolt's hole and one
// short of the head cone.
var CrossZ = [2]float64{17, 49}

// HandleProfile is the handle's meridian: (distance along the pin from the sketch origin at HandleFrom, radius). A small
// barrel with a waist and a flared outer end, with a bore for the pin's shank and a counterbore for its head. Every
// corner is cut by a small chamfer, so the handle is made of cones and cylinders. The origin is 1 behind the inner end.
// The bore (radius 5.55) clears the shank by RegularFit. Its top end is a 30 degree cone 0.4 mm off the pin's head
// cone, into the counterbore (radius 7.05) that clears the head by 0.6 mm.
var HandleProfile = [...][2]float64{
	{1.5, 5.55}, {45.24, 5.55}, {47.84, 7.05}, {50.5, 7.05}, {51, 7.55}, {51, 8.3}, {49.8, 9.3},
	{44, 8}, {37, 7.25}, {23, 8}, {9, 7.8}, {2.2, 7.2}, {1, 6.6}, {1, 6.05},
}

// Params specifies a crank in mm. The shaft runs in a purchased bearing, so its diameter equals the bearing's bore. A
// nil Placement selects the local frame.
type Params struct {
	// ShaftDiameter is the diameter of the D-flat shaft the arm presses onto.
	ShaftDiameter float64
	// FlatDepth is how deep the shaft's flat cuts below its round.
	FlatDepth float64
	// BoreClearance is the radial clearance of the arm's D bore round the shaft, on the round and on the flat.
	BoreClearance float64
	// Throw is the distance from the shaft axis to the pin axis.
	Throw     float64
	Placement *r3.Frame
}

// Params605ZZ returns a crank for the 5 mm shaft of a 605ZZ bearing (ballbearing.Params605ZZ), with the planetary
// gearbox shafts' 0.5 mm flat.
func Params605ZZ() Params {
	return Params{ShaftDiameter: ballbearing.Params605ZZ().BoreDiameter, FlatDepth: 0.5,
		BoreClearance: ballbearing.PressFitClearance, Throw: 55}
}

// Params6082RS returns a crank for the 8 mm shaft of a 608-2RS bearing (ballbearing.Params6082RS), with a 0.75 mm
// flat: the hand-crank siren's crank. A 608ZZ has the same bore.
func Params6082RS() Params {
	return Params{ShaftDiameter: ballbearing.Params6082RS().BoreDiameter, FlatDepth: 0.75,
		BoreClearance: ballbearing.PressFitClearance, Throw: 55}
}

// DefaultParams returns Params605ZZ.
func DefaultParams() Params { return Params605ZZ() }

// Dimensions are the values Derive computes from Params, in mm, in the local frame.
type Dimensions struct {
	BoreRadius float64 // r_b: ShaftDiameter/2 + BoreClearance
	BoreFlat   float64 // f_b: distance from the axis to the bore's flat, on the +Y side
	BoreLoop   float64 // length of the bore's outline in the back face: its arc the long way round plus its flat
	// Site is the pin bolt: its head's top face is flush with the arm's back face at the pin axis, and it runs along
	// +Z into the nut cavity in the pin's stub.
	Site m2fastener.Site
	// CrossSites are the two cross bolts that clamp the pin's halves, at CrossZ.
	CrossSites [2]m2fastener.Site
	// ClampY is the clamp bolt's axis, y = -(r_b + 1.3 + ClampHoleWall), at mid thickness. SideEnd is where the arm's
	// sides stop being straight, ClampY - ClampSideRun. Reach is how far the end arc runs behind the shaft axis.
	ClampY, SideEnd, Reach float64
	// ClampSite is the clamp bolt: its head's top face is flush with the +X side face at (ArmRadius, ClampY), and it
	// runs along -X. Its Cavity is the nut pocket's floor, NutCavityHeight inside the -X side face, so CavityFar and
	// the bolt's tip both lie on that face.
	ClampSite m2fastener.Site
}

// CornerCenter is the centre of the right corner arc; the left one is its mirror in x = 0. EndCenter is the centre of
// the end arc, on x = 0.
func (dm Dimensions) CornerCenter() r3.Vec { return r3.Vec{X: ArmRadius - ClampCornerR, Y: dm.SideEnd} }

// EndCenter is the centre of the end arc, which touches both corner arcs from inside.
func (dm Dimensions) EndCenter() r3.Vec {
	dx := ArmRadius - ClampCornerR
	return r3.Vec{Y: dm.SideEnd + math.Sqrt((ClampEndR-ClampCornerR)*(ClampEndR-ClampCornerR)-dx*dx)}
}

// Validate reports the first problem Derive finds.
func (p Params) Validate() error {
	_, err := p.Derive()
	return err
}

// Derive validates p and computes its dimensions.
func (p Params) Derive() (Dimensions, error) {
	for _, f := range []struct {
		name  string
		value float64
	}{
		{"ShaftDiameter", p.ShaftDiameter}, {"FlatDepth", p.FlatDepth}, {"BoreClearance", p.BoreClearance},
		{"Throw", p.Throw},
	} {
		if math.IsNaN(f.value) || math.IsInf(f.value, 0) || f.value <= 0 {
			return Dimensions{}, fmt.Errorf("crankhandle: %s must be finite and positive", f.name)
		}
	}
	if p.FlatDepth > p.ShaftDiameter/4 {
		return Dimensions{}, errors.New("crankhandle: FlatDepth must be at most a quarter of ShaftDiameter")
	}
	if p.BoreClearance < ballbearing.PressFitClearance || p.BoreClearance > 0.5 {
		return Dimensions{}, fmt.Errorf("crankhandle: BoreClearance must be from %g to 0.5 mm",
			ballbearing.PressFitClearance)
	}
	dm := Dimensions{
		BoreRadius: p.ShaftDiameter/2 + p.BoreClearance,
		BoreFlat:   p.ShaftDiameter/2 - p.FlatDepth + p.BoreClearance,
	}
	half := math.Sqrt(dm.BoreRadius*dm.BoreRadius - dm.BoreFlat*dm.BoreFlat)
	dm.BoreLoop = 2*dm.BoreRadius*(math.Pi-math.Acos(dm.BoreFlat/dm.BoreRadius)) + 2*half
	if ArmRadius-dm.BoreRadius < PressWall {
		return Dimensions{}, fmt.Errorf("crankhandle: ShaftDiameter leaves less than a %.1f mm wall in the %g mm arm",
			PressWall, 2*ArmRadius)
	}
	if p.Throw < 2*ArmRadius {
		return Dimensions{}, fmt.Errorf("crankhandle: Throw must be at least %g mm", 2*ArmRadius)
	}
	if p.Placement != nil && !p.Placement.IsValid() {
		return Dimensions{}, errors.New("crankhandle: Placement must be a valid frame")
	}
	dm.Site = m2fastener.Site{
		Head:   r3.Vec{Y: p.Throw},
		Dir:    r3.Vec{Z: 1},
		Corner: r3.Vec{X: 1},
		Length: PinBolt,
		Cavity: PinCavity,
	}
	if err := dm.Site.Validate(); err != nil {
		return Dimensions{}, fmt.Errorf("crankhandle: %w", err)
	}
	dm.ClampY = -(dm.BoreRadius + m2fastener.HoleRadius + ClampHoleWall)
	dm.SideEnd = dm.ClampY - ClampSideRun
	dm.Reach = ClampEndR - dm.EndCenter().Y
	dm.ClampSite = m2fastener.Site{
		Head:   r3.Vec{X: ArmRadius, Y: dm.ClampY, Z: ArmThickness / 2},
		Dir:    r3.Vec{X: -1},
		Corner: r3.Vec{Y: 1},
		Length: ClampBolt,
		Cavity: 2*ArmRadius - m2fastener.NutCavityHeight,
	}
	if err := dm.ClampSite.Validate(); err != nil {
		return Dimensions{}, fmt.Errorf("crankhandle: clamp bolt: %w", err)
	}
	if dm.ClampSite.Tip() != dm.ClampSite.CavityFar() {
		return Dimensions{}, errors.New("crankhandle: clamp bolt: its tip must end flush with the -X side face")
	}
	for i, z := range CrossZ {
		dm.CrossSites[i] = m2fastener.Site{
			Head:   r3.Vec{Y: p.Throw + CrossFlat, Z: z},
			Dir:    r3.Vec{Y: -1},
			Corner: r3.Vec{Z: 1},
			Length: CrossBolt,
			Cavity: CrossCavity,
		}
		if err := dm.CrossSites[i].Validate(); err != nil {
			return Dimensions{}, fmt.Errorf("crankhandle: cross bolt %d: %w", i+1, err)
		}
		if dm.CrossSites[i].Tip() != dm.CrossSites[i].CavityFar() {
			return Dimensions{}, fmt.Errorf("crankhandle: cross bolt %d: its tip must end flush with the -Y flat", i+1)
		}
	}
	return dm, nil
}
