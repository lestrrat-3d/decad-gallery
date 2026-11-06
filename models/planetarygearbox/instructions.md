# Planetary gearbox construction instructions

Use this contract to derive [steps.md](steps.md), then generate the Go package and tests.
This file owns geometry, dimensions, input rules and case inputs. The steps own
interfaces, Sketch/Decad calls and acceptance assertions.

## Inputs

Build a compact planetary gearbox. A fixed internal ring gear is the housing. In each stage
a sun gear drives `n` planet gears that roll inside the ring, and the planets' pins ride on a
carrier. The sun is the input and the carrier is the output. A stage count above one stacks
identical stages in the same ring: each carrier drives the next stage's sun through a
shared stub shaft, and the last carrier ends in the output shaft.

Every part that turns on another part rides on a purchased 608ZZ bearing. The planets turn on
their pins, so each planet holds one bearing in a pocket and the bearing's inner race sits on
the pin. The sun and the stub shaft turn with the carrier or the input shaft that drives them,
so they hold no bearing. The ring is fixed.

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
| `Tc` | `CarrierThickness` | float64 | 3 | Axial thickness of a carrier's hub and arms |
| `g` | `StageGap` | float64 | 0.5 | Axial gap between a stage's gears, its carrier, the next stage and the ring ends |
| `ds` | `ShaftDiameter` | float64 | 5 | Sun bore, stub shaft and output shaft diameter |
| `cb` | `BearingClearance` | float64 | 0.1 | Radial clearance between a planet's bearing and its pocket |
| `wc` | `CarrierWall` | float64 | 3 | Carrier material around the shaft and around each planet pin |
| `w` | `RingWall` | float64 | 4 | Radial wall of the ring behind the tooth roots |
| `Lo` | `OutputShaftLength` | float64 | 10 | Output shaft length above the last carrier |
| `j` | `Backlash` | float64 | 0.05 | Tooth thinning of the sun and planets at the pitch circle; the ring's tooth spaces are widened by the same amount |
| — | `Placement` | `*r3.Frame` | nil | Reference point and plane |

Use finite inputs. Zero fields are not default requests; `DefaultParams` supplies the defaults.
A nil `Placement` is the one exception.

## Derived dimensions

| Dimension | Formula |
| --- | --- |
| Sun, planet, ring pitch radius | `m Zs / 2`, `m Zp / 2`, `m Zr / 2` |
| Orbit radius `a` | `m (Zs + Zp) / 2` |
| Planet root, tip radius | `(m Zp - 2.5 m)/2`, `(m Zp + 2 m)/2` |
| Ring tip radius (smallest ring radius) | `m Zr/2 - m` |
| Ring root radius (largest tooth-space radius) | `m Zr/2 + 1.25 m` |
| Ring outer radius | ring root radius `+ w` |
| Bearing bore, outside diameter, width `db`, `Db`, `Wb` | 8, 22, 7 (the 608ZZ, from `ballbearing.Params608ZZ`) |
| Pin diameter `dp` | `db` |
| Planet pocket diameter | `Db + 2 cb` |
| Planet lip bore diameter | `Db - 4` |
| Planet lip thickness | `T - Wb` |
| Carrier hub diameter | `ds + 2 wc` |
| Pin boss diameter | `dp + 2 wc` |
| Carrier arm width | `dp` |
| Stage pitch `P` | `T + Tc + 2g` |
| Ring length `L` | `K P + g` |
| Reduction of one stage | `1 + Zr/Zs` (ring fixed, sun in, carrier out) |
| Total reduction | `(1 + Zr/Zs)^K` |

The ring tip radius leaves `m/4` radial clearance over the planet root circle, and the ring
root radius leaves `m/4` over the planet tip circle, as the sun-planet pair has.

## Input checks and limits

- Reject `j < 0` or `j > m/2`.
- Reject `m <= 0`, `T <= 0`, `Tc <= 0`, `g < 0`, `ds <= 0`, `cb < 0`, `wc <= 0`, `w <= 0`,
  `Lo <= 0`, `Samples < 3` or a nonfinite length.
- Reject `α` outside `(0, π/2)`, `n < 2` and `K < 1`.
- Reject tooth counts below 3 (each gear is built as a spur gear, which has the same limit).
- Reject `Zr != Zs + 2 Zp`. The three gears share axis and module only then.
- Reject `(Zs + Zr) mod n != 0`. Only then can the planets sit at equal angles and all
  mesh with one sun and one ring at once.
- Reject neighbouring planets that collide: the distance between adjacent planet centers
  `2 a sin(π/n)` must be at least the planet tip diameter plus `m/4`.
- Reject a sun bore (`ds`) that leaves less than `m` of wall under the sun's root circle, and a
  planet pocket that leaves less than `m` of wall under the planet's root circle.
- Reject `T - Wb < 1`: the planet needs a lip at least 1 mm thick under the pocket.
- Reject a stub shaft that does not fit between the pin bosses (`ds/2 + boss/2 >= a`), carrier
  arms that reach past the ring's tip radius minus `m/2` (`a + boss/2 >= ring tip radius - m/2`), and
  arms so wide that neighbouring arms overlap on the hub.
- Reject a non-nil `Placement` that is not a valid frame.

## Gears

All three gears are involute spur gears of one module and pressure angle, built by
`models/spurgear`. Do not copy its geometry.

- Sun: `Zs` teeth, thickness `T`, centered bore of diameter `ds`, spur backlash `j`.
- Planet: `Zp` teeth, thickness `T`, centered bore of the lip bore diameter, spur backlash `j`.
  A bearing pocket of the pocket diameter is then cut from the planet's upper face, `Wb` deep,
  which leaves a lip `T - Wb` thick under it with the lip bore through it. The lip bore is
  smaller than the bearing's outside diameter, so the bearing cannot leave through the lower face.
- Ring: an outer cylinder of radius `a_out` and length `L`, from which the tooth spaces are
  cut. The cut tool is an external `Zr`-tooth spur gear of length `L` whose root radius is
  the ring tip radius and whose tip radius is the ring root radius. Its teeth are the
  ring's tooth spaces, because the space of an internal gear at its pitch circle is the
  tooth of the mating external gear. The tool has no root fillet. Its spur backlash is `-j`, which thickens its teeth by `j` of arc
  at the pitch circle and so widens the ring's tooth spaces by `j`. A sun-planet pair and a
  planet-ring pair each have `2j` of circumferential play at the pitch circle.

## Meshing phase

The sun's tooth zero, and every stage, lie on `+U`. Planet `i` (0-based) sits at
orbit angle `φᵢ = 2π i/n` and the center `(a cos φᵢ, a sin φᵢ)`. Its tooth zero is turned by

`θᵢ = ((Zs + Zp) φᵢ + π (Zp - 1)) / Zp`

about its own axis, and the ring's tool tooth zero is turned by `θr = π (Zp - 1)/Zr`.

These angles make every planet's teeth enter the sun's gaps and the ring's gaps on the
line between the centers. They are consistent for all planets because `(Zs + Zr)/n` is an
integer. The default gives `θr = 0`.

## Axial layout

Stage `k` (0-based) starts at `z = k P` above the reference plane, with a gear zone
`[kP, kP + T]`.

- Sun `k`, planets `k` and the ring teeth lie in the gear zone.
- Carrier `k`'s hub and arms lie in `[kP + T + g, kP + T + g + Tc]`. `n` pins of diameter `dp` hang
  from the arm ends at the planet centers down to `z = kP`, and pass through the bearings and the
  lip bores.
- Stub shaft of carrier `k`: a cylinder of diameter `ds` on the axis from the carrier's
  midplane up. For `k < K - 1` it ends at the top of sun `k + 1`, `(k + 1) P + T`, and
  passes through that sun's bore. For the last carrier it is the output shaft and ends at
  `kP + T + g + Tc + Lo`.
- The ring spans `[-g, K P]`. The first sun's bore is the input shaft's seat.

- Bearing `i` of stage `k`: a 608ZZ envelope on the planet's axis, from `kP + T - Wb` to `kP + T`,
  so it fills the pocket and its upper face is level with the planet's. Its bore is the pin's
  diameter. Bearings are separate bodies.

Each carrier, with its pins and stub, is one solid body. A carrier is a hub of the hub
diameter on the axis with `n` arms. Arm `i` runs from the hub along the direction of planet `i`
to a round boss of the boss diameter centered on that planet's pin, and is the arm width wide.
Hub, arms and bosses are one outline extruded `Tc`; no plate fills the space between the arms. The
pins hang from the arm ends down to `z = kP` and the stub shaft rises from the hub.

## Parameter table

The gearbox records its inputs and derived dimensions as named entries of the document's
parameter table: `module`, `pressure_angle`, `sun_teeth`, `planet_teeth`, `ring_teeth`,
`planet_count`, `stage_count`, `tooth_backlash`, `gear_thickness`, `carrier_thickness`, `stage_gap`,
`shaft_diameter`, `bearing_bore_diameter`, `bearing_outer_diameter`, `bearing_width`,
`bearing_clearance`, `lip_width`, `carrier_wall`, `ring_wall`, `output_shaft_length`,
`orbit_radius`, `ring_tip_radius`, `ring_root_radius`, `ring_outer_radius`, `pin_diameter`,
`pocket_diameter`, `lip_bore_diameter`, `hub_diameter`, `boss_diameter`, `arm_width`,
`stage_pitch`, `ring_length`. The spur gear builds
write their own entries (`teeth`, `thickness`, `bore_diameter` and others) into the same
table, so those hold the last gear built.

## Sketch constraints

Follow the repository sketch rules: anchor every sketch at the origin of its plane,
dimension every circle's center from that origin and its diameter by a bound parameter,
solve and verify each sketch is fully constrained before extruding. The spur gear builds
verify their own sketches.

## Acceptance cases

| Case | Overrides | Required result |
| --- | --- | --- |
| Default | None | 1 ring, 1 sun, 3 planets, 3 bearings, 1 carrier, all solids; ratio 1 + 71/13 |
| Two stages | `K = 2` | 1 ring, 2 suns, 6 planets, 6 bearings, 2 carriers; the first carrier's stub fills the second sun's bore |
| Bearings | Default | Each pocket is empty where its bearing sits; the bearing is 22 mm across, 7 mm wide and level with the planet's upper face; the planet weighs the plain gear with the lip bore minus the pocket's ring |
| Carrier | Default | Material lies along an arm and not between two arms at the same radius; the carrier weighs less than a plate that reaches as far as the arms |
| Mesh | Default | No planet outline vertex lies inside the sun or the ring, and no sun or ring vertex inside a planet |
| Backlash | Default and `j = 0.2` | The `j = 0.2` sun and planet have less volume, and its ring has less volume, than the default's |
| Phase | Default with one planet turned half a tooth about its own axis | The same check finds vertices inside the sun and the ring |
| Placed | Default with frame origin `(10,-5,7)`, `U=+Y`, `V=+Z` | Same volumes as the unplaced box; bounds follow the frame |
| Invalid | `Zr != Zs + 2Zp`; `(Zs + Zr) mod n != 0`; colliding planets; arms that reach the ring; a gear too thin for the lip; `j` below 0 or above `m/2`; an invalid placement | An error naming the failing rule |

## Verification

Build the Default case first. Check every sketch's constraint status, each body's
solidity, bounds and volume, the meshing phase, and the stub and pin fit. Check the Two
stages case once the Default passes. Render the real body through `tools/render/`.
