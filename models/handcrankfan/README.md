# Hand-crank fan

A handheld fan. You hold the grip in your left hand and turn the crank on the right with your right hand. The fan
faces you and blows air toward you.

![Hand-crank fan seen from the operator's side](images/hero.png)

![One crank turn: the crank side on the left and the gears, with the crank arm removed, on the right](images/motion.gif)

![The fan exploded by Decad along its joints: grip and its screws, gearbox, spur and bevel gears, shafts, arm, handle and pin, and the fan shaft, hub, blades and bolts](images/exploded.png)

The crank shaft sits 40 mm below the fan's axis, so the crank handle swings clear of the blades. A pair of
equal spur gears inside the gearbox carries its turn up to the drive shaft, and a straight-tooth bevel gear pair turns that through 90°
and doubles its speed: one forward crank turn (clockwise, seen from the right) turns the four-blade rotor
twice, clockwise as the operator sees it. The spur pair reverses the turn, so the blades are mirrored to push air
toward the operator. Each blade is shaped like a small wind turbine blade: a thick round root, a widest section near
the hub, and a taper to a thin tip, with the angle falling from the root to the tip. Its concave face leads the
motion and faces the operator.

Each blade is a separate part with a round stud that slides into a socket in the hub. One M2 x 16 bolt and nut
hold it, so a damaged blade comes off and is replaced. Everything else is meant to be 3D-printed, except seven 608-2RS
ball bearings, one wherever a shaft crosses a wall.
The grip hangs under the gearbox. It is an oval of about 35 to 38 mm, raked like a pistol grip, with a flared base
and a palm swell, and two M2 screws hold it. The grip starts below the lowered floor, and the crank sits at the height of the gearbox's lower half, 40 mm below the fan. The crank handle is a small 50 mm barrel, 16 mm thick, for the thumb, index and middle fingers. It spins on a headed pin, so it
neither slides off nor wobbles, and the arm sits on the shaft's splines in a tight-fit bore, with one more M2 screw holding the pin. Outside
corners are rounded. The rotor has no guard.

The gears are the [bevel gear pair](../bevelgear/) and two [spur gears](../spurgear/). Each shaft is 8 mm
with three grooves, and each gear, the crank arm and the fan hub have a bore with three teeth that sit in them, so six
faces carry the torque and the round shaft still slides through every bore. The gearbox is one flat-walled tray with a post across its left end and a back wall, 51 mm
wide, 88 mm tall and 80 mm deep, not counting the bearing blocks. The spur gears sit in it left of the bevel gear, and
both shafts slide in through the post and carry a collar outside it. A bearing sits in a seat in the post, in the crank-side wall,
in the crank block and in the fan wall and block, so there is no boss or sleeve. The tray prints in two halves that
meet on the plane through the crank shaft and the drive shaft, so each of those seats is an open half-trough in each half and
no bore lies on its side. A half prints with its flat face up, the front half on the fan block and the back half on the
back wall. The fan shaft is vertical in the front half, and the one seat that opens onto the bed has a 45° cone for a
ceiling. Four M2 screws and nuts, in ears on the outside, hold the halves together. The hub's blade sockets are plain
6 mm holes, small enough to print on their side.

## Parts

Each part is shown once, from the real model, in place at the default pose. A fan has one of each part except
the gearbox screws, the grip screws, the bearings, and the blades with their bolts, which are repeated as the names say. The bought parts are
the screws, the nuts and the bearings. Everything else is printed.

| Image | Name |
| --- | --- |
| <img src="images/parts/gearbox-front.png" alt="Gearbox, front half" width="240"> | Gearbox, front half |
| <img src="images/parts/gearbox-back.png" alt="Gearbox, back half" width="240"> | Gearbox, back half |
| <img src="images/parts/gearbox-screw.png" alt="Gearbox screw and nut, M2 x 12 (four)" width="240"> | Gearbox screw and nut, M2 x 12 (four) |
| <img src="images/parts/grip.png" alt="Grip" width="240"> | Grip |
| <img src="images/parts/grip-screw.png" alt="Grip screw, M2 x 16 (two)" width="240"> | Grip screw, M2 x 16 (two) |
| <img src="images/parts/crank-gear.png" alt="Crank gear" width="240"> | Crank gear |
| <img src="images/parts/drive-gear.png" alt="Drive gear" width="240"> | Drive gear |
| <img src="images/parts/driving-gear.png" alt="Driving bevel gear" width="240"> | Driving bevel gear |
| <img src="images/parts/pinion.png" alt="Pinion bevel gear" width="240"> | Pinion bevel gear |
| <img src="images/parts/crank-shaft.png" alt="Crank shaft" width="240"> | Crank shaft |
| <img src="images/parts/drive-shaft.png" alt="Drive shaft" width="240"> | Drive shaft |
| <img src="images/parts/fan-shaft.png" alt="Fan shaft" width="240"> | Fan shaft |
| <img src="images/parts/bearing.png" alt="Ball bearing, 608-2RS (seven)" width="240"> | Ball bearing, 608-2RS (seven) |
| <img src="images/parts/crank-arm.png" alt="Crank arm" width="240"> | Crank arm |
| <img src="images/parts/crank-pin.png" alt="Crank pin" width="240"> | Crank pin |
| <img src="images/parts/crank-pin-screw.png" alt="Crank pin screw and nut, M2 x 10" width="240"> | Crank pin screw and nut, M2 x 10 |
| <img src="images/parts/crank-handle.png" alt="Crank handle" width="240"> | Crank handle |
| <img src="images/parts/fan-hub.png" alt="Fan hub" width="240"> | Fan hub |
| <img src="images/parts/fan-blade.png" alt="Fan blade (one for each blade)" width="240"> | Fan blade (one for each blade) |
| <img src="images/parts/fan-bolt.png" alt="Blade bolt and nut, M2 x 16 (one for each blade)" width="240"> | Blade bolt and nut, M2 x 16 (one for each blade) |

You can vary:

- The crank angle, which also sets the fan angle at twice that value.
- The blade pitch, from 10° to 40°, which turns each whole blade; it is the blade angle at three quarters of the radius.
- The number of blades, 3 or 4.

For how to build it from Go, see the [developer notes](../../docs/models/handcrankfan.md).
