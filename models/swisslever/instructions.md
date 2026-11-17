# Swiss lever escapement

## Purpose and source

Construct a desk-scale, printable Swiss lever escapement as one assembly: a 15-tooth club-tooth escape wheel, a lever
with two pressed-in pallet stones, a fork, a dart and two short horns, a balance with a roller table, an impulse pin, a
safety roller with a crescent, a printed hairspring, two banking posts and a two-plate frame on six purchased 605ZZ
bearings. A knob on the escape wheel arbor drives it by hand. It is the source for downstream ports. All lengths are in
millimeters; this document gives angles in degrees, and the code uses radians.

`models/escapement` (the Graham escapement) sets the conventions this model copies: the two-plate frame, 605ZZ bearings
from `models/ballbearing`, stepped arbors with collars, split shafts, M2 fasteners from `internal/m2fastener`, body
containers, the corner, fit and edge rules, a closed-form `Motion` and a quasi-static action test on real sections.

## Choices

### Club teeth

Use **club teeth**, the true Swiss form. A club tooth carries a short impulse face of its own between its locking corner
(the toe) and its letting-off corner (the heel), so the lift is split between tooth and pallet (4.0° on the tooth and
4.5° on the pallet here). Every corner of this club tooth is 71° or blunter (toe 109.5°, heel 71.5°), so it prints at a
0.4 mm nozzle as drawn; a pointed (English) ratchet tooth ends in a tip of 30° or less that a printer rounds over by
0.2 mm or more, a fifth of the 1.0 mm lock, which would move every event of the action. The club tooth also needs no
thin tip to survive the impact of each drop. The cost is a second impulse contact (the pallet's locking corner on the tooth's face), which is still a
closed-form corner-on-line contact.

### Drive

Drive the escape wheel arbor directly with a **winding knob** pressed on its front end, as the Graham model's knob is
pressed on its arbor. The visitor turns the knob clockwise with a fingertip; the escapement lets the wheel advance one
tooth per balance period and the balance takes up its swing. Estimated torque at the knob (Balance and hairspring) is 4
to 19 N·mm, far below what a fingertip gives. A spring drive would need a click, a pawl carrier, a mainspring, a
retainer, an anchor pin and a gear pair (six to eight more parts); with the 15-tooth wheel and a 0.58 s balance it would
run for well under a minute per wind without a gear pair. That option is left for a later change.

### Hairspring

Include a **printed hairspring**: a `models/spiralspring` body in PETG (imported, not copied), its hub on the balance
staff's flat and its eye on a stud post of the front plate. A lever escapement has no restoring force without it: the
lever only unlocks and gives impulse, so a free balance would turn once and never return to the fork. The spring rate
of 1.83 N·mm/rad with the 156 g·cm² balance gives a period of about 0.58 s. The model draws the spring relaxed at
every pose (Motion and pose explains why).

### Pallet stones

The pallet stones are **separate printed parts**, pressed into keyed slots that run through the lever. The escape
wheel runs 0.55 mm in front of the lever plate, because the tangent geometry of a 2.5-tooth span puts the pallet arbor
only 4.42 mm above the heel circle, closer than any printed hub. Only the stones reach forward into the wheel's plane.
Separate stones keep the lever a flat plate with through slots, so no stone root leaves an inside corner between print
layers, and they can be printed in red. Each stone section is a trapezoid, so it fits its own slot one way only.

### Parts kept together

The lever's dart is part of the lever, so the guard freedom does not depend on a second D-flat fit. The impulse pin is
part of the roller table. The safety roller is a separate pressed cylinder: its crescent then runs through its whole
length and leaves no floor.

## Scale and print size

| Item | Size |
|---|---|
| Escape wheel | 15 club teeth, heel (tip) circle `R_h` 40.1546 (80.3 across), toe circle `R_t` 38.6, root 32.0, 3 mm thick |
| Lever | pallet arbor to balance staff `D` 48; total swing 14° (banking ±7°); plate 6.5 thick; 53 mm across the pallets |
| Pallet stones | trapezoids about 4.8 × 7.0 in section, 10.6 long |
| Balance | rim 84 across (`r` 36 to 42), 5 thick, 4 arms; 13.1 g in PLA; roller table 22 across; safety roller 12.89 across |
| Hairspring | 60 across, 8 turns, strip 1.5 × 3, PETG |
| Frame | back and front plates 112 × 196 × 6; 27 mm between the plates |
| Movement depth | 43.30 mm, from the back plate's rear face (`z -6`) to the knob's front face (`z 37.30`) |
| Bearings | six purchased 605ZZ (5 × 14 × 5); arbors 5.0 at the seats, equal to the bore |
| Fasteners | four M2 × 10 bolts and four M2 nuts |

Every part fits a 220 × 220 mm bed; the largest are the plates (112 × 196). The movement lies on the back plate's rear
face on a desk, with the knob on top. Nothing stands out of the rear face, so it needs no stand and no mounting holes.

## Frame and conventions

- Model frame: origin `O` on the escape wheel arbor axis in the plane of the back plate's inner face. `+X` right, `+Y`
  up (toward the balance), `+Z` toward the viewer (the front). The back plate spans `z ∈ [-6, 0]`.
- `math angle` = counter-clockwise from `+X` seen from `+Z`. `rot(v, a)` turns `v` by the math angle `a`. `(r∠a)` is
  the point at distance `r` and math angle `a`.
- Wheel angle `φ` and lever angle `ψ` are **clockwise-positive** seen from `+Z`; a part turned by a clockwise angle `a`
  is rotated by math angle `-a`. Balance angle `θ` is **counter-clockwise-positive**, so `θ` and `ψ` have the same sign
  while the impulse pin is in the fork.
- The escape wheel turns clockwise. Positive `ψ` moves the entry (left) pallet out of the wheel and the exit (right)
  pallet into it.
- Axes: `O = (0, 0)`, `P = (0, d) = (0, 44.5714)` pallet arbor, `B = (0, d + D) = (0, 92.5714)` balance staff. All three
  lie on the line `x = 0` (a straight-line lever).
- Lever frame: origin `P`, axes parallel to the world at `ψ = 0`; `+Y` points from `P` toward `B`. A lever-frame point
  `a` lies at world `P + rot(a, -ψ)`. A world point `W` seen at lever angle `ψ` has lever-frame position
  `rot(W - P, ψ)`.
- Wheel frame: origin `O`; tooth 0's toe on `+Y`. A wheel-frame point `w` lies at world `rot(w, -φ)`.
- Roller frame: origin `B`, axes parallel to the world at `θ = 0`. A roller-frame point `q` lies at world
  `B + rot(q, θ)`. At `θ = 0` the impulse pin lies on the ray from `B` toward `P` (roller-frame `-Y`).
- `Placement *r3.Frame` maps the model frame (origin `O`, `U = +X`, `V = +Y`) into the world; nil is identity.

## Escapement geometry

### Fixed constants

| Symbol | Value | Meaning |
|---|---|---|
| `N` | 15 | escape wheel teeth; pitch `τ = 24°` |
| `R_t` | 38.6 | toe circle: radius of each tooth's locking corner |
| `θ_s` | 30° | half the angle the pallets span: 2.5 pitches, `60°` in all |
| `κ` | 2.5° | lock: lever angle from first lock to unlock at the drop |
| `λ_t` | 4.0° | tooth lift: lever angle while the pallet's locking corner rides the tooth's club face |
| `λ_p` | 4.5° | pallet lift: lever angle while the tooth's heel rides the pallet's impulse face |
| `σ` | 1.5° | slide (run to the banking) after the drop |
| `δ` | 12° | draw: tilt of each locking face from the arc about `P` |
| `a_c` | 3° | club width: angle about `O` from toe to heel |
| `w` | 4.6 | pallet width, measured radially about `P` |
| `ρ` | 16° | tooth front-face lean from the inward radial at the toe, toward `-X` |
| `χ` | 20° | tooth back-face lean from the inward radial at the heel, toward `-X` |
| `R_r` | 32.0 | escape wheel root circle |
| `L_s` | 7.0 | pallet stone length along its locking face |
| `β_b` | 6° | turn of each stone's back face toward its locking face (the key) |
| `ov` | 0.5 | overlap of every stacked or patterned union |

### Derived values

| Value | Formula | Default |
|---|---|---|
| `λ` total lift | `λ_t + λ_p` | 8.5° |
| `α` banking | `(λ + κ)/2 + σ` | 7.0° |
| `d` centre distance | `R_t / cos θ_s` | 44.5714 |
| `r_m` lock radius | `R_t tan θ_s` | 22.2857 |
| `ψ_Eu` entry unlocks | `(κ - λ)/2` | -3.0° |
| `ψ_Et` entry tooth lift ends | `ψ_Eu + λ_t` | +1.0° |
| `ψ_Ef` entry releases (drop to exit) | `(λ + κ)/2` | +5.5° |
| `ψ_Xu`, `ψ_Xt`, `ψ_Xf` (exit) | `-ψ_Eu`, `-ψ_Et`, `-ψ_Ef` | +3.0°, -1.0°, -5.5° |
| `I_L`, `I_R` lock points | `(R_t∠(90° ± θ_s))` | (-19.3000, 33.4286), (19.3000, 33.4286) |
| `C_E` entry locking corner | `rot(I_L - P, ψ_Eu)` | (-19.8567, -10.1175); `r` 22.2857, -153.0000° |
| `C_X` exit locking corner | `rot(I_R - P, ψ_Xu)` | (19.8567, -10.1175); `r` 22.2857, -27.0000° |
| `R_h` heel circle | `|P + rot(C_E, -ψ_Et)|` | 40.1546 (the same from `C_X` at `ψ_Xt`) |
| `X_E` entry release point | heel circle ∩ circle `(P, r_m - w)`, left | (-15.9182, 36.8647) |
| `X_X` exit release point | heel circle ∩ circle `(P, r_m + w)`, right | (23.9037, 32.2647) |
| `F_E` entry letting-off corner | `rot(X_E - P, ψ_Ef)` | (-15.1063, -9.1970); `r` 17.6857, -148.6661° |
| `F_X` exit letting-off corner | `rot(X_X - P, ψ_Xf)` | (22.6141, -14.5412); `r` 26.8857, -32.7417° |
| `u_E` entry locking face direction | `rot(-rot(C_E, 90°)/r_m, -δ)` | angle 105.0000° |
| `u_X` exit locking face direction | `rot(rot(C_X, 90°)/r_m, -δ)` | angle 51.0000° |
| `φ_E` | `(90° - ∠I_L) mod τ` | 18.0000° (tooth 2's toe at `I_L`) |
| `φ_X` | `φ_E + τ/2` | 30.0000° (tooth 0's toe at `I_R`) |

The circle intersection of a radius `r` about `P` with a circle `R` about `O` is `y = (R² - r² + d²)/(2d)`,
`x = ±sqrt(R² - y²)`. The zero-draw locking face is the tangent of the circle about `P` through the corner; the draw
turns it clockwise by `δ` on both pallets, so that the tooth's push draws the pallet into the wheel (toward its banking).
A face direction `u` points from the locking corner along the face away from the impulse face.

### Escape wheel tooth (wheel frame, tooth 0)

| Point | Definition | Default |
|---|---|---|
| `T` toe | `(R_t∠90°)` | (0, 38.6) |
| `H` heel | `(R_h∠(90° + a_c))` | (-2.1015, 40.0996) |
| `L` front foot | line from `T` along `(-sin ρ, -cos ρ)` ∩ circle `r = R_r - ov` | (-2.0551, 31.4329) |
| `K` back foot | line from `H` along `rot(-H/|H|, -χ)` ∩ circle `r = R_r - ov` | (-4.8453, 31.1251) |

Tooth 0 is the quadrilateral `L → T → H → K` (front face `L–T`, club face `T–H`, back face `H–K`, foot `K–L` inside the
root disc). At `r 32` the feet are (-1.9089, 31.9430) and (-4.6832, 31.6555), a root width of 2.7892. Tooth `j` is tooth 0
turned by math angle `j τ`. The club face rises 1.5546 from toe to heel.

### Pallet stones (lever frame)

| Stone | Corners in order: locking corner `C`, letting-off corner `F`, back corner `G`, top corner `Hs` |
|---|---|
| Entry | `C_E` (-19.8567, -10.1175); `F_E` (-15.1063, -9.1970); `G_E` (-17.6475, -2.5769); `Hs_E = C_E + L_s u_E` (-21.6685, -3.3560) |
| Exit | `C_X` (19.8567, -10.1175); `F_X` (22.6141, -14.5412); `G_X` (26.5920, -8.4157); `Hs_X = C_X + L_s u_X` (24.2620, -4.6775) |

`G` is where the back face from `F` along `rot(u, β_b)` (entry: 111.0°, exit: 57.0°) meets the line through `Hs`
parallel to `C → F`. Faces: locking face `C–Hs`, impulse face `C–F`, back face `F–G`, top `G–Hs`. Interior angles: entry
`C` 94.03°, `F` 79.97°, `G` 100.03°, `Hs` 85.97°; exit `C` 109.06°, `F` 64.94°, `G` 115.06°, `Hs` 70.94°. Section areas
31.19 and 31.82 mm². Each stone is that section extruded over `z ∈ [3.30, 13.90]`. Because the trapezoid is not
centrally symmetric, a stone turned 180° about its length, or turned over, does not enter its slot, and the entry stone
does not enter the exit slot.

### Action of wheel and pallets

The lever angle `ψ` drives the wheel angle `φ` in closed form. "Swing +" is `ψ` increasing (the entry pallet works);
"swing −" is `ψ` decreasing (the exit pallet works).

Swing +, tooth `k_E = 2`, reference `φ_E`:

1. `ψ ≤ ψ_Eu` (lock, entry): the toe of tooth 2 lies on the entry locking face. `φ` solves: the toe `(R_t∠(90° + 2τ - φ))`
   lies on the world line through `P + rot(C_E, -ψ)` along `rot(u_E, -ψ)`, on the side `s ≥ 0` from the corner. With the
   draw, `φ` falls as the lever unlocks: from 18.4047° at `ψ = -α` to 18.0000° at `ψ_Eu` (recoil 0.4047°, 0.273 mm at
   the toe).
2. `ψ_Eu < ψ ≤ ψ_Et` (tooth lift): the world point `Q = P + rot(C_E, -ψ)` lies on tooth 2's club line. With the club
   line's unit normal `n` and offset `c = n · T_2` in the wheel frame, `φ = ∠n - ∠Q ± acos(c / |Q|)`, the root nearest
   the previous `φ`. `φ` runs from 18.0000° to 21.0775°.
3. `ψ_Et < ψ ≤ ψ_Ef` (pallet lift): tooth 2's heel lies on the world segment from `P + rot(C_E, -ψ)` to
   `P + rot(F_E, -ψ)`: intersect the heel circle with it (parameter in `[0, 1]`) and take `φ = ∠H_2 - ∠(point)`.
   `φ` runs from 21.0775° to 27.6452°.
4. `ψ > ψ_Ef` (lock, exit): the toe of tooth 0 lies on the exit locking face (as 1, with `C_X`, `u_X`). The drop at
   `ψ_Ef` takes `φ` from 27.6452° to 30.3414° (2.6962°, 1.890 mm at the heel circle); the slide to `α` adds 0.2388° to
   30.5803°.

Swing −, tooth `k_X = 0`, reference `φ_X`: the same four steps with `C_X`, `F_X`, `u_X` and the exit events. Lock until
`ψ_Xu` (from 30.5803° at `ψ = α` back to 30.0000°, recoil 0.5803°); tooth lift to 32.9225° at `ψ_Xt`; pallet lift to
39.5334° at `ψ_Xf`; drop onto tooth 3 at the entry face, 42.2728° (2.7395°, 1.920 mm); slide to 42.4047° at `-α`.

Per beat the club gives 3.0775° (entry) and 2.9225° (exit) of wheel turn and the pallet 6.5678° and 6.6108°. The two
beats are 12.1756° and 11.8244° from banking to banking, 24° per period. The lock at the landing is 0.985 mm (entry)
and 1.003 mm (exit) along the face; at the banking the toe stands 1.569 mm (entry) from the corner.

### Numerical verification

A quasi-static simulation drives the lever between the bankings and, at each step, turns the wheel as far forward as the
tooth and stone polygons allow; a stone pushing a tooth from behind moves the wheel forward, and a drawn locking face
pushes it back. On the analytic outlines (2000 steps per period, periods 2 and 3):

| Quantity | Result |
|---|---|
| Wheel advance per period | 24.000° (one tooth) |
| Wheel angle against the closed form | equal within 1e-8° away from the two drop events |
| Drops | 2 per period, at `ψ` +5.52° and -5.52°: 1.933 mm and 1.989 mm (the closed-form 1.890 and 1.920 plus one step) |
| Landing face | the locking face, both times |
| E1: least gap, every non-contact tooth to both stones | 1.031 |
| E2: least gap, stones to the root circle, over the swing | 2.310 |
| E3: stone top corners outside the heel circle, over the swing | 1.452 |
| Escape wheel to the pallet arbor body (`r 3.3`) | 1.117 (`d - R_h - 3.3`) |

Print-error study on the analytic polygons at 700 steps per period (`dr` toe and heel radius error, `dw` pallet
impulse-face length error, `dd` centre distance error):

| Case `(dr, dw, dd)` | Advance | Drops (mm) | E1 |
|---|---|---|---|
| nominal | 24° | 1.933 / 1.989 | 1.098 |
| `(-0.2, 0, 0)` | 24° | 1.796 / 1.903 | 1.214 |
| `(+0.2, 0, 0)` | 24° | 2.064 / 2.103 | 0.912 |
| `(0, +0.2, 0)` | 24° | 1.732 / 1.809 | 0.898 |
| `(0, -0.2, 0)` | 24° | 2.090 / 2.165 | 1.297 |
| `(0, 0, +0.15)` | 24° | 1.705 / 2.079 | 1.195 |
| `(0, 0, -0.15)` | 24° | 2.134 / 1.820 | 0.753 |
| `(+0.2, 0, -0.15)` | 24° | 2.301 / 1.959 | 0.636 |
| `(+0.2, +0.2, -0.15)` | 24° | 2.108 / 1.787 | 0.437 |
| `(-0.2, +0.2, -0.15)` | 24° | 1.859 / 1.622 | 0.796 |
| `(-0.2, -0.2, +0.15)` | 24° | 1.786 / 2.188 | 1.568 |
| `(+0.2, -0.2, +0.15)` | 24° | 2.005 / 2.348 | 1.284 |
| `(-0.2, 0, +0.15)` | 24° | 1.595 / 2.008 | 1.369 |

The 1.5° slide is what keeps the `(+0.2, *, -0.15)` cases running: there the release comes up to 1.34° after `ψ_Ef`,
and with a 0.75° slide the lever reached the banking before the tooth dropped.

## Lever, fork and guard

### Fork constants and derived points (lever frame)

| Symbol | Value | Meaning |
|---|---|---|
| `D` | 48 | pallet arbor to balance staff |
| `r_j` | 8.5 | impulse pin centre radius about `B` |
| `r_p` | 1.5 | impulse pin radius (3.0 across) |
| `c_s` | 0.25 | gap from the centred pin to each corner round at `θ = 0`, `ψ = 0` (the slot shake) |
| `β_K` | 75° | direction of each corner round's centre from the centred pin, measured from `-Y` |
| `b_a` | 3.0 | half width of the lever's main arm |
| `r_c` | `(b_a - (r_p + c_s) sin β_K)/(1 + sin β_K)` = 0.66616 | corner round radius; the round is tangent to the arm side |
| `K_R` | `(0, D - r_j) + (r_p + r_c + c_s)(sin β_K, -cos β_K)` = (2.33384, 38.87465) | right corner round centre; `K_L` its mirror |
| `r_n` | `K_R.x - r_c` = 1.66767 | notch semicircle radius, centre `N_c = (0, K_R.y)`; bottom at `y` 37.20698 |

The fork end is the arm itself: each side line `x = ±b_a` runs up to the top of its corner round. Each round (a
half-turn arc over the top, from the arm side at `(±3.0, 38.87465)` to the notch at `(±1.66767, 38.87465)`) is a horn
tip; the material between the notch and the arm side, 1.333 wide, is a horn. The horns end in these full rounds instead
of long curved faces because the safety roller shares their levels (Guard below); their tips stay 1.945 mm from the
safety roller at every lever angle. The notch is the semicircle of radius `r_n` below the two rounds' inner ends.

### Pin-in-fork action

The pin centre at balance angle `θ` is `Q(θ) = (0, D) + rot((0, -r_j), θ)` in the P frame (world axes). A corner round
with centre `Kc` touches the pin when `|rot(Q, ψ) - Kc| = r_p + r_c`: intersect the circle `|q| = |Q|` with the circle
`|q - Kc| = r_p + r_c` and take `ψ = ∠Kc - ∠(intersection)`, choosing the root that leaves `Kc` on its own side of the
pin (`(Kc.x - q.x)·sign(Kc.x) > 0`). Every contact lands on the round's upper half: directions from `K_R` of 126.2° to
163.2°, inside its 0°–180° arc.

Swing + (`θ` increasing; the entry pallet works):

| Interval | Lever | Default `θ` |
|---|---|---|
| `θ < θ1` | on the left banking, `ψ = -α` | `θ1` = -27.33541°: the pin meets `K_R` with the lever at `-α` |
| `θ1 ≤ θ < θ2` | unlocking: the pin pushes `K_R`, `ψ = ψ_KR(θ)` | `θ2` = -11.81885°: `ψ_KR` reaches `ψ_Eu` |
| at `θ2` | impulse starts; the wheel drives the lever across the shake from -3.0° to -2.02444° | |
| `θ2 ≤ θ < θ3` | impulse: `K_L` drives the pin, `ψ = ψ_KL(θ)`; the drop comes at `θ_d` = 22.12829° (`ψ = ψ_Ef`), then the draw keeps `K_L` on the pin | |
| `θ ≥ θ3` | on the right banking, `ψ = +α`; the pin leaves `K_L` | `θ3` = 27.33541° |

Swing − mirrors it: `θ > -θ1` banking `+α`; unlocking by `K_L` until `ψ_Xu`; impulse by `K_R`; banking `-α` from `-θ3`.
The roller lift (the balance angle with the pin touching the fork) is `2 θ3` = 54.671°. Pressure angle (between the
contact normal and the corner's direction of travel about `P`): 50.4° at `θ1` and `θ3`, 19.5° at `θ2`, 13.3° at `θ = 0`,
35.9° at the drop.

Corner freedom: with the lever on a banking, the outer round stands 1.32721 from the pin's path: the distance from `B`
seen in the lever frame, `(-D sin α, D cos α)`, to `K_R`, less `r_c + r_j + r_p`.

### Guard: dart and safety roller

| Item | Value |
|---|---|
| Dart | band S only (`z ∈ [3.30, 5.80]`): a tongue 1.2 wide (sides `x = ±0.6`) ending in a full round `r_d` 0.6 about `T_d = (0, D - e)`, `e` 5.85, so `T_d = (0, 42.15)` |
| Dart root | two gap rounds `r` 0.53384 about `(±1.13384, 38.87465)`, each tangent to its corner round and to the dart side |
| Safety roller | cylinder `r_s` 6.444 about `B`, `z ∈ [3.30, 10.80]` |
| Guard freedom | the lever turns from `α` to `ψ_g` = 5.0° before the dart touches the roller: 2.0°, half the 4.0° lock at the banking |
| Guard gap at the banking | `|T_d - (-D sin α, D cos α)| - r_d - r_s` = 0.9799 |
| Corner freedom with the dart on the roller | 0.3993: the returning pin cannot strike a horn tip while the dart guards |
| Crescent | circle `r_cr` 4.03 whose centre lies 9.0 from `B` on the pin's ray; its near side is 4.97 from `B` (1.62 of wall to the bore); both lips (63.4°) rounded `r` 0.4 |
| Dart in the crescent | least gap 0.2686 over the whole swing, the dart passing at `|θ| ≤ 22.5°` |
| Dart depth at `θ = 0` | `r_s + r_d - e` = 1.194 inside the roller's circle |

The crescent and the pin share the roller ray at local angle `-90° - γ_f` (Balance and hairspring). The dart lies 0.55
behind the pin's tip in `z`, so pin and dart never meet.

### Banking

Two banking posts of the back plate, `r` 2.0, centres at world `(±7.1564, 61.8279)` (lever frame `rot((±(b_a + 2.0), 18.0), ∓α)`
about `P`), stand over `z ∈ [0, 9.80]`. At `ψ = ±α` the arm side `x = ±b_a` touches its post at lever-frame `y = 18`
(a stated contact). The build pose backs the lever off by 0.05°.

### Lever outline (lever frame, counter-clockwise)

Band F outline (`z ∈ [5.80, 9.80]`); band S (`z ∈ [3.30, 5.80]`) is the same with the notch replaced by the dart and its
gap rounds. Both stone slots run through both bands.

| Point | Value | Definition |
|---|---|---|
| `JL`, `JR` | (-3.0, 12.0), (3.0, 12.0) | where the arm sides leave the pallet frame |
| `VE1` | (-24.1557, -1.8008) | the entry slot's locking side and top, each moved out 1.95 (2.0 from the stone), meet |
| `VE2` | (-21.3062, -12.4356) | the entry locking side and impulse side, moved out 1.95 |
| `VE3` | (-12.3844, -10.7068) | the entry impulse side and back side, moved out 1.95 |
| `VX1` | (17.4057, -9.9662) | the exit locking side and impulse side, moved out 1.95 |
| `VX2` | (22.5795, -18.2666) | the exit impulse side and back side, moved out 1.95 |
| `VX3` | (28.9624, -8.4377) | the exit back side and top, moved out 1.95 |
| `VX4` | (24.4743, -1.2372) | the exit top and locking side, moved out 1.95 |

Loop: `JR` → arm side `x = 3.0` up to the right corner round → round over the top → notch semicircle (band F) → left
round → arm side `x = -3.0` down to `JL` → line to `VE1` → line to `VE2` → line to `VE3` → line to `VX1` → line to
`VX2` → line to `VX3` → line to `VX4` → line to `JR`. Fillets: `JL` and `JR` radius 3.0 (half the arm width; 56.9° and
58.4° inside turns), `VX1` radius 1.5 (59.5°). `VE3` turns 9.5° and stays sharp. The convex corners `VE1` (108.1°
material angle), `VE2` (94.0°), `VX2` (64.9°), `VX3` (115.1°) and `VX4` (153.6°) stay sharp. The lower edge `VE3–VX1`
passes 10.40 below `P`, so the hub lies inside the plate. Slots: each stone section moved out by 0.05 (press fit):

| Slot | Corners `C', F', G', H'` |
|---|---|
| Entry | (-19.8930, -10.1755), (-15.0382, -9.2347), (-17.6159, -2.5198), (-21.7306, -3.3171) |
| Exit | (19.7954, -10.1137), (22.6132, -14.6343), (26.6513, -8.4162), (24.2673, -4.5915) |

Every slot corner keeps 2.0 of lever round it (1.95 of wall). The lever's D press bore (body family) is at `P`. Plate
area about 970 mm² per face.

## Balance and hairspring

| Part | Geometry |
|---|---|
| Safety roller | cylinder `r` 6.444, `z ∈ [3.30, 10.80]`, D press bore (body family), crescent cut through its length; local frame: bore flat on `+X`, crescent centre at `(9.0∠(-90° - γ_f))` |
| Roller table | disc `r` 11.0, `z ∈ [10.85, 13.35]`, D press bore; impulse pin `r` 1.5 at `(8.5∠(-90° - γ_f))`, `z ∈ [6.35, 10.85]`, standing on the rear face with a 0.5 root round. Area 380 mm² |
| Balance wheel | rim `r` 36 to 42, four arms 4 wide at local 45°, 135°, 225°, 315°, hub `r` 7, `z ∈ [13.40, 18.40]`; window corners rounded 2.0; hub boss `r` 7, `z ∈ [18.40, 19.40]`, 0.5 root round; D press bore |
| Hairspring | `spiralspring.Build` with the parameters below; placement origin `(B, z 19.65)`, `U = (cos γ_s, sin γ_s, 0)` |
| Hairspring retainer | disc `r` 5.0, `z ∈ [22.90, 25.90]`, D press bore |
| Hairspring stud | post of the front plate, `r` 1.5, from the plate's inner face (`z` 27.0) back to `z` 18.95, 0.5 root round, at the eye centre |

`spiralspring.Params` (constants of this model): `OuterDiameter` 60, `HubDiameter` 12, `Thickness` 1.5, `Height` 3,
`MinGap` 1.0, `Turns` 0, `ArborDiameter` 6.6, `ArborFlatDepth` 0.75, `BoreClearance` 0.25, `AnchorPinDiameter` 3.0,
`AnchorClearance` 0.4. `Derive` gives 8 turns, pitch 2.6667, gap 1.1667, `rho0` 8.5, bore radius 3.55, bore flat 2.80
from the axis (the staff's flat is at 2.55), eye centre `E = (-29.2500, -13.0044)` (radius 32.0106, math angle
-156.0303°), eye radius 3.4, eye hole 3.8, fillet 0.5833, taper offset 0.0722, active length 967.611.

- `γ_s = 180° - ∠E = -23.9697°` puts the eye centre at `B + (-32.0106, 0)` = (-32.0106, 92.5714), on the front plate's
  balance bar. The spring's bore flat faces world `γ_f = γ_s + 90° = 66.0303°`.
- The staff, safety roller, roller table, balance wheel and retainer are built with their D flats on local `+X` and
  turned about `B` by math angle `γ_f + θ`.
- Spring rate `k = E h t³ / (12 L)` with PETG `E` 2.1 GPa (1.9 to 2.2): 1.8312 N·mm/rad. Strain `(t/2) θ / L`: 0.24 %
  at 180°, 0.37 % at 270° (PETG yields near 3.5 %).
- Balance inertia (PLA 1.24 g/cm³, rim, arms, hub, rollers): 15,617 g·mm². Period `2π sqrt(I/k)` = 0.580 s (0.567 to
  0.610 s over the modulus range), 12,400 beats per hour. The wheel advances one tooth per period: 8.7 s per turn.
- Energy and drive (estimates, not measurements): at 180° the balance stores 9.0 mJ; with an assumed `Q` of 20 to 40
  and 35 % escapement efficiency it needs 9.7 to 19.4 N·mm at the knob; at 120°, 4.3 to 8.6 N·mm.

## Drive

Winding knob: disc `r` 11 about `O`, `z ∈ [34.30, 37.30]` (380 mm², no chamfer), D press socket (end family: arc
`r 2.05`, chord `x 1.55`) through, rear keyed counterbore (arc `r 3.75`, chord `x 3.35`) cut over `z ∈ [33.30, 36.30]`, so
its floor is 1.0 thick. Knob Collar: keyed D ring (arc `r 3.7`, chord `x 3.3`) over `z ∈ [33.25, 36.25]`, with the end
family D bore through; pressed on until its bore chord stops on the end of the escape arbor's `x 1.5` flat at `z 33.25`
(stated contact); the knob then goes on until its floor stands 0.05 from the ring. Both turn with the escape arbor.
The action is designed and checked for clockwise turning only.

## Arbors, bearings and axial layout

Bearing seats are 5.0 across (Part clearances: a shaft through a purchased bearing equals its bore). Body sections and
bore families (all D flats on local `+X`) are the Graham model's:

| Name | Value | Use |
|---|---|---|
| seat | `r 2.5` | every bearing seat |
| body | `r 3.3`, flat plane `x 2.55` | arbor body between the seats |
| shoulder | `r 3.7` | collars (`2·3.7` = 7.4, inside the 605ZZ inner ring face of 8.0) |
| end | `r 2.0`, flat `x 1.5` | the escape arbor's end outside the front plate |
| body family | arc `r 3.35`, chord `x 2.60` | D press bores on the body: escape wheel, lever, safety roller, roller table, balance wheel, retainer |
| end family | arc `r 2.05`, chord `x 1.55` | D press bores on the end: knob, Knob Collar |

Every step that does not face a bearing is a 30° cone (its length along the axis is 1.732 × the radius change). A step
that faces a bearing across a gap stays square. A pressed part whose bore corner faces a cone stands 0.05 from it,
measured normal to the cone (cone end shifted by `(dr cos 30° - 0.05)/sin 30°` = -0.0134 for `dr` 0.05).

| Arbor | Axis | Meridian `(r, z)` | Flats (flat plane, span) |
|---|---|---|---|
| Escape Arbor | `O` | `(0,-5) (2.5,-5) (2.5,0.25) (3.3,0.25) (3.3,13.3634) (3.7,14.0562) (3.7,26.75) (2.5,26.75) (2.5,32.25) (2.0,33.116) (2.0,37.25) (0,37.25)` | `x 2.55`: `[0.20, 13.35]`, ramp past 13.35 (leaves the shoulder at `z 15.342`, reaches `x 4.0` at `15.861`); `x 1.5`: `[33.25, 38.25]` |
| Pallet Arbor | `P` | `(0,-5) (2.5,-5) (2.5,0.25) (3.7,0.25) (3.7,2.5938) (3.3,3.2866) (3.3,26.75) (2.5,26.75) (2.5,32.0) (0,32.0)` | `x 2.55`: ramp behind 3.30 (leaves the shoulder at `z 1.308`), `[3.30, 26.80]` |
| Balance Staff | `B` | the Pallet Arbor's meridian | the Pallet Arbor's flats |

The Pallet Arbor and the Balance Staff are the same part, built twice. Axial stacks (z):

| Item | Escape (`O`) | Pallet (`P`) | Balance (`B`) |
|---|---|---|---|
| back bearing | `[-5, 0]` | `[-5, 0]` | `[-5, 0]` |
| rear collar | body step `z 0.25` | shoulder `[0.25, 3.2866]` | shoulder `[0.25, 3.2866]` |
| parts | escape wheel `[10.35, 13.35]` | lever `[3.30, 9.80]`, stones `[3.30, 13.90]` | safety roller `[3.30, 10.80]`, roller table `[10.85, 13.35]`, balance wheel `[13.40, 18.40]` + boss to 19.40, hairspring `[19.65, 22.65]`, retainer `[22.90, 25.90]` |
| front collar | shoulder `[13.3634, 26.75]` | body step `z 26.75` | body step `z 26.75` |
| front bearing | `[27, 32]` | `[27, 32]` | `[27, 32]` |
| outside | Knob Collar `[33.25, 36.25]`, knob `[34.30, 37.30]` | none (ends at 32) | none (ends at 32) |

- The escape wheel slides on from the arbor's rear end until its front bore corner stands 0.05 from the shoulder's cone.
- The lever is pressed on from the front until its rear bore corner stands 0.05 from the pallet shoulder's cone; the two
  stones are pressed into it first, flush with its rear face.
- On the staff, from the front: the safety roller (stopped 0.05 from the shoulder's cone), the roller table (pressed to
  0.05 from the safety roller), the balance wheel (0.05 from the roller table), the hairspring (slides on the flat),
  the retainer (pressed until the hairspring hub turns freely by 0.25 each side).
- Print orientations are listed under Materials and print orientation.

### Arbor collars

`docs/model-pipeline.md` Arbor collars applies, with its amendment for a body step on the side parts are fitted from.

| Collar | Shaft | Carried part | Flush side | z | Length | Option |
|---|---|---|---|---|---|---|
| Escape body step | Escape Arbor | Escape Wheel | back escape bearing | step at 0.25, `r 3.3` | n/a | amendment: wheel rear face 10.10 from the step |
| Escape front shoulder | Escape Arbor | Escape Wheel (press stop) | front escape bearing | `[13.3634, 26.75]` | 13.39 | 1: the shoulder fills the gap; cone `[13.3634, 14.0562]` |
| Pallet shoulder | Pallet Arbor | Lever (press stop) | back pallet bearing | `[0.25, 3.2866]` | 3.04 | 1; cone `[2.5938, 3.2866]` |
| Pallet body step | Pallet Arbor | Lever | front pallet bearing | step at 26.75, `r 3.3` | n/a | amendment: lever front face 16.95 from the step |
| Staff shoulder | Balance Staff | Safety Roller (press stop) | back balance bearing | `[0.25, 3.2866]` | 3.04 | 1; cone `[2.5938, 3.2866]` |
| Staff body step | Balance Staff | Hairspring Retainer | front balance bearing | step at 26.75, `r 3.3` | n/a | amendment: retainer front face 1.10 from the bearing |
| Knob Collar | own keyed ring | Winding Knob | front plate outer face (`z 33`) | `[33.25, 36.25]` | 3.0 | 3 → 2: knob moved forward 1.05; keyed counterbore, floor 36.30 |

The roller table and the balance wheel stop on the parts behind them (press stops, Fits T4p) and face no flush side.
The hairspring hub faces the balance boss and the retainer, not a bearing, so neither is a collar. Each body step is
6.6 across against the 6.8 shoulder limit and the 8.0 inner ring face (`ballbearing.Catalog`), with the 0.3 mm land of
the Graham model.

### Split shafts

| Part | Split plane | What holds the halves |
|---|---|---|
| Escape Arbor | holds the axis and the `+X` flats' normal: local normal `+Y`, turned with the arbor (math angle `-(φ - 0.05°)`) | the two 605ZZ inner races, the escape wheel and the Knob Collar |
| Pallet Arbor | the same, turned by `-ψ` about `P` | the two inner races and the lever |
| Balance Staff | the same, turned by `γ_f + θ` about `B` | the two inner races and the pressed balance parts |

Each half stands 3.7 above its seam. Every flat faces local `+X`. The posts of the plates and the impulse pin are
features of their parts, not separate pins (as the Graham crutch pin and pillars are).

## Frame plates and fasteners

### Plates

| Item | Back plate | Front plate |
|---|---|---|
| outline | rounded rectangle `x ∈ [-56, 56]`, `y ∈ [-50, 146]`, corner `r 10` | the same |
| z | `[-6, 0]` | `[27, 33]` |
| bars | horizontal, `|y - c| ≤ 4` for `c` = 0, 44.5714, 92.5714; hub rings `r 10` at `O`, `P`, `B` | the same |
| windows | W1, W2, W3 left and right of a vertical bar `|x| ≤ 11`, W4 | W1, W2, W3, W4 |
| pockets | `r 7.05`, `z ∈ [-5, 0]`, lip bore `r 5` through, at `O`, `P`, `B` | `r 7.05`, `z ∈ [27, 32]`, lip bore `r 5` |
| posts | 4 pillars `r 5`, `z ∈ [0, 27]`; 2 banking posts `r 2`, `z ∈ [0, 9.80]`; 0.5 root rounds | hairspring stud `r 1.5`, `z ∈ [18.95, 27]`, 0.5 root round |
| fasteners | nut cavities at the pillars | head pockets and holes at the pillars |

Pillars at `(-48, -40)`, `(48, -40)`, `(-48, 136)`, `(48, 136)` (bolt and nut 1 to 4 in this order).

Windows (each corner of the bounding rectangle a tangent arc `r 6`; each junction of a bar edge and a hub arc a fillet
`r 1.0`; `h = sqrt(10² - 4²) = 9.1652`):

| Window | Bounds | Hub arcs |
|---|---|---|
| W1 | `x ∈ [-45, 45]`, `y ∈ [-33.5, -4]` | `O` ring under the top edge, from `(-h, -4)` to `(h, -4)` through `(0, -10)` |
| W2 | `x ∈ [-45, 45]`, `y ∈ [4, 40.5714]` | `O` ring over the bottom edge through `(0, 10)`; `P` ring under the top edge through `(0, 34.5714)` |
| W3 (front) | `x ∈ [-45, 45]`, `y ∈ [48.5714, 88.5714]` | `P` ring through `(0, 54.5714)`; `B` ring through `(0, 82.5714)` |
| W3 (back) | `x ∈ [-45, -11]` and `[11, 45]`, `y ∈ [48.5714, 88.5714]` | none |
| W4 | `x ∈ [-45, 45]`, `y ∈ [96.5714, 129]` | `B` ring over the bottom edge through `(0, 102.5714)` |

Material: side rims `|x| ∈ [45, 56]`; bottom rim `y ≤ -33.5`; top rim `y ≥ 129`; each pillar keeps 1.5 mm to W1 and 2 mm to
W4; the banking posts (with their rounds, `|x| ≤ 9.66`) keep 1.34 to the back plate's vertical bar edge; the stud keeps
2.0 to its bar's edges.

### Fasteners (`AGENTS.md` Fasteners)

Use `internal/m2fastener`. Each pillar has one site: head top at `(px, py, 33)`, `Dir -Z`, `Corner +X`, length 10,
`Cavity` 8.

| Item | Value |
|---|---|
| bolt | M2 × 10, head top flush at `z 33` |
| head pocket | `r 2.2`, `z [31, 33]`; 4 mm of plate under it |
| clearance hole | `r 1.3`, `z [20, 34]`, through both plates' pillar region |
| nut cavity | `r 2.51`, `z [23, 25]`, enclosed; roof 2.0 to the pillar top; wall 2.49 |
| bolt tip | `z 21`, inside the pillar hole |
| print pause | back plate printed rear face down: pause at 31 mm above the bed (`z 25`), drop the nuts in, resume |

The nuts are embedded; `steps.md` (Fasteners) says why no open pocket works. The pillar tops are clamped against the
front plate's inner face (stated contact).

## Parameters

| Field | Meaning | Default | Range |
|---|---|---|---|
| `Amplitude` | balance amplitude `Θ`, radians | 180° | 45° to 270° |
| `Phase` | pose, fraction of a balance period | 0.25 (balance at rest) | 0 up to but not including 1 |
| `Placement` | frame mapping the model frame into the world | nil | a valid frame |

`Validate` rejects nonfinite values, each range with an error naming the field, and an invalid `Placement`; zero values
are not default requests. Every other dimension is a constant of this document, so one parameter branch builds every
sketch. `Amplitude` and `Phase` change only the pose.

Exported API (package `swisslever`):

| Symbol | Meaning |
|---|---|
| `DefaultParams() Params`, `(Params) Validate() error` | defaults and checks |
| `Layout(p Params) (Geometry, error)` | every derived value and point of this document |
| `Motion(p Params) (State, error)`, `(*Geometry) StateAt(t float64) (State, error)` | closed-form state (Motion and pose) |
| `(*Geometry) WheelAt(psi float64, plus bool) (float64, string, error)` | wheel angle and stage for a lever angle and swing |
| `(*Geometry) LeverAt(theta float64, plus bool) (float64, string, error)` | lever angle and stage for a balance angle and swing |
| `NewDocument(opts ...decad.DocumentOption) *decad.Document` | `decad.New` with `decad.WithChainTolerance(0.1 mm)` first |
| `Build(ctx, doc, p) (*Movement, error)` | builds and poses every body; validates first; contextual errors |
| `(*Movement) Named() []NamedBody` | every body with its display name, in the Bodies table order |

`State`: `Phase`, `BalanceAngle` (`θ`), `LeverAngle` (`ψ`), `WheelAngle` (`φ`), `LeverStage` (`"banking"`, `"unlock"`,
`"impulse"`), `WheelStage` (`"lock-entry"`, `"tooth-lift-entry"`, `"pallet-lift-entry"`, `"lock-exit"`, and the exit
equivalents), `Locked` (`"entry"`, `"exit"` or `""`). `Movement`: one field per body, `Sketches []*sketch.Sketch`,
`State`.

## Bodies and containers

| # | Body name | Kind | Container path | Motion |
|---|---|---|---|---|
| 1 | Back Plate | printed | Frame | fixed |
| 2 | Front Plate | printed | Frame | fixed |
| 3-8 | Back Escape Bearing, Back Pallet Bearing, Back Balance Bearing, Front Escape Bearing, Front Pallet Bearing, Front Balance Bearing | 605ZZ | Bearings | fixed |
| 9 | Escape Arbor | printed | Arbors | `φ` about `O` |
| 10 | Pallet Arbor | printed | Arbors | `ψ` about `P` |
| 11 | Balance Staff | printed | Arbors | `γ_f + θ` about `B` |
| 12 | Escape Wheel | printed | Escapement / Escape Wheel | `φ` |
| 13 | Lever | printed | Escapement / Lever | `ψ` |
| 14 | Entry Stone | printed | Escapement / Pallet Stones | `ψ` |
| 15 | Exit Stone | printed | Escapement / Pallet Stones | `ψ` |
| 16 | Safety Roller | printed | Balance / Rollers | `γ_f + θ` |
| 17 | Roller Table | printed | Balance / Rollers | `γ_f + θ` |
| 18 | Balance Wheel | printed | Balance / Balance Wheel | `γ_f + θ` |
| 19 | Hairspring | printed (PETG) | Balance / Hairspring | fixed (drawn relaxed) |
| 20 | Hairspring Retainer | printed | Balance / Hairspring Retainer | `γ_f + θ` |
| 21 | Winding Knob | printed | Drive / Winding Knob | `φ` |
| 22 | Knob Collar | printed | Collars | `φ` |
| 23-26 | Bolt 1..4 | M2 × 10 | Bolts | fixed |
| 27-30 | Nut 1..4 | M2 | Nuts | fixed |

Use `Container.ChildNamed` for every path; `doc.VerifyContainers().Err()` must be nil.

## Fits

Classes from `docs/model-pipeline.md` Part clearances: Tight 0.2, Regular 0.35, Loose 0.5; a built gap is the class plus
0.05 or more. Bearings, bolts and nuts keep nominal size. `models/README.md` label: **Mixed**.

| ID | Parts | Fit | Class | Built gap |
|---|---|---|---|---|
| T1 | each arbor in its two bearings | radial | none: diameter = bore | 0 |
| T2 | body steps and shoulders to bearing faces | axial | Tight | 0.25 |
| T3 | D press bores of escape wheel, lever, safety roller, roller table, balance wheel, retainer on `r 3.3` | radial and at the flat | press | 0.05 |
| T4p | escape wheel front bore corner to its shoulder cone; lever and safety roller rear bore corners to their shoulder cones (normal to the cone); roller table to safety roller; balance wheel to roller table; knob floor to the Knob Collar | axial | press stop | 0.05 |
| P1 | stones in their slots (every face) | press | press | 0.05 |
| P2 | knob and Knob Collar on the escape arbor end; Knob Collar in the knob's keyed counterbore | press | press | 0.05 |
| S1 | hairspring hub D bore on the staff (arc and flat) | radial | Tight | 0.25 |
| S2 | hairspring eye on the stud | radial | Regular | 0.40 |
| S3 | hairspring hub to the balance boss and to the retainer | axial | Tight | 0.25 |
| L1 | hairspring coils to the balance wheel's front face; stud tip to the balance wheel | axial | Loose | 1.25; 0.55 |
| F1 | pin to each corner round at `θ = 0`, lever centred | radial | Tight | 0.25 (shake) |
| F2 | pin to the notch bottom | radial | Loose | 0.793 |
| F3 | pin tip to the dart's top face (`z 5.80`) | axial | Loose | 0.55 |
| F4 | corner rounds to the pin path at the banking | 2D | Loose | 1.327 |
| G1 | dart to the safety roller at the banking | 2D | Loose | 0.980 |
| G2 | dart in the crescent, over the swing | 2D | Tight | 0.269 |
| G3 | horn tips (corner rounds) to the safety roller, over the swing | 2D | Loose | 1.945 |
| G4 | pin root round to the safety roller (both turn with the staff) | radial | Tight | 1.62 (built; 0.274 or more required) |
| E1 | non-contact teeth to the stones, over the swing | 2D | Loose | 1.03 |
| E2 | stones to the escape wheel root circle | 2D | Loose | 2.31 |
| E4 | escape wheel to the lever plate; stones' front ends past the wheel's front face | axial | Loose | 0.55; 0.55 |
| E5 | escape wheel heel circle to the pallet arbor body | radial | Loose | 1.117 |
| R1 | lever front face to the roller table; fork corners to the pin root round | axial | Loose | 1.05; 0.55 |
| R2 | balance rim (`r 42`) to the pallet arbor body | radial | Loose | 2.70 |
| R3 | lever to the back plate; balance wheel to the lever | axial | Loose | 3.30; 3.60 |
| K1 | Knob Collar to the front plate's outer face; knob to the front plate | axial | Tight; Loose | 0.25; 1.30 |
| B1 | each 605ZZ in its pocket | radial | bearing seat | 0.05 |

Stated contacts (no class): pillar tops on the front plate; bolt heads on their pocket floors; bolts in their nuts; the
Knob Collar's chord on the end of the escape arbor's flat; the lever arm on a banking post at `ψ = ±α`. Working
contacts: toe on a locking face, corner on a club face, heel on an impulse face, pin on a corner round, dart on the
safety roller. The build pose backs the wheel and the lever off by 0.05° (Motion and pose).

## Retained sharp corners

Rule: "Corners and roots" in `docs/model-pipeline.md`. Parts are cut in their print orientation (Materials).

| Part | Corner | Reason |
|---|---|---|
| Lever | the four corners of each slot | mating corners: each stone corner sits in its slot corner |
| Lever | the bottom of each gap round (`r 0.534`) beside the dart | the largest round the 1.07 gap between the dart and the horn holds; the section mesh reads one of the two at 50.1°, 0.1° over the inside-corner limit |
| Knob, Knob Collar | the keyed counterbore's and ring's D corners, and the end-family socket corners | mating corners, as in the Graham model |
| Escape Arbor | the end of the `x 1.5` flat at `z 33.25` and the Knob Collar's chord on it | mating corners (press stop) |
| Arbors | each square step facing a bearing (`z 0.25` and `z 26.75`) | mating corner with the bearing's inner ring |
| Plates | each bearing pocket floor's edge | between layers; mating corner with the bearing's outer race |
| Winding Knob | its keyed counterbore floor's edge | between layers; mating corner with the Knob Collar |

Rounds below half their section, each limited by a neighbour: the impulse pin's root round 0.5 (pin 3.0 across; the
fork corners pass 1.05 below the roller table, and a larger round would leave less than the Loose 0.55 to them); the
banking posts' root rounds 0.5 (posts 4.0 across; the lever arm swings 0.5 from the root); the plate window corners
1.0 (rigid plate, as in the Graham model); the lever's `VX1` round 1.5 (the slot's 1.95 wall is the thinner section).

Corners between layers that open upward take a round: the pillars', banking posts', stud's, impulse pin's and balance
boss's roots (0.5, from `decad.Boss`), and the lever's dart root: where the band F notch semicircle meets the top of
band S (`z 5.80`), a fillet of radius 0.5. Half the dart's 1.2 width would be 0.6, but the dart's tip round is
0.6 across its radius, so a 0.6 root round degenerates against it; 0.5 is the 0.5 minimum round.

The escape wheel's teeth (toe 109.5°, heel 71.5°, roots rounded 1.4) and the pallet stones (corners 64.9° and blunter)
read no corner finding, so they carry no exemption. The toe, heel and the stones' `C` and `F` are still the engagement
edges, kept as drawn.

## Edge finishing

Faces of 400 mm² or more and their 0.5 chamfers, all through `internal/edgefinish.Chamfer`:

| Part | Chamfered | Kept sharp (reason) |
|---|---|---|
| Back plate, Front plate | outer loop and every window on both faces; pocket rims (`r 7.05`) on the inner face; lip-bore rims (`r 5`) on the outer face | pillar holes, cavities, head pockets (runs under 20 mm) |
| Escape Wheel | the five window loops on both faces | tooth outline (no 20 mm run) |
| Lever | outer loop at `z 3.30` and at `z 9.80` (the band S and band F outlines) | slots (each side under 20 mm, sharp corners); the dart top at `z 5.80` (under 400 mm²) |
| Balance Wheel | outer loop (`r 42`) and the four window loops on both faces, less the four edges on the circle `r 7.5` where the boss's root round meets the front face | the boss face (154 mm²); those four edges, where the round is tangent to the face and there is no corner to chamfer |
| Roller Table, Safety Roller, Retainer, Knob, Knob Collar, Stones, Arbors | none | faces under 400 mm² (roller table 380, knob 380) |
| Hairspring | its caps, by `models/spiralspring` | |

The lever's notch rim loses 0.5 at `z 9.80`, so the pin engages the corner rounds over `z ∈ [6.35, 9.30]`; the dart
loses 0.5 at `z 3.30` and guards over `z ∈ [3.80, 5.80]`.

## Motion and pose

Closed form for `Motion(p)` (`t = Phase`, `Θ = Amplitude`):

- `θ(t) = -Θ cos 2πt`; swing + for `t ∈ [0, 0.5]`, swing − for `t ∈ (0.5, 1)`.
- `ψ = LeverAt(θ, swing)` (Pin-in-fork action). `φ = WheelAt(ψ, swing)` (Action of wheel and pallets), with
  `φ(t) ∈ [18.4047°, 42.4047°)` and `φ` after one period larger by `τ`.
- Default phase events at `Θ` 180°: `θ1` at `t` 0.22574, `θ2` 0.23954, drop 0.26962, `θ3` 0.27426; mirrored on swing −.
- Default `Phase` 0.25: `θ = 0`, swing +, impulse: `ψ` 0.37662°, `φ` 20.61688° (tooth lift on tooth 2).

`Build` pose (local frames, one rigid transform each, then `Placement`):

| Part | Transform |
|---|---|
| Escape Wheel, Escape Arbor, Winding Knob, Knob Collar | about `O` by math angle `-(φ_b - 0.05°)` |
| Lever, Entry Stone, Exit Stone, Pallet Arbor | about `P` by math angle `-ψ_b` |
| Balance Staff, Safety Roller, Roller Table, Balance Wheel, Hairspring Retainer | about `B` by math angle `γ_f + θ` |
| Hairspring, plates, bearings, bolts, nuts | fixed |

`ψ_b` backs the lever off its last contact by 0.05°: on a banking `ψ_b = ±(α - 0.05°)`; while the pin pushes a corner
(unlock) `ψ_b = ψ + 0.05°·s`, and while a corner pushes the pin (impulse) `ψ_b = ψ - 0.05°·s`, with `s = +1` on swing +
and `-1` on swing −. `φ_b = WheelAt(ψ_b, swing)`. Default build pose: `ψ_b` 0.32662°, `φ_b - 0.05°` 20.52966°.

The hairspring is drawn relaxed at every phase: a rigid body cannot wind, and turning it with the balance would tear its
eye off the stud. Its hub's D bore is fixed; the staff's flat corners (`r 3.3` at `x 2.55`) can turn 7.45° inside the bore
(flat at 2.80) before touching it, so the pose checks treat the Hairspring–Balance Staff pair as clear only for
`|θ| ≤ 7°`.

## Geometry checks

Tests live in `models/swisslever`. Build each case once per test binary. Never pass meshes or bodies to
`require.Equal`; report counts, names and scalars only.

| Test | Case(s) | Required result |
|---|---|---|
| `TestValidate` | each field out of range, NaN, invalid `Placement` | error naming the field; `Build` returns it before touching the document |
| `TestLayout` | default | every default number of this document to 1e-4 |
| `TestMotion` | `Θ` 180°, 45°, 270° | 4000 samples: `φ(1) - φ(0) = τ` to 1e-9; `φ` falls only while a toe slides on a drawn locking face, by at most the recoils 0.4047° and 0.5803°; `|ψ| ≤ α`; jumps of `φ` only at `ψ_Ef`, `ψ_Xf` (equal to the drops) and of `ψ` only at `θ2`, `-θ2` |
| `TestActionAnalytic` | default | the wheel/stone checker on analytic polygons reproduces Numerical verification: advance 24° ± 1e-6°, drops 1.933 and 1.989 ± 0.05 mm, closed-form `φ` within 1e-6° away from the drops, E1 ≥ 1.0, E2 ≥ 2.0, E3 ≥ 1.0 |
| `TestEscapementAction` | default | the same checker on `SectionOutline` (tolerance 0.02 or the held mesh bound) of the real Escape Wheel and both stones at `z 11.85`: one tooth per period ± 0.05°, two drops of 1.70 mm or more landing on a locking face, closed-form `φ` within 0.05° away from the drops, E1 ≥ 0.9, E2 ≥ 2.0 |
| `TestErrorStudy` | the 13 cases of the print-error table | analytic checker: one tooth per period, drops ≥ 1.55 mm, E1 ≥ 0.4 |
| `TestForkAction` | `Θ` 180° | on sections of the real Lever (`z 7.8`), Roller Table pin (`z 7.8`) and Safety Roller and dart (`z 4.8`), posed by `LeverAt`: the pin never overlaps the lever by more than the section bound; it touches a corner round (gap ≤ bound) on `[θ1, θ3]` and `[-θ3, -θ1]` only; F1 shake 0.25, F2 ≥ 0.75, F4 ≥ 1.30 at the bankings; G1 ≥ 0.95, G2 ≥ 0.25, G3 ≥ 1.9 |
| `TestGuard` | default | guard freedom 2.0° ± 0.01° and less than `κ + σ`; with the lever at `ψ_g`, corner freedom ≥ 0.39 over the whole pin path; contact directions on each round inside its arc |
| `TestSketches` | default | every `Movement.Sketches` entry fully constrained (`internal/sketchcheck`) |
| `TestInventory` | default, Phase 0.6 | 30 solid bodies, unique names, `Named` order = Bodies table, containers as Bodies table, `VerifyContainers().Err()` nil |
| `TestStonesKeyed` | default | each stone fits its slot (gap 0.05 all round); turned 180° about its length, turned over, or put in the other slot it overlaps the lever by 0.3 mm or more |
| `TestCorners` | default | `internal/cornercheck` on every printed body: no finding outside Retained sharp corners; each exemption matches a finding |
| `TestSplitShafts` | default; Phase 0.6 | `splitcheck.Check` and `cornercheck.CheckAt` at `SplitHeights` on the three arbors, as in the Graham model |
| `TestArborCollars` | default | each collar of the table: end faces, cone, length ≥ 3.0, radius 3.7; body steps `r 3.3` with the 0.3 land and the part face ≥ 1.0 from the flush side; Knob Collar as the Graham model's |
| `TestPressFitWalls` | default | wall ≥ 1.6 round the bore of the safety roller (1.62 at the crescent), roller table, balance wheel hub, retainer, lever hub, Knob Collar and knob floor; ≥ 1.6 round each slot (1.95) |
| `TestInstallability` | default | every between-plate part with a body-family bore passes a 605ZZ seat (bore wall ≥ 2.54 from its axis) |
| `TestFits` | default | each Fits row's lower bound (`internal/meshgap` or `SectionOutline.Clearance`) ≥ its class minimum; press rows 0.05 ± bound; T1 radius 2.5 ± bound |
| `TestFasteners` | default | heads at `z 33` ± 1e-6 in their pockets; nuts inside their cavities; tips inside the pillars; no overlap; 1 mm of material round pockets and cavities |
| `TestHairspring` | default | the `Derive` numbers above, `k` 1.8312, eye centre at (-32.0106, 92.5714) ± 1e-4, eye hole 0.40 round the stud |
| `TestEdgeFinishing` | default | each chamfered part lighter than its unchamfered build by the bevel volume; kept loops keep their edges |
| `TestPlacement` | origin `(10, -5, 7)`, `U = +Y`, `V = +Z` | same volumes; bounds follow the frame |
| `TestPoseNoOverlap` | Phase 0, 0.1, 0.226, 0.24, 0.25, 0.27, 0.6, 0.75 | no two bodies overlap (wheel–stones and pin–fork by section, others by `meshgap`); Hairspring–Balance Staff checked only where `|θ| ≤ 7°` |
| `TestCentreLine` | default | `O`, `P`, `B` at `x = 0` to 1e-9; plates symmetric about `x = 0` within the section bound |

The checkers are test helpers. Wheel/stone checker: drive `ψ` with the dwell law `ψ = clamp(-1.25 α cos 2πt, -α, α)`,
2000 steps per period, 3 periods, measure periods 2 and 3; prune teeth to those within 40° of `I_L` or `I_R`; at each
step resolve an overlap by the smallest turn either way (a stone pushing a tooth from behind turns the wheel forward),
then advance by 0.02° and bisect to 1e-9 rad. Fork checker: sample `θ` from `-60°` to `60°` in 0.03° steps on both
swings.

## Parts that look alike

This design declares no look-alike set, so no marks are cut. The Pallet Arbor and the Balance Staff are the same part.
The Escape Arbor is 42.25 long with its shoulder at the front; the other two are 37 long with the shoulder at the rear.
The two stones differ in section and each enters only its own slot (TestStonesKeyed); they have no round face to mark.
The README parts table states these differences.

## Materials and print orientation

| Part | Material | Orientation |
|---|---|---|
| Hairspring | PETG | flat, axis vertical (`models/spiralspring`) |
| Entry Stone, Exit Stone | PLA or PETG (red suggested) | lying on the back face, length along the bed |
| Impulse pin (on the Roller Table) | as the roller table | roller table front face down, pin up |
| Arbors | PLA | split halves on their cut faces |
| Lever | PLA | rear face (band S, with the dart) down |
| Escape Wheel, Balance Wheel | PLA | rear face down |
| Safety Roller, Retainer, Knob Collar, Knob | PLA | axis vertical, Knob front face down |
| Back plate, Front plate | PLA | back plate rear face down (print pause for the nuts); front plate front face down |

Assembly: press the stones into the lever flush with its rear face; press the lever on the pallet arbor; build the staff
stack from the front; drop the nuts in the back plate's pause; put the arbors into the back bearings, the front bearings
into the front plate, hook the hairspring eye on the stud as the front plate goes on, bolt the plates, press on the Knob
Collar and the knob. The hub's D flat sets the beat: at rest the pin lies on the line of centres.
