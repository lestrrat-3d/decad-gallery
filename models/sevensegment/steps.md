# Seven-segment display Decad steps

Governing geometry: `instructions.md`. Each section names the operations in build order and its check.
Every sketch is verified fully constrained before its geometry is used. A decad refusal stops the work.

## 1. Validate and derive (params.go)

### Go API

Package `sevensegment` (`models/sevensegment`). Every exported symbol:

```go
// Segments and truth table
type Segment int
const ( SegA Segment = iota; SegB; SegC; SegD; SegE; SegF; SegG )
func (s Segment) String() string            // "a".."g"
func Segments() [7]Segment                  // a..g
func Lit(digit int, s Segment) (bool, error)

// Inputs
type Params struct {
    Digit     int       // 0..9, the digit shown; sets θ = 36·Digit and every slider's height
    Placement *r3.Frame // nil = the world frame above
}
func DefaultParams() Params                 // Digit 2, nil Placement
func (p Params) Validate() error
func (p Params) Derive() (Dimensions, error)

// Derived, no geometry
type Ramp struct {
    From   int     // digit before the step; the step is From → (From+1) mod 10
    Rising bool
    Start  float64 // cam-local degrees, ψ_From + RampStart (not wrapped)
    End    float64 // Start + (RampEnd - RampStart)
}
type RampGeometry struct {
    C1, C2, T1, T2 r3.Vec // cam-local, Z = 0
    LineLength     float64
}
func (r Ramp) Geometry() RampGeometry
type Dimensions struct {
    Digit      int
    Theta      float64      // degrees
    Ramps      [7][]Ramp    // indexed by Segment, in increasing Start
    SeatAngle  float64      // pawl head centre past the lock face, degrees
    SeatRadius float64
    LockPhase  float64      // wheel-local angle of lock face 0 at θ = 0: PawlAngle − SeatAngle
    PawlArmInner, PawlArmOuter float64 // RatchetTip + LooseFit, + PawlArmRoot (outer edge at the root)
    CrestAngle float64      // ψ' of maximum pawl lift
    LiftMax    float64      // δ at the crest
    Sites      [10]m2fastener.Site // T1..T4, G1..G4, D1, D2 in the world frame (before Placement)
}
func (d Dimensions) Station(s Segment) float64             // y_i
func (d Dimensions) GrooveRadius(s Segment, psi float64) float64
func (d Dimensions) Lift(s Segment, theta float64) float64 // GrooveRadius(s, 90+theta) - OffRadius
func (d Dimensions) PressureAngle(s Segment, psi float64) float64
func (d Dimensions) RatchetRadius(psi float64) float64     // outline radius at wheel-local ψ (front view)
func (d Dimensions) PawlDeflection(psiRel float64) (delta, contactRadius float64)
func (d Dimensions) PawlContact(psiRel float64) (delta, rc, beta float64) // beta: contact normal from the radial, clockwise +

// Ratchet flank
type FlankKnot struct{ Rel, Alpha float64 } // flank angle Alpha (degrees) at Rel degrees past a lock face
var RatchetFlank = [...]FlankKnot{{0, 50}, {18, 50}, {20, 42.5}, {22, 38}, {24, 35.3}, {26, 34.2}, {30, 33.4},
    {35.5, 31.8}, {36, 15}}
func FlankAngle(rel float64) float64         // linear between the knots

// Build
const (
    FrameContainerName, WallsContainerName, PlatesContainerName, DetentPlateContainerName,
    CamshaftContainerName, ShaftContainerName, CamsContainerName, RatchetWheelContainerName,
    CollarRingContainerName, KnobContainerName, SlidersContainerName, BearingsContainerName,
    BoltsContainerName, NutsContainerName = "Frame", "Walls", "Plates", "Detent Plate", "Camshaft", "Shaft",
    "Cams", "Ratchet Wheel", "Collar Ring", "Knob", "Segment Sliders", "Bearings", "Bolts", "Nuts"
)
type Parts struct {
    FrontWall, RearWall, DetentPlate, GuidePlate, TopPlate *decad.Body
    Shaft, RatchetWheel, CollarRing, Knob                  *decad.Body
    Cams, Sliders                                          [7]*decad.Body // indexed by Segment
    Sketches   []*sketch.Sketch
    Dimensions Dimensions
    Sites      [10]m2fastener.Site // placed
}
type Model struct {
    Parts
    Bearings [2]*ballbearing.Model // front, rear
    Bolts    [10]*m2bolt.Model
    Nuts     [10]*m2nut.Model
}
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error)
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)
```

`NewDocument(opts ...decad.DocumentOption) *decad.Document` is added only if a Boolean refuses with
`decad.ErrUnsupported` at a held-bound gate (onewayclutch `document.go` pattern, with the measured bound in its
comment). Any other decad failure stops the task.

Exported constants (values above): `SegmentSpan 33, SegmentLength 30, SegmentWidth 6, SegmentGap 1.5,
Travel 3.0, FaceZ 88.25, ShaftZ 28.0, StationPitch 11.0,
CamRadius 25.0, CamThickness 3.8, HubRadius 5.0, OffRadius 18.0, OnRadius 21.0, GrooveWidth 3.8, GrooveDepth 1.9,
RampStart 8.0, RampEnd 29.5, RampRound 2.5, EntryAngle 18.0, PinRadius 1.5, PinLength 2.05, StemWidth 7.0,
StemThickness 3.6, StemRound 0.3, RatchetTeeth 10, RatchetRoot 6.5, RatchetTip 11.876
(= flank(36), an exported var because it is not constant-foldable), RatchetRootRound 0.6,
RatchetThickness 10.65, PawlTipRound 0.4, PawlFlankGap 0.10, PawlAngle 95.0, PawlArmLength 22.0, PawlArmRoot 1.90,
PawlArmTip 1.2, WindowRadius 18.8, DetentThickness 12.0, WallThickness 8.0,
WallHalfWidth 27.0, Seam 0.25,
PressFit 0.05, TightFit 0.25, RegularFit 0.40, LooseFit 0.55, TimingAllowance 1.5, KnobRadius 20.0,
KnobThickness 12.0, CollarRingRadius 6.0, CollarRingLength 3.5, RearCollarRadius 3.8`.

Validation (first failure wins, prefix `sevensegment: `):

1. `digit %d out of range 0..9`
2. `invalid placement frame` (Placement non-nil and `!IsValid()`)

`BuildParts`/`Build` reject nil ctx (`context must not be nil`) and nil doc (`document must not be nil`), derive,
then check `ctx.Err()` before geometry. Every geometry error is wrapped `sevensegment: <part>: <op>: %w`. A body that
is not `IsSolid()` fails `sevensegment: <part> is not a solid`.

Files (package): `params.go` (constants, Segment, truth table, Params, Validate, Derive, Dimensions methods, ramp and
detent math, sites), `sketches.go` (run, sketch helpers, polar helpers, meridian, extrude/profile),
`cams.go`, `detent.go` (ratchet wheel, detent plate), `shaft.go` (shaft, collar ring, knob), `sliders.go`,
`frame.go` (walls, guide plate, top plate), `sevensegment.go` (package doc, containers, Parts, Model, BuildParts,
Build, pose). Tests use testify `require` and `t.Context()`. Tests of the public API and the assembled model are in
the external package `sevensegment_test`. Per-part tests in Tasks 4-8 need the unexported builders before `BuildParts`
exists, so they are internal test files (`*_internal_test.go`, `package sevensegment`), as models/swisslever does;
no per-part builder is exported.

Check: `TestSevenSegmentTruthTable`, `TestSevenSegmentValidate`, `TestSevenSegmentCamProfile`,
`TestSevenSegmentDetent`, `TestSevenSegmentChannel` (params_test.go), all without geometry. Derive solves the pawl
seat numerically and `PawlDeflection` by bisection as in the deflection model.


### Flag API (added with the flag mechanism)

```go
const (
	FaceZ           = 88.25
	RotorAxisZ      = 82.6
	RotorHalfWidth  = 3.75
	RotorHalfLength = 12.25
	JournalRadius   = 0.6
	PinOffset       = 1.5

	FlagsContainerName      = "Flags"
	FlagUnitName            = "Flag"
	YellowFaceContainerName = "Yellow Face"
	BlackBodyContainerName  = "Black Body"
)

// Flag is one printed two-colour rotor.
type Flag struct{ Yellow, Black *decad.Body }

// Parts gains CradlePlate *decad.Body and Flags [7]Flag (indexed by Segment).

// FlagAngle returns segment s's modeled rotor angle in degrees at camshaft angle theta:
// atan2(Lift(s, theta) - 1.75 + raise_s, PinOffset), raise_s 0 (a, g, d), 0.20 (b, c), 0.25 (e, f).
func (d Dimensions) FlagAngle(s Segment, theta float64) float64
```

## 2. Sketch helpers (sketches.go)

Follow `models/onewayclutch/sketches.go`: `run` holds ctx, doc, params, dimensions and the sketch list;
`sketchOn(frame)`, `profile(s, holes)` (Solve, `sketchcheck.VerifyFullyConstrained`, one valid profile with the
given hole count), `extrude`, `newPolar`/`axis`, `meridian` (points placed with `sketchcheck.DimensionPoint`).
Check: every sketch built in every test is fully constrained.

## 3. Shaft (shaft.go)

### Shaft (`Shaft`, printed, split)

Local frame: origin (0, 0, 28), U = +Z, V = +X, N = +Y (local z = world y; local +x = world +Z, the flat normal).

Revolve this meridian `(z, r)` about local Z with `decad.FullRevolution{}` (pattern: onewayclutch `meridian`):

```
(-72.2, 0) (-72.2, 2.2) (-71.9, 2.5) (-49.25, 2.5) (-49.25, 3.2) (39.7, 3.2)
(40.739, 3.8) (42.75, 3.8) (42.75, 2.5) (47.7, 2.5) (48.0, 2.2) (48.0, 0)
```

Then cut three flat tools (box prisms in the local frame, extruded symmetric ±4 along local y)
(`flatTool(x0, x1, z0, z1)`: x from x0 to x1):

- Cam flat: `x ≥ 2.7` over `z ∈ [−50.25, 39.2]`; square end at z 39.2 (cam a's D chord stops on it).
- Cam back flat: `x ≤ −2.7` over the same span; square end at z 39.2.
- Stub flat: `x ≥ 2.0` over `z ∈ [−73.2, −57.85]`; square end at z −57.85 (the collar ring's D chord stops on it,
  0.05 short of the ring's back face at −57.9). No run-out: both flats end square as D-chord stops, listed as
  mating pairs (split-shaft "Corners").

Sections: knob stub r 2.5 D (flat 2.0) [−72.2, −57.85]; front journal r 2.5 round [−57.85, −49.25] through the lip
hole and the front bearing; cam section r 3.2 double D (flats ±2.7) [−49.25, 39.2]; rear collar r 3.8; rear journal r 2.5
round in the rear bearing. End chamfers 0.3 at both ends. Length 120.2.

Failure: return `sevensegment: shaft: <op>: <err>`.
Check: `TestSevenSegmentShaft`, `TestSevenSegmentCollars`.

## 4. Cams (cams.go)

### Cams (`Cams`, 7 bodies)

Local frame per cam: origin (0, y_i + 2.35, 28) (the groove face), U = +X, V = +Z, N = −Y. Disc at local
z ∈ [−3.8, 0], hub toward local +z.

1. Revolve meridian `(r, z)`: `(0,−3.8) (24.5,−3.8) (25,−3.3) (25,−0.5) (24.5,0) (5.5,0)`, then an 8-chord quarter
   round about (5.5, 0.5) from (5.5, 0) to (5.0, 0.5) (pattern: `m2fastener.quarterRound`), then `(5.0,7.15) (0,7.15)`.
   Rim chamfers 0.5 are in the meridian; the hub root round 0.5 is a corner between layers.
2. Cut the double-D bore: profile on local z = −4.8, two arcs r 3.25 joined by flats at v = ±2.75
   (doubleDBoreLoop); extrude 12.95 along +z. The first flat is on local +V (world +Z at θ = 0) on every D bore in
   the model; the cam and ratchet bores add the opposite flat.
3. Cut the groove tool (next section): profile on local z = +1.0, extrude `decad.Against` 2.9 (to z −1.9).
4. Cut the look-alike holes: circles r 0.75 at radius 12 (front cams) or 8 (rear cams) on angles 270°, 295°,
   320°, 345° (count per cam, "Look-alike parts"), extruded through local z ∈ [−4.8, 1.0]. Each hole has its own
   sketch on the cam frame, centre placed by horizontal and vertical distances from the origin (rule text).

Groove (one sketch per cam; profile with one hole):

- Walls are offsets of the centreline by ±`GrooveWidth/2` = ±1.9. Groove depth `GrooveDepth` 1.9.
- Dwell arcs about the origin: inner wall r − 1.9, outer wall r + 1.9 (OFF: 16.1 / 19.9; ON: 19.1 / 22.9).
- Each ramp is arc–line–arc on the centreline with `RampRound` ρ = 2.5:
  - Rising ramp from ψ_a to ψ_b = ψ_a + 21.5: arc 1 centre `C1 = 20.5∠ψ_a` (outside, r_off + ρ), arc 2 centre
    `C2 = 18.5∠ψ_b` (inside, r_on − ρ), joined by their internal common tangent. With ψ_a = 8°: C1 (20.3005, 2.8530),
    C2 (16.1016, 9.1098), tangent points T1 (17.8231, 3.1883) at 18.106∠10.142°, T2 (18.5790, 8.7746) at
    20.547∠25.281°, line length 5.6372. Unit normal n of the line satisfies `n·(C2 − C1) = 2ρ`, `T1 = C1 + ρn`,
    `T2 = C2 − ρn`, pick the root whose tangent points lie inside (ψ_a, ψ_b) in order (see "Ramp geometry algorithm").
  - Falling ramp: mirror of the rising ramp about the window's mid-angle: `C1 = 18.5∠ψ_a`, `C2 = 20.5∠ψ_b`,
    T1 at 20.547∠(ψ_a + 4.219°), T2 at 18.106∠(ψ_a + 19.358°).
  - Wall radii: an arc whose centre is outside the centreline (away from the origin) has inner wall ρ + 1.9 = 4.4 and
    outer wall ρ − 1.9 = 0.6; an arc whose centre is inside has inner 0.6 and outer 4.4. Lines offset ±1.9.
- Entry channel (outer wall only): radial slot on the ray `EntryAngle` 18.0°, width 3.8 (sides parallel to the ray at
  ±1.9), from the outer wall's ON dwell arc (r 22.9) out to a cap line at distance 26.0 from the origin. It lies in
  the dwell [11.5, 26] of every cam (all ON at digit 8); its sides meet r 22.9 at 18.0 ± 4.76°.
- Sketch constraints (no point in the groove sketch is placed with `DimensionPoint`):
  - Polar construction per onewayclutch `newPolar`/`axis`: a horizontal reference line, one construction ray per
    ramp boundary angle (ψ_a, ψ_b) and one at `EntryAngle`.
  - Each ramp centre: `NewPointOnLine(C, ray)` + `NewDistance(origin, C, d)` (d = 20.5 or 18.5).
  - Each dwell–ramp junction point J (both walls): `NewPointOnLine(J, ray)`; its radius comes from the dwell arc.
  - Each dwell arc: centre = sketch origin; `NewRadius` on the arc (fixes its start); its end radius follows from
    the arc's own equal-radius property. Do not also dimension J's distance (redundant).
  - Each ramp wall arc: centre = the shared ramp centre point; no radius constraint (its start J fixes it).
  - Each ramp wall line: shares end points with its arcs; `NewTangent(line, arc)` at both ends. The tangent points
    are then fixed by tangency plus the arc's equal-radius property.
  - Channel: side lines `NewParallel(entryRay, side)` + `NewDistancePointLine(origin, side, 1.9)`; side's inner end
    is the end point of the split ON dwell arcs; outer ends lie on the cap line (`NewPerpendicular(entryRay, cap)`,
    `NewDistancePointLine(origin, cap, 26.0)`).
  - If `Verify` reports redundancy at a tangent junction, drop that junction's tangency (the geometry is already
    tangent by construction) and record the case in steps.md; do not add fixed points.
- Ramp geometry algorithm (Derive, no sketch):
  ```
  base = atan2(C2 - C1); off = acos(2ρ / |C2 - C1|)
  for φ1 in (base + off, base - off):
      n = (cos φ1, sin φ1); d = (n.y, -n.x)
      require (C2 - C1)·d > 0
      sweep1 = wrap((ψ_a + 180°) - φ1) in (0, 180°]   # arc 1 runs clockwise about C1
      sweep2 = wrap(ψ_b - (φ1 + 180°)) in (0, 180°]  # arc 2 runs counter-clockwise about C2
      accept the first φ1 meeting all three
  ```
- Pressure angle: angle between the centreline's tangent and the circumferential direction at the follower point;
  max 33.0°.

Cam local→world: frame origin (0, y_i + 2.35, 28), U = +X, V = +Z. Printed back face down (groove face and hub up,
axis vertical); no other orientation prints the groove and hub without support.
5. Engrave the letter: draw its outline loops on the back face's plane (horizontal and vertical lines, each held by its distance
   from the axis), take the profile with the counter as its hole, and `decad.Emboss` it engraved 0.5 mm with
   `decad.Faces(decad.Facing(-Z))`, the one planar face looking along local -Z.
Check: `TestSevenSegmentCam` (cams b and e first), `TestSevenSegmentCamInscription`.

## 5. Ratchet wheel (detent.go)

### Ratchet wheel (`Ratchet Wheel`)

Local frame: origin (0, −37.85, 28) (back face), U = +X, V = +Z, N = −Y; extrude `Along` 10.65 (to world y −48.5).
Printed with its axis vertical, either face down.

Outline (one closed loop) plus the double-D bore hole (r 3.25, flats at ±2.75 along V). Ten teeth, tooth j occupying wheel-local
angles `[φ_j, φ_j + 36]`, `φ_j = LockPhase + 36 j`, `LockPhase = PawlAngle − SeatAngle` (≈ 90.295°, from Derive) so
the pawl A head seats on angle `PawlAngle` 95°:

- Lock face: radial line on the ray φ_j from `RatchetTip` (11.876) in to `RatchetRoot` 6.5.
- Root round: arc `RatchetRootRound` 0.6 tangent to the lock face and to the flank (centre and end computed in
  `rootRound`; placed with `DimensionPoint`, as the spur gear's root contacts are placed). It stays outside the seated
  head (`TestSevenSegmentDetent` checks the seat lies past it); it is below half the tooth section, so it is listed. On
  the 50° flank it lifts the lowest outline point to 7.41.
- Flank: `r(rel) = RatchetRoot · exp(∫ tan α d rel)`, rel = ψ − φ_j in degrees, where the flank angle α (between the
  flank and the circle through it) is linear in rel between the `RatchetFlank` knots (rel°, α°): (0, 50) (18, 50)
  (20, 42.5) (22, 38) (24, 35.3) (26, 34.2) (30, 33.4) (35.5, 31.8) (36, 15). Each piece integrates in closed form,
  `−(ln cos(a₀ + b x) − ln cos a₀) / b` with b = dα/d rel in rad/rad, or `tan(a₀) x` where b = 0 (`flank` in
  `params.go`; `FlankAngle` interpolates the knots). The knots are tuned so the release budget is nearly flat over
  `ψ' ∈ [7.98, 29.55]`. The flank runs from the root round's end to rel 36 (r 11.876), drawn as a `CreateFitSpline`
  through 17 samples placed with `sketchcheck.DimensionPoint` (pattern: `models/spurgear/tooth_section.go` flank
  splines). `RatchetRoot` 6.5 is the flank's start radius on the lock
  face ray; the outline itself stays above 6.5 because of the root round.
- Tip: the flank meets the next lock face at a sharp corner (engagement corner of a ratchet tooth, exempt).

No orientation key: a reversed wheel makes the knob turn counter-clockwise, which the README's assembly check catches.
Check: `TestSevenSegmentRatchet`.

## 6. Collar ring (shaft.go)

### Collar ring (`Collar Ring`)

Local frame origin (0, −57.9, 28), U = +X, V = +Z. Circle r 6 with a D hole (r 2.55, flat 2.05 on +V); extrude
`Along` 3.5. Pressed on the knob stub until its D chord stops on the stub flat's square end (0.05 gap modeled), which
leaves it 0.4 from the front wall's outer face; the knob sits 0.05 in front of it. Printed axis vertical.

## 7. Knob (shaft.go)

### Knob (`Knob`)

Local frame origin (0, −61.45, 28), U = +X, V = +Z. Revolve meridian `(r, z)`: `(0,0) (19.5,0) (20,0.5) (20,11.5)
(19.5,12) (0,12)`. Cut a blind D bore (r 2.55, flat 2.05 on +V) over local z ∈ [−1, 10.8] (floor 1.2 thick; shaft
end 0.05 short of it). Printed front face down.

## 8. Detent plate (detent.go)

### Detent

### Pawl geometry (in the detent plate)

Front-view sketch coordinates (u, v) = (x, z − 28), origin on the shaft axis, at θ = 0. Each pawl is drawn in its own
pawl frame: seat on +u, arm along +v, origin on the shaft axis. Pawl A is that frame turned by `PawlAngle` 95° (head
above the wheel), pawl B by 275°.

- Seat: tip round radius `PawlTipRound` 0.4 centred at `S`, the point at distance 0.4 + `TightFit` (0.65) from the lock
  face line and 0.4 + `PawlFlankGap` (0.50) from the flank of tooth 0 (wheel at θ = 0). Derive solves it numerically:
  `SeatAngle` 4.705° past the lock face, `SeatRadius` 7.924. The 0.10 float to the flank is a working-contact float
  (instructions.md Part fits), so the head meets the flank between ψ' 0.5 and 2.0 and lifts early.
- Lock flank: line parallel to the lock face ray at distance 0.25 on its counter-clockwise side, tangent to the tip
  round, from the tip round out to the arm's outer edge (u = `PawlArmInner` + 1.2, v ≈ −0.86).
- Back flank: line tangent to the tip round on its other side, to `B = (PawlArmInner, 2.26)` (`pawlBackV`); round 1.0
  at B.
- Arm: a straight cantilever tangent to the wheel at the head, anchored counter-clockwise of it, so friction pulls the
  arm along its length and the head lifts radially. Inner edge on u = `PawlArmInner` = `RatchetTip` + `LooseFit` =
  12.426 from B to the root at v = 2.26 + `PawlArmLength` 22.0 = 24.26. Outer edge: a line from the lock flank's outer
  end (thickness `PawlArmTip` 1.2) to (`PawlArmInner` + `PawlArmRoot` 1.90, 24.26). `PawlArmOuter` = 14.326, the outer
  edge at the root.
- Root rounds: 2.0 on the inner (tension) side, 1.0 on the outer side (at least half of 1.90).
- Slots: the inner slot is 3.0 wide (its side on u = `PawlArmInner` − 3.0); the outer slot's side lies on
  u = `PawlArmInner` + 3.55 at the root and + 4.75 where it meets the window circle (`pawlOuterSide`, `pawlOuterOpen`),
  so the slot is 1.65 wide at the root and 2.85 at the window; the side did not move when the arm thickened. Slot-end
  rounds 0.5.
- Window: circle `WindowRadius` 18.8 joining the four slot sides, rounds 1.0 where they meet. At `PawlAngle` 95°,
  pawl A's slot end lies 1.44 from the plate edge (x −25.56) and pawl B's outer slot keeps 1.22 to the D2 head
  pocket in the section 1 mm into the pocket (`TestSevenSegmentDetentPlate` requires 1.0).
- Sketch constraints: every line is held from the origin or the polar axes. With `arm` = the axis at the pawl's angle
  + 90°: outer slot side and arm outer edge by `NewDistancePointLine` from the origin and `NewAngle(arm, line, a)` (two
  point distances were near-singular); root lines `NewPerpendicular(arm, line)` plus distance 24.26; arm inner edge and
  inner slot side `NewParallel(arm, line)` plus distance; lock flank parallel to the axis at `LockPhase` (+180° for pawl
  B) and tangent to the tip arc; back flank tangent to the tip arc plus its distance from the origin; tip arc and window
  arcs by radius. Only the seat centre is a `DimensionPoint`. The rounds are `sketch.CreateFillet` calls, one per
  corner.

Deflection model (`Dimensions.PawlDeflection(ψ')` and `Dimensions.PawlContact(ψ')`, used by tests and the budget):

```
pawl frame: lock face of tooth 0 on the ray at −SeatAngle, seat on +u
wheel outline W: polyline of one tooth (lock face, root round, flank at 0.1° steps), repeated for teeth -1, 0, 1
head centre H(δ, ψ') = (SeatRadius + δ) ∠ (ψ')        # turning θ by ψ' moves the head to +ψ' against the wheel
δ(ψ') = least δ >= 0 with dist(H, W) >= PawlTipRound  # bisection on [0, 7], 40 steps
r_c(ψ') = |closest point P of W to H(δ(ψ'), ψ')|
β(ψ') = angle of the normal H − P from the radial through H, positive clockwise (PawlContact)
crest = argmax δ
```

Expected (`TestSevenSegmentDetent` asserts the contact angle, the crest and the lift): first contact between 0.5 and
2.0 (δ(1.0) 0.003), δ(8) 1.15, δ(12) 1.88, δ(24) 3.51, crest 31.3°, δ_max 4.35 (asserted 4.35 ± 0.03). Arm strain and
the budget are under instructions.md Determinacy. Derive exposes `PawlArmInner`, `PawlArmOuter` and `LockPhase` as
`Dimensions` fields (they depend on the computed `RatchetTip` and `SeatAngle`).

### Detent plate (`Detent Plate`)

Sketch frame origin (0, −37.25, 28), U = +X, V = +Z, N = −Y; extrude `Along` 12 (to y −49.25).

1. Profile: rectangle u ∈ [−27, 27], v ∈ [−28, 21] (z 0..49); holes: the window loop (Pawl geometry: the circle
   `WindowRadius` 18.8 with two slots, each holding a straight arm whose root joins the plate at the slot's end) and two
   bolt holes r 1.3 at (±21, −18) (world (±21, z 10)). Select the profile with 3 holes.
2. Chamfer both faces' outer loop 0.5 (`edgefinish.Chamfer`, `Long(Outer)`). The window loop's face edges get no
   chamfer (pawl contact and bending section; listed).
3. Cut `m2fastener.PocketTool` at sites D1, D2 (head pockets in the inner face, y −37.25 to −39.25).

Printed inner face down; pawls bend in the print plane. Recommend PETG.
Check: `TestSevenSegmentDetentPlate`.

## 10. Walls (frame.go)

### Walls (`Walls`: Front Wall, Rear Wall)

Front wall sketch frame origin (0, −49.5, 28), U = +X, V = +Z, N = −Y, extrude `Along` 8. Rear wall origin
(0, 43.0, 28), U = −X, V = +Z (N = +Y), extrude `Along` 8. Sketch rectangle u ∈ [−28.5, 28.5], v ∈ [−28, 53.25]
(z 0..81.25): the plate is 57 wide and the ribs end 1.5 inside its edges (|x| 27, z 79.75).

1. Extrude the plate; union four rib boxes at |x| ∈ [20, 27] extending inward from the inner face: ledge ribs
   z [49.75, 53.75], top flange ribs z [75.75, 79.75]; depth 10 except the front ledge ribs, 18.7 (to y −30.8).
   Ribs are unioned before any chamfer so no chamfer notch is left where a rib meets the plate rim.
2. Chamfer 0.5 the outer loop of each face (`edgefinish.Chamfer`, `Long(Outer)`, one call for the outer face and one for
   the inner face). The ribs stand 1.5 inside the plate's edges, so they do not cut into the inner face's outer loop.
3. `decad.Fillet` radius 0.5 on every concave rib-root edge in the inner-face plane (ledge: top, bottom and x = ±20
   edges; top flange: bottom and x = ±20 edges). Pattern: `models/planetarygearbox/cover.go` `roundBossRoot`.
4. Cut the seat/lip revolve tool about the shaft axis. Meridian `(r, d)`, d measured from the inner face (d 0)
   through the wall to the outer face (d 8): `(0,−1) (7.55,−1) (7.55,0) (7.05,0.5) (7.05,5.0) (4.5,5.0) (4.5,7.5)
   (5.0,8.0) (5.0,9.0) (0,9.0)`. Seat r 7.05 (605ZZ `SeatDiameter/2`) 5 deep with a 0.5 lead-in at the inner face,
   lip 3 thick with hole r 4.5 and a 0.5 chamfer at the outer face.
5. Cut fastener tools: ledge ribs `m2fastener.PocketTool` (G sites, head pockets from the rib underside) and
   `HoleTool`; top flange ribs `HexPocketTool(site, NutCavityHeight + Stock)` (T sites) and `HoleTool`; front wall
   only: `HexPocketTool` + `HoleTool` at D1, D2 (nut pockets open to the outer face).
   Cut the fastener tools before the step 4 seat tool: after the revolve cut, the front wall's G head pockets
   meet a held mesh coarser than their chord tolerance and decad refuses them.
6. Cut look-alike holes (r 0.75, through): front 1 hole, rear 2 holes, on radius 10 about the shaft axis at 270°,
   295° (front-view angles), one sketch per hole as for the cams.

Printed outer face down; ribs rise from the plate; rib pockets print sideways with a hex corner up.

## 9. Flags: rotors, sliders, cradle, top plate, guide columns

Operations in build order, each from the instructions.md sections of the same name (every construction was proven in a
one-off scratch build before the port, which is not in the repository; a decad refusal stops the work).

- `rotors.go`: `flag(seg)` builds the two bodies of one flag in a local frame (axis on local Z): tips by `decad.Intersect` of
  two octagonal prisms, the rounded-square section, the yellow slab and tips (P1 and P2 with `yellowTipK` 12.95,
  `yellowTipInner` 12.95 and `yellowTipEnd` 11.6) split by `Intersect`/`Cut`, the flat, the notches, the pocket stadium
  sketch and the revolved stubs (instructions.md "Rotor"). Then each body is placed in two steps: a `decad.Transform` by
  the frame origin's local coordinates and `placement.Place` with the rotation-only frame (decad item D3).
- `sliders.go`: `flagSlider(seg)` builds the main profile, the b-f blade extension, the tongue and pin (`rootedTongue`),
  the 0.3 stem chamfers and the groove pin.
- `frame.go`: `cradlePlate()`, `flagTopPlate()` (windows from `windowHalfWidth` 2.95, `windowChamfer` 12.65 and
  `windowEnd` 11.696) and `columnGuidePlate()` with `boreTool`, `rootedBox`, `rootedPost`,
  `mouthCone` and `flagSites()`.
- `params.go`: the exported flag constants and `Dimensions.FlagAngle`.

Checks: `TestSevenSegmentFlag*`, `TestSevenSegmentRotor*`, `TestSevenSegmentSlider*`, `TestSevenSegmentFrame`,
`TestSevenSegmentVisibility`, `TestSevenSegmentCaptive` and the fit and rule tests.

## 13. Fasteners and bearings

### Fasteners

All via `internal/m2fastener` (`PocketTool`, `HoleTool`, `HexPocketTool`, `PlaceBolt`, `PlaceNut`). Seams are modeled
open (0.25); tips end 0.25 inside a face with the seam open and flush when tightened, as in onewayclutch.

| Sites | Joins | Bolt | Head (centre of head top) | Dir | Corner | Cavity | Nut pocket |
| --- | --- | --- | --- | --- | --- | ---: | --- |
| T1-T4 | top plate → top flange ribs | M2 × 10 | (∓23.2, −44.5, 88.25) and (∓23.2, 38.0, 88.25) | (0,0,−1) | (0,1,0) | 10.5 | rib underside z [75.75, 77.75] |
| G1-G4 | ledge ribs → guide plate | M2 × 6 | (∓23.2, −34.2, 49.75) and (∓23.2, 38.0, 49.75) | (0,0,1) | (0,1,0) | 6.25 | guide plate top z [56, 58] |
| D1-D2 | detent plate → front wall | M2 × 18 | (∓21, −37.25, 10) | (0,−1,0) | (0,0,1) | 18.25 | front wall outer face y [−57.5, −55.5] |

Order T1..T4, G1..G4, D1..D2; within T and G: −x front, +x front, −x rear, +x rear.
Tips: T at z 76.25, G at z 57.75, D at y −57.25. Each nut is 1.55 of 1.6 engaged open, fully when closed.
Material round pockets, measured after the 0.5 chamfers: top plate head pockets reach x 25.4 → 1.1 to the
chamfered edge; guide plate hex pockets (flats ±2.174) reach x 25.374 → 1.13; rib pockets 1.026 to the rib's
x = 20 side and 1.626 to x 27; G head pockets in the front ledge ribs end at y −32.0, 1.2 from the rib end (−30.8).
G1/G2 heads are fitted from below between the detent plate's inner face (y −37.25) and cam d's disc
(y ≥ −30.65); nothing occupies z 41-49.75 at x ±23.2 there. No nut is embedded; every pocket opens to a face.

### Bearings

`ballbearing.Build(ctx, doc, ballbearing.Params605ZZ() + Placement)`: front placement origin (0, −49.5, 28),
U = +X, V = +Z (extrudes toward −Y); rear origin (0, 43.0, 28), U = +Z, V = +X (toward +Y). Bearings do not turn
with θ.
Check: `TestSevenSegmentFasteners`.

## 14. Assembly, pose and containers (sevensegment.go)

### Pose and placement

1. Build every part in its frame (above).
2. Camshaft group (shaft, cams, ratchet wheel, collar ring, knob): `decad.Transform` by
   `r3.RotationAround((0,0,28), (0,1,0), θ)`, θ = 36 · Digit.
3. Sliders: translate +3.0 Z where `Lit(Digit, s)`.
4. Every printed body and both bearings: `placement.Place(ctx, body, p.Placement)`. Fastener sites are carried
   through the placement with onewayclutch `placeSite`; bolts and nuts are built from the placed sites with
   `m2fastener.PlaceBolt`/`PlaceNut` and are not placed again.

### Bodies and containers

| Container (parent) | Bodies |
| --- | --- |
| `Frame` (root) | — (unit) |
| `Walls` (Frame) | Front Wall, Rear Wall |
| `Plates` (Frame) | Guide Plate, Cradle Plate, Top Plate |
| `Detent Plate` (Frame) | Detent Plate |
| `Camshaft` (root) | — (unit) |
| `Shaft` (Camshaft) | Shaft |
| `Cams` (Camshaft) | 7 cams |
| `Ratchet Wheel` (Camshaft) | Ratchet Wheel |
| `Collar Ring` (Camshaft) | Collar Ring |
| `Knob` (Camshaft) | Knob |
| `Segment Sliders` (root) | 7 sliders |
| `Flags` (root) | 7 `Flag` units, each with a `Yellow Face` and a `Black Body` |
| `Bearings` (root) | 2 × 605ZZ |
| `Bolts` (root) | 4 × M2 × 10, 4 × M2 × 6, 2 × M2 × 18 |
| `Nuts` (root) | 10 × M2 nut |

60 bodies. Use `NewChild` for units, `ChildNamed` for kinds, `AddBody`; require `doc.VerifyContainers().Err() == nil`.
The body set never changes with `Digit`.
Check: `TestSevenSegmentDefault`, `TestSevenSegmentContainers`,
`TestSevenSegmentAllDigits`, `TestSevenSegmentPlacement`.

## 15. Rule checks and images

Rule tests (fits, corners, edge finishing, look-alike, split shaft) follow instructions.md and
`models/onewayclutch` test files. Images and presets follow docs/rendering.md and the preset list in
`tools/render/sevensegment.go`.

## Wall inset note

Walls: plate rectangle u +-28.5, v -28 to 53.25 (z 81.25); ribs unchanged at |x| 20 to 27. Chamfer both faces' whole
outer loop with `edgefinish.Chamfer(Long(Outer))`. Round each rib root with `decad.Reflex()` edges in one call per rib.
Top plate: y -49.25 to 42.75.
