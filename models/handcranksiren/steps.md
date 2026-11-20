# Hand-crank siren steps

Each step maps a requirement of `instructions.md` to a Sketch or Decad operation. Every siren sketch is solved, verified
fully constrained and recorded in `Assembly.Sketches`. All dimensions are in `instructions.md`; this file gives the
operations, their order and the checks. "Rework from the spur-train siren" at the end lists what the change deletes,
what it regenerates and how to split it into pull requests.

## 1. Frames, parameters and helpers

- `params.go`: `Ratio` (6), `minSlots` 12, `maxSlots` 40, `Params`, `DefaultParams` (`{4π/3, 34}`), `Validate`,
  `PitchHz`, `State` (`CrankAngle`, `RotorAngle`, `GearboxInput`, `Axis`, `ArmDirection`), `Motion`. `axisHeight` 84
  replaces `rotorHeight`; `crankOffset`, `stage1Dist` and `stage2Dist` go.
- `document.go`: `NewDocument` returns `planetarygearbox.NewDocument(opts...)`.
- `fits.go`: unchanged (`tightFit` 0.25, `regularFit` 0.40, `looseFit` 0.55, `seatClearance` 0.05). Add
  `pressFit` 0.05 for the D bores and the spacer faces.
- `build.go`: `Build` validates, builds the gearbox, then every siren part at crank angle 0, poses each by its motion,
  records it and returns every sketch. `part(name, group, motion, create)` poses through `motionTransform(s, name,
  motion)`:
  - `fixed`: identity.
  - `crank`: rotation about A along −Y by `CrankAngle`.
  - `rotor`: rotation about A along −Y by `RotorAngle`.
  - `gearbox`: `gearboxMotion.Transform(name, s.GearboxInput)`, where `gearboxMotion` is
    `planetarygearbox.NewMotion(GearboxParams(), planetarygearbox.CarrierDriven)`, made once per build. A body the
    motion reports fixed keeps the identity.
- `sections.go`: keep `frame`, `section`, `profile`, `extrude`, `revolve`, `cylinder`, `box`, `cut`, `union`,
  `roundedPolygon`, `roundedPrism`, `meridianPolygon`, `port`, `dShape`, `dBoreTool`. `dBoreTool(axis, from, length,
  clearance)` takes the planetary flat: a cylinder of radius 4.0 + clearance minus a box from 3.5 + clearance toward +Z.
  Delete `dShaft`.
- Check: `TestValidate`, `TestMotion`, `TestPitch`, `TestRatio`.

## 2. Gearbox (`gearbox.go`, new content)

1. `GearboxParams()`: `planetarygearbox.DefaultParamsFor("planet608-shaft608")`, then `InputShaftLength` 24.05,
   `OutputShaftLength` 22.05 and `Placement` = `frame((0, 0, 84), U (0, 0, 1), V (1, 0, 0))`. It returns
   `DefaultParamsFor`'s error wrapped as `handcranksiren: gearbox parameters: %w`; `Motion` and `Build` return it.
2. `gearbox()`: `planetarygearbox.BuildParts(b.ctx, b.doc, GearboxParams())`. Fail on its error.
3. For each body of `box.EarBolts`, `b.doc.Remove(body)`; then set `box.EarBolts = nil`.
4. For each `Part` of `box.Parts()` whose name is not an `EarBoltName`, record `Part{Name: part.Name, Group: "Gearbox/" +
   part.Container, Motion: "gearbox", Body: part.Body}` posed through `motionTransform`. Keep `box` in
   `Assembly.Gearbox`.
5. Fail unless each ear nut's bounds lie in Y [1.0, 2.6] about its ear's bore line, so a change of the gearbox's ear
   cavity cannot pass silently.
- Check: `TestGearboxParams`; the `gearbox pose` and `parts` subtests.

## 3. Housing (`housing.go`)

`ground` is `shaftAxis{(0, 0, 0)}`, so `ground.across(y)` has U = +X, V = −Z and normal +Y, and a point (X, Z) is
(u, v) = (X, −Z). `A` is `shaftAxis{(0, 0, 84)}`, and a post's axis is `shaftAxis{(x, 0, z)}`.

1. housing-front: `section(ground.across(−11.25))`, `roundedPolygon` of {(−68, 0), (−68, −152), (107, −152), (107, 0)}
   with radii 13, counter-clockwise in (u, v), and `circle(s, 0, −84, 14.55)`; `profile` with 1 hole;
   `extrude(s, 1, 10)`.
2. Cut the recess: `cylinder(A.across(−5.30), 44.1, 5.05)` (to Y −0.25, 1 past the face).
3. `edgefinish.Chamfer(body, +Y, {−11.25, −1.25, −5.30}, Long(All), 0.5)`.
4. Posts: for each (x, z) of (±52, 32), (±52, 136), `cylinder(shaftAxis{(x, 0, z)}.across(−2.25), 5, 28.25)`;
   `union` the plate and the four posts.
5. Round each post root with `decad.Fillet(body, Edges(Circular(), EndpointAt(seam)).Exactly(1), 5)`, the seam found
   as the planetary gearbox finds its boss root (`circleSeam` at Y −1.25, radius 5 about the post's axis). Then round
   the recess floor's outer corner (radius 44.1 about A at Y −5.30) by 0.5 the same way. Require a solid after each.
6. `fastenerCuts` for every site whose head host or nut host is housing-front, in the site order of section 9.
7. housing-rear: `section(ground.across(26.25))`, the outline {(−68, 0), (−68, −152), (68, −152), (68, 0)}, radii 13,
   counter-clockwise in (u, v), extruded 8.25.
   Cut `cylinder(A.across(25.25), 9, 10.25)` (the lip bore) and `cylinder(A.across(27.25), 11.05, 8.25)` (the seat,
   1 past the +Y face). `edgefinish.Chamfer(body, +Y, {26.25, 34.50}, Long(All), 0.5)`. Then `fastenerCuts` for each
   `housing-bolt-*` (pocket and hole).
- Check: rows T1, T13, L8, L9, S1, S2 of `clearances`; `fasteners`; `edges`; `corners`.

## 4. Bearings and spacers (`bearings.go`, `spacers.go`)

- `ballbearing.Build` with `ballbearing.Params608ZZ()` and Placement origin at the bearing's −Y face on A, normal +Y:
  `bearing-rotor` at Y −22.50 and `bearing-crank` at Y 27.50. Append each `Model.Sketch`. Group `Bearings`, motion
  `fixed`.
- spacer: built once about A at Y 0, then translated. `cylinder(A.across(0), 5.75, 3.5)` minus
  `dBoreTool(A, −1, 5.5, 0.05)`. Place one copy with its −Y face at Y −26.25 (`rotor-spacer`, motion `rotor`) and one
  at Y 34.75 (`crank-spacer`, motion `crank`), both in `Spacers`.
- Check: the `bearings` and `collars` subtests; rows T4, T5, T7, T9, R1.

## 5. Siren head (`drum.go`, `rotor.go`, `stator.go`, `collector.go`, `horn.go`)

Meridian sketches of the drum and rotor sit on origin (0, 0, 84), U = +X, V = +Y, normal +Z, and revolve about
`ConstructionAxis{(0, 0, 84), (0, 1, 0)}`. Every Y constant of the earlier head moves by −9.80 except as listed here.

- `drum.go`: revolve the drum `meridianPolygon` of the 18 vertices in `instructions.md`. Round the hub root (circle of
  radius 13.55 at Y −21.30) by 1.25 with `decad.Fillet`. Cut three windows (`port(21, 31, 40°, round 1)` on the plane Y
  −26.30, U at 60°, extruded 6, `Patterned` Count 3, then `Cut`). `edgefinish.Chamfer(body, +Y, {−21.30, −25.30},
  windows only (Long(Inner), more than one edge), 0.5)`. Union three posts (radius 5 at radius 30, Y [−21.30, −11.50],
  patterned at 0°, 120°, 240°). Fastener cuts for drum-bolt (pocket and hole) and collector-bolt.
- `rotor.go`: disc radius 45 over [−37.30, −34.30]; chamfer both cap outer loops 0.5 before any cut; vanes (rectangle
  u [`rv`, 44], v [−1, 1], Y [−35.30, −26.30]) patterned Count N, with `vaneRadius(N) = max(11, 0.51·N)`; hub radius 9
  over [−35.30, −26.30]; `Union(vanes, hub)` then `Union(disc, that)`; ports `port(30, 42, π/(2N), round 1)` at Y
  −38.30, extruded 5, patterned Count N, cut; D bore `dBoreTool(A, −35.30, 10, 0.05)`.
- `stator.go`: disc radius 58 over [−41.45, −38.45]; cap outer loops chamfered 0.5; the same ports; a hole for each
  collector-bolt.
- `collector.go`: the eight steps of the earlier collector with every Y moved by −9.80 (`rotor.across(−106.70)`, bore
  `cylinder(rotor.across(−50.70), 44, 10)`, cavity `roundedPrism(rotor.across(−101.70), …, 52)`, `finish(body, −41.70,
  −106.70)`, ceiling chamfer at −49.70, throat on `frame((−45, throatY, 84), …)` with `throatY` −75.70).
- `horn.go`: unchanged in its own frame, with O = (−50, −75.70, 84).
- Check: the `ports` and `outlet` subtests; rows T2, T3, T11, T5, T6, R2, L1, L6, L10; the `vanes` subtest.

## 6. Crank (`crank.go`)

- `crankParams()`: `crankhandle.Params6082RS()` with `FlatDepth = GearboxParams().FlatDepth` and the placement frame
  origin (0, 38.30, 84), U (−1, 0, 0), V (0, 0, 1). `armFace` is 38.30 and `pinAxis` (0, 0, 139).
- Build it with `crankhandle.BuildParts` once. Fail unless its `Dimensions.BoreRadius` is 4.05 and `BoreFlat` is 3.55,
  the output shaft's D grown by 0.05. The crank is built with `DoubleFlat` false here, because the output shaft has one
  flat, and the longer hub (17) is the migration's: it lengthens the output shaft by 9.0 and decides on a second flat. Record its sketches with the siren's, and return its `Arm`, `Pin` and `Handle` as
  crank-arm, crank-pin and crank-handle, motion `crank`, group `Crank`.
- `crankSites` carries crankhandle's own sites from `crankParams().Derive()` through the placement, as before. After
  `BuildParts`, fail unless they equal the placed `Parts.Site`, `Parts.CrossSites` and `Parts.ClampSites` (two sites, `crank-clamp-bolt-1..2`). Then
  `hardware` for each `crank-` site.
- Check: the `sweep` and `split shafts` subtests; rows T5, T6, T8, R3, R4, L5. Sections of the arm use 0.0102.

## 7. Grip (`grip.go`)

1. `section(frame((79, 0, 0), U (0,1,0), V (0,0,1)))`, then `roundedPolygon` of the outer outline
   {(−85.25, 5), (−11.50, 5), (−11.50, 125), (−85.25, 125)} with radii 10 and of the window
   {(−55.25, 12), (−23.25, 12), (−23.25, 118), (−55.25, 118)} with radii 3, both counter-clockwise in (u, v).
2. `extrude(s, 1, 26)`.
3. `edgefinish.Chamfer(body, +X, {79, 105}, Long(All), 2.0)`.
4. Pocket and hole for each grip-bolt. housing-front cuts the hole and hex pocket of each grip-bolt (section 3, step 6).
- Check: rows T12, L7; the `grip`, `knuckle room` and `window height` subtests.

## 8. Containers (`containers.go`)

`moveBodies` runs last. For each part it splits `Group` on `/` and walks `doc.Root().ChildNamed` one level at a time,
so `Gearbox/Planet Gears` lands in `Planet Gears` under `Gearbox`, and no container is created empty. Delete
`GearPairsContainerName`, `PairContainerName`, `ShaftsContainerName` and `pairOf`; add `SpacersContainerName` and
`GearboxContainerName`.
- Check: the `containers` subtest.

## 9. Fasteners (`fasteners.go`)

`fastenerSite` wraps `m2fastener.Site` with names, hosts, motion and a nut seat: `cavity` (round, closed), `hex`
(open to the face at `CavityFar`), `slot` (open to the post's side along `Corner`) or `none` (the ear bolts, whose nut
is the gearbox's). `hardware(site)` adds the bolt, then the nut unless the seat is `none`. `fastenerCuts` cuts, per
host:

- head host: `m2fastener.PocketTool` and `HoleTool`;
- nut host, `cavity`: `HoleTool` and `CavityTool`;
- nut host, `hex`: `HoleTool` and `HexPocketTool(site, NutCavityHeight+Stock)`;
- nut host, `slot`: `HoleTool`, `HexPocketTool(site, NutCavityHeight)` and a slot box on `site.Across(site.Cavity)`
  from u 0 to u 7 (2 past the post's side), v ±2.1737 (the hexagon's flats), extruded `NutCavityHeight` along Dir.
  Union the box and the pocket, then cut. The box's sketch is a rectangle anchored to the origin by its v sides'
  distances and its u 0 side through the origin, fully constrained.

Site order: ear, housing, drum, collector, throat, grip, crank-pin, crank-cross, crank-clamp (1 then 2). `Fasteners(p)` in
`export_test.go` poses every site.

Why no embedded nut: every new siren nut has an open seat. The housing posts take a slot because both of the posts'
ends are covered (housing-front's own plate and housing-rear), so the slot opens to the post's side instead. The ear
bolts use the gearbox's embedded nut: an open hex pocket in the ear's rear face would need an M2 × 22 bolt through the
10 mm plate, which is not a standard length, and would change the planetary gearbox. The collector and throat joints do
not change, so they keep their cavities (AGENTS.md "Fasteners": models built before the rule keep them until their
fasteners change).
- Check: the `fasteners` subtest.

## 10. Checks

One test per rule. `TestHandCrankSirenDefault` builds the default once and runs the subtests below.
`TestHandCrankSirenVariant` runs `slots_12` (CrankAngle 0) and `slots_40` (CrankAngle 4π/3).

| Rule | Test | What it requires |
| --- | --- | --- |
| Parameters | `TestValidate` | Slots 11 and 41 rejected, 12 and 40 accepted; non-finite CrankAngle rejected |
| Motion | `TestMotion` | RotorAngle 6·CrankAngle, GearboxInput −CrankAngle, ArmDirection, Axis (0, 0, 84) |
| Pitch | `TestPitch` | 408, 144 and 480 Hz |
| Ratio | `TestRatio` | `Ratio` equals the planetary CarrierDriven motion's `Ratio()` within 1e-12 |
| Gearbox parameters | `TestGearboxParams` | the kit's bearings and teeth, Li 24.05, Lo 22.05, the placement, `Validate` nil |
| Parts | `.../parts` | 105 parts with unique names, groups and motions as listed; no live gearbox ear bolt; `Gearbox.EarBolts` nil |
| Sketch constraints | `.../sketches` | every recorded sketch `FullyConstrained`, DOF 0, no redundant or conflicting constraint |
| Body containers | `.../containers` | the container names, each body in its container, `doc.VerifyContainers().Err()` nil |
| Motion through the gearbox | `.../gearbox pose` | each rotor part's transform maps 3 test points as the Input Shaft's does, each crank part's as the Output Shaft's, within 1e-9; the ports line up at the default |
| Part clearances | `.../clearances` | every Fits row's proven lower bound meets its class |
| Bearing shafts and seats | `.../bearings` | T4 sections touch on the round side; S1 seat gap proven ≥ 0.02; S2 0.25 off the floor |
| Arbor collars | `.../collars` | each collar of the table is 3.0 or longer between its end faces, 11.5 across (under 12), and keeps its listed gaps |
| Fasteners | `.../fasteners` | each head flush, each nut inside its cavity, pocket or slot, each tip at or inside a face, no fastener overlaps its parts, each ear bolt passes its ear nut |
| Ports and air path | `.../ports`, `.../outlet` | `Slots` ports; open at the default; the outlet sections moved 9.80 toward −Y |
| Grip | `.../grip`, `.../knuckle room`, `.../window height` | the hand box and window box empty, 25 mm round the bar's inner front edge, window ≥ 100 at the bar face |
| Crank sweep | `.../sweep` | the crank at 16 angles overlaps no fixed part and keeps L5 |
| Orientation | `.../orientation` | horn frontmost, grip left, crank right, only the housing plates reach Z 0 |
| Solidity | `.../tessellate` | `VerifyAll` on every body |
| Surface edge finishing | `.../edges` | the Edge finishing table, the spacers not qualifying |
| No overlap | `.../overlap` | no two parts overlap, the gearbox's bodies included |
| Press-fit walls | `.../press-fit walls` | rotor hub 4.95, spacers 1.7, crank arm 1.6 (1.05 web at each clamp hole), each ring printed axis vertical |
| Corners and roots | `.../corners` | `cornercheck` on every siren printed part, only the listed corners |
| Split shafts | `.../split shafts` | `splitcheck.Check` and `cornercheck.CheckAt` on the posed Input Shaft, Output Shaft and crank pin with the listed normals |
| Vanes | `TestHandCrankSirenVariant/.../vanes`, `TestVaneRadius` | `rv` by the formula; ≥ 1.2 between neighbouring vanes at 12, 22, 34 and 40 slots |

The variants also run `parts`, `sketches`, `containers`, `ports`, `outlet`, `orientation`, `overlap`, `corners`, `split
shafts` and rows R2 and L1.

## Rework from the spur-train siren

### Deleted

- Files: `gears.go`, `shafts.go`, `mesh_test.go`, `marks_test.go`. `gearbox.go` and `bearings.go` are rewritten.
- Constants and fields: `crankOffset`, `stage1Dist`, `stage2Dist`, `State.IntermediateAngle`, `State.Intermediate`,
  `State.Rotor`, `State.Crank`, the `intermediate` motion, `gearboxBoltLength`, `gearboxCavity`, `frontFace`,
  `rearFace`, `BearingSites`, `GearPairsContainerName`, `PairContainerName`, `ShaftsContainerName`, `pairOf`, `dShaft`
  and the `spurgear` and `gearroot` imports. `drumCavity`, `gripCavity`, `gripHeadY`, `drumHead`, `collectorBoltHead`,
  `armFace` and `pinAxis` take the new values.
- Parts: gearbox-front, gearbox-frame, gearbox-rear, the four gears, the three siren shafts, the six 608-2RS bearings,
  the gearbox-front, gearbox-rear and old drum and grip bolts and nuts.

### Images and docs to regenerate

All in the same pull request as the geometry (AGENTS.md "Model images"):

- `models/handcranksiren/images/`: `hero.png`, `index.png`, `exploded.png`, `cutaway.png`, `view-rear.png`,
  `view-side.png`, `view-top.png`, and a new `exploded-gearbox.png` (the planetary gearbox's `Set` and `Explode` on the
  siren's gearbox, so its inner parts show).
- `images/parts/`: delete `crank-gear`, `intermediate-pinion`, `intermediate-gear`, `rotor-pinion`, `crank-shaft`,
  `intermediate-shaft`, `rotor-shaft`, `gearbox-front`, `gearbox-frame`, `gearbox-rear` and the old `gearbox-bolt`. Add
  `housing-front`, `housing-rear`, `spacer`, `ear-bolt`, `housing-bolt`, and the gearbox's `ring`, `cover`, `sun`,
  `planet`, `input-shaft`, `output-shaft`, `carrier`, `carrier-pin`, `collar`, `gearbox-bolt` (M2 × 6) and `ear-nut`.
  Regenerate the rest (`drum`, `collector`, `stator`, `rotor`, `horn`, `grip`, `bearing` as a 608ZZ, the crank parts and
  the remaining bolts).
- `models/handcranksiren/README.md`: the summary, How it works (ratio 6, 34 slots, 408 Hz), the holding and
  fastener text, the print table, the parts table and the exploded alt text.
- `models/README.md`: the siren row's description and its index image.
- `docs/models/handcranksiren.md`, the siren paragraph of `docs/rendering.md`, and `tools/render/handcranksiren.go`
  (part list, groups, exploded offsets) with `handcranksiren_views_test.go`.
- `models/crankhandle` (README, instructions, `params.go` and `crankhandle.go` comments, `docs/models/crankhandle.md`):
  the siren now builds `Params6082RS()` with a 0.5 flat. The text only; no crankhandle geometry or image changes.
- `scripts/ci/run_tests.py`, `scripts/ci/test_run_tests.py` and `.github/workflows/repository.yml`: the `slots24` case
  becomes `slots40`, and the solid count in the comment becomes 105.
- The build status stays **Not built yet**; `UNBUILT` keeps `handcranksiren`.

### Pull requests

No planetary gearbox change comes first: the ears, the shafts and the collar rings are used as they are.

1. **Spike.** Add `GearboxParams()` and `TestGearboxParams`, and a test that builds the gearbox with it in a
   `planetarygearbox.NewDocument()` document, then builds the current rotor and crank in the same document, recording
   the build time. Switch `NewDocument` to the 0.1 mm chain tolerance and run the current siren suite once to see whether
   the rotor, crank and fit sections still pass at it. If any step fails, stop and report the failing case.
2. **Gearbox swap.** Sections 1 to 9 except the slot range: Ratio 6, the gearbox, the housing, the bearings and spacers,
   the head moved 9.80, the drum's hub and windows, the crank on the output shaft, the grip, the fasteners, the
   containers and every check of section 10, with `Slots` still [12, 24], 20 by default and CrankAngle 5π/4 (the rotor
   angle 7.5π still lines the ports up for 12, 20 and 24). Regenerate every image and doc listed above. The default
   pitch reads 240 Hz in this pull request.
3. **Slot range.** `Slots` [12, 40], default 34, CrankAngle 4π/3, `vaneRadius`, the `vanes` checks, the `slots40` CI
   case, and the images and docs again (rotor, stator, hero and the pitch text).

Each pull request runs the focused siren tests locally and leaves the full suite to CI.
