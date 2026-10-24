# Cycloidal drive construction contract

This contract is the gallery's source for the cycloidal drive. It follows the Fusion add-in's
`fusion360-gear-generator/spec/cycloidal/instructions.md` and `epitrochoid-trace.md` for the
dimensions, the profile and the validity checks, and it names where the gallery builds the same
geometry another way. Lengths are millimeters.

## Inputs

`Params` carries every dialog input of the add-in, in its order. Names in the second column are the
parameter-table names `defineParameters` registers.

| Field | Table name | Default | Meaning |
| --- | --- | --- | --- |
| `PinCount` | `pin_count` | 16 | `N`, the ring-pin positions. The rotor has `L = N - 1` lobes. |
| `PinCircleDiameter` | `pin_circle_diameter` | 90 | `2R`, the pitch circle. |
| `PinDiameter` | `pin_diameter` | 0 | Ring-pin diameter. 0 derives `Rr`. |
| `Eccentricity` | `eccentricity` | 1.5 | `E`. |
| `DiskClearance` | `disk_clearance` | 0.3 | `c`, backlash cut into the profile. |
| `DiscCount` | `disc_count` | 1 | 1, or 2 discs 180° apart. |
| `DiscThickness` | `disc_thickness` | 8 | Axial thickness of each disc. |
| `DiscGap` | `disc_gap` | 0.5 | Gap between disc 1 and disc 2. |
| `CenterBearingDiameter` | `center_bearing_diameter` | 30 | Cam outer diameter, and the disc center bore before clearance. |
| `InputShaftDiameter` | `input_shaft_diameter` | 8 | Bore through the cam. 0 leaves the cam solid. |
| `BearingClearance` | `bearing_clearance` | 0.2 | Diametral clearance of the disc bore over the cam. |
| `OutputPinCircleDiameter` | `output_pin_circle_diameter` | 50 | `2Rop`. |
| `OutputPinCount` | `output_pin_count` | 6 | `M`. |
| `OutputPinDiameter` | `output_pin_diameter` | 0 | Output-pin diameter. 0 derives it. |
| `HousingWall` | `wall` | 3 | Minimum radial wall of the casing. |
| `BaseThickness` | `base_thickness` | 5 | Axial thickness of the housing base. |
| `OutputPlateThickness` | `output_plate_thickness` | 5 | Axial thickness of the output plate. |
| `ChamferSize` | `chamfer_size` | 0.5 | 45° chamfer on the housing and plate rims and on the pin ends. 0 is none. |
| `Placement` | | nil | Frame whose origin is a point on the drive axis and whose plane holds disc 1's lower face. The stack grows along its normal. |

The add-in's Target Plane, Anchor Point and Parent Component selections become `Placement`. The
per-field status text boxes are dialog furniture and have no gallery counterpart.

## Derived values

`Rr = PinDiameter / 2` when `PinDiameter > 0`, else `(E + R sin(π/N)) / 2`. `Rr_eff = Rr + c`.
`Rv = R - Rr_eff - E`. `D_pin = OutputPinDiameter` when it is positive, else `Rop sin(π/M) - E`.
`D_hole = D_pin + 2E`. The housing outer diameter is `2 (R - Rr + 2E + wall)`, its base inner diameter
is `2 (R - Rr - wall)`, and the output plate diameter is `2 Rop + D_pin + 2 wall`.
`Rr` and `D_hole` are numeric snapshots in the parameter table because they choose between a derived
and an overridden value. Every other derived name is an expression.

## Validity

`Problems(p)` returns every failing check, in the add-in's order and with its wording, and `Validate`
wraps them in `ErrInvalid`: two discs need even `N` and `M`; `E < Rr < R sin(π/N)`; `D_pin > 0`;
`D_hole < 2 Rop sin(π/M)`; `E < R/N`; `Rop < Rv`; the no-undercut guard `Rr_eff < ρ_min`, with the
bound `E*` found by 40 bisection steps; `ISD < CBD`; `E + ISD/2 < CBD/2`; and
`(CBD + clearance)/2 < Rop - D_hole/2`. The gallery adds the count floors (`N >= 4`, `M >= 3`), the
sign checks on every length, and one more rule: two discs need `DiscGap > 0`, because the cam sections
join through the gap.

## Geometry

**Rotor profile.** The profile is the equidistant of a shortened epitrochoid:

```
psi = atan2(sin((1-N) t), R/(E N) - cos((1-N) t))
x0 =  R cos t - Rr_eff cos(t + psi) - E cos(N t)
y0 = -R sin t + Rr_eff sin(t + psi) + E sin(N t)
```

rotated by the disc's clocking and centered on the disc center. Disc 0 is centered at `+E` along the
placement U axis with no clocking. Disc 1 is centered at `-E` and turned by π, which is disc 0 turned
half a turn about the drive axis. One lobe spans `t` in `[0, 2π/L]`, valley to valley, and its ends lie
on the root circle of radius `Rv`.

**Sampling.** One lobe is traced at 2000 steps and thinned so consecutive points turn by 5°. Each lobe
has two inflections where the curvature changes sign. Decad certifies a fit spline only when its
curvature has one sign, so each lobe is drawn as one fit spline per sign, three per lobe, sharing the
points at the cuts. Decad certifies a body by work proportional to its spline pieces times their
points. When a build with 5° sampling exceeds that ceiling, as 23 lobes do, the disc is built again
with 8°, 12° and then 20° sampling.

**Rotor disc.** One sketch on the disc's plane holds the closed outer loop of all lobes, the center
bore of diameter `CenterBearingDiameter + BearingClearance` centered on the disc center, and the `M`
output holes of diameter `D_hole` on the circle of radius `Rop` about the disc center, the first at the
disc's clocking angle. One region of that sketch, with `1 + M` holes, is extruded by `DiscThickness`.
Disc `d` starts at `d (DiscThickness + DiscGap)`.

**Casing contour.** The casing's inner wall is the disc's swept envelope pushed out by `c`. As the cam
turns by θ the disc center orbits the drive axis at `E` and the disc turns by `-θ/L`. For every
direction in one pin pitch `[-π/N, π/N]`, 81 of them, the contour takes the largest distance from the
drive axis at which any traced profile crosses that direction, measured on the segment between
neighboring profile points (720 cam angles, 2000 profile steps). The result is mirrored about the pin
and repeated `N` times. Its lowest point is the pin surface `R - Rr` and its highest is
`R - Rr + 2E`. The add-in bins the sweep by radius instead, which leaves a jagged contour that decad's
curvature certificate refuses.

**Housing.** One outer cylinder of diameter `2 (R - Rr + 2E + wall)` runs from the underside of the base,
`1 mm + BaseThickness` below disc 1, to the top of the stack. A cavity bounded by the contour (`N` pitches,
each split at its inflections into three fit splines through every third sampled point, and through every
second, fourth, sixth or single point when decad refuses) is cut from one plane gap above the base to past
the top. That leaves the casing wall and a base floor. A cylinder of diameter `2 (R - Rr - wall)` is then
cut through the floor. No two bodies share a face, which decad does not decide.

**Cam.** Section `d` is a cylinder of diameter `CenterBearingDiameter` centered on disc `d`'s center. A
one-disc cam is one cylinder over the disc. A two-disc cam joins a cylinder over disc 0 and the gap and a
cylinder over the gap and disc 1, which overlap through the gap. The input bore of diameter
`InputShaftDiameter` on the drive axis is cut through the whole cam when it is positive.

**Output.** The plate lies one plane gap above the stack, with `M` sockets on the circle of radius `Rop`
about the drive axis. Each pin has diameter `D_pin` and runs from the top of the plate down to the lower
face of disc 1, through every disc's output holes. Pin `k` is at angle `2πk/M`.

**Chamfers.** `ChamferSize` chamfers the outer circles of the housing's two end faces, the outer circle of
both plate faces and the ends of every pin. The rotor discs are not chamfered: the outer loop of a cap is
made only of fit splines, and decad cannot offset a chain of free-form edges exactly.

## Result

`Build` returns a `Drive` whose bodies carry the add-in's names: `Cycloidal Disk 1` (and `2`), `Housing`,
`Eccentric Cam`, `Output Plate` and `Output Pin 1` to `Output Pin M`. `Drive.Named` lists them. The
add-in groups them into four sub-components in Fusion's occurrence tree. Decad has no component tree, so
the grouping is only the order of `Named`. The document needs `decad.WithChainTolerance(units.Millimeters(0.1))`.

## Checks

Every body passes 0.1 mm `VerifyAll`. Every sketch is fully constrained: points are dimensioned from the
sketch origin, which is the drive axis, and circle diameters are bound to the parameter table. At the home
pose the rotor clears the casing, the cam and the output pins, and the housing clears the cam and the plate.
