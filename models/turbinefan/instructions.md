# Turbine fan

## Purpose and source

Construct a fan of three replaceable blades on one printed hub. The hub pinches a D-flat shaft the way the crank
handle's arm does ([`models/crankhandle`](../crankhandle/)): a blind D bore, a 1 mm slit from the bore out through a clamp
tab, and one M2 clamp bolt across the tab. Each blade is the hand-crank fan's blade
([`models/handcrankfan`](../handcrankfan/)), with its round root stud that slides into a radial socket in the hub and one
M2 bolt that runs parallel to the shaft through the hub and the stud into a captive nut. Undoing that bolt frees the
blade. The parts are:

- a hub, a straight prism with a clamp tab, cut by the bore, the slit, three sockets, and the bolt pockets and cavities;
- three blades, one part printed three times;
- four M2 x 16 bolts and four M2 nuts: one bolt and nut for each blade, and the clamp bolt and its nut.

The shaft is not modeled. It belongs to the mechanism, and its diameter equals the bore of the purchased bearing it runs
in ([Part clearances](../../docs/model-pipeline.md#part-clearances)). The presets read that bore from
[`models/ballbearing`](../ballbearing/). The bolts and nuts are [`models/m2bolt`](../m2bolt/) and
[`models/m2nut`](../m2nut/), placed with `internal/m2fastener`.

All dimensions are in millimeters; angles are radians in code and degrees in prose. The model reuses two other items by
import and copies no geometry: `crankhandle` exports the D bore, slit and nut pocket tools and the tail of the clamp
tab, and `handcrankfan` exports `BuildBlade` and `BuildBladeMirrored`, the blade socket tool and the hub and blade-joint constants.

## Symbols

| Symbol | Meaning | Formula |
| --- | --- | --- |
| `D`, `d`, `c` | shaft diameter, flat depth, bore clearance | `ShaftDiameter`, `FlatDepth`, `BoreClearance` |
| `r_b` | bore radius | `D/2 + c` |
| `f_b` | distance from the axis to the bore's flat, toward +Y | `D/2 - d + c` |
| `L_b` | length of the bore's outline | `2 r_b (pi - acos(f_b / r_b)) + 2 sqrt(r_b^2 - f_b^2)` |
| `R`, `W`, `rho` | hub radius, tab half width, tab root round | `HubRadius` 14, `TabHalfWidth` 9, `TabRootR` 1 |
| `F` | right root round's centre | `(W + rho, -sqrt((R + rho)^2 - (W + rho)^2))` |
| `P` | where the right root round meets the disc | `R F / (R + rho)` |
| `y_r` | where the root round meets the straight side | `F.y` |
| `y_c` | the clamp bolt's axis | `y_r - ClampSideRun` |
| `y_s` | where the tab's sides stop being straight | `y_c - ClampSideRun` |
| `R_e` | the tail's end arc's reach behind the axis | `crankhandle.TailReach(W, y_s)` |
| `u_k` | blade k's radial direction | `(cos t_k, sin t_k, 0)`, `t_k = 90 + 120 (k - 1)` degrees |

## API

Package: `models/turbinefan`. Exported surface:

- The fixed dimensions as constants: `BladeCount` (3), `HubRadius` (14), `HubLength` (20), `HubMid` (10), `FirstBladeAngle`
  (90 degrees), `TabHalfWidth` (9), `TabRootR` (1), `ClampBolt` (16), and the values it takes from `crankhandle`:
  `BoreFloor`, `ChamferSize`, `PressWall`, `SlitWidth`, `SlitInBore`, `ClampSideRun`, `ClampCornerR`, `ClampEndR`,
  `NutPocketRound` and `RegularFit`. `BladeAngle(k)` is blade k's angle about +Z.
- The container names `HubContainerName` (`Hub`), `BladesContainerName` (`Blades`), `BoltsContainerName` (`Bolts`) and
  `NutsContainerName` (`Nuts`).
- `Params` with `ShaftDiameter`, `FlatDepth`, `BoreClearance`, `BladePitch`, `Hand` and `Placement *r3.Frame` (nil selects
  the local frame). `Hand` is `RightHand` (the zero value) or `LeftHand`.
- `Params605ZZ()`, `Params608ZZ()` and `DefaultParams()`, which returns `Params605ZZ()`.
- `Params.Validate() error` and `Params.Derive() (Dimensions, error)`. `Dimensions` holds `BoreRadius`, `BoreFlat`,
  `BoreLoop`, `RootCenter`, `RootDisc`, `RootSide`, `ClampY`, `SideEnd`, `Reach`, `TailCorner`, `TailEnd`, `ClampSite`
  and `BladeSites`, in the local frame.
- `NewDocument(opts ...decad.DocumentOption)` returns `decad.New(WithChainTolerance(0.1 mm), opts...)`. The blade's stud
  union refuses a plain document; build in such a document.
- `BuildParts(ctx, doc, p) (*Parts, error)` adds the hub and the three blades and puts them in no container. `Parts`
  holds `Hub`, `Blades`, every `Sketches`, `Dimensions`, and in the world frame `BladeSites` and `ClampSite`.
- `Build(ctx, doc, p) (*Model, error)` adds the parts, the bolts and the nuts, each in its container. `Model` embeds
  `Parts` and holds `BladeBolts`, `BladeNuts`, `ClampBolt` and `ClampNut`.

The exports this model needs from other items:

- `crankhandle.BoreTool`, `SlitTool` and `NutPocketTool` build the cutters; `TailCenters`, `TailReach` and `SketchTail`
  build the tab's tail. Each is the code `crankhandle` ran before, moved, with the `run` receiver replaced by arguments.
- `handcrankfan.BuildBlade(ctx, doc, pitch)` returns the unplaced blade and its sketches, and `BuildBladeMirrored` the
  blade of the other hand (its sections unflipped and its pitch turn reversed); `BladeSocketTool(ctx, doc, f)`
  builds a socket cutter; `HubRadius`, `HubLength`, `BladeBoltAt`, `BladeBoltLength`, `BladeBoltCavity`, the stud and
  socket constants, `DefaultBladePitch`, `MinBladePitch`, `MaxBladePitch`, `BladeRootRadius()`, `BladeTipRadius()` and
  `Beta(r, pitch)`.

## Presets and defaults

| Field | `Params605ZZ` (default) | `Params608ZZ` |
| --- | --- | --- |
| `ShaftDiameter` | 5, `ballbearing.Params605ZZ().BoreDiameter` | 8, `ballbearing.Params608ZZ().BoreDiameter` |
| `FlatDepth` | 0.5, the planetary gearbox shafts' flat | 0.75, the siren shafts' flat |
| `BoreClearance` | 0.05, `ballbearing.PressFitClearance` | 0.05 |
| `BladePitch` | 25 degrees, `handcrankfan.DefaultBladePitch` | 25 degrees |
| `Hand` | `RightHand` | `RightHand` |

Derived: `r_b` 2.55 and 4.05, `f_b` 2.05 and 3.30, `L_b` 15.8 and 25.1, `F` (10, -11.1803), `P` (9.3333, -10.4350), `y_r`
-11.1803, `y_c` -14.7803, `y_s` -18.3803, `R_e` 23.4688 for both. A 608-2RS has the same bore as a 608ZZ.

## Validation

Every message starts with `turbinefan: `. Derive checks in this order and returns the first failure:

1. `ShaftDiameter`, `FlatDepth`, `BoreClearance` and `BladePitch` are finite and positive: `<Field> must be finite and
   positive`.
2. `d <= D/4`.
3. `ballbearing.PressFitClearance <= c <= 0.5`.
4. `BladeSocketStart - r_b >= PressWall`, so 1.6 mm stays between the bore and the socket floors at r 5.7: `r_b` at most
   4.1. At the default clearance this allows `D` up to 8.1; at `D` = 8 it allows `c` up to 0.1.
5. `BladePitch` lies in [10 degrees, 40 degrees].
6. `Hand` is `RightHand` or `LeftHand`: `Hand must be RightHand or LeftHand`.
7. `Placement` is nil or a valid frame.
8. `ClampSite.Validate()` and each `BladeSites[k].Validate()`, and the clamp site's `Tip()` equals its `CavityFar()`, so
   the bolt's tip ends flush with the -X side face. These checks are constant today; they stop a constant edit passing
   silently.

BuildParts and Build reject a nil context and a nil document, derive the dimensions, and check `ctx.Err()` before any
geometry. A rejected build adds no body.

## Local frame and hand

The origin is on the shaft axis in the hub's back face, the face toward the mechanism. +Z runs along the shaft away from
the mechanism to the front face at z = 20. The bore's flat faces +Y; the slit and the clamp tab point to -Y. Blade k lies
along `u_k`: blade 1 along +Y, blade 2 at 210 degrees and blade 3 at 330 degrees, counter-clockwise seen from the front.
A `Placement` maps this frame onto the given one after each part's last operation.

Hand: a right-hand rotor blows air forward, along +Z and away from the mechanism, when the shaft turns clockwise seen from
the front. A left-hand rotor does the same when the shaft turns counter-clockwise: its blades are the right-hand blades
reflected in the plane x = 0 (`handcrankfan.BuildBladeMirrored`), which maps blade 1 to itself and swaps blades 2 and 3.
The hub, bolts and nuts do not depend on the hand. For the right hand, a blade's chord from its trailing edge to its leading edge at radius r runs along
`(cos beta, 0, -sin beta)` with `beta = handcrankfan.Beta(r, BladePitch)`: the leading edge leads toward +X at the top
and sits upstream (-Z), and the concave pressure face faces the front.

## Geometry

1. **Hub outline.** A sketch on z = 0: the disc of radius `R` about the axis, with the clamp tab below it. The disc is
   an arc from `P` over +Y to its mirror. Each root is an arc of radius `rho` about `F` (or its mirror) from `P` to the
   side's top `(W, y_r)`, tangent to the disc externally and to the vertical side. The sides run at x = ±`W` down to
   y = `y_s`, where `crankhandle.SketchTail` adds the corner arcs of radius `ClampCornerR` and the end arc of radius
   `ClampEndR`, every join tangent. Three circles of radius 1.3 at `BladeBoltAt` (9.5) along `u_k` are the blade bolts'
   holes. The roots are concave, so the outline has the only inside corners the tab gives, rounded by `rho`.
2. **Hub body.** Extrude 20 along +Z, then chamfer the outer loop of both faces by `ChamferSize`. Then cut, in this
   order:
   - the slit: `crankhandle.SlitTool` on z = -1, from `y = -R_e - 2` up to `y = -r_b + SlitInBore`, through the hub (21
     deep + 1);
   - the blind D bore: `crankhandle.BoreTool` on z = -1, `r_b` and `f_b`, to z = 19, so the front face keeps a
     `BoreFloor` floor;
   - the three sockets: `handcrankfan.BladeSocketTool` on the frame with origin `(0, 5.7, 10)`, U = +Z and V = +X
     (normal +Y), and its two copies from `decad.PatternCopies` with a circular pattern of three about +Z;
   - for each blade site, `m2fastener.PocketTool` (head pocket, z 18 to 20, radius 2.2) and `m2fastener.CavityTool` (the
     round nut cavity, radius 2.51, z 2.4 to 4.4). The holes are the outline's circles, through the hub;
   - the clamp: `crankhandle.NutPocketTool(ClampSite)` (the hexagonal nut pocket open to the -X face, x -7 to -9, corners
     along +-Z), `m2fastener.PocketTool(ClampSite)` (x 9 to 7) and `m2fastener.HoleTool(ClampSite)` (x 10 to -10,
     crossing the slit).

   The slit goes before the bore, as in the crank handle. All cuts are in the local frame; `placement.Place` runs last.
3. **Blades.** `handcrankfan.BuildBlade(ctx, doc, BladePitch)` (`BuildBladeMirrored` for `LeftHand`) returns blade 1 in its own frame (origin on the fan axis in
   the hub's mid plane, +Z its radial line, +Y downstream, the blade moving toward -X). `decad.Transform` by the frame
   with origin `(0, 0, 10)`, U = (-1, 0, 0) and V = (0, 0, 1) turns it into place: blade-local Z becomes +Y, local Y
   becomes +Z and local X becomes -X. `decad.PatternCopies` with the hub's pattern gives blades 2 and 3. Do not
   transform blade 1 again before copying.
4. **Blade facts.** From handcrankfan: a loft from r 14.4 to 64 through ten stations; a stud of radius 3 from r 6.1 to 16
   with a 0.3 chamfer on its inner end; a bolt hole of radius 1.3 along the fan axis at r 9.5 through the stud; sections
   that are NACA 4-digit mean lines blended with an ellipse at the root; a blade angle `Beta(r, BladePitch)` that is
   constant inboard of r 21.5; a trailing edge 1.0 thick. The rotor is 128 across. Every blade vertex lies between z 0
   and 20: the measured range is z 5.02 to 17.72 at the default pitch, 4.17 to 19.39 at 40 degrees and 5.85 to 15.55 at
   10 degrees.
5. **Fasteners.** See "Fasteners" below.
6. **Containers.** `Build` files the hub, the three blades, the four bolts and the four nuts in their containers, each a
   child of the document root.

## Layout numbers

| Item | Value |
| --- | --- |
| Hub envelope | x [-14, 14], y [-23.47, 14], z [0, 20] |
| Tab side faces | flat from y -11.18 to -18.38 (7.2 long) at x = +-9 |
| Clamp head pocket ring on the +X face | y -16.98 to -12.58: 1.4 of flat face beyond it on each side |
| Clamp nut pocket flats on the -X face | y `y_c` +- 2.17: 1.43 of face beyond each |
| Clamp nut pocket corners to the faces | z 7.49 to 12.51: 7.49 to each face |
| Clamp hole to the bore | 10.9 (605ZZ), 9.4 (608ZZ) |
| Socket mouths of blades 2 and 3 | 210 and 330 degrees, each +-14.06 degrees on the disc |
| Tab roots meet the disc | 228.19 and 311.81 degrees: 4.13 degrees (1.01 mm of disc arc) beyond each socket mouth |
| Wall from the bore to the socket floors (r 5.7) | 3.15 (605ZZ), 1.65 (608ZZ), 2.85 (`regular-bore`) |
| Wall from the bore's flat to blade 1's socket floor | 3.65 (605ZZ), 2.40 (608ZZ) |
| Blade nut cavity to the outline, to the back face, to the socket | 1.99, 2.4, 2.2 |
| Blade head pocket to the outline | 2.3 |

The tab roots are rounded by 1.0, below the 4.25 the stressed-root rule asks for the 8.5 mm jaw each side of the slit. A
round of 4.25 would meet the disc at 316.55 degrees, inside blade 3's socket mouth (to 315.95 degrees), so the socket
would cut the round at a near tangent. 1.0 keeps 1.01 mm of plain disc between them.

## Fasteners

All four bolts are M2 x 16 (`m2bolt`), all four nuts `m2nut`, placed with `m2fastener.PlaceBolt` and `PlaceNut` at the
placed sites. M2 x 14 is not a standard length; M2 x 18 would stand 2 mm proud of the 18 mm tab.

| Bolt | Head | Hole | Nut | Rule |
| --- | --- | --- | --- | --- |
| Blade bolt k | flush in the front face (z 20), pocket radius 2.2, 2 deep | radius 1.3 through the hub and the stud | round cavity radius 2.51, z 2.4 to 4.4, enclosed by the hub: drop it in during a print pause; the nut is at z 2.6 to 4.2 and the tip at z 2.0, inside the hole | enclosed nut |
| Clamp bolt | flush in the tab's +X face (x 9), pocket 2 deep | radius 1.3, x 10 to -10, across the slit | hexagonal pocket open to the -X face (x -7 to -9), corners on +-Z, floor round 0.6; the tip ends flush at x -9 | pinch-clamp exception (AGENTS.md "Fasteners") |

The blade bolt's site is handcrankfan's, turned about the shaft axis for blades 2 and 3: head at `BladeBoltAt` from the axis
along the blade's radial line at z 20, `Dir` -Z, `Corner` the radial line, `Length` 16 and `Cavity` 15.6. The clamp site is
`m2fastener.Site{Head: (9, y_c, 10), Dir: (-1, 0, 0), Corner: (0, 0, 1), Length: 16, Cavity: 16}`: crankhandle's clamp
with a 16 mm bolt across an 18 mm tab, so `Cavity` and the tip meet on the -X face.

The clamp nut's hexagon has its corners on the circle of radius 2.51 (0.2 past the nut's corners) with a corner up and one
down, so the flats are the +Y and -Y sides, 4.35 across. The nut's corners (radius 2.31) cannot pass the flats (2.17), so
the nut cannot turn. The pocket's roof is two 30 degree slopes that bridge, as in the crank handle.

## Replaceable blades

- To remove a blade, undo its bolt from the front face and pull the blade out along its radial line. The stud leaves the
  socket after 7.9 mm (the stud's inner end at r 6.1 to the hub's rim at r 14). The nut stays in its enclosed cavity,
  because the 2.6 mm hole is narrower than the nut's 4.0 mm across flats. The hub can stay on the shaft, since every
  blade bolt's head is on the front face.
- A left-hand blade is a different part from a right-hand one: a fan takes three of one hand.
- To fit a blade, slide the stud into the socket with the concave face toward the front face, push the bolt in from the
  front through the stud into the nut and tighten it. All three blades are one part, and any blade fits any socket.
- The stud is round, so a blade also goes in turned half a turn about its radial line. That blade faces the wrong way.
  The README tells the builder which way it goes; the model has no key.

## Part fits

The model is **Mixed**: the bore fit is adjustable and the blade fits, bolts and nuts keep their nominal size.

| Fit | Class | Built clearance |
| --- | --- | --- |
| D bore round the shaft | Press fit, the hub turns with the shaft | `c` = 0.05 radial, adjustable 0.05 to 0.5 (validation rule 4) |
| Stud in its socket | Regular | 0.40 radial (3.4 against 3.0) |
| Stud's inner end to the socket floor | Regular | 0.40 (6.1 against 5.7), with the 0.3 stud chamfer clear of the 0.5 floor round |
| Blade root to the hub's rim | Regular | 0.40 (14.4 against 14) |
| Blades 2 and 3 to the tab's roots and sides | Regular | at least 1.3 (the least measured at z 10 over the three pitch cases is 1.33) |
| Bolts, nuts | Fastener rules | head pocket 0.3, hole 0.3, cavity 0.2 |
| Clamp nut in its hexagonal pocket | Fastener rules | 0.17 at each flat, 0.2 past each corner, 0.2 to the floor |
| Bolt shank in the stud's hole | Fastener rules | 0.3 |
| Bolt in its own nut | nominal exception | not measured |

The hub presses onto the shaft, and the clamp bolt closes the slit to pinch the bore on it. Nothing holds it along the
shaft but that grip. The shaft should reach past z 11.3, the far edge of the clamp bolt's hole, for a full grip. The
bore is blind with a 1 mm floor at the front face.

## Valid range

- `ShaftDiameter` up to 8.1 at the default clearance; `FlatDepth` up to `D/4`; `BoreClearance` 0.05 to 0.5, and for the
  8 mm shaft 0.05 to 0.1.
- `BladePitch` 10 to 40 degrees.

## Representative cases

- `605ZZ`: the default.
- `608ZZ`: `Params608ZZ`, the largest bore, 1.65 of wall to the socket floors.
- `regular-bore`: `Params605ZZ` with `BoreClearance` 0.35.
- `steep`: `Params608ZZ` with `BladePitch` 40 degrees.
- `flat`: `Params605ZZ` with `BladePitch` 10 degrees.
- Left hand: `Params605ZZ` with `Hand` `LeftHand`. Its test proves the mirror relation only: each blade's volume and bounds
  equal the right-hand fan's mirrored blade reflected in x = 0. The full per-blade checks run for the right hand.
- Placed: the frame with origin `(10, -5, 7)`, U `(0, 1, 0)` and V `(0, 0, 1)`.

## Retained sharp corners and edges

The rule is "Corners and roots" in `docs/model-pipeline.md`. The hub prints front face down with its axis vertical, and
each blade prints standing on its stud end with its radial line vertical.

| Part | Corner or edge | Reason |
| --- | --- | --- |
| Hub | The two corners of the D bore | Mating corners of the shaft's flat. They turn about 36 degrees, so they need no round. |
| Hub | The bore's rim in the back face, at 608ZZ (`L_b` 25.1 mm, opened by the slit) | Not chamfered: Decad refuses the chamfer (crankhandle `steps.md`, "Known decad limit"). |
| Hub | The slit's outline in both faces | The 1 mm clamp gap: a 0.5 mm chamfer each side would open it to 2 mm. |
| Hub | The clamp bolt's head pocket floor rim, 2 mm into the +X side face | The 0.3 mm head clearance leaves no room to round it. |
| Hub | The six corners of the clamp nut's hexagonal pocket | Mating corners: they match the sharp corners of the M2 nut. They run parallel to the bed, so no layer section reads them. |
| Hub | The clamp pocket's 0.6 floor round, where a layer cuts it at a slant | The pocket's slopes lie 30 degrees off the bed, so a layer chords the round into turns of 51.6 to 55.2 degrees; the test exempts that band only. |
| Hub | The blade head pockets' floor rims, and the nut cavities' floors and roofs | Between layers: the Fasteners rule sizes them. |
| Hub | The socket mouths | Between layers, out of the checker's sections. |
| Hub | The socket floor's 0.5 round, where a layer off the socket's axis cuts it at a slant | A layer at z 7 or 13 chords the round into turns of about 58 degrees on the floor's outline, 5.7 to 6.6 from the shaft axis; the test exempts that band only. |
| Hub | The slit's R0.5 end round in the bore's floor | It is the full round of half the slit's width that "Corners and roots" asks for. The held boolean mesh chords it coarsely enough that the check reads 50.4 degrees, in every bore case. |
| Blade | The four corners where the bolt hole crosses the stud, at r 9.5 and 10.2 | The material angle is 64 to 90 degrees; the checker adds the stud's curvature (handcrankfan). |
| Blade | The step where the stud meets the root face (r 14.4), opening downward when printed stud end down | Between layers; inherited from handcrankfan, which does not list it. |

Rounds below half their section: the tab's two roots (1.0 on the 8.5 mm jaw) and the socket floor round (0.5 on the 3.4
socket, handcrankfan's).

Press-fit wall: at least 1.6 round the bore everywhere, 1.65 at the 608ZZ socket floors. The hub prints with its bore
axis vertical.

Surface edge finishing: both faces are large (about 850 mm2), so the outer loop of each is chamfered by 0.5 before the
cuts. The tab's side faces are 144 mm2, not large. The inner loops on the faces are short runs: the bolt holes (8.2
long), the head pockets (13.8 long) and the 605ZZ bore rim (15.8); the 608ZZ bore rim is listed above. The blades have
no large planar face.

Body containers: `Hub`, `Blades`, `Bolts` and `Nuts`. Look-alike part marks: none; the three blades are identical copies
and the four bolts are one bought part. Split shafts: the model has no printed shaft, arbor, axle or pin; the blade's
stud is part of the blade. Arbor collars: no printed shaft either. The consuming mechanism's shaft carries the hub and
owns the collar between the hub's back face and its bearing or plate. Gear train clearances: no gears.

## Printing

| Part | Prints with | Pause |
| --- | --- | --- |
| Hub | front face down, axis vertical | at 17.6 mm above the bed (the nut cavities' roofs): drop an M2 nut in each of the three cavities |
| Blade (three) | standing on the stud end, radial line vertical | none |

The hub's sockets (6.8 across), the clamp hole and the head pocket are horizontal holes small enough to print without
support; the clamp nut pocket's roof is two 30 degree slopes that bridge. The bore opens upward from its floor on the
bed.

To assemble:

1. Fit each blade and its bolt (see "Replaceable blades").
2. Press the hub onto the shaft, flat to flat.
3. Set an M2 nut in the hexagonal pocket in the tab's -X face, push the clamp bolt in from the +X face and tighten it.

## Checks and images

Check every case above: every sketch fully constrained (25 from `BuildParts` and 20 from `Build`), the twelve bodies and
their containers, the default's bounds and `VerifyAll` meshes, the placed fan against the local one, the blade envelope
(every blade vertex at 0 <= z <= 20 and r <= 64.5), the bore's clearance round the nominal shaft, its floor and wall, the
slit joining the bore to the end arc and ending in the floor, the face chamfers round the whole outline, the clamp bolt and
every wall round the clamp, each blade bolt's head, tip and nut with the hub enclosing the nut and no overlap, every Regular
fit, the blades' pitch and hand, the copies turned by 120 and 240 degrees, the replacement paths, and
`internal/cornercheck` on the hub (axis +Z) and each blade along its radial line.
