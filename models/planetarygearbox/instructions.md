# Planetary gearbox construction instructions

Use this contract to derive [steps.md](steps.md), then generate the Go package and tests.
This file owns geometry, dimensions, input rules and case inputs. The steps own
interfaces, Sketch/Decad calls and acceptance assertions.

## Inputs

Build a compact planetary gearbox. A fixed internal ring gear is the housing. In each stage
a sun gear drives `n` planet gears that roll inside the ring, and the planets' pins ride on a
carrier. The sun is the input and the carrier is the output. A stage count above one stacks
identical stages in the same ring: each carrier drives the next stage's sun through a
link shaft, and the last carrier drives the output shaft.

Every part that turns on another part rides on a purchased ball bearing from the
[bearing catalog](../ballbearing/instructions.md#catalog). Two roles pick a bearing: the planet bearing, and the shaft
bearing. Both are 605ZZ by default. The planets turn on
their pins, so each planet holds one planet bearing in a pocket and the bearing's inner race sits on
the pin. The input shaft and the output shaft turn in the housing, so each rides in a shaft bearing in an
end cover. The sun turns with the input shaft and each later sun with the carrier before it, so
they hold no bearing. The ring and the covers are the fixed housing.

The housing is the ring gear and two end covers. The covers are open frames, a rim, spokes and a
bearing hub, so they use little material. Eight M2 bolts and eight M2 nuts hold the housing together, four
bolts per cover, built from `models/m2bolt` and `models/m2nut`. Each cover's four bolts have 6 mm length (the shortest
standard length), so the bolt length does not depend on the stage count. A bolt's head sits in a round pocket in the
cover's outer face and ends level with that face. Its nut sits in a closed round cavity in the ring wall, 1 mm behind the
ring's end face, so the ring holds the nut and it cannot fall out; the bolt hole is the cavity's only opening. The nut
is dropped in during a print pause. The bolt passes through the cover's remaining 1 mm, the ring's 1 mm roof and the nut,
and its tip stays inside the ring's bolt hole. This follows the repository fastener rules (see `AGENTS.md`, Fasteners).

The ring carries mounting ears so the gearbox can be fixed in place. The ears are tabs on the ring's outside, each with a
bore through it parallel to the gearbox axis. The bore diameter is an input, because the model does not know what the
user passes through it: a bolt, a pin or a shaft. The model adds no bolt, head pocket or nut cavity for an ear.

The input and the output connect as follows. The shafts are separate bodies from the carriers, so each part
prints in its own best orientation. The input shaft is a rod as wide as the shaft bearing's bore (5 mm by default): it passes through the front
cover's bearing and fills the first sun's bore. A link shaft runs through each carrier's hole and the next sun's bore, which is
how stages link. The output shaft runs through the last carrier's hole and the rear cover's bearing. Every shaft
is as wide as the shaft bearing's bore (5 mm for the nominal 605ZZ) where it passes through a bearing or a sun, so it
has no clearance against the bearing. Each carrier pin is as wide as the planet bearing's bore. The input shaft is a round cylinder with a flat `Sf` deep on the +X side, which gives it a D
section where the first sun sits on it. Where a bearing sits it is a full cylinder. The link and output shafts are
thin-thick: a thin rod of the shaft diameter, and a thicker part of the shaft collar diameter with the same flat, a D section, that is
pressed through the carrier's centre hole. The sun's bore and the carrier's hole are D-shaped to match, so the input
shaft turns the first sun, and each link or output shaft turns its carrier and, for a link shaft, the next sun,
without slipping. The flat also lets a coupling grip the input and output shaft ends. The contract does not model the
coupling, a set screw or the motor.

The input shaft carries a collar, a round bulge wider than the shaft, between the front bearing's inner race and the
first sun. A collar is the printed stand-in for a washer. It keeps the turning part clear of the bearing and the
cover. The carrier pins, the link shaft and the output shaft have no collar. Each has a D thick part instead, pressed
through the carrier: the pin collar diameter for a pin and the shaft collar diameter for a shaft, and its step keeps the planet bearing, the next sun or the rear
bearing clear of the plate.

Lengths are millimeters and angles are radians. The gearbox is anchored by one `r3.Frame`:
its origin is a point on the gearbox axis, `U × V` is the axis direction, and the first
stage's gears start on the reference plane. A nil `Placement` is the world XY frame at the
origin, so the axis is world +Z.

| Symbol | Parameter | Type | Default | Meaning |
| --- | --- | --- | --- | --- |
| `m` | `Module` | float64 | 1 | Tooth module of all three gears |
| `α` | `PressureAngle` | float64 | `20π/180` | Pressure angle of all three gears |
| `Zs` | `SunTeeth` | int | 13 | Sun tooth count |
| `Zp` | `PlanetTeeth` | int | 29 | Planet tooth count |
| `Zr` | `RingTeeth` | int | 71 | Ring tooth count |
| `n` | `PlanetCount` | int | 3 | Planets per stage |
| `K` | `Stages` | int | 1 | Stacked stages in the one ring |
| `S` | `Samples` | int | 15 | Fit points per tooth flank |
| `T` | `GearThickness` | float64 | 8 | Axial thickness of the gears of a stage |
| `Tc` | `CarrierThickness` | float64 | 3 | Axial thickness of a carrier's plate |
| `g` | `StageGap` | float64 | 1 | Axial gap between a stage's gears, its carrier, the next stage and the ring ends; at least 0.75 |
| `Tv` | `CoverThickness` | float64 | 3 | Axial thickness of an end cover's frame |
| `db'` | `BoltHoleDiameter` | float64 | 2.6 | Diameter of the four clearance holes; the default 2.6 is 0.3 mm radially wider than the M2 shank |
| `Sf` | `FlatDepth` | float64 | 0.5 | Depth of the flat on each shaft; zero leaves round shafts and round bores |
| `Li` | `InputShaftLength` | float64 | 12 | How far the input shaft reaches out of the front cover's bearing boss |
| `cb` | `BearingClearance` | float64 | 0.05 | Radial clearance between each purchased bearing and its custom pocket |
| `Sc` | `SeatChamfer` | float64 | 0.5 | Setback of the chamfer on the entrance of each bearing pocket, in the planets and the covers; zero leaves it sharp |
| `Ec` | `EdgeChamfer` | float64 | 0.5 | Setback of the 45° chamfer on the edges of the large flat faces of the ring, the covers and the carriers; at least 0.5 |
| `wc` | `CarrierWall` | float64 | 3 | Carrier material around each planet pin, and cover material around a bearing |
| `w` | `RingWall` | float64 | 7.1 | Radial wall of the ring behind the tooth roots; it holds the bolt holes |
| `Lo` | `OutputShaftLength` | float64 | 12 | How far the output shaft reaches out of the rear cover's bearing boss |
| `Ne` | `EarCount` | int | 3 | Mounting ears on the ring; zero leaves the ring a plain cylinder |
| `de` | `EarBoreDiameter` | float64 | 4 | Diameter of the bore through each ear; positive when there are ears |
| `j` | `Backlash` | float64 | 0.15 | Tooth thinning of the sun and planets at the pitch circle; the ring's tooth spaces are widened by the same amount |
| — | `Placement` | `*r3.Frame` | nil | Reference point and plane |

Use finite inputs. Zero fields are not default requests; `DefaultParams` supplies the defaults.
A nil `Placement` is the one exception.

## Derived dimensions

| Dimension | Formula |
| --- | --- |
| Sun, planet, ring pitch radius | `m Zs / 2`, `m Zp / 2`, `m Zr / 2` |
| Orbit radius `a` | `m (Zs + Zp) / 2` |
| Measurement margin `ε` | 0.02 mm on external gear tips and the planet root |
| Sun root radius | `m Zs/2 - m - c` |
| Planet root radius, tip radius | `m Zp/2 - m - c - ε`, `m Zp/2 + m - ε` |
| Ring tip radius (smallest ring radius) | `m Zr/2 - m` |
| Root-to-tip clearance `c` | `max(0.25 m, 0.5)` |
| Sun tip radius | `m Zs/2 + m - ε` |
| Ring root radius (largest tooth-space radius) | `m Zr/2 + m + c` |
| Ring outer radius | ring root radius `+ w` |
| Shaft diameter `ds` | Shaft bearing bore (no clearance against the bearing) |
| Shaft bore diameter | Shaft diameter `+ 0.1` (0.05 mm radial clearance, a press fit; see Part clearances) |
| Planet bearing lift | `0.25` above the pocket floor; the bearing extends `0.25` above the planet |
| Ear half width `we` | `de / 2 + 3` |
| Ear bore center radius `Re` | ring outer radius `+ we` |
| Bolt circle radius | (ring root radius + ring outer radius) / 2 |
| Cover rim inner radius | bolt circle radius `-` head pocket radius `- 1.5` |
| Head pocket diameter | `m2bolt.HeadDiameter + 0.6` (depth `m2bolt.HeadHeight`) |
| Nut cavity diameter | `2 (m2nut.AcrossFlats / √3 + 0.2)`, the circle through the nut's corners plus 0.2 mm radially (height `m2nut.Thickness + 0.4`, starting 1 mm behind the ring's end face) |
| Cover boss diameter | cover pocket diameter `+ 2 wc` |
| Cover hub diameter | cover boss diameter `+ 2 wc` |
| Planet bearing bore, outside diameter, width `dbp`, `Dbp`, `Wbp` | The planet bearing's catalog entry (5, 14, 5 for the default 605ZZ) |
| Shaft bearing bore, outside diameter, width `dbs`, `Dbs`, `Wbs` | The shaft bearing's catalog entry (5, 14, 5 for the default 605ZZ) |
| Pin diameter `dp` | `dbp` |
| Pin collar diameter `dpc` | `dp + 2.4` (7.4 by default), the thick part of a pin |
| Planet pocket diameter | `Dbp + 2 cb` |
| Planet lip bore diameter | `Dbp - 4` |
| Planet lip thickness | `T - Wbp` |
| Cover pocket diameter | `Dbs + 2 cb` |
| Cover lip bore diameter | `Dbs - 4` |
| Carrier hub diameter | `ds + 2 wc` |
| Pin boss diameter | `dp + 2 wc` |
| Shaft collar diameter | `ds + 2.4` (7.4 by default); the thick part of a link or output shaft and the input shaft's collar, D with flat `Sf` |
| Pin socket | The pin's thick D (pin collar diameter) grown 0.05 radially |
| Carrier centre hole | The shaft's thick D (shaft collar diameter) grown 0.05 radially |
| Carrier arm and cover spoke width | `dp + 2` |
| Stage pitch `P` | `T + Tc + 2g` |
| Ring length `L` | `K P + g` |
| Reduction of one stage | `1 + Zr/Zs` (ring fixed, sun in, carrier out) |
| Total reduction | `(1 + Zr/Zs)^K` |

The sun-planet and planet-ring meshes keep at least `c` between each gear's tip circle and its mate's root circle. Non-meshing planet tips stay at least 0.5 mm apart. Other custom fits use 0.25 mm nominal gaps so their measured
lower bounds can meet the 0.2 mm Tight minimum in
[Part clearances](../../docs/model-pipeline.md#part-clearances). D-shaped shaft bores leave
the same gap normal to their circular and flat faces.

## Input checks and limits

- Reject `j < 0` or `j > m/2`.
- Reject `m <= 0`, `T <= 0`, `Tc <= 0`, `g < 0.75`, `Tv <= 0`, `db' < 2.6`, `Li <= 0`, `cb < 0.05`, `wc <= 0`,
  `w <= 0`, `Lo <= 0`, `Samples < 3` or a nonfinite length.
- Reject `α` outside `(0, π/2)`, `n < 2` and `K < 1`.
- Keep the placed tip-to-root clearance in every mesh at `max(0.25 m, 0.5 mm)` or greater.
- Reject tooth counts below 3 (each gear is built as a spur gear, which has the same limit).
- Reject `Zr != Zs + 2 Zp`. The three gears share axis and module only then.
- Reject `(Zs + Zr) mod n != 0`. Only then can the planets sit at equal angles and all
  mesh with one sun and one ring at once.
- Reject neighbouring planets unless their centers are at least the planet tip diameter plus 0.5 mm apart:
  `2 a sin(π/n) >= planet tip diameter + 0.5`.
- Reject a sun bore (`ds + 0.1`) that leaves less than `m` of wall under the sun's root circle, and a
  planet pocket that leaves less than `m` of wall under the planet's root circle.
- Reject `Sf < 0` or `Sf >= 1.2`, the collar margin (so a thick part's flat stays outside its rod).
- Reject `Tc < 2`, so each press fit grips over at least 2 mm of plate.
- Reject a carrier wall that leaves less than 1 mm of plate around a pin socket at mid-plate:
  `boss/2 - (dpc/2 + 0.05) < 1`, that is `wc < 2.25` for the default pin.
- Reject a carrier wall and edge chamfer that leave less than 0.5 mm between a pin socket and the chamfered outline at a
  face: `boss/2 - (dpc/2 + 0.05) - Ec < 0.5`.
- Reject bolt holes that leave less than 1.5 mm of material between a hole and the ring's tooth
  roots or outer wall, and a cover bearing hub too wide for spokes inside the rim.
- Reject `db'` below the M2 shank diameter plus 0.2 mm, a cover thinner than `m2bolt.HeadHeight + 1`, head or nut pockets that
  leave less than 1 mm of ring wall toward the tooth roots or the outer wall, and a ring too short for a nut cavity at each end.
- Reject `Ne < 0`, a nonfinite `de`, and `de <= 0` when `Ne > 0`.
- Reject ears whose bore centers are closer than `2 we + 1` (`2 Re sin(π/Ne) < 2 we + 1`), and an ear that lies over a bolt
  site: an ear at angle `θ` and a bolt angle `β` with `cos(θ - β) > 0` and `bolt circle radius × |sin(θ - β)|` below
  `we` plus the larger pocket radius plus 1.
- Reject a seat chamfer outside `[0, 1]`.
- Reject an edge chamfer outside `[0.5, 1]`.
- Reject `T - Wbp < 1`: the planet needs a lip at least 1 mm thick under the pocket.
- Reject a shaft that does not fit between the pin bosses (`ds/2 + boss/2 >= a`) and a carrier plate that
  leaves no more than 0.5 mm between its edge and the ring tip radius (`a + boss/2 >= ring tip radius - 0.5`).
- Reject a non-nil `Placement` that is not a valid frame.

## Gears

All three gears are involute spur gears of one module and pressure angle, built by
`models/spurgear`. Do not copy its geometry.

- Sun: `Zs` teeth, thickness `T`, spur backlash `j`, and a centered bore of the shaft's size: a circle of
  diameter the shaft bore diameter (the shaft diameter plus 0.1 mm) with a flat `Sf` inside its rim, which leaves the sun a D-shaped bore
  (a round bore when `Sf = 0`). The D bore is cut after the gear is built, with a tool of that outline.
- Planet: `Zp` teeth, thickness `T`, centered bore of the planet lip bore diameter, spur backlash `j`.
  A bearing pocket of the planet pocket diameter is then cut from the planet's upper face, `Wbp` deep,
  which leaves a lip `T - Wbp` thick under it with the lip bore through it. The lip bore is
  smaller than the bearing's outside diameter, so the bearing cannot leave through the lower face. The pocket's
  entrance on the upper face is chamfered by `Sc`, so the bearing can be pressed into the tight printed seat, as the
  [ball bearing README](../ballbearing/README.md) advises.
- Ring: an outer cylinder of radius `a_out` and length `L`, with four bolt holes of diameter `db'`
  through it at the bolt circle radius and at the angles `π/4 + iπ/2`, from which the tooth spaces are
  cut. The holes are inner loops of the cylinder's one section. The cut tool is an external `Zr`-tooth spur gear of length `L` whose root radius is
  the ring tip radius and whose tip radius is the ring root radius. Its teeth are the
  ring's tooth spaces, because the space of an internal gear at its pitch circle is the
  tooth of the mating external gear. The tool has no root fillet. Its spur backlash is `-j`, which thickens its teeth by `j` of arc
  at the pitch circle and so widens the ring's tooth spaces by `j`. A sun-planet pair and a
  planet-ring pair each have `2j` of circumferential play at the pitch circle.
- Mounting ears: with `Ne > 0` the ring blank's outline is not a circle but one closed loop of arcs of the ring's outer
  radius between the ears and, for each ear, two straight edges and a half circle of radius `we` round its bore. Ear `i`
  lies at angle `π/2 + 2πi/Ne`, so the first ear is on +Y. Its axis is the line from the origin to the bore's center at
  radius `Re`. Its straight edges run parallel to the axis, `we` either side of it, from the outer circle to the half
  circle's ends. Its bore is a hole of the section, `de` across, centered at `Re` on the axis. Every ear runs the whole
  ring length, so its faces lie in the ring's two end planes. The bore clears the covers' outer wall, which has the
  ring's outer radius, by 3 mm. With `Ne = 0` the outline is the circle.

## Meshing phase

The sun's tooth zero, and every stage, lie on `+U`. Planet `i` (0-based) sits at
orbit angle `φᵢ = 2π i/n` and the center `(a cos φᵢ, a sin φᵢ)`. Its tooth zero is turned by

`θᵢ = ((Zs + Zp) φᵢ + π (Zp - 1)) / Zp`

about its own axis, and the ring's tool tooth zero is turned by `θr = π (Zp - 1)/Zr`.

These angles make every planet's teeth enter the sun's gaps and the ring's gaps on the
line between the centers. They are consistent for all planets because `(Zs + Zr)/n` is an
integer. The default gives `θr = 0`.

## Housing

The housing is the ring and two covers, the front cover on the first stage's side and the rear cover on
the last stage's side. A cover is built once: a frame from z=0 to `Tv`, and on its +Z face a bearing boss.
The front cover is turned over about the X axis so its boss faces away from the ring.

- Frame: one extruded section with eight holes. Its outer circle has the ring's outer radius. A hub
  circle of the cover hub diameter surrounds the bearing, three spokes of the spoke width run from the
  hub to a rim whose inner radius is the cover rim inner radius, and the four bolt holes lie in the rim at
  the ring's bolt angles. The cover lip bore diameter is a through hole in the hub. Between the spokes are three
  windows. Spoke `k` runs along the angle `π/6 + 2πk/3`.
- Boss: a ring of the cover boss diameter around a pocket of the cover pocket diameter, `Wbs` tall, standing on
  the frame's +Z face. It starts 1 mm inside the frame so the union overlaps. The boss pocket opens away
  from the frame; its floor is the frame's hub, so the hub's lip bore holds the bearing in.
- The boss pocket's entrance is chamfered by `Sc`, as the planet pocket's is.
- Bearing: a shaft bearing envelope has 0.05 mm radial clearance in the boss pocket and rests on its floor, the cover frame's +Z face, with no axial offset.
- Input shaft: a round rod of diameter `ds` on the axis from `Li` below the front boss to level with the first
  sun's top. It fills the front bearing's bore with no clearance and has 0.05 mm radial clearance in the sun's bore, a press fit, so the sun turns with the shaft. Its flat runs
  from the sun's lower face to the top.

## Axial layout

Stage `k` (0-based) starts at `z = k P` above the reference plane, with a gear zone
`[kP, kP + T]`.

- Sun `k`, planets `k` and the ring teeth lie in the gear zone.
- Bearing `i` of stage `k`: a planet bearing envelope on the planet's axis, from `kP + T - Wbp + 0.25` to
  `kP + T + 0.25`. Its lower face clears the pocket floor by 0.25 mm, its upper face extends
  0.25 mm above the planet, and its bore fits the pin of the same diameter with no clearance.
  Bearings are separate bodies.
- Carrier `k`'s plate lies in `[kP + T + g, kP + T + g + Tc]`. Each planet has a separate pin on the planet's axis,
  thin-thick. Its thin rod of diameter `dp` runs from `z = kP`, through the lip bore and the bearing, up to 0.05 mm
  into the thick part. Its thick part is a D of the pin collar diameter with a flat `Sf` deep on the pin's outer side.
  It starts 0.25 mm above the bearing's top face (`g - 0.5` below the plate) and passes through a D socket in the
  plate, ending level with the plate's upper face. The socket is the thick D grown by 0.05 mm radially and cuts through
  the whole plate. The pin is a press fit in the plate, and the gap is the exception to the 0.2 mm Tight minimum in
  Part clearances. Nothing stands above the plate. The pin prints standing on its thick end.
- Shaft of stage `k`: thin-thick, its bottom level with the carrier's lower face `kP + T + g`. Its thick part is a D of
  the shaft collar diameter with a flat `Sf` deep on +X, pressed through the carrier's centre hole (the thick D grown
  0.05 mm radially). For `k < K - 1` it is the link shaft. Its thick part ends 0.25 mm below the next sun,
  at `Tc + g - 0.25` above the carrier's lower face. Its thin rod of diameter `ds` reaches 0.05 mm into the thick part
  and ends at the top of sun `k + 1`, `(k + 1) P + T`, passing through that sun's bore. The flat of the thin rod runs
  from the next sun's lower face to the shaft's top. For the last stage it is the output shaft. Its thick part ends
  0.25 mm below the rear bearing's inner face, at `Tc + g + Tv - 0.25` above the carrier's lower face. Its thin rod ends
  `Lo` above the rear cover's boss, at `K P + Tv + Wbs + Lo`, and is round in the bearing and the stub. The step bears
  on the next sun's lower face (link) or the bearing's inner race (output) after 0.25 mm of play. A shaft prints
  standing on its thick end.
- The ring spans `[-g, K P]`.
- The front cover's frame spans `[-g - Tv, -g]` and its boss `[-g - Tv - Wbs, -g - Tv]`; the rear cover's
  frame spans `[K P, K P + Tv]` and its boss `[K P + Tv, K P + Tv + Wbs]`. The front bearing fills
  `[-g - Tv - Wbs, -g - Tv]` and the rear bearing `[K P + Tv, K P + Tv + Wbs]`.
- The input shaft spans `[-g - Tv - Wbs - Li, T]`; a collar fills `[-g - Tv, -0.5]`, from the front bearing's inner
  face to 0.5 mm short of the first sun's lower face. That 0.5 mm is the collar clearance.

Each carrier is one solid body, a plate with a socket for each pin, and each pin is its own body, as are the shafts.
A carrier's plate is the convex hull of the `n` pin bosses, a rounded polygon: a boss circle of the boss diameter
around each pin and a straight line tangent to the circles of neighbouring pins on the outside. The plate is solid
between the pins, so it takes the sideways load the planets put on their pins. The sockets and the centre hole pass
through the plate. The plate is extruded `Tc`; it needs `Tc >= 2` so each press fit grips over at least 2 mm.

## Document

The ring is cut for its nut cavities and tooth spaces and then chamfered round the ears, and the chamfer meets facets of
the tooth-space cut that the evaluator holds to about 0.0034 mm. A plain `decad.New` document refuses that chamfer. The
package therefore exposes `NewDocument`, which sets `decad.WithChainTolerance` to 0.1 mm. `Build` needs a document made
with it when `Ne > 0`. With `Ne = 0` a plain document works.

## Parameter table

The gearbox records its inputs and derived dimensions as named entries of the document's
parameter table: `module`, `pressure_angle`, `sun_teeth`, `planet_teeth`, `ring_teeth`,
`planet_count`, `stage_count`, `tooth_backlash`, `gear_thickness`, `carrier_thickness`, `stage_gap`,
`cover_thickness`, `bolt_hole_diameter`, `ear_count`, `ear_bore_diameter`, `ear_wall`, `input_shaft_length`, `flat_depth`,
`planet_bearing_bore_diameter`, `planet_bearing_outer_diameter`, `planet_bearing_width`,
`shaft_bearing_bore_diameter`, `shaft_bearing_outer_diameter`, `shaft_bearing_width`,
`bearing_clearance`, `gear_root_clearance`, `gear_root_margin`, `gear_tip_measurement_margin`,
`shaft_bore_clearance`, `planet_bearing_lift`,
`lip_width`, `carrier_wall`, `ring_wall`, `output_shaft_length`,
`orbit_radius`, `ring_tip_radius`, `ring_root_radius`, `ring_outer_radius`, `pin_diameter`,
`pin_collar_diameter`, `planet_pocket_diameter`, `planet_lip_bore_diameter`, `cover_pocket_diameter`,
`cover_lip_bore_diameter`, `seat_chamfer`, `edge_chamfer`, `shaft_collar_diameter`, `shaft_bore_diameter`,
`collar_margin`, `boss_diameter`, `arm_width`,
`arm_margin`,
`shaft_diameter`, `cover_boss_diameter`, `cover_hub_diameter`, `bolt_circle_radius`,
`stage_pitch`, `ring_length`. The spur gear builds
write their own entries (`teeth`, `thickness`, `bore_diameter` and others) into the same
table, so those hold the last gear built.

## Sketch constraints

Follow the repository sketch rules: anchor every sketch at the origin of its plane,
dimension every circle's center from that origin and its diameter by a bound parameter,
solve and verify each sketch is fully constrained before extruding. The spur gear builds
verify their own sketches.

## Edge finishing

Surface edge finishing chamfers every convex boundary edge of a flat face of at least 400 mm² (net area) along a run of
at least 20 mm. In this model the large flat faces are the ring's two end faces, both faces of each cover's frame and
both faces of each carrier plate. The sun, the planets, the shafts and the bearing boss end are smaller than 400 mm²,
and the bearings, bolts and nuts keep their nominal dimensions.

- Chamfer by `Ec`: the outer boundary of each of those faces (the ring's includes its ears), the three window loops of each cover on both of its faces,
  and the cover lip's edge on the inner face. The bearing rests on the other side of the lip, so no seat shrinks.
- Keep sharp: the ring's tooth edges, because they are the gear flanks and take the root clearance; the pin sockets
  and each carrier's centre hole, which are press-fit rims; each bolt hole, head pocket and nut cavity, whose edges are shorter than 20 mm;
  the pocket and boss edges, which are not convex edges of the face.

## Acceptance cases

| Case | Overrides | Required result |
| --- | --- | --- |
| Default | None | 1 ring, 2 covers, 2 cover bearings, 1 input shaft, 1 sun, 3 planets, 3 bearings, 1 carrier, 1 output shaft, 8 bolts, 8 nuts, all solids; ratio 1 + 71/13 |
| Fasteners | Default | Every bolt head ends level with its cover's outer face, every nut sits in its cavity 1 mm behind the ring's end face, and no bolt or nut overlaps the cover or ring it passes through |
| Two stages | `K = 2` | The default parts plus a second stage: 2 suns, 6 planets, 6 bearings, 2 carriers and a link shaft before the output shaft; the link shaft fills the second sun's bore |
| Shaft flat | Default | The input shaft has a flat on +X where the first sun sits and the sun's D bore has the matching tooth: at the flat the sun has material beyond the shaft's edge, and elsewhere in the bore the sun is absent; under the front bearing the shaft is a full cylinder; the shaft's volume is a round rod's plus the collar's less the flat's circle segment over the sun's thickness; in a two-stage gearbox the second sun's bore has the same tooth for the link shaft's thin rod |
| Housing | Default | The covers and bearings sit at the axial positions above; the bolt holes of the ring and both covers line up; a cover has material on a spoke and none in a window and weighs less than a disc; the input shaft fills the first sun's bore |
| Bearings | Default | Each pocket is empty where its bearing sits; the 14 mm × 5 mm bearing clears the pocket by 0.05 mm radially and is lifted by 0.25 mm axially; the planet weighs the plain gear with the lip bore minus the pocket's ring |
| Seat chamfer | Default and `Sc = 0` | A planet and a cover each weigh less than the `Sc = 0` ones by a ring of triangular section, `Sc²/2` times the pocket's circumference, within the volume bounds plus 1; every body stays one solid |
| Edge finishing | Default | Per [Surface edge finishing](../../docs/model-pipeline.md#surface-edge-finishing): every flat face of at least 400 mm² on the ring, the covers and the carriers has a 45° bevel of `Ec` along its whole outer boundary; the covers' windows and the cover lip's inner edge are bevelled too, and the ring's rims sit `Ec` inside its outer radius. The ring's tooth edges, the pin sockets, the centre holes and the bolt holes stay sharp (see Edge finishing); each carrier face has `n + 1` inner loops, none bevelled |
| Carrier pins | Default | Each planet has its own pin body, starting at the gears' lower face: a 5 mm rod and a D thick part that starts 0.2 to 0.3 mm above the bearing and ends level with the carrier's upper face; the thick part has 0.05 mm of radial clearance in its socket, proven to about 0.026 mm; the pin's volume is the rod's plus the D prism's |
| Carrier shafts | Default and `K = 2` | Each link and output shaft has a thick D part pressed into the carrier's centre hole with 0.05 mm of clearance, its bottom level with the carrier's lower face, ending 0.25 mm short of the next sun or the rear bearing; the thin rod fills the sun's bore or the bearing; the volume is the D prism's plus the rod's less the flat's segment over the thin rod's flat span |
| Carrier | Default | The plate is solid between the pins out to the tangent edge and absent beyond it; the centre hole is a wider D, open in the middle with a tooth where the flat goes; a socket for each pin is open in both faces; nothing stands above the plate, so the carrier's top is the plate's top |
| Mesh | Default | No planet outline vertex lies inside the sun or the ring, and no sun or ring vertex inside a planet |
| Gear roots | Module 0.5, 1 and 2; default placed gear bodies | Root and tip circles leave `max(0.25 m, 0.5 mm)` at the sun-planet and planet-ring meshes; the built model's root parameters match the same radii |
| Backlash | Default and `j = 0.2` | The `j = 0.2` sun and planet have less volume, and its ring has less volume, than the default's |
| Phase | Default with one planet turned half a tooth about its own axis | The same check finds vertices inside the sun and the ring |
| Placed | Default with frame origin `(10,-5,7)`, `U=+Y`, `V=+Z` | Same volumes as the unplaced box; bounds follow the frame |
| Ears | Default; `Ne = 2`, `de = 6`; `Ne = 0` | The default ring has three ears at 90°, 210° and 330° that run the ring's whole length: the section has no material in a bore or beyond an ear's tip or side, and material in the ear's wall and between the wall and the ring; the bore clears the covers' outer wall by at least 3 mm; the ring's bounds follow the ears. `Ne = 2`, `de = 6` gives two larger ears on the Y axis. `Ne = 0` gives the plain cylinder with the earlier rim bevel and builds in a plain document. Every ear sketch is fully constrained |
| Invalid | `Zr != Zs + 2Zp`; `(Zs + Zr) mod n != 0`; colliding planets; a carrier plate that reaches the ring; a gear too thin for the lip; `j` below 0 or above `m/2`; a negative ear count; ears without a bore; ears over the bolt sites; ears too close together; a flat depth of 1.2 or more; a carrier wall under 2.25; an invalid placement | An error naming the failing rule |

## Verification

Build the Default case first. Check every sketch's constraint status, each body's
solidity, bounds and volume, the meshing phase, and the shaft and pin fit. Check the Two
stages case once the Default passes. Render the real body through `tools/render/`: the hero shows
the assembled two-stage gearbox with the ring and both covers cut open along the XZ plane, three more images show
it assembled and uncut from the output end, the input end and the side, and an animation shows the two-stage
gearbox turning from three cameras, its parts moved rigidly with the kinematic angles (carrier `Zs/(Zs+Zr)` of
its sun's, planet `1 - Zr/Zp` of its carrier's).
