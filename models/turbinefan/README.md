# Turbine fan

**Build status:** Built and tested. A blade for the 192 mm rotor has been printed with its nose rib, with the best
quality of any blade print so far.

![Three-blade turbine fan on a clamping hub, seen from the front and above](images/hero.png)

A three-blade fan whose blades are shaped like small wind turbine blades: with a thick round
root, a widest section near the hub, and a taper to a thin tip. The rotor is 192 mm across by default, and any size from
128 to 224 mm works. The [hand-crank fan](../handcrankfan/) uses the same blades at the 128 mm size. The hub is a plain
round cylinder that holds a D-flat shaft with one M2 bolt used as a set screw, and each blade bolts into the hub on its
own, so a blade can be swapped. The same hub also bolts to the adapter ring of the [one-way clutch](../onewayclutch/) with
two more M2 bolts, instead of gripping a shaft, so the fan coasts when the shaft stops.

The fan comes in two hands. Seen from the front, the right-hand fan (the default) turns clockwise and the left-hand fan
counter-clockwise; either blows air forward, away from the mechanism. The left-hand blades are the right-hand blades
reflected, so a left-hand fan takes three left-hand blades. The pictures show the right hand.

The hub fits a shaft that runs in a purchased bearing, because the shaft's diameter equals the bearing's bore. The
default fits the 5 mm shaft of a 605ZZ bearing; a second preset fits the 8 mm shaft of a 608ZZ bearing.

**Clearance adjustment:** Mixed. The bore's clearance is a parameter; the blade fits, bolts and nuts keep their nominal size.

## Views

From the front, the D bore shows through the hub's front face, with the blade bolts' heads flush beside it.

![The fan from the front, with blade 1 up and a bolt head in the hub's front face for each blade](images/front.png)

From behind, the D bore shows in the back face, which goes against the mechanism, with its flat toward the set screw and
the set screw nut's slot in the back face. Two more small holes, at 30° and 150°, are the blind bolt holes that take the
clutch adapter's bolts.

![The fan from behind, showing the D bore, its flat toward the set screw, and the set screw nut's slot in the back face](images/back.png)

From the side, the set screw's counterbore sits in the round face, the blade nut slots open below the blades (the two
adapter nut slots open there too, nearer the back face), and the blades show their twist.

![The fan from the side, showing the set screw's counterbore in the round face, the blade nut slots below the blades, and the twist of the blades](images/side.png)

## The set screw

![The hub cut across the set screw's line: the M2 x 10 set screw in its counterbore, its nut in the slot from the back face, and its tip on the shaft's flat in the D bore](images/setscrew.png)

One M2 x 10 bolt (M2 x 6 on the 128 mm rotor) runs into the hub from the side, through a nut in a slot open to the back face, and presses the shaft's
flat. Its head sits below the round face, so nothing stands out of the hub. The D flat turns the hub with the shaft; the
screw stops the hub sliding along the shaft and rattling. The gallery's rules ask for a few sharp edges to stay on the hub,
which [instructions.md](instructions.md) lists: the D bore's two corners and the nut slot's end corner.

## Replacing a blade

Each blade has a hexagonal stud that slides into a radial socket of the same shape in the hub, with 0.25 mm between their flats in the hub. One M2 x 25 bolt (M2 x 16 on the 128 mm rotor) runs parallel to the shaft, from
the hub's front face through the hub and the stud into an M2 nut. The nut sits in a slot open to the round face, below the
blade, so you can fit it with the hub on its shaft.

![The hub cut through blade 1's radial line and the shaft axis: the stud in its socket, the bolt through the hub and the stud, its nut in the slot behind, and the set screw](images/joint.png)

To remove a blade:

1. Undo its bolt from the front face. The nut is free in its slot: take it out through the slot's mouth below the blade
   root, or leave it.
2. Pull the blade out along its radial line. The stud leaves the socket after about 11 mm.

To fit a blade:

1. Slide an M2 nut, flat, into the slot's mouth in the round face below the socket until it stops at the slot's end.
2. Slide the stud into the socket with the blade's concave face toward the front face. The stud is a hexagon, so a blade also
   goes in turned half a turn, but then it faces the wrong way.
3. Push the bolt in from the front through the stud into the nut and tighten it.

All blades of one hand are one part and any of them fits any socket. The hub can stay on the shaft. To take the hub off its
shaft, loosen the set screw about two turns; it stays in its nut.

## Exploded

![The fan pulled apart: the three blades, the hub, the three blade bolts in front, the three blade nuts pulled out of their slots, and the set screw and its nut](images/exploded.png)

The set screw nut sits behind the hub in that view, so the same parts are shown again from behind.

![The exploded fan from behind: the hub, the three blades, the three blade nuts, the set screw nut and the set screw](images/exploded-back.png)

## Parts

| Image | Name |
| --- | --- |
| <img src="images/parts/hub.png" alt="Hub" width="240"> | Hub (printed), with or without the clutch adapter |
| <img src="images/parts/blade.png" alt="Blade" width="240"> | Blade (printed), 3 |
| <img src="images/parts/bolt.png" alt="M2 x 25 bolt" width="240"> | M2 × 25 socket head cap screw ([`models/fastener`](../fastener)), 3: the blade bolts |
| <img src="images/parts/setscrew.png" alt="M2 x 10 bolt" width="240"> | M2 × 10 socket head cap screw ([`models/fastener`](../fastener)), 1: the set screw |
| <img src="images/parts/nut.png" alt="M2 nut" width="240"> | M2 hex nut ([`models/fastener`](../fastener)), 4 |
| <img src="images/parts/harness.png" alt="Nose rib, a thin wedge-shaped wall that is taller at one end" width="240"> | Nose rib (printed with each blade, removed and discarded) |

## Sizes

The 605ZZ hub (left) and the 608ZZ hub (right), from behind. They differ in the bore. Both are 37.8 mm across and 30 mm long.

![The 605ZZ hub and the 608ZZ hub side by side, from behind](images/sizes.png)

## On a one-way clutch

With `Mount` set to `ClutchMount` (`ParamsForClutch` fills it in from a clutch), the fan uses the same hub, with no flange and
no clamp ring, but it holds no shaft. Two M2 × 10 bolts from the rear of the [one-way clutch's](../onewayclutch/) adapter ring
run into nuts in the hub's two adapter slots, and the adapter bolts to the clutch's ears as its own README describes. The
shaft drives the fan through the clutch, and the fan coasts when the shaft stops or turns back. The fan is turned 180 degrees
from the clutch, so the hub's bore flat and the shaft's flat point the same way and each ear bolt lies midway between two
blades. A hex key reaches it from the front. The pictures show the fan on the default (605ZZ, clockwise) clutch with its adapter.

![The clutch-mount fan on the one-way clutch and its adapter ring, seen from the front and above](images/clutch.png)

![The clutch-mount fan on the one-way clutch, seen from behind, with the three ear nuts in the ring's rear pockets](images/clutch-back.png)

The cutaway is cut by the plane x = 0, which holds one ear bolt, one adapter bolt and one of the clutch's own bolts.

![The clutch and fan cut through an ear bolt and an adapter bolt: the ear bolt through the adapter and the ring's ear into its nut, the adapter bolt from the adapter's rear face into its nut in the hub, and a clutch bolt beside them](images/clutch-cutaway.png)

![The clutch-mount fan pulled apart along the shaft, in assembly order from the left: the three ear bolts, the three blade bolts, the three blades with their nuts, the hub, the two adapter nuts, the two adapter bolts, the adapter ring, the clutch and the three ear nuts](images/clutch-exploded.png)

**Hand and freewheel.** The ring is driven when the shaft turns the clutch's `Freewheel` direction and coasts that way about a
stopped shaft. A right-hand fan blows forward turning clockwise seen from the front.

| Fan `Hand` | Clutch `Freewheel` | Result |
| --- | --- | --- |
| `RightHand` | `Clockwise` | the shaft drives the fan forward and the fan coasts on when the shaft stops (matched) |
| `LeftHand` | `CounterClockwise` | the same, counter-clockwise (matched) |
| `RightHand` | `CounterClockwise` | the shaft drives the fan backward, toward the mechanism (mismatched) |
| `LeftHand` | `Clockwise` | backward, as above (mismatched) |

`ParamsForClutch` takes `WithBladePitch` to keep a chosen pitch. It always returns a matched pair; a hand-built pair is not checked.

**Mounting space.** Leave room to hold an M2 nut in each of the ring's three rear ear pockets (r 19 to 24 mm at the ring's
rear face). The hub's rim stays 2.7 mm clear of each ear bolt's line, so a bolt head and a hex key pass it from the
front. In front are the adapter (51.4 mm across on the 605ZZ clutch, 53.0 to 53.9 mm on the 8 mm clutch, 6 mm thick) and
the fan. The shaft may end anywhere from inside the
clutch to the adapter's front face. It must not enter the hub: a shaft in the hub's D bore would lock the fan to the shaft and
defeat the clutch. The adapter's bore (7 mm on the 605ZZ clutch, 10 mm on the 8 mm clutch) clears it.

**Assembly.** The adapter's bolts into the hub cannot be reached once the adapter is on the clutch, and the adapter and the
hub cover the clutch's own bolt heads, so work in this order.

1. Fit the blades to the hub, as above.
2. Slide an M2 nut into each of the hub's two adapter slots from the round face. Set the hub on the adapter's front face,
   with the adapter's face pockets toward the back, and drive two M2 × 10 bolts from the adapter's rear pockets into the nuts.
3. Assemble the clutch off the shaft (see the clutch's README), press it on the shaft and tighten its three bolts. The
   2.6 mm access holes in the adapter pass a 1.5 mm hex key to those heads if you need to retighten them.
4. Hold an M2 nut in each of the ring's three rear ear pockets, set the adapter against the ring's front end and drive the
   three M2 × 25 ear bolts from the front between the blades with a 1.5 mm hex key.

**Printing.** The hub prints the same way as the plain hub: front face down, axis vertical, no supports.

| Image | Name |
| --- | --- |
| <img src="images/parts/adapter-bolt.png" alt="M2 x 10 bolt" width="240"> | M2 × 10 socket head cap screw ([`models/fastener`](../fastener)), 2: the adapter bolts |
| <img src="images/parts/ear-bolt.png" alt="M2 x 25 bolt" width="240"> | M2 × 25 socket head cap screw ([`models/fastener`](../fastener)), 3: the ear bolts, with the adapter |

The clutch mount takes the hub, blades, blade bolts and blade nuts above, the adapter ring from the one-way clutch (printed,
`onewayclutch.BuildAdapter`), two M2 × 10 adapter bolts and two M2 nuts for them ([`models/fastener`](../fastener)), and the three
M2 × 25 ear bolts and three nuts that hold the adapter to the clutch. It needs no set screw. The hub's set screw hole and
slot stay empty.

## Parameters

All dimensions are in millimeters, angles in radians. `Params.Validate` rejects a value outside its limit with an error
that names it.

| Parameter | 605ZZ (default) | 608ZZ | Limit |
| --- | --- | --- | --- |
| `ShaftDiameter` | 5 | 8 | Above 0; keeps a 1.6 mm wall to the blade sockets (at most 12.1 at 192 mm, 8.1 at 128 mm) |
| `FlatDepth` | 0.5 | 0.75 | Above 0, at most a quarter of `ShaftDiameter`, and leaving the shaft's flat 2 mm and the bore's flat 2.8 mm wide |
| `BoreClearance` | 0.05 | 0.05 | 0.05 to 0.5; for the 8 mm shaft 0.05 to 0.1 |
| `RotorDiameter` | 192 | 192 | 128 to 224; it sizes the hub, the blades, the blade bolts and the set screw |
| `BladePitch` | 25° | 25° | 10° to 40°, the blade angle at 75 % of the tip radius (72 mm at 192) |
| `Hand` | `RightHand` | `RightHand` | `RightHand` or `LeftHand` |
| `Mount` | `ShaftMount` | `ShaftMount` | `ShaftMount` or `ClutchMount`; `ClutchMount` bolts the hub to a clutch's adapter ring and leaves out the set screw |
| `Placement` | none | none | A valid frame; it puts the hub's front face (`ShaftMount`) or the clutch cover's outer face (`ClutchMount`) on its origin. None leaves the parts in the build frame |

At the default 192 mm the hub is a cylinder 37.8 mm across and 30 mm long and the rotor is 192 mm across; at 128 mm the
hub is 28 mm across and 20 mm long. The hub's radius stops growing at 18.9 mm, so above 192 mm the hub keeps that width and
only grows longer (35 mm at 224 mm). The cap keeps a hex key's way to the ear bolts of the smallest
[one-way clutch](../onewayclutch/). The set screw is M2 × 10 (M2 × 6 at 128 mm) and the blade bolts are M2 × 25 (M2 × 16 at
128 mm), the longest standard lengths that fit. One hub serves both mounts.

## Printing

Print the hub front face down, with its bore axis vertical. The hub needs no print pause: every nut sits in a slot open to a
face, so you put the nuts in after printing.

Print each blade lying on the bed, leading edge down. The blade's root end is tilted about 13° up from the bed (13.54°
at 128 mm, 13.35° at 192 mm, 13.75° at 224 mm, at the default pitch), so it rests on the bed at two small spots: a flat on the end of the
stud, and the leading edge near the widest part of the blade. The layers then run along the blade, which is the strong
direction for the load it carries in use. There is no raft.

![The blade lying on the bed, seen from the stud end on the pressure side, with the nose rib standing on the bed under its leading edge from the middle of the blade to the tip](images/print.png)

![The same print seen from the suction side, tip on the left: the nose rib's sloped top follows the blade's leading edge just below it, and the blade rests on the bed at its stud flat and its leading edge](images/print-side.png)

![The print seen from the tip end: the nose rib, a narrow wall that is wider at the bed, stands under the blade's leading edge](images/print-end.png)

![The first layer seen from above, tip on the right: the blade's two bed spots, on the stud flat and on the leading edge, and the long, narrow footprint of the nose rib, which widens toward the tip](images/print-bed.png)

A blade has printed and come out usable with this setup:

- Slicer: OrcaSlicer.
- Filament: PETG for the blade.
- Support: slicer support, build plate only, under the root and the stud, with PLA as the support interface.
- Brim: outer only, 5 mm, with a 0.1 mm brim gap.
- Layers: 0.08 or 0.12 mm.
- Walls: 4. Infill: 100 %.

On that print, the leading edge drooped 1 to 2 mm from about r 40 mm to the tip: the slicer put no support under it,
because the strip of the leading edge that faces down is only 0.3 to 1.4 mm wide. So the model now builds a nose rib with
each blade. The rib is a thin wall that stands on the bed under the leading edge, with its top 0.16 mm below the blade.
At 192 mm it holds the leading edge from about r 43 mm to r 93 mm. The stretch from r 40 to 43 mm and the last 2 to 3 mm
of the tip have no support. The rib takes 1,180 mm³ at 192 mm, about 0.2 times the blade.

A 192 mm rotor's blade has been printed with the rib, and it came out with the best quality of any blade print so far.
To print a blade with it:

1. Build the blade and its rib with `BuildBladePrint` (see the [developer notes](../../docs/models/turbinefan.md)).
   Load them as two separate objects, not as parts of one object, so each takes its own brim setting.
2. Keep the settings above for the blade. In the object list, set the rib's brim type to "No-brim". The rib starts 6.05
   to 6.24 mm from the blade's leading-edge spot, so the blade's brim does not reach it.
3. Print the rib in the blade's filament, so it needs no filament change.
4. Use 0.08 mm layers when you print the rib, so its 0.16 mm gap is two whole layers.
5. After printing, lift the blade off. Pull the rib away sideways, then peel the brim. Lightly sand the line the rib
   leaves along the leading edge. Check that the stud slides into a hub socket.

To assemble:

1. Hold the hub back face up and drop an M2 nut into the set screw slot. It stops at the slot's hexagonal end, in line with
   the hole.
2. Push the M2 x 10 set screw into the counterbore on the side of the flat and turn it into the nut about two turns. Stop
   while its tip is still out of the bore (look into the bore from the back).
3. Fit each blade as described above.
4. Press the hub onto the shaft, flat to flat, the flat toward the set screw, at least 6 mm deep.
5. Tighten the set screw with a 1.5 mm hex key until its tip presses the flat.

For how to build this fan from Go, see the [developer notes](../../docs/models/turbinefan.md). The construction is in
[instructions.md](instructions.md) and [steps.md](steps.md).
