# Turbine fan

**Build status:** Not built yet. No physical copy of this model has been made and tested, so its dimensions and fits are unproven.

![Three-blade turbine fan on a clamping hub, seen from the front and above](images/hero.png)

A three-blade fan whose blades are shaped like small wind turbine blades: with a thick round
root, a widest section near the hub, and a taper to a thin tip. The [hand-crank fan](../handcrankfan/) uses the same
blades. The hub is a plain round cylinder that holds a D-flat shaft with one M2 bolt used as a set screw, and each blade bolts into the hub on its own, so a blade can
be swapped.

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
the set screw nut's slot in the back face.

![The fan from behind, showing the D bore, its flat toward the set screw, and the set screw nut's slot in the back face](images/back.png)

From the side, the set screw's counterbore sits in the round face, the blade nut slots open below the blades, and the
blades show their twist.

![The fan from the side, showing the set screw's counterbore in the round face, the blade nut slots below the blades, and the twist of the blades](images/side.png)

## The set screw

![The hub cut across the set screw's line: the M2 x 6 set screw in its counterbore, its nut in the slot from the back face, and its tip on the shaft's flat in the D bore](images/setscrew.png)

One M2 x 6 bolt runs into the hub from the side, through a nut in a slot open to the back face, and presses the shaft's
flat. Its head sits below the round face, so nothing stands out of the hub. The D flat turns the hub with the shaft; the
screw stops the hub sliding along the shaft and rattling. The gallery's rules ask for a few sharp edges to stay on the hub,
which [instructions.md](instructions.md) lists: the D bore's two corners and the nut slot's end corner.

## Replacing a blade

Each blade has a round stud that slides into a radial socket in the hub. One M2 x 16 bolt runs parallel to the shaft, from
the hub's front face through the hub and the stud into an M2 nut. The nut sits in a slot open to the round face, below the
blade, so you can fit it with the hub on its shaft.

![The hub cut through blade 1's radial line and the shaft axis: the stud in its socket, the bolt through the hub and the stud, its nut in the slot behind, and the set screw](images/joint.png)

To remove a blade:

1. Undo its bolt from the front face. The nut is free in its slot: take it out through the slot's mouth below the blade
   root, or leave it.
2. Pull the blade out along its radial line. The stud leaves the socket after about 8 mm.

To fit a blade:

1. Slide an M2 nut, flat, into the slot's mouth in the round face below the socket until it stops at the slot's end.
2. Slide the stud into the socket with the blade's concave face toward the front face. The stud is round, so a blade also
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
| <img src="images/parts/hub.png" alt="Hub" width="240"> | Hub (printed) |
| <img src="images/parts/blade.png" alt="Blade" width="240"> | Blade (printed), 3 |
| <img src="images/parts/bolt.png" alt="M2 x 16 bolt" width="240"> | M2 × 16 socket head cap screw ([`models/m2bolt`](../m2bolt)), 3: the blade bolts |
| <img src="images/parts/setscrew.png" alt="M2 x 6 bolt" width="240"> | M2 × 6 socket head cap screw ([`models/m2bolt`](../m2bolt)), 1: the set screw |
| <img src="images/parts/nut.png" alt="M2 nut" width="240"> | M2 hex nut ([`models/m2nut`](../m2nut)), 4 |

## Sizes

The 605ZZ hub (left) and the 608ZZ hub (right), from behind. They differ in the bore.

![The 605ZZ hub and the 608ZZ hub side by side, from behind](images/sizes.png)

## Parameters

All dimensions are in millimeters, angles in radians. `Params.Validate` rejects a value outside its limit with an error
that names it.

| Parameter | 605ZZ (default) | 608ZZ | Limit |
| --- | --- | --- | --- |
| `ShaftDiameter` | 5 | 8 | Above 0; keeps a 1.6 mm wall to the blade sockets (at most 8.1) |
| `FlatDepth` | 0.5 | 0.75 | Above 0, at most a quarter of `ShaftDiameter`, and leaving the shaft's flat 2 mm and the bore's flat 2.8 mm wide |
| `BoreClearance` | 0.05 | 0.05 | 0.05 to 0.5; for the 8 mm shaft 0.05 to 0.1 |
| `BladePitch` | 25° | 25° | 10° to 40°, the blade angle at 48 mm |
| `Hand` | `RightHand` | `RightHand` | `RightHand` or `LeftHand` |
| `Placement` | none | none | A valid frame |

The hub is a cylinder 28 mm across and 20 mm long. The rotor is 128 mm across.

## Printing

Print the hub front face down, with its bore axis vertical, and each blade standing on its stud end. The hub needs no print
pause: every nut sits in a slot open to a face, so you put the nuts in after printing.

To assemble:

1. Hold the hub back face up and drop an M2 nut into the set screw slot. It stops at the slot's hexagonal end, in line with
   the hole.
2. Push the M2 x 6 set screw into the counterbore on the side of the flat and turn it into the nut about two turns. Stop
   while its tip is still out of the bore (look into the bore from the back).
3. Fit each blade as described above.
4. Press the hub onto the shaft, flat to flat, the flat toward the set screw, at least 6 mm deep.
5. Tighten the set screw with a 1.5 mm hex key until its tip presses the flat.

For how to build this fan from Go, see the [developer notes](../../docs/models/turbinefan.md). The construction is in
[instructions.md](instructions.md) and [steps.md](steps.md).
