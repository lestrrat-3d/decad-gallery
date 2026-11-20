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
bolts per cover, built from the M2 preset of `models/fastener`. Each cover's four bolts have 6 mm length (the shortest
standard length), so the bolt length does not depend on the stage count. A bolt's head sits in a round pocket in the
cover's outer face and ends level with that face. Its nut sits in a closed round cavity in the ring wall, 1.8 mm behind
the ring's end face, so the ring holds the nut and it cannot fall out; the bolt hole is the cavity's only opening. The nut
is dropped in during a print pause. The bolt passes through the cover's remaining 1 mm, the ring's 1.8 mm roof and the
nut, and its tip stays inside the ring's bolt hole. This follows the repository fastener rules (see `AGENTS.md`,
Fasteners).

The ring carries M2 mounting ears so the gearbox can be fixed in place. The ears are tabs on the ring's outside, each with
an M2 clearance bore through it parallel to the gearbox axis. Each ear holds one M2 bolt and one M2 nut from
the M2 preset of `models/fastener`. The bolt's head sits in a round pocket in the ear's rear face and ends level with
that face. Its nut sits in a closed round cavity near the ear's front face, dropped in during the same print pause as the
front housing nuts, and the bolt runs through the whole nut. The bolt's tip stays inside the ear. The gallery's bolt
clamps nothing else: the ear is the nut side of a mounting joint. A part mounted against the ears' front faces takes an
M2 bolt from its own side into each ear's nut, in place of the gallery's bolt.

The Fasteners rules embed a nut only when no other solution exists. [steps.md](steps.md) says why the ear nuts and the
housing nuts are embedded: with a hex pocket open to a face, no standard bolt length runs through an ear's nut with its
head flush and its tip inside the one-stage ring, and the part mounted on the ears, or a cover, covers the face where an
open pocket would hold the nut.

The input and the output connect as follows. The shafts are separate bodies from the carriers, so each part
prints in its own best orientation. The input shaft is a rod as wide as the shaft bearing's bore (5 mm by default): it passes through the front
cover's bearing and fills the first sun's bore. A link shaft runs through each carrier's hole and the next sun's bore, which is
how stages link. The output shaft runs through the last carrier's hole and the rear cover's bearing. Every shaft
is as wide as the shaft bearing's bore (5 mm for the nominal 605ZZ) where it passes through a bearing, so it
has no clearance against the bearing. Where a sun sits on it, a shaft is as wide as the bearing's bore up to 6 mm and
6 mm wide for a wider bore (the 8 mm bore of a 608ZZ), so the sun's bore keeps at least 1.6 mm of wall. Each carrier pin is as wide as the planet bearing's bore. The input shaft is a round cylinder with a flat `Sf` deep on the +X side, which gives it a D
section where the first sun sits on it. Where a bearing sits it is a full cylinder of the shaft diameter, and where
the sun sits it is a cylinder of the sun shaft diameter. The collar between them takes the step. The link and output
shafts are thin-thick: a thin rod of the shaft diameter (the sun shaft diameter for a link shaft, which meets no bearing), and a thicker part of the shaft collar diameter with the same flat, a D section, that is
pressed through the carrier's centre hole. The sun's bore and the carrier's hole are D-shaped to match, so the input
shaft turns the first sun, and each link or output shaft turns its carrier and, for a link shaft, the next sun,
without slipping. The input and output shafts each have a flat on the free end that runs to the end and ends square
1.25 mm outside the bearing boss. A collar ring, a separate printed part, is pressed on that flat next to the boss (see
Arbor collars), and a part pressed on the free end beyond it, such as a rotor, a crank or a coupling, grips the same flat.
The contract does not model that part, a set screw or a motor.

Every shaft and pin prints as two halves lying on their cut faces, see [Split shafts](#split-shafts). Read along that
seam, a square step of a shaft is an inside corner, so a step that does not face a purchased bearing is a cone at 30° to
the axis. A step that faces a bearing's inner ring keeps its square corner, and so does the end of a flat on which a D
bore stops: the first sun's on the input shaft, and a collar ring's on the free end of the input or the output shaft.

The input shaft carries a collar, a round bulge wider than the shaft, between the front bearing's inner race and the
first sun. A collar is the printed stand-in for a washer. It keeps the turning part clear of the bearing and the
cover. Its bearing side is a square step that faces the bearing's inner ring. Its sun side is a 30° cone that takes the
collar's end and meets the shaft 0.5 mm below the sun, so the collar keeps its whole length. The carrier pins, the link shaft and the output shaft have no collar. Each has a D thick part instead, pressed
through the carrier: the pin collar diameter for a pin and the shaft collar diameter for a shaft. A pin's and the
output shaft's thick part ends in a square step that faces a bearing's inner ring and keeps the planet bearing or the rear
bearing clear of the plate. The link shaft's thick part ends level with the plate's upper face and a 30° cone carries it
on to the rod, into a countersink in the next sun.

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
| `Li` | `InputShaftLength` | float64 | 14 | How far the input shaft reaches out of the front cover's bearing boss; at least 5.25, the collar ring's 4.25 and 1 mm of free end |
| `cb` | `BearingClearance` | float64 | 0.05 | Radial clearance between each purchased bearing and its custom pocket |
| `bp` | `PlanetBearing` | string | `"605ZZ"` | Catalog bearing each planet turns on; `"605ZZ"` or `"608ZZ"` (the [ballbearing catalog](../ballbearing/instructions.md#catalog)). Each carrier pin is as wide as its bore |
| `bs` | `ShaftBearing` | string | `"605ZZ"` | Catalog bearing each cover holds for the input or output shaft; `"605ZZ"` or `"608ZZ"`. Each shaft is as wide as its bore |
| `Sc` | `SeatChamfer` | float64 | 0.5 | Setback of the chamfer on the entrance of each bearing pocket, in the planets and the covers; zero leaves it sharp |
| `Ec` | `EdgeChamfer` | float64 | 0.5 | Setback of the 45° chamfer on the edges of the large flat faces of the ring, the covers and the carriers; at least 0.5 |
| `wc` | `CarrierWall` | float64 | 3 | Carrier material around each planet pin, and cover material around a bearing |
| `w` | `RingWall` | float64 | 7.1 | Radial wall of the ring behind the tooth roots; it holds the bolt holes |
| `Lo` | `OutputShaftLength` | float64 | 14 | How far the output shaft reaches out of the rear cover's bearing boss; at least 5.25, as `Li` |
| `Ne` | `EarCount` | int | 3 | Mounting ears on the ring; zero leaves the ring a plain cylinder |
| `de` | `EarBoreDiameter` | float64 | 2.6 | Diameter of the M2 clearance bore through each ear; at least 2.6 and under the nut's 4 mm across its flats when there are ears |
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
| Sun shaft diameter `dss` | `min(ds, 6)`: the shaft's section through a sun (5 mm for a 605ZZ, 6 mm for a 608ZZ) |
| Sun bore diameter (`shaft_bore_diameter`) | `dss + 0.1` (0.05 mm radial clearance, a press fit; see Part clearances) |
| Sun bore wall | Sun root radius `-` half the sun bore diameter: 2.45 mm for every kit, at least 1.6 mm (press-fit wall of [Corners and roots](../../docs/model-pipeline.md#corners-and-roots)) |
| Planet bearing lift | `0.25` above the pocket floor; the bearing extends `0.25` above the planet |
| Ear half width `we` | `de / 2 + 3` |
| Ear bore center radius `Re` | ring outer radius `+ we` |
| Ear root round `re` | `min(w, 2 we) / 2` (half the thinner of the ring wall and the ear's width; 3.55 by default) |
| Cover window round | spoke width `/ 2` (3.5 by default, 5 for a 608ZZ planet bearing) |
| Cover boss root round | `min(wc, Tv) / 2` (1.5 by default) |
| Bolt circle radius | (ring root radius + ring outer radius) / 2 |
| Cover rim inner radius | bolt circle radius `-` head pocket radius `- 1.5` |
| Head pocket diameter | `2 * m2fastener.HeadPocketRadius` (depth `m2fastener.BoltSize.HeadHeight`) |
| Nut cavity diameter | `2 (m2fastener.NutSize.AcrossFlats / √3 + 0.2)`, the circle through the nut's corners plus 0.2 mm radially (height `m2fastener.NutSize.Thickness + 0.4`, starting 1.8 mm behind the ring's end face) |
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
| Cone length | `√3` times the radius change: 2.078 mm for the 1.2 mm from the rod to the shaft collar or the pin collar, 3.811 mm for the 2.2 mm from the 6 mm sun section to the 10.4 mm thick part of a 608ZZ shaft. A cone is at 30° to the axis |
| Free-end flat end | `0.25 + 1` = 1.25 outside each cover's bearing boss: the collar ring's gap to the bearing and its relief |
| Collar ring | `4` long; face diameter the shaft collar diameter `ds + 2.4` (7.4); outside diameter `ds + 0.1 + 2 × 1.7` (8.5; 11.5 for a 608ZZ shaft); D bore the shaft's D grown 0.05 radially (5.1); relief `ds + 0.8` (5.8) across, 1 deep with its 30° cone |
| Ear bolt length `le` | The longest of `m2fastener.BoltSize.StandardLengths` no longer than `L - m2fastener.BoltSize.HeadHeight`: 10 for one stage (`L` 14), 25 for two (`L` 27) |
| Ear nut cavity floor | `max(1.8, L - 2 - le - 0.2)` above the ring's front face: 1.8 for one and two stages, so the cavity is `[1.8, 3.8]` as the front housing nuts' are; deeper when the bolt's tip ends further in |
| Sun countersink | The link shaft's cone moved out by 0.25 mm normal to it, `0.25 / cos 30°` = 0.289 mm along the radius, up to the sun's bore radius (see Gears) |
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
- Reject a sun bore (`dss + 0.1`) that leaves less than `max(m, 1.6)` of wall under the sun's root circle, and a
  planet pocket that leaves less than `m` of wall under the planet's root circle.
- Reject a bearing name not in the catalog: `bp` or `bs` that is not `"605ZZ"` or `"608ZZ"`. This check comes first.
- Reject `Sf < 0` or `Sf >= 1.2`, the collar margin (so a thick part's flat stays outside its rod), and `Sf > min(dbp, dbs) / 4`.
- Reject `Tc < 2`, so each press fit grips over at least 2 mm of plate.
- Reject a carrier wall that leaves less than 1 mm of plate around a pin socket at mid-plate:
  `boss/2 - (dpc/2 + 0.05) < 1`, that is `wc < 2.25` for the default pin.
- Reject a carrier wall and edge chamfer that leave less than 0.5 mm between a pin socket and the chamfered outline at a
  face: `boss/2 - (dpc/2 + 0.05) - Ec < 0.5`.
- Reject bolt holes that leave less than 1.5 mm of material between a hole and the ring's tooth
  roots or outer wall, and a cover bearing hub too wide for spokes inside the rim.
- Reject `db'` below the M2 shank diameter plus 0.2 mm, a cover thinner than `m2fastener.BoltSize.HeadHeight + 1`, head or nut pockets that
  leave less than 1 mm of ring wall toward the tooth roots or the outer wall, and a ring too short for a nut cavity at each end.
- Reject `Ne < 0`, a nonfinite `de`, and `de <= 0` when `Ne > 0`. With ears, reject `de < 2.6` (the M2 clearance hole)
  and `de >= m2fastener.NutSize.AcrossFlats`, so the nut cannot pass the bore, and a ring too short for the shortest standard bolt
  (`L - 2 < 6`).
- Reject `Li < 5.25` or `Lo < 5.25`: each free end must hold its collar ring and stand 1 mm beyond it.
- Reject ears whose bore centers are closer than `2 we + 2 re + 1` (`2 Re sin(π/Ne) < 2 we + 2 re + 1`), so the root
  rounds of neighbouring ears leave 1 mm of the outer circle between them, and an ear that lies over a bolt
  site: an ear at angle `θ` and a bolt angle `β` with `cos(θ - β) > 0` and `bolt circle radius × |sin(θ - β)|` below
  `we` plus the larger pocket radius plus 1.
- Reject a seat chamfer outside `[0, 1]`.
- Reject an edge chamfer outside `[0.5, 1]`.
- Reject `T - Wbp < 1`: the planet needs a lip at least 1 mm thick under the pocket.
- Reject a shaft that does not fit between the pin bosses (`ds/2 + boss/2 >= a`) and a carrier plate that
  leaves no more than 0.5 mm between its edge and the ring tip radius (`a + boss/2 >= ring tip radius - 0.5`).
- Reject a non-nil `Placement` that is not a valid frame.
- Reject a shaft collar that is not narrower than the shaft bearing's inner race (`ds + 2.4 >= ` its `InnerRaceDiameter`),
  and a pin thick part (`dpc`) that is not narrower than the planet bearing's inner race.
- Reject a shaft collar that does not pass the cover lip bore with 0.25 mm on each side (`ds + 2.4 + 0.5 > Dbs - 4`).
- Reject a carrier whose centre hole and pin socket leave less than 1 mm of plate between them:
  `a - (collar/2 + 0.05) - (dpc/2 + 0.05) < 1`.

`Problems(p)` returns every failing rule in this order, and `Validate` returns the first. When an input check fails,
only the input problems are returned. Each bearing-forced minimum (the sun wall, the planet pocket wall, the lip
thickness and the cover hub) also names the minimum it needs, for example `SunTeeth must be at least 12 for the 5 mm
shaft section at module 1`. At module 1 it then names the tooth set of the kit with the same bearings, as `try 13/29/71`.

## Gears

All three gears are involute spur gears of one module and pressure angle, built by
`models/spurgear`. Do not copy its geometry.

- Sun: `Zs` teeth, thickness `T`, spur backlash `j`, and a centered bore of the shaft's sun section: a circle of
  diameter the sun bore diameter (the sun shaft diameter plus 0.1 mm) with a flat `Sf` inside its rim, which leaves the sun a D-shaped bore
  (a round bore when `Sf = 0`). The D bore is cut after the gear is built, with a tool of that outline.
- Sun fed by a link shaft (every sun after the first): the same sun with a countersink cut in its rear face, a cone at 30°
  to the axis that is the link shaft's cone moved out 0.25 mm normal to it. The shaft's cone starts at the plate's upper
  face, `g` below the sun's rear face, at the shaft collar radius, so the countersink's radius at the rear face is
  `ds/2 + 1.2 + 0.289 - g/√3` (3.41 by default) and it narrows to the bore radius `1.49 mm` above the face (`√3 × (ds/2 + 1.2 + 0.289 - bore radius) - g`).
  The D bore's tooth begins there, so the sun keeps the 0.25 mm gap it had to the shaft's old square step. The countersink is a
  revolved profile and a Boolean cut after the D bore. The first sun sits on the input shaft and has none.
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
  ring's outer radius, by 3 mm. Each straight edge meets the outer circle in a round of radius `re`, tangent to both,
  because the ears carry the gearbox's whole reaction torque (a stressed root). With one ear the arc before and after the
  ear is the same arc. The rounds are drawn with `sketch.CreateFillet`, which adds the tangencies and the radius
  dimension and leaves the sketch fully constrained. With `Ne = 0` the outline is the circle.
- Ear pockets: with the housing nut cavities, and before the tooth spaces, each ear takes a head pocket and a nut cavity
  on its bore's axis, cut with the housing's own tools. The head pocket is a circle of the head pocket diameter, as deep
  as the head is tall (2 mm), open to the ear's rear face, the one in the ring's rear end plane. The nut cavity is a
  closed circle of the nut cavity diameter and height, its floor the ear nut cavity floor above the front face. The
  bore is each cavity's only opening. The ear's wall round each pocket is `we` less its radius, 2.1 mm round the head
  pocket and 1.79 mm round the nut cavity by default, and the head pocket lies 2.1 mm beyond the covers' outer wall, so
  a driver reaches the head past the rear cover.

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
  windows. Spoke `k` runs along the angle `π/6 + 2πk/3`. Each window's four corners are rounded with the spoke
  width over two, because the spokes carry the bearing's load from the hub to the rim (stressed roots). The window's
  straight edges run parallel to their spoke's axis, half the spoke width from it, and its arcs are about the origin, so
  the window is held by those relations and the rounds, not by corner coordinates. The check that the hub fits inside
  the rim leaves room for the rounds: a window's straight edge must be longer than the spoke width.
- Boss: a ring of the cover boss diameter around a pocket of the cover pocket diameter, `Wbs` tall, standing on
  the frame's +Z face. It starts 1 mm inside the frame so the union overlaps. The boss pocket opens away
  from the frame; its floor is the frame's hub, so the hub's lip bore holds the bearing in. The circular edge where the
  boss's outside wall stands on the frame is rounded by `min(wc, Tv) / 2`, 1.5 mm by default. It opens upward when the
  cover prints frame down, so it takes a round. The round lies inside the hub, clear of the window rounds.
- The boss pocket's entrance is chamfered by `Sc`, as the planet pocket's is.
- Bearing: a shaft bearing envelope has 0.05 mm radial clearance in the boss pocket and rests on its floor, the cover frame's +Z face, with no axial offset.
- Collar rings: one printed collar ring on the free end of each of the input and output shafts, built once and placed
  twice; the front one is turned over about X. See Arbor collars for its shape.
- Input shaft: a round rod of diameter `ds` on the axis from `Li` below the front boss to the collar, then a round rod
  of diameter `dss` from the collar to level with the first sun's top. The `ds` rod fills the front bearing's bore with
  no clearance. The `dss` rod has 0.05 mm radial clearance in the sun's bore, a press fit, so the sun turns with the
  shaft. The whole shaft, collar included, is one revolved meridian (see the axial layout); for a shaft bearing with a bore of 6 mm or
  less the two rods are one diameter. Its flat on the `dss` rod runs
  from the sun's lower face to the top, and a second flat on the free end runs from `Li` outside the boss to 1.25 mm short
  of the boss, where it ends square.

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
  0.05 mm radially). For `k < K - 1` it is the link shaft. It is one revolved meridian: the thick part up to the plate's
  upper face (`Tc`, 3 mm), a 30° cone from there to the rod (`√3` times the radius change long: 2.078 mm, ending `Tc + 2.078` = 5.078 above the carrier's
  lower face; 3.811 mm, ending 6.811 above it, for the 6 mm sun section on a 10.4 mm thick part), and the rod of the sun section, diameter `dss`, up to the top of sun `k + 1`, `(k + 1) P + T`, passing through that sun's bore.
  Its flat is one plane that follows the shaft: at `Sf` inside the thick part's rim up to the plate, along the cone at the
  same 30°, `Sf` inside its surface, and at `Sf` inside the rod's rim from the cone's end to the top. The flat has no
  end. The sun's countersink (see Gears) faces the cone and its D bore's tooth starts above the flat's ramp. For the last stage it is the output shaft. Its thick part ends
  0.25 mm below the rear bearing's inner face, at `Tc + g + Tv - 0.25` above the carrier's lower face, in a square step that faces
  the bearing's inner ring. Its thin rod ends `Lo` above the rear cover's boss, at `K P + Tv + Wbs + Lo`, and is round in
  the bearing and the stub. Its flat on the free end runs at full depth from 1.25 mm above the boss to the top (z 22.25
  to 35 by default) and ends square toward the boss, where the rear collar ring's D bore stops on it. The step bears
  on the bearing's inner race after 0.25 mm of play. A shaft prints
  standing on its thick end.
- The ring spans `[-g, K P]`.
- The front cover's frame spans `[-g - Tv, -g]` and its boss `[-g - Tv - Wbs, -g - Tv]`; the rear cover's
  frame spans `[K P, K P + Tv]` and its boss `[K P + Tv, K P + Tv + Wbs]`. The front bearing fills
  `[-g - Tv - Wbs, -g - Tv]` and the rear bearing `[K P + Tv, K P + Tv + Wbs]`.
- The input shaft spans `[-g - Tv - Wbs - Li, T]`, one revolved meridian; a collar fills `[-g - Tv + 0.25, -0.5]`, from 0.25 mm above the
  front bearing's inner face to 0.5 mm short of the first sun's lower face. The 0.25 mm is the Tight gap to the
  bearing and the 0.5 mm is the collar clearance. The collar is `g + Tv - 0.75` long, 3.25 mm by default and never under
  3.0 mm, because `g >= 0.75` and `Tv >= 3`. Its bearing side is a square step. Its sun side is a 30° cone that meets the sun section
  at `-0.5`, so the collar's whole length, measured to where the cone meets the rod, is the full 3.25 mm; the
  cylinder at the collar radius is 1.17 mm long (`[-3.75, -2.578]`) and the cone is `[-2.578, -0.5]`. The first sun's lower face is 0.5 mm
  above the cone's end, which is the closest the sun comes to the collar. With a 608ZZ shaft bearing the sun section is 6 mm,
  and a cone down 2.2 mm of radius would be 3.81 mm long, longer than the collar. The collar then keeps a 0.8 mm cylinder at its
  bearing-side face, at radius 4.41 mm instead of 5.2 mm (the radius from which a 30° cone reaches the 3 mm sun section at `-0.5`), and the cone takes the other 2.45 mm. Its sun-side flat, on the sun section, runs at full depth from the sun's lower face (z 0) to
  the top and ends there in a square step, where the sun's D bore stops (see Retained sharp corners). Its free-end flat runs at full depth
  from the shaft's free end to 1.25 mm below the boss (z −23 to −10.25 by default) and ends square there, where the
  front collar ring's D bore stops on it. It is on the bearing section.
- The front collar ring spans `[-g - Tv - Wbs - 4.25, -g - Tv - Wbs - 0.25]` (`[-13.25, -9.25]` by default) and the rear
  one `[K P + Tv + Wbs + 0.25, K P + Tv + Wbs + 4.25]` (`[21.25, 25.25]`). Beyond each, `Li - 4.25` or `Lo - 4.25`
  (9.75 mm by default) of free end is left for a pressed part.

Each carrier is one solid body, a plate with a socket for each pin, and each pin is its own body, as are the shafts.
A carrier's plate is the convex hull of the `n` pin bosses, a rounded polygon: a boss circle of the boss diameter
around each pin and a straight line tangent to the circles of neighbouring pins on the outside. The plate is solid
between the pins, so it takes the sideways load the planets put on their pins. The sockets and the centre hole pass
through the plate. The plate is extruded `Tc`; it needs `Tc >= 2` so each press fit grips over at least 2 mm.

## Arbor collars

[Arbor collars](../../docs/model-pipeline.md#arbor-collars) applies. Inside the housing each collar is a raised part of a
shaft's or pin's own body. Outside it, each free end carries a separate printed collar ring. The table lists every one
with the default dimensions, from the axial layout above. `z` is the height above the reference plane. The shaft collar diameter is `ds + 2.4` (7.4; 10.4 with
a 608ZZ shaft bearing) and the pin collar diameter is `dp + 2.4` (7.4). A collar that ends in a cone is measured to where the
cone meets the rod, as the rule asks, and every cone is 2.078 mm long (1.2 mm of radius at 30° to the axis).
`TestInputShaftCollar` and `TestCollarRings` measure the input shaft collar and the two collar rings on the built bodies.

| Collar | Shaft | Carried part | Flush side | z | Len | Option |
|---|---|---|---|---|---|---|
| Input shaft collar | Input shaft | First sun, which turns with the shaft | Front bearing's inner face, `z -4` | `[-3.75, -0.5]`, cone `[-2.578, -0.5]` | 3.25 | 1: the 4 mm gap, less the 0.25 Tight gap at the bearing and the 0.5 collar clearance at the sun |
| Output shaft thick part | Output shaft | Last carrier, which turns with the shaft | Rear bearing's inner face, `z 16` | `[9, 15.75]` | 6.75 | 1: 3.75 above the plate's upper face (`z 12`), 0.25 short of the bearing |
| Link shaft thick part | Link shaft | First carrier, and the next sun above it | Carrier plate, `[9, 12]` | `[9, 12]`, cone `[12, 14.078]` | 5.078 | None of the three: the part is pressed through the plate and ends level with its upper face, and the 0.75 mm above the plate (to the next sun, 1 mm up) cannot hold a cone, so the cone runs along the rod into the next sun's countersink |
| Carrier pin thick part | Carrier pin | Planet bearing, which turns on the pin | Carrier plate, `[9, 12]` | `[8.5, 12]` | 3.5 | None of the three: the part is pressed through the plate, 0.5 mm of it stands below, and the planet bearing top is 0.25 below that |
| Front collar ring | Input shaft | A part pressed on the free end outside the frame (not modeled) | Front bearing's outer face, flush with the boss face, `z -9` | `[-13.25, -9.25]` | 4.0 | 1: the free end is the gap; the ring is separate because the input shaft collar stops the shaft from passing the bearing |
| Rear collar ring | Output shaft | A part pressed on the free end outside the frame (not modeled) | Rear bearing's outer face, flush with the boss face, `z 21` | `[21.25, 25.25]` | 4.0 | 1: as the front ring; the output shaft's thick part stops the shaft from passing the bearing |

Each collar ring is a separate printed collar, pressed on the free end's flat. A collar of the shaft's own body there
would stop the shaft from passing its bearing during assembly, because the input shaft collar or the output shaft's thick
part already stands on the bearing's inner side. Assemble each shaft through its bearing from inside the housing, then
press the ring on from the free end until its D bore's tooth meets the square end of the flat. That locates the ring
0.25 mm off the bearing's outer face, a Tight gap, and holds the shaft in the bearing from both sides. The ring is:

- 4 mm long, measured from its bearing face to its outer face.
- At its bearing face, `ds + 2.4` across (7.4; 10.4 with a 608ZZ shaft bearing). It bears on the bearing's inner ring
  only, inside the inner race diameter (8; 12).
- Over its pressed length, from 1 mm off the bearing face to the outer face, a D bore that is the shaft's D grown 0.05 mm
  radially, a press fit, with a wall of 1.7 mm round it (8.5 mm across; 11.5 with a 608ZZ shaft bearing). That keeps the
  1.6 mm press-fit wall of [Corners and roots](../../docs/model-pipeline.md#corners-and-roots) with room for the measured
  bound. The ring prints with its axis vertical, outer face down.
- A 30° cone from the face diameter out to the full one, 0.953 mm long, so the face shrinks without a step.
- At the bearing face a relief in the bore, `ds + 0.8` across, a Regular 0.4 mm gap round the round shaft, which keeps
  0.8 mm of wall under the face's narrow rim. The relief ends in a 30° cone that meets the D bore 1 mm from the face.

A part pressed on the free end beyond the ring sits against its outer face with the press-fit gap (0.05 mm), and the
ring keeps it 4.25 mm off the boss. The outer face is a plain square face, so the pressed part needs no countersink.

The pin and link shaft thick parts use none of the three options. Each is a press-fit plug in the carrier plate, with
3.0 mm of its length inside the plate. The free gap beside it is too short for option 1. For a pin, the planet bearing's
top face (`z 8.25`) to the plate's lower face (`z 9`) is 0.75 mm, which is 0.5 mm after the 0.25 mm fit gap. For the link
shaft, the plate's upper face (`z 12`) to the next sun's lower face (`z 13`) is 1 mm. Option 2 needs a counterbore in the
carried part and forbids the collar from passing through the flush side, but these
thick parts pass through the plate. The construction keeps them as they are, and the table lists them so a later change
can decide. The link shaft's cone meets the next sun's countersink with the 0.25 mm gap the shaft's old square step had to the
sun, measured normal to the cone (Fits, below).

### Fits

| Fit | Surfaces | Gap | Class |
|---|---|---|---|
| Link shaft cone to the next sun's countersink | The cone and the countersink, both at 30° to the axis | 0.25 mm normal to the cone, 0.289 mm along the radius | Tight (0.2 mm minimum); it is the gap the old square step had to the sun's rear face |
| Input shaft collar's cone to the first sun | The cone's end, at the rod, and the sun's lower face | 0.5 mm axial (collar clearance) | Loose (0.5 mm); unchanged |
| Link shaft thick part to the first carrier's hole, output shaft thick part to the last carrier's hole, pin thick part to its socket | D to D | 0.05 mm radial | Press fit, unchanged |
| Sun's D bore to a shaft's rod | D to D | 0.05 mm radial | Press fit, unchanged |
| Collar ring's D bore to the free end's flat | D to D | 0.05 mm radial | Press fit |
| Collar ring's bearing face to the cover bearing's outer face | Face to face | 0.25 mm axial | Tight (0.2 mm minimum) |
| Collar ring's relief to the round shaft | Bore to rod | 0.4 mm radial | Regular (0.35 mm minimum) |
| A part pressed on a free end to its collar ring's outer face | Face to face | 0.05 mm axial | Press fit; the part is not modeled |

## Split shafts

[Split shafts](../../docs/model-pipeline.md#split-shafts) applies. The model builds each part below as one body and does
not model the split. Each prints as two halves, each lying on its cut face.

| Part | Split plane (normal) | What holds the halves |
|---|---|---|
| Input Shaft | Holds the axis and the flat's normal, +X, so its normal is ±Y | The front bearing's inner race, a no-clearance fit on the rod, the first sun's D bore, a press fit on both halves, and the front collar ring's D bore and the pressed part on the free end |
| Link Shaft (each stage but the last) | The same, normal ±Y | The carrier's D hole, a press fit on the thick part, and the next sun's D bore, a press fit on the rod |
| Output Shaft | The same, normal ±Y | The last carrier's D hole, a press fit on the thick part, the rear bearing's inner race, a no-clearance fit on the rod, and the rear collar ring's D bore and the pressed part on the free end |
| Carrier Pin (each) | Holds the axis and the direction its flat faces, which is its orbit direction `(cos φ, sin φ)`, so its normal is the tangent `(-sin φ, cos φ)` | The planet bearing's inner race, a no-clearance fit on the rod, and the carrier's D socket, a press fit on the thick part |

Each part is mirror-symmetric about its split plane, has no downward face wider than a 6 mm bridge, and stands at least 3.7 mm
(4.41 mm for the input shaft and 5.2 mm for the link and output shafts with a 608ZZ shaft bearing) above its cut face. `TestSplitShafts` runs `splitcheck.Check` on each part of the
default build and then `cornercheck.CheckAt` along the split normal at the heights `cornercheck.SplitHeights` returns for
the smallest radius that meets a step or a cone (`dss/2` for the input and link shafts, `ds/2` for the output shaft, `dp/2` for a pin), 0.1 mm to 0.8 of it from the seam on each side in 0.25 mm steps. It
exempts only the corners listed under Retained sharp corners. `TestSplitShaftsVariants` does the same for the three other
kits, two stages and `Sf = 0`, and runs in the scheduled full suite. With `Sf = 0` no part has a flat, so the split
plane takes the +X direction (the pin's orbit direction for a pin) as its reference, which is where the flat faces when
`Sf > 0`.

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
`shaft_bearing_bore_diameter`, `shaft_bearing_outer_diameter`, `shaft_bearing_width`, `sun_shaft_diameter`,
`bearing_clearance`, `gear_root_clearance`, `gear_root_margin`, `gear_tip_measurement_margin`,
`shaft_bore_clearance`, `planet_bearing_lift`,
`lip_width`, `carrier_wall`, `ring_wall`, `output_shaft_length`,
`orbit_radius`, `ring_tip_radius`, `ring_root_radius`, `ring_outer_radius`, `pin_diameter`,
`pin_collar_diameter`, `planet_pocket_diameter`, `planet_lip_bore_diameter`, `cover_pocket_diameter`,
`cover_lip_bore_diameter`, `seat_chamfer`, `edge_chamfer`, `shaft_collar_diameter`, `shaft_bore_diameter`,
`collar_ring_wall`, `collar_ring_relief_gap`, `collar_ring_length`,
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
verify their own sketches. The input and link shafts, the sun countersink, the collar ring and its relief are revolved
from one meridian sketch on the XZ
plane (`U` the radius, `V` the height) that starts on the axis: the first corner is dimensioned from the origin, each
vertical edge takes its length, each horizontal edge and each cone edge takes its corner's distance from the axis, bound
to the table (`shaft_diameter / 2`, `shaft_collar_diameter / 2`, `shaft_bore_diameter / 2`, and for the collar ring
`shaft_diameter / 2 + shaft_bore_clearance + collar_ring_wall` and `shaft_diameter / 2 + collar_ring_relief_gap`), and
each cone edge takes
its 30° angle to the first vertical edge. The flat tools are the same kind of sketch, extruded symmetrically along Y.

## Retained sharp corners

Rule: "Corners and roots" in `docs/model-pipeline.md`. Printed parts are cut in their print orientation (axis +Z).
Exempt corners:

| Part | Corner | Reason |
|---|---|---|
| Ring Gear | tooth tips and roots, annulus from the ring tip radius `- 0.2` to the ring root radius `+ 0.2` about the axis (34.3 to 37.2 by default; 33.8 to 36.7 for the 70-tooth kits) | tooth generator: the tool that cuts the tooth spaces has no root fillet, and the internal tips are working flanks |
| Planet Gear | tooth tips and roots, annulus from the planet root radius `- 0.2` to the planet tip radius `+ 0.2` about each planet's axis (12.78 to 15.68 by default; 12.28 to 15.18 for the 28-tooth kits) | tooth generator; the tips turn 127° |
| Covers | the bearing pocket's floor corner (between layers) | mating corner: the purchased bearing's outer ring sits on the floor, and its 0.15 to 0.3 mm corner chamfer leaves no room for a 0.5 mm round |
| Planet Gear | the planet bearing pocket's floor corner (between layers) | the same |
| Covers | the floor corners of the four bolt-head pockets (between layers) | Fasteners rule: the pocket is the head diameter plus 0.3 mm and as deep as the head, and the head's flat underside needs the whole floor |
| Ring Gear | the eight housing nut cavities and the ear nut cavities, one per ear (between layers) | Fasteners rule: closed cavities whose roofs print as bridges over the dropped-in nut |
| Ring Gear | the floor corners of the ear head pockets, one per ear (between layers) | Fasteners rule, as the covers' head pockets |
| Input Shaft | the collar's bearing-side step: the inside corner at radius `ds/2` and `z -3.75` (between layers; in the split halves, every section across the seam) | mating corner with the front bearing: the step faces the inner ring across the 0.25 mm Tight gap, and the ring's corner sits in this corner, so a cone or a round would reach it |
| Output Shaft | the thick part's step: the inside corner at radius `ds/2` and `z 15.75` (between layers; in the split halves, every section across the seam) | mating corner with the rear bearing, as above |
| Carrier Pins | the thick part's step: the inside corner at radius `dp/2` and `z 8.5` (between layers; in the split halves, every section across the seam) | mating corner with the planet bearing's inner ring, as above |
| Input Shaft, first Sun Gear | the square start of the shaft's sun-side flat at `z 0`, at the chord `dss/2 - Sf` from the axis, and the sun's D bore chord that stops on it (between layers; in the split halves, every section across the seam that cuts the flat); none with `Sf = 0` | mating corner pair: the sun's tooth begins at its lower face on the end of the flat. A ramp there would run into the collar's cone (0.5 mm below) and the sun's chord would ride on it. Both corners stay sharp |
| Input Shaft and Front Collar, Output Shaft and Rear Collar | the square end of each free end's flat, 1.25 mm outside its boss (`z -10.25` and `z 22.25` by default), at the chord `ds/2 - Sf` from the axis, and the collar ring's D bore chord that stops on it (between layers; in the split halves, every section across the seam that cuts the flat); none with `Sf = 0` | mating corner pair: the end of the flat locates the collar ring, so it is square, and the ring's tooth begins on it. Both corners stay sharp |

Every other step of a shaft is a 30° cone, and every other flat runs on to the end of the part or follows a cone, so the
split halves have no other inside corner (see Split shafts). The free ends' flats end square on the collar rings' teeth. The collar, link shaft and output shaft steps that earlier
builds listed as retained, and the link shaft's step to the next sun, are no longer exempt except as the rows above say.

Rounds below half their section: none. The ring's ear roots, the cover windows' corners and the cover boss root are
rounded at half their section.

Press-fit walls: the sun's D bore keeps 2.45 mm of wall to the tooth roots in every kit, each carrier keeps 1.75 mm
or more round its pin sockets and its shaft hole, and each collar ring keeps 1.7 mm round its D bore over its pressed
length, so the 1.6 mm press-fit wall clause holds. Each collar ring prints with its axis vertical. A shaft through a 608ZZ
narrows to 6 mm where a sun sits on it, because an 8.1 mm bore would leave 1.45 mm under a 14-tooth sun's root circle.

The 1.6 mm clause covers printed press-fit tubes only, not the pocket that holds a purchased bearing. The 608ZZ planet
bearing pocket therefore leaves 1.43 mm of wall to the planet root circle in the 28-tooth kits (the one-module minimum
plus 0.43 mm), which is allowed: the bearing seat grips a purchased part with a small radial fit and loads no printed
tube. See [Corners and roots](../../docs/model-pipeline.md#corners-and-roots).

Known pending violation, not changed here: with two or more stages in the two kits with a 608ZZ shaft bearing
(`planet605-shaft608` and `planet608-shaft608`), the countersink of a sun fed by a link shaft (10.4 mm thick part) opens
to 4.91 mm radius at its rear face, which leaves 0.59 mm of wall under the 5.5 mm sun root there (0.8 mm is the minimum
for a wall), rising to 0.8 mm at 0.37 mm above the face. The 6 mm sun section does not change it, because the cone starts at
the thick part's radius. With the 605ZZ shaft bearings the same wall is 1.59 mm. A shorter thick part would shorten the
carrier's press fit. It is a separate decision.

## Edge finishing

Surface edge finishing chamfers every convex boundary edge of a flat face of at least 400 mm² (net area) along a run of
at least 20 mm. In this model the large flat faces are the ring's two end faces, both faces of each cover's frame and
both faces of each carrier plate. The sun, the planets, the shafts, the collar rings and the bearing boss end are smaller
than 400 mm², and the bearings, bolts and nuts keep their nominal dimensions.

- Chamfer by `Ec`: the outer boundary of each of those faces (the ring's includes its ears and their root rounds), the three window loops of each cover on both of its faces (the chamfer follows the rounded corners),
  and the cover lip's edge on the inner face. The bearing rests on the other side of the lip, so no seat shrinks.
- Keep sharp: the ring's tooth edges, because they are the gear flanks and take the root clearance; the pin sockets
  and each carrier's centre hole, which are press-fit rims; each bolt hole, head pocket and nut cavity, the ears' included, whose edges are shorter than 20 mm;
  the pocket and boss edges, which are not convex edges of the face.

## Bearing kits

A kit is one supported pair of bearings with a tooth set that passes the checks. The module is 1 and each stage has
three planets. `SupportedCombinations` lists the kits in this order and `DefaultParamsFor(kit)` returns
`DefaultParams` with the kit's bearings and teeth. The default kit comes first and equals `DefaultParams`.

| Kit | Planet bearing | Shaft bearing | Teeth `Zs/Zp/Zr` | One-stage ratio |
| --- | --- | --- | --- | --- |
| `planet605-shaft605` | 605ZZ | 605ZZ | 13/29/71 | 6.4615 |
| `planet605-shaft608` | 605ZZ | 608ZZ | 14/28/70 | 6.0 |
| `planet608-shaft605` | 608ZZ | 605ZZ | 13/29/71 | 6.4615 |
| `planet608-shaft608` | 608ZZ | 608ZZ | 14/28/70 | 6.0 |

The shaft is 6 mm wide through a sun (6.1 mm bore, 2.45 mm of wall under the 14-tooth sun's 5.5 mm root radius), and the
608ZZ shaft kits keep 14/28/70 for their ratio. An unchanged 8.1 mm bore would need `Zs >= 16` for the 1.6 mm wall. The 608ZZ planet pocket
leaves 0.93 mm of wall over the one-module minimum with 29 planet teeth and 0.43 mm with 28. The model never changes
teeth to fit a bearing. An unknown kit name is an error that lists the supported kits.

## Acceptance cases

| Case | Overrides | Required result |
| --- | --- | --- |
| Default | None | 1 ring, 2 covers, 2 cover bearings, 1 input shaft, 1 sun, 3 planets, 3 bearings, 3 carrier pins, 1 carrier, 1 output shaft, 2 collar rings, 8 housing bolts and 8 nuts, 3 ear bolts and 3 ear nuts, all solids (42 bodies); ratio 1 + 71/13 |
| Fasteners | Default | Every bolt head ends level with its cover's outer face, every nut sits in its cavity 1.8 mm behind the ring's end face, and no bolt or nut overlaps the cover or ring it passes through. Each ear bolt is a standard length whose head ends level with the ear's rear face and whose tip stays inside the ear; its nut rests in its cavity wholly between the head and the tip, keeps its 1.8 mm roof and 1 mm under the head pocket, and neither overlaps the ring |
| Collar rings | Default and each kit | Each ring is 4 mm long (3.0 mm or more) and stands 0.25 mm (0.2 mm or more) off its bearing's outer face; its face is under the bearing's inner race diameter; its D bore keeps the press-fit gap (proven to 0.02 mm) and a 1.6 mm wall to the outside at mid-press; its relief keeps 0.35 mm to the shaft; its tooth starts 1 mm from the face, where the shaft's flat ends |
| Kits | Each kit of Bearing kits | Each kit validates and has its bearings and teeth. Each non-default kit builds and passes the default's fit, edge, container, fastener, gear-clearance and explode checks. `planet608-shaft605` and `planet608-shaft608` build only when `PLANETARY_ALL_KITS` is set, because each build takes 6 to 10 minutes; otherwise their subtests are skipped |
| Two stages | `K = 2` | The default parts plus a second stage: 2 suns, 6 planets, 6 bearings, 2 carriers and a link shaft before the output shaft; the link shaft fills the second sun's bore, and its 30° cone keeps 0.25 mm normal to the second sun's countersink (0.289 mm along the radius, read at three heights up the countersink) |
| Shaft flat | Default | The input shaft has a flat on +X where the first sun sits and the sun's D bore has the matching tooth: at the flat the sun has material beyond the shaft's edge, and elsewhere in the bore the sun is absent; under the front bearing the shaft is a full cylinder; the shaft's volume is a round rod's plus the collar's, whose sun side is a cone, less the flat's circle segment over the sun's thickness and over the free end's flat, which ends square 1.25 mm outside the boss; in a two-stage gearbox the second sun's bore has the same tooth for the link shaft's thin rod |
| Housing | Default | The covers and bearings sit at the axial positions above; the bolt holes of the ring and both covers line up; a cover has material on a spoke and none in a window and weighs less than a disc; the input shaft fills the first sun's bore |
| Bearings | Default | Each pocket is empty where its bearing sits; the 14 mm × 5 mm bearing clears the pocket by 0.05 mm radially and is lifted by 0.25 mm axially; the planet weighs the plain gear with the lip bore minus the pocket's ring |
| Seat chamfer | Default and `Sc = 0` | A planet and a cover each weigh less than the `Sc = 0` ones by a ring of triangular section, `Sc²/2` times the pocket's circumference, within the volume bounds plus 1; every body stays one solid |
| Edge finishing | Default | Per [Surface edge finishing](../../docs/model-pipeline.md#surface-edge-finishing): every flat face of at least 400 mm² on the ring, the covers and the carriers has a 45° bevel of `Ec` along its whole outer boundary; the covers' windows and the cover lip's inner edge are bevelled too, and the ring's rims sit `Ec` inside its outer radius. The ring's tooth edges, the pin sockets, the centre holes and the bolt holes stay sharp (see Edge finishing); each carrier face has `n + 1` inner loops, none bevelled |
| Carrier pins | Default | Each planet has its own pin body, starting at the gears' lower face: a 5 mm rod and a D thick part that starts 0.2 to 0.3 mm above the bearing and ends level with the carrier's upper face; the thick part has 0.05 mm of radial clearance in its socket, proven to about 0.026 mm; the pin's volume is the rod's plus the D prism's |
| Carrier shafts | Default and `K = 2` | Each link and output shaft has a thick D part pressed into the carrier's centre hole with 0.05 mm of clearance, its bottom level with the carrier's lower face, ending 0.25 mm short of the next sun or the rear bearing; the thin rod fills the sun's bore or the bearing; the link shaft's thick part ends level with the plate's upper face and a 30° cone runs on to the rod; the volume is the D prism's (plus the cone's) plus the rod's less the flat's segment over the thin rod's flat span (the output shaft's free end from 1.25 mm outside the boss) and, along the link shaft's cone, the segment that shrinks with it |
| Carrier | Default | The plate is solid between the pins out to the tangent edge and absent beyond it; the centre hole is a wider D, open in the middle with a tooth where the flat goes; a socket for each pin is open in both faces; nothing stands above the plate, so the carrier's top is the plate's top |
| Mesh | Default | No planet outline vertex lies inside the sun or the ring, and no sun or ring vertex inside a planet |
| Gear roots | Module 0.5, 1 and 2; default placed gear bodies | Root and tip circles leave `max(0.25 m, 0.5 mm)` at the sun-planet and planet-ring meshes; the built model's root parameters match the same radii |
| Backlash | Default and `j = 0.2` | The `j = 0.2` sun and planet have less volume, and its ring has less volume, than the default's |
| Phase | Default with one planet turned half a tooth about its own axis | The same check finds vertices inside the sun and the ring |
| Placed | Default with frame origin `(10,-5,7)`, `U=+Y`, `V=+Z` | Same volumes as the unplaced box; bounds follow the frame |
| Ears | Default; `Ne = 2`, `de = 3.5`; `Ne = 0` | The default ring has three ears at 90°, 210° and 330° that run the ring's whole length: the section has no material in a bore or beyond an ear's tip or side, and material in the ear's wall, in each root round and between the wall and the ring; each ear's head pocket and nut cavity are open inside their radius with 1 mm or more of ear round them; the ring's volume exceeds the plain ring's by the ears' sections plus two root rounds each, less the pockets; the bore clears the covers' outer wall by at least 3 mm and the head pocket lies beyond it; the ring's bounds follow the ears. `Ne = 2`, `de = 3.5` gives two larger ears on the Y axis. `Ne = 0` gives the plain cylinder with no ear fasteners, the earlier rim bevel, and builds in a plain document. Every ear sketch is fully constrained |
| Corners | Default (`TestCorners`); the other kits, `Ne = 0, 1, 6` and `K = 2` (`TestCornersVariants`) | Per [Corners and roots](../../docs/model-pipeline.md#corners-and-roots): every printed body, cut along +Z at the middle of each run of layers with `internal/cornercheck`, has no finding outside Retained sharp corners. The default runs in pull request CI. The variants, about 5 minutes, run in the scheduled full suite, including the two kits with a 608ZZ planet bearing. Every new sketch is fully constrained in every case |
| Split shafts | Default (`TestSplitShafts`); the other kits, `K = 2` and `Sf = 0` (`TestSplitShaftsVariants`) | Per [Split shafts](../../docs/model-pipeline.md#split-shafts): every shaft and pin, built and posed, passes `splitcheck.Check` with the split normal that Split shafts lists, and its two halves read on their cut faces with `cornercheck.CheckAt` along that normal at the `cornercheck.SplitHeights` heights have no finding outside the listed bearing-facing steps and the square flat start. Every exemption claims a finding. The default runs in pull request CI; the variants run in the scheduled full suite |
| Invalid | `Zr != Zs + 2Zp`; `(Zs + Zr) mod n != 0`; colliding planets; a carrier plate that reaches the ring; a gear too thin for the lip; `j` below 0 or above `m/2`; a negative ear count; ears without a bore; an ear bore under 2.6 or of 4 or more; ears over the bolt sites; ears too close together (counting their root rounds); a flat depth of 1.2 or more; a carrier wall under 2.25; `Li` or `Lo` under 5.25; an invalid placement | An error naming the failing rule |

## Verification

Build the Default case first. Check every sketch's constraint status, each body's
solidity, bounds and volume, the meshing phase, and the shaft and pin fit. Check the Two
stages case once the Default passes. Render the real body through `tools/render/`: the hero shows
the assembled two-stage gearbox with the ring and both covers cut open along the XZ plane, three more images show
it assembled and uncut from the output end, the input end and the side, and an animation shows the two-stage
gearbox turning from three cameras, its parts moved rigidly with the kinematic angles (carrier `Zs/(Zs+Zr)` of
its sun's, planet `1 - Zr/Zp` of its carrier's). The collar rings turn with their shafts. An ear section image cuts the
one-stage ring along the plane through the axis and the first ear, with the ear's bolt and nut in place.

Every sketch the build makes is verified fully constrained before its profile is used (`profileWithHoles` and
`extrudeDisc`), so a build that returns proves it for that case. The tests build every branch of the new sketches: the
collar ring with a D bore (default) and a round one (`Sf = 0`, `TestPartClearances`), for the 605ZZ and the 608ZZ shaft
bearing (`TestKitGearboxes`); the ear pockets with three, two and no ears; and the free ends' flats for one and two stages.

Each non-default kit that has a page, `models/planetarygearbox/kit-<kit>.md`, shows a hero, three assembled views, an exploded view and a parts table rendered through `tools/render/` with `-kit <kit>`. The exploded view is at least 1600 × 900 pixels and fills the page width.
