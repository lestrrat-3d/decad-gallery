# Turbine fan

## Purpose and source

Construct a fan of three replaceable blades on one printed hub. The hub is a plain round cylinder, radius 14 and 20 long,
with a D bore for a D-flat shaft. One radial M2 x 6 bolt acts as a set screw: it threads through a nut in a hex slot open
to the hub's back face, and its tip presses the shaft's flat. Its head floats, recessed, in a counterbore cut into the
round face, so nothing stands outside radius 14 except the three blades. Each blade is the hand-crank fan's blade
([`models/handcrankfan`](../handcrankfan/)), with its round root stud that slides into a radial socket in the hub and one M2
bolt that runs parallel to the shaft through the hub and the stud into a nut. That nut sits in a hex slot open to the round
face, below the blade, so no nut is embedded and the hub needs no print pause. The parts are:

- a hub, a cylinder cut by the D bore, three sockets, the blade bolts' pockets and nut slots, and the set screw's
  counterbore, hole and nut slot;
- three blades, one part printed three times;
- four M2 bolts (three M2 x 16 blade bolts and the M2 x 6 set screw) and four M2 nuts.

The shaft is not modeled. It belongs to the mechanism, and its diameter equals the bore of the purchased bearing it runs
in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets read that bore from
[`models/ballbearing`](../ballbearing/). The bolts and nuts are [`models/m2bolt`](../m2bolt/) and
[`models/m2nut`](../m2nut/), placed with `internal/m2fastener`.

All dimensions are in millimeters; angles are radians in code and degrees in prose. The model reuses `handcrankfan` by
import and copies no geometry: it exports `BuildBlade`, `BuildBladeMirrored`, the blade socket tool and the hub and
blade-joint constants. It reads `PressWall`, `ChamferSize`, `RegularFit` and the shaft flats from `crankhandle`.

## Symbols

| Symbol | Meaning | Formula |
| --- | --- | --- |
| `D`, `d`, `c` | shaft diameter, flat depth, bore clearance | `ShaftDiameter`, `FlatDepth`, `BoreClearance` |
| `r_b` | bore radius | `D/2 + c` |
| `f_b` | distance from the axis to the bore's flat; the flat lies at y = -`f_b` | `D/2 - d + c` |
| `w_b` | width of the bore's flat | `2 sqrt(r_b^2 - f_b^2)` |
| `L_b` | length of the bore's outline | `2 r_b (pi - acos(f_b / r_b)) + w_b` |
| `w_s` | width of the shaft's flat | `2 sqrt(d (D - d))` |
| `rho_t` | the set screw's tip, as a distance from the axis along -Y | `D/2 - d - c` |
| `rho_h` | the set screw's head top | `rho_t + HeadPocketDepth + SetScrewBolt` |
| `rho_f` | the counterbore's floor | `rho_h - HeadPocketDepth - SetScrewTravel` |
| `u_k`, `v_k` | blade k's radial direction and its counter-clockwise tangent | `(cos t_k, sin t_k, 0)`, `(-sin t_k, cos t_k, 0)`, `t_k = 90 + 120 (k - 1)` degrees |

`rho` is always the distance from the shaft axis along -Y.

## API

Package: `models/turbinefan`. Exported surface:

- The fixed dimensions as constants: `BladeCount` (3), `HubRadius` (14), `HubLength` (20), `HubMid` (10), `FirstBladeAngle`
  (90 degrees), `SetScrewBolt` (6), `SetScrewZ` (4), `SetScrewTravel` (0.4), `CounterboreRound` (0.5), `NutSlotRound`
  (0.5), and the values it takes from `crankhandle`: `ChamferSize`, `PressWall` and `RegularFit`. `BladeAngle(k)` is blade
  k's angle about +Z.
- The container names `HubContainerName` (`Hub`), `BladesContainerName` (`Blades`), `BoltsContainerName` (`Bolts`),
  `NutsContainerName` (`Nuts`) and `ClampContainerName` (`Clamp Ring`).
- The clutch-mount constants `ClutchTurn`, `FlangeThickness`, `FlangeRadius`, `FlangeRound`, `ClutchBoreClearance`,
  `ClutchSeam`, `ClampThickness`, `ClampInnerRadius`, `ClampOuterRadius`, `ClampBack`, `EarBoltLength`, `EarBoltCavity`,
  `MinWall` and `MountZ` (see "Clutch mount").
- `Hand` with `RightHand` (the zero value) and `LeftHand`; `Hand.Freewheel()` returns the `onewayclutch.Direction` that
  drives a fan of that hand forward.
- `Mount` with `ShaftMount` (the zero value) and `ClutchMount`; `EarBoltAngle(k)` is ear bolt k's angle in the fan frame.
- `Params` with `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `BladePitch`, `Hand`, `Mount` and `Placement *r3.Frame`
  (nil selects the local frame).
- `MountFrame(clutch *r3.Frame) (r3.Frame, error)` and `ParamsForClutch(onewayclutch.Params) (Params, error)`.
- `Params605ZZ()`, `Params608ZZ()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`. `Dimensions` holds `BoreRadius`, `BoreFlat`,
  `BoreFlatWidth`, `BoreLoop`, `ShaftFlatWidth`, `SetScrewTip`, `SetScrewHead`, `CounterboreFloor`, `SetScrewSite`,
  `BladeSites`, `ClutchBoreRadius` and `EarSites`, in the local frame.
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.1 mm), opts...)`. The blade's stud
  union refuses a plain document; build in such a document.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the hub and the three blades and puts them in no container. `Parts`
  holds `Hub`, `Blades`, `Clamp` (nil in `ShaftMount`), every `Sketches`, `Dimensions`, and in the world frame `BladeSites`,
  `SetScrewSite` and `EarSites`.
- `Build(ctx, doc, p) (*Model, error)` adds the parts, the bolts and the nuts, each in its container. `Model` embeds `Parts`
  and holds `BladeBolts`, `BladeNuts`, `SetScrew` (`*m2bolt.Model`) and `SetScrewNut` (`*m2nut.Model`), and in
  `ClutchMount` `EarBolts` and `EarNuts` (the set screw and its nut are nil).

The exports this model needs from other items: `handcrankfan.BuildBlade(ctx, doc, pitch)` returns the unplaced blade and its
sketches, `BuildBladeMirrored` the blade of the other hand, `BladeSocketTool(ctx, doc, f)` builds a socket cutter; and
`HubRadius`, `HubLength`, `BladeBoltAt`, `BladeBoltLength`, `BladeBoltCavity`, the stud and socket constants,
`DefaultBladePitch`, `MinBladePitch`, `MaxBladePitch`, `BladeRootRadius()`, `BladeTipRadius()` and `Beta(r, pitch)`. From
`crankhandle`: `PressWall`, `ChamferSize`, `RegularFit`, `Params605ZZ` and `Params6082RS`.

## Presets and defaults

| Field | `Params605ZZ` (default) | `Params608ZZ` |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params608ZZ().BoreDiameter` |
| `FlatDepth` | 0.5, the planetary gearbox shafts' flat | 0.75, the siren shafts' flat |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `BladePitch` | 25 degrees, `handcrankfan.DefaultBladePitch` | 25 degrees |
| `Hand` | `RightHand` | `RightHand` |

| Derived | 605ZZ | 608ZZ | `regular-bore` (605ZZ, c 0.35) |
| --- | --- | --- | --- |
| `r_b` | 2.55 | 4.05 | 2.85 |
| `f_b` | 2.05 | 3.30 | 2.35 |
| `w_b` | 3.0332 | 4.6957 | 3.2249 |
| `L_b` | 15.81 | 25.12 | 17.70 |
| `w_s` | 3.0 | 4.6637 | 3.0 |
| `rho_t` | 1.95 | 3.20 | 1.65 |
| `rho_h` | 9.95 | 11.20 | 9.65 |
| `rho_f` | 7.55 | 8.80 | 7.25 |

A 608-2RS has the same bore as a 608ZZ. `rho_t` is the flat of a shaft pushed `c` toward +Y until its round meets the bore's
round, which is where the tightened screw leaves it. The tip then stands `2c` (0.1 at the presets) inside the bore's flat
(exception X2).

## Validation

Every message starts with `turbinefan: `. Derive checks in this order and returns the first failure:

1. `ShaftDiameter`, `FlatDepth`, `BoreClearance` and `BladePitch` are finite and positive: `<Field> must be finite and
   positive`.
2. `d <= D/4`.
3. `ballbearing.PressFitClearance <= c <= 0.5`.
4. `Mount` is `ShaftMount` or `ClutchMount`, else `Mount must be ShaftMount or ClutchMount`.
5. `ShaftMount`: `BladeSocketStart - r_b >= PressWall`, so 1.6 mm stays between the bore and the socket floors at r 5.7:
   `r_b` at most 4.1. `ClutchMount`: `BladeSocketStart - r_c >= MinWall`, else `ShaftDiameter leaves less than a 0.8 mm wall
   between the clutch mount's bore and the blade sockets`.
6. `ShaftMount` only: `w_s >= m2bolt.ShankDiameter`, else `FlatDepth must leave the shaft's flat at least 2 mm wide for the
   set screw's tip`.
7. `ShaftMount` only: `w_b >= 2 (HoleRadius + 0.1)`, else `FlatDepth must leave the bore's flat at least 2.8 mm wide for the
   set screw's hole`. The 0.1 keeps the hole's cylinder off the bore's round.
8. `BladePitch` lies in [10 degrees, 40 degrees].
9. `Hand` is `RightHand` or `LeftHand`.
10. `Placement` is nil or a valid frame.
11. The site checks, constant over the valid range. `ShaftMount`, in this order: `SetScrewSite.Validate()` (`set screw:
    ...`); `rho_h <= sqrt(HubRadius^2 - (HeadDiameter/2)^2)` (13.8705), else `set screw: its head would stand out of the
    round face`; `Cavity - HeadPocketDepth - SetScrewTravel >= 1`, else `set screw: less than 1 mm between its counterbore
    and its nut slot`; `SetScrewZ - HeadPocketRadius >= 1`, else `set screw: less than 1 mm between its counterbore and the
    back face`; then each `BladeSites[k].Validate()`. `ClutchMount`: each `BladeSites[k].Validate()` (`blade bolt %d: ...`),
    then each `EarSites[k].Validate()` (`ear bolt %d: ...`).

`FlatDepth` 0.2 on the 605ZZ fails rule 6. 0.25 passes rule 6 and fails rule 7.

BuildParts and Build reject a nil context and a nil document, derive the dimensions, and check `ctx.Err()` before any
geometry. A rejected build adds no body.

## Local frame and hand

The origin is on the shaft axis in the hub's back face, the face toward the mechanism. +Z runs along the shaft away from
the mechanism to the front face at z = 20. The bore's flat faces -Y, at y = -`f_b`, and the set screw enters from -Y along
the line x = 0, z = `SetScrewZ` = 4, pointing +Y. Blade k lies along `u_k`: blade 1 along +Y, blade 2 at 210 degrees and
blade 3 at 330 degrees, counter-clockwise seen from the front. The set screw lies at 270 degrees, midway between blades 2
and 3. A `Placement` maps this frame onto the given one after each part's last operation.

Hand: a right-hand rotor blows air forward, along +Z and away from the mechanism, when the shaft turns clockwise seen from
the front. A left-hand rotor does the same when the shaft turns counter-clockwise: its blades are the right-hand blades
reflected in the plane x = 0 (`handcrankfan.BuildBladeMirrored`), which maps blade 1 to itself and swaps blades 2 and 3.
The hub, bolts and nuts do not depend on the hand. For the right hand, a blade's chord from its trailing edge to its
leading edge at radius r runs along `(cos beta, 0, -sin beta)` with `beta = handcrankfan.Beta(r, BladePitch)`: the leading
edge leads toward +X at the top and sits upstream (-Z), and the concave pressure face faces the front.

## Geometry

1. **Hub outline.** A sketch on z = 0: the circle of radius 14 about the axis; the D bore, an arc of radius `r_b` about the
   axis from `(w_b/2, -f_b)` counter-clockwise over +Y to `(-w_b/2, -f_b)`, closed by its flat, a horizontal line at distance
   `f_b` from the origin; and three circles of radius 1.3 at `BladeBoltAt` (9.5) along `u_k`, the blade bolts' holes.
   The profile has four holes: the three bolt holes and the bore. The bore runs through.
2. **Hub body.** Extrude 20 along +Z, then `edgefinish.Chamfer` of both faces by `ChamferSize`, taking the long loops: the
   outer circle on both faces and, when `L_b` is 20 or more (608ZZ), the D loop. Then cut, in this order:
   - the three sockets: `handcrankfan.BladeSocketTool` on the frame with origin `(0, 5.7, 10)`, U = +Z and V = +X (normal
     +Y), and its two copies from `decad.PatternCopies` with a circular pattern of three about +Z;
   - for each blade k, `m2fastener.PocketTool` (head pocket, z 18 to 20, radius 2.2) then the blade nut slot (below);
   - the set screw's counterbore (below), `m2fastener.HoleTool(SetScrewSite)` (radius 1.3, from `rho_h + 1` to `rho_t - 1`, its
     inner end inside the bore) and the set screw's nut slot (below).
3. **Nut slot.** A hexagon about the bolt line with its corners on the circle of radius 2.51, one corner at the slot's end,
   whose two sides toward the opening run on as straight walls 2.1737 from the line, out through the part's face; 2.0 deep.
   - Set screw slot: plane `y = -(f_b + 1.6)`, extruded to `y = -(f_b + 3.6)`, U = -Z and V = +X, reaching `z = 6.51`
     at its end corner and open at z = -1 (below the back face). Every edge of both slot faces is rounded by `NutSlotRound`.
   - Blade slot k: plane z = 2.4, U = `u_k`, V = `v_k`, the corner at r 6.99 on the radial line, open through the rim; z 2.4 to
     4.4. Its three hexagon corners, the vertical edges along Z where the end meets the walls, are rounded by `NutSlotRound`: each turns 60 degrees, so a layer would read it as an inside corner.
4. **Counterbore.** A revolved meridian about the screw's line: from the floor at `rho_f` a quarter-round of radius
   `CounterboreRound` (eight chords) up to radius 2.2, out through the round face. The head (radius 1.9) floats
   `SetScrewTravel` above the floor, where the round's surface is at radius 2.19, so it clears it by 0.29.
5. **Blades.** `handcrankfan.BuildBlade(ctx, doc, BladePitch)` (`BuildBladeMirrored` for `LeftHand`) returns blade 1 in its own
   frame (origin on the fan axis in the hub's mid plane, +Z its radial line, +Y downstream, the blade moving toward -X).
   `decad.Transform` by the frame with origin `(0, 0, 10)`, U = (-1, 0, 0) and V = (0, 0, 1) turns it into place: blade-local
   Z becomes +Y, local Y becomes +Z and local X becomes -X. `decad.PatternCopies` with the hub's pattern gives blades 2 and
   3. Do not transform blade 1 again before copying.
6. **Blade facts.** From handcrankfan: a loft from r 14.4 to 64 through ten stations; a stud of radius 3 from r 6.1 to 16
   with a 0.3 chamfer on its inner end; a bolt hole of radius 1.3 along the fan axis at r 9.5 through the stud; sections
   that are NACA 4-digit mean lines blended with an ellipse at the root; a blade angle `Beta(r, BladePitch)` that is
   constant inboard of r 21.5; a trailing edge 1.0 thick. The rotor is 128 across. Every blade vertex lies between z 0 and
   20: the measured range is z 5.02 to 17.72 at the default pitch, 4.17 to 19.39 at 40 degrees and 5.85 to 15.55 at 10
   degrees.
7. **Containers.** `Build` files the hub, the three blades, the four bolts and the four nuts in their containers, each a
   child of the document root.

## Layout numbers

| Item | 605ZZ | 608ZZ | `regular-bore` |
| --- | --- | --- | --- |
| Hub envelope | x, y [-14, 14], z [0, 20] | same | same |
| Bore flat at y | -2.05 | -3.30 | -2.35 |
| Set screw: tip rho, head top rho | 1.95, 9.95 | 3.20, 11.20 | 1.65, 9.65 |
| Head top recessed below the round face at x = 0 | 4.05 | 2.80 | 4.35 |
| Counterbore r 2.2, floor rim round 0.5: floor rho; z | 7.55; 1.8 to 6.2 | 8.80 | 7.25 |
| Set screw hole r 1.3: rho | 2.05 to 7.55 | 3.30 to 8.80 | 2.35 to 7.25 |
| Set screw nut slot: rho; x; z | 3.65 to 5.65; +-2.17; 0 to 6.51 | 4.90 to 6.90 | 3.95 to 5.95 |
| Set screw nut: rho; x; z | 3.85 to 5.45; +-2.0; 1.69 to 6.31 | 5.10 to 6.70 | 4.15 to 5.75 |
| Wall, bore's flat to the slot's inner face | 1.60 | 1.60 | 1.60 |
| Wall, bore's corner to the slot's nearest edge | 1.73 | 1.60 (the flat covers the slot) | 1.70 |
| Material, slot's outer face to counterbore floor | 1.90 | 1.90 | 1.30 |
| Material, counterbore to back face | 1.8 | 1.8 | 1.8 |
| Blade nut slot: from the corner at r 6.99 out through the rim; tangential; z | +-2.17; 2.4 to 4.4 | same | same |
| Blade nut: along `u_k` r 7.19 to 11.81; z | 2.6 to 4.2 | same | same |
| Blade nut slot to socket, back face, bore | 2.2, 2.4, 4.44 | 2.2, 2.4, 2.94 | 2.2, 2.4, 4.14 |
| Set screw slot to blade nut slots, to blade bolt holes | 3.88, 4.76 | same | same |
| Counterbore to blade nut slots | 4.15 | 4.76 | 4.05 |
| Set screw features to sockets (lower bound) | 1.90 | 2.13 | 1.94 |
| Blade nut slots to each other | 12.1 | same | same |
| Wall, bore to socket floors (r 5.7) | 3.15 | 1.65 | 2.85 |
| Bore flat width / shaft flat width | 3.03 / 3.00 | 4.70 / 4.66 | 3.22 / 3.00 |

At 608ZZ the bore's 0.5 lead-in chamfer leaves 1.1 between the bore and the set screw slot over z 0 to 0.5. That band is
the lead-in, not pressed length.

## Fasteners

No nut is embedded: the set screw's nut and the blade nuts sit in hex slots open to a face (AGENTS.md "Fasteners"). Both
slots use the hexagon of the Fasteners rule (corners on the 2.51 circle, 2.0 deep). The slot's sides stand 2.17 from the
bolt line, less than the nut's 2.31 corner radius, so they stop the nut turning. The tightened bolt holds the nut: the set
screw pulls its nut toward -Y onto the slot's outer face, and a blade bolt pulls its nut toward +Z onto the slot's face at
z 4.4. The set screw's nut is wholly inside the hub (z 1.69 to 6.31); the blade nut's outer corner is at r 11.81, inside the
14 rim. The 0.5 rounds leave 0.07 of clearance at the nut's flats and 0.12 at each rounded corner of the blade slot (a 120 degree corner rounded by 0.5 takes 0.077 of the 0.2 past the nut's corner).

| Bolt | Head | Hole | Nut | Rule |
| --- | --- | --- | --- | --- |
| Blade bolt k (M2 x 16) | flush in the front face, pocket radius 2.2, 2 deep | radius 1.3 through the hub and the stud | hex slot open to the round face, entered radially, z 2.4 to 4.4 | open hex seat |
| Set screw (M2 x 6) | recessed in a counterbore radius 2.2, 0.4 above its rounded floor (X1) | radius 1.3 from the counterbore floor through the press-fit wall (X3) into the bore | hex slot open to the back face, entered along +Z | open hex seat |

`Build` places the set screw with `m2fastener.PlaceBolt` at `SetScrewSite` and its nut with `PlaceNut`. `SetScrewSite` is
`m2fastener.Site{Head: (0, -rho_h, SetScrewZ), Dir: (0, 1, 0), Corner: (0, 0, 1), Length: SetScrewBolt, Cavity:
rho_h - f_b - PressWall - NutCavityHeight}`; `Cavity` is 4.3 at the presets, so the slot spans `f_b + 1.6` to `f_b + 3.6`
and its inner face keeps exactly `PressWall` to the bore's flat. `Corner` +Z puts the nut's corners up and down and its
flats on +-X. `Tip()` is 8, so the tip lies at `rho_t`. The blade bolt sites are handcrankfan's, turned about the shaft
axis for blades 2 and 3: head at `BladeBoltAt` from the axis along the blade's radial line at z 20, `Dir` -Z, `Corner` the
radial line, `Length` 16 and `Cavity` 15.6; the blade nut slot takes the place of the round cavity.

Why a slot open to the back face does not work for the blade nuts: a blade bolt runs along -Z, so a seat open to the back
face is a pocket coaxial with it at z 0 to 2. The M2 x 16 tip at z 2.0 does not reach a nut at z 0.2 to 1.8, and the back
face is against the mechanism, so with the hub on its shaft the nut would fall out behind the hub whenever a blade bolt is
undone and could not be put back. Replacing a blade with the hub on its shaft is a requirement. A slot open to the round
face, along the blade's radial line, works: its mouth is on the rim at z 2.4 to 4.4, below the blade root. The lowest blade
point over the nut's last 4.6 mm of travel (r 13 to 19) is z 5.94 / 5.55 at 25 degrees and 5.98 / 5.22 at 40 degrees,
against the nut's top at 4.2. Beyond r 19 the steep blade comes down to z 4.20, so tweezers bring the nut in from below,
tilted. A tangential slot was rejected: avoiding the set screw forces two slots to converge, leaving 0.79 between them.

Set screw notes: the tip's face (radius 0.75 inside its 0.25 chamfer; shank radius 1.0) lies within the shaft's flat (half
width 1.5 at 605ZZ, 2.33 at 608ZZ); validation rule 5 keeps that flat 2 mm wide or more. Nothing stands proud: the head stays
2.8 or more inside the round face. The counterbore leaves 0.3 radial round the head and 0.4 under it, and 1.0 or more of
material under the counterbore and round every pocket.

Retention and strength, estimates only. The torque goes through the D flat by shape; the set screw removes the 0.05 play and
holds the hub along the shaft by friction at the tip and at the bore's far side. For a hand-tightened M2 (preload 100 to
300 N, friction 0.2) the axial hold is about 40 to 120 N and the friction torque about 0.09 to 0.27 N m; a hand-turned
128 mm fan's air load is estimated well under 0.05 N m and its thrust under 1 N. The nut's seat is backed by 1.9 of hub
(1.3 at `regular-bore`) before the counterbore; to fail, the nut must punch a plug about 2.1 in radius through it, a shear
area of about 25 mm2 (17 at `regular-bore`), or 375 to 500 N (255 to 340 N) at an assumed 15 to 20 MPa across layers.
What only a printed test settles: the preload the web takes before it cracks or creeps, the friction values, and how much
creep loosens the screw.

Balance: the net imbalance on the -Y side is about 1.1 g mm (605ZZ) and 1.8 g mm (608ZZ) at full infill, up to about 3.4 g mm at
sparse infill. The three blade slots are identical turned copies and cancel. No balance mass is added.

## Replaceable blades

- To remove a blade, undo its bolt from the front face. The blade nut is free in its slot: take it out through the slot's
  mouth below the blade root, or leave it. Pull the blade out along its radial line. The stud leaves the socket after 7.9 mm
  (the stud's inner end at r 6.1 to the hub's rim at r 14). The hub can stay on the shaft, since every blade bolt's head is
  on the front face.
- To fit a blade, slide an M2 nut, flat, into the slot's mouth in the round face below the socket until it stops at the
  slot's end; slide the stud into the socket with the concave face toward the front face; push the bolt in from the front
  through the stud into the nut and tighten it. All three blades are one part, and any blade fits any socket.
- A left-hand blade is a different part from a right-hand one: a fan takes three of one hand.
- The stud is round, so a blade also goes in turned half a turn about its radial line. That blade faces the wrong way. The
  README tells the builder which way it goes; the model has no key.
- To take the hub off its shaft, loosen the set screw about two turns; it stays in its nut.

## Part fits

The model is **Mixed**: the bore fit is adjustable and the blade fits, bolts and nuts keep their nominal size.

| Fit | Class | Built clearance |
| --- | --- | --- |
| D bore round the shaft | Press fit, the hub turns with the shaft | `c` = 0.05 radial, adjustable 0.05 to 0.5 (8 mm shaft: 0.05 to 0.1) |
| Set screw tip on the shaft's flat | contact by design | nominal exception: the shaft is not modeled; the tip sits on the pushed shaft's flat (X2) |
| Set screw head in its counterbore | Fastener rules | 0.3 radial, 0.4 under the head |
| Nuts in their slots | Fastener rules | 0.17 at the flats (0.07 at the rounds), 0.2 past the corners and at each face |
| Stud in its socket | Regular | 0.40 radial (3.4 against 3.0) |
| Stud's inner end to the socket floor | Regular | 0.40 (6.1 against 5.7), with the 0.3 stud chamfer clear of the 0.5 floor round |
| Blade root to the hub's rim | Regular | 0.40 (14.4 against 14) |
| Bolts, shank in the stud, bolt in its nut | Fastener rules / nominal exception | head pocket 0.3, hole 0.3, cavity 0.2; the bolt in its own nut is not measured |

The hub presses onto the shaft, flat to flat, and the set screw presses the flat. The shaft must reach at least 6 mm into the
hub (past z 6, beyond the hole's edge at 5.3) and should end at or before the front face. The bore runs through.

## Named exceptions

The user confirms these.

- X1. The set screw's head is recessed, not flush: 4.05 (605ZZ), 2.80 (608ZZ) and 4.35 (`regular-bore`) below the round face
  at x = 0. No standard M2 length ends flush, and a set screw's head must float. Nothing stands proud. The user approved
  this.
- X2. The set screw's tip stands `2c` (0.1 at the presets) inside the bore, on the flat of the shaft it presses. That is the
  set screw's job; it stands outside no outer face.
- X3. The set screw's 2.6 hole crosses the 1.6 press-fit wall round the bore, at z 2.7 to 5.3. The wall is 1.6 or more
  everywhere else over the pressed length.

## Valid range

- `ShaftDiameter` up to 8.1 at the default clearance. `FlatDepth` at most `D/4`, with the shaft's flat 2 mm wide or more and
  the bore's flat 2.8 mm wide or more (for D = 5 and c = 0.05: 0.42 to 1.25, the lower limit from rule 6, `d (5.1 - d) >=
  1.96`; rule 5 alone would allow 0.21). `BoreClearance` 0.05 to 0.5, and for the 8 mm shaft 0.05 to 0.1.
- `BladePitch` 10 to 40 degrees.

## Representative cases

- `605ZZ`: the default; the least wall to the set screw slot is 1.6.
- `608ZZ`: the largest bore, 1.65 to the socket floors, the D loop chamfered.
- `regular-bore`: `Params605ZZ` with `BoreClearance` 0.35, the thinnest web behind the nut, 1.3.
- `steep`: `Params608ZZ` with `BladePitch` 40 degrees.
- `flat`: `Params605ZZ` with `BladePitch` 10 degrees.
- Left hand: `Params605ZZ` with `Hand` `LeftHand`. Its test proves the mirror relation (each blade's volume and bounds equal the
  right-hand fan's mirrored blade reflected in x = 0) and that the blade nut slots stay reachable; the full per-blade checks
  run for the right hand.
- Placed: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.

## Retained sharp corners and edges

The rule is "Corners and roots" in `docs/model-pipeline.md`. The hub prints front face down with its axis vertical, and each
blade prints standing on its stud end with its radial line vertical.

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Hub | The two corners of the D bore | Mating corners of the shaft's flat. They turn about 36 degrees, so they need no round. |
| Hub | The set screw slot's corner edge at z 6.51 (along Y) and the blade slots' faces' edges (z 2.4 and 4.4) | The set screw slot's corner is a mating corner with the nut's corner and runs parallel to the bed; the blade slots' faces are between layers, sized by the Fasteners rule. |
| Hub | The counterbore's 0.5 floor round, where a layer off the screw's line cuts it at a slant | A layer at z 2.5, 3.4 and 5.5 chords the round into turns of 50.1 to 53.1 degrees; the test exempts that band only. |
| Hub | The set screw slot's 0.5 rounds where a layer cuts its sloped end (z 5.26 to 6.51) | Layers at z 5.5 and 5.8 read turns of 53.6 to 58.5 degrees; the test exempts that band only. |
| Hub | The blade head pockets' floor rims | Between layers. |
| Hub | The socket mouths | Between layers, out of the checker's sections. |
| Hub | The socket floor's 0.5 round, where a layer off the socket's axis cuts it at a slant | A layer at z 7 or 13 chords the round into turns of 57.8 to 59.8 degrees on the floor's outline, 5.7 to 6.6 from the shaft axis; the test exempts that band only. |
| Blade | The four corners where the bolt hole crosses the stud, at r 9.5 and 10.2 | The material angle is 64 to 90 degrees; the checker adds the stud's curvature (handcrankfan). |
| Blade | The step where the stud meets the root face (r 14.4), opening downward when printed stud end down | Between layers; inherited from handcrankfan, which does not list it. |

Rounds below half their section: the socket floor round only. The hub has no stressed root. Every hub wall is 1.1 mm or
more (the 608ZZ lead-in band), elsewhere 1.3 or more; the outward corners where the slots and the counterbore meet the round
face are about 99 degrees, above the 60 degree minimum.

Press-fit wall: 1.6 or more round the bore over its pressed length: 1.60 to the set screw slot, 1.65 to the 608ZZ socket
floors. The set screw's 2.6 hole crosses that wall (X3). The hub prints with its bore axis vertical.

Surface edge finishing: both faces are large (about 570 mm2 on the back face and 580 on the front at 605ZZ; 541 and 551 at
608ZZ). One `edgefinish.Chamfer` call chamfers both faces' outer circles by 0.5, and both D loops when 20 or longer (608ZZ,
25.1). Short runs are left sharp: the bolt holes (8.2), the blade head pockets (13.8), the set screw slot's mouth (12.7), and
the 605ZZ and `regular-bore` D loops (15.8, 17.7). The round face and the cuts in it are not planar.

Body containers: `Hub`, `Blades`, `Bolts` and `Nuts`. Look-alike part marks: none; the bolts and nuts are bought parts. Split
shafts: the model has no printed shaft, arbor, axle or pin; the blade's stud is part of the blade. Arbor collars: no
printed shaft either. The consuming mechanism's shaft carries the hub and owns the collar between the hub's back face and its
bearing or plate. Gear train clearances: no gears.

## Printing

| Part | Prints with | Pause |
| --- | --- | --- |
| Hub | front face down, axis vertical | none |
| Blade (three) | standing on the stud end, radial line vertical | none |

The set screw slot opens upward in print, its hexagonal end down. Each blade slot's roof (z 2.4) is a 4.35 wide bridge. The
counterbore and the holes are horizontal round holes 4.4 and 2.6 across.

To assemble:

1. Hold the hub back face up and drop an M2 nut into the set screw slot. It stops at the slot's hexagonal end, in line with
   the hole.
2. Push the M2 x 6 set screw into the counterbore on the -Y side and turn it into the nut about two turns. Stop while its tip
   is still out of the bore (look into the bore from the back).
3. Fit each blade (see "Replaceable blades").
4. Press the hub onto the shaft, flat to flat, the flat toward the set screw, at least 6 mm deep.
5. Tighten the set screw with a 1.5 mm hex key until its tip presses the flat.

## Checks and images

Check every case above: every sketch fully constrained (23 from `BuildParts` and 20 from `Build`), the twelve bodies and their
containers, the default's bounds and `VerifyAll` meshes, the placed fan against the local one, the blade envelope (every
blade vertex at 0 <= z <= 20 and r <= 64.5), the bore's clearance round the nominal shaft, its wall and that it runs through,
the face chamfers, the set screw (head, tip, floating head over a rounded floor, nut, slot shape, no overlap, every wall),
each blade bolt's head, tip and nut with its slot and no overlap, access to the set screw's path and each blade nut's path,
every Regular fit, the blades' pitch and hand, the copies turned by 120 and 240 degrees, the replacement paths, and
`internal/cornercheck` on the hub (axis +Z) and each blade along its radial line.

## Clutch mount

### Purpose, frame turn and stack

`Params.Mount = ClutchMount` bolts the fan to the three ring ears of [`models/onewayclutch`](../onewayclutch/) instead of
gripping the shaft. The shaft drives the fan through the clutch, and the fan coasts when the shaft stops or turns back.
The default (`ShaftMount`) fan does not change: same bodies, tests and images. The clutch is not modified; the fan side
supplies everything that holds the ear bolts, including a clamp ring behind the ring.

The fan local frame is unchanged (origin on the axis in the hub's back face, +Z forward, blade 1 along +Y). In
`ClutchMount` the flange lies at z -3 to 0, behind the hub. `MountFrame(C)` maps the fan frame to clutch-frame origin
(0, 0, `MountZ`) with U = (cos 60, sin 60, 0) and V = (-sin 60, cos 60, 0), then through the clutch frame C. A fan angle
is the clutch angle plus 60 degrees, so the clutch ears at 90, 210 and 330 degrees land at fan angles 30, 150 and 270,
midway between the blades, and a hex key reaches each ear bolt from the front. Both frames share +Z.

| Item | Fan z | Clutch z |
| --- | ---: | ---: |
| Hub front face | 20 | 43 |
| Flange front face, ear bolt head tops | 0 | 23 |
| Flange back face | -3 | 20 |
| Ring front end (seam 0.25) | -3.25 | 19.75 |
| Clutch cover outer face, cover bolt heads | -4.5 | 18.5 |
| Ring back end | -24.25 | -1.25 |
| Clamp ring front face (seam 0.25) | -24.5 | -1.5 |
| Ear nut slot | -25.5 to -27.5 | -2.5 to -4.5 |
| Ear bolt tip (seams open) | -27.0 | -4.0 |
| Clamp ring back face | -28.5 | -5.5 |

The clamped faces are modeled with the clutch's 0.25 mm seam and close when the bolts are tightened, as the clutch's own
plates do. The clutch-mount bore is round, radius `D/2 + 0.55`, and does not touch the shaft. `FlatDepth` and
`BoreClearance` stay validated by rules 1 to 3 and shape nothing in `ClutchMount`.

### Constants

| Name | Value | Definition / reason |
| --- | ---: | --- |
| `ClutchTurn` | pi/3 | fan frame turn about Z from the clutch frame |
| `FlangeThickness` | 3.0 | `m2fastener.HeadPocketDepth + 1.0`: head pocket plus 1 mm under it |
| `FlangeRadius` | 28.0 | covers the ear tips (`onewayclutch.EarReach` 27.1); 1.8 mm round each head pocket inside the rim chamfer |
| `FlangeRound` | 1.5 | root round of the hub on the flange: half the flange's 3.0 |
| `ClutchBoreClearance` | 0.55 | `onewayclutch.LooseFit` |
| `ClutchSeam` | 0.25 | `onewayclutch.Seam` |
| `ClampThickness` | 4.0 | 1 + `m2fastener.NutCavityHeight` + 1 |
| `ClampInnerRadius` | 19.0 | clears the clutch's 37 mm circle (`onewayclutch.RingCounter` 18.5) by 0.5; passes the back plate (18.1) |
| `ClampOuterRadius` | 28.0 | `FlangeRadius` |
| `ClampBack` | -28.5 | `-(FlangeThickness + 2*ClutchSeam + onewayclutch.RingHeight + ClampThickness)` |
| `EarBoltLength` | 25.0 | M2 x 25, in `m2bolt.StandardLengths()`, its longest entry |
| `EarBoltCavity` | 25.5 | `-(ClampBack + 1 + m2fastener.NutCavityHeight)` |
| `MinWall` | 0.8 | least wall of a part that does not bend |
| `MountZ` | 23.0 | `onewayclutch.RingTo + ClutchSeam + FlangeThickness` |
| `ClampContainerName` | `"Clamp Ring"` | container of the clamp ring |

The stack the bolt must span is flange 3.0 + 0.25 + ring 21.0 + 0.25 + clamp 4.0; 25 mm reaches through only when the
flange is 3.0.

### Validation

See the rules above: rule 4 (Mount), rule 5 (clutch wall, `D` at most 8.7), rules 6 and 7 skipped, rule 11 (blade then
ear sites). `EarSites[k]` (k = 0..2, angle `a = EarBoltAngle(k+1)`, `u = (cos a, sin a, 0)`) is `m2fastener.Site{Head:
(EarCircle cos a, EarCircle sin a, 0), Dir: (0, 0, -1), Corner: u, Length: EarBoltLength, Cavity: EarBoltCavity}`, with
`EarCircle` = `onewayclutch.EarCircle` (23.5). `Tip()` is 27 and `CavityFar()` is 27.5. In `ClutchMount` the set screw
fields stay zero; `ClutchBoreRadius` is `r_c = D/2 + ClutchBoreClearance` (0 in `ShaftMount`).

### Geometry

1. **Hub, one revolved body.** A meridian sketch on the frame with origin 0, U = +Z and V = +X, each vertex `(r, z)`
   placed with `sketchcheck.DimensionPoint`, lines joining consecutive vertices and closing the loop. With `c` = 0.5,
   `R` = 28, `T` = 3, `rho` = 1.5, `rh` = 14, `H` = 20 and `bc` = `2 pi r_c >= edgefinish.LongEdgeRun` (true at D = 8):
   `(r_c, -T+c)`, `(r_c+c, -T)` when `bc` else `(r_c, -T)`; `(R-c, -T)`, `(R, -T+c)`, `(R, -c)`, `(R-c, 0)`;
   `(rh+rho, 0)`; eight round points `(rh+rho + rho cos a_i, rho + rho sin a_i)`, `a_i = 3 pi/2 - (pi/2)(i/8)`, ending at
   `(rh, rho)`; `(rh, H-c)`, `(rh-c, H)`; `(r_c+c, H)`, `(r_c, H-c)` when `bc` else `(r_c, H)`. That is 17 vertices, 19
   with `bc`. The chamfers are in the meridian; no `edgefinish` call runs on this body. Revolve about the Z axis.
   Cuts, in order: the three sockets; for each blade the hole tool, the pocket tool and the nut slot; for each ear the
   pocket tool (z -2 to 0) then the hole tool (through the flange).
2. **Clamp ring.** A sketch on z = `ClampBack`: circles of radius 28 and 19 about the origin, and three circles of radius
   1.3 at the ear sites; extrude 4.0 along +Z. `edgefinish.Chamfer` of both faces by 0.5 over the long loops (outer and
   inner circles). Then at each ear a nut slot (`slotTool`) of reach `ClampOuterRadius + Stock - EarCircle` (5.5): z -27.5
   to -25.5, 1.0 from each face, its corner at r 20.99 (1.99 from the bore), its mouth through the rim.
3. **Blades, bolts, nuts.** As in `ShaftMount`; the blades are the same part, and the blade nut slots open above the
   flange.
4. **Counts.** Bodies: 17. Sketches from `BuildParts`: 33 (1 meridian, 1 socket tool, 9 blade tools, 6 ear tools, 12 blade
   sketches, 1 clamp outline and 3 clamp slots); `Build` adds 30 fastener sketches. Containers: `Hub` (hub), `Blades`,
   `Clamp Ring`, `Bolts` (blade bolts 1 to 3, ear bolts 1 to 3), `Nuts` (blade nuts, ear nuts).

### Fasteners

Each M2 x 25 ear bolt's head sits flush in a pocket in the flange's front face and the bolt passes the flange, the ring
ear (21 mm) and the clamp ring. Each nut sits in a hex slot open to the clamp ring's rim: no nut is embedded, so no print
pause (AGENTS.md "Fasteners"). The tightened bolt pulls the nut onto the slot's front face.

| Check | Seams open (model) | Seams closed (tightened) |
| --- | --- | --- |
| Head top | flush with flange front face, z 0 | same |
| Material under head pocket | 1.0 (z -3 to -2) | same |
| Shank in ear bore | 0.3 radial (bore 2.6, shank 2.0) | same |
| Tip above clamp back face | 1.5 | 1.0 |
| Nut position | z -25.7 to -27.3 | pulled onto the slot's front face, -25.0 to -26.6 |
| Tip past the nut's far face | inside the nut, 1.3 of 1.6 engaged | 0.4 past the nut |
| Slot walls | 1.0 to each clamp face | same |
| Nut in slot | 0.17 at the flats (0.07 beside the rounds), 0.12 at each rounded corner, 0.2 at each face | same |

The clamp ring moves 0.5 toward the head when both seams close: 0.25 at the flange and 0.25 at the clamp ring.

### Hand and freewheel

The ring is driven when the shaft turns the `Freewheel` direction (seen from the front), and coasts that way about a
stopped shaft. A right-hand fan blows forward turning clockwise seen from the front.

| Fan `Hand` | Clutch `Freewheel` | Result |
| --- | --- | --- |
| `RightHand` | `Clockwise` | shaft clockwise drives the fan forward; the fan coasts on when the shaft stops (matched) |
| `LeftHand` | `CounterClockwise` | same, counter-clockwise (matched) |
| `RightHand` | `CounterClockwise` | shaft counter-clockwise drives the fan backward (mismatched) |
| `LeftHand` | `Clockwise` | backward (mismatched) |

`ParamsForClutch` always returns a matched pair. `Derive` cannot check a hand-built pair.

### Part fits

| Pair | Class | Modeled | Test bound |
| --- | --- | ---: | --- |
| Flange back face / ring front end | clamped seam | 0.25 | at least 0.20 |
| Clamp ring front face / ring back end | clamped seam | 0.25 | at least 0.20 |
| Flange back face / cover, cover bolt heads | Regular, after the ring's 0.85 play and the seam closing | 1.50 | at least 1.45 |
| Clamp ring / back plate, clutch nuts | Regular | 1.50 axial, plus radial separation | at least 1.45 |
| Clamp ring inner bore / clutch's 37 mm circle | Loose | 0.50 | every clamp vertex at r 18.95 or more |
| Round bore / nominal shaft | Loose | 0.55 radial | at least 0.50 |
| Ear bolt shank / ear bore, clamp hole, flange hole | fastener rule | 0.30 | at least 0.28 in 0.01 sections |
| Ear nut / slot | fastener rule | 0.17 flats (0.07 beside the rounds), 0.12 rounded corners, 0.20 faces | clear |
| Blades / flange front face | Regular | 4.17 or more (lowest blade point at 40 degrees) | every blade vertex at r 28 or less at z 0.35 or more |

The ring floats 0.85 each way between the clutch's cones, and the bolted fan floats with it.

### Retained sharp corners, edge finishing and printing

- Hub: prints back face (flange) down, axis vertical, no supports. Each slot roof (z 4.4) is a 4.35 bridge. The root round
  (1.5) is the upward-opening corner between layers.
- Clamp ring: either face down, axis vertical. Each slot roof is a 4.35 bridge.
- Retained sharp corners: the blade and ear head pocket floor rims; the blade slot faces and clamp slot faces at z -27.5
  and -25.5 (between layers); the blade bolt hole blind ends; the socket mouths; the socket floor round band, exempt at
  z 7 and 13 as today.
- Short runs left sharp: blade bolt holes, ear holes and clamp holes (8.2), head pockets (13.8) and the 605ZZ bore (19.16).
- Large faces (mm2, 605ZZ / 608ZZ): back face 2434 / 2398, flange front 1708, hub front 586 / 551, each clamp face 1313.
- Containers: see Geometry. No look-alike parts, printed shaft, arbor collars or gears.

### Mounting space, assembly and cases

- Behind the ring: r 19 to 28, 4.25 mm deep from the ring's back end (clutch z -1.25 to -5.5). In front: the flange (56
  across, clutch z 20 to 23) and the fan. The shaft may end anywhere from inside the clutch to the hub's front face.
- Assembly: (1) fit the blades; (2) assemble the clutch off the shaft; (3) hang the clamp ring round the shaft and press
  the clutch on, back plate first (the 36.2 plate passes the 38 clamp bore), tighten the clutch's bolts; (4) hold the
  clamp ring against the ring's back end with its holes on the ear bores and slide an M2 nut into each slot from the rim;
  (5) set the flange against the ring's front end and push the three M2 x 25 bolts through, tightening from the front with a
  1.5 mm hex key between the blades.
- Cases: `clutch-605ZZ` (`ParamsForClutch(onewayclutch.DefaultParams())`, `Placement` nil), `clutch-608ZZ-left`
  (`{Shaft: "608ZZ", Freewheel: CounterClockwise, BoreClearance: 0.05}`) and `assembly-605ZZ` (the fan placed on the real
  clutch; `TestTurbineFanOnClutch`).
