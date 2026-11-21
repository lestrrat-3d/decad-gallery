# Turbine fan

## Purpose and source

Construct a fan of three replaceable blades on one printed hub, sized by its rotor diameter `D_r` (`Params.RotorDiameter`,
128 to 224 mm, 192 by default; "Size" below). The hub is a plain round cylinder, radius `HubRadius` and `HubLength` long
(18.9 and 30 at 192; 14 and 20 at 128), with a D bore for a D-flat shaft. One radial M2 bolt acts as a set screw (M2 x 10
at 192, M2 x 6 at 128): it threads through a nut in a hex slot open to the hub's back face, and its tip presses the
shaft's flat. Its head floats, recessed, in a counterbore cut into the round face, so nothing stands outside the hub's
radius except the three blades. Each blade is this model's blade
(`blade.go`), with its round root stud that slides into a radial socket in the hub and one M2
bolt that runs parallel to the shaft through the hub and the stud into a nut. That nut sits in a hex slot open to the round
face, below the blade, so no nut is embedded and the hub needs no print pause. The same hub also has two blind M2 bolt
holes in its back face, each with a hex nut slot open to the round face: they take the two M2 x 10 bolts that hold it to
the standard adapter ring of a one-way clutch (see "Clutch mount"). There is one hub for both mounts; it has no flange and
no clamp ring. The parts are:

- a hub, a cylinder cut by the D bore, three sockets, the blade bolts' pockets and nut slots, the set screw's
  counterbore, hole and nut slot, and the two adapter bolts' holes and nut slots;
- three blades, one part printed three times;
- four M2 bolts (three blade bolts, M2 x 25 at 192, and the set screw, M2 x 10) and four M2 nuts in `ShaftMount`; five M2
  bolts (the blade bolts and two M2 x 10 adapter bolts) and five M2 nuts in `ClutchMount`, which has no set screw.

The shaft is not modeled. It belongs to the mechanism, and its diameter equals the bore of the purchased bearing it runs
in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets read that bore from
[`models/ballbearing`](../ballbearing/). The bolts and nuts are the M2 preset of
[`models/fastener`](../fastener/), placed with `internal/m2fastener`.

All dimensions are in millimeters; angles are radians in code and degrees in prose. The model owns the blade and its
joint: it exports `SizeFor`, `BuildBlade`, `BuildBladeMirrored` and the blade socket tool, and
[`models/handcrankfan`](../handcrankfan/) imports them with `Params608ZZ` at `RotorDiameter` 128. It reads `PressWall`, `ChamferSize`, `RegularFit` and the shaft flats from `crankhandle`.

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
| `s`, `j` | blade scale and hub scale | `s = D_r / 128`, `j = HubRadius / 14` |
| `L_s` | the set screw's length, `Dimensions.SetScrewBolt` | the longest `m2fastener.BoltSize.StandardLengths` entry with `rho_t + HeadPocketDepth + L_s <= sqrt(HubRadius^2 - (HeadDiameter/2)^2)` |
| `rho_h` | the set screw's head top | `rho_t + HeadPocketDepth + L_s` |
| `rho_f` | the counterbore's floor | `rho_h - HeadPocketDepth - SetScrewTravel` |
| `u_k`, `v_k` | blade k's radial direction and its counter-clockwise tangent | `(cos t_k, sin t_k, 0)`, `(-sin t_k, cos t_k, 0)`, `t_k = 90 + 120 (k - 1)` degrees |

| `PrintTilt` | the blade's radial line above the bed in the print pose: its rest tilt | `RestTilt`, 11.4, 11.8 and 12.5 degrees at 128, 192 and 224 |
| `PrintLean` | the chord at `RefRadius` turned from vertical toward the pressure face in the print pose | -15 degrees |
| `PrintLift` | the z of the blade's stud end corner in the print pose: on the bed | 0 |

`rho` is always the distance from the shaft axis along -Y.

## API

Package: `models/turbinefan`. Exported surface:

- The fixed dimensions as constants: `BladeCount` (3), `FirstBladeAngle` (90 degrees), `SetScrewZ` (4), `SetScrewTravel`
  (0.4), `CounterboreRound` (0.5), `NutSlotRound` (0.5), `BladeSocketFloorRound` (0.5), and the values it takes from
  `crankhandle`: `ChamferSize`, `PressWall` and `RegularFit`. `BladeAngle(k)` is blade k's angle about +Z.
- The size constants `BaseRotorDiameter` (128), `DefaultRotorDiameter` (192), `MinRotorDiameter` (128),
  `MaxRotorDiameter` (224), `BaseHubRadius` (14), `MaxHubRadius` (18.9) and `BladeBoltTipFloor` (2), and `SizeFor`,
  `Size` and `Size.Beta` ("Size" below).
- The container names `HubContainerName` (`Hub`), `BladesContainerName` (`Blades`), `BoltsContainerName` (`Bolts`),
  `NutsContainerName` (`Nuts`).
- The clutch-mount constants `ClutchTurn`, `ClutchSeam`, `MountZ` and `ClutchAnchorZ` (see "Clutch mount").
- `Hand` with `RightHand` (the zero value) and `LeftHand`; `Hand.Freewheel()` returns the `onewayclutch.Direction` that
  drives a fan of that hand forward.
- `Mount` with `ShaftMount` (the zero value) and `ClutchMount`; `EarBoltAngle(k)` is ear bolt k's angle in the fan frame.
- `Params` with `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `RotorDiameter`, `BladePitch`, `Hand`, `Mount` and `Placement *r3.Frame`
  (nil leaves every part in the local frame; a frame puts the placement point on it, see "Local frame and hand").
- `MountFrame(clutch *r3.Frame) (r3.Frame, error)` and `ParamsForClutch(onewayclutch.Params, ...ClutchOption) (Params, error)`.
  `WithBladePitch(pitch)` and `WithRotorDiameter(d)` are the `ClutchOption`s; each replaces its default and the result is
  validated.
- `Params605ZZ()`, `Params608ZZ()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`. `Dimensions` holds `BoreRadius`, `BoreFlat`,
  `BoreFlatWidth`, `BoreLoop`, `ShaftFlatWidth`, `SetScrewTip`, `SetScrewHead`, `CounterboreFloor`, `SetScrewBolt`,
  `SetScrewSite`, `BladeSites`, `AdapterSites` and `Size`, in the local frame. `SetScrewSite` and `AdapterSites` are filled in both modes.
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.1 mm), opts...)`. The blade's stud
  union refuses a plain document; build in such a document.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the hub and the three blades and puts them in no container. `Parts`
  holds `Hub`, `Blades`, every `Sketches`, `Dimensions`, and in the world frame `BladeSites`, `SetScrewSite` and
  `AdapterSites`.
- `Build(ctx, doc, p) (*Model, error)` adds the parts, the bolts and the nuts, each in its container. `Model` embeds `Parts`
  and holds `BladeBolts`, `BladeNuts`, `SetScrew` (`*fastener.Bolt`) and `SetScrewNut` (`*fastener.Nut`), and in
  `ClutchMount` `AdapterBolts` and `AdapterNuts` (the set screw and its nut are nil). `Build` builds neither the adapter nor
  the clutch: the caller builds them in the clutch's own document (see "Clutch mount").

The blade exports other models use: `BuildBlade(ctx, doc, pitch, size)` returns the unplaced blade and its twelve sketches,
`BuildBladeMirrored` the blade of the other hand (its sections not flipped and its pitch turn reversed, so it is the first
blade reflected in the plane x = 0 of its local frame), `BladeSocketTool(ctx, doc, f, size)` builds a socket cutter, each
with a `Size` from `SizeFor` (any other value is rejected); and `BladeSocketFloorRound`, `DefaultBladePitch`,
`MinBladePitch` and `MaxBladePitch`. The sizes the blade and the joint need (`HubRadius`, `HubLength`, `BladeBoltAt`,
`BladeBoltLength`, `BladeBoltCavity`, `BladeStudRadius`, `BladeStudStart`, `BladeStudEnd`, `BladeSocketRadius`,
`BladeSocketStart`, the tip and root radii) are the fields of `Size`, and `Size.Beta(r, pitch)` is the blade angle. From
`crankhandle`: `PressWall`, `ChamferSize`, `RegularFit`, `Params605ZZ` and `Params6082RS`.

The blade's print aid ("Print pose and cradle" below), in `print.go` and `harness.go`:

- The pose constants `PrintLean` (-15 degrees), `PrintLift` (0), `RestTiltMin` (5 degrees) and `RestTiltMax` (20 degrees),
  and the cradle constants `CradleGapZ` (0.2), `CradleGapXY` (0.4), `CradleWall` (2.0), `CradleTopDrop` (0.3),
  `CradleTopBand` (0.1), `CradleTopSmooth` (1.0), `CradleTopPasses` (8), `CradleStep` (0.5), `CradleFitPoints` (33),
  `CradleYStep` (0.05), `CradleTipMargin` (1.0), `CradlePlanWindow` (1.0), `StudSlotFlats` (16) and `SectionStep` (0.25),
  and the container name `HarnessContainerName` (`Harness`).
- `RestTilt(ctx, blade, size, pitch, hand) (float64, error)` returns the tilt at which a blade from `BuildBlade` or
  `BuildBladeMirrored` rests on both its stud end corner and its leading edge. `PrintPoseFor(size, pitch, hand, tilt)
  (r3.Frame, error)` returns the frame that moves that blade into the print frame.
- `CradleSection` (`X`, `Outline`, `LeftY`, `LeftZ`, `RightY`, `RightZ`, `LeftLimit`, `RightLimit`) and `BladeHarness`
  (`Body`, `Sketches`, `Sections`, `XStart`, `XEnd`, `StepZone`, `WallTop`).
- `BuildBladeHarness(ctx, doc, blade, size, hand, tilt) (*BladeHarness, error)` builds the cradle for a posed blade and
  puts it in no container.
- `BladePrint` (`Blade`, `BladeSketches`, `Harness`, `Pose`, `Tilt`) and `BuildBladePrint(ctx, doc, p) (*BladePrint,
  error)`, which validates `p` (`Params.Derive`), builds the blade of `p.Hand` at `p.BladePitch`, finds its `RestTilt`,
  poses it, builds its cradle and files them in `Blades` and `Harness`. It ignores `Mount` and `Placement`. A rejected
  build adds no body; a later geometry failure may leave bodies in the document, which the caller discards.

## Size

`SizeFor(D_r)` returns the `Size` of a rotor of diameter `D_r`, or `turbinefan: RotorDiameter must lie in [128, 224] mm`
for a value that is not finite or lies outside that range. The blade scale is `s = D_r / 128`. The hub radius is
`HubRadius = min(14 s, MaxHubRadius)` and the hub scale is `j = HubRadius / 14` (compute it this way, so the cap is
exact). Every airfoil section, the hub length, the pitch reference radius and the hold radius scale by `s`; the hub
radius, the round blade root, the stud, the socket and the blade bolt circle scale by `j`.

| `Size` field | Formula | 128 | 192 (default) | 224 |
| --- | --- | ---: | ---: | ---: |
| `Scale` s, `HubScale` j | | 1, 1 | 1.5, 1.35 | 1.75, 1.35 |
| `TipRadius` | `D_r / 2` | 64 | 96 | 112 |
| `HubRadius` | `min(14 s, 18.9)` | 14 | 18.9 | 18.9 |
| `HubLength`, `HubMid` | `20 s`, `HubLength / 2` | 20, 10 | 30, 15 | 35, 17.5 |
| `RootRadius` | `HubRadius + RegularFit` | 14.4 | 19.3 | 19.3 |
| `RefRadius`, `HoldRadius` | `48 s`, `21.5 s` | 48, 21.5 | 72, 32.25 | 84, 37.625 |
| `BladeSocketStart` | `5.7 j` | 5.7 | 7.695 | 7.695 |
| `BladeStudStart` | `BladeSocketStart + RegularFit` | 6.1 | 8.095 | 8.095 |
| `BladeStudRadius` | `3 j` | 3 | 4.05 | 4.05 |
| `BladeSocketRadius` | `BladeStudRadius + RegularFit` | 3.4 | 4.45 | 4.45 |
| `BladeStudEnd` | `16 j` | 16 | 21.6 | 21.6 |
| `BladeBoltAt` | `9.5 j` | 9.5 | 12.825 | 12.825 |
| `BladeBoltLength` | the longest `m2fastener.BoltSize.StandardLengths` entry `L` with `HubLength - HeadPocketDepth - L >= BladeBoltTipFloor` (2) | 16 | 25 | 25 |
| `BladeBoltCavity` | `BladeBoltLength - 0.4` | 15.6 | 24.6 | 24.6 |

`SizeFor` also fails when no standard length fits, or when the blade nut slot's near face `HubLength - BladeBoltCavity`
lies above `HubMid - BladeSocketRadius - 1` (under 1 mm of hub between the slot and the socket). The values that do not
scale are the M2 fastener sizes, every fit clearance (0.05, 0.2, 0.35, 0.4, 0.5), `PressWall` 1.6, `ChamferSize` 0.5,
every corner round (0.5) and the stud's 0.3 chamfer, the trailing edge's 1.0, `SetScrewZ`, `SetScrewTravel`, `sidePoints`
and every thickness ratio, camber and ellipse weight of the station table.

`MaxHubRadius` is 18.9 because of the one-way clutch mount. The smallest clutch (605ZZ, `EarCircle` 21.6) carries its ear
bolts on a circle of 21.6. Their heads are driven from the front past the hub's rim into pockets of radius
`HeadPocketRadius` (2.2) in the adapter, and a 1.5 mm hex key must pass too. The cap is `21.6 - 2.2 - 0.5`, which leaves
the same 0.5 margin the blade test uses; a hub of radius 21 (14 times 1.5) would cover the ear bolts. A test requires
`MaxHubRadius` to equal that expression. One hub serves both mounts.

Station table, with `r` and `c` in mm (the first two are the round root, scaled by `j`; the rest the airfoil, scaled by
`s`; thickness ratio, camber and ellipse weight are as "Blade shape" gives):

| Station | `r` | chord `c` |
| --- | --- | --- |
| 1 | `RootRadius` (14.4 j) | `8 j` |
| 2 | `RootRadius + 1.9 j` | `8 j` |
| 3 to 10 | `s` x (19, 21.5, 25, 30, 38, 46, 55, 64) | `s` x (11.5, 15.5, 17.8, 16.5, 14.5, 12.5, 10.5, 8.0) |

The blade angle is `beta(r) = alphaD + atan(tan(p0 - alphaD) RefRadius / max(r, HoldRadius)) + (BladePitch - p0)`, so
an airfoil section at `s r` has the angle that 128's section at `r` has: the airfoil is 128's blade enlarged `s` times
about its stacking line. Only the round root (by `j`) and the 1.0 trailing edge differ.

The set screw's length `L_s` is picked by the rule in "Symbols" (M2 x 6 at 128, M2 x 10 at 192 and 224, both bores), and
`SetScrewSite`, the counterbore and every use of the length read `Dimensions.SetScrewBolt`. A rotor with no fitting
length fails with `turbinefan: set screw: no standard length keeps its head inside the round face`.

## Presets and defaults

| Field | `Params605ZZ` (default) | `Params608ZZ` |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params608ZZ().BoreDiameter` |
| `FlatDepth` | 0.5, the planetary gearbox shafts' flat | 0.75, the siren shafts' flat |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `RotorDiameter` | 192, `DefaultRotorDiameter` (valid 128 to 224) | 192 |
| `BladePitch` | 25 degrees, `DefaultBladePitch`, the blade angle at `RefRadius` | 25 degrees |
| `Hand` | `RightHand` | `RightHand` |

| Derived | 605ZZ | 608ZZ | `regular-bore` (605ZZ, c 0.35) |
| --- | --- | --- | --- |
| `r_b` | 2.55 | 4.05 | 2.85 |
| `f_b` | 2.05 | 3.30 | 2.35 |
| `w_b` | 3.0332 | 4.6957 | 3.2249 |
| `L_b` | 15.81 | 25.12 | 17.70 |
| `w_s` | 3.0 | 4.6637 | 3.0 |
| `rho_t` | 1.95 | 3.20 | 1.65 |
| `rho_h` at 128 (M2 x 6) | 9.95 | 11.20 | 9.65 |
| `rho_f` at 128 | 7.55 | 8.80 | 7.25 |
| `rho_h` at 192 and 224 (M2 x 10) | 13.95 | 15.20 | 13.65 |
| `rho_f` at 192 and 224 | 11.55 | 12.80 | 11.25 |

A 608-2RS has the same bore as a 608ZZ. `rho_t` is the flat of a shaft pushed `c` toward +Y until its round meets the bore's
round, which is where the tightened screw leaves it. The tip then stands `2c` (0.1 at the presets) inside the bore's flat
(exception X2).

## Validation

Every message starts with `turbinefan: `. Derive checks in this order and returns the first failure:

1. `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `BladePitch` and `RotorDiameter` are finite and positive: `<Field> must be
   finite and positive`.
2. `RotorDiameter` lies in [128, 224] (through `SizeFor`, whose other errors surface here with the `turbinefan: `
   prefix): `RotorDiameter must lie in [128, 224] mm`.
3. `d <= D/4`.
4. `ballbearing.PressFitClearance <= c <= 0.5`.
5. `Mount` is `ShaftMount` or `ClutchMount`, else `Mount must be ShaftMount or ClutchMount`.
6. `BladeSocketStart - r_b >= PressWall`, so 1.6 mm stays between the bore and the socket floors: `r_b` at most 4.1 at 128
   (`BladeSocketStart` 5.7) and 6.095 at 192 and 224 (7.695). Both mounts take this rule: one hub serves both.
7. `w_s >= m2fastener.BoltSize.ShankDiameter`, else `FlatDepth must leave the shaft's flat at least 2 mm wide for the
   set screw's tip`.
8. `w_b >= 2 (HoleRadius + 0.1)`, else `FlatDepth must leave the bore's flat at least 2.8 mm wide for the
   set screw's hole`. The 0.1 keeps the hole's cylinder off the bore's round.
9. `BladePitch` lies in [10 degrees, 40 degrees].
10. `Hand` is `RightHand` or `LeftHand`.
11. `Placement` is nil or a valid frame.
12. The site checks, the same in both mounts, in this order: the set screw's length `L_s` exists (`set screw: no standard
    length keeps its head inside the round face`); `SetScrewSite.Validate()` (`set screw: ...`); `Cavity -
    HeadPocketDepth - SetScrewTravel >= 1`, else `set screw: less than 1 mm between its counterbore and its nut slot`;
    `SetScrewZ - HeadPocketRadius >= 1`, else `set screw: less than 1 mm between its counterbore and the back face`; each
    `AdapterSites[k].Validate()` (`adapter bolt %d: ...`); then each `BladeSites[k].Validate()`. The head's limit
    `rho_h <= sqrt(HubRadius^2 - (HeadDiameter/2)^2)` is 13.8705 at 128 and 18.8 at 192; the rule that picks `L_s` keeps it.

`FlatDepth` 0.2 on the 605ZZ fails rule 7. 0.25 passes rule 7 and fails rule 8, in either mount.

BuildParts and Build reject a nil context and a nil document, derive the dimensions, and check `ctx.Err()` before any
geometry. A rejected build adds no body.

## Local frame and hand

The origin is on the shaft axis in the hub's back face, the face toward the mechanism. +Z runs along the shaft away from
the mechanism to the front face at z = `HubLength`. The bore's flat faces -Y, at y = -`f_b`, and the set screw enters from -Y along
the line x = 0, z = `SetScrewZ` = 4, pointing +Y. Blade k lies along `u_k`: blade 1 along +Y, blade 2 at 210 degrees and
blade 3 at 330 degrees, counter-clockwise seen from the front. The set screw lies at 270 degrees, midway between blades 2
and 3. A `Placement` maps this frame onto the given one after each part's last operation, so that the placement point lands on
the given frame's origin and the local axes land on its U, V and normal. The placement point is on the shaft axis. In
`ShaftMount` it is the hub's front face (local z = `HubLength`, the face the blades face), and the hub extends `HubLength` back
along the given frame's axis. In `ClutchMount` it is the clutch cover's outer face (local z = `ClutchAnchorZ` = -7.75), with
the clutch behind it and the adapter and fan in front. A nil `Placement` moves nothing: the parts stay in the local frame
above, with its origin in the hub's back face. Derived sites are in the local frame in every case.

Hand: a right-hand rotor blows air forward, along +Z and away from the mechanism, when the shaft turns clockwise seen from
the front. A left-hand rotor does the same when the shaft turns counter-clockwise: its blades are the right-hand blades
reflected in the plane x = 0 (`BuildBladeMirrored`), which maps blade 1 to itself and swaps blades 2 and 3.
The hub, bolts and nuts do not depend on the hand. For the right hand, a blade's chord from its trailing edge to its
leading edge at radius r runs along `(cos beta, 0, -sin beta)` with `beta = Size.Beta(r, BladePitch)`: the leading
edge leads toward +X at the top and sits upstream (-Z), and the concave pressure face faces the front.

## Geometry

1. **Hub outline.** A sketch on z = 0: the circle of radius `HubRadius` about the axis; the D bore, an arc of radius `r_b` about the
   axis from `(w_b/2, -f_b)` counter-clockwise over +Y to `(-w_b/2, -f_b)`, closed by its flat, a horizontal line at distance
   `f_b` from the origin; and three circles of radius 1.3 at `BladeBoltAt` (9.5 j) along `u_k`, the blade bolts' holes.
   The profile has four holes: the three bolt holes and the bore. The bore runs through.
2. **Hub body.** Extrude `HubLength` along +Z, then `edgefinish.Chamfer` of both faces by `ChamferSize`, taking the long loops: the
   outer circle on both faces and, when `L_b` is 20 or more (608ZZ), the D loop. Then cut, in this order:
   - the three sockets: `BladeSocketTool` on the frame with origin `(0, BladeSocketStart, HubMid)`, U = +Z and V = +X
     (normal +Y), and its two copies from `decad.PatternCopies` with a circular pattern of three about +Z;
   - for each blade k, `m2fastener.PocketTool` (head pocket, z `HubLength - 2` to `HubLength`, radius 2.2) then the blade
     nut slot (below);
   - the set screw's counterbore (below), `m2fastener.HoleTool(SetScrewSite)` (radius 1.3, from `rho_h + 1` to `rho_t - 1`, its
     inner end inside the bore) and the set screw's nut slot (below);
   - for each adapter bolt k = 1, 2 (angles 30 and 150 degrees), `m2fastener.HoleTool(AdapterSites[k])` (radius 1.3, blind,
     from the back face to z 6.75) then the adapter nut slot (below).
3. **Nut slot.** A hexagon about the bolt line with its corners on the circle of radius 2.51, one corner at the slot's end,
   whose two sides toward the opening run on as straight walls 2.1737 from the line, out through the part's face; 2.0 deep.
   The slot's size does not scale.
   - Set screw slot: plane `y = -(f_b + 1.6)`, extruded to `y = -(f_b + 3.6)`, U = -Z and V = +X, reaching `z = 6.51`
     at its end corner and open at z = -1 (below the back face). Every edge of both slot faces is rounded by `NutSlotRound`.
   - Adapter slot k: `slotTool` with a `Dir` +Z site puts the slot plane at `Head.Z + Cavity`, z = 1.5, U = `u_k` (the
     radial line at 30 or 150 degrees), V = `v_k`, the corner at r 6.99, open through the rim; z 1.5 to 3.5. Its three hexagon
     corners are rounded by `NutSlotRound`, as the blade slots' are.
   - Blade slot k: plane z = `HubLength - BladeBoltCavity - 2` (2.4 at 128, 3.4 at 192, 8.4 at 224), U = `u_k`, V = `v_k`, the
     corner at r `BladeBoltAt - 2.51` (6.99 at 128, 10.315 at 192) on the radial line, open through the rim; 2.0 high.
     Its three hexagon corners, the vertical edges along Z where the end meets the walls, are rounded by `NutSlotRound`:
     each turns 60 degrees, so a layer would read it as an inside corner.
4. **Counterbore.** A revolved meridian about the screw's line: from the floor at `rho_f` a quarter-round of radius
   `CounterboreRound` (eight chords) up to radius 2.2, out through the round face. The head (radius 1.9) floats
   `SetScrewTravel` above the floor, where the round's surface is at radius 2.19, so it clears it by 0.29.
5. **Blades.** `BuildBlade(ctx, doc, BladePitch, Size)` (`BuildBladeMirrored` for `LeftHand`) returns blade 1 in its own
   frame (origin on the fan axis in the hub's mid plane, +Z its radial line, +Y downstream, the blade moving toward -X).
   `decad.Transform` by the frame with origin `(0, 0, HubMid)`, U = (-1, 0, 0) and V = (0, 0, 1) turns it into place: blade-local
   Z becomes +Y, local Y becomes +Z and local X becomes -X. `decad.PatternCopies` with the hub's pattern gives blades 2 and
   3. Do not transform blade 1 again before copying.
6. **Blade facts.** A loft from `RootRadius` to `TipRadius` through ten stations; a stud of radius `BladeStudRadius`
   from r `BladeStudStart` to `BladeStudEnd` with a 0.3 chamfer on its inner end; a bolt hole of radius 1.3 along the fan
   axis at r `BladeBoltAt` through the stud, cut by a cylinder from `-(BladeStudRadius + 2)` of length `2 (BladeStudRadius +
   2)`; sections that are NACA 4-digit mean lines blended with an ellipse at the root; a blade angle `Size.Beta(r,
   BladePitch)` that is constant inboard of `HoldRadius`; a trailing edge 1.0 thick. The rotor is `D_r` across. Every
   blade vertex lies between z 0 and `HubLength`. Measured z ranges (`TestTurbineFanEnvelope`): 128 mm, 5.02 to 17.72 at
   the default pitch; at 192 mm 7.51 to 26.39 at the default pitch, 6.24 to 28.94 at 40 degrees and 8.76 to 23.10 at 10
   degrees; at 224 mm and 40 degrees 7.27 to 33.71.
7. **Containers.** `Build` files the hub, the three blades, the bolts and the nuts in their containers, each a child of the
   document root: four bolts and four nuts in `ShaftMount`, five of each in `ClutchMount`.

## Layout numbers

Numbers that depend on the size are given at 128 and at 192 (224 differs from 192 only in the hub's length, 35, and what
hangs on it). The clearances between slots, holes and the counterbore at 192 are the in-plane distances of their outlines,
computed with the same method that reproduces the 128 values; `TestTurbineFanBladeBolts`, `SetScrew` and `AdapterCuts` check
the walls and fits that the rules require.

| Item | 605ZZ | 608ZZ | `regular-bore` |
| --- | --- | --- | --- |
| Hub envelope | x, y [-14, 14], z [0, 20] at 128; [-18.9, 18.9], [0, 30] at 192 | same | same |
| Bore flat at y | -2.05 | -3.30 | -2.35 |
| Set screw: tip rho, head top rho | 1.95, 9.95 at 128; 13.95 at 192 | 3.20, 11.20; 15.20 | 1.65, 9.65; 13.65 |
| Head top recessed below the round face at x = 0 | 4.05 at 128; 4.95 at 192 | 2.80; 3.70 | 4.35; 5.25 |
| Counterbore r 2.2, floor rim round 0.5: floor rho; z | 7.55; 1.8 to 6.2 at 128; 11.55 at 192 | 8.80; 12.80 | 7.25; 11.25 |
| Set screw hole r 1.3: rho | 2.05 to 7.55 at 128; to 11.55 at 192 | 3.30 to 8.80; to 12.80 | 2.35 to 7.25; to 11.25 |
| Set screw nut slot: rho; x; z | 3.65 to 5.65; +-2.17; 0 to 6.51 | 4.90 to 6.90 | 3.95 to 5.95 |
| Set screw nut: rho; x; z | 3.85 to 5.45; +-2.0; 1.69 to 6.31 | 5.10 to 6.70 | 4.15 to 5.75 |
| Wall, bore's flat to the slot's inner face | 1.60 | 1.60 | 1.60 |
| Wall, bore's corner to the slot's nearest edge | 1.73 | 1.60 (the flat covers the slot) | 1.70 |
| Material, slot's outer face to counterbore floor | 1.90 at 128; 5.90 at 192 | 1.90; 5.90 | 1.30; 5.30 |
| Material, counterbore to back face | 1.8 | 1.8 | 1.8 |
| Blade nut slot: from the corner at r `BladeBoltAt - 2.51` out through the rim; tangential; z | 6.99; +-2.17; 2.4 to 4.4 at 128; 10.315; 3.4 to 5.4 at 192 | same | same |
| Blade nut: along `u_k` r `BladeBoltAt - 2.31` to `+ 2.31`; z | 7.19 to 11.81; 2.6 to 4.2 at 128; 10.515 to 15.135; 3.6 to 5.2 at 192 | same | same |
| Adapter bolt hole r 1.3 and nut slot (30 and 150 degrees): hole z; slot z, corner r | -6.25 to 6.75, blind; 1.5 to 3.5, 6.99 | same | same |
| Hub under the adapter slot (to the back face); slot corner to the bore | 1.5; 4.44 | 1.5; 2.89 | 1.5; 4.14 |
| Adapter slot to blade nut slots, to sockets | 3.9 or more, 3.5 or more at 128; 6.78 and 4 or more at 192 (the sockets lie above z 10.55, the adapter slot below z 3.5) | same | same |
| Blade nut slot to socket (socket bottom minus slot top), back face, bore | 2.2, 2.4, 4.44 at 128; 5.15, 3.4, 4.44 at 192 | 2.2, 2.4, 2.94; 5.15, 3.4, 2.94 | 2.2, 2.4, 4.14; 5.15, 3.4, 4.14 |
| Set screw slot to blade nut slots, to blade bolt holes | 3.88, 4.76 at 128; 6.76, 7.67 at 192 (7.63 at 608ZZ) | same | same |
| Counterbore to blade nut slots | 4.15 at 128; 7.77 at 192 | 4.76; 8.47 | 4.05; 7.63 |
| Set screw features to sockets (lower bound) | 1.90 at 128; 4 or more at 192 (the slot's end at z 6.51 to the sockets' bottom at z 10.55) | 2.13; same | 1.94; same |
| Blade nut slots to each other | 12.1 at 128; 17.87 at 192 | same | same |
| Wall, bore to socket floors (r `BladeSocketStart`) | 3.15 at 128; 5.145 at 192 | 1.65; 3.645 | 2.85; 4.845 |
| Bore flat width / shaft flat width | 3.03 / 3.00 | 4.70 / 4.66 | 3.22 / 3.00 |

At 608ZZ the bore's 0.5 lead-in chamfer leaves 1.1 between the bore and the set screw slot over z 0 to 0.5. That band is
the lead-in, not pressed length.

## Fasteners

No nut is embedded: the set screw's nut and the blade nuts sit in hex slots open to a face (AGENTS.md "Fasteners"). Both
slots use the hexagon of the Fasteners rule (corners on the 2.51 circle, 2.0 deep). The slot's sides stand 2.17 from the
bolt line, less than the nut's 2.31 corner radius, so they stop the nut turning. The tightened bolt holds the nut: the set
screw pulls its nut toward -Y onto the slot's outer face, and a blade bolt pulls its nut toward +Z onto the slot's face at
the slot's face at `HubLength - BladeBoltCavity` (4.4 at 128, 5.4 at 192). The set screw's nut is wholly inside the hub
(z 1.69 to 6.31); the blade nut's outer corner is at r `BladeBoltAt + 2.31` (11.81 at 128, 15.135 at 192, inside the rim
at 14 and 18.9 by 2.19 and 3.77); an adapter nut (z 1.7 to 3.3) reaches r 12.43 at most. The 0.5 rounds leave 0.07 of
clearance at the nut's flats and 0.12 at each rounded corner of the blade slot (a 120 degree corner rounded by 0.5 takes
0.077 of the 0.2 past the nut's corner).

| Bolt | Head | Hole | Nut | Rule |
| --- | --- | --- | --- | --- |
| Blade bolt k (`BladeBoltLength`: M2 x 16 at 128, M2 x 25 at 192) | flush in the front face, pocket radius 2.2, 2 deep | radius 1.3 through the hub and the stud | hex slot open to the round face, entered radially, z 2.4 to 4.4 at 128, 3.4 to 5.4 at 192 | open hex seat |
| Set screw (`SetScrewBolt`: M2 x 6 at 128, M2 x 10 at 192) | recessed in a counterbore radius 2.2, 0.4 above its rounded floor (X1) | radius 1.3 from the counterbore floor through the press-fit wall (X3) into the bore | hex slot open to the back face, entered along +Z | open hex seat |
| Adapter bolt k (M2 x 10, `ClutchMount` only) | flush in the adapter's rear face (z -6.25), in the adapter's pocket, not in the hub | radius 1.3 from the hub's back face to z 6.75, blind; the bolt's tip (z 5.75) stays inside | hex slot open to the round face, entered radially, z 1.5 to 3.5 | open hex seat |

`Build` places the set screw with `m2fastener.PlaceBolt` at `SetScrewSite` and its nut with `PlaceNut`. `SetScrewSite` is
`m2fastener.Site{Head: (0, -rho_h, SetScrewZ), Dir: (0, 1, 0), Corner: (0, 0, 1), Length: L_s, Cavity:
rho_h - f_b - PressWall - NutCavityHeight}`; `Cavity` is 4.3 at 128 (the presets) and 8.3 at 192, so the slot spans `f_b + 1.6`
to `f_b + 3.6` and its inner face keeps exactly `PressWall` to the bore's flat. `Corner` +Z puts the nut's corners up and
down and its flats on +-X. `Tip()` is `L_s + 2`, so the tip lies at `rho_t`. The blade bolt sites are turned about the
shaft axis for blades 2 and 3: head at `BladeBoltAt` from the axis along the blade's radial line at z `HubLength`, `Dir`
-Z, `Corner` the radial line, `Length` `BladeBoltLength` and `Cavity` `BladeBoltCavity`; the blade nut slot takes the place of
the round cavity. The bolt's tip lies at `HubLength - 2 - BladeBoltLength`: z 2.0 at 128, 3.0 at 192 and 8.0 at 224, at or
above `BladeBoltTipFloor`, so the slot keeps 2.4 of hub under it.

Why a slot open to the back face does not work for the blade nuts: a blade bolt runs along -Z, so a seat open to the back
face is a pocket coaxial with it at z 0 to 2. The bolt's tip at z 2.0 (128) does not reach a nut at z 0.2 to 1.8, and the back
face is against the mechanism, so with the hub on its shaft the nut would fall out behind the hub whenever a blade bolt is
undone and could not be put back. Replacing a blade with the hub on its shaft is a requirement. A slot open to the round
face, along the blade's radial line, works: its mouth is on the rim at z 2.4 to 4.4 (128), below the blade root. At 128 the
lowest blade point over the nut's last 4.6 mm of travel (r 13 to 19) is z 5.94 / 5.55 at 25 degrees and 5.98 / 5.22 at 40
degrees, against the nut's top at 4.2. Beyond r 19 the steep blade comes down to z 4.20, so tweezers bring the nut in from
below, tilted. At 192 the blade root is higher above the slot (lowest blade point 6.26 at 40 degrees against the nut's top
at 5.2). A tangential slot was rejected: avoiding the set screw forces two slots to converge, leaving 0.79 between them.

Set screw notes: the tip's face (radius 0.75 inside its 0.25 chamfer; shank radius 1.0) lies within the shaft's flat (half
width 1.5 at 605ZZ, 2.33 at 608ZZ); validation rule 7 keeps that flat 2 mm wide or more. Nothing stands proud: the head stays
2.8 or more inside the round face (2.80 at 128, 3.70 or more at 192). The counterbore leaves 0.3 radial round the head and 0.4 under it, and 1.0 or more of
material under the counterbore and round every pocket.

Retention and strength, estimates only. The torque goes through the D flat by shape; the set screw removes the 0.05 play and
holds the hub along the shaft by friction at the tip and at the bore's far side. For a hand-tightened M2 (preload 100 to
300 N, friction 0.2) the axial hold is about 40 to 120 N and the friction torque about 0.09 to 0.27 N m; a hand-turned
128 mm fan's air load is estimated well under 0.05 N m and its thrust under 1 N. The air torque goes with about the fifth
power of the diameter at the same speed and the thrust with the fourth, so the 224 mm fan's loads are up to about 7.6 and 5
times those: torque up to about 0.4 N m and thrust under 5 N, at a hand-turned speed. The D flat carries the torque by
shape, and the set screw's 40 to 120 N axial hold exceeds the thrust. The nut's seat is backed by 1.9 of hub at 128 (5.9
at 192; 1.3 and 5.3 at `regular-bore`) before the counterbore; to fail, the nut must punch a plug about 2.1 in radius
through it, a shear area of about 25 mm2 at 128 (17 at `regular-bore`), or 375 to 500 N (255 to 340 N) at an assumed 15 to
20 MPa across layers, and far more at 192.
What only a printed test settles: the preload the web takes before it cracks or creeps, the friction values, and how much
creep loosens the screw.

Balance: the net imbalance on the -Y side at 128 is about 1.1 g mm (605ZZ) and 1.8 g mm (608ZZ) at full infill, up to about 3.4
g mm at sparse infill (not re-estimated for the larger hub, whose set screw is longer and whose counterbore is deeper).
The three blade slots are identical turned copies and cancel. No balance mass is added.

## Blade shape

The blade is a smooth loft through ten sections, stacked on its radial line through the hub's mid plane and starting at
r `RootRadius`, 0.4 outside the hub rim. A root stud of radius `BladeStudRadius`, from r `BladeStudStart` to r
`BladeStudEnd`, is joined to it, with a bolt hole of radius 1.3 parallel to the shaft at r `BladeBoltAt`. The stud's inner
end has a 0.3 chamfer, so its corner stays clear of the socket floor's round. The stud engages the socket from r
`BladeStudStart` to r `HubRadius` with a 0.4 clearance (a Regular fit). The shape follows a wind turbine blade: a round
root carries the joint, the section turns into a thick oval and an airfoil at the maximum chord (r 25 s), and the chord
tapers to the tip. The table is written at 128 (`s` = `j` = 1); "Size" gives the scaled radii and chords.

| r | 14.4 | 16.3 | 19 | 21.5 | 25 | 30 | 38 | 46 | 55 | 64 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| chord c | 8.0 | 8.0 | 11.5 | 15.5 | 17.8 | 16.5 | 14.5 | 12.5 | 10.5 | 8.0 |
| thickness ratio t/c | 1.00 | 1.00 | 0.60 | 0.38 | 0.26 | 0.22 | 0.19 | 0.18 | 0.17 | 0.22 |
| camber m (of c) | 0 | 0 | 0.010 | 0.025 | 0.035 | 0.040 | 0.040 | 0.040 | 0.035 | 0.030 |
| ellipse weight w | 1 | 1 | 0.60 | 0.25 | 0 | 0 | 0 | 0 | 0 | 0 |

Each section, at chord fraction x from the leading edge, has the NACA 4-digit mean line `yc(x)` with camber `m` at
`p = 0.40`. Its half thickness is `(1 - w) naca(x) + w ell(x)`, where
`naca(x) = (t/2 - 0.15 tTE) / 0.10002 * n(x) + tTE x / 2`,
`n(x) = 0.2969 sqrt(x) - 0.1260 x - 0.3516 x^2 + 0.2843 x^3 - 0.1036 x^4`,
`ell(x) = t sqrt(x (1 - x)) + tTE x / 2`, `t = (t/c) c` and the trailing-edge thickness `tTE = 1.0`. It is laid off
normal to the chord on both sides of `c yc(x)`. The blade angle is
`beta(r) = alphaD + atan(tan(p0 - alphaD) * RefRadius / max(r, HoldRadius)) + (BladePitch - p0)`, with `alphaD = 5` degrees
and `p0 = 25` degrees (`RefRadius` 48 s, `HoldRadius` 21.5 s): the flow angle of a uniform axial inflow plus a design angle
of attack, constant inboard of `HoldRadius`. A
fan adds energy where a turbine takes it, so the angle is flow plus attack, not flow minus attack. The section is turned to
`beta(r)` at `BladePitch = 25` degrees about its stacking point, the mean-line point at chord fraction `0.40 + 0.10 w`,
which lies on the radial line. The loft always uses the sections at 25 degrees. For another pitch the finished loft is
turned by `BladePitch - 25` degrees about the radial line before the stud joins it, because Decad's twisted loft panics on
some section tables that depend on the pitch. Outboard camber is held to 3 to 4 %, because Decad's loft refuses thin
sections with more. The trailing edge is a flat 1.0 wide, which keeps its two corners more than the checker's 0.8 window
apart (1.2 would thin the leading edge until it fails near the tip), and the tip's ratio of 0.22 gives the leading edge a
radius above 0.4 near the tip. These two make the blade pass the Corners and roots check: the trailing edge is a blunt tip of
at least 0.8, and the leading edge is rounded by at least 0.4.

## Replaceable blades

- To remove a blade, undo its bolt from the front face. The blade nut is free in its slot: take it out through the slot's
  mouth below the blade root, or leave it. Pull the blade out along its radial line. The stud leaves the socket after `HubRadius -
  BladeStudStart` mm (7.9 at 128, 10.805 at 192 and 224: the stud's inner end to the hub's rim). The hub can stay on the shaft, since every blade bolt's head is
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
| Stud in its socket | Regular | 0.40 radial (3.4 against 3.0 at 128; 4.45 against 4.05 at 192) |
| Stud's inner end to the socket floor | Regular | 0.40 (`BladeStudStart` against `BladeSocketStart`), with the 0.3 stud chamfer clear of the 0.5 floor round |
| Blade root to the hub's rim | Regular | 0.40 (`RootRadius` against `HubRadius`) |
| Bolts, shank in the stud, bolt in its nut | Fastener rules / nominal exception | head pocket 0.3, hole 0.3, cavity 0.2; the bolt in its own nut is not measured |

The hub presses onto the shaft, flat to flat, and the set screw presses the flat. The shaft must reach at least 6 mm into the
hub (past z 6, beyond the hole's edge at 5.3) and should end at or before the front face (z `HubLength`). The bore runs through.

## Named exceptions

The user confirms these.

- X1. The set screw's head is recessed, not flush: at 128 4.05 (605ZZ), 2.80 (608ZZ) and 4.35 (`regular-bore`) below the
  round face at x = 0, and at 192 4.95, 3.70 and 5.25. No standard M2 length ends flush, and a set screw's head must float. Nothing stands proud. The user approved
  this.
- X2. The set screw's tip stands `2c` (0.1 at the presets) inside the bore, on the flat of the shaft it presses. That is the
  set screw's job; it stands outside no outer face.
- X3. The set screw's 2.6 hole crosses the 1.6 press-fit wall round the bore, at z 2.7 to 5.3. The wall is 1.6 or more
  everywhere else over the pressed length.
- X4. The print cradle's gap to the blade (`CradleGapZ` 0.2 up, `CradleGapXY` 0.4 sideways) is a support interface, not a
  part fit, so "Part clearances" does not apply to it. The cradle and the blade are proven apart (`decad.Intersect`
  empty); the slicer prints the gap as a peelable interface.

## Valid range

- `RotorDiameter` 128 to 224. The tested sizes are 128 (`compact`), 192 (the default) and 224 (`large`).
- `ShaftDiameter` up to 8.1 at the default clearance and 128 mm, and 12.1 at 192 and 224 (rule 6). `FlatDepth` at most `D/4`, with the shaft's flat 2 mm wide or more and
  the bore's flat 2.8 mm wide or more (for D = 5 and c = 0.05: 0.42 to 1.25, the lower limit from rule 6, `d (5.1 - d) >=
  1.96`; rule 5 alone would allow 0.21). `BoreClearance` 0.05 to 0.5, and for the 8 mm shaft 0.05 to 0.1.
- `BladePitch` 10 to 40 degrees, the blade angle at `RefRadius` (75 % of the tip radius).

## Representative cases

- `605ZZ`: the default (192 mm); the least wall to the set screw slot is 1.6.
- `608ZZ`: the largest bore, 3.645 to the socket floors at 192, the D loop chamfered.
- `regular-bore`: `Params605ZZ` with `BoreClearance` 0.35, the thinnest web behind the nut, 1.3.
- `steep`: `Params608ZZ` with `BladePitch` 40 degrees.
- `flat`: `Params605ZZ` with `BladePitch` 10 degrees.
- `compact`: `Params608ZZ` with `RotorDiameter` 128: the fan as it was before `RotorDiameter` existed (hub 14 x 20, M2 x 16
  blade bolts, M2 x 6 set screw, 1.65 to the socket floors).
- `large`: `Params605ZZ` with `RotorDiameter` 224 and `BladePitch` 40 degrees: the largest rotor at the steepest pitch (hub 18.9 x 35,
  blade bolt tip at z 8).
- Left hand: `Params605ZZ` with `Hand` `LeftHand`. Its test proves the mirror relation (each blade's volume and bounds equal the
  right-hand fan's mirrored blade reflected in x = 0) and that the blade nut slots stay reachable; the full per-blade checks
  run for the right hand.
- Placed: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.

## Print pose and cradle

The blade prints lying down, its radial line rising toward the tip, so layers run along the blade and take the
sideways force it carries in use. It rests on the bed at two points, the stud end corner and the leading edge near
stations 4 and 5, and lies in a one-piece cradle that fits its contour below its widest lines, like a mold half. The
cradle is a separate body offset from the blade by a support gap, so the slicer prints the contact as a peelable
interface. There is no raft.

### Pose

The pose is set from the blade's own chord at `RefRadius`: `u`, the in-section up, is that chord (leading to trailing
edge, X negated for `LeftHand`) turned by `PrintLean` (-15 degrees, mirrored for `LeftHand`) toward the pressure face, so
the leading edge points down. The radial line rises at the tilt `t` from the bed. `PrintPoseFor(size, pitch, hand, t)`
maps the blade's local `u` to `(-sin t, 0, cos t)` and `-Y` (the in-section side) to `-y`, with frame origin
`(-BladeStudStart cos t, 0, -zc(t))`, `zc(t) = (BladeStudStart + 0.3) sin t - BladeStudRadius cos t`, so the stud end
corner (the stud's chamfered rim on the down side) lies at z = 0 (`PrintLift`). The posed loft does not depend on the
pitch, because the pose follows the chord; only the bolt hole turns. The left hand is the right hand reflected in y = 0.

`RestTilt` finds the tilt at which the loft's lowest point reaches the bed too: it meshes the blade at 0.05, keeps the
vertices beyond r `RootRadius + 0.5`, and bisects 50 times on [`RestTiltMin`, `RestTiltMax`] for
`zc(t) = min (sin t v.Z + cos t v . u)`. It reads 0.198526, 0.206587 and 0.218303 rad (11.4, 11.8 and 12.5 degrees) at 128,
192 and 224, the same at every pitch and hand. The blade then touches the bed at the stud corner (0.67 to 0.76 mm2 in the
first layer) and the leading edge near stations 4 and 5 (3.4, 5.3 and 6.2 mm2 at 128, 192 and 224).

### Constants

| Name | Value | Reason |
| --- | --- | --- |
| `CradleGapZ` | 0.2 | vertical support gap: one 0.2 mm layer, the usual support top distance |
| `CradleGapXY` | 0.4 | horizontal support gap: about one line width; two paths closer than that fuse |
| `CradleWall` | 2.0 | wall beyond each widest line: five lines |
| `CradleTopDrop` | 0.3 | each wall's top under its smoothed limit, so no cradle stands at or above the widest line |
| `CradleTopBand` | 0.1 | the wall's limit is the lowest blade outline point within this of the widest y, steady on a near-vertical face |
| `CradleTopSmooth`, `CradleTopPasses` | 1.0, 8 | each wall top is averaged over 1.0 either side and lowered back under its limit, 8 times |
| `CradleStep` | 0.5 | cradle section spacing along x |
| `CradleFitPoints` | 33 | fit points per shadow section; the twisted loft allows 32 spans |
| `CradleYStep` | 0.05 | y spacing of the lower envelope's samples |
| `CradleTipMargin` | 1.0 | the cradle ends this far before the tip cap's lowest point |
| `CradlePlanWindow` | 1.0 | the plan's outer edge takes the widest point within this in x, so the bed face has no notch narrower than its chamfer |
| `StudSlotFlats` | 16 | flats of the stud slot's half polygon |
| `SectionStep` | 0.25 | spacing of the blade sections the layout reads |

### Construction (`BuildBladeHarness`)

The blade must already be posed. `F` is `PrintPoseFor(size, DefaultBladePitch, hand, t)`.

1. **Validate**: nil context, document or blade; a `Size` that `SizeFor` did not make; a hand that is neither; a tilt
   outside [`RestTiltMin`, `RestTiltMax`].
2. **Sections**: cut the posed blade by planes x = const every `SectionStep` (`SectionOutline` at 0.02, retried at
   0.05). `low(x)` is the loop holding a slice's lowest point.
3. **Range**: from 0.5 past the stud end rim's greatest x to `CradleTipMargin` before the least x of station 10's
   leading and trailing points at `TipRadius`.
4. **Cradle sections** (`CradleSection`) every `CradleStep`: over the slices within `CradleStep`, the min-y and max-y
   points (the widest points), the lower envelope on a `CradleYStep` grid, offset by `(n_y CradleGapXY, n_z
   CradleGapZ)` along its outward normal (the ends straight out by `CradleGapXY`), resampled at `CradleFitPoints` equal
   polar angles about `(0, z0)`, `z0 = box.Max.Z + 2`, so the twisted loft's sections stay star-shaped. `LeftLimit` and
   `RightLimit` are the lowest outline points within `CradleTopBand` of each widest y, read along every outline segment.
5. **Step zone** (`StepZone`): `[x_root - h - CradleStep, x_root + h + CradleStep]`, `h = 4 j sin t`, round the root step.
6. **Shadow tool**: `decad.LoftSections` through the sections from `StepZone[0] - 1.0` on, each on a world-XY plane at its
   x, its offset outline closed up to `zTop = box.Max.Z + 3`; the sections before `StepZone[1]` copy the first section past
   it. Turned by the frame `((0, 0, z0), +Y, +Z)` into place.
7. **Stud slot**: a half polygon of `StudSlotFlats` flats round a half circle of radius `BladeStudRadius + CradleGapXY`,
   `CradleGapXY - CradleGapZ` under the stud axis, carried straight up, extruded along the axis from 1 before the stud end
   to 0.5 past the root.
8. **Block**: the sections but the first and last, those from `StepZone[0] - 1.0` to `StepZone[1]` taking the first clean
   section's values, as the shadow tool does (a raw section there would raise a fin on each wall). The plan runs
   `CradleWall` outside the widest lines widened over `CradlePlanWindow`, on z = 0, its bed loop chamfered 0.5; the
   profile slab starts at z = -1 and stands 0.5 over the higher wall top. Each wall's top (`WallTop`) is its limit
   smoothed `CradleTopPasses` times (averaged over `CradleTopSmooth`, then lowered back under the limit) less
   `CradleTopDrop`; a cap on each side cuts what lies beyond that side's widest line and above its wall top.
9. **Cradle**: the block cut by the stud slot, then by the shadow tool. It must be one piece. Near the bed contacts the
   shadow cuts it down to the bed: under the leading-edge contact its bed face has a hole (13 to 24 mm2), and under the
   stud's first 2 mm there is no cradle. Where the blade is under 0.6 mm over the bed (the near-bed band, `CradleGapZ`
   plus two layers) the cradle cannot fit under it; those surfaces start one or two layers up and sag onto the bed
   slightly.

The harness has the shadow sections (from `StepZone[0] - 1.0` on) plus 7 sketches: the stud slot, the block's
plan and profile, and each cap's plan and profile, all fully constrained. Sampled points are placed by
`sketchcheck.DimensionPoint`: they are sampled data with no geometric relation to constrain.

### Overhangs and fit

Every cradle face points up, stands vertical or lies on the bed; every blade face pointing down more than 45 degrees rests
over the cradle (within 1.0, 2.2 in the step zone and its lead-in), lies in the bolt hole's band, on the stud end's 0.3
chamfer (45 to 51 degrees, within 0.9 mm of the bed, before the cradle starts), past the cradle's tip end
(the last 2 to 4 mm of the tip, a short cantilever) or in the near-bed band. The cradle stays within `CradleGapXY + 0.3` of
the blade below its widest lines (0.608 at most outside the step zone, measured on the cradle's 0.09 to 0.14 mesh), 2.2
in the step zone, except the bolt hole's band, the root step's face (which faces the stud end and needs no support) and
the near-bed band. No cradle stands above either widest line.

### Removal and cleanup

1. Lift the blade straight up out of the cradle; its two bed patches come off the bed with it.
2. Peel the cradle off the bed and discard it.
3. Every face the cradle supported shows support-interface texture: most of the pressure face, the leading edge, the
   lower suction face, the root and the stud's underside. Sand along the chord with 400 grit. Lightly sand the two bed
   patches (elephant's foot on the stud corner and the leading edge). Check that the stud slides into a hub socket.

## Print aids

The print cradle (`BuildBladeHarness`, `BuildBladePrint`) is a print aid: a body printed with the blade only to support
it while printing, then removed and discarded. "Corners and roots" exempts it (`docs/model-pipeline.md`); its walls,
lips and bed-face cuts have sharp corners by design.

## Retained sharp corners and edges

The rule is "Corners and roots" in `docs/model-pipeline.md`. The hub prints front face down with its axis vertical. Each
blade prints lying in its print pose ("Print pose and cradle"); the test reads it both across its radial line and lying,
at every 1 mm above the bed. The cradle is a print aid ("Print aids") and is exempt.

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Hub | The two corners of the D bore | Mating corners of the shaft's flat. They turn about 36 degrees, so they need no round. |
| Hub | The set screw slot's corner edge at z 6.51 (along Y), the blade slots' faces' edges (z `HubLength - BladeBoltCavity - 2` and `HubLength - BladeBoltCavity`), the adapter slots' faces' edges (z 1.5 and 3.5) and the adapter holes' blind ends (z 6.75) | The set screw slot's corner is a mating corner with the nut's corner and runs parallel to the bed; the blade slots' faces are between layers, sized by the Fasteners rule. |
| Hub | The counterbore's 0.5 floor round, where a layer off the screw's line cuts it at a slant | A layer at z 2.5, 3.4 and 5.5 chords the round into turns of 50.1 to 53.1 degrees; the test exempts that band only. |
| Hub | The set screw slot's 0.5 side rounds, low in the slot (z up to 2.6) | A layer at z 0.75 or about 2.45 chords the round into a turn of 50.0 to 50.6 degrees, 0.6 over the limit, on some hubs and not on others (none on `compact`); the test exempts that band only, and the exemption need not claim a finding. |
| Hub | The set screw slot's 0.5 rounds where a layer cuts its sloped end (z 5.0 to 6.51) | Layers at z 5.125, 5.5 and 5.8 read turns of 53.6 to 58.5 degrees; the test exempts that band only. The adapter holes' blind floor at z 6.75 adds the mid heights 5.575 and 12.375, which cut the same round and the socket floor rounds at a slant, so the face-level call exempts both. |
| Hub | The blade head pockets' floor rims | Between layers. |
| Hub | The adapter slots' 0.5 rounds | The hexagon's three corners are rounded like the blade slots'; a layer at z 2.5 cuts them and reads no finding. |
| Hub | The socket mouths | Between layers, out of the checker's sections. |
| Hub | The socket floor's 0.5 round, where a layer off the socket's axis cuts it at a slant | A layer at `HubMid - 3 j` or `HubMid + 3 j` chords the round into turns of 58.0 to 59.5 degrees near where it meets the socket wall; the face levels read 52.1 (compact, z 12.375) to 61.8 (large, z 21.7) there. The test exempts the round only, measured in each socket's frame: `BladeSocketStart` to `BladeSocketStart + 0.5` along the socket's axis and `BladeSocketRadius - 0.5` to `BladeSocketRadius` from it. |
| Blade | At 128, the four corners where the bolt hole crosses the stud, at r `BladeBoltAt` and `BladeBoltAt + 0.7` | The material angle is 64 to 90 degrees; the checker adds the stud's curvature and reads tips of 125 to 130 degrees. At 192 and 224 the stud is wider (radius 4.05), the angle exceeds 60 degrees and the layers read no corner. |
| Blade | The step where the stud meets the root face (r `RootRadius`) | Lying, the step lies inside the layers and reads inside corners of 75 to 107 degrees, within 1.0 of r `RootRadius`. Retained pending a root fillet, not an exemption: the model is on `CORNER_PENDING` and `CORNER_PARTIAL` in `scripts/check_repository.py` until a 1.0 stud-root fillet lands. |
| Blade | The leading edge and the trailing-edge flat where a layer cuts them obliquely | Lying at the rest tilt, a layer cuts each edge at 11 to 21 degrees, so each section's loop ends in a stretched tip (122 to 165 degrees) within 1.5 of the loop's least or greatest x. The edges are smooth and pass when read across the radial line. |
| Blade | The bolt hole where it crosses the stud, read lying | Lying, the layers cut the hole's walls where it crosses the stud and read inside corners of 82 to 110 degrees and tips of 123 to 165 degrees, within 1.8 of r `BladeBoltAt`; the same corner as the row above, at every size. |

Rounds below half their section: the socket floor round only. The hub has no stressed root. Every hub wall is 1.1 mm or
more (the 608ZZ lead-in band), elsewhere 1.3 or more (all at 128; the larger hub keeps those walls and adds more); the outward corners where the slots and the counterbore meet the round
face are about 99 degrees, above the 60 degree minimum.

Press-fit wall: 1.6 or more round the bore over its pressed length: 1.60 to the set screw slot, 1.65 to the 608ZZ socket
floors at 128 and 3.645 at 192. The set screw's 2.6 hole crosses that wall (X3). The hub prints with its bore axis vertical.

Surface edge finishing: both faces are large (about 570 mm2 on the back face and 580 on the front at 605ZZ; 541 and 551 at
608ZZ, at 128; about 1100 mm2 at 192). One `edgefinish.Chamfer` call chamfers both faces' outer circles by 0.5, and both D loops when 20 or longer (608ZZ,
25.1). Short runs are left sharp: the bolt holes (8.2), the blade head pockets (13.8), the set screw slot's mouth (12.7), and
the 605ZZ and `regular-bore` D loops (15.8, 17.7). None of these runs scales with the size. The round face and the cuts in it are not planar.
The print cradle's bed face is large (534 mm2 at 128, 1059 at 192, 1383 at 224): its outer loop is chamfered 0.5 on the
plan prism before the block is cut. Its largest other planar run, the tip-end face, is under 400 mm2 (207 to 252), so the
rest of the cradle takes no chamfer. Two bed-face edges stay sharp: the rim of the hole the blade's shadow cuts under the
leading-edge contact, and the edge where the stud slot cuts the cradle's stud end down to the bed. The blade sets their
angle, and the chamfer cannot reach them.

Body containers: `Hub`, `Blades`, `Bolts` and `Nuts`; `BuildBladePrint` files the posed blade in `Blades` and its cradle,
one piece, in `Harness`. Look-alike part marks: none; the bolts and nuts are bought parts. Split
shafts: the model has no printed shaft, arbor, axle or pin; the blade's stud is part of the blade. Arbor collars: no
printed shaft either. The consuming mechanism's shaft carries the hub and owns the collar between the hub's back face and its
bearing or plate. Gear train clearances: no gears.

## Printing

| Part | Prints with | Pause |
| --- | --- | --- |
| Hub | front face down, axis vertical | none |
| Blade (three) | lying on the bed in its print cradle, leading edge down ("Print pose and cradle") | none |

The blade lies on the bed at its rest tilt, touching it at its stud end corner and its leading edge, in a cradle that
fits its contour; slicer supports stay off. No brim is needed with PLA or PETG. If the stud corner lifts, a 3 mm slicer
brim on the blade's two bed patches fixes it but marks the stud end and the leading edge; with ABS or ASA, put a 5 mm brim
on the cradle only. The set screw slot opens upward in print, its hexagonal end down. Each
blade slot's roof (z `HubLength - BladeBoltCavity - 2`) is a 4.35 wide bridge. The
counterbore and the holes are horizontal round holes 4.4 and 2.6 across.

To assemble:

1. Hold the hub back face up and drop an M2 nut into the set screw slot. It stops at the slot's hexagonal end, in line with
   the hole.
2. Push the set screw (M2 x 10 at 192) into the counterbore on the -Y side and turn it into the nut about two turns. Stop while its tip
   is still out of the bore (look into the bore from the back).
3. Fit each blade (see "Replaceable blades").
4. Press the hub onto the shaft, flat to flat, the flat toward the set screw, at least 6 mm deep.
5. Tighten the set screw with a 1.5 mm hex key until its tip presses the flat.

## Checks and images

Check every case above (compact, default and large sizes alike): every sketch fully constrained (27 from `BuildParts`: the hub's own, with two adapter hole sketches
and two adapter slot sketches, and the blades'; 20 from `Build` in `ShaftMount`, 25 in `ClutchMount`), the twelve bodies (fourteen in `ClutchMount`) and their
containers, the bounds at 128, 192 and 224 and the `VerifyAll` meshes, the placed fan against the local one, the blade
envelope (every blade vertex at 0 <= z <= `HubLength` and r <= `TipRadius` + 0.5), the bore's clearance round the nominal shaft, its wall and that it runs through,
the face chamfers, the set screw (head, tip, floating head over a rounded floor, nut, slot shape, no overlap, every wall),
each blade bolt's head, tip and nut with its slot and no overlap, both adapter cuts on the `ShaftMount` hub (slot open to the
round face, 1.5 of hub under it, the wall to the bore), access to the set screw's path and each blade nut's path,
every Regular fit, the blades' pitch and hand, the copies turned by 120 and 240 degrees, the replacement paths, and
`internal/cornercheck` on the hub (axis +Z) and each blade along its radial line and lying in its print pose.

The print aid's checks run on six cases built once each (128/25 right, 192/25 right, 192/25 left, 192/40 left, 224/25
left, 224/40 left): `TestPrintPose` (the rest tilt per size, the same at every pitch and hand, the stud corner on the
bed), `TestBuildBladeHarness` (one solid piece on the bed, every sketch fully constrained, the bed face over 400 mm2, the
left hand mirroring the right), `TestBladeHarnessVerifyAll`, `TestBladeHarnessSeparate` (`decad.Intersect` empty),
`TestBladePrintBedContact` (two bed patches), `TestBladeHarnessBedFace` (one outline, at most one hole under the
leading edge), `TestBladeHarnessWallTop` (under the widest line, no fin, no zigzag), `TestBladeHarnessContour` (the
cradle within `CradleGapXY + 0.3` of the blade), `TestBladeHarnessNothingAbove`, `TestBladeHarnessOverhangs` and
`TestBuildBladePrint`. The images `print.png`, `print-side.png`, `print-end.png` and `parts/harness.png` show the
print job.

## Clutch mount

### Purpose, frame turn and stack

`Params.Mount = ClutchMount` bolts the fan to the standard adapter ring (`onewayclutch.BuildAdapter`) of
[`models/onewayclutch`](../onewayclutch/) instead of gripping the shaft. The shaft drives the fan through the clutch, and
the fan coasts when the shaft stops or turns back. The hub is the `ShaftMount` hub: one part, no flange and no clamp ring.
The clutch and the adapter are not modified by the fan; the fan side supplies only the hub's two adapter bolt holes and
nut slots, which are present in both mounts.

The fan local frame is unchanged (origin on the axis in the hub's back face, +Z forward, blade 1 along +Y). `MountFrame(C)`
returns the fan's placement point: clutch-frame origin (0, 0, `CoverTo` = 18.5), the clutch cover's outer face, with
U = (cos 180, sin 180, 0) = (-1, 0, 0) and V = (-sin 180, cos 180, 0) = (0, -1, 0), then through the clutch frame C. The fan
frame's own origin is `ClutchAnchorZ` = -7.75 from that point, so the hub's back face lands at clutch z `MountZ` = 26.25.
A fan angle is the clutch angle plus 180 degrees, so the bore's flat (-Y in the fan frame) points where the shaft's flat
(+Y in the clutch frame) does. The clutch's ears and face holes at 90, 210 and 330 degrees land at fan angles 270, 30 and
150, midway between the blades at 90, 210 and 330, and a hex key reaches each ear bolt from the front. The fan uses the face
holes at 210 and 330 degrees (`FaceSites[1]` and `[2]`, fan 30 and 150); the hole at 90 degrees (`FaceSites[0]`, fan 270)
lies under the hub's solid back face and stays empty. Both frames share +Z.

| Item | Fan z | Clutch z |
| --- | ---: | ---: |
| Hub front face | `HubLength` (30 at 192) | `MountZ + HubLength` (56.25 at 192; 46.25 at 128) |
| Hub back face (seam 0.25) | 0 | 26.25 |
| Adapter front face | -0.25 | 26 |
| Adapter nut slot | 1.5 to 3.5 | 27.75 to 29.75 |
| Adapter bolt tip (seam open) | 5.75 | 32 |
| Blind hole floor | 6.75 | 33 |
| Adapter rear face, adapter bolt head tops (flush) | -6.25 | 20 |
| Ring front end (seam 0.25) | -6.5 | 19.75 |
| Clutch cover outer face, cover bolt heads | -7.75 | 18.5 |

The clamped faces are modeled with the clutch's 0.25 mm seam and close when the bolts are tightened, as the clutch's own
plates do. The hub has no bore contact with the shaft: the adapter's bore (7 mm on the 605ZZ clutch, 10 mm on the 8 mm clutch) clears the shaft, and the shaft stops short
of the hub's back face.

### Constants

| Name | Value | Definition / reason |
| --- | ---: | --- |
| `ClutchTurn` | pi | fan frame turn about Z from the clutch frame; it lines the bore's flat up with the shaft's |
| `ClutchSeam` | 0.25 | `onewayclutch.Seam` |
| `MountZ` | 26.25 | `onewayclutch.AdapterTo + ClutchSeam`: the fan frame's origin in the clutch frame |
| `ClutchAnchorZ` | -7.75 | `onewayclutch.CoverTo - MountZ`: the placement point of a `ClutchMount` fan, in the fan frame |

The adapter's own constants are in [`models/onewayclutch`](../onewayclutch/instructions.md): `FaceCircle` 9.5, `FaceLength` 10
(an `m2fastener.BoltSize.StandardLengths` entry) and `FaceNutDepth` 1.5. The fan reads them; it does not redefine them.

### Validation

The `ClutchMount` hub passes the same rules as the `ShaftMount` hub (rules 1 to 12 above). `AdapterSites[k]` (k = 0, 1, angle
`a = EarBoltAngle(k+2)`, 30 and 150 degrees, `u = (cos a, sin a, 0)`) is `m2fastener.Site{Head: (FaceCircle u, -(AdapterThickness
+ ClutchSeam)) = (9.5 cos a, 9.5 sin a, -6.25), Dir: (0, 0, 1), Corner: u, Length: FaceLength, Cavity: AdapterThickness +
ClutchSeam + FaceNutDepth}`. `Cavity` is 7.75, `Tip()` is 12 and `CavityFar()` is 9.75. In `ClutchMount` the set screw's
fields are filled, since the hub carries its cuts, but `Build` places no set screw.

### Geometry

1. **Hub.** As in "Geometry" above, with the adapter holes and slots, in `ClutchMount` too.
2. **Blades, bolts, nuts.** As in `ShaftMount`; the blades are the same part.
3. **Adapter bolts.** `Build` places each adapter bolt with `m2fastener.PlaceBolt` and its nut with `PlaceNut` at
   `AdapterSites[k]` through the placement. The adapter is the caller's: `doc := onewayclutch.NewDocument();
   onewayclutch.Build(ctx, doc, c); onewayclutch.BuildAdapter(ctx, doc, c)`, then `turbinefan.Build(ctx, turbinefan.NewDocument(),
   ParamsForClutch(c))`. The fan builds in its own document because both models file their hub in a container named `Hub`;
   one document would put the clutch hub and the fan hub in one container.
4. **Counts.** Bodies: 14 (hub, three blades, five bolts and five nuts). Sketches from `BuildParts`: 27 (1 hub outline, 1 socket tool, 6 blade pocket and slot tools, 3 set screw
   tools, 2 adapter hole tools, 2 adapter slot tools and 12 blade sketches); `Build` adds 25 fastener sketches (ten for five bolts, fifteen for five nuts). Containers: `Hub`, `Blades`,
   `Bolts` (blade bolts 1 to 3, adapter bolts 1 and 2) and `Nuts`.

### Fasteners

Each M2 x 10 adapter bolt's head sits flush in a pocket in the adapter's rear face (the adapter's pocket, `m2fastener.PocketTool`
at `onewayclutch.Dimensions.FaceSites`), and the bolt passes the adapter and the seam into the hub's blind hole. Each nut sits
in a hex slot open to the hub's round face, `FaceNutDepth` (1.5) behind its far face: no nut is embedded, so no print pause
(AGENTS.md "Fasteners"). The tightened bolt pulls the nut onto the slot's rear face.

| Check | Seams open (model) | Seams closed (tightened) |
| --- | --- | --- |
| Head top | flush with the adapter's rear face, fan z -6.25 | same |
| Tip | z 5.75, inside the blind hole (floor 6.75) and 2.25 past the slot's far face | z 5.50 |
| Nut position | z 1.7 to 3.3 | pulled onto the slot's rear face, z 1.5 to 3.1 |
| Shank in the hub's hole, in the adapter's hole | 0.3 radial (hole 2.6, shank 2.0) | same |
| Material under the slot | 1.5 | same |
| Nut in slot | 0.17 at the flats (0.07 beside the rounds), 0.12 at each rounded corner, 0.2 at each face | same |

The bolts cannot be reached once the adapter is on the clutch: bolt the hub to the adapter on the bench first (see
"Assembly").

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
| Hub back face / adapter front face | clamped seam | 0.25 | at least 0.20 |
| Adapter rear face / ring front end | clamped seam | 0.25 | at least 0.20 |
| Adapter rear face / cover, cover bolt heads | Regular, after the ring's 0.85 play and the seam closing | 1.50 | at least 1.45 |
| Adapter bolt heads / cover | Regular | 1.50 | at least 1.45 |
| Adapter bolt shank / adapter hole, hub hole | fastener rule | 0.30 | at least 0.28 |
| Adapter nut / slot | fastener rule | 0.17 flats (0.07 beside the rounds), 0.12 rounded corners, 0.20 faces | clear |
| Blades / adapter | Regular | 5.25 at 128; 7.75 at 192 | at least 5.0 |
| Blade / ear bolt (hex key room) | Regular | 5.6 at 128; 28.8 at 192 | at least 5.0 |
| Hub rim / ear bolt line (head and key room) | Regular | 7.6 at 128 (rim 14); 2.7 at 192 (rim 18.9) | at least `HeadPocketRadius + 0.5` (2.7) |

The ring floats 0.85 each way between the clutch's cones, and the bolted fan floats with it. Blade fits are unchanged.

Measured by `TestTurbineFanOnClutch` at 192 mm (mesh gap lower bounds, 605ZZ / 608ZZ clutch): hub / adapter 0.244 / 0.241;
adapter / ring 0.231 / 0.232; adapter / cover 1.495 / 1.492; adapter / the clutch bolt head 1.498 / 1.498; adapter bolt head
/ cover 1.497 / 1.493; each blade / adapter 7.746 / 7.746; blade 1 / ear bolt 1 28.842 / 29.591. At 128 the blade / adapter
gap was 5.251 / 5.252 and blade 1 / ear bolt 1 26.316 / 27.078. `TestTurbineFanClutchAccess` measures the hub's least gap to an
ear bolt line, over the ear circles of every clutch preset, at 2.700 (192 and 224) and 7.600 (128).

The hub's radius is capped at `MaxHubRadius` because of that gap: a 21 mm rim would cover the 605ZZ clutch's ear circle
(21.6), and the ear bolts' heads, driven past the hub from the front, would not fit between the rim and the blades' roots.

### Retained sharp corners, edge finishing and printing

- Hub: the same part as in `ShaftMount`: prints front face down, axis vertical, no supports. Each adapter slot roof (z 3.5)
  is a 4.35 bridge. The blades print as in `ShaftMount`, lying in their cradle.
- Retained sharp corners: as the hub's, plus the adapter slots' faces at z 1.5 and 3.5 (between layers) and the adapter holes'
  blind ends (z 6.75).
- Short runs left sharp: the adapter holes (8.2).
- Containers: see Geometry. No look-alike parts, printed shaft, arbor collars or gears. The adapter's own edge finishing,
  corners and walls are in [`models/onewayclutch`](../onewayclutch/instructions.md).

### Mounting space, assembly and cases

- Behind the ring: room to hold an M2 nut in each of the ring's three rear ear pockets (r 19 to 24 at the ring's rear face).
  In front: the adapter (51.4 across on the 605ZZ clutch, 53.0 to 53.9 on the 8 mm clutch, clutch z 20 to 26) and the fan. The shaft may end anywhere from inside the clutch to the
  adapter's front face (z 26); it must not enter the hub's back face. A shaft that entered the D bore would lock the fan to
  the shaft and defeat the clutch.
- Assembly: (1) fit the blades; (2) slide two M2 nuts into the hub's adapter slots from the round face, set the hub on the
  adapter's front face (its pockets toward the back) and drive two M2 x 10 bolts from the adapter's rear pockets; (3) assemble
  the clutch, press it on the shaft and tighten its three bolts (the 2.6 mm access holes in the adapter pass a 1.5 mm hex
  key to the heads when the adapter is on); (4) hold an M2 nut in each of the ring's three rear ear pockets, set the adapter
  against the ring's front end and drive the three M2 x 25 ear bolts from the front between the blades with a 1.5 mm hex key.
- Cases: `clutch-605ZZ` (`ParamsForClutch(onewayclutch.DefaultParams())`, `Placement` nil), `clutch-608ZZ-left`
  (`{Shaft: "608ZZ", Freewheel: CounterClockwise, BoreClearance: 0.05}`) and the fan placed on the real clutch with its adapter
  (`TestTurbineFanOnClutch`).
