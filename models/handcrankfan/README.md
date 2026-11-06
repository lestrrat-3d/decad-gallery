# Hand-crank fan

A handheld fan. You hold the grip in your left hand and turn the crank on the right with your right hand. The fan
faces you and blows air toward you.

![Hand-crank fan seen from the operator's side](images/hero.png)

![The fan exploded by Decad along its joints: grip and its screws, gearbox, crank shaft, arm, handle and pin, and the fan shaft, hub, blades and bolts](images/exploded.png)

A straight-tooth bevel gear pair turns the crank's rotation through 90° and doubles its speed: one forward
crank turn (clockwise, seen from the right) turns the four-blade rotor twice, counter-clockwise as the
operator sees it. Each blade is shaped like a small wind turbine blade: a thick round root, a widest section near
the hub, and a taper to a thin tip, with the angle falling from the root to the tip. Its concave face leads the
motion and faces the operator.

Each blade is a separate part with a round stud that slides into a socket in the hub. One M2 x 16 bolt and nut
hold it, so a damaged blade comes off and is replaced. Everything else is meant to be 3D-printed.
The grip hangs under the gearbox. It is an oval of about 35 to 38 mm, raked like a pistol grip, with a flared base
and a palm swell, and two M2 screws hold it. The crank handle is a small 50 mm barrel, 16 mm thick, for the thumb, index and middle fingers. It spins on a headed pin, so it
neither slides off nor wobbles, and the arm is clamped to the shaft and the pin by two more M2 screws. Outside
corners are rounded. The rotor has no guard.

The gears are the [bevel gear pair](../bevelgear/). The gears sit on plain shafts with no keys or retaining
hardware. The gearbox is a tray with thick flat walls and no boss or sleeve, so it prints floor down without
support.

You can vary:

- The crank angle, which also sets the fan angle at twice that value.
- The blade pitch, from 10° to 40°, which turns each whole blade; it is the blade angle at three quarters of the radius.
- The number of blades, 3 or 4.

For how to build it from Go, see the [developer notes](../../docs/models/handcrankfan.md).
