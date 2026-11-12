# Graham escapement

## Purpose and source

Construct a desk-clock-size Graham deadbeat anchor escapement as one assembly: a weight-driven great wheel and drum,
a 30-tooth escape wheel, an anchor, a crutch and a pendulum, all in a two-plate frame on seven purchased 605ZZ
bearings. It is the source for downstream ports. All lengths are in millimeters.

## Choice of escapement

Build a **Graham deadbeat anchor escapement** driven by a falling weight, regulated by a pendulum through a crutch.

- Graham locks on arcs centred on the pallet arbor, so a locked tooth exerts no torque on the anchor (no recoil).
  Small print errors change only the lock depth and the drop, not the action. The tolerance study below keeps
  one tooth per period, a lock on the locking face and a drop of 0.94 mm or more under ±0.2 mm print errors.
- Recoil anchor: simpler, but recoil wears printed teeth and is the less instructive mechanism.
- Swiss lever and pin-pallet: need a balance wheel and hairspring. A printed hairspring is not reliable, and the
  lever, roller, guard pin and banking add tiny parts below comfortable print size.
- Pendulum + crutch + weight is the classic regulator for Graham. Every part prints without supports in the
  orientation given under Arbors (arbors print on end with a 1.2 mm collar ledge).

## Scale and print size

The model is desk-clock size, not a watch-size miniature.

| Item | Size |
|---|---|
| Escape wheel | 80 mm across (tip radius 40), 30 teeth, 3 mm thick, tooth 7 mm tall |
| Anchor | pallets 2.8 mm wide, 5 mm thick; lock depth about 1.0 to 1.1 mm; drop 1.39 mm at the tooth tip |
| Gears | module 1.5: 13-tooth pinion, 48-tooth great wheel (75 mm across) |
| Frame | back plate 122 × 184 mm, front plate 122 × 168 mm, 6 mm thick; 28 mm between plates |
| Movement depth | 51.75 mm, from the back plate's rear face to the pendulum's suspension cap |
| Pendulum | 180 mm from pivot to bob centre by default (120 to 250), bob 50 mm across; 215 mm long overall |
| Bearings | seven purchased 605ZZ (5 × 14 × 5); arbors 5.0 mm, equal to the bearing bore |
| Fasteners | four M2 × 10 bolts and four M2 nuts |

Every part fits a 220 × 220 mm bed. The longest parts are the default pendulum (215 mm overall) and the back
plate (184 mm). A 250 mm pendulum (285 mm overall) needs the diagonal of a 220 mm bed.

## Frame and conventions

- Model frame: origin `O` on the escape arbor axis in the plane of the back plate's inner face. `+X` right, `+Y` up,
  `+Z` toward the viewer (the front). The back plate's inner face is `z = 0`; the wall is on the `-Z` side.
- Lengths in mm. Code angles in radians; this doc gives degrees.
- `math angle` = counter-clockwise from `+X` seen from `+Z`.
- Wheel angle `φ`, anchor angle `ψ` and pendulum angle `β` are **clockwise-positive** seen from `+Z`.
  A part turned by a clockwise angle `a` is rotated by math angle `-a`.
- The escape wheel turns clockwise. The pinion turns with it. The great wheel and the drum turn counter-clockwise.
- Axes (default): `O = (0, 0)`, `P = (0, d) = (0, 56.5685)` pallet arbor, `S = (0, d + 14) = (0, 70.5685)` pendulum
  suspension, `G = 45.75 (cos -60°, sin -60°) = (22.8750, -39.6207)` great wheel arbor.
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

A weight on a cord hangs from the left side of the drum (`x = G_x - 7`). It turns the drum counter-clockwise. Two
flexure pawls on the drum push the internal ratchet of the great wheel. The great wheel drives the 13-tooth pinion
on the escape wheel clockwise. Winding: turn the winding knob clockwise. The drum turns clockwise and the pawls click
over the ratchet. The model includes no cord or weight.

### Gear pair

| Item | Value |
|---|---|
| module `m`, pressure angle | 1.5, 20° |
| pinion `Zp`, great wheel `Zg` | 13, 48; ratio 48/13 |
| centre distance | `m (Zp + Zg)/2 = 45.75`, `G` at math angle -60° from `O` |
| root clearance `c` | `max(0.25 m, 0.5) = 0.5` |
| pinion radii | root `9.75 - 1.5 - 0.5 - 0.02 = 7.73`, tip `9.75 + 1.5 - 0.02 = 11.23` |
| great wheel radii | root `36 - 1.5 - 0.5 - 0.02 = 33.98`, tip `36 + 1.5 - 0.02 = 37.48` |
| tip-to-root | `45.75 - 11.23 - 33.98 = 0.54`, `45.75 - 37.48 - 7.73 = 0.54` (rule: ≥ 0.5) |
| backlash `j` | `Backlash`, default 0.15 per gear → 0.3 play |
| pinion | `spurgear` variant, 13 teeth, thickness 21, `z ∈ [1.5, 22.5]`, no bore (D bore cut after unions) |
| great wheel | `spurgear` variant, 48 teeth, thickness 7, `z ∈ [1.5, 8.5]`, `BoreDiameter 5.8` (round, Regular fit) |
| mesh phase at `φ = 0` | pinion tooth 0 at math angle `θp0 = -60°` (toward `G`); great wheel tooth 0 at `θg0 = 120° + 180°/48 = 123.75°` (gap facing `O`) |

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

### Drum

Drum local frame: axis through `G`, D-flat of its bore on `+X`.

1. Revolve the meridian `(r, z)`: `(0, 8.75) (7, 8.75) (7, 16.75) (11.5, 21.25) (3.7, 21.25) (3.7, 27.75) (0, 27.75)`:
   barrel `r 7`, a 45° cone to the flange `r 11.5`, front face `z 21.25`, spacer boss `r 3.7` to `z 27.75`. The
   flange's front face is `π (11.5² - 3.7²) = 372.5 mm²`, under the 400 mm² of Surface edge finishing (with `r 12`
   and the 7.4 mm boss it would be 409 mm²).
2. Cut the D press bore (`r 2.55`, flat at `x = 2.05`), tool `z ∈ [7.75, 28.75]`.
3. Pawl layer `z ∈ [5.25, 9.25]`: core circle `r 7.5` with a D relief hole (`r 2.8`, flat `x 2.3`, 0.3 mm clear of
   the arbor; the revolve's bore holds the drum), unioned with two pawls.
   Pawl `j` (`g = 90° + 180° j`), one closed loop (angles in the drum frame):
   `(7.0∠g+10°)` → arc `r 7.0` cw → `(7.0∠g)` → line → `(9.0∠g)` → arc `r 9.0` cw → `(9.0∠g-120°)` → line →
   `(10.5∠g-120°)` → line → `(12.5∠g-116°)` → arc `r 12.5` ccw → `(12.5∠g-112°)` → line → `(10.5∠g-112°)` → arc
   `r 10.5` ccw → `(10.5∠g+10°)` → line → `(7.0∠g+10°)`.
   Contact face: radial line at `g - 112°` from `r 10.5` to `12.5`, facing counter-clockwise.
4. Union the pawl layer with the bored revolve.

Facts: the arm (`r 9 to 10.5`, 120° long, 19.4 mm, 1.5 thick) has a 1.5 mm slot to the core and 1.0 mm to the
ratchet tips. Winding pushes the pawl tip in from `r 12.5` past the land at `r 11.5`, a 1.0 mm deflection plus
0.3 mm margin. Bending strain is about 0.8 %. Under load the arm is in tension. Barrel cord length 8.0 mm.

Pose: the drum's local frame sits at great-wheel-local angle `+20.8°`, so each pawl's contact face lies 1.2° (0.25 mm
at `r 12`) clockwise of a ratchet radial face. The pawl tooth occupies `[30k - 5.2°, 30k - 1.2°]` at `r 12.5`. That is
inside the gap `[30k - 6.94°, 30k]`.

### Winding knob

Circle `r 10` about `G`, `z ∈ [34.25, 37.25]`, D press socket (`r 2.55`, flat `x 2.05`). Pressed on the great
wheel arbor's front end.

### Running time (README text)

Escape wheel turn = 30 periods (25.5 s at 180 mm). Drum turn = 48/13 escape turns (94 s). Treat run length per
metre of cord as a README estimate, not a model check.

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
| press (0.05 radial, Part clearances press-fit rule) | D: arc `r 2.55`, chord at `x = 2.05` | escape wheel, anchor, drum revolve, crutch, knob, cap, front-plate suspension socket |
| relief (0.3, not a fit) | D: arc `r 2.8`, chord at `x = 2.3` | drum pawl-layer core |
| Regular (0.35 + 0.05 margin) | round, diameter 5.8 | great wheel (turns freely on the round arbor when winding) |

Axial stacks (z):

| Item | Escape (`O`) | Pallet (`P`) | Great wheel (`G`) |
|---|---|---|---|
| back bearing | `[-5, 0]` | `[-5, 0]` | `[-5, 0]` |
| collar | `[25.05, 27.75]` | `[19.95, 20.95]` | `[0.25, 1.25]` |
| parts | escape wheel `[0.25, 25]` | anchor `[21, 27.75]` | great wheel `[1.5, 8.5]`, drum `[5.25, 27.75]` |
| front bearing | `[28, 33]` | `[28, 33]` | `[28, 33]` |
| outside | none (ends at 33) | crutch `[34.25, 37.25]` + pin to 43.25 | knob `[34.25, 37.25]` |

- Escape wheel: rear boss `r 3.7`, `z [0.25, 2.0]`; pinion `[1.5, 22.5]`; wheel `[22, 25]`. Pressed on from the
  arbor's rear end until its front face is 0.05 from the collar (press-fit gap). The arbor and wheel then move as one:
  the collar stops 0.25 from the front bearing, the rear boss 0.25 from the back bearing.
- Anchor: pressed on from the front until its rear face is 0.05 from the pallet collar; its boss stops 0.25 from the
  front bearing. The crutch, pressed on the front end, stops the arbor backward 0.25 from the front plate.
- Great wheel turns freely on the round arbor, 0.25 from the collar and 0.25 from the drum's pawl layer. The drum is
  pressed on the flat to that position (README: press until the great wheel just turns freely); its boss stops 0.25
  from the front bearing.
- Print orientations (README): escape wheel wheel-face down; anchor rear face down; drum pawl layer down; great wheel
  rear face down; crutch and pendulum rear face down; arbors on end (1.2 mm collar ledge).

Bearings: seven `ballbearing.Params605ZZ()` envelopes, axis `+Z`, lower cap at: back `O, P, G` `z -5`; front `O, P, G`
`z 28`; pendulum `S` `z 39`. Pockets: diameter `14 + 2·0.05 = 14.1`, 5 deep.

## Frame plates and fasteners

### Back plate (`z ∈ [-6, 0]`, plus pillars to `z 28`)

1. Outline: rounded rectangle `x ∈ [-52, 70]`, `y ∈ [-86, 98]`, corner radius 10. Holes: lip bores `r 5` at
   `O, P, G`; mounting holes `r 2.25` at `(-30, 90)` and `(48, 90)`. Extrude `z ∈ [-6, 0]`.
2. Pillars: circles `r 5` at `(-44, 74)`, `(62, 74)`, `(-44, -78)`, `(62, -78)`, each extruded `z ∈ [-0.5, 28]`;
   union all with the plate in one call.
3. Cut one tool, the union of: pockets `r 7.05` at `O, P, G` over `z ∈ [-5, 1]`; per pillar
   `m2fastener.HoleTool(site)` and `m2fastener.CavityTool(site)` (Fasteners).
4. One `edgefinish.Chamfer(ctx, body, +Z, {-6, 0}, keep, edgefinish.MinChamfer)`. `keep` takes outer loops, and
   single-edge circular loops whose start point lies within 0.01 mm of radius 7.05 (pocket rims, face `z 0`) or 5
   (lip-bore rims, face `z -6`) about `O`, `P` or `G`. Expect 22 edges.

### Front plate (`z ∈ [28, 34]`)

1. Outline: rounded rectangle `x ∈ [-52, 70]`, `y ∈ [-86, 82]`, radius 10. Holes: lip bores `r 5` at `O, P, G`;
   no bolt holes in the outline (the hole tools cut them). Extrude `z ∈ [28, 34]`.
2. Cut one tool: pockets `r 7.05` at `O, P, G` over `z ∈ [27, 33]`; per pillar `m2fastener.HoleTool(site)` and
   `m2fastener.PocketTool(site)`; the suspension socket, D press outline about `S` (`r 2.55`, chord at
   `x = S_x + 2.05`) over `z ∈ [31, 35]`.
3. One `edgefinish.Chamfer` over `{28, 34}`: outer loops, pocket rims on `z 28`, lip-bore rims on `z 34` (same
   radius test). Expect 22 edges.

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
| `Phase` | pose, fraction of a period | 0 | 0 up to but not including 1 |
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
| 12 | Great Wheel Arbor | printed | Arbors | with the drum |
| 13 | Escape Wheel (wheel + pinion + rear boss) | printed | Escapement / Escape Wheel | φ |
| 14 | Anchor | printed | Escapement / Anchor | ψ |
| 15 | Crutch | printed | Pendulum Assembly / Crutch | ψ |
| 16 | Pendulum | printed | Pendulum Assembly / Pendulum | β about `S` |
| 17 | Suspension Pin | printed | Pendulum Assembly / Suspension | fixed |
| 18 | Suspension Cap | printed | Pendulum Assembly / Suspension | fixed |
| 19 | Great Wheel | printed | Drive / Great Wheel | φ·13/48 ccw about `G` |
| 20 | Drum | printed | Drive / Drum | as great wheel |
| 21 | Winding Knob | printed | Drive / Winding Knob | as great wheel |
| 22-25 | Bolt 1..4 | M2 × 10 | Bolts | fixed |
| 26-29 | Nut 1..4 | M2 | Nuts | fixed |

Bolt/nut `i` sits at pillar `i` in the order `(-44, 74)`, `(62, 74)`, `(-44, -78)`, `(62, -78)`. Use
`Container.ChildNamed` for kinds with several bodies; `doc.VerifyContainers().Err()` must be nil.

## Fits

Classes from `docs/model-pipeline.md` Part clearances: Tight 0.2, Regular 0.35, Loose 0.5 (built gap = class + 0.05
margin, or larger). Nominal parts (bearings, bolts, nuts) keep nominal size.

| ID | Parts | Fit | Class | Built gap |
|---|---|---|---|---|
| T1 | each arbor in its two bearings; suspension pin shank in the pendulum bearing | radial | none: diameter = bore (Part clearances) | 0 |
| T2 | collars, spacer bosses, rear boss, shoulder, cap to bearing faces | axial | Tight | 0.25 |
| T3 | escape wheel, anchor, drum D bores on their arbors (parts that turn with the arbor) | radial and at the flat | press fit | 0.05 |
| T4 | great wheel rear face to its collar; drum pawl layer to the ratchet pocket floor (the great wheel turns relative to both) | axial | Tight | 0.25 |
| T4p | escape wheel front face to escape collar; anchor rear face to pallet collar (pressed parts, press stop) | axial | press fit | 0.05 |
| T5 | crutch and knob rear faces to the front plate's front face | axial | Tight | 0.25 |
| T6 | crutch pin in the pendulum slot | each side | Tight | 0.25 |
| T7 | drum pawl contact face to ratchet radial face at the build pose (working contact) | angular | Tight | 0.25 at `r 12` |
| R1 | great wheel bore (5.8) on the great wheel arbor (5.0) | radial | Regular | 0.40 |
| L1 | escape wheel to drum flange | axial | Loose | 0.75 |
| L2 | drum boss (`r 3.7`) to escape wheel tips; great wheel arbor (`r 2.5`) to escape wheel tips | radial | Loose | 2.05; 3.25 |
| L3 | pendulum to crutch; pendulum to knob | axial | Loose | 0.50 |
| L4 | anchor to front plate; escape wheel to front plate | axial | Loose | 2.0; 3.0 |
| L5 | pawl arm to ratchet tips; pinion tips to drum pawl layer | radial | Loose | 1.0; 22 |
| L6 | shoulder (`r 3.7`) in the pendulum lip bore (`r 5`) | radial | Loose | 1.30 |
| E1 | anchor to escape wheel apart from the working contact, over the swing | 2D | Loose | ≥ 0.6 (escapement check) |
| E2 | pallets to the root circle at `A = 6°` | 2D | Loose | ≥ 1.2 |
| E3 | pallet tops and arms to the tip circle | 2D | Loose | ≥ 2.3 |
| P1 | crutch, knob on arbor ends; suspension pin rear peg in the front plate; cap on the pin | press | press-fit rule | 0.05 |
| B1 | each 605ZZ in its pocket | radial | bearing-seat exception | 0.05 |
| G1 | pinion / great wheel | tip-to-root | gear rule | ≥ 0.5 at 5 angles |

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
| Back plate | outer loop both faces; lip-bore rims on the rear face; pocket rims (seat chamfer) | mounting holes, pillar holes (< 20 mm) |
| Front plate | outer loop both faces; lip-bore rims on the front face; pocket rims | bolt holes, head pockets, suspension socket (< 20 mm) |
| Escape Wheel | 6 window loops on both faces | tooth outline: pieces 2 to 8 mm, corners, no 20 mm run |
| Anchor | whole outer loop of both faces, pallets included | none. The wheel spans `z [22, 25]`, 1 mm inside each anchor face, so the 0.5 chamfer leaves the working faces whole |
| Pendulum | outer loop of both faces; pocket rim; lip-bore rim on the rear face | slot (fit T6) |
| Great Wheel | none | front face outer loop = tooth flanks; ratchet pocket loop = functional teeth |
| Drum, crutch, knob, pin, cap, arbors | none | faces under 400 mm² (crutch 321, knob 294, drum flange front 372.5) |

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
| Drum, Great Wheel Arbor, Winding Knob | rotate about `G` by `θg0 + 20.8° + (φ - 0.05°)·13/48` |
| everything else | fixed |

The escape wheel is built with tooth 0 on `+Y` and the pinion's tooth 0 at -60°. Arbor flats and part D bores are
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
| `TestInventory` | default, Phase 0.3 | 29 bodies, all solid, unique names, Named order = Bodies table; containers and membership as Bodies table; `VerifyContainers().Err()` nil |
| `TestGearMesh` | default; `Backlash 0` | pinion and great wheel outlines at `z 3`, 5 samples over one pinion pitch with the mesh relation: tip-to-root ≥ 0.5 both ways, no overlap |
| `TestRatchet` | default | at `z 7` outlines: pawl-to-face gap 0.25 ± (0.02 + OffSurface); drum turned ccw 2.5° relative to the wheel overlaps (drive engages); turned cw 2.5° does not; pawl arm to ratchet tips ≥ 0.95 |
| `TestCoupling` | Phase 0, 0.25, 0.5, 0.75 | crutch pin to slot sides ≥ 0.2 each side; pendulum not overlapping crutch, knob or plate |
| `TestFits` | default | each Fits row's lower bound (`internal/meshgap` or `SectionOutline.Clearance`) ≥ class minimum; press and seat rows 0.05 ± bound; T1: shaft radius at each bearing seat 2.5 ± bound |
| `TestFasteners` | default | each head top at `z 34` ± 1e-6 and in its pocket; each nut inside its cavity bounds; bolt tip inside the pillar; no bolt or nut overlaps the plate it passes; 1 mm of material round pockets and cavities |
| `TestEdgeFinishing` | default | Edge finishing table (volume drop per part; sharp loops keep their edges) |
| `TestPlacement` | origin `(10, -5, 7)`, `U = +Y`, `V = +Z` | same volumes as unplaced; bounds follow the frame |
| `TestPoseNoOverlap` | Phase 0, 0.1, 0.35, 0.6, 0.85 | no two bodies overlap (escape wheel/anchor, pinion/great wheel, drum/great wheel by outline; others by meshgap) |
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

## Look-alike parts

This design has no "Look-alike parts" section, so `docs/model-pipeline.md` Look-alike part marks does not apply
and no marks are cut. The parts a person could confuse are the three arbors. Marks cannot be placed on them: a
1.5 mm hole with 1 mm of material each side needs a face wider than 3.5 mm off the axis, and a 5.0 mm arbor has
none. They differ by sight instead. The escape arbor is 38 mm long and the other two are 42.2 mm. The pallet arbor's
collar is mid-length (`z 20`) and the great wheel arbor's is at its rear end (`z 0.75`). The README parts table
states these differences.
