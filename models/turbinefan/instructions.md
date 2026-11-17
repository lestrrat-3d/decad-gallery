# Turbine fan

## Purpose and source

Construct a fan of three replaceable blades on one printed hub. The hub is a plain round cylinder, radius 14 and 20 long,
with a D bore for a D-flat shaft. One radial M2 x 6 bolt acts as a set screw: it threads through a nut in a hex slot open
to the hub's back face, and its tip presses the shaft's flat. Its head floats, recessed, in a counterbore cut into the
round face, so nothing stands outside radius 14 except the three blades. Each blade is this model's blade
(`blade.go`), with its round root stud that slides into a radial socket in the hub and one M2
bolt that runs parallel to the shaft through the hub and the stud into a nut. That nut sits in a hex slot open to the round
face, below the blade, so no nut is embedded and the hub needs no print pause. The same hub also has two blind M2 bolt
holes in its back face, each with a hex nut slot open to the round face: they take the two M2 x 10 bolts that hold it to
the standard adapter ring of a one-way clutch (see "Clutch mount"). There is one hub for both mounts; it has no flange and
no clamp ring. The parts are:

- a hub, a cylinder cut by the D bore, three sockets, the blade bolts' pockets and nut slots, the set screw's
  counterbore, hole and nut slot, and the two adapter bolts' holes and nut slots;
- three blades, one part printed three times;
- four M2 bolts (three M2 x 16 blade bolts and the M2 x 6 set screw) and four M2 nuts in `ShaftMount`; five M2 bolts (the
  blade bolts and two M2 x 10 adapter bolts) and five M2 nuts in `ClutchMount`, which has no set screw.

The shaft is not modeled. It belongs to the mechanism, and its diameter equals the bore of the purchased bearing it runs
in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets read that bore from
[`models/ballbearing`](../ballbearing/). The bolts and nuts are [`models/m2bolt`](../m2bolt/) and
[`models/m2nut`](../m2nut/), placed with `internal/m2fastener`.

All dimensions are in millimeters; angles are radians in code and degrees in prose. The model owns the blade and its
joint: it exports `BuildBlade`, `BuildBladeMirrored`, the blade socket tool and the hub and blade-joint constants, and
[`models/handcrankfan`](../handcrankfan/) imports them with `Params608ZZ`. It reads `PressWall`, `ChamferSize`, `RegularFit` and the shaft flats from `crankhandle`.

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
  `NutsContainerName` (`Nuts`).
- The clutch-mount constants `ClutchTurn`, `ClutchSeam`, `MountZ` and `ClutchAnchorZ` (see "Clutch mount").
- `Hand` with `RightHand` (the zero value) and `LeftHand`; `Hand.Freewheel()` returns the `onewayclutch.Direction` that
  drives a fan of that hand forward.
- `Mount` with `ShaftMount` (the zero value) and `ClutchMount`; `EarBoltAngle(k)` is ear bolt k's angle in the fan frame.
- `Params` with `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `BladePitch`, `Hand`, `Mount` and `Placement *r3.Frame`
  (nil leaves every part in the local frame; a frame puts the placement point on it, see "Local frame and hand").
- `MountFrame(clutch *r3.Frame) (r3.Frame, error)` and `ParamsForClutch(onewayclutch.Params, ...ClutchOption) (Params, error)`.
  `WithBladePitch(pitch)` is the one `ClutchOption`; it replaces `DefaultBladePitch` and the result is validated.
- `Params605ZZ()`, `Params608ZZ()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`. `Dimensions` holds `BoreRadius`, `BoreFlat`,
  `BoreFlatWidth`, `BoreLoop`, `ShaftFlatWidth`, `SetScrewTip`, `SetScrewHead`, `CounterboreFloor`, `SetScrewSite`,
  `BladeSites` and `AdapterSites`, in the local frame. `SetScrewSite` and `AdapterSites` are filled in both modes.
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.1 mm), opts...)`. The blade's stud
  union refuses a plain document; build in such a document.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the hub and the three blades and puts them in no container. `Parts`
  holds `Hub`, `Blades`, every `Sketches`, `Dimensions`, and in the world frame `BladeSites`, `SetScrewSite` and
  `AdapterSites`.
- `Build(ctx, doc, p) (*Model, error)` adds the parts, the bolts and the nuts, each in its container. `Model` embeds `Parts`
  and holds `BladeBolts`, `BladeNuts`, `SetScrew` (`*m2bolt.Model`) and `SetScrewNut` (`*m2nut.Model`), and in
  `ClutchMount` `AdapterBolts` and `AdapterNuts` (the set screw and its nut are nil). `Build` builds neither the adapter nor
  the clutch: the caller builds them in the clutch's own document (see "Clutch mount").

The blade exports other models use: `BuildBlade(ctx, doc, pitch)` returns the unplaced blade and its twelve sketches,
`BuildBladeMirrored` the blade of the other hand (its sections not flipped and its pitch turn reversed, so it is the first
blade reflected in the plane x = 0 of its local frame), `BladeSocketTool(ctx, doc, f)` builds a socket cutter; and
`HubRadius`, `HubLength`, `BladeBoltAt`, `BladeBoltLength`, `BladeBoltCavity`, `BladeStudRadius`, `BladeStudStart`,
`BladeStudEnd`, `BladeSocketRadius`, `BladeSocketStart`, `BladeSocketFloorRound`, `DefaultBladePitch`, `MinBladePitch`,
`MaxBladePitch`, `BladeRootRadius()`, `BladeTipRadius()` and `Beta(r, pitch)`. From `crankhandle`: `PressWall`,
`ChamferSize`, `RegularFit`, `Params605ZZ` and `Params6082RS`.

## Presets and defaults

| Field | `Params605ZZ` (default) | `Params608ZZ` |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params608ZZ().BoreDiameter` |
| `FlatDepth` | 0.5, the planetary gearbox shafts' flat | 0.75, the siren shafts' flat |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `BladePitch` | 25 degrees, `DefaultBladePitch` | 25 degrees |
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
5. `BladeSocketStart - r_b >= PressWall`, so 1.6 mm stays between the bore and the socket floors at r 5.7: `r_b` at most
   4.1. Both mounts take this rule: one hub serves both.
6. `w_s >= m2bolt.ShankDiameter`, else `FlatDepth must leave the shaft's flat at least 2 mm wide for the
   set screw's tip`.
7. `w_b >= 2 (HoleRadius + 0.1)`, else `FlatDepth must leave the bore's flat at least 2.8 mm wide for the
   set screw's hole`. The 0.1 keeps the hole's cylinder off the bore's round.
8. `BladePitch` lies in [10 degrees, 40 degrees].
9. `Hand` is `RightHand` or `LeftHand`.
10. `Placement` is nil or a valid frame.
11. The site checks, constant over the valid range and the same in both mounts, in this order: `SetScrewSite.Validate()`
    (`set screw: ...`); `rho_h <= sqrt(HubRadius^2 - (HeadDiameter/2)^2)` (13.8705), else `set screw: its head would stand
    out of the round face`; `Cavity - HeadPocketDepth - SetScrewTravel >= 1`, else `set screw: less than 1 mm between its
    counterbore and its nut slot`; `SetScrewZ - HeadPocketRadius >= 1`, else `set screw: less than 1 mm between its
    counterbore and the back face`; each `AdapterSites[k].Validate()` (`adapter bolt %d: ...`); then each
    `BladeSites[k].Validate()`.

`FlatDepth` 0.2 on the 605ZZ fails rule 6. 0.25 passes rule 6 and fails rule 7, in either mount.

BuildParts and Build reject a nil context and a nil document, derive the dimensions, and check `ctx.Err()` before any
geometry. A rejected build adds no body.

## Local frame and hand

The origin is on the shaft axis in the hub's back face, the face toward the mechanism. +Z runs along the shaft away from
the mechanism to the front face at z = 20. The bore's flat faces -Y, at y = -`f_b`, and the set screw enters from -Y along
the line x = 0, z = `SetScrewZ` = 4, pointing +Y. Blade k lies along `u_k`: blade 1 along +Y, blade 2 at 210 degrees and
blade 3 at 330 degrees, counter-clockwise seen from the front. The set screw lies at 270 degrees, midway between blades 2
and 3. A `Placement` maps this frame onto the given one after each part's last operation, so that the placement point lands on
the given frame's origin and the local axes land on its U, V and normal. The placement point is on the shaft axis. In
`ShaftMount` it is the hub's front face (local z = 20, the face the blades face), and the hub extends `HubLength` back
along the given frame's axis. In `ClutchMount` it is the clutch cover's outer face (local z = `ClutchAnchorZ` = -7.75), with
the clutch behind it and the adapter and fan in front. A nil `Placement` moves nothing: the parts stay in the local frame
above, with its origin in the hub's back face. Derived sites are in the local frame in every case.

Hand: a right-hand rotor blows air forward, along +Z and away from the mechanism, when the shaft turns clockwise seen from
the front. A left-hand rotor does the same when the shaft turns counter-clockwise: its blades are the right-hand blades
reflected in the plane x = 0 (`BuildBladeMirrored`), which maps blade 1 to itself and swaps blades 2 and 3.
The hub, bolts and nuts do not depend on the hand. For the right hand, a blade's chord from its trailing edge to its
leading edge at radius r runs along `(cos beta, 0, -sin beta)` with `beta = Beta(r, BladePitch)`: the leading
edge leads toward +X at the top and sits upstream (-Z), and the concave pressure face faces the front.

## Geometry

1. **Hub outline.** A sketch on z = 0: the circle of radius 14 about the axis; the D bore, an arc of radius `r_b` about the
   axis from `(w_b/2, -f_b)` counter-clockwise over +Y to `(-w_b/2, -f_b)`, closed by its flat, a horizontal line at distance
   `f_b` from the origin; and three circles of radius 1.3 at `BladeBoltAt` (9.5) along `u_k`, the blade bolts' holes.
   The profile has four holes: the three bolt holes and the bore. The bore runs through.
2. **Hub body.** Extrude 20 along +Z, then `edgefinish.Chamfer` of both faces by `ChamferSize`, taking the long loops: the
   outer circle on both faces and, when `L_b` is 20 or more (608ZZ), the D loop. Then cut, in this order:
   - the three sockets: `BladeSocketTool` on the frame with origin `(0, 5.7, 10)`, U = +Z and V = +X (normal
     +Y), and its two copies from `decad.PatternCopies` with a circular pattern of three about +Z;
   - for each blade k, `m2fastener.PocketTool` (head pocket, z 18 to 20, radius 2.2) then the blade nut slot (below);
   - the set screw's counterbore (below), `m2fastener.HoleTool(SetScrewSite)` (radius 1.3, from `rho_h + 1` to `rho_t - 1`, its
     inner end inside the bore) and the set screw's nut slot (below);
   - for each adapter bolt k = 1, 2 (angles 30 and 150 degrees), `m2fastener.HoleTool(AdapterSites[k])` (radius 1.3, blind,
     from the back face to z 6.75) then the adapter nut slot (below).
3. **Nut slot.** A hexagon about the bolt line with its corners on the circle of radius 2.51, one corner at the slot's end,
   whose two sides toward the opening run on as straight walls 2.1737 from the line, out through the part's face; 2.0 deep.
   - Set screw slot: plane `y = -(f_b + 1.6)`, extruded to `y = -(f_b + 3.6)`, U = -Z and V = +X, reaching `z = 6.51`
     at its end corner and open at z = -1 (below the back face). Every edge of both slot faces is rounded by `NutSlotRound`.
   - Adapter slot k: `slotTool` with a `Dir` +Z site puts the slot plane at `Head.Z + Cavity`, z = 1.5, U = `u_k` (the
     radial line at 30 or 150 degrees), V = `v_k`, the corner at r 6.99, open through the rim; z 1.5 to 3.5. Its three hexagon
     corners are rounded by `NutSlotRound`, as the blade slots' are.
   - Blade slot k: plane z = 2.4, U = `u_k`, V = `v_k`, the corner at r 6.99 on the radial line, open through the rim; z 2.4 to
     4.4. Its three hexagon corners, the vertical edges along Z where the end meets the walls, are rounded by `NutSlotRound`: each turns 60 degrees, so a layer would read it as an inside corner.
4. **Counterbore.** A revolved meridian about the screw's line: from the floor at `rho_f` a quarter-round of radius
   `CounterboreRound` (eight chords) up to radius 2.2, out through the round face. The head (radius 1.9) floats
   `SetScrewTravel` above the floor, where the round's surface is at radius 2.19, so it clears it by 0.29.
5. **Blades.** `BuildBlade(ctx, doc, BladePitch)` (`BuildBladeMirrored` for `LeftHand`) returns blade 1 in its own
   frame (origin on the fan axis in the hub's mid plane, +Z its radial line, +Y downstream, the blade moving toward -X).
   `decad.Transform` by the frame with origin `(0, 0, 10)`, U = (-1, 0, 0) and V = (0, 0, 1) turns it into place: blade-local
   Z becomes +Y, local Y becomes +Z and local X becomes -X. `decad.PatternCopies` with the hub's pattern gives blades 2 and
   3. Do not transform blade 1 again before copying.
6. **Blade facts.** A loft from r 14.4 to 64 through ten stations; a stud of radius 3 from r 6.1 to 16
   with a 0.3 chamfer on its inner end; a bolt hole of radius 1.3 along the fan axis at r 9.5 through the stud; sections
   that are NACA 4-digit mean lines blended with an ellipse at the root; a blade angle `Beta(r, BladePitch)` that is
   constant inboard of r 21.5; a trailing edge 1.0 thick. The rotor is 128 across. Every blade vertex lies between z 0 and
   20: the measured range is z 5.02 to 17.72 at the default pitch, 4.17 to 19.39 at 40 degrees and 5.85 to 15.55 at 10
   degrees.
7. **Containers.** `Build` files the hub, the three blades, the bolts and the nuts in their containers, each a child of the
   document root: four bolts and four nuts in `ShaftMount`, five of each in `ClutchMount`.

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
| Adapter bolt hole r 1.3 and nut slot (30 and 150 degrees): hole z; slot z, corner r | -6.25 to 6.75, blind; 1.5 to 3.5, 6.99 | same | same |
| Hub under the adapter slot (to the back face); slot corner to the bore | 1.5; 4.44 | 1.5; 2.89 | 1.5; 4.14 |
| Adapter slot to blade nut slots, to sockets | 3.9 or more, 3.5 or more | same | same |
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
14 rim; an adapter nut (z 1.7 to 3.3) reaches r 12.43 at most. The 0.5 rounds leave 0.07 of clearance at the nut's flats and 0.12 at each rounded corner of the blade slot (a 120 degree corner rounded by 0.5 takes 0.077 of the 0.2 past the nut's corner).

| Bolt | Head | Hole | Nut | Rule |
| --- | --- | --- | --- | --- |
| Blade bolt k (M2 x 16) | flush in the front face, pocket radius 2.2, 2 deep | radius 1.3 through the hub and the stud | hex slot open to the round face, entered radially, z 2.4 to 4.4 | open hex seat |
| Set screw (M2 x 6) | recessed in a counterbore radius 2.2, 0.4 above its rounded floor (X1) | radius 1.3 from the counterbore floor through the press-fit wall (X3) into the bore | hex slot open to the back face, entered along +Z | open hex seat |
| Adapter bolt k (M2 x 10, `ClutchMount` only) | flush in the adapter's rear face (z -6.25), in the adapter's pocket, not in the hub | radius 1.3 from the hub's back face to z 6.75, blind; the bolt's tip (z 5.75) stays inside | hex slot open to the round face, entered radially, z 1.5 to 3.5 | open hex seat |

`Build` places the set screw with `m2fastener.PlaceBolt` at `SetScrewSite` and its nut with `PlaceNut`. `SetScrewSite` is
`m2fastener.Site{Head: (0, -rho_h, SetScrewZ), Dir: (0, 1, 0), Corner: (0, 0, 1), Length: SetScrewBolt, Cavity:
rho_h - f_b - PressWall - NutCavityHeight}`; `Cavity` is 4.3 at the presets, so the slot spans `f_b + 1.6` to `f_b + 3.6`
and its inner face keeps exactly `PressWall` to the bore's flat. `Corner` +Z puts the nut's corners up and down and its
flats on +-X. `Tip()` is 8, so the tip lies at `rho_t`. The blade bolt sites are turned about the shaft
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

## Blade shape

The blade is a smooth loft through ten sections, stacked on its radial line through the hub's mid plane and starting at
r 14.4, 0.4 outside the hub rim. A root stud of radius 3, from r 6.1 to r 16, is joined to it, with a bolt hole of radius
1.3 parallel to the shaft at r 9.5. The stud's inner end has a 0.3 chamfer, so its corner stays clear of the socket floor's
round. The stud engages the socket from r 6.1 to r 14 with a 0.4 clearance (a Regular fit). The shape follows a wind
turbine blade: a round root carries the joint, the section turns into a thick oval and an airfoil at the maximum chord
(r 25), and the chord tapers to the tip.

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
`beta(r) = alphaD + atan(tan(p0 - alphaD) * 48 / max(r, 21.5)) + (BladePitch - p0)`, with `alphaD = 5` degrees and
`p0 = 25` degrees: the flow angle of a uniform axial inflow plus a design angle of attack, constant inboard of r 21.5. A
fan adds energy where a turbine takes it, so the angle is flow plus attack, not flow minus attack. The section is turned to
`beta(r)` at `BladePitch = 25` degrees about its stacking point, the mean-line point at chord fraction `0.40 + 0.10 w`,
which lies on the radial line. The loft always uses the sections at 25 degrees. For another pitch the finished loft is
turned by `BladePitch - 25` degrees about the radial line before the stud joins it, because Decad's twisted loft panics on
some section tables that depend on the pitch. Outboard camber is held to 3 to 4 %, because Decad's loft refuses thin
sections with more. The trailing edge is a flat 1.0 wide, which keeps its two corners more than the checker's 0.8 window
apart (1.2 would thin the leading edge until it fails near the tip), and the tip's ratio of 0.22 gives the leading edge a
radius above 0.4 near r 64. These two make the blade pass the Corners and roots check: the trailing edge is a blunt tip of
at least 0.8, and the leading edge is rounded by at least 0.4.

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
| Hub | The set screw slot's corner edge at z 6.51 (along Y), the blade slots' faces' edges (z 2.4 and 4.4), the adapter slots' faces' edges (z 1.5 and 3.5) and the adapter holes' blind ends (z 6.75) | The set screw slot's corner is a mating corner with the nut's corner and runs parallel to the bed; the blade slots' faces are between layers, sized by the Fasteners rule. |
| Hub | The counterbore's 0.5 floor round, where a layer off the screw's line cuts it at a slant | A layer at z 2.5, 3.4 and 5.5 chords the round into turns of 50.1 to 53.1 degrees; the test exempts that band only. |
| Hub | The set screw slot's 0.5 rounds where a layer cuts its sloped end (z 5.26 to 6.51) | Layers at z 5.5 and 5.8 read turns of 53.6 to 58.5 degrees; the test exempts that band only. The adapter holes' blind floor at z 6.75 adds the mid heights 5.575 and 12.375, which cut the same round and the socket floor rounds at a slant, so the face-level call exempts both. |
| Hub | The blade head pockets' floor rims | Between layers. |
| Hub | The adapter slots' 0.5 rounds | The hexagon's three corners are rounded like the blade slots'; a layer at z 2.5 cuts them and reads no finding. |
| Hub | The socket mouths | Between layers, out of the checker's sections. |
| Hub | The socket floor's 0.5 round, where a layer off the socket's axis cuts it at a slant | A layer at z 7 or 13 chords the round into turns of 57.8 to 59.8 degrees on the floor's outline, 5.7 to 6.6 from the shaft axis; the test exempts that band only. |
| Blade | The four corners where the bolt hole crosses the stud, at r 9.5 and 10.2 | The material angle is 64 to 90 degrees; the checker adds the stud's curvature. |
| Blade | The step where the stud meets the root face (r 14.4), opening downward when printed stud end down | Between layers. |

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

Check every case above: every sketch fully constrained (27 from `BuildParts`: the hub's own, with two adapter hole sketches
and two adapter slot sketches, and the blades'; 20 from `Build` in `ShaftMount`, 25 in `ClutchMount`), the twelve bodies (fourteen in `ClutchMount`) and their
containers, the default's bounds and `VerifyAll` meshes, the placed fan against the local one, the blade envelope (every
blade vertex at 0 <= z <= 20 and r <= 64.5), the bore's clearance round the nominal shaft, its wall and that it runs through,
the face chamfers, the set screw (head, tip, floating head over a rounded floor, nut, slot shape, no overlap, every wall),
each blade bolt's head, tip and nut with its slot and no overlap, both adapter cuts on the `ShaftMount` hub (slot open to the
round face, 1.5 of hub under it, the wall to the bore), access to the set screw's path and each blade nut's path,
every Regular fit, the blades' pitch and hand, the copies turned by 120 and 240 degrees, the replacement paths, and
`internal/cornercheck` on the hub (axis +Z) and each blade along its radial line.

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
| Hub front face | 20 | 46.25 |
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
(an `m2bolt.StandardLengths()` entry) and `FaceNutDepth` 1.5. The fan reads them; it does not redefine them.

### Validation

The `ClutchMount` hub passes the same rules as the `ShaftMount` hub (rules 1 to 11 above). `AdapterSites[k]` (k = 0, 1, angle
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
| Blades / adapter | Regular | 5.25 | at least 5.0 |
| Blade / ear bolt (hex key room) | Regular | 5.6 | at least 5.0 |

The ring floats 0.85 each way between the clutch's cones, and the bolted fan floats with it. Blade fits are unchanged.

Measured by `TestTurbineFanOnClutch` (mesh gap lower bounds, 605ZZ / 608ZZ clutch): hub / adapter 0.245 / 0.242; adapter /
ring 0.231 / 0.232; adapter / cover 1.495 / 1.492; adapter / the clutch bolt head 1.498 / 1.498; adapter bolt head / cover
1.497 / 1.493; each blade / adapter 5.251 / 5.252; blade 1 / ear bolt 1 26.316 / 27.078.

### Retained sharp corners, edge finishing and printing

- Hub: the same part as in `ShaftMount`: prints front face down, axis vertical, no supports. Each adapter slot roof (z 3.5)
  is a 4.35 bridge.
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
