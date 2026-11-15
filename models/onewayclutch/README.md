# One-way clutch

![A one-way clutch cut at the hub's mid-height: three pawl heads seated in the teeth of the ring, the arms behind them, and the back plate and nuts below](images/hero.png)

A printed one-way clutch for a D-flat shaft. A hub with three flexing pawls presses on the shaft and meets a 42 mm ring
with 24 internal teeth. Turn the shaft one way and the pawls catch the teeth, so the ring turns with it. Turn the shaft
the other way and the teeth push the pawls inward, the arms bend, and the ring stays still with a click for each tooth.
A back plate and a cover hold the ring on the hub. Three M2 × 16 bolts and nuts clamp the stack. Every printed part
prints with its axis vertical and no supports.

You choose the shaft from three presets (`605ZZ`, `608ZZ` and `608ZZ-gearbox`) and the direction in which the ring turns
freely. The shaft is not part of this model: a shaft that runs in a bearing has the bearing's bore as its diameter, and
the mechanism you build owns it.

**Clearance adjustment:** Mixed. The bore clearance is a parameter; the bolts and nuts keep their nominal size.

## Views

From the front, the cover shows the three bolt heads, flush in their pockets, round the shaft's D hole.

![The clutch from the front: the cover with three bolt heads and the D bore](images/front.png)

From the back, the back plate's outer face goes against the mechanism.

![The clutch from the back: the back plate with the D bore](images/back.png)

From the side, the ring's outside is 42 mm across and has chamfered ends.

![The clutch from the side](images/side.png)

## How it works

![The clutch cut at the hub's mid-height and seen along the shaft: the hub's core with three arms, each ending in a head seated in a gap between the ring's teeth](images/section.png)

The picture cuts the clutch at the hub's mid-height and looks along the shaft.

- Each pawl is a thin curved arm printed in one piece with the hub. The arm ends in a head, and the head sits in the
  gap between two of the ring's 24 teeth.
- Each tooth has a radial lock face on one side and a ramp on the other. Turning the hub toward the lock face presses
  the head against it, and the ring turns with the hub.
- Turning the hub the other way, the ramp presses the head inward. The arm bends in the plane of the print, so the
  bending loads the printed lines along the arm, and the head clicks over the tooth.
- A slot behind each arm gives it room to bend. The arm is 1.2 mm thick.

![The clutch cut through the shaft axis: the plates, the ring held between them by cones, a bolt and its nut](images/cutaway.png)

The second cut runs through the shaft axis. Each plate has a rim that fits a counterbore in the ring's end, with a 45°
cone beside it, so the plates hold the ring between them and it cannot slide along the shaft. The ring keeps 0.4 mm of
radial gap to each rim.

## Freewheel direction

`Freewheel` is the direction the ring turns freely about a held shaft, seen from the front (the cover). The shaft
turning the same way drives the ring along with it; the shaft turning the other way slips under it. The two directions
are mirror images: the hub and the ring are mirrored, and the plates, bolts and nuts are the same.

![Two clutches cut at the hub's mid-height: clockwise freewheel on the left, counter-clockwise freewheel on the right](images/directions.png)

The left clutch (`Clockwise`) lets the ring turn clockwise about a held shaft, seen from the front. The right clutch
(`CounterClockwise`) lets it turn counter-clockwise.

## Exploded

![The clutch pulled apart along the shaft axis, from the back plate's side on the left: three M2 nuts, the back plate, the hub, the ring, the cover, and three M2 x 16 bolts](images/exploded.png)

From the left: the three M2 nuts, the back plate, the hub, the ring, the cover, and the three M2 × 16 bolts.

## Parts

| Image | Name |
| --- | --- |
| <img src="images/parts/hub.png" alt="Hub" width="240"> | Hub with its three pawls (printed) |
| <img src="images/parts/ring.png" alt="Ring" width="240"> | Ring with 24 internal teeth (printed) |
| <img src="images/parts/back-plate.png" alt="Back plate" width="240"> | Back plate, with three nut cavities (printed) |
| <img src="images/parts/cover.png" alt="Cover" width="240"> | Cover, with three head pockets (printed) |
| <img src="images/parts/bolt.png" alt="M2 x 16 bolt" width="240"> | M2 × 16 socket head cap screw ([`models/m2bolt`](../m2bolt)), 3 |
| <img src="images/parts/nut.png" alt="M2 nut" width="240"> | M2 hex nut ([`models/m2nut`](../m2nut)), 3 |

## Sizes

The three presets differ only in the D bore of the hub and the two plates. From the left: `605ZZ`, `608ZZ` and
`608ZZ-gearbox`.

![Three clutches cut at the hub's mid-height, with the D bores of the three shaft presets](images/sizes.png)

## Parameters

All dimensions are in millimeters. `Params.Validate` rejects a value outside its limit with an error that names it.

| Parameter | Default | Meaning and limit |
| --- | --- | --- |
| `Shaft` | `605ZZ` | A preset name: `605ZZ` (5 mm shaft, 0.5 mm flat), `608ZZ` (8 mm, 0.75 mm flat) or `608ZZ-gearbox` (8 mm, 0.5 mm flat) |
| `Freewheel` | `Clockwise` | `Clockwise` or `CounterClockwise`; the zero value is rejected |
| `BoreClearance` | 0.05 | Radial clearance of every D bore, 0.05 to 0.5, and small enough to keep 1.6 mm of wall between the bore and the nut cavities |
| `Placement` | none | A valid frame |

The ring is 42 mm across and the stack is 19.5 mm high. Neither depends on the preset.

## Printing

Print the hub in PETG, since its arms flex at every click; PLA works too. Print the hub and the ring on either face, and
the cover with its outer face down, with no supports. Print the back plate with its outer face down and pause the print
just above the nut cavities, at 3.5 mm, to drop an M2 nut into each one.

To assemble:

1. Press the back plate onto the shaft, its outer face toward the mechanism.
2. Press the hub onto the shaft.
3. Push each pawl head in with a finger and slide the ring over the hub, turning it in the freewheel direction.
4. Press the cover onto the shaft.
5. Drive the three M2 × 16 bolts through the cover and the hub into the nuts, and tighten them.

For how to build this clutch from Go, see the [developer notes](../../docs/models/onewayclutch.md). The construction is
in [instructions.md](instructions.md) and [steps.md](steps.md).
