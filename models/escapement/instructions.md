# Graham escapement

## Purpose and source

Construct a desk-clock-size Graham deadbeat anchor escapement as one assembly: a spring-driven great wheel and pawl carrier,
a 30-tooth escape wheel, an anchor, a crutch and a pendulum, all in a two-plate frame on seven purchased 605ZZ
bearings. It is the source for downstream ports. All lengths are in millimeters.

## Choice of escapement

Build a **Graham deadbeat anchor escapement** driven by an open spiral mainspring, regulated by a pendulum through a crutch.

- Graham locks on arcs centred on the pallet arbor, so a locked tooth exerts no torque on the anchor (no recoil).
  Small print errors change only the lock depth and the drop, not the action. The tolerance study below keeps
  one tooth per period, a lock on the locking face and a drop of 0.94 mm or more under ±0.2 mm print errors.
- Recoil anchor: simpler, but recoil wears printed teeth and is the less instructive mechanism.
- Swiss lever and pin-pallet: need a balance wheel and hairspring. A printed hairspring is not reliable, and the
  lever, roller, guard pin and banking add tiny parts below comfortable print size.
- Pendulum + crutch + spring is a regulator for Graham. Every part prints without supports in the
  orientation given under Arbors (arbors print on end with a 1.2 mm collar ledge).

## Scale and print size

The model is desk-clock size, not a watch-size miniature.

| Item | Size |
|---|---|
| Escape wheel | 80 mm across (tip radius 40), 30 teeth, 3 mm thick, tooth 7 mm tall |
| Anchor | pallets 2.8 mm wide, 5 mm thick; lock depth about 1.0 to 1.1 mm; drop 1.39 mm at the tooth tip |
| Gears | module 1.5: 13-tooth pinion, 48-tooth great wheel (75 mm across) |
| Frame | open skeleton plates, back plate 116 × 194 mm, front plate 116 × 178 mm, 6 mm thick; 28 mm between plates |
| Movement depth | 51.75 mm, from the back plate's rear face to the pendulum's suspension cap |
| Pendulum | 180 mm from pivot to bob centre by default (120 to 250), bob 50 mm across; 215 mm long overall |
| Bearings | seven purchased 605ZZ (5 × 14 × 5); arbors 5.0 mm, equal to the bearing bore |
| Fasteners | four M2 × 10 bolts and four M2 nuts |

Every part fits a 220 × 220 mm bed. The longest parts are the default pendulum (215 mm overall) and the back
plate (194 mm). A 250 mm pendulum (285 mm overall) needs the diagonal of a 220 mm bed.

## Frame and conventions

- Model frame: origin `O` on the escape arbor axis in the plane of the back plate's inner face. `+X` right, `+Y` up,
  `+Z` toward the viewer (the front). The back plate's inner face is `z = 0`; the wall is on the `-Z` side.
- Lengths in mm. Code angles in radians; this doc gives degrees.
- `math angle` = counter-clockwise from `+X` seen from `+Z`.
- Wheel angle `φ`, anchor angle `ψ` and pendulum angle `β` are **clockwise-positive** seen from `+Z`.
  A part turned by a clockwise angle `a` is rotated by math angle `-a`.
- The escape wheel turns clockwise. The pinion turns with it. The great wheel turns counter-clockwise, and so do the spring's inner end and the pawl carrier while the spring runs down.
- Axes (default): `O = (0, 0)`, `P = (0, d) = (0, 56.5685)` pallet arbor, `S = (0, d + 14) = (0, 70.5685)` pendulum
  suspension, `G = (0, -45.75)` great wheel arbor, directly below `O`. All four axes lie on the plumb line `x = 0`.
- Default pose `Phase = 0.25`: `ψ = 0` and `β = 0`, so the anchor and the pendulum hang plumb. The entry pallet is mid-impulse
  there (ψ between `ψ_Eu` = -0.5° and `ψ_Ef` = +2°); the build pose's 0.05° wheel back-off keeps the tooth off the pallet.
- `Placement *r3.Frame` maps the model frame (origin `O`, `U = +X`, `V = +Y`) into the world; nil is identity.

## Escapement geometry

### Fixed constants

| Symbol | Value | Meaning |
|---|---|---|
| `N` | 30 | escape wheel teeth |
| `R` | 40 | tooth tip radius |
| `H` | 7 | tooth height; root radius `Rr = R - H = 33` |
| `ρ` | 10° | rake: angle from the radial line at the tip to the tooth's front (locking) face, tooth leaning forward |
| `χ` | 30° | angle from the radial line at the tip to the tooth's back face |
| `Sp` | 7.5 | pallets span 7.5 tooth pitches |
| `w` | 2.8 | pallet width, measured radially about `P` |
| `Lpal` | 9° | pallet length about `P` beyond the locking corner, toward the arm |
| `b` | 6 | anchor arm width |
| `rh` | 8 | anchor hub radius |
| `Te` | 3 | escape wheel thickness, `z ∈ [22, 25]` |
| `Ta` | 5 | anchor thickness, `z ∈ [21, 26]` |
| `ov` | 0.5 | overlap of every stacked or patterned union, so unions cross faces instead of sharing them |

### Derived values (formulas, defaults)

| Value | Formula | Default |
|---|---|---|
| pitch `τ` | `2π / N` | 12° |
| half span `θ` | `Sp τ / 2` | 45° |
| centre distance `d` | `R / cos θ` | 56.5685 |
| tangent radius `rm` | `R tan θ` | 40 |
| `r_in`, `r_out` | `rm ∓ w / 2` | 38.6, 41.4 |
| entry lock radius | `r_out` (entry pallet locks on its outer face) | 41.4 |
| exit lock radius | `r_in` (exit pallet locks on its inner face) | 38.6 |

`tip(r, side)`: the point where the tip circle `|X - O| = R` meets the circle `|X - P| = r`:
`y = (R² - r² + d²) / (2d)`, `x = ±sqrt(R² - y²)`, `-` for the left (entry) side, `+` for the right (exit) side.

| Point | Definition | Default (world) | Math angle about `O` |
|---|---|---|---|
| `I_L` | `tip(r_out, left)`: entry lock point | (-29.2569, 27.2770) | 137.0058° |
| `F_L` | `tip(r_in, left)`: entry release point | (-27.2770, 29.2569) | 132.9942° |
| `I_R` | `tip(r_in, right)`: exit lock point | (27.2770, 29.2569) | 47.0058° |
| `F_R` | `tip(r_out, right)`: exit release point | (29.2569, 27.2770) | 42.9942° |

`∠I_L - ∠I_R = 90.0000° = 7.5 τ`, so each beat advances the wheel exactly `τ/2`.

### Event angles of the anchor

`λ` = `LiftAngle` (impulse angle, default 2.5°), `κ` = `LockAngle` (default 1.5°). Positive `ψ` (clockwise) moves the
entry (left) pallet out of the wheel and the exit (right) pallet into it.

| Event | Formula | Default |
|---|---|---|
| `ψ_Eu` entry unlocks | `-(λ - κ)/2` | -0.5° |
| `ψ_Ef` entry releases (impulse ends, drop to exit) | `(λ + κ)/2` | +2.0° |
| `ψ_Xu` exit unlocks | `(λ - κ)/2` | +0.5° |
| `ψ_Xf` exit releases (drop to entry) | `-(λ + κ)/2` | -2.0° |

### Pallet corners

The anchor is drawn at `ψ = 0` in a sketch whose origin is `P` (axes parallel to world). A world point `W` seen at
anchor angle `ψ` has anchor-frame position `A = rot(W - P, +ψ)` (`rot` = math rotation). Each corner is the point that
lies on the tip circle at its event:

| Corner | Formula (anchor frame, relative to `P`) | Default | r about `P` | math angle about `P` |
|---|---|---|---|---|
| `C_E` entry locking corner | `rot(I_L - P, ψ_Eu)` | (-29.5114, -29.0351) | 41.4 | -135.4661° |
| `F_E` entry far corner | `rot(F_L - P, ψ_Ef)` | (-26.3072, -28.2470) | 38.6 | -132.9636° |
| `C_X` exit locking corner | `rot(I_R - P, ψ_Xu)` | (27.5143, -27.0726) | 38.6 | -44.5364° |
| `F_X` exit far corner | `rot(F_R - P, ψ_Xf)` | (28.2168, -30.2948) | 41.4 | -47.0339° |

- Entry pallet: annular band `r ∈ [r_in, r_out]` about `P`. Its locking face is the arc `r_out` from `C_E` up to the
  top angle `a_E = ∠C_E - Lpal = -144.4661°`. Its inner face is the arc `r_in` from `F_E` up to `a_E`. Its impulse
  face is the straight line `C_E → F_E`.
- Exit pallet: same band. Locking face is the arc `r_in` from `C_X` up to `a_X = ∠C_X + Lpal = -35.5364°`. Outer face
  is arc `r_out` from `F_X` to `a_X`. Impulse face is the line `C_X → F_X`.
- Far corners lie deeper in the wheel than locking corners by `r λ` (about 1.7 mm).

### Anchor outline (one closed loop plus the bore)

Arms run along the rays from `P` at `a_E` and `a_X`. Unit vectors: `uE = (cos a_E, sin a_E)`,
`nE = (cos(a_E - 90°), sin(a_E - 90°))`, `uX = (cos a_X, sin a_X)`, `nX = (cos(a_X + 90°), sin(a_X + 90°))`,
`s0 = sqrt(rh² - b²) = 5.2915`.

| Point | Formula | Default (relative to `P`) |
|---|---|---|
| `G_E` | `r_in uE` | (-31.4116, -22.4337) |
| `H_E` | `rh uE` | (-6.5102, -4.6495) |
| `K_E` | `r_out uE` | (-33.6901, -24.0610) |
| `M_E` | `K_E + b nE` | (-37.1773, -19.1784) |
| `N_E` | `s0 uE + b nE` (on the hub circle) | (-7.7932, 1.8073) |
| `G_X` | `r_in uX` | (31.4106, -22.4351) |
| `H_X` | `rh uX` | (6.5100, -4.6498) |
| `K_X` | `r_out uX` | (33.6891, -24.0625) |
| `M_X` | `K_X + b nX` | (37.1764, -19.1800) |
| `N_X` | `s0 uX + b nX` (on the hub circle) | (7.7933, 1.8070) |

Loop, in order (arcs centred on `P`; "ccw" = increasing math angle):

1. line `C_E → F_E` (entry impulse face)
2. arc `r_in` from `F_E` to `G_E` (cw)
3. line `G_E → H_E` (entry arm inner side)
4. arc `rh` from `H_E` to `H_X` ccw through -90° (hub underside)
5. line `H_X → G_X` (exit arm inner side)
6. arc `r_in` from `G_X` to `C_X` (cw) (exit locking face)
7. line `C_X → F_X` (exit impulse face)
8. arc `r_out` from `F_X` to `K_X` (ccw)
9. line `K_X → M_X` (exit arm end)
10. line `M_X → N_X` (exit arm outer side)
11. arc `rh` from `N_X` (13.0540°) ccw over the top to `N_E` (166.9435°)
12. line `N_E → M_E` (entry arm outer side)
13. line `M_E → K_E` (entry arm end)
14. arc `r_out` from `K_E` to `C_E` (ccw) (entry locking face)

Bore: D-shaped (see Arbors), cut after the boss union. Front spacer boss: circle `r 3.7` about `P`,
`z ∈ [25.5, 27.75]` (overlaps the anchor body by 0.5).

### Escape wheel

- Disc: root circle `Rr = 33` with 6 windows, extruded `z ∈ [22, 25]`. Window `k` (`k = 0..5`) lies between the spokes
  at math angles `σk = 90° + 60° k` and `σk+1`. Spoke half-width 2.5. Window corners: on `r 13` at
  `σk + asin(2.5/13)` and `σk+1 - asin(2.5/13)`; on `r 28` at `σk + asin(2.5/28)` and `σk+1 - asin(2.5/28)`. Window
  sides are straight lines parallel to the spoke centrelines, 2.5 away.
- Tooth 0: triangle with tip `T = (0, 40)` and feet on `r = Rr - ov = 32.5`. Front-face foot `L` on the line from
  `T` at angle `ρ` from the inward radial, leaning back (toward `-X`): `L = (-1.3272, 32.4729)`. Back-face foot `B` on
  the line from `T` at angle `χ`: `B = (-4.5118, 32.1853)`. At `r 33` the feet are `(-1.2384, 32.9768)` and
  `(-4.1961, 32.7321)`, a root thickness of 2.97 mm.
- Teeth: tooth 0 patterned 30× about `O` (`decad.PatternCopies`, CircularPattern), unioned with the disc.
- Tooth `j` tip at math angle `90° + j τ` in the part frame. The wheel turns clockwise, so the front face leads.


### Numerical verification

A quasi-static simulation drives the anchor by `ψ = -A cos 2πt`. It advances
the wheel at each step as far as exact contact with the pallet regions allows. It tests tooth triangles against
pallet regions bounded by arcs about `P` and the impulse lines.

Default (`κ 1.5°`, `λ 2.5°`, `A 5°`), periods 2 and 3 (period 1 is warm-up):

| Quantity | Result |
|---|---|
| Wheel advance per period | 12.00000° (one tooth) |
| Recoil | 0.000000° |
| Drops per period | 2: onto the exit pallet at ψ +2.000°, onto the entry pallet at ψ -2.000° |
| Landing face | the locking face both times |
| Drop | 1.9885° of wheel = 1.388 mm at the tip, both sides |
| Lock angle / lift angle measured | 1.514° / 2.486° (step quantisation; design 1.5° / 2.5°) |
| Locking corner depth below the tip circle at landing | entry 1.084 mm, exit 1.009 mm |
| Least gap, tooth outline beyond 1 mm from its tip, to its pallet | 0.126 mm (front face clears the lock face) |
| Least gap, every other tooth to both pallets | 0.613 mm |
| Least gap, pallets to the root circle | 1.947 mm (1.228 mm at A = 6°) |
| Pallet tops and arms to the wheel centre | 42.36 mm and 42.34 mm at A = 6° (2.3 mm outside the tips) |

Parameter sweep (`κ ∈ {1.25, 1.5}`, `λ ∈ {2, 2.5, 3}`, `A ∈ {(λ+κ)/2 + 1°, 5°, 6°}`): every case advances one tooth per
period, lands on the locking faces, keeps the other-teeth gap ≥ 0.595 mm and the root gap ≥ 1.049 mm. `κ = 2°` drops
the other-teeth gap to 0.40 mm, so `κ` stops at 1.5°.

Print-error study (default parameters; `dr` tip radius error, `dw` pallet width error, `dd` centre distance error):

| Case | Advance | Drops (mm) | Lock depth (mm) | Other-teeth gap | Root gap |
|---|---|---|---|---|---|
| nominal | 12° | 1.369 / 1.372 | 1.094 / 1.019 | 0.613 | 1.947 |
| `dr -0.2` | 12° | 1.368 / 1.344 | 0.684 / 0.623 | 0.785 | 1.947 |
| `dw +0.2` | 12° | 1.161 / 1.164 | 1.093 / 1.013 | 0.452 | 1.938 |
| `dw -0.2` | 12° | 1.577 / 1.579 | 1.095 / 1.025 | 0.773 | 1.957 |
| `dd +0.15` | 12° | 1.586 / 1.171 | 0.887 / 0.792 | 0.575 | 2.050 |
| `dd -0.15` | 12° | 1.155 / 1.606 | 1.299 / 1.225 | 0.332 | 1.845 |
| `dr -0.2, dw +0.2, dd -0.15` | 12° | 0.941 / 1.368 | 0.890 / 0.826 | 0.335 | 1.836 |
| `dr -0.2, dw -0.2, dd +0.15` | 12° | 1.759 / 1.354 | 0.497 / 0.399 | 0.925 | 2.060 |

(The drops in this table include the impulse progress inside one step; the closed-form drop is 1.388 mm.)

## Pendulum and crutch

- Crutch: pressed on the pallet arbor's front end, turns with the anchor. Outline in a sketch with origin `P`:
  hub circle `r 6`; arm 5 wide (`x = ±2.5`) straight down; end circle `r 4` centred `(0, -45)`. Outline points:
  `A_R = (2.5, -5.4544)`, `A_L = (-2.5, -5.4544)`, `B_L = (-2.5, -41.8775)`, `B_R = (2.5, -41.8775)`. Loop: hub arc
  ccw `A_R → A_L` over the top, line `A_L → B_L`, end arc ccw `B_L → B_R` round the bottom, line `B_R → A_R`.
  `z ∈ [34.25, 37.25]`. Crutch pin: circle `r 1.5` at `(0, -45)`, `z ∈ [36.75, 43.25]` (overlaps 0.5). D press socket
  (`r 2.55`, flat at `x = 2.05`) through the hub.
- Pendulum: pivots on a 605ZZ on the suspension pin at `S = P + (0, 14)`. Outline in a sketch with origin `S`: eye
  circle `r 10`; rod 8 wide (`x = ±4`) down to a bob circle `r 25` centred `(0, -Lp)`. Outline points:
  `E_R = (4, -9.1652)`, `E_L = (-4, -9.1652)`, `Q_L = (-4, -Lp + 24.6779)`, `Q_R = (4, -Lp + 24.6779)`. Loop: eye arc
  ccw `E_R → E_L` over the top, line `E_L → Q_L`, bob arc ccw `Q_L → Q_R` round the bottom, line `Q_R → E_R`. Holes:
  lip bore circle `r 5` at the origin; crutch slot = stadium from `(0, -61.25)` to `(0, -56.75)`, `r 1.75` (3.5 × 8,
  centred 59 below `S`). Extrude `z ∈ [37.75, 43.75]`. Bearing pocket: circle `r 7.05`, cut `z ∈ [38.75, 44.75]`
  (5 deep from the front face, leaving a 1 mm lip).
- Coupling: the crutch pin centre lies on the pendulum centreline. `β = atan2(Lc sin ψ, e + Lc cos ψ)` with
  `Lc = 45`, `e = 14`. Anchor `A = 5°` gives pendulum 3.814°; `A = 6°` gives 4.577°. The pin's distance from `S`
  changes by 0.06 mm over the swing, inside the 8 mm slot.
- Period `T = 2π sqrt(Lp / 9.81 m/s²)`: 0.695 s at 120 mm, 0.851 s at 180 mm, 1.003 s at 250 mm.
- Suspension pin (fixed): revolve of meridian `(r, z)`: `(0, 31.05) (2.5, 31.05) (2.5, 34) (3.7, 34) (3.7, 38.75)
  (2.5, 38.75) (2.5, 45.70) (0, 45.70)`, about the axis through `S`. The shank is 5.0, the pendulum bearing's bore.
  Flats 0.5 deep on `+X` (flat plane `x = 2.0`) cut by boxes over `z ∈ [30.05, 33.95]` (rear peg, pressed into the
  front plate) and `z ∈ [44.25, 46.70]` (front peg, carries the cap). The bearing sits on the round shank.
- Suspension cap: circle `r 3.7` at `S`, `z ∈ [44.25, 45.75]`, D press socket (`r 2.55`, flat `x 2.05`) through.
- Pendulum bearing: 605ZZ at `S`, `z ∈ [39, 44]`. It sits 0.25 above the pocket floor and stands 0.25 proud of the
  pendulum's front face, as the planetary gearbox's planet bearings do.

## Drive train

An open spiral mainspring on the great wheel arbor drives the movement. Its inner hub sits on the arbor's flat and its outer
eye hooks on a fixed pin. Two flexure pawls on the pawl carrier, which is pressed on the same arbor, push the internal
ratchet of the great wheel. The great wheel drives the 13-tooth pinion on the escape wheel clockwise. Winding: stop the
pendulum and turn the winding knob clockwise. The arbor turns clockwise, the spring tightens, and the pawls click over the
ratchet while the escapement holds the great wheel. The model has no cord or weight.

### Gear pair

| Item | Value |
|---|---|
| module `m`, pressure angle | 1.5, 20° |
| pinion `Zp`, great wheel `Zg` | 13, 48; ratio 48/13 |
| centre distance | `m (Zp + Zg)/2 = 45.75`, `G` at math angle -90° from `O` (directly below) |
| root clearance `c` | `max(0.25 m, 0.5) = 0.5` |
| pinion radii | root `9.75 - 1.5 - 0.5 - 0.02 = 7.73`, tip `9.75 + 1.5 - 0.02 = 11.23` |
| great wheel radii | root `36 - 1.5 - 0.5 - 0.02 = 33.98`, tip `36 + 1.5 - 0.02 = 37.48` |
| tip-to-root | `45.75 - 11.23 - 33.98 = 0.54`, `45.75 - 37.48 - 7.73 = 0.54` (rule: ≥ 0.5) |
| backlash `j` | `Backlash`, default 0.15 per gear → 0.3 play |
| pinion | `spurgear` variant, 13 teeth, thickness 21, `z ∈ [1.5, 22.5]`, no bore (D bore cut after unions) |
| great wheel | `spurgear` variant, 48 teeth, thickness 7, `z ∈ [1.5, 8.5]`, `BoreDiameter 5.8` (round, Regular fit) |
| mesh phase at `φ = 0` | pinion tooth 0 at math angle `θp0 = -90°` (toward `G`); great wheel tooth 0 at `θg0 = 90° + 180°/48 = 93.75°` (gap facing `O`) |

Root-clearance variant: `spurgear.Variant{Define: gearroot.Define(gearroot.Clearance(1.5), 0.02)}` from the shared
`internal/gearroot` package, for both gears. It sets `root_radius = pitch_radius - module - 0.5 - 0.02` and
`tip_radius = pitch_radius + module - 0.02`. Do not write a model-local variant.

### Internal ratchet (in the great wheel)

Cut from the great wheel's front face, in its local frame (tooth 0 of the gear on `+X`). Tool = disc `r 11.5` ∪
12 triangles `{(11.0∠30k°), (13.5∠30(k+1)°), (11.0∠30(k+1)°)}`, `k = 0..11`. The tool spans `z ∈ [5.0, 9.5]`, so the
pocket floor is at `z 5.0` (3.5 mm floor under it) and the pocket opens on the front face `z 8.5`.

Resulting ratchet tooth `k` (material between gap `k-1` and gap `k`): a radial face at `30k°` from `r 13.5` in to
`r 11.5`, facing clockwise. A tip land on `r 11.5` from `30k°` to `30k° + 12.07°`. A slope out to the root `r 13.5`
at `30(k+1)°`. At `r 12.5` the gap spans `[30k + 23.06°, 30(k+1)°]`.

### Spring choice

Use an **open spiral mainspring on the great wheel arbor**, with its outer eye hooked on a fixed pin. It is not in a barrel.
It stays visible through the open front plate. It reuses the winding path: the knob turns the great wheel arbor, and the pawl
carrier on that arbor drives the great wheel's internal ratchet counter-clockwise. `models/spiralspring` builds the spring
(imported, not copied: `spiralspring.Params`, `Derive`, `Build`). The item winds counter-clockwise outward seen from its `+Z`
and tightens when the arbor turns clockwise, so with its `+Z` along the movement's `+Z` the hand matches. A going barrel
would need a new great wheel and would hide the spring, so it is rejected.

### Spring dimensions and torque

`spiralspring.Params` (a constant set of the escapement; the user cannot change it): `OuterDiameter` 64, `HubDiameter` 12,
`Thickness` 1.5, `Height` 6, `MinGap` 1.0, `Turns` 0 (as many as fit), `ArborDiameter` 5, `ArborFlatDepth` 0.5,
`BoreClearance` 0.25, `AnchorPinDiameter` 3, `AnchorClearance` 0.4. `Derive` gives `N = 9` turns, pitch 2.6, gap 1.1,
`rho0` 8.4 and an active length of 1142.3 mm. The eye centre is `(-31.250, -13.399)` local: radius 34.0015 from the axis, at
math angle -156.7916°. The eye radius is 3.4 and the eye hole 3.8. The item's hub-wrap strain is 1.56 %.

A uniform-curvature spiral gives `k = E h t³ / (12 L)` and bending strain `ε = (t/2) θ / L`, with printed PETG at `E` =
2.1 GPa (1.9 to 2.2):

| Wound (arbor turns past relaxed) | Torque | Strain |
|---|---|---|
| 0.41 | 8.0 N·mm (design minimum) | 0.17 % |
| 1.5 | 29.2 N·mm | 0.62 % |
| 2.0 | 39.0 N·mm | 0.83 % |
| 2.25 (full wind) | 43.9 N·mm | 0.93 % |
| 2.57 | coils packed on the hub (geometric limit, estimate) | 1.06 % |

`k` = 3.10 N·mm/rad. The pendulum (about 20 g effective at 0.16 m) swings 3.81° and stores 0.069 mJ. With an assumed Q of 10
to 30, a 40 % escapement efficiency and a 90 % gear efficiency, the great wheel arbor needs 0.7 to 2.1 N·mm; the design
minimum of 8 N·mm allows 4 to 11 times that for printed friction. These losses are estimates, not measurements.

Run time: from full wind (2.25 turns) down to 8 N·mm (0.41 turns) the arbor and great wheel turn 1.84 times. That is
1.84 × 48/13 = 6.8 escape wheel turns, 204 pendulum periods, **about 2.9 minutes** at the default 180 mm pendulum. A spring
this size cannot do better without breaking the strain limit or the pinion clearance.

Printing: PETG (or nylon), as `models/spiralspring` says. Full-wind strain 0.93 % is about a quarter of PETG's yield strain
(3.5 to 4 %). PETG creeps under a held load, so the README says to let the spring run down after a showing. PLA (`E` ≈
3.5 GPa) would give 73 N·mm at full wind but is brittle near 2 % and creeps at room temperature, so it is not recommended.
The strip is 1.5 mm, four 0.375 mm lines, with 1.1 mm between coils, 6 mm tall, printed flat with no support. Full-wind
torque gives 10.7 N·mm at the escape wheel, about 20 to 60 times the estimated need. The pendulum's amplitude grows with
drive, and above an anchor amplitude of about 7.5° the pallets reach the wheel's roots (1.23 mm clearance at 6°, close to 0 at
7.5°). The model cannot predict the running amplitude, so the first print must check it at full wind.

### What the spring replaces

| Before | After |
|---|---|
| Drum (barrel, cone, flange, boss, pawl layer) | **Pawl Carrier**: the pawl layer and core plus a short boss; no barrel, cone or flange |
| Cord and weight (README text only) | **Mainspring** on the great wheel arbor; no cord or weight text |
| Drum boss that stopped the arbor at the front bearing | **Spring Spacer**: a pressed tube from the spring hub to the front bearing |
| none | **Spring Anchor Pin**: pressed into the front plate's spine; carries the spring's eye |

The great wheel, the internal ratchet, the pawl shapes and their engagement, the 13/48 pair, the knob and the great wheel
arbor (same meridian and flats) do not change.

### Pawl Carrier

Carrier local frame: axis through the origin (moved onto `G`), D-flat of its bore on `+X`.

1. Pawl layer `z ∈ [5.25, 9.25]`: core circle `r 7.5`, unioned with two pawls.
   Pawl `j` (`g = 90° - δb + 180° j`, with the offset `δb` of Pose), one closed loop (angles in the carrier frame):
   `(7.0∠g+10°)` → arc `r 7.0` cw → `(7.0∠g)` → line → `(9.0∠g)` → arc `r 9.0` cw → `(9.0∠g-120°)` → line →
   `(10.5∠g-120°)` → line → `(12.5∠g-116°)` → arc `r 12.5` ccw → `(12.5∠g-112°)` → line → `(10.5∠g-112°)` → arc
   `r 10.5` ccw → `(10.5∠g+10°)` → line → `(7.0∠g+10°)`.
   Contact face: radial line at `g - 112°` from `r 10.5` to `12.5`, facing counter-clockwise.
2. Boss `r 3.7`, `z ∈ [8.75, 11.0]` (0.5 overlap into the layer).
3. Union, then cut the D press bore (`r 2.55`, flat at `x = 2.05`), tool `z ∈ [4.25, 12]`.

Facts: the arm (`r 9 to 10.5`, 120° long, 19.4 mm, 1.5 thick) has a 1.5 mm slot to the core and 1.0 mm to the ratchet tips.
Winding pushes the pawl tip in from `r 12.5` past the land at `r 11.5`, a 1.0 mm deflection plus 0.3 mm margin. Bending strain
is about 0.8 %. Under load the arm is in tension.

Each pawl's contact face lies 1.2° (0.25 mm at `r 12`) clockwise of a ratchet radial face at great-wheel-local `+20.8°`
(`DrumOffset`). The pawl tooth occupies `[30k - 5.2°, 30k - 1.2°]` at `r 12.5`, inside the gap `[30k - 6.94°, 30k]`.

### Spring Spacer and Spring Anchor Pin

Spring Spacer: disc `r 3.7`, `z ∈ [17.5, 27.75]`, D press bore (`r 2.55`, flat `x 2.05`), pressed on the arbor's flat.
Spring Anchor Pin: round rod `r 1.5` at `Q = (0, G_y - 34.0015) = (0, -79.7515)`, `z ∈ [10.25, 30.95]`, pressed into a blind
socket in the front plate's inner face: `r 1.55`, `z ∈ [28, 31]` (the tool spans `z ∈ [27, 31]`). Its tip is 1 mm behind the
eye's lower face. `Q` is on the front plate's spine (`|x| ≤ 4`, between the `G` hub and the bottom rim), leaving 2.45 mm of
wall beside the socket and 3.0 mm to the window corner, and on the centre line, so the eye hangs straight below the arbor.
The pin stops 1.75 mm in front of the great wheel's face (`z 8.5`), whose radius 37.48 covers `Q`. The eye cannot slide off:
the spring hub is held on the arbor between the carrier and the spacer.

### Arbor stack

Great wheel arbor at `G = (0, -45.75)`, z from the back:

| z | Item |
|---|---|
| `[-5, 0]` | back bearing |
| `[0.25, 1.25]` | arbor collar |
| `[1.5, 8.5]` | great wheel, free on the round arbor (pocket floor `z 5.0`) |
| `[5.25, 9.25]` | Pawl Carrier pawl layer |
| `[8.75, 11.0]` | Pawl Carrier boss `r 3.7` (0.5 overlap into the layer) |
| `[11.25, 17.25]` | Mainspring (hub and coils), `Height` 6 |
| `[17.5, 27.75]` | Spring Spacer `r 3.7` |
| `[28, 33]` | front bearing |
| `[34.25, 37.25]` | winding knob |

The Pawl Carrier and the Spring Spacer are press fits on the arbor flat. The spring hub slides on the flat with 0.25 clearance,
and its two faces stand 0.25 from the carrier boss and from the spacer. Assembly from the arbor's front end: the great wheel;
the Pawl Carrier, pressed until the great wheel turns freely with about 0.25 play; the Mainspring; the Spring Spacer, pressed
until the spring hub turns freely by about 0.25. Then the arbor goes into the frame, the spring's eye goes onto the anchor pin,
and the knob is pressed on in front. The arbor's flat span `[5.25, 27.75]` already covers the carrier, the spring hub and the spacer.

### Spring ends

The inner end is the item's hub: its D bore (arc `r 2.75`, flat 2.25 from the axis on local `+Y`) sits on the arbor's flat, and
the flat carries the torque. The outer end is the item's eye (hole 3.8) on the Spring Anchor Pin.

### Spring pose

The Mainspring is built by `spiralspring.Build` with `Placement` origin `(G_x, G_y, 11.25)`, `U = (cos γs, sin γs, 0)`,
`V = (-sin γs, cos γs, 0)`, where `γs = -90° - ∠E = 66.7916°` puts the eye straight below `G` at `Q`. The spring is fixed:
it does not turn with `Phase`. The great wheel arbor, Pawl Carrier, Spring Spacer and winding knob turn about `G` by
`θarb = θg0 + 20.8° + δb + (φ - 0.05°)·13/48`, with the constant `δb = 41.7639°`. `δb` aligns the arbor's flat with the
spring's bore flat (world angle `γs + 90° = 156.7916°`) at the default `Phase` 0.25, where the old drum and arbor pose is
115.0277°. `Layout` computes `δb` from `DefaultParams()` and `Derive`, not from a literal. The Pawl Carrier's local frame has its
bore flat on `+X`, and its pawls are drawn at `g = 90° - δb` and `270° - δb`, so each pawl stays at great-wheel-local `+20.8°`
from a ratchet face. Across `Phase ∈ [0, 1)` the arbor turns at most 3.25° (one period × 13/48) against the fixed spring, and the
arbor's flat corners can turn 11.0° inside the spring's bore before touching it, so no phase overlaps.

### Winding knob

Circle `r 9` about `G`, `z ∈ [34.25, 37.25]` (area `π 9² ≈ 254 mm²`, no chamfer; from the front it hides no more than the front plate's `r 10` hub ring at `G`), D press socket (`r 2.55`, flat `x 2.05`). Pressed on the great
wheel arbor's front end.

## Arbors, bearings and axial layout

Arbor diameter 5.0: `docs/model-pipeline.md` Part clearances makes a shaft through a purchased bearing's bore equal
to the bore, with no clearance and no fit class. Flat depth `Sf = 0.5` on `+X` (flat plane `x = 2.0`). Collars,
spacer bosses, the shoulder and the cap are `r 3.7` (diameter 7.4 = 5.0 + 2·1.2, the planetary gearbox's
`collarMargin`; a 605 inner race is about 8 mm across). They bear on the bearing's inner ring only.

Arbors are full revolves of a staircase meridian `(r, z)` about their axis. Flats are cut with box tools
`x ∈ [2.0, 3.5]`, `y ∈ [-3.5, 3.5]` over the listed spans; the boxes of one arbor are unioned and cut once. Probe:
revolve and box cut work on the new pin.

| Arbor | Axis | Meridian `(r, z)` | Flat spans (box z) |
|---|---|---|---|
| Escape Arbor | `O` | `(0,-5) (2.5,-5) (2.5,25.05) (3.7,25.05) (3.7,27.75) (2.5,27.75) (2.5,33) (0,33)` | `[0.5, 25.0]` |
| Pallet Arbor | `P` | `(0,-5) (2.5,-5) (2.5,19.95) (3.7,19.95) (3.7,20.95) (2.5,20.95) (2.5,37.2) (0,37.2)` | `[21.0, 27.75]`, `[34.25, 38.2]` |
| Great Wheel Arbor | `G` | `(0,-5) (2.5,-5) (2.5,0.25) (3.7,0.25) (3.7,1.25) (2.5,1.25) (2.5,37.2) (0,37.2)` | `[5.25, 27.75]`, `[34.25, 38.2]` |

Bore outlines (D = circular arc plus chord, flat on `+X`):

| Fit | Shape | Used by |
|---|---|---|
| press (0.05 radial, Part clearances press-fit rule) | D: arc `r 2.55`, chord at `x = 2.05` | escape wheel, anchor, pawl carrier, spring spacer, crutch, knob, cap, front-plate suspension socket |
| Regular (0.35 + 0.05 margin) | round, diameter 5.8 | great wheel (turns freely on the round arbor when winding) |

Axial stacks (z):

| Item | Escape (`O`) | Pallet (`P`) | Great wheel (`G`) |
|---|---|---|---|
| back bearing | `[-5, 0]` | `[-5, 0]` | `[-5, 0]` |
| collar | `[25.05, 27.75]` | `[19.95, 20.95]` | `[0.25, 1.25]` |
| parts | escape wheel `[0.25, 25]` | anchor `[21, 27.75]` | great wheel `[1.5, 8.5]`, carrier `[5.25, 11.0]`, spring `[11.25, 17.25]`, spacer `[17.5, 27.75]` |
| front bearing | `[28, 33]` | `[28, 33]` | `[28, 33]` |
| outside | none (ends at 33) | crutch `[34.25, 37.25]` + pin to 43.25 | knob `[34.25, 37.25]` |

- Escape wheel: rear boss `r 3.7`, `z [0.25, 2.0]`; pinion `[1.5, 22.5]`; wheel `[22, 25]`. Pressed on from the
  arbor's rear end until its front face is 0.05 from the collar (press-fit gap). The arbor and wheel then move as one:
  the collar stops 0.25 from the front bearing, the rear boss 0.25 from the back bearing.
- Anchor: pressed on from the front until its rear face is 0.05 from the pallet collar; its boss stops 0.25 from the
  front bearing. The crutch, pressed on the front end, stops the arbor backward 0.25 from the front plate.
- Great wheel turns freely on the round arbor, 0.25 from the collar and 0.25 from the pawl carrier's pawl layer. The carrier is
  pressed on the flat to that position (README: press until the great wheel just turns freely); its boss stops 0.25
  from the front bearing.
- Print orientations (README): escape wheel wheel-face down; anchor rear face down; pawl carrier pawl layer down; mainspring flat, in PETG; great wheel
  rear face down; crutch and pendulum rear face down; arbors on end (1.2 mm collar ledge).

Bearings: seven `ballbearing.Params605ZZ()` envelopes, axis `+Z`, lower cap at: back `O, P, G` `z -5`; front `O, P, G`
`z 28`; pendulum `S` `z 39`. Pockets: diameter `14 + 2·0.05 = 14.1`, 5 deep.

## Frame plates and fasteners

### Open skeleton plates

Both plates are open frames, mirror-symmetric about `x = 0`: a rim, a centre spine 8 mm wide (behind the 8 mm pendulum
rod), and a hub ring of radius 10 around each bearing. Two windows, mirror images of each other, open the rest.

| Item | Back plate | Front plate |
|---|---|---|
| outline | rounded rectangle `x ∈ [-58, 58]`, `y ∈ [-96, 98]`, corner `r 10` | `x ∈ [-58, 58]`, `y ∈ [-96, 82]`, corner `r 10` |
| z | `[-6, 0]`, pillars to `28` | `[28, 34]` |
| windows | two (Window loop) | the same two |
| mounting holes | `r 2.25` at `(-40, 90)` and `(40, 90)` | none |
| lip bores, pockets | lip bore `r 5` and pocket `r 7.05` at `O, P, G` | as the back plate |
| other | pillars `r 5` | suspension socket at `S` (in the top rim) |

Pillars, and the bolt sites in this order: `(-50, 74)`, `(50, 74)`, `(-50, -88)`, `(50, -88)`. Each pillar keeps 1 mm of
rim between its edge and a window (`x 45` against `x 44`; `y -83` against `y -82`), and lies 2.17 mm inside the
outline's corner arc.

#### Window loop

Right window (`sx = +1`); the left window is its mirror (`x → -x`). `P = (0, 56.5685)`, `O = (0, 0)`, `G = (0, -45.75)`,
`h = sqrt(10² - 4²) = 9.1652`, `fx = sqrt(10² - (62 - 56.5685)²) = 8.3958`. One closed loop of 6 lines and 5 arcs, in order:

1. line `A (4, -82) → B (38, -82)`
2. arc `r 6`, centre `(38, -76)`, `B → C (44, -76)`
3. line `C → D (44, 56)`
4. arc `r 6`, centre `(38, 56)`, `D → E (38, 62)`
5. line `E → F (fx, 62)` = `(8.3958, 62)`
6. arc `r 10` about `P`, `F → H (4, P_y - h)` = `(4, 47.4033)`, clockwise (the hub ring bulges into the window)
7. line `H → I (4, h)` = `(4, 9.1652)`
8. arc `r 10` about `O`, `I → J (4, -9.1652)`, clockwise
9. line `J → K (4, G_y + h)` = `(4, -36.5848)`
10. arc `r 10` about `G`, `K → L (4, G_y - h)` = `(4, -54.9152)`, clockwise
11. line `L → A`

Material this leaves: spine `|x| ≤ 4` from `y -82` to `62`; hub rings `r 10` about `P, O, G` (2.95 mm round each 7.05 mm
pocket); top rim `y ∈ [62, 82]` (front) or `[62, 98]` (back), which holds `S` and the mounting holes; bottom rim
`y ∈ [-96, -82]`; side rims `|x| ∈ [44, 58]`.

#### Construction

The plate section is one sketch: the outline, both window loops, the three lip bores `r 5` at `O, P, G` and (back plate)
the mounting holes, extruded once to 6 mm.
Do not build the plate as a union of rim, spine and hub prisms: Decad refuses a union of coplanar prisms.

1. One sketch at the plate's low face: the rounded-rectangle outline, the right window, the left window, the three lip circles `r 5` and,
   on the back plate, the mounting circles. Profile with 5 holes (front) or 7 holes (back). Extrude 6.
2. Back plate: union the four pillars `r 5`, `z ∈ [-0.5, 28]`, in one call.
3. Cut one tool, the union of: pockets `r 7.05` (back `z ∈ [-5, 1]`, front `z ∈ [27, 33]`); per site `m2fastener.HoleTool` and `CavityTool` (back) or `HoleTool` and `PocketTool` (front);
   on the front plate the suspension socket, D press outline about `S` (`r 2.55`, chord at `x = 2.05`) over `z ∈ [31, 35]`.
4. One `edgefinish.Chamfer(ctx, body, +Z, {low, low + 6}, keep, edgefinish.MinChamfer)`. `keep` takes outer loops, inner
   loops of more than 4 edges (the windows; the 2-edge D socket has faceted edges that `edgefinish` cannot name), and
   single-edge `decad.Circle3` loops of radius 7.05 (the seat rims, inner face) or 5 (the lip-bore rims, outer face)
   (±0.01) centred within 0.01 of `O`, `P` or `G`.

### Fasteners (`AGENTS.md` Fasteners)

Use the shared `internal/m2fastener` package; do not size pockets, holes or cavities by hand. One site per pillar:

Each pillar has one site: head top at `(px, py, 34)`, direction `-Z` toward the tip, nut corners on `+X`, bolt
length 10, nut cavity beginning 8 mm from the head top.

| Item | Value (from `m2fastener`) |
|---|---|
| bolt | `PlaceBolt(site)`: M2 × 10, head top flush at `z 34` |
| head pocket | `PocketTool(site)`: `r 2.2`, 2.0 deep (`z [32, 34]`); 4 mm of plate under it |
| clearance hole | `HoleTool(site)`: `r 1.3`, from 1 mm in front of the plate to 1 mm past the tip: `z [21, 35]`; cuts both plates |
| nut | `PlaceNut(site)`: `z [24.2, 25.8]`, corners on `+X` |
| nut cavity | `CavityTool(site)`: `r 2.51`, `z [24, 26]`; 2 mm roof to the pillar top; wall 2.49 |
| bolt tip | `z 22` (`Tip() = 12` from the head top), inside the pillar hole |
| print pause | back plate prints rear face down; pause at 32 mm above the bed (`z 26`), drop the four nuts in, resume |

Probe: the four sites' tools cut both plates and `PlaceBolt`/`PlaceNut` build on the new pin.

Pillar tops are clamped against the front plate's inner face by the bolts. That is a stated contact, the same as
the planetary gearbox's covers on its ring.

## Parameters

| Field | Meaning | Default | Range |
|---|---|---|---|
| `PendulumLength` | pivot to bob centre, mm | 180 | 120 to 250 |
| `Amplitude` | anchor amplitude, radians | 5° | `(LiftAngle + LockAngle)/2 + 1°` to 6° |
| `LockAngle` | `κ`, radians | 1.5° | 1.25° to 1.5° |
| `LiftAngle` | `λ`, radians | 2.5° | 2° to 3° |
| `Phase` | pose, fraction of a period | 0.25 (pendulum at rest) | 0 up to but not including 1 |
| `Backlash` | gear tooth thinning, mm | 0.15 | 0 to 0.3 |
| `Placement` | frame mapping the model frame into the world | nil (identity) | a valid frame |

- `Validate` rejects nonfinite values, each range above with an error naming the field, and an invalid `Placement`.
  Zero values are not default requests; `DefaultParams` supplies defaults.
- Every other dimension is a fixed constant of this doc. `LockAngle`, `LiftAngle` change only the anchor outline;
  `PendulumLength` only the pendulum outline; `Phase`, `Amplitude` only the pose; `Backlash` only the gears.

Exported API (package `escapement`):

| Symbol | Meaning |
|---|---|
| `DefaultParams() Params`, `(Params) Validate() error` | defaults and checks |
| `Layout(p Params) (Geometry, error)` | every derived value and point of Escapement geometry, Drive train, axes |
| `Motion(p Params) (State, error)` | closed-form state at `p.Phase` (Motion and pose) |
| `NewDocument(opts ...decad.DocumentOption) *decad.Document` | `decad.New` with `decad.WithChainTolerance(0.1 mm)` first |
| `Build(ctx, doc, p) (*Movement, error)` | builds and poses all bodies; validates first; errors are contextual |
| `(*Movement) Named() []NamedBody` | every body with its display name, in the Bodies table order |

`State`: `Phase`, `AnchorAngle` (ψ), `WheelAngle` (φ), `PendulumAngle` (β), `GreatWheelAngle` (= φ·13/48,
counter-clockwise), `Locked` (`"entry"`, `"exit"` or `""` during impulse). `Movement`: one field per body (Bodies),
`Sketches []*sketch.Sketch`, `State`.

## Bodies and containers

| # | Body name | Kind | Container path | Motion |
|---|---|---|---|---|
| 1 | Back Plate | printed | Frame | fixed |
| 2 | Front Plate | printed | Frame | fixed |
| 3-9 | Back Escape Bearing, Back Pallet Bearing, Back Great Wheel Bearing, Front Escape Bearing, Front Pallet Bearing, Front Great Wheel Bearing, Pendulum Bearing | 605ZZ | Bearings | fixed; pendulum's turns with the pendulum |
| 10 | Escape Arbor | printed | Arbors | φ about `O` |
| 11 | Pallet Arbor | printed | Arbors | ψ about `P` |
| 12 | Great Wheel Arbor | printed | Arbors | with the pawl carrier |
| 13 | Escape Wheel (wheel + pinion + rear boss) | printed | Escapement / Escape Wheel | φ |
| 14 | Anchor | printed | Escapement / Anchor | ψ |
| 15 | Crutch | printed | Pendulum Assembly / Crutch | ψ |
| 16 | Pendulum | printed | Pendulum Assembly / Pendulum | β about `S` |
| 17 | Suspension Pin | printed | Pendulum Assembly / Suspension | fixed |
| 18 | Suspension Cap | printed | Pendulum Assembly / Suspension | fixed |
| 19 | Great Wheel | printed | Drive / Great Wheel | φ·13/48 ccw about `G` |
| 20 | Pawl Carrier | printed | Drive / Pawl Carrier | arbor turn `θarb` |
| 21 | Mainspring | printed (PETG) | Drive / Mainspring | fixed |
| 22 | Spring Spacer | printed | Drive / Spring Spacer | arbor turn |
| 23 | Spring Anchor Pin | printed | Drive / Spring Anchor | fixed |
| 24 | Winding Knob | printed | Drive / Winding Knob | arbor turn |
| 25-28 | Bolt 1..4 | M2 × 10 | Bolts | fixed |
| 29-32 | Nut 1..4 | M2 | Nuts | fixed |

Bolt/nut `i` sits at pillar `i` in the order `(-50, 74)`, `(50, 74)`, `(-50, -88)`, `(50, -88)`. Use
`Container.ChildNamed` for kinds with several bodies; `doc.VerifyContainers().Err()` must be nil.

## Fits

Classes from `docs/model-pipeline.md` Part clearances: Tight 0.2, Regular 0.35, Loose 0.5 (built gap = class + 0.05
margin, or larger). Nominal parts (bearings, bolts, nuts) keep nominal size.

| ID | Parts | Fit | Class | Built gap |
|---|---|---|---|---|
| T1 | each arbor in its two bearings; suspension pin shank in the pendulum bearing | radial | none: diameter = bore (Part clearances) | 0 |
| T2 | collars, spacer bosses, rear boss, shoulder, cap to bearing faces | axial | Tight | 0.25 |
| T3 | escape wheel, anchor, Pawl Carrier, Spring Spacer D bores on their arbors (parts that turn with the arbor) | radial and at the flat | press fit | 0.05 |
| T4 | great wheel rear face to its collar; Pawl Carrier pawl layer to the ratchet pocket floor (the great wheel turns relative to both) | axial | Tight | 0.25 |
| T4p | escape wheel front face to escape collar; anchor rear face to pallet collar (pressed parts, press stop) | axial | press fit | 0.05 |
| T5 | crutch and knob rear faces to the front plate's front face | axial | Tight | 0.25 |
| T6 | crutch pin in the pendulum slot | each side | Tight | 0.25 |
| T7 | carrier pawl contact face to ratchet radial face at the build pose (working contact) | angular | Tight | 0.25 at `r 12` |
| R1 | great wheel bore (5.8) on the great wheel arbor (5.0) | radial | Regular | 0.40 |
| L1 | Mainspring to the escape wheel (replaces the drum flange) | axial | Loose | 4.75 |
| L2 | Spring Spacer (`r 3.7`) to escape wheel tips; great wheel arbor (`r 2.5`) to escape wheel tips | radial | Loose | 2.05; 3.25 |
| L3 | pendulum to crutch; pendulum to knob | axial | Loose | 0.50 |
| L4 | anchor to front plate; escape wheel to front plate | axial | Loose | 2.0; 3.0 |
| L5 | pawl arm to ratchet tips; pinion tips to the pawl layer | radial | Loose | 1.0; 22 |
| L6 | shoulder (`r 3.7`) in the pendulum lip bore (`r 5`) | radial | Loose | 1.30 |
| E1 | anchor to escape wheel apart from the working contact, over the swing | 2D | Loose | ≥ 0.6 (escapement check) |
| E2 | pallets to the root circle at `A = 6°` | 2D | Loose | ≥ 1.2 |
| E3 | pallet tops and arms to the tip circle | 2D | Loose | ≥ 2.3 |
| P1 | crutch, knob on arbor ends; suspension pin rear peg in the front plate; cap on the pin | press | press-fit rule | 0.05 |
| B1 | each 605ZZ in its pocket | radial | bearing-seat exception | 0.05 |
| G1 | pinion / great wheel | tip-to-root | gear rule | ≥ 0.5 at 5 angles |
| S1 | Mainspring hub D bore on the great wheel arbor | radial, at the flat | Tight | 0.25 |
| S2 | Mainspring eye on the Spring Anchor Pin | radial | Regular | 0.40 |
| S3 | Mainspring hub faces to the carrier boss (`z 11.0`) and to the spacer (`z 17.5`) | axial | Tight | 0.25 |
| S4 | Spring Anchor Pin in the front plate socket | press | press | 0.05 |
| S5 | Spring Spacer front face to the front bearing | axial | Tight (T2) | 0.25 |
| S6 | Mainspring coils (relaxed) to the pinion tips | radial | Loose | 2.52 |
| L7 | Mainspring to the great wheel face; to the pawl layer; anchor pin tip to the great wheel face | axial | Loose | 2.75; 2.0; 1.75 |
| L8 | Mainspring coils (other than the eye) to the anchor pin | radial | Loose | ≥ 0.5 (the item keeps the eye `g` = 1.1 from the previous turn) |

L3 (0.50), L4 (2.0; 3.0) and L5 (1.0; 22.02) do not depend on the arbor size and are unchanged.

Stated contacts (no class): pillar tops on the front plate; suspension pin shoulder on the front plate's front face;
each bolt head on its pocket floor; bolt in its own nut (nominal exception). The escape tooth on a pallet and the
pawl on the ratchet are working contacts. The build pose backs the wheel off by 0.05° (Motion and pose) so the
bodies do not touch.

`models/README.md` label: **Mixed**.

## Edge finishing

Large faces (≥ 400 mm²) and their chamfers (0.5, 45°), all through `internal/edgefinish.Chamfer` (one call per part):

| Part | Chamfered | Kept sharp (reason) |
|---|---|---|
| Back plate | outer loop and both windows on both faces; pocket rims (seat chamfer); lip-bore rims on the rear face | mounting holes, pillar holes (< 20 mm) |
| Front plate | outer loop and both windows on both faces; pocket rims; lip-bore rims on the front face | bolt holes, head pockets, suspension socket (< 20 mm) |
| Escape Wheel | 6 window loops on both faces | tooth outline: pieces 2 to 8 mm, corners, no 20 mm run |
| Anchor | whole outer loop of both faces, pallets included | none. The wheel spans `z [22, 25]`, 1 mm inside each anchor face, so the 0.5 chamfer leaves the working faces whole |
| Pendulum | outer loop of both faces; pocket rim; lip-bore rim on the rear face | slot (fit T6) |
| Great Wheel | none | front face outer loop = tooth flanks; ratchet pocket loop = functional teeth |
| Pawl Carrier, Spring Spacer, Spring Anchor Pin, crutch, knob, pin, cap, arbors | none | faces under 400 mm² (crutch 321, knob 254) |
| Mainspring | its caps, by `models/spiralspring` | the item chamfers its own caps |

Test: each chamfered part weighs less than its unchamfered build by the bevel volume. Each kept-sharp loop keeps
its edges (face count).

## Motion and pose

Closed form for `Motion(p)` (`t = Phase`, `A = Amplitude`):

- `ψ(t) = -A cos 2πt`. `t ∈ [0, 0.5]` swings `+`; `(0.5, 1)` swings `-`.
- `φ_E = (90° - ∠I_L) mod τ` = 0.99425°; `φ_X = φ_E + τ/2` = 6.99425°.
- `Q_E(ψ)`: the point where the tip circle meets the segment from `P + rot(C_E, -ψ)` to `P + rot(F_E, -ψ)`
  (solve `|C + s (F - C)| = R`, `s ∈ [0, 1]`). `Q_X(ψ)` likewise with `C_X`, `F_X`.
- Swing `+`: `ψ < ψ_Eu` → `φ = φ_E` (entry locked). `ψ_Eu ≤ ψ ≤ ψ_Ef` → `φ = φ_E + ∠I_L - ∠Q_E(ψ)`.
  `ψ > ψ_Ef` → `φ = φ_X` (exit locked).
- Swing `-`: `ψ > ψ_Xu` → `φ = φ_X`. `ψ_Xf ≤ ψ ≤ ψ_Xu` → `φ = φ_X + ∠I_R - ∠Q_X(ψ)`. `ψ < ψ_Xf` → `φ = φ_E + τ`.
- `β = atan2(45 sin ψ, 14 + 45 cos ψ)`. `GreatWheelAngle = φ · 13/48`.
- Drop on the wheel: `τ/2 - (∠I_L - ∠F_L)` = 1.98848° (1.388 mm).

`Build` pose (all parts built in local frames, then one rigid transform each, composed with `Placement`):

| Part | Transform |
|---|---|
| Escape Wheel, Escape Arbor | rotate about `O` by math angle `-(φ - 0.05°)` |
| Anchor, Pallet Arbor, Crutch | rotate about `P` by `-ψ` |
| Pendulum, Pendulum Bearing | rotate about `S` by `-β` |
| Great Wheel | rotate about `G` by `θg0 + (φ - 0.05°)·13/48` |
| Pawl Carrier, Spring Spacer, Great Wheel Arbor, Winding Knob | rotate about `G` by `θg0 + 20.8° + δb + (φ - 0.05°)·13/48` (Spring pose) |
| Mainspring, Spring Anchor Pin | fixed |
| everything else | fixed |

The escape wheel is built with tooth 0 on `+Y` and the pinion's tooth 0 at -90°. Arbor flats and part D bores are
on `+X` in each local frame.

## Geometry checks

Tests live in `models/escapement`. Build each parameter case once per test binary (`sync.Once` or `TestMain`).
Never pass meshes or bodies to `require.Equal`; report counts, names and scalars only.

| Test | Case(s) | Required result |
|---|---|---|
| `TestValidate` | each field below/above its range, NaN, invalid `Placement` | error naming the field; `Build` returns it before touching the document |
| `TestLayout` | default | every default number in Escapement geometry and Drive train to 1e-4 |
| `TestMotion` | default; `κ 1.25°, λ 3°, A 6°` | 4000 samples: φ never decreases; φ(1) - φ(0) = τ to 1e-9; φ constant while locked; jumps only at `ψ_Ef`, `ψ_Xf`, each equal to the drop; pendulum and great wheel relations hold |
| `TestActionAnalytic` | default | polygon checker on analytic outlines reproduces Numerical verification within 0.02 mm / 0.05° |
| `TestEscapementAction` | default; `{κ 1.25°, λ 3°, A 6°}`; `{κ 1.5°, λ 2°, A 6°}` | see below |
| `TestSketches` | 3 branch cases (Sketch constraint plan) | every `Movement.Sketches` entry fully constrained |
| `TestInventory` | default, Phase 0.3 | 32 bodies, all solid, unique names, Named order = Bodies table; containers and membership as Bodies table; `VerifyContainers().Err()` nil |
| `TestGearMesh` | default; `Backlash 0` | pinion and great wheel outlines at `z 3`, 5 samples over one pinion pitch with the mesh relation: tip-to-root ≥ 0.5 both ways, no overlap |
| `TestRatchet` | default | at `z 7` outlines: pawl-to-face gap 0.25 ± (0.02 + OffSurface); carrier turned ccw 2.5° relative to the wheel overlaps (drive engages); turned cw 2.5° does not; pawl arm to ratchet tips ≥ 0.95 |
| `TestCoupling` | Phase 0, 0.25, 0.5, 0.75 | crutch pin to slot sides ≥ 0.2 each side; pendulum not overlapping crutch, knob or plate |
| `TestFits` | default | each Fits row's lower bound (`internal/meshgap` or `SectionOutline.Clearance`) ≥ class minimum; press and seat rows 0.05 ± bound; T1: shaft radius at each bearing seat 2.5 ± bound |
| `TestFasteners` | default | each head top at `z 34` ± 1e-6 and in its pocket; each nut inside its cavity bounds; bolt tip inside the pillar; no bolt or nut overlaps the plate it passes; 1 mm of material round pockets and cavities |
| `TestEdgeFinishing` | default | Edge finishing table (volume drop per part; sharp loops keep their edges) |
| `TestPlacement` | origin `(10, -5, 7)`, `U = +Y`, `V = +Z` | same volumes as unplaced; bounds follow the frame |
| `TestPoseNoOverlap` | Phase 0, 0.1, 0.35, 0.6, 0.85 | no two bodies overlap (escape wheel/anchor, pinion/great wheel, carrier/great wheel by outline; others by meshgap) |
| `TestExploded` (render module) | exploded offsets | no two exploded parts overlap |

`TestEscapementAction` requirements, on `SectionOutline` (tolerance 0.02) of the real Escape Wheel and Anchor at
`z 23.5`:

- one tooth per period (±0.05°);
- 2 drops per period, both landing with the tip within `0.03 + OffSurface` of the lock radius about `P`;
- lock and lift within 0.1° of `κ` and `λ`; drop ≥ 1.30 mm at the tip;
- recoil ≤ (sum of both OffSurface) / R;
- E1 ≥ 0.5, E2 ≥ 1.0, E3 ≥ 2.0 (Fits).

Escapement checker (a test helper): a quasi-static simulation on generic closed polylines. Prune segments to the annulus `[Rr - 1, R + 1]` about `O` within ±30° of `I_L`/`I_R`. 2000 steps per
period, 3 periods, measure periods 2 and 3. Advance by 0.02° then bisect to 1e-9 rad.

## Open frame and centre line

When the clock hangs, the parts that set its time stand on one plumb line. That line runs through the suspension `S`, the
pallet arbor `P`, the escape arbor `O` and the great wheel arbor `G`. The pendulum hangs straight down it at rest, and the
frame is mirror-symmetric about it. The default pose shows the pendulum at rest, so every still image reads vertical. The
model is for a wall mount and has no stand.

The Fits table is unchanged. Only positions moved: `G`, the pillars, the bolt sites and the mounting holes. Every distance
from `O` and `P` to the drive parts is the same, because the drive turned about `O` by 30°.

Front view (from `+Z`, pendulum at rest), through the front plate's windows:

| Part | Visible | Hidden by |
|---|---|---|
| crutch, crutch pin, pendulum, suspension cap | all (in front of the plate) | none |
| anchor | pallets, impulse and locking faces, arms | the hub at `P` (`r 10`) covers its hub |
| escape wheel | all 30 teeth except near `(0, ±40)`; spokes and windows | the spine strip `|x| ≤ 4` (behind the pendulum rod), the hub at `O` |
| great wheel | teeth and face, except the spine strip, the hub at `G` and its lowest 1.2 mm (bottom rim) | none |
| ratchet and pawls | ratchet teeth `r 11.5` to `13.5`, pawl teeth `r 10.5` to `12.5` | the hub ring and knob inside `r 10` |
| mainspring | all coils and the eye | the hub ring and the knob inside `r 10` |
| carrier boss, spacer, pinion | none | the hub at `G`; the escape wheel's hub (pinion) |

Back view (from `-Z`) through the back plate's windows: the great wheel's back face and teeth, the pinion ring (`r 10` to
`11.23` beyond the hub, except the spine strip), and the escape wheel and anchor behind them. The pinion and great wheel
mesh lies on the centre line, so the spine strip covers it. It shows in the angled and side views.

Acceptance cases added for the spring drive: `TestSpringLayout` (spring `N` 9, pitch 2.6, gap 1.1, active length 1142.3, eye radius 34.0015, `γs` 66.7916°, `δb` 41.7639°, `Q` `(0, -79.7515)`, `SpringRate` 3.10, `FullWindTorque` 43.9, `FullWindStrain` 0.93 %, `RunTurns` 1.84), `TestMainspring`, `TestPawlCarrier`, `TestFrontPlateSpringAnchor` and `TestSpringFits`.

Acceptance cases added: `TestCentreLine` (`O, P, S, G` at `x = 0` to 1e-9; default `Motion` gives ψ = 0 and β = 0; each
plate's bounds are symmetric within the section bound; pillars and mounting holes in mirror pairs), `TestPlateWindows`
(by section outline at mid-plane, the points `(±29.5114, 27.5334)`, `(±28.2168, 26.2738)`, `(±37.1764, 37.3885)`,
`(±40, 0)`, `(±20, 0)`, `(±30, -45.75)`, `(±37, -45.75)`, `(±12.5, -45.75)` and `(±10.8, 0)` are open; the hub rings
at `(±8.5, 0)`, the spine at `(0, 30)`, the rim beside each pillar at `(±47, 74)` and `(±47, -88)` and, on the front plate,
the socket wall at `(0, 75)` are material) and `TestPlateMaterial` (each pocket circle at least 2.0 mm from a window, each
bolt hole at least 3.0 mm, the socket at least 5.0 mm).

## Look-alike parts

This design has no "Look-alike parts" section, so `docs/model-pipeline.md` Look-alike part marks does not apply
and no marks are cut. The parts a person could confuse are the three arbors. Marks cannot be placed on them: a
1.5 mm hole with 1 mm of material each side needs a face wider than 3.5 mm off the axis, and a 5.0 mm arbor has
none. They differ by sight instead. The escape arbor is 38 mm long and the other two are 42.2 mm. The pallet arbor's
collar is mid-length (`z 20`) and the great wheel arbor's is at its rear end (`z 0.75`). The README parts table
states these differences.
