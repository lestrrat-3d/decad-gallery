# Seven-segment display instructions

A cam-driven, 3D-printable seven-segment digit. Seven grooved face cams on one camshaft raise and lower seven
two-colour rotating flags, and a push-button click indexer (a rotor and a fixed crown, as in a pen) turns the shaft one
step per press and locks it at every digit, so every rest position shows exactly one valid digit 0-9. The box stands on
its back wall with the button end up. Units are millimetres and degrees. Inputs are `Digit` and `Placement` only.

## Inputs and validation

- `Digit` 0..9 (default 2) sets the camshaft angle `theta = 36 * Digit` and every slider's height.
- `Placement` is nil (the world frame below) or a valid frame.
- Validation, first failure wins, prefix `sevensegment: `: `digit %d out of range 0..9`, then `invalid placement frame`.

## Assumptions

- The box stands on its back wall (the stance of the stack below, with model -Y up); the display face looks toward the viewer. A press of the button turns the shaft one digit.
- Display face as modeled is horizontal (face up, table top) for the Z stack below. Each segment is a two-colour flag that turns 90 degrees: a yellow
  face when lit, a black face when unlit. Every printed body is one filament colour (printed on a multi-filament printer).
- Digit glyphs: 6 and 9 with tails (6 lights a, 9 lights d); 7 lights a, b, c only.
- Input is a button pressed 6.0 mm along the shaft axis, one digit per press 0→1→…→9→0. The segments' roles are swapped against the usual order (a and d, b and e, c and f) so each digit reads upright with the button end up.
- Purchased parts: two 608ZZ bearings (`ballbearing.Lookup("608ZZ")`), M2 bolts and nuts (`fastener` via
  `internal/m2fastener`), and one KSSC K series 1087 compression spring (`compressionspring.KSSC1087`). No
  purchased shafts.
- Click indexer assumptions: a press needs 5.64 to 7.53 N of spring force plus the ramps' friction, and the cams' share. At
  Scale 1.0 that was friction 3 to 6 N, peak about 11 to 14 N, and up to 41 N.mm at the shaft, about 5.6 N of press. At Scale 1.5
  the cams and sliders take up to 201.1 N.mm at the shaft while five segments change together, about 14.1 N of press on an
  ideal ramp (Cam follower drive), so a press is about 20 to 22 N and up to about 35 N with friction. The spring is a
  purchased KSSC K series 1087 from MonotaRO, order code 01353223 (`compressionspring.KSSC1087`). The supplier's
  allowable load is 7.15 N at 6.4 mm length, so the pressed length of 6.0 mm is 0.4 mm past it, and fatigue life is
  not published.
- Fit margin convention follows `onewayclutch`: `TightFit 0.25`, `RegularFit 0.40`, `LooseFit 0.55`,
  `PressFit 0.05`, `Seam 0.25` (clamped seam, closed when bolts are tight).
- Inputs are `Digit` and `Placement` only. The exported constants and `Dimensions` fields are exactly those in
  the exported API in steps.md; every other dimension in this doc is an unexported constant.
- Go is written by the implementer directly from steps.md. The fresh-emitter flow of model-pipeline step 3
  (`scripts/modelgen`) exists only for spur and is not used here.
- The sides of the frame stay open (no side panels) so the cams and sliders show, as the escapements do.
- Containers follow the nesting rule: units `Frame` and `Camshaft` hold their kinds.
- Model README name and `models/README.md` row title: "Seven-segment display".

## Scale

`Scale` (`params.go`) is the size of the display against its base design. It is 1.5. Unless a line says "at Scale 1.5", the
lengths, heights and volumes in the sections below are the base design's, at `Scale` 1.0; multiply a shape by 1.5 and leave its
fits, print limits, whole-layer thicknesses and purchased hardware alone. "Scale 1.5 values" below lists what is not a plain
product. Every shape (the layout, the rotors, the sliders, the cams, the frame and the pivot blocks) is written in
the code as `Scale` times its base size in mm. These keep their size in mm at any scale, and the code adds them outside
the product:

- Fits and clearances (`TightFit`, `RegularFit`, `LooseFit`, `PressFit`, `Seam`, the 0.2 journal clearance, the 0.25 end gap).
- Print limits: the 0.8 shells and tube walls, the tip flats, and the 0.5 edge chamfer and inside-corner round.
- Plate thicknesses that are whole layers (the cradle and top plates).
- Purchased hardware: the M2 bolts and nuts, the bearings and the click indexer's spring, with the click indexer's size.

A value that is a size plus a fit, such as the rotor bore `RB` (swept circle 5.096 plus 0.45), is written as
`Scale*size + fit`, so a change of `Scale` moves the size and leaves the fit.

### Scale 1.5 values

What differs from a plain 1.5 times the base design, and why:

| Item | Scale 1.0 | Scale 1.5 | Why |
| --- | --- | --- | --- |
| Cradle / top plate thickness | 2.6 / 5.4 | 4.0 / 8.0 | whole layers (1.5 x 2.6 is 3.9, 1.5 x 5.4 is 8.1) |
| Rotor axis `RotorAxisZ`, display face | 82.6, 88.25 | 124.0, 132.25 | cradle bottom 120.0 + 4.0, + 0.25 seam + 8.0; every height of the flag mechanism hangs from them |
| Wall thickness | 8.0 | 12.0 | 1.5 x 8.0 |
| Bearings | 2 x 605ZZ (Ø5 bore, 5 wide) | 2 x 608ZZ (`ballbearing.Lookup("608ZZ")`: bore 8, outside 22, 7 wide, inner ring face 12) | a Ø5 journal cannot scale; the catalog lists no bore chamfer, so a local 0.3 is assumed (`bearingBoreChamfer`) |
| Bearing seat | r 7.05, 5 deep, lip hole r 4.5 | r 11.05, 7 deep, lip hole r 6.5, lip 5.0 thick | seat = outside radius + 0.05; lip hole 0.5 outside the inner ring's face radius 6.0 |
| Shaft | journal r 2.5, cam section r 3.2, rear collar r 3.8 | journal r 4.0, cam section r 4.8, collar r 5.7, knob stub r 2.5 | journal = bearing bore; the click indexer keeps its size, and its spring (inside radius 2.9) does not fit a r 4.0 stub |
| Knob stub step | none | cone r 4.0 to 2.5 over y -81.5 to -84.1 (29.98 degrees) | inside the front wall's lip hole; it starts 0.25 outside the bearing's face (the shaft's end play); it is not a collar (nothing faces it) |
| Front step, rear collar | r 3.2 / 3.8 | r 4.8 / 5.7, rear collar 5.25 long from the cone's start | inside the inner ring's face r 6.0 (1.2 and 0.3), clear of the bore chamfer (4.0 + 0.3 + 0.3 = 4.6) |
| Pivot block | 3.2 x 3.1 x 2.0, pin Ø0.8 in a Ø1.2 bore | 4.8 x 5.9 x 3.0, pin Ø1.4 in a Ø1.8 bore | study option B; the block's height is `2*(cradleThick - 1.0 - PressFit)`, so 1.0 of cradle stays under the pocket floor; bore roof truncated at 1.00 |
| Bolts | T M2 x 10, G M2 x 6, H M2 x 10 | T M2 x 16, G M2 x 10, H M2 x 16 | standard lengths (`BoltSize.StandardLengths`); cavities follow from the plate stacks |
| Housing base thickness | 4.0 | 6.0 | the H bolts: no standard length fits a 12 mm wall with a 4.0 base (tip 12 is short of the nut, 18 stands out of the wall) |
| Click indexer | teeth 4.0, stroke 6.0 | teeth 8.0, stroke 10.0 | design change: the cams' load grows 5.1 times in torque, the spring and the bought parts do not, so the ramps are made half as steep |
| Wall look-alike hole circle | r 10 | r 13.5 | 1.5 x 10 is under the housing flange (r 14.55 to 24.8) |
| Cam hub length | 7.15 | 10.75 | pitch less cam thickness less the 0.05 stack gap |
| Window | 5.9 wide, lip 0.704 / 0.487 | 9.65 wide, lip 1.512 / 1.248 | the 0.8 inset from the rotor's faces and 1.1 from its pyramid edges do not scale; the lip is over the 0.8 minimum, so the exception for it is dropped |

Checks at Scale 1.5, all from the tests:

- Rest bands (slot clearance 0.20, `PinLinePlay` 0.469 unchanged): a, g, d design unlit [-59.60, -41.76], lit [37.85, 56.63];
  worst case unlit end -41.01, lit start 37.02, pin radius 3.832 (limit 4.15 asserted); b to f design unlit end -42.37, lit
  [37.16, 56.09]; worst case -40.12 and 34.55, pin radius 3.954 against the slot's limit 4.25.
- Drive: ratio 1.423 (a, g, d) and 1.186 (b to f) at friction 0.35 (needs 1.40 and 1.17); lock friction 0.426 and 0.385 (needs
  0.42 and 0.38), with the stem's vertical edges chamfered 0.5. Peak shaft torque 201.1 N.mm at step 6 to 7; the cams and sliders weigh 3.375 times as much (`followerLoad` x 3.375).
- Rotor centre of mass 1.029 (A) and 1.265 (B) from the axis, at the same angles; the rest bias passes at 1.50 (A) and 1.75 (B).
- Tongue sweep, turned over: tongue to the yellow shell 0.843 (b to f 0.832), to the black leg 2.325 (1.945); pin to the pocket 0.943 (0.932).
- Click indexer: press turn 32.09 degrees, release turn 3.91, timing margin 2.54 against 29.55; ramp rate 4.0625 degrees per mm;
  the press share of the cams on an ideal ramp is 201.1 N.mm x 32.09 degrees / 8.0 mm = 14.1 N, plus the spring's 5.64 to 7.53 N.
  Nobody has measured it, and a printed 1.5x camshaft's dwell drag, and the torque at which a r 2.5 printed stub breaks, are not known.
- The crown's long ramp is 12.8 degrees, under the friction angle (19.3 degrees at friction 0.35). A rotor let go early therefore
  does not slide back to the old digit by the spring alone, at either scale; "Detent and rest angles" below says it does, and is wrong.
- Surface edge finishing: at 1.5 two planar faces of every slider (the stem's side faces, 402 to 562 mm2) reach 400 mm2 and carry the 0.5
  chamfer on every long run of their outer loops (`edgefinish.ChamferRuns`), made before the groove pin is added. The stem's vertical
  edges are among those runs, so the 0.3 bevels of Scale 1.0 are now 0.5 (`stemChamfer`), because Fusion cannot chamfer an edge that
  already carries a smaller bevel. a, g and d build and pass; b to f wait on a decad fix (the tongue's flank is a Faceted face 0.0006 to
  0.0027 mm inside the stem's print face instead of merging with it).

## Design

### Coordinates and layout

World frame (the model's local frame before `Placement`):

- Origin under the digit centre, 3.0 below the walls' lower edges. +Z up. +Y toward the digit's top (segment a), −Y toward the viewer
  and the button end (segment d). +X to the right seen from the button side.
- Shaft axis: line `x = 0, z = ShaftZ (29.5)`, along +Y.
- Front view = looking along +Y from −Y: +X right, +Z up. Clockwise in the front view is right-hand rotation about +Y.
- Camshaft angle `θ`, degrees, right-hand about +Y (clockwise seen from the button end). Digit k rests at `θ = 36k`.
- A part frame with `U = +X, V = +Z` has normal `−Y`; "cam-local angle ψ" is measured in such a frame from +U toward
  +V. A right-hand turn by θ about +Y moves material at cam-local angle ψ to ψ − θ, so the follower at the top
  (ψ = 90°) reads cam-local angle `ψ = 90 + θ`. Digit k is under the follower at `ψ_k = 90 + 36k`.

Stations (slider centre plane, Y): `y_i = −33 + 11 i` for i = 0..6 in the order d, e, c, g, b, f, a:
d −33, e −22, c −11, g 0, b 11, f 22, a 33.

Stack along Y (world):

| Part | Y from | Y to | Notes |
| --- | ---: | ---: | --- |
| Button | −81.4 | −74.95 | head r 11, 23.9 above the wall at rest; flush with the bezel (−75.4) at the bottom of its 6.0 stroke (Scale 1.5: −118.15 to −107.65, 31.9 above the wall, flush with the bezel at −108.15 after 10.0) |
| Housing | −75.4 | −57.75 | bezel at −75.4; base r 24.8 on the wall, 0.25 (Seam) from its outer face |
| Rotor | −75.0 | −63.9 | slides 2.0 toward the wall |
| Hub | −67.3 | −57.9 | keyed hub; 0.4 (Regular) to the front wall's outer face |
| Spring | −69.4 | −61.4 | KSSC 1087 |
| Shaft stub flat end | −57.85 | | square end of the stub flat; the hub's D chord stops here (0.05) |
| Front wall | −57.5 | −49.5 | 8 thick; lip [−57.5, −54.5], seat [−54.5, −49.5] |
| Front bearing 605ZZ (608ZZ at 1.5) | −54.5 | −49.5 | outer race in the seat |
| Shaft front step | −49.25 | | d 6.4 → 5, 0.25 (Tight) from the bearing's inner face |
| Cam i hub | y_i − 4.8 | y_i + 2.35 | r 5.0; front end 0.05 from cam i−1 |
| Slider i | y_i − 1.8 | y_i + 1.8 | stem; pin to y_i + 3.85 |
| Cam i disc | y_i + 2.35 | y_i + 6.15 | groove on the −Y face; 0.55 (Loose) to the slider |
| Shaft flat end | 39.2 | | square ends of the two flats; cam a's chords stop here |
| Rear collar | 39.7 | 42.75 | cone 39.7→40.739 (r 3.2→3.8), cylinder r 3.8 |
| Rear bearing 605ZZ (608ZZ at 1.5) | 43.0 | 48.0 | |
| Rear wall | 43.0 | 51.0 | seat [43.0, 48.0], lip [48.0, 51.0] |

Cam a disc is [35.35, 39.15], cam d disc [−30.65, −26.85], cam d hub [−37.8, −30.65].

Stack along Z:

| Part | Z from | Z to |
| --- | ---: | ---: |
| Cams (r 23.5 about z 29.5) | 6.0 | 53.0 |
| Walls | 3.0 | 81.25 |
| Ledge ribs (guide plate seats) | 49.75 | 53.75 |
| Guide plate | 54.0 | 58.0 |
| Top flange ribs | 75.75 | 79.75 |
| Cradle plate | 80.0 | 82.6 |
| Top plate (display face at 88.25) | 82.85 | 88.25 |
| Slider pin centre (OFF / ON) | 46.0 / 49.0 | |

X: the walls span x ∈ [−24.9, 24.9]; the guide plate, cradle plate and top plate span x ∈ [−23.4, 23.4].

Overall: 49.8 (X) × 108.5 (Y, frame [−57.5, 51.0]) × 85.25 (Z, z [3.0, 88.25]), the button end 23.9 more toward −Y.

Ribs (all at |x| ∈ [16.4, 23.4], part of the walls): front ledge ribs y [−49.5, −30.8], front top flange ribs
y [−49.5, −39.5], rear ledge ribs y [28.8, 43.0], rear top flange ribs y [33.0, 43.0]. The guide-plate bolts go in
from below beside the cams: G1/G2 at y −34.2 in front of cam d's disc, G3/G4 at y 32.0 between the discs of cams f
and a. No cam lies under a G head.

### Digit geometry

`SegmentSpan` S = 33 is the pitch of the segment centres: a, g, d at (0, 33), (0, 0), (0, −33) with the long axis X, and b, c, e, f at
(±16.5, ±16.5) with the long axis Y. Each segment is a two-colour rotor in a window of the top plate ("Rotor" below).

### Truth table and ramp table

Lit segments (1 = ON):

| Digit | a | b | c | d | e | f | g |
| ---: | - | - | - | - | - | - | - |
| 0 | 1 | 1 | 1 | 1 | 1 | 1 | 0 |
| 1 | 0 | 1 | 1 | 0 | 0 | 0 | 0 |
| 2 | 1 | 1 | 0 | 1 | 1 | 0 | 1 |
| 3 | 1 | 1 | 1 | 1 | 0 | 0 | 1 |
| 4 | 0 | 1 | 1 | 0 | 0 | 1 | 1 |
| 5 | 1 | 0 | 1 | 1 | 0 | 1 | 1 |
| 6 | 1 | 0 | 1 | 1 | 1 | 1 | 1 |
| 7 | 1 | 1 | 1 | 0 | 0 | 0 | 0 |
| 8 | 1 | 1 | 1 | 1 | 1 | 1 | 1 |
| 9 | 1 | 1 | 1 | 1 | 0 | 1 | 1 |

Groove centreline radius `R_s(ψ)` (cam-local ψ): `OffRadius 16.5` where the segment is OFF, `OnRadius 19.5` where ON.
Lift = `R − 16.5`. The transition k→k+1 (when the state changes) is a ramp over `ψ ∈ [ψ_k + RampStart, ψ_k + RampEnd]`
= `[ψ_k + 8.0, ψ_k + 29.5]`. Everything else is dwell. Dwell around digit k: `[ψ_k − 6.5, ψ_k + 8.0]` (14.5°).

Ramp windows (cam-local, degrees, mod 360):

| Step | Window | Step | Window |
| --- | --- | --- | --- |
| 0→1 | [98, 119.5] | 5→6 | [278, 299.5] |
| 1→2 | [134, 155.5] | 6→7 | [314, 335.5] |
| 2→3 | [170, 191.5] | 7→8 | [350, 11.5] (wraps) |
| 3→4 | [206, 227.5] | 8→9 | [26, 47.5] |
| 4→5 | [242, 263.5] | 9→0 | [62, 83.5] |

Ramps per cam (R rising OFF→ON, F falling ON→OFF):

| Cam | Ramps | Count |
| --- | --- | ---: |
| a | 0→1 F, 1→2 R, 3→4 F, 4→5 R | 4 |
| b | 4→5 F, 6→7 R | 2 |
| c | 1→2 F, 2→3 R | 2 |
| d | 0→1 F, 1→2 R, 3→4 F, 4→5 R, 6→7 F, 7→8 R | 6 |
| e | 0→1 F, 1→2 R, 2→3 F, 5→6 R, 6→7 F, 7→8 R, 8→9 F, 9→0 R | 8 |
| f | 0→1 F, 3→4 R, 6→7 F, 7→8 R | 4 |
| g | 1→2 R, 6→7 F, 7→8 R, 9→0 F | 4 |

Rising ramp lift vs. angle after its window start (shape from "Cams"; falling is the mirror):

| ψ − ψ_k | 8 | 9 | 10 | 12 | 14 | 16 | 18 | 20 | 22 | 24 | 26 | 28 | 29 | 29.5 |
| --- | -- | -- | -- | -- | -- | -- | -- | -- | -- | -- | -- | -- | -- | -- |
| lift | 0 | 0.019 | 0.078 | 0.297 | 0.555 | 0.842 | 1.161 | 1.514 | 1.904 | 2.335 | 2.743 | 2.954 | 2.995 | 3.000 |

Max pressure angle 35.2°. Max slope 13.38 mm/rad. Lift differs from its dwell by more than 0.05 only for
`ψ − ψ_k ∈ [9.61, 27.93]` (the "active zone").

### Detent and rest angles

The click indexer (see Click indexer) is the detent: it locks the shaft at every digit angle, and a press turns the shaft
one step. Nothing in the camshaft is a stiff detent, so there is no climbing torque to budget. What the indexer must
guarantee instead is timing: by the end of a press the shaft has turned far enough that every cam has finished moving its
segment, so the new digit shows on the downstroke.

- Press turn: `PressTurn()` = 32.5 degrees (the ramp) less what the tooth lands take, 4.0625 degrees per mm of height
  (8.0 mm teeth; 8.125 and 4.0 mm at Scale 1.0) times (tip cut + the larger of the mating tip cut and root land): 0.41, so
  32.09 degrees (31.69 at Scale 1.0).
- Cams finished: `SegmentsDoneBy` 29.55 (the cam ramp end `RampEnd` 29.5 plus 0.05 of play).
- Timing margin: `TimingMargin()` = 32.09 - 29.55 = 2.54 degrees (2.14 at Scale 1.0). `TestClickerPressTurn` derives it from the tooth specs.
- Release turn: 36 - 32.09 = 3.91 degrees, driven by the fixed crown's short ramp. It never turns the shaft back.
- Early release: if the button is let go before the rotor passes the crown's tip (30 degrees), the rotor is on the crown's
  long ramp, 12.8 degrees at r 13.4. That is under the friction angle (19.3 degrees at friction 0.35), so the spring alone does
  not slide it back to the old digit: the rotor stays where it stopped on the ramp, and a second press finishes the step. This
  corrects the earlier claim that it slides back.
- The stub's D-bore play stays closed: the rotor drives the shaft forward in every phase and nothing turns it back.

### Camshaft

All rotating parts below are built in their own local frame, axis on local +Z through the local origin, every
Boolean done there, then moved by one `decad.Transform` to the world (local→world below), then turned by θ about the
shaft axis, then `placement.Place`. Reason: `internal/placement` documents Booleans failing on bodies built at an
offset (gear tooth unions). Sliders, walls and plates are prisms with simple cuts and unions, built in
world coordinates as onewayclutch builds its plates and ring at z offsets; if one of their Booleans refuses, move
that part to a local frame the same way before reporting a decad failure.

#### Shaft (`Shaft`, printed, split)

Local frame: origin (0, 0, 29.5), U = +Z, V = +X, N = +Y (local z = world y; local +x = world +Z, the flat normal).

Revolve this meridian `(z, r)` about local Z with `decad.FullRevolution{}` (pattern: onewayclutch `meridian`):

```
(-72.2, 0) (-72.2, 2.2) (-71.9, 2.5) (-49.25, 2.5) (-49.25, 3.2) (39.7, 3.2)
(40.739, 3.8) (42.75, 3.8) (42.75, 2.5) (47.7, 2.5) (48.0, 2.2) (48.0, 0)
```

Then cut three flat tools (box prisms in the local frame, extruded symmetric ±4 along local y):

- Cam flat: `x ≥ 2.7` over `z ∈ [−50.25, 39.2]`; square end at z 39.2 (cam a's D chord stops on it).
- Cam back flat: `x ≤ −2.7` over the same span, square end at z 39.2 (cam a's second chord stops on it).
- Stub flat: `x ≥ 2.0` over `z ∈ [−73.2, −57.85]`; square end at z −57.85 (the hub's D chord stops on it,
  0.05 short of the hub's back face at −57.9). No run-out: both flats end square as D-chord stops, listed as
  mating pairs (split-shaft "Corners").

Sections: button stub r 2.5 D (flat 2.0) [−72.2, −57.85]; front journal r 2.5 round [−57.85, −49.25] through the lip
hole and the front bearing; cam section r 3.2 double D (flats ±2.7) [−49.25, 39.2]; rear collar r 3.8; rear journal r 2.5
round in the rear bearing. End chamfers 0.3 at both ends. Length 120.2.

Failure: return `sevensegment: shaft: <op>: <err>`.

#### Cams (`Cams`, 7 bodies)

Local frame per cam: origin (0, y_i + 2.35, 29.5) (the groove face), U = +X, V = +Z, N = −Y. Disc at local
z ∈ [−3.8, 0], hub toward local +z.

1. Revolve meridian `(r, z)`: `(0,−3.8) (23,−3.8) (23.5,−3.3) (23.5,−0.5) (23,0) (5.5,0)`, then an 8-chord quarter
   round about (5.5, 0.5) from (5.5, 0) to (5.0, 0.5) (pattern: `m2fastener.quarterRound`), then `(5.0,7.15) (0,7.15)`.
   Rim chamfers 0.5 are in the meridian; the hub root round 0.5 is a corner between layers.
2. Cut the double-D bore: profile on local z = −4.8, two arcs r 3.25 joined by flats at v = ±2.75
   (doubleDBoreLoop); extrude 12.95 along +z. The first flat is on local +V (world +Z at θ = 0) on every D bore in
   the model; the cam bores add the opposite flat.
3. Cut the groove tool (next section): profile on local z = +1.0, extrude `decad.Against` 2.9 (to z −1.9).
4. Cut the look-alike holes: circles r 0.75 at radius 12 (front cams) or 8 (rear cams) on angles 270°, 295°,
   320°, 345° (count per cam, "Look-alike parts"), extruded through local z ∈ [−4.8, 1.0]. Each hole has its own
   sketch on the cam frame, centre placed by horizontal and vertical distances from the origin (rule text).

Groove (one sketch per cam; profile with one hole):

- Walls are offsets of the centreline by ±`GrooveWidth/2` = ±1.9. Groove depth `GrooveDepth` 1.9.
- Dwell arcs about the origin: inner wall r − 1.9, outer wall r + 1.9 (OFF: 14.6 / 18.4; ON: 17.6 / 21.4).
- Each ramp is arc–line–arc on the centreline with `RampRound` ρ = 2.5:
  - Rising ramp from ψ_a to ψ_b = ψ_a + 21.5: arc 1 centre `C1 = 19.0∠ψ_a` (outside, r_off + ρ), arc 2 centre
    `C2 = 17.0∠ψ_b` (inside, r_on − ρ), joined by their internal common tangent. With ψ_a = 8°: C1 (18.8151, 2.6443),
    C2 (14.7960, 8.3712), tangent points T1 (16.3574, 3.1022) at 16.649∠10.739°, T2 (17.2538, 7.9133) at
    18.982∠24.638°, line length 4.8939. Unit normal n of the line satisfies `n·(C2 − C1) = 2ρ`, `T1 = C1 + ρn`,
    `T2 = C2 − ρn`, pick the root whose tangent points lie inside (ψ_a, ψ_b) in order (see "Ramp geometry algorithm").
  - Falling ramp: mirror of the rising ramp about the window's mid-angle: `C1 = 17.0∠ψ_a`, `C2 = 19.0∠ψ_b`,
    T1 at 18.982∠(ψ_a + 4.862°), T2 at 16.649∠(ψ_a + 18.761°).
  - Wall radii: an arc whose centre is outside the centreline (away from the origin) has inner wall ρ + 1.9 = 4.4 and
    outer wall ρ − 1.9 = 0.6; an arc whose centre is inside has inner 0.6 and outer 4.4. Lines offset ±1.9.
- Entry channel (outer wall only): radial slot on the ray `EntryAngle` 18.0°, width 3.8 (sides parallel to the ray at
  ±1.9), from the outer wall's ON dwell arc (r 21.4) out to a cap line at distance 24.5 from the origin. It lies in
  the dwell [11.5, 26] of every cam (all ON at digit 8); its sides meet r 21.4 at 18.0 ± 5.09°.
- Sketch constraints (no point in the groove sketch is placed with `DimensionPoint`):
  - Polar construction per onewayclutch `newPolar`/`axis`: a horizontal reference line, one construction ray per
    ramp boundary angle (ψ_a, ψ_b) and one at `EntryAngle`.
  - Each ramp centre: `NewPointOnLine(C, ray)` + `NewDistance(origin, C, d)` (d = 19.0 or 17.0).
  - Each dwell–ramp junction point J (both walls): `NewPointOnLine(J, ray)`; its radius comes from the dwell arc.
  - Each dwell arc: centre = sketch origin; `NewRadius` on the arc (fixes its start); its end radius follows from
    the arc's own equal-radius property. Do not also dimension J's distance (redundant).
  - Each ramp wall arc: centre = the shared ramp centre point; no radius constraint (its start J fixes it).
  - Each ramp wall line: shares end points with its arcs; `NewTangent(line, arc)` at both ends. The tangent points
    are then fixed by tangency plus the arc's equal-radius property.
  - Channel: side lines `NewParallel(entryRay, side)` + `NewDistancePointLine(origin, side, 1.9)`; side's inner end
    is the end point of the split ON dwell arcs; outer ends lie on the cap line (`NewPerpendicular(entryRay, cap)`,
    `NewDistancePointLine(origin, cap, 24.5)`).
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

Cam local→world: frame origin (0, y_i + 2.35, 29.5), U = +X, V = +Z. Printed back face down (groove face and hub up,
axis vertical); no other orientation prints the groove and hub without support.

### Click indexer

The box stands on its back wall, so model -Y points up and the knob end is the top. A push button on the end of the shaft
turns it one digit per press, like a pen's click mechanism. Five bodies and a spring, from the front wall up (Y in mm
at digit 0 and rest; up is -Y):

| Part | Container | Axis range (Y) | What it is |
|---|---|---|---|
| Hub | `Hub` | -67.3 to -57.9 | Keyed hub, pressed on the stub's D flat (replaces the collar ring). Base 3.5 long with the D bore (r 2.55, flat 2.05), a cup wall r 9 to 10.5 up to the rim at -67.3, three keys r 8.05 to 9.5, 2.0 wide. The rim is the rotor's lower stop. |
| Spring | `Spring` | -69.4 to -61.4 | KSSC K series 1087, MonotaRO order code 01353223 (`compressionspring.KSSC1087`), 8.0 mm installed at rest (5.64 N, 0.9408 x (14.0 - 8.0)), 6.0 pressed (7.53 N, 0.9408 x (14.0 - 6.0)). The supplier's allowable load is 7.15 N at 6.4 mm length, so the pressed length is 0.4 mm past it; fatigue life is not published. Its outside diameter is 7.0 mm; the skirt's bore leaves 0.55 mm radial clearance to it. It stands on the hub's base and pushes the rotor's plate up. |
| Rotor | `Rotor` | -75.0 to -63.9 | Skirt r 4.05 to 8.45 with three slots that take the hub's keys (Tight 0.2 a side), a plate r 3 to 14.2, inner teeth r 3.2 to 11 (4.0 tall) that meet the button, outer teeth r 12.8 to 14 (1.6 tall) that lock in the crown. Slides 2.0 mm along the axis. |
| Housing | `Housing` | -75.4 to -57.75 | Base flange r 14.55 to 24.8, 4.0 thick, on the front wall (two M2x10 bolts H1, H2 at x +-21, heads sunk in the base, nuts in hex pockets open to the wall's inner face); tube r 14.55 to 16.55; fixed crown ring r 11.55 to 14.55 with ten notches; three slots r 11.05 to 12.7, 3.0 wide, for the button's lugs. Its top, the bezel, is 17.9 above the wall. |
| Button | `Button` | -81.4 to -74.95 | Head r 11.0, 2.5 thick, with a recess r 3.2 and 1.4 deep under it (the stub's tip is at -72.2); ten teeth under it, r 3.4 to 10.85, 4.0 tall; three lugs r 11.4 to 12.2, 2.0 wide, hanging 3.8 below the head's underside. |

The head stands 23.9 above the front wall at rest and 17.9 at the bottom of its 6.0 mm stroke, flush with the bezel (at Scale 1.5, 31.9 and 21.9, stroke 10.0, teeth 8.0, the housing base 6.0 thick, the lugs 7.8 below the head; all click indexer offsets from the front wall move with the wall: its outer face is at −86.25).

Teeth. Every ring has ten sawtooth teeth with helicoid ramps about the shaft axis. A tooth rises from its sharp valley
over `Rise` degrees to its peak and falls over `Fall` degrees; `Rise + Fall` is 36.

| Ring | Height | Rise / Fall | Tip cut | Root land | Radii |
|---|---|---|---|---|---|
| Rotor inner | 4.0 | 3.5 / 32.5 | 0.05 | 0.03 | 3.2 to 11 |
| Button | 4.0 | 3.5 / 32.5 | 0.05 | 0.03 | 3.4 to 10.85 |
| Rotor outer | 1.6 | 30 / 6 | 0.05 | 0.03 | 12.8 to 14 |
| Crown | 1.6 | 30 / 6 | 0.05 | 0.03 | 12.8 to 14.85 (runs 0.3 into the tube) |

Tooth tips are engagement edges and tooth roots are retained sharp corners (see Retained sharp corners).

One press:

1. Rest: the spring holds the rotor up, its outer teeth in the crown's notches. The digit is locked.
2. The first 2.0 mm: the button pushes the rotor down, clear of the crown, onto the hub's rim stop.
3. The next 8.0 mm (4.0 at Scale 1.0): the ramps turn the rotor 32.09 degrees (31.69), and the hub and the shaft with it.
4. Release: the spring lifts the rotor onto the crown's short ramp, which turns it the last 3.91 degrees (4.31) into the next notch.

Land gaps. Where a tip meets a root the lands leave a gap (`TestClickerLandGaps`): 0.02 mm between the rotor's outer teeth
and the crown at rest, and 0.02 mm between the rotor's inner teeth and the button at the bottom of the press. The ramps
themselves touch: the turn the lands cost is the 0.81 degrees above.

Build. Decad cannot join a helicoid wedge to a ring, so each tooth is cut once and copied (steps.md, "Face teeth"):
a partial revolve blank is cut by one rising and one falling `Coil` tool, the tooth is copied nine times, and the copies
are united into the ring's plate one at a time. The document is made with `NewDocument()`, whose chain tolerance is 0.02 mm.
Each tool's profile is a plain rectangle: Decad's `Coil` (pin `9fb7f30d8f5e`) chords long straight lines itself.

Printing. Rotor: flat, teeth up. Button: head down, teeth up. Housing: bezel face down, crown teeth up, base last. Hub: flat.
Face teeth are printed without supports: the ramps lie at 45 to 60 degrees to the layers and the notch walls are vertical.

## Flag mechanism

### Requirements

Each segment shows its state by colour: a flag that rotates between a yellow face (lit) and a black face (unlit).
Every printed body keeps one filament colour. The drive must not lock at any pose up to friction 0.5. The flag must be
captive in every orientation and must rest only in two bands. The click indexer's timing margin must stay valid, and the part count
must stay small.

Acceptance tests:

- Ray test on the real meshes (visibility_test.go) at elevations 90, 45 and 30 degrees, azimuths across and along each flag, in
  the physical (clamped, settled) geometry, over the rest bands at the design play:
  - lit, azimuths across the flag (+-s), rays that enter the window at `|u| <= 9.9` (flag-local u of the entry
    point): yellow is at least 65 % of those whose first hit is a flag body, and the flag is at least 90 / 55 / 20 %
    of them at 90 / 45 / 30 degrees. The computed worst values are 70 %, 100 / 65 / 25 %, so each threshold has a
    margin of 5 points. Azimuths along u and the whole-window share (estimated 59-79 %) are reported, not asserted;
  - whole window (`TestSevenSegmentWindowYellow`, rays on a 0.05 grid over the window opening at 90 degrees): the lit
    yellow share is at least 0.95 at phi 45, 0.82 at the lit band ends 34.47 (a, g, d) and 33.41 (b to f) and 0.76 at the
    b-f worst-case end 29.39 (the results are in the log of the test; the design computed 0.98, 0.85 to 0.87 and 0.80);
  - unlit: no ray hits a yellow body at the design-play band end (a, g, d -40.33; b to f -41.22), the worst-case end
    (-39.23; -37.90) and a 1 degree margin beyond it (-38.23; -36.90), with the rotor centred, 0.16 or 0.25 to either
    side and 0.30 along the axis either way, from elevations 90, 45 and 30 at four azimuths; at the margin poses with the
    0.25 side shift also from elevations 60 and 20 at 24 azimuths, grazing views included. Yellow first shows, for the
    onset harness (yellow sample points, views facing above 5 degrees), at -33.84 (0.16 side) and -35.25 (0.25 side);
    with grazing views counted the onset stops at -36.48 for every window width from 2.80 to 3.00, because the face then
    lies edge-on. The margins to the worst unlit band end at a 0.25 side shift with grazing views are 1.42 degrees (b to
    f) and 2.75 (a, g, d).
- Drive: at friction 0.5 the true drive arm (the pin on one wall of its slot) is at least 1.50 times the friction arms with
  the rotor settled on its pins and 1.35 with it fully across (computed 1.554 and 1.382).
- Timing: a press turns the shaft `PressTurn()` 32.09 degrees (31.69 at Scale 1.0), 2.54 past the 29.55 by which every cam has finished
  (`TimingMargin()`, derived from the tooth specs in `TestClickerPressTurn`). This is calculated on the built
  dimensions and not measured on a print.
- Captive at every digit, upright and turned over; pressing the button in either orientation meets no contact other than
  the designed ones.
- `go test ./models/sevensegment`, `go -C tools/render test -run SevenSegment .` and
  `python3 scripts/check_repository.py` pass.

**Out of scope:** cams (grooves, entry channel, letters), shaft and walls keep their geometry. The
click indexer replaced the knob, ratchet wheel, collar ring and detent plate and is specified
above. Fusion export, any build status change, and the two decad fixes (they belong to the decad repo) are out
of scope.

## Assumptions

- Friction PETG on PETG: 0.35 nominal, 0.5 worst. The drive is checked at 0.5; the budget at every friction up to the
  guarantee.
- PETG density 1.27 g/cm^3. Layer height 0.2: the cradle (2.6) and the top plate (5.4) are whole
  numbers of layers, so the seam plane, which carries the rotor axis, lies on a layer boundary.
- Seams S1 are modeled open (0.25) and closed when the bolts are tight. "Physical" means clamped: the cradle sits 0.25
  lower than modeled, the top plate 0.50 lower. The guide plate does not move (G bolts, no seam). Gravity settles each
  rotor to the bottom of its pivot bores (0.20 round the pins) and each slider onto its groove's inner wall (0.40 below the modeled
  pose). Each b-f slider droops to its tilt limit (Kinematics).
- Printed dimensions hold +-0.05 on radii and +-0.10 on lengths over 10 mm (the basis of the z play row).
- Known printability risk (geometry unchanged): the black rim of a rotor in layer 1 is about 0.3 mm wide at the sides,
  below a 0.4 mm line width, so a slicer may drop it and print that layer's rim as yellow only. The rim widens to 0.8 mm
  at mid height, where it prints. Check the first layer of the first print before the flags are used.
- Slicer: each flag is one object of two bodies (`Yellow Face`, `Black Body`) exported together, one filament each. The
  bodies share their boundary faces. The seven flags print on one plate, yellow face down: layers 1-12 hold yellow (the slab
  in layers 1-4, then two small yellow islands per rotor under the facets) and a black rim, so a plate takes about 24
  filament changes (about 8 without the tips).
- Split shafts: the rule covers printed shafts and pins. These are integral features of larger parts and stay whole:
  the pivot pins (Ø1.4 at Scale 1.5, Ø0.8 at 1.0) of the 14 pivot blocks, the slider rotor pins (Ø1.5), the slider groove pins (Ø4.5), and the
  registration pegs (Ø4.5) on the top plate and on the guide plate columns. None carries a part, a bearing or torque.
  Splitting any of them means splitting its whole parent part. Each is below the 4 mm size that the rule asks for (the Ø1.4 pivot pin joins the integral-pin exemption: nothing under the bias
  cap, a pin of Ø1.6 at most, reaches 4 mm); the
  pivot pin is listed with the slider's rotor pin as an integral pin (it stands upright in print, on its block, so the
  layers do not run along it). Loads: rotor pin 0.020 N, groove pin up to 0.8 N (Cam follower drive), pivot pins 0.012 N
  (up to 0.03 N while the pin lifts the rotor), pegs only locate (the M2 bolts clamp the plates). The rotor itself was
  accepted as exempt in review.
- Pivot pin strength. A pin stands upright on its block and bends across the layer bonds. At Scale 1.0 (Ø0.8): 0.012 N on a 1.25
  lever, 0.30 MPa against about 20 MPa layer bond, a factor of 67 (37 at 0.03 N on 0.9), breaking at about 0.8 N of finger force.
  At Scale 1.5 (Ø1.4, 1.875 lever, load 3.375 times as large, 0.0405 N): 0.28 MPa, a factor of 71, breaking at about 2.9 N. The pin can lift the rotor while it rises, near phi -30:
  the drive and slot friction then push up harder than the rotor's weight, so the pin is loaded on every side at some
  point of the swing (0.014 N on rotor A, 0.013 N on B).
- Pin strength. The rotor pin is Ø1.0 and 3.6 long. On the a/g/d slider it stands upright in print, so bending loads
  the layer bonds; on b-f it also points up. At 0.040 N (the larger F_x of the drive at friction 0.5), `M = 0.144 N.mm` and
  `sigma = 1.5 MPa` against about 20 MPa layer bond: a factor of 13, about 7 with a sharp root. The pin breaks at about 0.53 N. Nothing in the drive
  limits the force when a rotor jams: the hand on the button can exceed 0.53 N, and the pin then breaks before the rotor,
  the cradle or the cam. The pin is the intended weak link; the fix is reprinting one slider (about 1.2 g).
- The unit's sides (x = +-23.4) are open (only the front and rear walls exist), so the b-f arms can be reached from the
  side during assembly.

## Approach

Each segment is a square-section two-colour **rotor** (7.5 x 7.5, long edges rounded 0.5, pyramid tips to +-12.25)
on two printed **pivot blocks**, one at each end. Each block is pressed into a square pocket split between a new
**cradle plate** and the rewritten **top plate**, and a pin (Ø0.8 at Scale 1.0, Ø1.4 at 1.5) on its rotor-side face turns in a bore (Ø1.2, Ø1.8) in the
rotor's end. The rotor's top face shows through a window in the top plate. The yellow is a 0.8 slab on the face that
continues onto the 45 degree facet at each end (0.8 inside the facet's edges, to `|u| = 11.6`), so it follows the
window's pointed ends; it is a separate yellow body and the rest is black.

The slider keeps its stem, groove pin and cam; its cap is replaced by a narrow **tongue** that rises into a notch in the
rotor's hidden underside. A Ø1.0 **pin** on the tongue, parallel to the rotor axis, runs in a **radial slot** (a blind
pocket) of the rotor, on a vertical line 1.5 from the rotor axis. The cam's 3.0 lift turns the rotor through 90 degrees.
A **guide blade** on each slider runs in a Tight **tube** that hangs from the top plate. The plate that holds the
rotor's pivot blocks therefore also holds the blade line. Two **columns** on the guide plate carry pegs that register the
cradle; two pegs on the top plate register it to the cradle. The stem's Tight guide slot and the tube then set the
slider's tilt, which gravity holds at a fixed limit.

The slot is radial, so the contact normal is tangential and acts at the radius `sqrt(rho^2 - 0.25^2)` (0.98 to 2.73).
Sliding friction runs along the slot wall, with an arm of only the slot half-width 0.80. The net arm grows toward the
rests; no pose locks.

Alternatives considered (friction 0.5):

- **Scotch yoke on the slider:** net arm `R cos g - mu (R |sin g| + r_p)` reaches zero at g = 41.6. It locks at the
  rests.
- **Rack and pinion:** pitch radius 1.91 gives 1.2 teeth over the travel. Not printable.
- **Bell crank and link, or a 180-degree flag with a gear stage:** 7-14 more bodies and printed pin joints under Ø1.5.
- **7 mm rotor:** slider yaw, droop and z tolerances move the pin line by about +-0.5, which pushes the pin past the slot
  end and shows yellow when unlit.
- **Fork on a hub on the rotor axis:** the fork must reach 3 above the hub and hits the rotor at lit.

Body count: 75 (45 before the flags): the cradle plate (1), seven rotors of two bodies each (14) and the 14 pivot blocks
are added. Blades, tubes,
columns and pegs are features of existing bodies. Distinct printed segment parts: 3 slider types (a/g/d, b/c, e/f;
today 5) and 2 rotor types (A for a, g, d; B for b, c, e, f).

## Design

### Coordinates and frames

World frame as in `instructions.md`. Segment centres `(cx, cy)` from `segmentCentre`: a (0, 33), g (0, 0), d (0, -33),
b (16.5, 16.5), c (16.5, -16.5), e (-16.5, -16.5), f (-16.5, 16.5). Stations `y_i` (`Dimensions.Station`): a 33, b 11,
c -11, d -33, e -22, f 22, g 0.

- Rotor axis through `(cx, cy, RotorAxisZ)`, `RotorAxisZ = 82.6` (modeled; clamped 82.35; settled 82.1).
- `u` = axis direction: a, g, d +X; b, c, e, f +Y. `s = Z x u`: a, g, d +Y; b, c, e, f -X.
- Rotor frame `(x_r, y_r, u)`: at phi = 0, `x_r` = s and `y_r` = Z; phi is a right-hand turn about u. Polar angle beta
  from `x_r`. Diagonals `a = (x_r + y_r)/sqrt2` (beta 45) and `b = (y_r - x_r)/sqrt2` (beta 135).
- Faces: yellow normal beta 45 (a = 3.75), black beta 135 (b = 3.75), hidden 225 and 315. Pocket ray beta 0. Bottom
  flat normal beta 270.

### Z stack (modeled; physical in brackets)

| Item | Z |
| --- | --- |
| Guide plate (position unchanged, does not move) | 54-58 |
| Columns (5 x 5 front, 4.6 x 5 rear, 45-degree roots 0.5) / column pegs (Ø3.0) | 58-79.15 / 79.15-81.75 |
| Cradle plate (2.6 = 13 layers) | 80.0-82.6 [79.75-82.35] |
| Rotor axis | 82.6 [82.35 clamped, 82.1 settled] |
| Top plate (5.4 = 27 layers) / tubes / top-plate pegs (Ø3.0) | 82.85-88.25 / 77.4-82.85 / 81.25-82.85 [-0.5] |
| Tube roof (blade-hole top) | 87.45 [86.95] |
| `FaceZ` | 88.25 [87.75] |
| Rotor swept circle bottom | 77.504 [77.004] |
| Slider groove pin axis (unlit / lit) | 46.0 / 49.0 [-0.4] |
| Head or arm top (unlit / lit) | 72.8 / 75.8 [-0.4] (73.8 before the flags) |
| Rotor pin axis, a/g/d (unlit / lit) | 80.85 / 83.85; b/c +0.20, e/f +0.25 (droop raise) |
| Blade ridge (unlit / lit) | 83.55 / 86.55 [-0.4] |

Gaps at lit: head or arm top to the rotor's swept circle 1.70 [1.60]; tube bottom to head or arm top 1.60 [1.50];
blade root round top (76.4) to tube bottom 1.0; blade ridge to tube roof 0.90 [0.80]. Blade engagement in its tube at
unlit: ridge 6.15 [6.25], full width 5.55. Columns to cradle 0.85 [0.60]: the columns only locate, they carry nothing.

The head top moved down 1.0 because the turned-over sweep (Kinematics) showed the old 73.8 top entering the black
shell when a lit slider rises 1.28 (b-f: 1.88).

### Kinematics and play (params.go)

```
PinOffset s0 = 1.5
modeled pin height z_m(L) = L - 1.75 + raise_s     relative to the modeled axis; L = Lift(s, theta) in [0, 3]
raise_s = 0 (a, g, d), 0.20 (b, c), 0.25 (e, f)    the unlit droop less 0.05 (RaiseBelowDroop), built in
FlagAngle phi_m(L) = atan2(L - 1.75 + raise_s, 1.5)
  a/g/d -49.40 (L 0) / +39.81 (L 3); b/c -45.94 / +44.03; e/f -45.00 / +45.00
physical rest: slider -0.40, rotor axis -0.50, b-f droop -(raise_s + 0.05)
               -> pin relative to the settled axis: -1.65 (unlit) / +1.35 (lit); dwell offset c = 0.15 toward unlit
                  (a, g, d) and 0.20 (b to f, the pin 0.05 below its droop)
d phi / d z = s/rho^2: 16 deg/mm at unlit, 22.5 deg/mm at lit (0.05 mm of lift = 1.15 degrees)
```

Droop. Gravity (weight couple 0.09 N.mm against 0.009 N.mm of groove-pin friction) turns a b-f slider about its groove
pin to the tilt that its blade (0.25 across its hole) and its stem (0.25 in its slot) allow:
`theta = 0.5 / (blade ridge - 54)` = 0.0169 unlit (ridge 83.55), 0.0154 lit (86.55). The pin drops `theta * x_pin`
(15.0 b/c, 18.0 e/f): 0.254 / 0.305 unlit, 0.230 / 0.276 lit. `raise_s` is the unlit drop less 0.05, so the b-f unlit pins
sit 0.05 lower than a/g/d (further from lit: this buys the unlit margin, see Visibility) and the lit pins 0.02-0.03 higher.
The blade is not raised. Turned over, the tilt reverses and the pin rises by twice the droop (0.5 / 0.6).

Pin-line play at a rest:

| Source | s (+-) | z (+-) |
| --- | ---: | ---: |
| blade across its hole (0.25, see "Blade corners") | 0.25 | - |
| top plate to cradle (pegs, 0.05) | 0.05 | - |
| yaw (the stem, 7.0 wide in its Tight slot, limits it to 0.071 rad) | 0.08 | - |
| rotor on its pins, settled (bore r 0.6, pin r 0.4, 0.20 clearance: 0.20 sin(atan 0.5)) | 0.09 | - |
| cam inner-wall radius, groove-pin radius | - | 0.05 + 0.05 |
| slider length, groove pin to rotor pin | - | 0.10 |
| cradle seat (rib top), bore-to-pin clearance's printing error | - | 0.05 + 0.05 |
| cradle thickness (13 whole layers) | - | 0 |
| b-f droop limit (fits +-0.05 each, registration +-0.1, over 29.55, at x_pin 18) | - | 0.12 |
| total used | 0.47 | 0.24 |

Root-sum-square of z: 0.14 (a/g/d), 0.19 (b-f); 0.24 is used. Worst-case sums: 0.30 (a/g/d), 0.42 (b-f).

Rest bands. The drive leaves every rotor at its leading slot wall, and gravity holds it there: the centre of mass sits
0.712 toward beta 92.0 (A) and 0.909 toward beta 94.0 (B). Its torque is `W_r e |cos(beta_c + phi)|`. Against static
journal friction at friction 0.5 (`mu' 0.6 W_r`) it is:

| Pose | A | B |
| --- | ---: | ---: |
| worst case, a/g/d z 0.30: -39.23 / 33.20 | 1.61 / 1.53 | - |
| worst case, b-f z 0.42: -37.90 / 29.39 | - | 1.89 / 1.86 |

Bands (`bands()` in `flags_test.go`, slot clearance 0.20, c 0.15 for a, g, d and 0.20 for b to f):

```
lit   phi = atan(z/s) + asin(0.20/rho),  unlit phi = atan(z/s) - asin(0.20/rho)
a/g/d design (s 0.47, z 0.24, c 0.15): unlit [-66.7, -40.33], lit [34.47, 63.12], pin radius <= 2.73
a/g/d worst (z 0.30):                  unlit end -39.23, lit from 33.20, pin radius <= 2.77
b-f design (z 0.24, c 0.20):           unlit end -41.22, lit [33.41, 62.42]
b-f worst (z 0.42, c 0.20):            unlit end -37.90, lit from 29.39, pin radius <= 2.89 (the slot's limit is 2.90)
```

The pin's greatest radius is 2.73 (a, g, d) and 2.77 (b to f) at the design play, and 2.77 and 2.89 at the worst cases,
against the slot's limit of 2.90 (`SlotEnd` 2.7 plus the pin's 0.20 in the slot). Only the b-f worst case comes within
0.01 of it, so the pin may rest on the slot end there. That contact is radial and puts no torque on the rotor. The
visibility test finds no yellow at the b-f worst-case unlit end -37.90 or 1 degree beyond it at -36.90 (the old estimate
of -37.0 for where yellow first shows came from a section calculation, not from the meshes).

If the bias failed, the rotor could sit at the lagging wall, 13.5 degrees further from the end pose. The design does not
cover that case; the bias margin above is its guard, and visibility_test.go reports it.

### Rotor (`rotors.go`; 7 rotors, types A and B)

Build in a local frame (local x = x_r, local y = y_r, local z = u, origin on the axis at the rotor's middle), then one
`decad.Transform` to world, a turn by `FlagAngle` about u, and `placement.Place` (pattern: `cams.go`). Each body is
placed in two steps, the translation `(o.U, o.V, o.N)` (the frame origin's coordinates in the local frame) and then
`placement.Place` with the rotation-only frame at the world origin: the same rigid motion as one `placement.Place`, but
decad refuses the single placement of rotor B's black body once the yellow has tips (item D3, below). Booleans retire
their operands, so take `decad.Copy` before any operand is used twice.

| Name | Value | Note |
| --- | ---: | --- |
| `RotorHalfWidth` (exported) | 3.75 | face distance from the axis |
| `rotorEdgeRound` | 0.5 | long edges; swept radius `3.25*sqrt2 + 0.5 = 5.096` |
| `rotorSquareHalf` | 10.0 | square section for `|u| <= 10.0` |
| `RotorHalfLength` (exported) | 12.25 | end faces 3.0 square; pyramid edges sharp (convex 120 degrees) |
| `JournalRadius` (exported) | 0.6 | the rotor's end bores, floor at u 10.8, 0.5 round at the floor, teardrop roof, 45-degree mouth to r 0.93 |
| `pivotPinRadius` | 0.4 | the block's pin; bore less pin 0.20 |
| `rotorFlat` | -2.3 | keep `y_r >= -2.3` |
| `notchFloor` | 2.95 | shells under both faces 0.8; notch end chamfers 0.5; apex chamfer 0.8 |
| `slotStart`, `slotEnd`, `slotRadius` | 1.0, 2.7, 0.7 | stadium on beta 0; the pin (r 0.5) has 0.20 in it; pin limit rho 2.90 |
| `yellowDepth`, `yellowHalfWidth` | 0.8, 2.95 | slab only |
| `yellowTipK`, `yellowTipInner`, `yellowTipEnd` | 12.95, 12.95, 11.6 | tips: chamfer `\|b\| + \|u\| <= 12.95`, inner face `a >= 12.95 - \|u\|`, end at `\|u\| = 11.6` |

Notches and pockets (local u). A notch spans `[tongue_min - 2.1, tongue_max + 1.7]`; the pocket is 3.0 deep. The u play
is 1.0: blade 0.35, registration 0.05, yaw at the pin up to 0.36 (a/g/d) or 0.30 (b-f), rotor 0.25.

| Type | Segments | Tongue u | Notch u | Pocket u |
| --- | --- | --- | --- | --- |
| A | a, g, d | [-3.5, -1.9] | [-5.6, -0.2] | [-0.2, 2.8] |
| B | b, e (first); c, f (second) | [-7.3, -5.7]; [3.7, 5.3] | [-9.4, -4.0]; [1.6, 7.0] | [-4.0, -1.0]; [7.0, 10.0] |

Construction, in this order (proven in one-off scratch builds that are not in the repository; `TestSevenSegmentRotor` and
`TestSevenSegmentRotorAll` check the result):

1. Tips. Prism A: octagon in the plane of a and u, points `(a, u)`:
   `(4.25, -9.5) (4.25, 9.5) (1.5, 12.25) (-1.5, 12.25) (-4.25, 9.5) (-4.25, -9.5) (-1.5, -12.25) (1.5, -12.25)`,
   `decad.Symmetric{D: 14, FullLength: true}` along b. Prism B: the same in the plane of b and u with its end points at
   `(+-1.4, +-12.35)`, symmetric 14 along a. `tips = Intersect(A, B)` (1568.46).
2. Section: rounded square on local z = -13, extruded 26. Four lines, each held by
   `sketch.NewDistancePointLine(origin, line, 3.75)` and `sketch.NewAngle(reference, line, angle)` (angles 45, 135,
   225, 315), then `sketch.CreateFillet` 0.5 at each vertex. The sketch verifies fully constrained.
   `rotor = Intersect(tips, section)` (1252.29). Intersecting the section with A and then B is refused; this order is
   not.
3. Colour split. P1 in (a, u): `(2.95, -10) (2.95, 10) (1.35, 11.6) (6, 11.6) (6, -11.6) (1.35, -11.6)`, symmetric 14
   along b. Its inner face `a = 12.95 - |u|` runs parallel to the facet `a = 13.75 - |u|`, 0.8 below it, and meets the
   slab floor `a = 2.95` at `|u| = 10`. P2 in (b, u): `(2.95, -10) (2.95, 10) (0.85, 12.1) (-0.85, 12.1) (-2.95, 10)
   (-2.95, -10) (-0.85, -12.1) (0.85, -12.1)`, symmetric 14 along a: the chamfer `|b| + |u| <= 12.95` runs parallel to the
   pyramid edges, 0.8 inside them. `T = Intersect(P1, P2)`. `Yellow = Intersect(Copy(rotor), Copy(T))` (105.41),
   `Black = Cut(rotor, T)`. The yellow ends at `|u| = 11.6`: at
   11.6 the inner face is at `a = 1.35`, so the yellow stays 0.75 clear of the pivot bore (r 0.6). The window
   shows the facet only to 11.696, so black shows from 11.6 to 11.696 at the very point. The earlier yellow that followed the
   end facets all the way made type B's first notch cut fail ("mesh bound 1.58"); this one builds for both types.
4. Black only: cut the flat box `y_r <= -2.3` (all u).
5. Black only, each notch `[u0, u1]`: tool = `Intersect(Intersect(SA, SB), apex box)`. SA is the profile in (a, u)
   `(2.95, u0+0.5) (2.95, u1-0.5) (2.45, u1) (-6, u1) (-6, u0) (2.45, u0)`, symmetric 14 along b; SB is the same
   profile in (b, u), symmetric along a. The apex box keeps `y_r <= (2*2.95 - 0.8)/sqrt2 = 3.606` over `[u0-1, u1+1]`.
   A plain `Cut` (tool 414.76).
6. Black only, each pocket: stadium sketch on local z = pocket start - 0.6, extruded 3.6 (16.05). Centres
   `DimensionPoint(1.0, 0)` and `DimensionPoint(2.7, 0)`; arc points `(1, 0.7) (1, -0.7)` and `(2.7, -0.7) (2.7, 0.7)`;
   `CreateArc(c0, (1,0.7), (1,-0.7))`, `CreateArc(c1, (2.7,-0.7), (2.7,0.7))`; lines between them; constraints
   `NewRadius(arc0, 0.7), NewEqualRadius(arc0, arc1)` and `NewTangent` of both lines to both arcs (pattern
   `models/handcrankfan/crank.go` `crankArm`). `decad.Fillet` 0.5 on its far-end convex loop (pocket floor). Cut.
7. Black only, each end's pivot bore (end sgn = +1 for +u, mirrored for -u): cut the revolved bore, then the teardrop
   roof.
   - Bore: revolve the meridian `(r, u)` `(0, 10.8) (0.1, 10.8)`, an 6-chord quarter round of 0.5 about `(0.1, 11.3)` to
     `(0.6, 11.3)`, `(0.6, 11.92)` and the 45-degree countersink to `(0.93, 12.25)`, continued to `(1.13, 12.45)` and
     `(0, 12.45)`. The floor is at u 10.8 and the bore wall is 0.6 in radius.
   - Roof: a prism along u from 10.8 to 12.45 on a fully constrained sketch in a frame whose V axis is the rotor-local
     direction `(-1, -1)/sqrt2` (up in print: the rotor prints yellow face down). The section is the pentagon `(0, 0)
     (t, t) (h, 0.70) (-h, 0.70) (-t, t)` with `t = 0.6/sqrt2` and `h = 0.6 sqrt2 - 0.70`: 45-degree sides tangent to
     the bore, cut flat 0.70 from the axis (0.30 wide). Its lower corner is on the axis, so no side lies at the bore's own
     height. It adds the gable above the circle: a teardrop with a bridge of 0.30.
   The rotor loses 9.7 mm^3 against the stubs it had.

Results: type A 856.22 mm^3 in all (yellow 105.41; 0.01067 N at 1.27 g/cm^3), centre of mass 0.712 toward beta 92.0. Type B
655.90 (0.00817 N), 0.909 toward beta 94.0 (the smaller pin pocket leaves more material than the 864.00 / 661.75 before it). After step 3 yellow + black = 1252.29 = rotor. Each body
tessellates at 0.01.

`cornercheck` per body, print axis -n_yellow:
- yellow: no finding;
- black: four turns of 154-158 degrees at `(a 3.35, b +-2.55, u +-10.4)`, where the black rim meets the yellow chamfer in
  the section at the slab's mid height. They lie on the boundary that black shares with yellow; the printed object has
  the plain facet line there. Exempt them as shared boundary (radius 0.25).
- black: no other finding, with no exemption for the bore. The sampled heights are -3.35, -1.125 and +2.225 from the axis
  (the roof's flat adds the level that moves the middle one; it keeps the 0.4 height, where a Ø1.2 blind bore reads as a
  0.89 wide slot whose end cannot read under 50 degrees, out of the sample). Two rounds were deepened so that the heights
  read clean: the pin pocket's floor round is 0.6 (it read 52.8 degrees at 0.5), and the first notch of type B has a 0.7 mouth
  chamfer on its u0 end where it meets the pyramid slant on face 315 (a 45-degree wedge, 135-degree tip, read 135.0).

Print: yellow face down. Layers 1-4 hold the yellow slab with a black rim (0.3 at the sides in layer 1, 0.8 at mid height); layers 5-12 hold two small yellow islands under the facets at the tips. Notch
floors, end chamfers and the apex chamfer face up or at 45 degrees. Each end's Ø1.2 bore is horizontal in print: its roof
is a 45-degree teardrop with a 0.30 flat, a bridge, and the countersink is at 45 degrees. No support is needed. The
pin pocket's floor round (0.6) and the notch mouth chamfer (0.7 at 45 degrees) are the only rounds changed from the stub design.

Walls: slot to the yellow face and to hidden face 315: 1.11. Slot end to the beta-0 edge round: 1.70. Slot to the bottom
flat: 1.5. Pin at its limit to the notch's yellow floor: 0.21. Notch shells and the edge strip: 0.80.

### Sliders (`sliders.go`, rewritten above the guide plate)

Build pose = modeled unlit, groove pin on the groove centreline (z 46.0); lit is +3.0. Kept: groove pin (`r.pin`), stem
foot z 42.5, `StemWidth` 7, `StemThickness` 3.6, guide slot fit (Tight 0.25). Changed: the stem's vertical edges get a
0.3 `decad.Chamfer` (a 0.3 `Fillet` is refused on these edges). Removed: the old cap, post and their constants. Head or arm top 72.8.

All three types, in this order (proven in one-off scratch builds; `TestSevenSegmentSliders` checks the result and every sketch
verifies fully constrained):

1. Main profile (stem, head or arm, blade) as one sketch, extruded. Sketch fillets with `sketch.CreateFillet`, each
   followed by `NewHorizontal`/`NewVertical` on its two lines (`polygon()` now draws these lines).
2. b-f only: blade extension, a polygon in X-Z on y = y_i + 1.7, extruded +Y 1.3 (0.1 overlap), unioned.
3. Tongue and pin (`rootedTongue`, below), unioned.
4. `decad.Chamfer` 0.3 on the stem's four convex vertical edges (those that start at z 42.5).
5. Groove pin `r.pin(y_i + 1.8)`, unioned.

a/g/d (u = +X, s = +Y; print on the -X face). Profile in Y-Z on x = -3.5, extruded +X 7.0, points `(y - y_i, z)`:
`(-1.8, 42.5) (1.8, 42.5) (1.8, 71.15) (2.4, 71.75) (2.4, 72.8) (-6.7, 72.8) (-6.7, 82.95) (-7.25, 83.55) (-7.8, 82.95)
(-7.8, 72.8) (-8.6, 72.8) (-8.6, 70.8) (-1.8, 70.8)`. Fillets 0.6 at `(-6.7, 72.8)` and `(-7.8, 72.8)` (blade roots),
1.0 at `(-1.8, 70.8)` (extension root). The blade is 1.1 thick, `s [-7.8, -6.7]`, 7.0 long, with a 45-degree gable
(ridge 83.55). The head step at 71.15-71.75 is a 45-degree turn.

b/c (u = +Y, s = -X; print on the -Y face). Profile in X-Z on y = y_i - 1.8, extruded +Y 3.6:
`(-3.5, 42.5) (3.5, 42.5) (3.5, 70.3) (17.9, 70.3) (17.9, 72.8) (9.8, 72.8) (9.8, 82.95) (9.25, 83.55) (8.7, 82.95)
(8.7, 72.8) (-3.5, 72.8)`; fillets 1.25 at `(3.5, 70.3)`, 0.6 at `(9.8, 72.8)` and `(8.7, 72.8)`. Blade extension
polygon `(8.7, 72.8) (9.8, 72.8) (9.8, 82.95) (9.25, 83.55) (8.7, 82.95)`.

e/f: `(-3.5, 42.5) (3.5, 42.5) (3.5, 72.8) (-8.7, 72.8) (-8.7, 82.95) (-9.25, 83.55) (-9.8, 82.95) (-9.8, 72.8)
(-19.5, 72.8) (-19.5, 70.3) (-3.5, 70.3)`; fillets 0.6, 0.6 at the blade roots and 1.25 at `(-3.5, 70.3)`. Blade
extension polygon mirrored. Every b-f blade spans y `[y_i - 1.8, y_i + 3.0]` (4.8 long).

`rootedTongue(uAxis, ua, ue, sL, sR, sc, zp)`: the tongue runs along +u from the print face `ua` to its pin face `ue`.
Its strip spans `[sL, sR]` in the horizontal axis normal to u (world Y for a/g/d, world X for b-f). The pin axis is at
`(sc, zp)`.

| Type | u | ua, ue | sL, sR, sc | zp |
| --- | --- | --- | --- | ---: |
| a/g/d | X | -3.5, -1.9 | y_i+0.1, y_i+1.5, y_i+1.5 | 80.85 |
| b/c | Y | y_i-1.8, y_i-0.2 | 15.0, 16.4, 15.0 | 81.05 |
| e/f | Y | y_i-1.8, y_i-0.2 | -18.0, -16.6, -18.0 | 81.10 |

1. Hull prism p1. Sketch in (s, z) on the plane u = ua; extrude along +u to `ue + 3.8`. Points: feet at z 71.9 at
   `sR+0.9` and `sL-0.9`; up to 72.77; 8-chord quarter rounds r 0.8 about `(sR+0.8, 73.57)` (270 to 180 degrees) and
   `(sL-0.8, 73.57)` (0 to -90 degrees) (`quarterRound`, every vertex a `DimensionPoint`). The rounds bottom out 0.03
   inside the head. The first chord rises only 0.015, so the head top (72.8) crosses each round in its second chord,
   which leaves a 16.9 degree crease there. The chords are 0.157 long, so a corner check window at the crease holds
   that crease and at most two chord vertices (39.4 degrees), under the 50 degree rule. With r 0.7 the chords were
   0.137 long, the window held three vertices and the sum was 50.6 degrees, which failed. Then the strip sides up to `zp - 1.2`, and two hull lines
   tangent to an arc of radius 0.6 about `(sc, zp)`. Arc constraints: `NewRadius(arc, 0.6)`, `NewTangent(lineR, arc)`,
   `NewTangent(lineL, arc)`. The tangent points are `CreatePoint` (not fixed), placed by `tangentPoint`.
2. End cutter. Polygon in (u, z) on the plane s = sL - 1.5, extruded along s by `sR - sL + 3.0`:
   `(ue, zp+1.5)`, an 8-chord quarter round r 0.8 about `(ue+0.8, 73.57)` from 180 to 270 degrees,
   `(ue+0.9, 72.77) (ue+0.9, 71.5) (ue+4.0, 71.5) (ue+4.0, zp+1.5)`.
3. Pin. A circle r 0.50 at `(sc, zp)` on the plane u = ue - 0.1 (`DimensionPoint` centre, `NewRadius`), extruded 3.7
   along +u. `decad.Chamfer` 0.2 on its far circular edge (select the circular edge whose start lies at u = ue + 3.6).
4. `cutter = Cut(cutter, pin)`, `tongue = Cut(p1, cutter)`. The result is the tongue with r 0.8 roots on three sides,
   a sharp pin root (Decad item D2), and a Ø1.0 pin 3.6 long with a 0.2 tip chamfer.

Results (`TestSevenSegmentSliders`): solids, tessellate at 0.05, `cornercheck` along each print axis with no finding.
Volumes 978.76 (a/g/d), 981.00 (b/c), 995.37 (e/f) mm^3, under the 1069 bound (0.013 N).

Root sizes against the fits: tongue roots reach 73.57 (76.57 lit), at least 6.1 below the rotor axis at every upright
pose and outside the 5.096 swept circle; `TestSevenSegmentTongueSweep` covers the turned-over poses. The blade root round tops out at 76.4 at
lit, 1.0 below the tube bottom.

### Tongue, head and notch sweep (`sweep()` in `flags_test.go`)

`sweep()` (ported from a one-off script; it reads the tongue root radius and dip from the model) covers the head or arm top line and the tongue roots. Poses: slider s +-0.38 and z
from 0.24 below rest to `cap` above it, rotor anywhere in its 0.25 journal play, pin at either slot wall and within its
limit. The cap is the turned-over rise: 0.8 groove clearance + 0.48 tolerance = 1.28 (a/g/d), plus the b-f tilt reversal
0.6 = 1.88. Raw distances:

| Pair | All poses, cap 1.28 | All poses, cap 1.88 | Modeled poses |
| --- | ---: | ---: | ---: |
| tongue, roots, head top to yellow leg (a = 2.95) | 0.162 | 0.158 | 0.720 |
| tongue, roots, head top to black leg (b = 2.95) | 0.379 | 0.299 | 3.05 |
| pin to yellow leg | 0.262 | 0.262 | 0.770 |

With the old head top (73.8) the head reaches -0.796 into the black shell at cap 1.28. Lowering it 1.0 removes that.

### Pivot blocks (`pivot.go`; 14 blocks, one printed part)

One block stands at each rotor end: 14 identical bodies in the `Pivot Blocks` container, indexed by segment and end
(`Parts.PivotBlocks[seg][0]` at -u, `[1]` at +u). A block cannot serve two rotors: at the corners the pins are perpendicular,
and at b-c and e-f they point opposite ways, so only one of them could point up in print.

| Name | Value | Note |
| --- | ---: | --- |
| `pivotBlockSide` | 3.2 | across the rotor (s) |
| `pivotBlockHigh` | 3.1 | along z. The pocket is 3.45 tall as modeled and 3.2 clamped (the seam closes 0.25): 0.05 of press on each face |
| `pivotBlockFrom`, `pivotBlockLong` | 12.5, 2.0 | u 12.5 (the end wall; the rotor's end face is 0.25 from it) to 14.5 |
| `pivotPinRadius` | 0.4 | the pin: from u 12.5 to its tip at 11.25 (`pivotPinTip`), a 45-degree root cone of leg 0.25 that stays inside the end gap and a 0.1 tip chamfer |
| `pivotCornerR` | 0.45 | the block's rounds (long edges, and its outer face's corners in plan); the pocket's are 0.5 |

Construction: `roundedPrism` is the intersection of a section prism (the rounded rectangle across the rotor and z) and a
plan prism (u by s, its far corners rounded), a plain extrusion each, so no fillet is needed. The pin is a revolved meridian
`(0, 12.6) (0.75, 12.6) (0.4, 12.25) (0.4, 11.35) (0.3, 11.25) (0, 11.25)` unioned to the block (the cone starts inside the
block). The block is built once and 13 copies are placed through `toSegment` (a 180-degree turn about the local up axis
for the -u end, the rotor frame's two-step placement, as the rotors).

The pin axis is the rotor axis at the clamped seam plane: 82.35 physically (the cradle's 82.6 less 0.25). The pocket in the
cradle runs from z 81.0 (1.0 of the cradle under it, 8 layers below the seam) and in the top plate to z 84.45 (8 layers above
its underside), so the block is held by its 0.05 on both sides and both faces. The seam's extra 0.05 is on the roof, not the
pin axis: any extra play in z would move the b-f rest band (the b-f worst case unlit end -37.91 would reach -36.93, the pin
radius 2.89 would pass the slot's 2.90). A block that prints small in z and does not grip rests on the pocket floor and its
pin drops by up to 0.05: check the first print.

Checks (`TestSevenSegmentPivotBlockShape`, `TestSevenSegmentPivotPocketFit`, `TestSevenSegmentPivotPinInBore`,
`TestSevenSegmentPivotBlocks`, `TestSevenSegmentJournalFit`): bounds and pin on the block; the press (0.05 across, on the
floor, and 0.05 under the roof once the seam closes) on sections at u 13.5; the pin 0.20 inside its bore, engaged 0.75 at the
worst end play and 0.2 off the floor at the other; no block overlaps its rotor or the plates in the placed assembly;
`cornercheck` of the block with the print axis -u finds nothing. The plates' own `cornercheck` reads the pockets.

### Cradle plate (`frame.go`, "Cradle Plate" in `Plates`)

Built in this order (proven in a one-off scratch build; `TestSevenSegmentFlagPlates` checks the result):

1. Sketch on z 80.0: rectangle x +-23.4, y [-49.25, 42.75]. Holes: four T bolt holes r 1.3; four registration holes
   r 1.55, at (12, -45) and (-13.55, 39) for the column pegs, and at (-12, -45) and (14.5, 39) for the top-plate pegs.
   Extrude 2.6 (eight holes in the profile).
2. Seven passages, each cut on its own: rounded rectangle (corners 0.5) on z 79, extruded 5. Each is the slider's and
   tube's plan footprint + `LooseFit` 0.55:
   a/g/d `x [-5.1, 5.1], y [y_i - 9.4, y_i + 4.4]`; b/c `x [-4.05, 18.45], y [y_i - 3.4, y_i + 4.6]`;
   e/f `x [-20.05, 4.05]`, same y.
3. Cut each rotor's bore tool at z 82.6.
4. Mouth cones at (-12, -45) and (14.5, 39): a circle r 2.45 on z 82.7, extruded -Z 1.0 with
   `decad.WithTaper(units.Degrees(45))` (r 2.35 at the top face, r 1.55 at z 81.8). Cut each.
5. `edgefinish.Chamfer` 0.5 on the outer loops of both faces (z 80 and 82.6).

Results: one solid, walls of 0.799 or more in sections at z 80.02, 80.6, 81.4, 81.85, 82.2, 82.5, 82.58,
`cornercheck` (bottom down) no finding, tessellates at 0.05.

Inner loops. Top face (seam face): every inner loop stays sharp by design, mouths aside. Bottom face: every inner loop
(passages, T holes, registration holes, bore openings) stays sharp until decad item D1 is fixed. List both in
`instructions.md` under the edge-finishing exemptions.

Bore tool (one revolve per rotor), meridian `(r, u)`, symmetric in u, `RB = 5.546`, round `R = 0.8`:
- `(0, -14.5)`, the mirrored half, then for u >= 0: `(RB, 0)` up the cylinder; an 8-chord round of radius R centred at
  `r_c = RB - R = 4.746`, `u_c = (5.496 + sqrt2*10.354 - (RB - R) - sqrt3*R)/sqrt2 = 9.905`, from 0 to
  `atan(sqrt2)` = 54.7 degrees; the cone `r = 5.496 - sqrt2*(u - 10.354)` to `(2.461, 12.5)`;
- then the end wall at u 12.5 to the axis `(0, 12.5)`. The pivot block's pocket opens through it.

Pivot pockets (`pivotPocketTool`, one per rotor end, cut from both plates): the pocket takes the block with `PressFit`
0.05 on all four sides. Its section is 3.3 wide across the rotor (`s` +-1.65) and runs from 1.6 under the axis (z 81.0:
1.0 of the cradle stays under it) to 1.6 over the top plate's underside (z 84.45); its corners are rounded 0.5; it runs from the bore's end wall
(u 12.5) to u 14.55, with its far corners rounded 0.5 in plan. It is the intersection of two prisms (the section prism
and the plan prism), so every round is a plain extrusion. The block is 3.2 wide and 3.1 high (below).
The pockets keep 0.84 from each other at the eight corner junctions (the rounded corners are 1.84 apart), above the 0.8
wall and clear of the 0.898 bore walls, and 2.4 or more from the neighbouring bore cones, slider passages, pegs and
T-bolt pockets.

Clearances: cylinder 5.546 - 5.096 = 0.45; cone 0.400 normal to the sharp pyramid envelope; 0.375 at the round (R5,
Regular minimum 0.35). End wall 0.25 from the rotor's end face. Bore r 0.6 on a pin r 0.4: 0.20, the Tight minimum. With round 0.5 the
cylinder-to-cone round read 50-54 degrees in `cornercheck`; round 0.8 reads none.

### Top plate (`frame.go`, rewritten)

Built in this order (proven in a one-off scratch build; `TestSevenSegmentFlagPlates` checks the result):

1. Sketch on z 82.85: rectangle as the cradle. Extrude 5.4 (to `FaceZ` 88.25), no holes.
2. Tubes, one per segment, unioned. Each is `rootedBox(x0, x1, y0, y1, zb 77.4, zFace 82.85, ov 0.1, c 0.5)`:
   `Intersect` of the (x, z) prism `(x0, 77.4) (x1, 77.4) (x1, 82.35) (x1+0.6, 82.95) (x0-0.6, 82.95) (x0, 82.35)`
   extruded along y over `[y0 - 1.5, y1 + 1.5]`, and the same profile in (y, z) extruded along x. This builds a
   45-degree root of leg 0.5 into the tube. A `decad.Fillet` of the root loop is refused.
   Footprints: a/g/d `x [-4.65, 4.65], y [cy - 8.95, cy - 4.9]`; b/c `x [7.55, 11.55], y [y_i - 2.95, y_i + 4.15]`;
   e/f `x [-11.55, -7.55]`, same y. The c and f tubes end at u = 9.55, inside the bore's cylinder (round starts at
   9.905).
3. Cut each rotor's bore tool at z 82.85 (it trims the rotor-side tube walls).
4. Windows and T holes: one sketch on z 82.0 with a frame rectangle (x +-30, y [-52, 46]), four T circles r 1.3, and
   seven window octagons. Each octagon is `(+-9.7, +-2.95) (+-11.696, +-0.954)` in `(u, s)` about its segment centre
   (`windowHalfWidth` 2.95, `windowChamfer` 12.65: `|s| + |u| <= 12.65`, `windowEnd` 11.696), drawn by `polygon()`: each side is held
   by its direction and its distance from an anchor, and no vertex is fixed. The window is 5.9 wide and 23.39 long (area 130.0). Its long sides stand 0.8 inside the rotor's faces
   and its chamfers 1.1 inside the pyramid edges, so no yellow shows when the rotor is unlit. Extrude each loop without holes 7 along +Z and cut it on its own.
5. Pegs at (-12, -45) and (14.5, 39): revolve `(0, -1.6) (1.2, -1.6) (1.5, -1.3) (1.5, -0.5) (2.0, 0) (2.0, 0.1) (0, 0.1)`
   about Z, translate to z 82.85, union. The 45-degree collar enters the cradle's mouth cone.
6. Blade holes: rounded rectangle (corners `bladeHoleRound` 0.50) on z 77.0, extruded to 87.45, cut. The blade plus
   `bladeAcross` 0.25 across it and `bladeAlong` 0.35 along it: a/g/d `x [-3.85, 3.85], y [cy - 8.05, cy - 6.45]`; b/c
   `x [8.45, 10.05], y [y_i - 2.15, y_i + 3.35]`; e/f `x [-10.05, -8.45]` ("Blade corners").
7. `edgefinish.Chamfer` 0.5 on the outer loops of both faces.
8. `m2fastener.PocketTool` at T1-T4, cut.

Results: one solid, tessellates at 0.05. In sections at 18 heights from 77.45 to 88.2 the thinnest wall
is 0.800: the tube walls, which are 0.80 by design. Tests compare walls against 0.799. The rotor-side tube wall to the
bore is `6.35 - 5.546 = 0.804` (with the old bore 5.496 and hole edge 6.3 it was 0.80; the bore grew for the round, so
the blade and hole moved 0.05 away from the rotor). `cornercheck`
(display face down) finds nothing. Window lip `5.4 - sqrt(5.546^2 - 2.95^2) = 0.704` along the long sides and 0.487 at the shoulders of the
pointed ends at Scale 1.0 (0.495 on the mesh): thinner than 0.8, accepted and listed there. At Scale 1.5 the lip is
1.512 and 1.248 (`TestSevenSegmentWindowLip`), above 0.8, so it is not listed. The lip is a
vertical thickness (at least 3.5 layers, the plate prints face down); no horizontal wall section is thinner than 0.8. Window rims and seam-face inner
loops stay sharp (listed: the lip is too thin for a 0.5 chamfer).

Plan walls between neighbouring rotors' bores, cones and pivot pockets (from a one-off script, not in the repository): 0.898 at the eight corner
junctions (0.919 with the old bore 5.496), 4.0 between b-c and e-f.

### Guide plate (`frame.go`)

Built in this order (proven in a one-off scratch build). The base `guidePlate()` (slots, G holes, outer chamfers, hex pockets), then:

1. Two columns from z 58 to 79.15: 5 x 5 centred at (12, -45) and 4.6 (x) x 5 (y) centred at (-13.55, 39):
   `rootedPost(x0, x1, y0, y1, zFace 58,
   ztop 79.15, ov 0.1, c 0.5)`, the upward twin of `rootedBox`: profile
   `(x0-0.6, 57.9) (x1+0.6, 57.9) (x1, 58.5) (x1, 79.15) (x0, 79.15) (x0, 58.5)` in (x, z) and in (y, z), intersected.
   Union.
2. Pegs: revolve `(0, -0.1) (2.0, -0.1) (2.0, 0) (1.5, 0.5) (1.5, 2.3) (1.2, 2.6) (0, 2.6)` about Z, translate to
   (12, -45, 79.15) and (-13.55, 39, 79.15). Union.

Results: one solid, tessellates at 0.05. `cornercheck` (bottom down) reports only the 24 existing hex-pocket corners,
which the existing `hexAnnulus` exemption covers.

Clearances when clamped: column top 79.15 to cradle bottom 79.75 (0.60); peg collar top 79.65 to cradle bottom (0.10);
peg engagement 79.75-81.75 (2.0); peg tip to top plate 0.60. The top-plate peg collar in the cradle's mouth cone keeps
0.60 radial modeled and 0.35 clamped (0.25 normal). Its tip (80.75 clamped) stays in the through hole. Nothing bottoms
out before the seams close, so the pivot pockets close. The rear column keeps 0.55 (LooseFit) to the rear top
flange rib (x −15.85 against −16.4) and the front column 1.9; the rear column is 4.6 wide so rotor a's pyramid never
reaches it (0.69 to its swept envelope with 0.25 end play at the settled axis 82.1, `TestSevenSegmentWidthMargins`).
Both stay 2.8 or more from the G nut pockets, and clear of every slider and rotor.

### Friction audit (every contact; `audit()` in `flags_test.go`)

`mu' = sin(atan mu)`. The loads come from the built bodies at PETG 1.27 g/cm^3 (`TestSevenSegmentDriveAudit` logs them): W_r A
0.01079 N, B 0.00829 N; e and beta_c: A 0.712 / 92.0, B 0.909 / 94.0; the heaviest slider 0.0124 N. The pin bears on one
wall of its slot: the rotor's angle is the pin's polar angle plus or minus `asin(cl/rho)` with `cl` 0.20, the contact
normal and the imbalance torque follow that angle, and the friction arm of the wall is the pocket radius 0.70 (earlier
versions put the pin on its centre line with no wall offset, which under-counts F_x by 25 to 30 %). The pin line is
`s = PinOffset - PinLinePlay` = 1.03 settled and 0.92 with the journal fully across (a further 0.11); `z` runs over the
stroke and the play, -1.89 to 1.59; the pin radius `rho <= 2.90`; the true contact arm is `sqrt(rho^2 - 0.20^2)`.

| # | Contact | Kind | Lever arm / load | Friction effect |
| ---: | --- | --- | --- | --- |
| 1 | pin on slot wall | sliding | drive arm `sqrt(rho^2-0.04)` | arm 0.70: torque `mu*N*0.70` |
| 2 | pivot pins in the end bores (2) | rotating | load `W_r + N*sqrt(1+mu^2)` | friction circle `mu'*0.6` (the bore) |
| 3 | rotor end faces / end walls | sliding thrust | no axial load upright | `mu*W_r*2.1` only on its side |
| 4 | rotor imbalance | gravity | `W_r*e*|cos(beta_c + phi)|` | counted always (worst sign) |
| 5 | blade in its tube | sliding | lateral `H = N(|sin phi| + mu|cos phi|)` | drag `mu*H*(1 + 2*2/25)` |
| 6 | offset load couple | sliding | pin 18 (e/f), 15 (b/c), 1.5 (a/g/d) from the stem | drag `2*mu*(x/25)*V` |
| 7 | slider weight couple | sliding | centre of mass 7 from the stem (b-f) | drag `2*mu*(7/25)*W_s` |
| 8 | groove pin in groove | sliding | r 16.5 to 19.5 on the cam | see Cam follower drive |
| 9 | tongue, roots, head, pin tip, rotor/bore, tube/rotor, slider/passage | none | gaps > 0 at every pose | - |

Drive balance: `N (sqrt(rho^2 - 0.04) - mu*0.70 - mu'*0.6*sqrt(1+mu^2)) = mu'*0.6*W_r + W_r*e*|cos(beta_c + phi)|`.
Extra cam force per changing slider: `F_x = V(1 + 2 mu x/25) + mu H (1 + 4/25) + 2 mu W_s x_cg/25`, with
`V = N(|cos phi| + mu|sin phi|)`.

Drive ratio (the drive arm over the friction arms) at friction 0.5: 1.554 settled and 1.382 with the journal fully across
(`TestSevenSegmentDriveAudit` asserts at least 1.50 and 1.35). F_x at friction 0.5: A 0.0288 settled and 0.0409 across,
B 0.0387 settled and 0.0547 across. The design this replaces had a ratio of 1.250 (1.040 across) at 0.5.

The budget takes F_x with the journal fully across (budgetFx). The drive's own force pushes the bore across its pin: a load-set
model (the bore against its pin along the pin force plus the weight, turned by up to the friction angle) puts it 0.157 to
0.188 of the 0.20 clearance toward the pin at the pose of the largest F_x and gives F_x 5 to 9 % below the across value.

### Cam follower drive (TestSevenSegmentFollowerDrive)

Model: a rigid slider with Coulomb friction. The slider's contacts are the guide plate's underside (z 54) and the blade
ridge (83.55 + lift), the lean that `TestSevenSegmentStackPlay` uses. Each changing slider needs 0.067 N: the friction
audit's largest F_x (friction 0.5, journal across, 0.0547 N) plus the heaviest slider (0.0124 N).

- The groove pin's contact band (groove face 2.35 to the pin's chamfer 3.55 from the stem's centre plane) acts at
  e = 2.95 along +y.
- The stem's x contact can sit anywhere on its flat face (|y| <= 1.5, inside the 0.3 chamfers). The blade's x contact
  sits at most at y -7.8 (a, g, d: the blade's far face) or -1.8 (b to f: the print face). What yaw moment those
  contacts cannot carry goes to the stem's long faces as a couple 6.4 apart (stem width 7.0 less the chamfers). Shared
  with the blade instead (9.25 apart), the b-f ratio would be about 5 % higher.
- The test asserts at the nominal friction 0.35. At 0.5 every variant locks, today's 18/21 included.

Per unit groove-wall load at pressure angle α and friction µ: `F_v = cos α − µ sin α` along the stem, `H = sin α +
µ cos α` across it (along x). With `a = 54 − (46 + lift)`, `L = 83.55 + lift − 54`, `e` = 2.95 and `y_b` the blade
offset (−7.8 a/g/d, −1.8 b-f):

```
R1 = H (1 + a/L)                    stem on the slot wall at z 54
R2 = H a / L                        blade on its tube wall at the ridge
M  = max(0, e H + y_b R2 − 1.5 R1)  yaw the x contacts cannot carry
D   = F_v (1 − 2µe/L) / (µ (H (1 + 2a/L) + 2 M / 6.4))
net = F_v (1 − 2µe/L) − µ (H (1 + 2a/L) + 2 M / 6.4)
groove-wall load = 0.067 / net;  shaft torque per slider = load · H · R  (R the groove centreline radius)
```

D <= 1 locks the slider at any wall load. For a/g/d, M is 0 (the blade's far contact carries the yaw). Values from this
model, sampled every 0.05°:

| Radii | max α | D a/g/d at 0.35 | D b-f at 0.35 | D a/g/d, b-f at 0.5 | lock friction a/g/d, b-f |
| --- | --- | ---: | ---: | --- | --- |
| 18/21 (before) | 33.0° | 1.545 | 1.279 | 0.797, 0.661 | 0.445, 0.402 |
| 16.5/19.5 | 35.2° | 1.420 | 1.179 | 0.726, 0.603 | 0.425, 0.384 |

| Radii | wall load a/g/d, b-f at 0.35 | summed shaft torque at 0.35 (step) | press share |
| --- | --- | --- | ---: |
| 18/21 (before) | 0.313, 0.505 N | 30.4 N·mm (6→7) | 4.2 N |
| 16.5/19.5 | 0.393, 0.765 N | 40.6 N·mm (6→7) | 5.6 N |

Press share = torque × `PressTurn()` (32.09° at Scale 1.5, 31.69° at 1.0, in radians) / (`ButtonStroke − RotorTravel` = 8.0 mm at 1.5, 4.0 at 1.0), an ideal ramp; ramp
friction adds to it. Step 6→7 moves five sliders, four of them b-f. A locked slider stalls the button: a 13 N press
less the 7.5 N spring gives the shaft at most 40 N·mm, about 2.6 N on one groove wall at r 17.4, which bends the groove
pin's root (lever 1.15) to about 1.1 MPa against about 20 MPa of layer bond. No part breaks.

### Determinacy budget

The stiff detent is gone, so there is no detent budget to balance against the flag drag. The indexer's only requirement
is the timing margin of `TimingMargin()` (see Detent and rest angles). The flag drag of the display is carried by the
button press: it is up to 41 N.mm at the shaft, about 5.6 N of press on an ideal ramp (Cam follower drive).

### Visibility (section across the flag, physical geometry; the tables of this first part come from a one-off script)

Window 3.1, yellow 2.95, lip 0.861 in the script (this was the window of revisions 1 to 3; the window is now 2.95 wide,
see the whole-window results below), display face 1.90 above the yellow face at rest. Rays 0.05 apart from both sides;
rotor shifted -0.25 / 0 / +0.25 sideways.

| state | phi | 90: flag / yellow of flag | 45 | 30 |
| --- | ---: | --- | --- | --- |
| lit | 35.5 (band low end) | 100 % / 83 % | 69 % / 85 % | 40 % / 81 % |
| lit | 45 | 100 % / 94 % | 71 % / 89 % | 48 % / 85 % |
| lit | 65.2 (band high end) | 100 % / 70 % | 65 % / 78 % | 25 % / 75 % |
| unlit | -70 .. -38 | no yellow ray | no yellow ray | no yellow ray |

The table is the output of a one-off script that is not in the repository (it models a recess of 1.839, not 1.90, and gives
27 % instead of 25 % at 30 degrees and phi 65). `TestSevenSegmentVisibility` and `TestSevenSegmentWindowYellow` are the
repository's checks. The visibility test thresholds therefore sit 5 points
under the lower value. These shares hold inside `|u| <= 9.9`, across the flag (the test's grid now runs `|s| <= 2.9`).

**Whole window and unlit onset, on the real meshes** (yellow tips and the 2.95 window; `TestSevenSegmentWindowYellow`
holds the thresholds). Whole-window yellow share at 90 degrees, segment g (type B within 0.01):

| phi | 90: window yellow | 45: across s / along u | 30: across s / along u |
| ---: | ---: | --- | --- |
| 34.0 | 0.867 | 0.599 / 0.754 | 0.345 / 0.699 |
| 45 | 0.997 | 0.649 / 0.866 | 0.410 / 0.802 |
| 68.1 | 0.711 | 0.498 / 0.622 | 0.202 / 0.576 |

Before the tips and the window change the 90 degree shares were 0.720 / 0.804 / 0.584. The asserted flag grid (inside
`|u| <= 9.9`, `|s| <= 2.9`) gives a flag share of at least 1.000 / 0.684 / 0.403 at 90 / 45 / 30 degrees and a yellow share
of the flag of at least 0.719.

Unlit onset: the smallest phi at which any yellow sample point (each triangle's centroid, vertices and edge midpoints
pulled 2 % inward) of the turned rotor sees the viewer through the window, over elevations 90 / 60 / 45 / 30 and 16
azimuths, bisected to 0.05 degrees, with the rotor off centre by the side play 0.16 (pivot 0.11 plus registration 0.05)
and the axial play 0.30 (end walls 0.25 plus registration 0.05):

| case | before the tips and the window change | now |
| --- | ---: | ---: |
| centred | -33.64 | -31.35 |
| side +0.16 | -31.15 | -28.96 |
| side -0.16, with or without axial +-0.30 | -36.23 | -33.84 |
| axial +-0.30 | -33.64 | -31.35 |
| side -0.25, with or without axial +0.30 | -37.74 | -35.25 |

Against the b-f worst unlit band end -37.90 the margin is 1.42 degrees at both the 0.16 and the 0.25 side shift (the grazing
onset is -36.48 in both). Against the b-f design-play band end -41.22 it is 4.74.
The cause of the old thin margin: yellow shows once `h d > c - W`, where `c` is the rotor's half-size (3.75 on the long
part, `13.75 - |u|` on the tips), `W` the window's edge, `h` the yellow edge's height above the axis and `d` the turn
toward lit. With the window 0.8 inside the faces and 1.1 inside the pyramid edges (the long-side inset plus the axial play
0.30, because the chamfers are at 45 degrees) the tips are no worse than the long sides. Yellow on the facets with the
old window (chamfer 13.454) showed when unlit at -37.9 or earlier, which is why the window changed with the tips. The
lit yellow share of the whole window at 90 degrees rose from 0.80 to 1.00 at phi 45.

**With the less play, the thinner pin and the lower b-f pins** (the current model; the onset harness run on all seven
segments, which agree). Onset in degrees, views facing above 5 degrees and grazing views:

| case | facing > 5 degrees | grazing |
| --- | ---: | ---: |
| centred | -31.37 | -34.53 |
| side +0.16 / +0.25 | -30.78 / -30.12 | -31.68 / -30.47 |
| side -0.16 | -33.83 | -36.48 |
| side -0.25, with or without axial +-0.30 | -35.27 | -36.48 |

| margin to the worst unlit band end (grazing views) | side 0.16 | side 0.25 |
| --- | ---: | ---: |
| design-play end (a, g, d -40.33; b to f -41.22) | +3.85 (a, g, d), +4.74 (b to f) | the same |
| a, g, d worst case (-39.23) | +2.75 | +2.75 |
| b to f worst case (-37.90) | +1.42 | +1.42 |

The lit yellow share of the whole window at 90 degrees is 0.87 (a, g, d at 34.47), 0.85 (b to f at 33.41), 0.80 (b to f at
29.39) and 0.98 at rest at 45 degrees. The unlit rays are cast on grids offset by half a step: a ray in the plane u = -10,
where mesh edges of both bodies meet, can slip through a crack between triangles and report a yellow hit that is not
there. The b-f pins sit 0.05 below their droop to buy the b-f margin.

### Cam stack, axial play

The stack is held rearward only. Cam a's chords stop on the square ends of the shaft's flats at y 39.2, 0.05 behind
its back face, and each hub stands 0.05 behind the next part (cam d is the front of the stack). Pushed
rearward, a cam moves back by the gaps behind it: a 0.05, f 0.10, b 0.15, g 0.20, c 0.25, e 0.30, d 0.35. The rear collar's cone (0.55 behind cam a) is not reached. Forward nothing stops the stack before the
pin tips: the stack could move 1.0 to the
front bearing and wall, which "Arbor collars" allows for a pressed part's fitting side. A groove pin's engagement
(pin tip y_i + 3.85 past the groove face y_i + 2.35) is 1.5 as modeled. With the stack fully back and the slider
leaned forward as far as its stem slot (0.25 at z 54) and its blade hole (0.25 across for a, g, d; 0.35 along for b
to f, at the ridge) allow, it is at least 0.765 (cam d unlit), and 0.90 with the slider only shifted 0.25
(`TestSevenSegmentStackPlay` requires 0.75). The pin tip stands 0.40 from the groove floor as modeled. The slider's
own lean (up to 0.41 for b to f) can close that with the stack in place, and the stack sliding forward closes it
too; the floor then rubs the pin tip. On a level shaft nothing presses that contact along the shaft, so it adds no
torque. Tilted toward the button end by θ, the cams' weight (about 0.51 N) presses each floor on its pin tip and friction at the
groove radius adds about µ·0.51·sin θ·19.5 N·mm, about 0.06 N·mm per degree at friction 0.35. No locking feature
is designed. A collar ring in front of the wheel would face the front bearing's inner ring (diameter 8), leaving at
most 1.45 of wall round its bore against the 1.6 minimum, in a 1.0 gap against the 3.0 collar minimum. A collar of
the shaft's own body there would stop the cams being pressed on from the front. A lock would also not stop the
slider's lean from reaching the floor.

### Captivity and stops

- Rotor: its bores on the pins of two blocks pressed in the plates; u limited by the end walls (0.25), the pin engaged 0.75 at the worst end play and its tip 0.2 off the floor at the other.
- Pin: engagement at least 1.0; at the design play it never reaches its radial limit (2.73 for a, g, d and 2.77 for b to f,
  against 2.90).
- Slider: its blade stays at least 5.55 (full width) deep in its tube.
- Turned over: unlit sliders stop on the groove's outer wall (0.8); lit sliders stop on the groove wall too (0.8) and, at that rise, their blade ridge meets the tube roof: the roof is
  0.90 above the ridge as modeled and 0.80 in the physical geometry (the top plate 0.5 lower, the slider 0.4 lower), so
  a lit slider that rises 0.8 closes it to 0.0, and the b-f tilt lifts the blade a further 0.15 to 0.3. That contact is
  by design and carries only the slider's weight and the groove pin's push; the tube roof ends the slider's rise
  together with the groove wall. b-f sliders also tilt the other way (pin +0.6 / +0.7). `TestSevenSegmentTongueSweep` keeps every tongue, root,
  head and pin gap positive at those poses. At digit 8, at worst-case tolerance, the groove pin can enter the entry
  channel's mouth by 0.24. The channel is 3.1 deep, so the pin cannot escape. If the knob is turned in that state, the
  channel edge meets the pin more than 1.2 above its centre, 58 degrees or more from tangential, and pushes it back
  into the groove.
- Rest angles: two bands (Kinematics); the slot walls are the stops; gravity holds the leading wall.

### Clearances (fit table; modeled values on the placed bodies)

| ID | Parts | Class | Modeled | Measure |
| --- | --- | --- | ---: | --- |
| R1 | groove pin / groove walls | Regular | 0.40 radial | `meshgap` per digit |
| T3 | stem / guide slot | Tight | 0.25 (unchanged) | section z 56 |
| T5 | pivot pins / rotor end bores | Tight minimum | 0.20 | section through the pin's middle |
| P5 | pivot blocks / pockets | press | 0.05 on four sides (the roof's 0.05 once the seam closes) | sections at u 13.5 |
| T6 | rotor end faces / end walls | Tight | 0.25 | bounds |
| T7 | pin / pocket walls | Tight minimum | 0.20 | section normal to u at the pocket middle |
| T8 | tongue / notch shells | Tight | >= 0.72 | section normal to u at the tongue's middle |
| T9 | blade / tube | Tight across, 0.35 along (see "Blade corners") | 0.25 across, >= 0.20 at the corners (0.21) | section z 81 |
| R5 | rotor / bore | Regular | 0.45 cylinder, 0.40 normal on cones, 0.375 at the round | sections |
| L9 | slider and tube / cradle passage | Loose | 0.55 | section z 80.5 |
| L10 | head or arm top / rotor | Loose | 1.70 [1.60] (lit) | bounds |
| L11 | blade ridge / tube roof (lit) | Loose | 0.90 [0.80] | bounds |
| L12 | tube bottom / head or arm top (lit) | Loose | 1.60 [1.50] | bounds |
| P3 | pegs / registration holes | Press | 0.05 radial | section |
| P4 | top-plate peg collar / cradle mouth cone | - | 0.60 [0.35] radial | section |
| S1 | cradle / ribs, top plate / cradle | clamped seam | 0.25 | bounds |
| S2 | columns / cradle | locating only | 0.85 [0.60] | bounds |

R3 (cap/window) is deleted; the other rows are unchanged. R5 at the round (0.375) meets the pipeline's Regular minimum
of 0.35; the repo constant `RegularFit` 0.40 is the nominal, not the minimum.

Uncertainty of the Tight-minimum fits. "Part clearances" asks for a lower bound that includes the measurement uncertainty.
Three fits sit at the Tight minimum 0.2 by their dimensions: the rotor's end bores (bore 0.60 less pin 0.40), the pin in its
pocket (0.70 less 0.50) and the blade corners (0.21, from the sketch dimensions). `TestSevenSegmentJournalFit` measures the
pivot on the built bodies (bore against pin, 0.20). T7 and T8 are measured on meshes that carry a
tessellation bound of 0.04, so the tests check 0.2 - 0.04 and 0.7 - 0.04: they prove a lower bound of 0.16 for T7, not 0.20,
and the value 0.20 rests on the analytic radii. The blade corners are checked on sections to 0.20 (modeled 0.21). These
three fits therefore do not reach the Tight class with the uncertainty included; they are accepted at the Tight minimum
and listed here.

Measuring T7 and T8: the pin sits 0.25 from its pocket wall on the same rotor body, so a whole-body `meshgap` between
slider and rotor returns T7, not T8. Measure both on sections: `cornercheck.Outline(ctx, body, uAxis, u)` of the
placed slider and of the placed black body at the plane through the tongue's middle (T8; A: u = -2.7; B: -6.5 and 4.5,
local) and through the pocket's middle (T7; A: 1.3; B: -2.5 and 8.5). The fit is the smallest distance between the two
outlines.

### Assembly (README and instructions.md)

1. Assemble walls, camshaft, bearings, the click indexer and guide plate (with its columns) as today. Press the button to
   digit 8 (all lit: least droop).
2. Press the cradle onto the column pegs through its holes at (12, -45) and (-13.55, 39). It rests on the top flange ribs,
   0.85 above the columns.
3. On the bench, join each rotor to its slider: rotor yellow face up and 2.0 along +u from its place, notch over the
   tongue, then slide it -u so the pin enters the pocket. Type A (one notch) goes to a, g, d; type B (two notches) to
   b, c, e, f.
4. Push a pivot block onto each end of every unit (its pin into the rotor's end bore). Keep the unit level and lower it
   through its cradle passage and guide slot, groove pin first through the slot's pin notch, into its cam's entry
   channel. The blocks drop into the cradle's open half-pockets; press each one down with a finger until it seats. A block
   can slip off its pin if the unit tilts before it is seated.
5. Lower the top plate, pegs over the cradle's holes at (-12, -45) and (14.5, 39). The a/g/d blades stand upright. Each
   b-f arm sags by up to about 1 mm at its blade ridge; the blade gable (0.6) and the hole clearance (0.25) catch 0.85.
   Reach in from the open side (b/c from +x, e/f from -x) and lift each b-f arm's outer end until its blade enters its
   tube, then let the plate down. Fit four M2 x 10 bolts from the top.
6. Click through 0-9 and check each digit.

### Rule tables

- Print orientations: rotors yellow face down; pivot blocks outer face down, pin up; a/g/d sliders -X face down, b-f sliders -Y face down; cradle bottom
  down, top plate display face down, guide plate bottom down.
- Retained sharp corners (listed):
  - (window rims: listed at Scale 1.0, lip 0.704 / 0.487; at Scale 1.5 the lip is 1.512 / 1.248, over 0.8, so nothing is listed);
  - seam-face inner loops of the cradle and the top plate;
  - cradle bottom-face inner loops (decad item D1);
  - pin roots on the tongues (decad item D2);
  - the 45-degree head step of a/g/d sliders;
  - pyramid facet edges (convex, 120 degrees);
  - bore end-wall corner (35-degree turn).
- Corners met by turns of 45 degrees or less:
  - notch end chamfers (0.5) and apex chamfer (0.8);
  - tube roots and column roots (built-in 45-degree, leg 0.5);
  - peg collars (45-degree, 0.5);
  - pivot pin roots (45-degree cone, leg 0.25) and tips (chamfer 0.1), the bore countersink (45 degrees) and the notch mouth chamfer of type B (0.7).
- Rounded:
  - tongue roots 0.8 (8 chords; three sides);
  - blade roots 0.6, extension root 1.0, arm root 1.25;
  - pocket floor 0.6, bore cylinder/cone 0.8, the pivot pockets' corners 0.5 (the block's 0.45);
  - passage corners 0.5.
- Edge finishing: plates as above; rotor faces (150 mm^2), slider faces (< 400 mm^2) and columns: none.
- Split shafts: the shaft only (unchanged). Pivot pins, rotor pins, groove pins and pegs: see Assumptions.

### Decad items (report to the decad repo; out of scope here)

Each decad item below describes its repro; the scratch builds that held them are not in the repository.

- **D1. Cradle bottom inner-loop chamfer.**
  - Repro (a one-off test): a cradle (cradle with passages and bores, then `edgefinish.Chamfer` with `Long(Inner)` on z 80).
  - Refusals:
    - `Chamfer` fails with "cannot select a decad.FacetedCurve edge": the bore cut leaves faceted edges;
    - `ChamferRuns` fails with "open tangent chain ends on a face that is not a retained plane";
    - chamfering before the bores, cutting a 45-degree tapered mouth tool, or uniting passage and mouth into one tool
      each fail on chain depth at bore a or passage f.
  - Cost: unknown until the decad owner looks at it. It needs chamfers along faceted curves, or exact curves from
    revolve cuts.
  - Meanwhile: the loops stay sharp and are listed.
- **D2. Pin root collar on the tongue.**
  - Repro (a one-off test): a bar is cut by a box from which an extruded Ø1.1 pin and a 45-degree tapered collar
    (r 1.2, 0.75 long) were subtracted. It is refused with "cut's tool holds its mesh ... within a bound of 0.00046 mm,
    coarser than this pair's chord tolerance 0.000176 mm". Without the collar it builds.
  - Other refusals:
    - a revolved root, or a clipped collar unioned to the tongue: "mesh bound ... coarser than chord tolerance";
    - `Fillet` or `Chamfer` of the root circle after the union: "cap-loop chamfer's ruled patches cannot be certified
      disjoint", because the tongue face is only 0.05 wider than the pin at the top.
  - Cost: unknown. It needs exact cone-plane intersection curves through a third boolean.
  - Meanwhile: the root stays sharp and is listed. `cornercheck` does not see it (its edge lies in the bed plane, round
    the vertical pin).
- **D3. Placement of rotor B's black body.**
  - Repro: build `flagBodies(SegB)` with the yellow chamfer 12.65, then `decad.Transform` the black body by the
    translation `(-16.5, 82.6, 16.5)`. It is refused with "the result's float rounding folds its held mesh through itself,
    and the bounded float-box search did not clear it". Today's black body accepts all such moves, and so do the
    translations `(16.5, 16.5, 0)`, `(0, 82.6, 0)`, `(0, 0, 16.5)` and `(-16.5, 0, 0)`.
  - With the chamfer 12.95 the direct `placement.Place` fails for segments e and f only.
  - Meanwhile: each body is placed in two steps (translation, then rotation), which builds for all seven rotors.

### Frame

#### Walls (`Walls`: Front Wall, Rear Wall)

Front wall sketch frame origin (0, −49.5, 29.5), U = +X, V = +Z, N = −Y, extrude `Along` 8. Rear wall origin
(0, 43.0, 29.5), U = −X, V = +Z (N = +Y), extrude `Along` 8. Sketch rectangle u ∈ [−24.9, 24.9], v ∈ [−26.5, 51.75]
(z 3.0 to 81.25; the code's `wallHalf` 24.9 (`WallHalfWidth` + 1.5), `wallBottomZ` 3.0 and `wallTopZ` 81.25; the plate is
49.8 wide).

1. Extrude the plate; union four rib boxes at |x| ∈ [16.4, 23.4] extending inward from the inner face: ledge ribs
   z [49.75, 53.75], top flange ribs z [75.75, 79.75]; depth 10 except the ledge ribs: front 18.7 (to y −30.8),
   rear 14.2 (to y 28.8).
   Ribs are unioned before any chamfer so no chamfer notch is left where a rib meets the plate rim.
2. Chamfer 0.5: the outer face's outer loop (`edgefinish.Chamfer`, `Long(Outer)`), and on the inner face the runs
   of 20 mm or more of its outer loop (`edgefinish.ChamferRuns`, `Long(Outer)`), which skips the short rib-end runs.
3. `decad.Fillet` radius 0.5 on every concave rib-root edge in the inner-face plane (ledge: top, bottom and x = ±16.4
   edges; top flange: bottom and x = ±16.4 edges). Pattern: `models/planetarygearbox/cover.go` `roundBossRoot`.
4. Cut the seat/lip revolve tool about the shaft axis. Meridian `(r, d)`, d measured from the inner face (d 0)
   through the wall to the outer face (d 8): `(0,−1) (7.55,−1) (7.55,0) (7.05,0.5) (7.05,5.0) (4.5,5.0) (4.5,7.5)
   (5.0,8.0) (5.0,9.0) (0,9.0)`. Seat r 7.05 (605ZZ `SeatDiameter/2`) 5 deep with a 0.5 lead-in at the inner face,
   lip 3 thick with hole r 4.5 (at Scale 1.5: 608ZZ, seat r 11.05 7 deep, lip 5 thick with hole r 6.5, wall 12) and a 0.5 chamfer at the outer face.
5. Cut fastener tools: ledge ribs `m2fastener.PocketTool` (G sites, head pockets from the rib underside) and
   `HoleTool`; top flange ribs `HexPocketTool(site, NutCavityHeight + Stock)` (T sites) and `HoleTool`; front wall
   only: `HexPocketTool` + `HoleTool` at D1, D2 (nut pockets open to the outer face).
6. Cut look-alike holes (r 0.75, through): front 1 hole, rear 2 holes, on radius 10 about the shaft axis at 270°,
   295° (front-view angles), one sketch per hole as for the cams.

Printed outer face down; ribs rise from the plate; rib pockets print sideways with a hex corner up.

### Fasteners

T1-T4: head `(-+19.6, -44.5, 88.25)` and `(-+19.6, 38.0, 88.25)`, Dir -Z, Corner +Y, `Length 10`, `Cavity 10.5` (`Length 16`, `Cavity 16.5` at Scale 1.5). They
clamp the top plate and the cradle to the top flange ribs (nut pockets unchanged). The tip is at 76.25 modeled and 75.75
clamped. There is 3.4 of plate under each head.
G1-G4: heads `(-+19.6, -34.2, 49.75)` and `(-+19.6, 32.0, 49.75)`. D1-D2 are unchanged.


### Bearings

`ballbearing.Build(ctx, doc, ballbearing.Params608ZZ() + Placement)` (`Params605ZZ` at Scale 1.0): front placement origin (0, −49.5, 29.5),
U = +X, V = +Z (extrudes toward −Y); rear origin (0, 43.0, 29.5), U = +Z, V = +X (toward +Y). Bearings do not turn
with θ.

### Pose and placement

1. Build every part in its frame (above).
2. Camshaft group (shaft, cams, hub, rotor, spring): `decad.Transform` by
   `r3.RotationAround((0,0,29.5), (0,1,0), θ)`, θ = 36 · Digit.
3. Sliders: translate +3.0 Z where `Lit(Digit, s)`. Flags: turn by `FlagAngle(s, theta)` about the rotor axis.
4. Every printed body and both bearings: `placement.Place(ctx, body, p.Placement)`. Fastener sites are carried
   through the placement with onewayclutch `placeSite`; bolts and nuts are built from the placed sites with
   `m2fastener.PlaceBolt`/`PlaceNut` and are not placed again.

### Bodies and containers

| Container (parent) | Bodies |
| --- | --- |
| `Frame` (root) | unit |
| `Walls` (Frame) | unchanged |
| `Click Indexer` (root) | `Hub`, `Rotor`, `Housing`, `Button`, `Spring`, one body each |
| `Plates` (Frame) | Guide Plate, Cradle Plate, Top Plate |
| `Camshaft` (root) and its kinds | unchanged |
| `Segment Sliders` (root) | 7 sliders |
| `Pivot Blocks` (root) | 14 blocks, two per rotor |
| `Flags` (root) | unit |
| `Flag`, `Flag 2` ... `Flag 7` (Flags, `NewUniqueChild`, segment order a..g) | units |
| `Yellow Face`, `Black Body` (each Flag unit, `ChildNamed`) | 1 body each |
| `Bearings`, `Bolts`, `Nuts` (root) | 2; 4 x M2x10, 4 x M2x6, 2 x M2x18; 10 |

75 bodies: frame 5, camshaft 8, click indexer 5, sliders 7, flags 14, pivot blocks 14, bearings 2, bolts 10, nuts 10.

### Rule tables

#### Look-alike parts

| Set | Parts | Reason | Holes | Circle |
| --- | --- | --- | --- | --- |
| Front cams | cam d, cam e, cam c | same disc, different grooves | 1, 2, 3 | r 12 about the cam axis, from 270°, 25° apart |
| Rear cams | cam g, cam b, cam f, cam a | same | 1, 2, 3, 4 | r 8 |
| Walls | front wall, rear wall | same plate and seat | 1, 2 | r 10 about the bearing axis, from 270° |

Holes 1.5 across, through. Sliders are not marked: no round face, and each fits only its own window.

#### Cam inscription

Which face is which: the cam prints groove face up, so the letter face (the back face) is the bed side, the cam's underside as
printed. The groove face holds the groove and the entry channel, and the groove's walls leave no clear area for a letter of
this size, so the letter is not on the top. Installed, the groove faces the front and the letter faces the rear wall; the
letter is for identifying the cam before it goes on the shaft. Moving it to the groove face needs a smaller letter or a
lower-contrast mark; it is left as built and flagged for the owner.

Each cam carries a letter, engraved 0.5 mm deep in its back face (the face opposite the groove, local z -3.8): A, b, C, d,
E, F and G for the segments a to g, the hexadecimal letters of a seven-segment display. A letter is the union of its
segments' rectangular strokes, 0.9 mm wide (above the 0.8 mm printable minimum), 3.2 mm wide and 4.6 mm tall, centred 14
mm from the cam axis straight up in the cam's local frame (the sketch reads from outside the face, so the letter reads
correctly from the rear). Each outline is a closed run of horizontal and vertical lines, each held by its distance from the
sketch origin on the cam's axis, so the sketch is fully constrained with no point fixed. The letter lies between r 11.7 and
r 16.3: clear of the bore (3.25), the hub (5), the mark holes (radius 8 or 12 at 270 degrees and beyond), the groove (which
is on the other face) and the rim (23.5). Behind the groove (r 14.6 to 16.3) 1.4 mm of disc stays between the letter's
floor and the groove's floor. The mark holes stay: "Look-alike part marks" requires them, and the letter is for
people only.

#### Part fits

| ID | Parts | Class | Modeled | Measure |
| --- | --- | --- | ---: | --- |
| P1 | cam double-D bores and the hub's D bore / shaft | press | 0.05 radial | section mid-bore |
| P2 | wall seats / 608ZZ outer race | bearing seat | 0.05 radial | section |
| N1 | shaft journals / 608ZZ bore | nominal exception | 0 | — |
| T1 | front step / front bearing inner face | Tight | 0.25 | bounds |
| T2 | rear collar face / rear bearing inner face | Tight | 0.25 | bounds |
| T3 | slider stem / guide slot | Tight | 0.25 | section z 56 |
| R1 | pin / groove walls | Regular | 0.40 radial | `meshgap` per digit |
| R2 | pin tip / groove floor | Regular | 0.40 | bounds |
| R4 | hub / front wall outer face | Regular | 0.40 | bounds |
| L1 | stem / own cam face | Loose | 0.55 | bounds |
| L4 | cam a back face / rear collar cone | Loose | 0.55 axial | bounds |
| L5 | cam rims / guide plate | Loose | 1.0 | bounds |
| L6 | b-f arm outer ends (x 17.9, −19.5) / ribs: no common y (arms within y ±26.05, ribs beyond y ±28.8); arm lower end (OFF) / guide plate top | Loose | 2.75; 1.0 | bounds, section |
| L8 | stub flat end / hub D chord; cam flats' ends / cam a's chords | press | 0.05 | bounds |
| S1 | plates / ribs and wall tops | clamped seam | 0.25 | bounds |
| F | bolts, nuts | fastener rules | head 0.3, hole 0.3, nut 0.2 | bounds, no overlap |
| K1 | rotor plate / housing tube | Regular | 0.35 | dimensions (`TestClickerRadialFits`) |
| K2 | rotor skirt / spring | Tight | 0.55 | dimensions (`TestClickerRadialFits`) |
| K3 | hub key / rotor skirt slot | Tight | 0.2 a side | dimensions |
| K4 | button head / crown ring; lug / housing slot | Loose | 0.55; 0.5 | dimensions |
| K5 | mating teeth: tip against root | land gap | 0.02 | specs (`TestClickerLandGaps`) |

Measured on the built bodies: P1, P2 (`TestSevenSegmentBearings`), P3, T1, T2, T3, R1, R2, R4, L1, L4, L5, L8,
and the S1 and S2 rows for the guide plate and the
columns (`TestSevenSegmentFits`, `TestSevenSegmentAxialFits`, `TestSevenSegmentPinClearance`). N1 is checked as no overlap
(`TestSevenSegmentBearings`) and F by `TestSevenSegmentFasteners`. Not measured on the built bodies yet: L6 (the arm to the
rib and the guide plate), P4 (the peg collar to the cradle's mouth cone) and the S1 seam of the top plate to the cradle and of
the plates to the wall tops. The fit table at the end of "Flag mechanism" has its own gaps: T6, R5, L9, L10 and L12 are not
measured (L11 is measured by `TestSevenSegmentBladeRoof`). A first measurement of L9 on the sections at z 80.5 reads 0.45
between a top plate tube and the cradle passage against the 0.55 listed, which has not been traced.

Clearance adjustment: **Mixed** (fixed custom fits as constants; nominal bearings and fasteners).

#### Corners and roots

Print orientations (cornercheck axis): walls outer face down (axis ∓Y: front +Y up, rear −Y up); guide plate bottom face down (+Z); top plate display face down (−Z up); cams back face down;
hub, rotor axis vertical (teeth up); button head down, teeth up; housing bezel face down; sliders -X face down (a, g, d) or -Y face down (b-f); rotors yellow face down; pivot blocks outer face down, pin up (cornercheck axis -u); shaft as split halves.

Exemptions passed to `cornercheck` (each must match a finding in every case, as onewayclutch `corners_test.go`):

| Part | Corner | Reason | Exempt region |
| --- | --- | --- | --- |
| Guide plate | 6 corners of each G hex pocket | mating corners with the nut | `Annulus` about each G site |
| Walls (ribs) | pocket-floor corners of T hex pockets and G head pockets, seen in the Y-vertical sections | pocket floor meets the bolt hole; a round would fill the nut's or head's seat | `Disc` r 3.0 about each site's pocket floor centre |

#### Retained sharp corners

Corners recorded but not exempted (they pass the checker or lie between layers; the test does not exempt them):

| Part | Corner | Why no finding |
| --- | --- | --- |
| Cams, hub | D bore corners | mating with the shaft flats; turn ≈ 30°, passes |
| Rotor, button, housing crown | tooth tips (cut back 0.05) | engagement edges of the face teeth: exempt, the tip is where the teeth meet |
| Rotor, button, housing crown | tooth roots (a land of 0.03 above the sharp valley) | retained sharp corners, a documented exception for face teeth only: the 0.5 round of "Corners and roots" does not fit a 26 degree valley at r 3.2, and a round would fill the notch the mating tip enters |
| Walls | seat floor rim | between layers; mating with the bearing's outer race |
| Top plate, housing | head pocket floor rims | between layers; 0.3 head clearance leaves no room |
| Cams | engraved letter: the 90 degree corners of 0.9 mm strokes, too thin for a 0.5 mm round | listed and exempt: `Disc` r 4.2 about the letter's centre |
| Cams | groove floor rims, entry channel floor rims | between layers; a round would reach the pin (0.4 tip gap) |
| Sliders | tongue roots 0.8, blade roots 0.6, arm root 1.25 | rounded |

Split-shaft exemptions (`cornercheck.CheckAt`): front step and rear collar rear face (square, facing a purchased
bearing), cam flats' square ends with cam a's chords, stub flat's square end with the hub's D chord
(mating pairs).

Rounded as required: groove concave walls 0.6, window tips 0.6, slot corners 0.55, rib roots 0.5,
hub roots 0.5, stem corners 0.3 (convex). Press-fit walls ≥ 1.6: cam hubs 1.75, keyed hub 3.45 or more.

#### Surface edge finishing

| Part | Face(s) ≥ 400 mm² | Action |
| --- | --- | --- |
| Walls | outer and inner faces | 0.5 chamfer on the outer face loop and the inner face's long runs; seat and lip rims 0.5 (in the tool) |
| Guide plate | both faces | outer loop 0.5; slot rims retained (guide length) |
| Top plate | both faces | outer loop 0.5; window rims retained (lip 1.512 / 1.248 at Scale 1.5, over 0.8) |
| Cams | both disc faces | rim 0.5 (meridian); groove and channel rims retained (pin bearing depth) |
| Housing | base flange faces | outer loop 0.5 |
| Sliders, shaft, hub, rotor, button | none ≥ 400 (largest slider face ≈ 235) | none |

#### Split shaft

| Part | Split plane (θ = 0) | Held by |
| --- | --- | --- |
| Shaft | x = 0: holds the axis and the flats' normal +Z | two 608ZZ bearings + press-fit bores (cams, hub); no cross bolts |

The cam section carries a double flat (±2.7), which "Flats" allows; the stub keeps one flat facing +Z.

Tests: `splitcheck.Check` (no `Normal`; flats name the plane) and `cornercheck.CheckAt` at
`cornercheck.SplitHeights(point, normal, 2.5)`, exempting only the split-shaft exemptions above. Run both on the
posed shaft of Digit 0 and Digit 2 (θ 0 and 72); require the split normal to be the rotated +X.

#### Arbor collars

| Collar | Carried part | Flush side | Extent (Y) | Length | Diameter | Option |
| --- | --- | --- | --- | ---: | ---: | --- |
| Front step | cams (fitting side) | front bearing inner face −49.5 | step at −49.25 | — | 6.4 | two-bearing step rule; the step lies inside the cams' stack, nothing pushes the stack forward |
| Rear collar | cam a | rear bearing inner face 43.0 | [39.7, 42.75], cone [39.7, 40.739] | 3.05 | 7.6 | 1: gap 3.85 less 0.25 and 0.55 |
| Hub | rotor (end outside the frame) | front wall outer face −57.5 | [−61.4, −57.9] | 3.5 | 21 | 1, separate keyed hub located by the stub flat's square end: the front bearing slides on from the front end, so a body collar there would block it |

Rear collar diameter 11.4 < 608ZZ inner ring face 12. Front step radius 4.8 ≥ 4.0 + 0.3 chamfer clearance + 0.3.


## Printing and assembly

Assembly (README and instructions.md):

1. Press cams onto the shaft from the front end, groove faces toward the front, in order a, f, b, g, c, e, d; cam a's
   chords stop on the flats' ends; push the stack rearward to that stop.
2. Slide the front bearing onto the front journal and the front wall over it. Slide the rear bearing on and press the
   rear wall onto it.
3. Press the hub onto the stub until it stops on the end of the stub's flat. Drop the spring over the stub, inside the
   hub's cup, then the rotor (its skirt slots over the hub's keys, teeth up).
4. Bolt the housing to the front wall's outer face with two M2 x 10 bolts, heads in the base's pockets, nuts in the
   wall's inner-face pockets (drop each nut in before the bolt). The rotor's outer teeth must sit in the crown's notches.
5. Bolt the guide plate onto the ledge ribs from below (4 × M2 × 6, nuts dropped into its top pockets). The two
   front bolts go up in front of cam d's disc, the two rear bolts between the discs of cams f and a.
6. Slide the button into the housing, its three lugs in the housing's slots, teeth down. Turn the shaft by pressing the
   button until all seven entry channels are at the top (digit 8). Drop each slider through its guide slot,
   pin first, into its cam's channel.
7. See "Assembly" under the flag mechanism.
8. Press through 0-9 and check each digit. Each press must turn the display one digit forward and never back.

## Wall inset (decad constraint)

The wall plates are 49.8 wide (x +-24.9) and 78.25 tall, z 3.0 to 81.25, so every rib ends 1.5 mm inside the plate's sides and
top. The inner face's outer loop is then a plain rectangle that decad chamfers whole, and each rib root is one closed
loop that decad rounds with a complete-loop fillet (selected with `decad.Reflex()`). The top plate is 46.8 wide and sits
between the walls (y -49.25 to 42.75, as the guide plate does), on the top flange ribs at z 79.75 with a 0.25 seam.

## Blade corners (design changed by the implementer)

The design held the blade 0.25 (Tight) from its tube hole on every side. The hole is an inside corner of the top plate,
so its corners take a round of at least 0.5 mm ("Corners and roots"). The blade's corners are square, so the least gap at
a corner is `c sqrt 2 - r (sqrt 2 - 1)` (0.126 for `c` 0.25 and `r` 0.55). Rounding the blade's ends by 0.3 (an `Intersect`
of a plan prism and the gable prism) builds, but decad refuses to union that result to the head. So:

- The hole's corner round is 0.50, the least allowed. The blade keeps `bladeAcross` 0.25 from the walls across its thin
  side and `bladeAlong` 0.35 along its length. The least gap at a corner is then 0.21 (corner radius 0.50, 0.25 across and
  0.35 along), above the Tight minimum 0.2 (a listed use of the minimum; the Tight fit is 0.25). Test T9 measures 0.25 on
  the straight walls and at least 0.20 at the corners.
- The blade is 1.1 thick (it was 1.2): its rotor-side face stands 0.1 back (a/g/d: y_i - 6.7; b/c: x 9.8; e/f: x -9.8).
  The hole's rotor-side wall is where it was (a/g/d y cy - 6.45), so the tube wall between the hole and the rotor bore
  keeps 0.80 or more. The tube footprints keep `tubeWall` 0.80 around the hole on the other sides.
- The pin line's play along s is `PinLinePlay` = 0.47: the blade across its hole 0.25, the pegs 0.05, the yaw 0.08 and the
  settled rotor-on-pins play 0.09. With the rotor fully across it is 0.58.
- Every slider type builds, and the volumes are 979.14 (a, g, d), 981.64 (b, c) and 996.01 (e, f) before the thinner pin.

## Sketches that still place computed points (exception to the sketch rule)

Most sketches are held by geometry: `polygon()` holds each side by its direction and its distance from an anchor, the
rotor section, the plates' rectangles and circles, the meridians of the bores, the shaft's flat tool and the slider profiles
all verify fully constrained with no computed vertex fixed. Four places still fix computed points, and each is listed:

- Faceted arcs (the 8-chord rounds): `polygon()` fixes the vertices of a run of short slanted sides from its anchor, because
  the kernel refuses true arcs there ("mesh bound" and chain refusals, see the construction notes) and the chords stand for an
  arc. Smallest repro: the tongue's end cutter drawn with every chord held by direction and distance fails verification with
  "near-singular constraint system: conditioning 2.1e-05 is below 4e-05".
- `tongueOutline` (`sliders.go`): the strip sides, the feet, the chord roots and the hull's tangent points are dimensioned
  points around one real arc (radius 0.6, tangent lines). It has the same chord rounds; making its straight sides geometric was
  not attempted.
- The face teeth's tool profiles (`facetooth.go`) are fully constrained rectangles (a polygon held by direction and distance). They place no computed point.

Circles and their centres are not on this list: each hole, bore and pocket circle is placed by two distances from the
sketch origin and sized by its radius, which is an anchored dimension and not a computed point.

These four are exceptions only because AGENTS.md has no clause for them. Its sketch rule bans fixing every computed point to
force `DOF == 0` and asks for geometric and dimensional constraints; it says nothing about a chord run, a spline sample
or a dimensioned outline point that no constraint can hold.

## Turned-over b-f pose

Turned over, the b-f sliders tilt the other way. `TestSevenSegmentCaptive` poses that by a rigid shift and a turn about the
groove pin and the slot, at half the tilt limit, the rise 0.2 short of the full rise and the flags lowered 0.11. In that
pose no b-f slider vertex goes deeper than the mesh tolerance 0.07 into the top plate or into its black body (the deepest
are 0.038 and 0.058): the clearance there is about zero by design (the blade meets its tube roof and the pin its pocket at
the end of the groove's play), which a mesh cannot prove separated. The pose is an approximation, not a solved
configuration, so the check shows no gross overlap and not a clearance. The a, g and d sliders and every flag on its pivot pins are
tested to a gap of at least 0.
