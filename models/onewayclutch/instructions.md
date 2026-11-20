# One-way clutch instructions

A printed one-way clutch for a D-flat shaft. A hub pressed on the shaft carries three flexing pawls. They engage a ring
with 18 internal sawtooth teeth and three bolt ears, so the ring turns freely one way about the held shaft and is
driven the other way. A back plate and a cover hold the ring on the hub, clamped by three M2 bolts and nuts. Each nut
sits in a hex pocket open to the back plate's outer face. The ring is 38.2 mm across between its ears on the `605ZZ`
preset (39.8 to 40.7 mm on the 8 mm presets). The ears carry an M2 bore and a rear nut pocket each, so a separate
adapter ring (`BuildAdapter`) or any other part bolts to the front.
Every part except the bolts and nuts prints on an FDM printer with its axis vertical and no supports. The shaft is not modeled, as in `crankhandle`:
the mechanism owns it and any [Arbor collars](../../docs/model-pipeline.md#arbor-collars) that locate the clutch.

Units are millimeters. The local frame has its origin on the shaft axis in the back plate's outer face (the face toward
the mechanism), +Z runs along the axis to the cover, and the shaft's flat faces +Y. A `Placement` frame maps this
frame after each part's last operation.

## Mechanism

Compliant-pawl ratchet. Each pawl is a thin curved arm printed in one piece with the hub; it ends in a head. The heads
sit in the gaps of the ring's teeth. Turning the hub one way pushes each head against a tooth's radial lock face and
drives the ring. Turning it the other way, the teeth's ramps press the heads inward, the arms bend, and the heads click
over the teeth. The arms flex only in the XY plane, so their layers run along the arm and bending loads the printed
lines. A head leads its arm in the drive direction, so the drive load puts the arm in compression; the arm is curved
about the axis, so that pushes the head outward, deeper into the gap.

### Freewheel direction

`Freewheel` is the direction the ring turns freely about a held shaft, seen from the front (the cover, looking along
-Z). The shaft turning the same way drives the ring with it; the other way slips. One construction serves both
directions. `Direction.Sign()` is `-1` for `Clockwise` and `+1` for `CounterClockwise`. Every hub and ring sketch is
drawn in the counter-clockwise layout below with each point `(x, y)` mapped to `(σx, y)`, a mirror in the plane
`x = 0`. An angle θ of the layout maps to `θ' = 90° + σ(θ - 90°)`. `CreateArc(center, start, end)` runs
counter-clockwise, so each arc's start and end swap when `σ = -1`. The D flat and the three bolts are symmetric about
`x = 0`, so the plates, bolts and nuts are the same in both directions.

## Inputs and defaults

| Input | Default | Meaning |
| --- | --- | --- |
| `Shaft` | `605ZZ` | a preset name from the table below |
| `Freewheel` | `Clockwise` | `Clockwise` or `CounterClockwise`; zero is invalid |
| `BoreClearance` | 0.05 (`ballbearing.PressFitClearance`) | radial clearance of every D bore, on the round and the flat |
| `Placement` | nil | a frame; nil selects the local frame |

| Preset | Diameter | Flat depth | Across the flat | Source |
| --- | ---: | ---: | ---: | --- |
| `605ZZ` | 5 (`ballbearing.Lookup("605ZZ").ShaftDiameter()`) | 0.5 | 4.5 | planetary 605ZZ shafts, `crankhandle.Params605ZZ` |
| `608ZZ` | 8 | 0.75 | 7.25 | hand-crank siren shafts, `crankhandle.Params6082RS` |
| `608ZZ-gearbox` | 8 | 0.5 | 7.5 | planetary gearbox 608ZZ shaft kits |

The `605ZZ` clutch keeps the base size below. A larger bore makes the hub core, and every radius outside it, grow
(see "Derived dimensions"). The stack's z values, the fits, the bolts and the angles do not change. D bore radius
`r_b = D/2 + c`, flat at `f_b = D/2 - d + c` from the axis on the +Y side, where `c` is `BoreClearance`.

### Validation (first failure wins; every message starts `onewayclutch: `)

1. `Shaft` names a preset: `unknown shaft preset "<name>"; want one of 605ZZ, 608ZZ, 608ZZ-gearbox`.
2. `Freewheel` is `Clockwise` or `CounterClockwise`.
3. `BoreClearance` is finite and `0.05 <= c <= 0.5`.
4. The derived radii exist: the slot width converges (`onewayclutch: slot width did not converge`) and the head's flank
   leaves room for the hub's face chamfer (`onewayclutch: the head flank leaves no room for the hub's face chamfer`).
   Neither fires for a preset at any allowed `BoreClearance`; the second needs a shaft about 18 mm across.
5. `Placement` is nil or valid.
6. Each bolt site, each mount site and each face site passes `m2fastener.Site.Validate()`.

No rule limits `BoreClearance` below 0.5 any more: the core grows until the walls round the bolts hold, so every
preset builds at every allowed clearance.

`BuildParts`, `Build` and `BuildAdapter` reject a nil context and a nil document, derive, and check `ctx.Err()` before
any geometry. `BuildParts` and `Build` need a document from `NewDocument` (a chain tolerance of 0.02 mm); see "Ring".

## Fixed dimensions (exported constants)

| Constant | Value | Meaning |
| --- | ---: | --- |
| `FitMargin`; `TightFit`, `RegularFit`, `LooseFit` | 0.05; 0.25, 0.40, 0.55 | class minimum + margin |
| `Pawls`, `Teeth` | 3, 18 | all three heads engage at once; pitch 20° |
| `RootRound` | 0.8 | ring tooth root round, `HeadGap + TipRound` |
| `RingChamfer` | 0.5 | lead-in of the ring's outer loop and counterbores |
| `Ears`, `EarAngles` | 3, 90° 210° 330° | ears on the ring, the same in both directions |
| `EarRadius`, `EarRound` | 4.1, 1.5 | ear half width and end round, round where an ear side meets the outside |
| `RingHeight`, `ToothEndGap` | 21.0, 0.85 | ring height; tooth band end to the plate annulus |
| `EndStandoff` | 1.25 | `ToothEndGap + RegularFit`: how far each ring end stands past the plates' outer faces |
| `ArmThickness`, `MinArmThickness` | 1.5, 1.5 | arm thickness, at the least that prints reliably |
| `WebAngle`, `SlotAngle`, `LockAngle` | -20°, -6°, 90° | pawl 0's web back face, slot end, and ring lock face |
| `WebRound`, `TipRound`, `HeadGap` | 1.0, 0.4, 0.40 | web root round, head tip round, head to its gap (seated) |
| `HubHeight`, `Seam` | 10.0, 0.25 | hub and pawls; hub to each plate boss |
| `Recess` | 0.40 | annulus below the boss face (`RegularFit`) |
| `BackPlateThickness`, `CoverThickness` | 4.0, 4.0 | outer face to boss face |
| `PlateChamfer` | 0.5 | outer-face chamfer |
| `BoltAngles`, `BoltLength` | 30° 150° 270°, 16 | three M2 x 16 through bolts |
| `AdapterFrom`, `AdapterThickness`, `AdapterTo` | 20.0, 6.0, 26.0 | the adapter ring: one `Seam` in front of the ring, a head pocket plus 4 mm under it |
| `MountLength` | 25 | M2 x 25 adapter bolts, `m2fastener.BoltSize.StandardLengths` |
| `FaceCircle`, `FaceLength`, `FaceNutDepth` | 9.5, 10, 1.5 | the adapter's face bolts on the ear lines: circle radius, M2 x 10 (`m2fastener.BoltSize.StandardLengths`), and how far into the mating part the nut slot's near face lies |
| `PressWall` | 1.6 | least wall round a press-fit bore |

### Derived dimensions

`Derive` grows these radii from the bore radius `r_b`, so they are `Dimensions` fields, not constants. The constants
that set the rules are below. `Growth` is `ToothTip` minus its 605ZZ value, 13.75.

| Constant | Value | Meaning |
| --- | ---: | --- |
| `BaseCoreRadius` | 10.1 | least hub core radius |
| `WallMargin`, `BoltWall` | 0.05 (`FitMargin`), 1.65 | margin over `PressWall`, so a section reads 1.6 or more; modeled wall each side of a bolt hole and pocket |
| `MinSlotWidth`, `MinSlotMargin` | 1.6, 0.33 | least slot under an arm; least `SlotWidth - HeadTravel` |
| `ToothDepth`, `CounterStep`, `RingWall` | 2.35, 1.0, 2.0 | `ToothRoot - ToothTip`; `RingCounter - ToothRoot` (the 45° cone); `RingOuter - RingCounter` |
| `EarOffset`, `BossStep` | 2.5, 0.3 | `EarCircle - RingOuter`; `BossRadius - CoreRadius` |
| `MaxHubChamfer`, `HubChamferMargin` | 0.25, 0.01 | the hub's face chamfer, and its least gap to the head flank's cap |
| `AdapterShaftGap` | 1.0 | radial gap from the shaft to the adapter bore |

With `PocketReach = max(HeadPocketRadius, NutCavityRadius * sqrt(3)/2)` = 2.2 (a variable, since `NutCavityRadius` is):

```
CoreRadius    = max(BaseCoreRadius, r_b + PocketReach + HoleRadius + 2 * BoltWall)   // max(10.1, r_b + 6.8)
BoltCircle    = CoreRadius - HoleRadius - BoltWall                                   // CoreRadius - 2.95
```

The core grows only when `r_b > 3.3`, that is on the 8 mm presets. The wall from each bolt hole to the core is exactly
`BoltWall`; the wall from the bore to a head pocket is `BoltCircle - 2.2 - r_b`, which is `BoltWall` on a grown core and
more on the 605ZZ.

`SlotWidth` is the least slot that keeps `MinSlotMargin` under a fully pressed head. `HeadTravel` depends on `ToothTip`,
which depends on `SlotWidth`, so `Derive` iterates (at most 100 steps; it converges in a few):

```
s = MinSlotWidth
repeat: ArmInner = CoreRadius + s; ArmOuter = ArmInner + ArmThickness; ToothTip = ArmOuter + LooseFit
        ToothRoot = ToothTip + ToothDepth; derive the pawl values; next = max(MinSlotWidth, HeadTravel + MinSlotMargin)
        stop when |next - s| <= 1e-12
```

The rest follows:

```
RingCounter = ToothRoot + CounterStep         RingOuter = RingCounter + RingWall
EarCircle = RingOuter + EarOffset             EarReach = EarCircle + EarRadius
PlateRadius = RingCounter - RegularFit        BossRadius = CoreRadius + BossStep
AdapterRadius = EarReach                      AdapterInner = D/2 + AdapterShaftGap
HubChamfer = min(MaxHubChamfer, ArmOuter - FlankDistance - HubChamferMargin)
```

| Radius | 605ZZ, any `c` | 8 mm, `c` 0.05 | 8 mm, `c` 0.5 |
| --- | ---: | ---: | ---: |
| `CoreRadius` | 10.1 | 10.85 | 11.3 |
| `BoltCircle` | 7.15 | 7.90 | 8.35 |
| `SlotWidth` | 1.6 | 1.6275 | 1.645 |
| `ArmInner`, `ArmOuter` | 11.7, 13.2 | 12.4775, 13.9775 | 12.945, 14.445 |
| `ToothTip`, `ToothRoot` | 13.75, 16.1 | 14.5275, 16.8775 | 14.995, 17.345 |
| `RingCounter`, `RingOuter` | 17.1, 19.1 | 17.8775, 19.8775 | 18.345, 20.345 |
| `EarCircle`, `EarReach` | 21.6, 25.7 | 22.3775, 26.4775 | 22.845, 26.945 |
| `PlateRadius`, `BossRadius` | 16.7, 10.4 | 17.4775, 11.15 | 17.945, 11.6 |
| `AdapterInner`, `AdapterRadius` | 3.5, 25.7 | 5.0, 26.4775 | 5.0, 26.945 |
| `HubChamfer` | 0.25 | 0.212 | 0.190 |
| `Growth` | 0 | 0.7775 | 1.245 |

The ring is 38.2 mm across between the ears on the 605ZZ (39.76 to 40.69 mm on the 8 mm presets) and its ear tips lie
on a 51.4 mm circle (52.96 to 53.89 mm). The rest of this document gives the 605ZZ numbers, and each place that changes
with the preset says so. An 8 mm preset at `c` between 0.05 and 0.5 lies between the two columns. The `608ZZ` and
`608ZZ-gearbox` presets have the same `r_b`, so the same radii.
### Stack (z along the axis)

| Part | z from | z to | Notes |
| --- | ---: | ---: | --- |
| Back plate | 0 | 4.0 | boss face 4.0, annulus 3.6, rim cylinder 0.5 to 3.0 at r 16.7, hex pockets 0 to 2.0 open to z 0; 1.6 from the pocket floor to the annulus |
| Hub | 4.25 | 14.25 | 0.25 seams to both bosses; mid-plane 9.25 |
| Cover | 14.5 | 18.5 | boss face 14.5, annulus 14.9, rim cylinder 15.5 to 18.0, head pockets 16.5 to 18.5 |
| Ring | -1.25 | 19.75 | symmetric about z 9.25, the hub's mid-plane; stands 1.25 past both plates |
| Ring tooth band | 4.45 | 14.05 | 9.6 tall, inside the heads' 4.25 to 14.25 |
| Adapter ring | 20.0 | 26.0 | 0.25 seam to the ring's front end; ear head pockets 24.0 to 26.0, face head pockets 20.0 to 22.0 |

The plates span z 0 to 18.5 and the ring z -1.25 to 19.75, so the clutch is 21.0 mm long. The local origin stays on
the back plate's outer face. On the 605ZZ the ring is 38.2 mm across between its ears and the ear tips lie on a 51.4 mm
circle, so the clutch spans x -22.806 to 22.806 and y -19.1 to 25.7. On the 8 mm presets it spans x ±23.48 to ±23.88 and
y -19.88 to 26.95 (`-RingOuter` to `EarReach`).

## Construction

### Ring (prism, chamfer, two counterbore cuts, three nut pockets)

1. Outline sketch on z = -1.25: the outside, radius `RingOuter` (19.1), with three ears as the outer loop; as holes the 18-tooth loop
   and the three ear bores. Extrude `RingHeight` 21.0 along +Z. The extrusion is not a Boolean.
2. Chamfer the whole outer loop of both end faces by 0.5 in one call (`edgefinish.Chamfer`, `Long(Outer)`).
3. Cut two counterbore tools, revolved meridians `(r, z)`, from `Dimensions.RingBackTool()` and `RingFrontTool()`. Back:
   (`ToothTip - 1.5`, -2.25), (`RingCounter + 1.5`, -2.25), (`RingCounter`, -0.75), (`RingCounter`, `BandFrom - (RingCounter
   - ToothRoot)`), (`ToothRoot`, 4.45), (`ToothTip - 1.5`, 4.45); on the 605ZZ (12.25, -2.25), (18.6, -2.25), (17.1, -0.75),
   (17.1, 3.45), (16.1, 4.45), (12.25, 4.45). Front, the back one mirrored in z = 9.25: on the 605ZZ (12.25, 20.75),
   (18.6, 20.75), (17.1, 19.25), (17.1, 15.05), (16.1, 14.05), (12.25, 14.05). The counterbores take the
   plate rims; the 45° lead-in from the end face and the 45° cones from `RingCounter` down to `ToothRoot` hold the ring
   axially; the flats at z 4.45 and 14.05 are the teeth's end faces.
4. Cut three mount nut pockets, `m2fastener.HexPocketTool` at each mount site, `NutCavityHeight + Stock` deep from the
   site's `Cavity`, open to the rear face (z -1.25 to 0.75). The hexagon's corners lie on the 2.51 circle, with `Corner` at
   the ear angle + 30°, so a flat faces the axis: the inner flat is at `EarCircle - 2.174` (r 19.426 on the 605ZZ), 0.326
   outside `RingOuter`, so the pocket lies in the ear. The pocket keeps 1.83 to the counterbore's lead-in edge (r 17.6),
   1.59 to the ear's sides and end round (1.09 at the face after the 0.5 chamfer) and 1.34 at the section 0.25 in. The ear
   moves out with `RingOuter` by the same amount, so these walls hold on every preset. The pockets are cut after the chamfer, so
   their hexagons stay sharp. No nut is embedded: each pocket is open to a face (AGENTS.md "Fasteners").

The ring needs a document from `NewDocument()`, which carries a chain tolerance of 0.02 mm. The mount nut pockets at
the ears' rear ends meet held facets just past a plain document's gate (a held bound of 0.001328 mm against the
gate's 0.001324 mm for the third pocket, and 0.0010075 against 0.00095 for the back plate's cut). `decad.New()` makes
`Build` fail with `decad.ErrUnsupported` there (`TestOneWayClutchPlainDocumentRefusesBuild`). The ring's held bound is
0.0285 mm (volume 9845.74 ± 5.43 mm³).

The tooth loop is the whole void inside the teeth, one closed loop. Counter-clockwise layout: lock face `j` lies on the
radial axis at `φ_j = 90° + 20°j`, from `Q_j = R_root∠φ_j` in to `P_j = R_tip∠φ_j`; the ramp of tooth `j` runs from
`P_j` to `Q_{j+1}`. Each `Q_j` is rounded by `RootRound`. The tooth occupies the angles just counter-clockwise of each
lock face; the gap where a head sits lies just clockwise. On the 605ZZ the ramp line is at distance 13.338 from the axis
(`RampDistance`), at 145.940° (`RampAngle`); on the 8 mm presets 14.155 to 14.645 and 147.0° to 147.6°. The 18 teeth,
the 0.4 mm head gap and the 0.8 root round are the same at every radius, so the head is an exact 0.40 offset of its gap.

Drawing the teeth in the outline leaves the ring two Boolean cuts. A tooth cut after the counterbore cuts is refused
by decad, because it would meet the counterbore cuts' rims.

Ears: three full-height lugs at 90°, 210° and 330° (symmetric about `x = 0`, so not mirrored), each 8.2 mm wide with a
full end round of 4.1 about a point `EarCircle` (21.6) from the axis, the sides parallel to the ear's axis and tangent to the end
round. Each side meets the outside circle in a round of 1.5 (a stressed root, above half the 2.0 mm wall). An M2
clearance bore (radius 1.3) runs through each ear along the axis, 2.8 mm from its sides and end and 3.2 mm from the
counterbore. The adapter's bolts, or those of any other part, pass the bores into the rear nut pockets.

### Plates (prism, chamfer, cuts)

Profile on the plate's inner-side plane (back plate z = 0, cover z = 14.5): the circle `PlateRadius` (16.7) about the
origin, the D bore (arc of `r_b` the long way through -Y, flat at `y = f_b`), and three circles of
`m2fastener.HoleRadius` (1.3) at `BoltCircle` (7.15) on 30°, 150° and 270°. Extrude 4.0 along +Z. Then:

1. Chamfer the outer face (back plate z = 0, cover z = 18.5) by 0.5 in one call: its outer circle, plus the whole D loop
   when `BoreLoop >= 20` (the 8 mm presets; the 5 mm loop is 15.8). The bolt-hole and head-pocket loops are short.
2. Cut the inner-face tool, a revolved meridian `(r, z)`, from `Dimensions.BackPlateInner()` and `CoverInner()` with
   `b = BossRadius`, `p = PlateRadius`, `t = ToothRoot`. Back plate: (`b - 0.9`, 5.0), (`b - 0.9`, 3.4), (`b - 0.1`, 3.6),
   (`t`, 3.6), (`p + 1`, 3.6 - (`p + 1 - t`)), (`p + 1`, 5.0). Cover: (`b - 0.9`, 13.5), (`b - 0.9`, 15.1), (`b - 0.1`,
   14.9), (`t`, 14.9), (`p + 1`, 14.9 + (`p + 1 - t`)), (`p + 1`, 13.5). On the 605ZZ the back plate is (9.5, 5.0),
   (9.5, 3.4), (10.3, 3.6), (16.1, 3.6), (17.7, 2.0), (17.7, 5.0) and the cover (9.5, 13.5), (9.5, 15.1), (10.3, 14.9),
   (16.1, 14.9), (17.7, 16.5), (17.7, 13.5). It leaves the boss (`b - 0.5` at the face, a 0.4 mm 45° bevel; 9.9 on the
   605ZZ), the recessed annulus out to `ToothRoot` and a 45° cone to the rim. The boss follows the core
   (`BossRadius = CoreRadius + 0.3`), so on the 8 mm presets the boss's vertical edge lies 1.05 mm outside every bolt
   hole and no hole crosses it.
3. Back plate: cut `m2fastener.HexPocketTool` at each site (z 0 to 2.0, open to the outer face; the chamfer in step 1
   runs first, so the hex loops are never chamfered). Cover: cut `m2fastener.PocketTool` at each site (z 16.5 to
   18.5). The holes are already in the profile.

### Hub (one prism)

One sketch on z = 4.25 holds the whole outline, the D bore and the three bolt holes; extrude `HubHeight` along +Z. The
hub has no Boolean. Both faces are then chamfered by `HubChamfer` (0.25 on the 605ZZ) along their long runs (see
"Surface edge finishing").

### Hub outline, counter-clockwise layout, pawl 0 (pawl `k` is turned by 120°k)

The gap where pawl 0's head sits is bounded by the ring lock face at 90° (the line `x = 0`) and the previous tooth's
ramp, from `P = ToothTip∠70°` to `Q = ToothRoot∠90°`. The numbers below are the 605ZZ's (`P = 13.75∠70°`,
`Q = 16.1∠90°`); the 8 mm presets scale as "Derived dimensions" gives.

- Lock line: `x = HeadGap`, parallel to the 90° axis, on the gap side.
- Flank line: parallel to the ramp, `FlankDistance = RampDistance - HeadGap` = 12.938 from the axis, on the axis side.
- Apex `A` = lock ∩ flank = (0.400, 15.347). `TipRound` rounds it; its centre is (0.800, 14.594).
- `HeadReach` (outermost head radius) = 15.015; `HeadTravel` = `HeadReach - ToothTip` = 1.265.
- `LockEnd` `H_l` = lock ∩ circle `ArmInner` = (0.400, 11.693). `FlankEnd` `H_f` = flank ∩ circle `ArmOuter` = (5.078, 12.184),
  the root nearer `A`.

Outline of one period, counter-clockwise, material on the left:

1. Core arc, radius `CoreRadius`, from the slot round of pawl `k-1` to the web root round of pawl `k`.
2. Web root round, radius `WebRound`, tangent to the core arc and the web back face (a stressed root).
3. Web back face on the axis at `WebAngle`, out to `W = ArmOuter∠WebAngle`, a sharp 90° corner.
4. Arm outer arc, radius `ArmOuter`, from `W` to `H_f`.
5. Flank from `H_f` toward `A`, tip round, lock face down to `H_l` (a sharp 90° corner).
6. Arm inner arc, radius `ArmInner`, from `H_l` clockwise back to the slot round.
7. Slot-end full round, centre on the axis at `SlotAngle`, tangent to the arm inner arc (internal) and the core arc
   (external); radius `SlotWidth/2` = 0.8 (0.81 to 0.82 on the 8 mm presets) follows from the tangencies. It bulges toward the web.
8. Core arc again, on to the next pawl's web root round.

The head's lock face bears on the ring's lock face over 0.84 mm radially and 9.6 mm axially, on each of three heads. The
flexible arm runs 73.4° from the slot end to `H_f`, 15.9 mm at radius 12.6; full travel bends it to 1.12 % strain
(1.11 % from the built section; the test's cap is 1.2 %). The slot leaves `SlotWidth - HeadTravel` = 0.335 mm under a fully pressed head
(0.335 by section, 0.275 after the section's `2 off` bound, against a floor of 0.25). On the 8 mm presets `SlotWidth`
grows to keep 0.33 mm there, and the strain falls to 1.02 % (`c` 0.05) and 0.97 % (`c` 0.5). The thinnest hub walls are
2.85 to 3.30 from the bore to a bolt hole (605ZZ, `BoreClearance` 0.5 to 0.05; 2.55 on the 8 mm presets) and 1.65 from a
bolt hole to the core arc, on every preset.

### Bolts and nuts

Site `i` at angle `a_i` in {30°, 150°, 270°}: `m2fastener.Site{Head: (`BoltCircle` cos a, `BoltCircle` sin a, 18.5), Dir: (0, 0, -1),
Corner: (cos(a+30°), sin(a+30°), 0), Length: 16, Cavity: 16.5}`. The head lies flush with the cover's outer face in a 2 mm pocket.
The bolt passes the cover, the hub and the back plate. The nut sits in a hexagonal pocket (z 0 to 2.0) open to the back plate's outer face, with its corners on
`Corner` at radius 2.51 and its flats 2.174 from the bolt line, turned so that a flat faces the bore (the wall to the bore is
`BoltCircle - 2.174 - r_b`: 2.43 to 1.98 on the 605ZZ at `BoreClearance` 0.05 to 0.5, 1.68 on the 8 mm presets; the wall to the head pocket is 1.65 there); the nut sits from z 0.2 to 1.8. Two states matter. With the seams open (the model) the tip ends at z 0.5, inside the back plate, and 1.3 mm into the nut's 1.6 mm.
Tightening the bolts closes both 0.25 mm seams, so the tip moves 0.5 mm toward the back face: it ends flush with the
back plate's outer face and 0.4 mm past the nut's outer face (the nut sits on its pocket's floor), so the nut is fully
engaged and the tip never stands outside a face. The tightened state rules. The bolt holds the nut in its pocket, and
the pocket's flats stop it turning. The pocket's floor is 1.6 mm below the annulus. M2 x 16 is in `m2fastener.BoltSize.StandardLengths`.

## Bodies and containers

| Body | Count | Container | Printed |
| --- | ---: | --- | --- |
| Hub (core and three pawls) | 1 | `Hub` | yes |
| Ring | 1 | `Ring` | yes |
| Back plate, cover | 1 + 1 | `Plates` | yes |
| M2 x 16 bolt | 3 | `Bolts` | bought |
| M2 nut | 3 | `Nuts` | bought |
| Adapter ring (`BuildAdapter` only) | 1 | `Adapter` | yes |
| M2 x 25 mount bolt (`BuildAdapter` only) | 3 | `Bolts` | bought |
| M2 nut, mount (`BuildAdapter` only) | 3 | `Nuts` | bought |

`Build` makes ten bodies in five containers, each a child of the document root, filed with `doc.Root().ChildNamed` and
`AddBody`. `BuildAdapter` adds seven bodies: the adapter in its own `Adapter` container, and its bolts and nuts beside
the clutch's in `Bolts` and `Nuts`, which then hold two lengths (the turbine fan's `Bolts` mix lengths the same way).
A document with both holds 17 bodies in six containers. The body set does not change with any parameter. There is no
"Look-alike parts" section, so no part marks: the adapter (a 51.4 mm annulus on the 605ZZ, 6 mm thick, three head pockets) does not
look like the 33.4 mm plates or the ring (605ZZ sizes).

## Adapter ring

`BuildAdapter(ctx, doc, params)` builds the standard adapter that bolts to the ring's front: a disc with a bore (`2 * AdapterInner`: 7 mm on the 605ZZ, 10 mm on the 8 mm presets) and
three flush bolt heads. Another adapter uses the same ear interface: a flat back face at
`AdapterFrom`, `AdapterThickness` thick at each ear, a clearance hole and an `m2fastener.PocketTool` at each
`Dimensions.MountSites` entry. Its front interface for a part bolted to its face is `Dimensions.FaceSites`: three M2
clearance holes with a head pocket each in the adapter's rear face, and the mating part's back face one `Seam` in front
(z 26.25).

1. Sketch on z = 20.0: the circles `AdapterRadius` (25.7) and `AdapterInner` (3.5) about the origin, and a circle of
   `m2fastener.HoleRadius` on each ear's axis at `EarCircle`, one on the same axis at `FaceCircle` and one over each of the
   clutch's own bolt heads (`boltHoles`, at `BoltCircle` on the bolt axes): a polar reference, an axis per angle, the centre
   on the axis at a distance. Select the profile with ten holes (the bore, three ear holes, three face holes and three access
   holes). Extrude 6.0 along +Z.
2. Chamfer both faces' long loops by 0.5 in one `edgefinish.Chamfer` call. Each face is about 1960 mm² on the 605ZZ
   (outer circle 157.7 mm, bore 22.0 mm; every hole is a short run) and larger on the 8 mm presets.
3. Cut `m2fastener.PocketTool` at each mount site (z 24.0 to 26.0) then at each face site (z 20.0 to 22.0). The holes are in
   the profile.

Each mount site is `m2fastener.Site{Head: (`EarCircle` cos a, `EarCircle` sin a, 26.0), Dir: (0, 0, -1), Corner: (cos(a+30°),
sin(a+30°), 0), Length: 25, Cavity: 25.25}` for the ear angles `a`. Each face site is `Head: (9.5 cos a, 9.5 sin a, 20.0), Dir:
(0, 0, 1), Corner: (cos a, sin a, 0), Length: 10, Cavity: AdapterThickness + Seam + FaceNutDepth = 7.75`; the nut slot of
the mating part (its slot plane at z 27.75, 1.5 inside a part whose back face is at 26.25) is that part's to cut. Derive
checks each face site with `Site.Validate`, after the mount sites. Access hole centres sit over the clutch's bolt heads
(`BoltCircle` at 30°, 150° and 270°). Measured in sections of the built adapters: on the 605ZZ (7.15) an access hole is
2.35 mm from the bore (1.86 at the rear face, after the bore's chamfer) and 5.07 mm from the face pockets. On the 8 mm
presets the bore wall is 1.60 (`c` 0.05) to 2.05 mm (`c` 0.5), 1.11 to 1.56 at the rear face, and the face pocket wall
5.31 to 5.48 mm.

| Check | Seams open (model) | Seam closed (tightened) |
| --- | --- | --- |
| Head top | flush with the adapter's front face, z 26.0 | same |
| Face bolt head top (a mating part's bolt) | flush with the adapter's rear face, z 20.0 | same |
| Material under the head pocket | 4.0 | same |
| Shank in the adapter hole and the ear bore | 0.3 radial | same |
| Bolt tip | z -1.0, 0.25 inside the ring's rear face | z -1.25, flush with it |
| Nut | z -1.05 to 0.55, 1.55 of 1.6 engaged | pulled onto the pocket floor, -0.85 to 0.75; tip 0.4 past it |

The bolt must span the adapter, the 0.25 seam and the 21.0 ear and end within 0.25 of the ear's rear face. M2 x 20 would
need a 1.0 mm adapter, under the 3.0 that a head pocket plus 1 mm needs, and M2 x 25 needs 6.0. No nut is embedded: each
sits in a hex pocket open to the ring's rear face, and the bolt holds it once engaged. The adapter prints front face down,
so the head pockets open on the bed. It has no inside corner in a section. The head-pocket floor rims lie between layers
and are retained, with the face pockets' floor rims, which open upward when printed front face down. The adapter's outer
edge is 1.9 mm from each ear head pocket (1.4 after the chamfer), and every pocket keeps 1.5 mm of wall to the nearest other
outline (1.9 measured). The adapter's back face and the ring's front end are a clamped seam of 0.25, as in the turbine fan.
The adapter turns with the ring, and its bore (`AdapterInner`, the shaft's radius plus `AdapterShaftGap`) clears either shaft by 1.0 radially: 7 mm for the 5 mm shaft, 10 mm for the 8 mm shaft; the shaft ends
before the part bolted to the face. The cover lies 1.5 mm below it (the 1.25 standoff plus the 0.25 seam).

## Part fits

| ID | Parts | Class (min) | Modeled | Measured, proven lower bound | Measure |
| --- | --- | --- | ---: | ---: | --- |
| P1 | hub D bore / shaft | press fit | `c` (0.05) | 0.049 (at `c` 0.35: 0.349) | section at mid-height |
| P2 | back plate, cover D bores / shaft | press fit | `c` | 0.049 (at `c` 0.35: 0.349) | as P1 |
| T1 | hub / each plate boss | Tight 0.2 | 0.25 | 0.234 | mesh gap, placed meshes |
| R1 | pawls / plate annuli | Regular 0.35 | 0.65 | 0.638 | axial section, hub bounds |
| R2 | ring counterbores / plate rims | Regular | 0.40 radial | 0.382 | section mid-overlap (z 1.75 and 16.75); the rims overlap the counterbores over z 0.5 to 3.0 and 15.5 to 18.0 |
| R3 | ring cones / plate cones | Regular | 0.60 normal | 0.578 | axial section |
| R4 | ring tooth ends / plate annuli | Regular | 0.85 axial | 0.827 | axial section |
| R6 | ring end planes / every shaft-side body (plates, bolts, nuts, hub) | Regular | 1.25 | 0.40 with the ring at the end of its 0.85 play, 0.90 with the seams closed | bounds, and the ring meshes shifted past the play |
| R5 | each head / its tooth gap, seated | Regular | 0.40 | 0.372 | `meshgap` on the placed meshes (0.379 on the 605ZZ, 0.372 on the 8 mm presets, both directions) |
| L1 | arm outer arcs / tooth tips | Loose 0.5 | 0.55 | 0.530 | section at mid-height |
| F | bolts, nuts | fastener rules | head pocket 0.3, hole 0.3, nut 0.17 at each flat, 0.20 past each corner and to the floor; tip 0.5 inside the back face with seams open, flush when closed | | bounds, sections and no overlap |
| A | adapter / ring front end | clamped seam | 0.25 | 0.231 | mesh gap |

The model is **Mixed**: the bore clearance is a parameter, and the bolts and nuts keep their nominal size. The ring's
ears and outside fit in the user's driven part or housing belong to the consumer. `TestOneWayClutchFits` measures P1 to R5 and L1 for every
preset in both directions, for `BoreClearance` 0.35 on `605ZZ`, and the mesh gaps for a placed default. Each section is cut at the finest chord its body holds (0.01 for an exact part and for the
chamfered hub, whose held mesh bound is 0.006; 0.03 for the ring). Every radius comes from `Dimensions`, so the fits hold on the grown 8 mm clutch as on the 605ZZ. The measured column is the least value over those cases, less each section's
or mesh's proven displacement bound, so it sits a little under the modeled value. A D bore's corners stand more than `c` from the shaft's corners, since `c` is radial
on the round and on the flat. In the head-on-pocket-floor and bolt-in-nut pairs the meshes touch by design (nominal
pair).

`TestOneWayClutchPawls` checks the arm (1.5 mm thick, the enforced minimum; 1.12 % strain at full travel on the 605ZZ,
1.02 to 0.97 % on the 8 mm presets), the slot (1.6 mm on the 605ZZ, leaving 0.335 mm under a fully pressed head; 0.33 mm
on the 8 mm presets), the web (1.83 mm to the slot end's round on the 605ZZ, 2.00 mm (`c` 0.05) to 2.08 mm (`c` 0.5) on
the 8 mm presets, from the sections, wider than the arm) and the lock face (0.844 mm radial contact on the 605ZZ, 0.885
from the sections; 0.877 (`c` 0.05) to 0.895 mm (`c` 0.5) on the 8 mm presets, 0.907 from the sections at `c` 0.05;
9.6 mm axial), from `Dimensions` and from sections of the built hub. From the sections the strain is 1.11 % on the
605ZZ and 1.01 % on the 8 mm presets at `c` 0.05.

## Sketch constraints

Every sketch is solved, then `sketchcheck.VerifyFullyConstrained` before its profile is used. Use geometric
constraints; place computed points with `sketchcheck.DimensionPoint` only in meridians.

- Polar axes: a horizontal construction reference line from the origin, and one construction axis per angle, a line from
  the origin with `NewAngle(ref, axis, θ')` and `NewDistance(origin, end, 20)`, using the mapped angle.
- Hub sketch: three axes per pawl (web, slot, lock) and one per bolt. Arcs `CreateArc(origin, ...)` with `NewRadius`.
  Web back face: `NewParallel(webAxis, web)` and `W` on the axis. Lock line: `NewParallel(lockAxis, lock)` and
  `NewDistancePointLine(origin, lock, 0.40)`. Flank: `NewAngle(ref, flank, RampAngle')`, `NewDistancePointLine(origin,
  flank, FlankDistance)`. Rounds: `sketch.CreateFillet` for the web root (core arc, web line) and the tip (flank, lock). Slot
  round: centre `NewPointOnLine` on the slot axis, `NewTangentCircles` with the arm inner arc (internal) and the next
  core arc (external); no radius constraint. The D bore and bolt holes follow the plate profile. A fillet removes its
  corner point and the constraints on it, so each line is held by constraints on the line and its far end.
- Ring outline sketch: three construction axes at the ear angles with `NewAngle(ref, axis, e_k)`; each ear centre
  `NewPointOnLine` on its axis with `NewDistance(origin, centre, EarCircle)`; the end round `NewRadius(4.1)`; each side
  `NewParallel(axis, side)` and `NewTangent(side, endRound)`; the three outside arcs `NewRadius(RingOuter)` about the origin;
  the bores `NewRadius(1.3)` sharing the ear centres; then `CreateFillet` at all six side-to-outside corners, 1.5. The
  tooth loop is drawn in the same sketch (next item). Select the profile with four holes.
- Ring tooth loop: 18 axes at `φ_j`. `P_j` is `NewPointOnLine` on axis `j` with `NewDistance(origin, P_j, R_tip)`. The
  lock line `Q_j → P_j` is `NewParallel` to axis `j`. The ramp `P_j → Q_{j+1}` is `NewDistancePointLine(origin, ramp,
  RampDistance)`. `CreateFillet` at every `Q_j`, 0.8. One closed loop, a hole of the ring's outline.
- Bolt holes: each circle's centre is `NewPointOnLine` on its bolt axis with `NewDistance(origin, centre, BoltCircle)`.
- Meridians (ring counterbore tools, plate inner tools): the plane holds the axis, U along +Z and V radial; every vertex
  `DimensionPoint`; revolve with `decad.FullRevolution{}`.
- Plate profile: the hub's bore and holes, plus `NewRadius(PlateRadius)` on a circle about the origin.
- Fastener tools: `m2fastener` returns their sketches; they are recorded in `Parts.Sketches`.

## Surface edge finishing

| Part | Face | Net area | Long runs | Action |
| --- | --- | ---: | --- | --- |
| Back plate | outer face | 807.6 to 811.1 (605ZZ), 860.3 to 901.5 (8 mm, from the larger `PlateRadius`) | outer circle 104.9 (605ZZ), 109.8 to 112.8 (8 mm); D loop 25.1 to 25.3 at 8 mm; hex loops 15.06, short | chamfer 0.5 (one call) |
| Cover | outer face | as above | as above; head pockets 13.8, short | chamfer 0.5 (one call) |
| Plates | annulus | 481 | outer edge meets the 45° cone | the 1.1 mm cone is that edge's finish |
| Plates | boss face | about 270 | none | small, none |
| Hub, 605ZZ | each face | 406.1 (`c` 0.5) to 413.9 (`c` 0.05) | per pawl: arm outer arc 20.13, inner run; head run short; D loop 15.8, short | chamfer 0.25 on the long runs; head run sharp (below) |
| Hub, 8 mm | each face | 441.0 (`608ZZ-gearbox`, `c` 0.05) to 466.7 (`608ZZ`, `c` 0.5) | per pawl: arm outer arc, inner run; head run short; D loop 25.1 to 25.3 at `c` 0.05, 28.0 at `c` 0.5 | chamfer `HubChamfer` (0.212 to 0.190) on the long runs and the whole D loop; head run sharp |
| Ring | each end face | 357.4 front, 324.2 rear | outer loop; counterbore rim; ear bores and rear nut pockets, short | chamfer the outer loop 0.5 as a lead-in (kept; the pockets are cut after it); the counterbore rim by the tool's lead-in |
| Adapter | each face | about 1960 (605ZZ) | outer circle 157.7; bore 22.0 (605ZZ), 31.4 (8 mm); holes short | chamfer 0.5 (one call) |

Every hub's two faces are over the 400 mm² limit: 406.1 to 413.9 mm² on the 605ZZ and 441.0 to 466.7 mm² on the grown 8
mm hubs, measured in the mid-height section. Each takes a chamfer along the runs of 20 mm or more, and the head run stays sharp so the head keeps its 0.4 tip
round and its lock contact. `hub()` calls `edgefinish.ChamferRuns` with `HubChamfer` and keeps the exact prism only if it
returns `edgefinish.ErrNoEdge`, which no preset does (`TestOneWayClutchEdgeFinishing` checks every preset).
`edgefinish.ChamferRuns` chooses the edges on the extruded prism,
then chamfers in two decad calls: the partial runs (the arm's outer arc and the inner run of every pawl, 36 edges on both
faces), then the whole D loops, of which the 605ZZ has none that are long and the 8 mm hubs have. Each run ends at a convex corner (the web, the
lock face's root) or at the concave 11.43° corner beside the head's flank, where the bevel tapers out on the flank's
plane. The result is faceted, so its checks are sections 0.1 mm inside each face, not volumes.

The chamfer is `HubChamfer`, at most 0.25 mm and not the standard 0.5, and this is the documented smaller chamfer that
"Surface edge finishing" allows where functional geometry requires it. The flank's plane, carried back over the arm's
outer arc, lies `FlankDistance` from the axis (12.938 mm on the 605ZZ), and the arc is at `ArmOuter` (13.2), so the plane
can cap a bevel of at most `ArmOuter - FlankDistance` = 0.262 mm at that corner. `HubChamfer` stays `HubChamferMargin`
(0.01) under the cap: 0.25 on the 605ZZ (the cap is above it), 0.212 on the 8 mm presets at `c` 0.05 (cap 0.222) and
0.190 at `c` 0.5 (cap 0.200). A 0.5 mm bevel has no cap there. The cap falls about 0.05 mm per mm of growth, so a shaft
about 18 mm across would leave no room, and `Derive` rejects it. The bevel tapers out along the flank's plane over 0.911
mm on the 605ZZ, 0.739 mm (`c` 0.05) to 0.619 mm (`c` 0.5) on the 8 mm presets, in the section 0.1 mm inside a face
(`TestOneWayClutchEdgeFinishing`).

## Corners and roots

All four printed parts print with their axis vertical. Rounded as the rule asks: web roots 1.0, slot ends full round
0.8, the plate boss step as a 45° bevel, every cone-to-cylinder or cone-to-flat turn at 45°. The arm (1.5) meets the
bending-part minimum; every other wall is 0.8 or more. The hub, back plate and cover are press-fit rings with walls of
1.6 or more round the bore.

Retained sharp corners. `TestOneWayClutchCorners` reads sections at every layer run of each part, for every preset
and both directions, and exempts the hex pocket corners only (the back plate's at the bolts, the ring's at the mount sites), each exemption matching a finding in every case. `TestOneWayClutchAdapter` reads the adapter with no exemption. Each corner below is kept sharp by design. It
either passes the rule's limits (inside turn up to 50°, tip turn up to 120°) or lies between layers, where no section
reads it.

| Part | Corner | Reason | Test reading |
| --- | --- | --- | --- |
| Hub | each head's tip (0.4 round) and lock-face edges | Engagement corners of a pawl tooth. The 51.4° tip is blunted by the rule's minimum 0.4 round. | passes the tip limit |
| Hub, back plate, cover | the two corners of the D bore | Mating corners of the shaft's flat. | passes the inside limit (turn 28.8° to 36.5°) |
| Ring | the 18 tooth roots, 0.8 round | A ratchet tooth root, rounded below half its section: 0.8 is the largest round that keeps the head's 0.40 gap. | passes the inside limit |
| Ring | the 18 tooth tips | Engagement corners. | passes the tip limit |
| Cover | the floor rims of the three head pockets (between layers) | The 0.3 mm head clearance leaves no room. | between layers, not read |
| Back plate | the six corners of each of the three hex pockets (18), edges vertical | Mating corners: they match the M2 nut's sharp corners, as in `models/crankhandle`. | read in the z 0 to 2.0 run: 60° inside turns; exempt with `cornercheck.Annulus` about each site, radius 2.51 ± 0.3 |
| Back plate | the ceiling rim of each hex pocket at z 2.0 | A 45° chamfer of 0.5 mm would fill the corner the nut's corners sit in. | between layers, not read |
| Ring | the six corners of each of the three mount nut pockets (18), edges vertical | Mating corners: they match the M2 nut's sharp corners, as the back plate's do. | read in the z -1.25 to 0.75 run: 60° inside turns; exempt with `cornercheck.Annulus` about each mount site, radius 2.51 ± 0.3 |
| Ring | the ceiling rim of each mount nut pocket at z 0.75 | As the back plate's. | between layers, not read |
| Ring | the six ear root rounds (1.5), ear end rounds and bores | Rounded as the rule asks: 1.5 is above half the 2.0 mm wall. | passes |
| Adapter | the head pocket floor rims, front and rear (between layers) | The 0.3 mm head clearance leaves no room. | between layers, not read |

The test also checks each press-fit ring's wall: in every layer run, the hub's, back plate's and cover's bore lies
1.6 mm or more from every other outline.

Not applicable: Split shafts (no printed shaft or pin), Gear train clearances (no gears), Arbor collars (the shaft is
the mechanism's; a collar bears on a plate's face and stays inside 32 mm across on the 605ZZ and 33.5 mm on the 8 mm presets, inside the plate face's flat, which ends at
2 * (`PlateRadius` - `PlateChamfer`): 32.4 mm on the 605ZZ and 34.0 to 34.9 mm on the 8 mm presets, so it never reaches the ring).

## Printing and assembly

- Material: PETG is recommended for the hub, since its arms flex at every click; PLA works at 1.12 % strain (605ZZ; less on the 8 mm presets).
- Hub and ring: either face down, no supports. The ears are vertical prisms with no overhang. The ring's tooth ends overhang 2.5 mm at the lower end; the 0.85 mm gap
  to the plate absorbs a little droop. The mount nut pockets open on the ring's rear face, on the bed or on top, and each
  pocket's ceiling bridges at most 5.02 mm round the bore, as the back plate's does.
- Back plate: outer face down, no pause. The hex pockets open on the bed; each pocket's ceiling bridges at most 5.02 mm
  around the 2.6 mm bolt hole.
- Cover: outer face down; the head pockets open on the bed.
- Adapter ring: front face down; the head pockets open on the bed.

Assembly. The nuts are not held until their bolts reach them, so assemble off the shaft with the bolts loose:

1. Push the three M2 x 16 bolts through the cover's head pockets. Stand the cover on its outer face, bolts up.
2. Slide the hub over the bolts onto the cover.
3. Push each pawl head in and slide the ring over the hub, turning it the way it turns freely.
4. Lay the back plate on, inner face down, with the bolts through its holes.
5. Drop an M2 nut into each hex pocket. Lift the stack, holding the plates together, and turn each bolt from the
   cover's side until it catches its nut. Leave them loose.
6. Press the clutch onto the shaft, back plate first, its outer face toward the mechanism. The loose bolts let the three
   D bores line up on the shaft.
7. Tighten the three bolts from the front. The pockets stop the nuts turning.

Adapter assembly, off the shaft, before the clutch is pressed on (the nuts drop into the ring's rear face, which
faces the mechanism):

1. Stand the adapter front face down and push the three M2 x 25 bolts up through its head pockets.
2. Lower the clutch onto it, ring front end down, with the bolts through the ear bores.
3. Drop a nut into each rear-face pocket and turn each bolt from the front until it catches. Then tighten.
