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
    Sites      [10]m2fastener.Site // T1..T4, G1..G4, H1, H2 in the world frame (before Placement)
}
func (d Dimensions) Station(s Segment) float64             // y_i
func (d Dimensions) GrooveRadius(s Segment, psi float64) float64
func (d Dimensions) Lift(s Segment, theta float64) float64 // GrooveRadius(s, 90+theta) - OffRadius
func (d Dimensions) PressureAngle(s Segment, psi float64) float64

// Click indexer motion (clicker_motion.go)
func PressTurn() float64     // shaft turn at the bottom of a press, derived from the tooth specs
func TimingMargin() float64  // PressTurn() - SegmentsDoneBy
func ReleaseTurn() float64   // 36 - PressTurn()
func PressPose(s float64) (drop, turn float64)
func ReleasePose(s float64) (drop, turn float64)

// Build
const (
    FrameContainerName, WallsContainerName, PlatesContainerName, CamshaftContainerName, ShaftContainerName,
    CamsContainerName, ClickerContainerName, HubContainerName, RotorContainerName, HousingContainerName,
    ButtonContainerName, SpringContainerName, SlidersContainerName, BearingsContainerName,
    BoltsContainerName, NutsContainerName = "Frame", "Walls", "Plates", "Camshaft", "Shaft", "Cams",
    "Click Indexer", "Hub", "Rotor", "Housing", "Button", "Spring", "Segment Sliders", "Bearings", "Bolts", "Nuts"
)
type Parts struct {
    FrontWall, RearWall, GuidePlate, TopPlate *decad.Body
    Shaft                                      *decad.Body
    Hub, Rotor, Housing, Button, Spring        *decad.Body
    Cams, Sliders                                          [7]*decad.Body // indexed by Segment
    Sketches   []*sketch.Sketch
    Dimensions Dimensions
    Sites      [10]m2fastener.Site // placed
}
type Model struct {
    Parts
    Bearings [2]*ballbearing.Model // front, rear
    Bolts    [10]*fastener.Bolt
    Nuts     [10]*fastener.Nut
}
func BuildParts(ctx context.Context, doc *decad.Document, p Params) (*Parts, error)
func Build(ctx context.Context, doc *decad.Document, p Params) (*Model, error)
```

`NewDocument() *decad.Document` makes the display's document with `decad.WithChainTolerance(0.02 mm)`: the face teeth's
Booleans need it (see "Face teeth"). Any other decad failure stops the task.

Exported constants (values above): `SegmentSpan 49.5,
Travel 4.5, FaceZ 132.25, ShaftZ 44.25, StationPitch 16.5,
CamRadius 35.25, CamThickness 5.7, HubRadius 7.5, OffRadius 24.75, OnRadius 29.25, GrooveWidth 5.3, GrooveDepth 2.85,
RampStart 8.0, RampEnd 29.5, RampRound 3.75, EntryAngle 18.0, PinRadius 2.25, PinLength 3.0,
StemThickness 5.4, WallThickness 12.0,
WallHalfWidth 35.1, Seam 0.25,
PressFit 0.05, TightFit 0.25, RegularFit 0.40, LooseFit 0.55, TimingAllowance 1.5, RearCollarRadius 5.7,
ClickerSteps 10, ClickerStep 36, RampAngle 32.5, ToothHeight 8.0, CrownHeight 1.6, CrownBack 30, RotorTravel 2.0,
ButtonStroke 10.0, SpringRestLength 8.0, SegmentsDoneBy 29.55, ChainTolerance 0.02`.

Validation (first failure wins, prefix `sevensegment: `):

1. `digit %d out of range 0..9`
2. `invalid placement frame` (Placement non-nil and `!IsValid()`)

`BuildParts`/`Build` reject nil ctx (`context must not be nil`) and nil doc (`document must not be nil`), derive,
then check `ctx.Err()` before geometry. Every geometry error is wrapped `sevensegment: <part>: <op>: %w`. A body that
is not `IsSolid()` fails `sevensegment: <part> is not a solid`.

Files (package): `params.go` (constants, Segment, truth table, Params, Validate, Derive, Dimensions methods, ramp
math, sites), `sketches.go` (run, sketch helpers, polar helpers, meridian, extrude/profile),
`cams.go`, `shaft.go` (shaft), `clicker.go` (hub, rotor, housing, button, spring), `facetooth.go` (face teeth),
`clicker_motion.go`, `sliders.go`,
`frame.go` (walls, guide plate, top plate), `sevensegment.go` (package doc, containers, Parts, Model, BuildParts,
Build, pose). Tests use testify `require` and `t.Context()`. Tests of the public API and the assembled model are in
the external package `sevensegment_test`. Per-part tests in Tasks 4-8 need the unexported builders before `BuildParts`
exists, so they are internal test files (`*_internal_test.go`, `package sevensegment`), as models/swisslever does;
no per-part builder is exported.

Check: `TestSevenSegmentTruthTable`, `TestSevenSegmentValidate`, `TestSevenSegmentCamProfile`,
`TestSevenSegmentChannel` (params_test.go), `TestSevenSegmentDigitsUpright`, all without geometry.


### Flag API (added with the flag mechanism)

```go
const (
	FaceZ           = 132.25
	RotorAxisZ      = 124.0
	RotorHalfWidth  = 5.625
	RotorHalfLength = 18.375
	JournalRadius   = 0.9 // the rotor's end bore
	PinOffset       = 2.25

	FlagsContainerName      = "Flags"
	FlagUnitName            = "Flag"
	YellowFaceContainerName = "Yellow Face"
	BlackBodyContainerName  = "Black Body"
)

// Flag is one printed two-colour rotor.
type Flag struct{ Yellow, Black *decad.Body }

// Parts gains CradlePlate *decad.Body and Flags [7]Flag (indexed by Segment).

// FlagAngle returns segment s's modeled rotor angle in degrees at camshaft angle theta:
// atan2(Lift(s, theta) - 2.625 + raise_s, PinOffset), raise_s 0 (a, g, d), 0.20 (b, c), 0.25 (e, f).
func (d Dimensions) FlagAngle(s Segment, theta float64) float64
```

## 2. Sketch helpers (sketches.go)

Follow `models/onewayclutch/sketches.go`: `run` holds ctx, doc, params, dimensions and the sketch list;
`sketchOn(frame)`, `profile(s, holes)` (Solve, `sketchcheck.VerifyFullyConstrained`, one valid profile with the
given hole count), `extrude`, `newPolar`/`axis`, `meridian` (points placed with `sketchcheck.DimensionPoint`).
Check: every sketch built in every test is fully constrained.

## 3. Shaft (shaft.go)

### Shaft (`Shaft`, printed, split)

Local frame: origin (0, 0, 44.25), U = +Z, V = +X, N = +Y (local z = world y; local +x = world +Z, the flat normal).

Revolve this meridian `(z, r)` about local Z with `decad.FullRevolution{}` (pattern: onewayclutch `meridian`):

```
(-100.95, 0) (-100.95, 2.2) (-100.65, 2.5) (-84.1, 2.5) (-81.5, 4.0) (-74.0, 4.0)
(-74.0, 4.8) (59.0, 4.8) (60.5585, 5.7) (64.25, 5.7) (64.25, 4.0) (71.2, 4.0) (71.5, 3.7) (71.5, 0)
```

Then cut three flat tools (box prisms in the local frame, extruded symmetric ±6 along local y)
(`flatTool(x0, x1, z0, z1)`: x from x0 to x1):

- Cam flat: `x ≥ 4.05` over `z ∈ [−75.0, 58.5]`; square end at z 58.5 (cam a's D chord stops on it).
- Cam back flat: `x ≤ −4.05` over the same span; square end at z 58.5.
- Stub flat: `x ≥ 2.0` over `z ∈ [−101.95, −86.6]`; square end at z −86.6 (the hub's D chord stops on it,
  0.05 short of the ring's back face at −86.65). No run-out: both flats end square as D-chord stops, listed as
  mating pairs (split-shaft "Corners").

Sections: button stub r 2.5 D (flat 2.0) [−100.95, −86.6]; stub r 2.5 round [−86.6, −84.1] through the front wall's lip
hole; a cone from r 2.5 to the journal's r 4.0 over [−84.1, −81.5] (29.98 degrees; it starts 0.25 outside the front
bearing's face, the shaft's end play, and is not a collar because nothing faces it); front journal r 4.0 round
[−81.5, −74.0] in the front bearing (608ZZ, bore 8, 7 wide: a journal equals the bearing bore); cam section r 4.8
double D (flats ±4.05) [−74.0, 59.0]; rear collar r 5.7 [59.0, 64.25] (a 30 degree cone from the cam section over the first
1.5585, 0.25 inside the rear wall at its end); rear journal r 4.0 round [64.25, 71.5] in the rear bearing. The knob stub
keeps r 2.5 because the click indexer's spring (inside radius 2.9) does not fit a r 4.0 stub. The collar's r 5.7 and the
front step's r 4.8 stay inside the inner ring's face r 6.0 (0.3 and 1.2), clear of the bore chamfer
(4.0 + 0.3 + 0.3 = 4.6). End chamfers 0.3 at both ends. Length 172.45.

The shaft's sketch is anchored at the display's centre plane (y 0), off the axis (`meridianAt`, anchor (2, 0)): the
default anchor, among the shortest sides, leaves this long shaft's constraint system poorly conditioned, and an anchor on
the axis duplicates a constraint.

Failure: return `sevensegment: shaft: <op>: <err>`.
Check: `TestSevenSegmentShaft`, `TestSevenSegmentCollars`.

## 4. Cams (cams.go)

### Cams (`Cams`, 7 bodies)

Local frame per cam: origin (0, y_i + 3.25, 44.25) (the groove face), U = +X, V = +Z, N = −Y. Disc at local
z ∈ [−5.7, 0], hub toward local +z.

1. Revolve meridian `(r, z)`: `(0,−5.7) (34.75,−5.7) (35.25,−5.2) (35.25,−0.5) (34.75,0) (8.0,0)`, then an 8-chord quarter
   round about (8.0, 0.5) from (8.0, 0) to (7.5, 0.5) (pattern: `m2fastener.quarterRound`), then `(7.5,10.75) (0,10.75)`.
   Rim chamfers 0.5 are in the meridian; the hub root round 0.5 is a corner between layers.
2. Cut the double-D bore: profile on local z = −6.7, two arcs r 4.85 joined by flats at v = ±4.1
   (doubleDBoreLoop); extrude 18.45 along +z. The first flat is on local +V (world +Z at θ = 0) on every D bore in
   the model; the cam bores add the opposite flat.
3. Cut the groove tool (next section): profile on local z = +1.0, extrude `decad.Against` 3.85 (to z −2.85).
4. Cut the look-alike holes: circles r 0.75 at radius 18 (front cams) or 12 (rear cams) on angles 270°, 295°,
   320°, 345° (count per cam, "Look-alike parts"), extruded through local z ∈ [−6.7, 11.75]. Each hole has its own
   sketch on the cam frame, centre placed by horizontal and vertical distances from the origin (rule text).

Groove (one sketch per cam; profile with one hole):

- Walls are offsets of the centreline by ±`GrooveWidth/2` = ±2.65. Groove depth `GrooveDepth` 2.85.
- Dwell arcs about the origin: inner wall r − 2.65, outer wall r + 2.65 (OFF: 22.1 / 27.4; ON: 26.6 / 31.9).
- Each ramp is arc–line–arc on the centreline with `RampRound` ρ = 3.75:
  - Rising ramp from ψ_a to ψ_b = ψ_a + 21.5: arc 1 centre `C1 = 28.5∠ψ_a` (outside, r_off + ρ), arc 2 centre
    `C2 = 25.5∠ψ_b` (inside, r_on − ρ), joined by their internal common tangent. With ψ_a = 8°: C1 (28.2226, 3.9664),
    C2 (22.1941, 12.5568), tangent points T1 (24.5361, 4.6533) at 24.973∠10.739°, T2 (25.8806, 11.8699) at
    28.473∠24.638°, line length 7.3408. Unit normal n of the line satisfies `n·(C2 − C1) = 2ρ`, `T1 = C1 + ρn`,
    `T2 = C2 − ρn`, pick the root whose tangent points lie inside (ψ_a, ψ_b) in order (see "Ramp geometry algorithm").
  - Falling ramp: mirror of the rising ramp about the window's mid-angle: `C1 = 25.5∠ψ_a`, `C2 = 28.5∠ψ_b`,
    T1 at 28.473∠(ψ_a + 4.862°), T2 at 24.973∠(ψ_a + 18.761°).
  - Wall radii: an arc whose centre is outside the centreline (away from the origin) has inner wall ρ + 2.65 = 6.4 and
    outer wall ρ − 2.65 = 1.1; an arc whose centre is inside has inner 1.1 and outer 6.4. Lines offset ±2.65.
- Entry channel (outer wall only): radial slot on the ray `EntryAngle` 18.0°, width 5.3 (sides parallel to the ray at
  ±2.65), from the outer wall's ON dwell arc (r 31.9) out to a cap line at distance 36.25 from the origin. It lies in
  the dwell [11.5, 26] of every cam (all ON at digit 8); its sides meet r 31.9 at 18.0 ± 4.77°.
- Sketch constraints (no point in the groove sketch is placed with `DimensionPoint`):
  - Polar construction per onewayclutch `newPolar`/`axis`: a horizontal reference line, one construction ray per
    ramp boundary angle (ψ_a, ψ_b) and one at `EntryAngle`.
  - Each ramp centre: `NewPointOnLine(C, ray)` + `NewDistance(origin, C, d)` (d = 28.5 or 25.5).
  - Each dwell–ramp junction point J (both walls): `NewPointOnLine(J, ray)`; its radius comes from the dwell arc.
  - Each dwell arc: centre = sketch origin; `NewRadius` on the arc (fixes its start); its end radius follows from
    the arc's own equal-radius property. Do not also dimension J's distance (redundant).
  - Each ramp wall arc: centre = the shared ramp centre point; no radius constraint (its start J fixes it).
  - Each ramp wall line: shares end points with its arcs; `NewTangent(line, arc)` at both ends. The tangent points
    are then fixed by tangency plus the arc's equal-radius property.
  - Channel: side lines `NewParallel(entryRay, side)` + `NewDistancePointLine(origin, side, 2.65)`; side's inner end
    is the end point of the split ON dwell arcs; outer ends lie on the cap line (`NewPerpendicular(entryRay, cap)`,
    `NewDistancePointLine(origin, cap, 36.25)`).
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
  max 35.2°.

Cam local→world: frame origin (0, y_i + 3.25, 44.25), U = +X, V = +Z. Printed back face down (groove face and hub up,
axis vertical); no other orientation prints the groove and hub without support.
5. Engrave the letter: draw its outline loops on the back face's plane (horizontal and vertical lines, each held by its distance
   from the axis), take the profile with the counter as its hole, and `decad.Emboss` it engraved 0.5 mm with
   `decad.Faces(decad.Facing(-Z))`, the one planar face looking along local -Z.
Check: `TestSevenSegmentCam` (cams b and e first), `TestSevenSegmentCamInscription`.

## 5. Click indexer (clicker.go, facetooth.go, clicker_motion.go)

Follow instructions.md "Click indexer". The teeth are 8.0 tall (`ToothHeight`) and the button's stroke is 10.0
(`ButtonStroke`). Bodies: `Hub`, `Rotor`, `Housing`, `Button` and the `Spring`
(`compressionspring.Build` with `KSSC1087`, turned half a turn about X and moved onto the shaft). The rotor, hub and
spring turn with the shaft (`spin`); the housing and button do not.

### Face teeth

Decad's `Coil` builds a helicoid ramp, but `Union` refuses a Coil result against a ring, and a `LoftSections` tooth is not an
admitted family. The recipe that builds (all booleans on analytic parts or on one tooth):

1. Make the document with `NewDocument()` (chain tolerance 0.02 mm).
2. Build every ring in a local frame whose Z is the shaft axis and whose angle runs counter-clockwise about +Z. A part
   whose teeth reach toward -Y is placed with `frontFrame` (its angle runs against the shaft's forward turn: shift the
   ring by the negative of the design angle). A part whose teeth reach toward +Y is placed with the frame
   (U = +Z, V = +X, N = +Y): its angle is the forward angle.
3. Tooth 0 is a partial `Revolve` of the rectangle (radii R0..R1, heights Bot..H - Tip) over the angle between the two ramps'
   feet, cut by one raw rising `Coil` (right-handed) and one raw falling `Coil` (left-handed about the axis pointing the
   other way). Each tool spans the radii R0 - 0.3 to R1 + 0.3 and starts or ends one degree outside the blank, so no tool cap
   touches material.
4. Each tool's profile is a plain rectangle, fully constrained (`Sketch.Verify` in every build). Decad's `Coil` chords long straight
   profile lines itself (pin `9fb7f30d8f5e` or later), so the profile edges need no splitting.
5. Copy tooth 0 nine times with `decad.Copy` and `WithCopyPlacement` (a turn of 36k degrees) and `Union` each into the ring's
   plate, one at a time. The plate is a revolve whose top lies `Plate` above the valley and buries the blanks' bottoms.
6. Never let a tooth wall lie on a plate wall: oversize the plate or the tooth ring, so a wall meets a face, not a wall.
   Decad refuses a wall on a wall ("operand facets overlap in one plane"). The button's teeth end at r 10.85 on a head of
   r 11.0; the rotor's plate reaches r 14.2 past the teeth at 14.0 and its bore r 3.0 lies inside the teeth at 3.2; the
   crown's teeth run 0.3 into the tube.
7. Union joins that Decad refuses on a toothed body (the button's keys) are done on the plain plate first.

Lands. A tip is cut back by `Tip` (0.05) and a root stands `Plate` above its valley (0.03 for every ring): the smallest lands that build. They leave the gaps `TestClickerLandGaps` reads and take 0.41 degrees of the press turn.

Check: `TestClickerBodies` (one lump, ten teeth in every ring), `TestClickerHeights`, `TestClickerPoseClearances`,
`TestClickerLandGaps`, `TestClickerPressTurn`, `TestClickerPoses`, `TestClickerRadialFits`, `TestSevenSegmentDigitsUpright`.

## 10. Walls (frame.go)

### Walls (`Walls`: Front Wall, Rear Wall)

Front wall sketch frame origin (0, −74.25, 44.25), U = +X, V = +Z, N = −Y, extrude `Along` 12. Rear wall origin
(0, 64.5, 44.25), U = −X, V = +Z (N = +Y), extrude `Along` 12. Sketch rectangle u ∈ [−36.6, 36.6], v ∈ [−39.75, 77.0]
(z 4.5 to 121.25; the code's `wallHalf` 36.6, `wallBottomZ` 4.5 and `wallTopZ` 121.25): the plate is 73.2 wide and the
ribs end 1.5 inside its edges (|x| 35.1, z 119.75).

1. Extrude the plate; union four rib boxes at |x| ∈ [24.6, 35.1] extending inward from the inner face: ledge ribs
   z [74.75, 80.75], top flange ribs z [113.75, 119.75]; depth 15 except the ledge ribs: front 26.35 (to y −47.9),
   rear 19.7 (to y 44.8).
   Ribs are unioned before any chamfer so no chamfer notch is left where a rib meets the plate rim.
2. Chamfer 0.5 the outer loop of each face (`edgefinish.Chamfer`, `Long(Outer)`, one call for the outer face and one for
   the inner face). The ribs stand 1.5 inside the plate's edges, so they do not cut into the inner face's outer loop.
3. `decad.Fillet` radius 0.5 on every concave rib-root edge in the inner-face plane (ledge: top, bottom and x = ±24.6
   edges; top flange: bottom and x = ±24.6 edges). Pattern: `models/planetarygearbox/cover.go` `roundBossRoot`.
4. Cut the seat/lip revolve tool about the shaft axis. Meridian `(r, d)`, d measured from the inner face (d 0)
   through the wall to the outer face (d 12): `(0,−1) (11.55,−1) (11.55,0) (11.05,0.5) (11.05,7.0) (6.5,7.0) (6.5,11.5)
   (7.0,12.0) (7.0,13.0) (0,13.0)`. Seat r 11.05 (608ZZ `SeatDiameter/2`) 7 deep with a 0.5 lead-in at the inner face,
   lip 5 thick with hole r 6.5 (0.5 outside the inner ring's face radius 6.0) and a 0.5 chamfer at the outer face.
5. Cut fastener tools: ledge ribs `m2fastener.PocketTool` (G sites, head pockets from the rib underside) and
   `HoleTool` (G bolts M2 x 10); top flange ribs `HexPocketTool(site, NutCavityHeight + Stock)` (T sites) and
   `HoleTool`; front wall only: `HexPocketTool` + `HoleTool` at H1, H2 (nut pockets open to the outer face). The T and H
   bolts are M2 x 16.
   Cut the fastener tools before the step 4 seat tool: after the revolve cut, the front wall's G head pockets
   meet a held mesh coarser than their chord tolerance and decad refuses them.
6. Cut look-alike holes (r 0.75, through): front 1 hole, rear 2 holes, on radius 13.5 about the shaft axis at 270°,
   295° (front-view angles), one sketch per hole as for the cams.

Printed outer face down; ribs rise from the plate; rib pockets print sideways with a hex corner up.

## 9. Flags: rotors, sliders, cradle, top plate, guide columns

Operations in build order, each from the instructions.md sections of the same name (every construction was proven in a
one-off scratch build before the port, which is not in the repository; a decad refusal stops the work).

- `rotors.go`: `flag(seg)` builds the two bodies of one flag in a local frame (axis on local Z): tips by `decad.Intersect` of
  two octagonal prisms, the rounded-square section, the yellow slab and tips (P1 and P2 with `yellowTipK` 19.825,
  `yellowTipInner` 19.825 and `yellowTipEnd` 17.4) split by `Intersect`/`Cut`, the flat, the notches, the pocket stadium
  sketch, the two end bores (a revolve with a floor round and a countersink) and their teardrop roofs (a prism on a fully
  constrained pentagon sketch), the 0.9 pocket floor round and the 0.7 mouth chamfer of type B's first notch
  (instructions.md "Rotor"). Then each body is placed in the world with one `placement.Place` of
  the rotor frame `rotorFrame(s)`.
- `sliders.go`: `flagSlider(seg)` builds the main profile, the b-f blade extension, the tongue and pin (`rootedTongue`),
  the 0.5 stem chamfers and the groove pin.
- `frame.go`: `cradlePlate()`, `flagTopPlate()` (windows from `windowHalfWidth` 4.825, `windowChamfer` 19.525 and
  `windowEnd` 17.544) and `columnGuidePlate()` with `boreTool`, `rootedBox`, `rootedPost`,
  `mouthCone` and `flagSites()`.
- `pivot.go`: `pivotBlock()` (the rounded 4.8 x 5.9 x 3.0 body from `roundedPrism`, unioned with the revolved Ø1.4 pin and its
  45-degree root, tip chamfered 0.1), `pivotPocketTool(seg, sgn)` (the same `roundedPrism` offset by `PressFit`, cut from
  the cradle and the top plate by `cutPivotPockets`) and `pivotBlocks()` (the prototype and 13 `decad.Copy`, each placed with one
  `placement.Place` of its rotor frame at the 14 rotor ends). `boreTool` now ends at the end wall at u 18.625 with no journal hole.
- `params.go`: the exported flag constants and `Dimensions.FlagAngle`.

Checks: `TestSevenSegmentFlag*`, `TestSevenSegmentRotor*`, `TestSevenSegmentSlider*`, `TestSevenSegmentFrame`,
`TestSevenSegmentVisibility`, `TestSevenSegmentCaptive` and the fit and rule tests.

## 13. Fasteners and bearings

### Fasteners

All via `internal/m2fastener` (`PocketTool`, `HoleTool`, `HexPocketTool`, `PlaceBolt`, `PlaceNut`). Seams are modeled
open (0.25); tips end 0.25 inside a face with the seam open and flush when tightened, as in onewayclutch.

| Sites | Joins | Bolt | Head (centre of head top) | Dir | Corner | Cavity | Nut pocket |
| --- | --- | --- | --- | --- | --- | ---: | --- |
| T1-T4 | top plate → top flange ribs | M2 × 16 | (∓31.3, −66.75, 132.25) and (∓31.3, 57.0, 132.25) | (0,0,−1) | (0,1,0) | 16.5 | rib underside z [113.75, 115.75] |
| G1-G4 | ledge ribs → guide plate | M2 × 10 | (∓31.3, −51.3, 74.75) and (∓31.3, 48.0, 74.75) | (0,0,1) | (0,1,0) | 10.25 | guide plate top z [85, 87] |
| H1-H2 | housing → front wall | M2 × 16 | (∓21, −92.5, 44.25) | (0,1,0) | (0,0,1) | 16.25 | front wall inner face y [−76.25, −74.25] |

Order T1..T4, G1..G4, H1..H2; within T and G: −x front, +x front, −x rear, +x rear.
Tips: T at z 114.25, G at z 86.75, H at y −74.5. Each nut is 1.55 of 1.6 engaged open, fully when closed.
Material round pockets, measured after the 0.5 chamfers: top plate head pockets reach x 33.5 → 1.1 to the
chamfered edge; guide plate hex pockets (flats ±2.174) reach x 33.474 → 1.13; rib pockets 4.526 to the rib's x = 24.6
side and 1.626 to x 35.1; G head pockets end 1.2 from the front ledge rib end (y −49.1 against −47.9) and 1.0 from the
rear one (y 45.8 against 44.8). G1/G2 go in from below in front of cam d's disc (y ≥ −46.25), G3/G4 between the discs of
cams f (y ≤ 41.95) and a (y ≥ 52.75); no cam stands under a G head. No nut is embedded; every pocket opens to a face.

### Bearings

`ballbearing.Build(ctx, doc, ballbearing.Params608ZZ() + Placement)`: front placement origin (0, −74.25, 44.25),
U = +X, V = +Z (extrudes toward −Y); rear origin (0, 64.5, 44.25), U = +Z, V = +X (toward +Y). Bearings do not turn
with θ.
Check: `TestSevenSegmentFasteners`.

## 14. Assembly, pose and containers (sevensegment.go)

### Pose and placement

1. Build every part in its frame (above).
2. Camshaft group (shaft, cams, hub, rotor, spring): `decad.Transform` by
   `r3.RotationAround((0,0,44.25), (0,1,0), θ)`, θ = 36 · Digit.
3. Sliders: translate +4.5 Z where `Lit(Digit, s)`.
4. Every printed body and both bearings: `placement.Place(ctx, body, p.Placement)`. Fastener sites are carried
   through the placement with onewayclutch `placeSite`; bolts and nuts are built from the placed sites with
   `m2fastener.PlaceBolt`/`PlaceNut` and are not placed again.

### Bodies and containers

| Container (parent) | Bodies |
| --- | --- |
| `Frame` (root) | — (unit) |
| `Walls` (Frame) | Front Wall, Rear Wall |
| `Plates` (Frame) | Guide Plate, Cradle Plate, Top Plate |
| `Camshaft` (root) | — (unit) |
| `Shaft` (Camshaft) | Shaft |
| `Cams` (Camshaft) | 7 cams |
| `Hub`, `Rotor`, `Housing`, `Button`, `Spring` (Click Indexer) | one body each |
| `Segment Sliders` (root) | 7 sliders |
| `Flags` (root) | 7 `Flag` units, each with a `Yellow Face` and a `Black Body` |
| `Bearings` (root) | 2 × 608ZZ |
| `Bolts` (root) | 6 × M2 × 16, 4 × M2 × 10 |
| `Nuts` (root) | 10 × M2 nut |

75 bodies, with `Pivot Blocks` (root) holding the 14 blocks. Use `NewChild` for units, `ChildNamed` for kinds, `AddBody`; require `doc.VerifyContainers().Err() == nil`.
The body set never changes with `Digit`.
Check: `TestSevenSegmentDefault`, `TestSevenSegmentContainers`,
`TestSevenSegmentAllDigits`, `TestSevenSegmentPlacement`.

## 15. Rule checks and images

Rule tests (fits, corners, edge finishing, look-alike, split shaft) follow instructions.md and
`models/onewayclutch` test files. Images and presets follow docs/rendering.md and the preset list in
`tools/render/sevensegment.go`.

## Wall inset note

Walls: plate rectangle u +-36.6, v -39.75 to 77.0 (z 4.5 to 121.25); ribs at |x| 24.6 to 35.1. Chamfer both faces' whole
outer loop with `edgefinish.Chamfer(Long(Outer))`. Round each rib root with `decad.Reflex()` edges in one call per rib.
Top plate: y -74.0 to 64.25.
