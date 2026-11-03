# Treadle spinning wheel

![Open Saxony spinning wheel](images/hero.png)

This is a work-in-progress model. Joint cuts and verification remain unfinished.
The images show all 195 parts of the default build, rendered from the real Decad body with Decad `ee1ee0377743`.
The [preview status](images/preview-status.json) records the part counts and that no part is omitted.
Every part pair of the default build has a conclusive non-overlap result.
Fits, shaft retention and moving clearances still need verification.

This wheel follows the flyer-and-bobbin arrangement in
[George Weil's labelled spinning wheel](https://www.georgeweil.com/blog/how-does-a-spinning-wheel-work/).
A foot treadle turns the large wheel through a crank and footman. A single drive band turns
the hollow flyer shaft. The bobbin rotates independently, and a Scotch brake slows it enough
for the flyer to wind yarn onto it.

The wheel has a 560 mm outside diameter and a 690 mm orifice height at default scale.
Four whorl grooves provide 7:1, 9:1, 12:1 and 17:1 flyer-to-wheel speed ratios.
The six-spoke wheel has a nominal 580 mm hub height. The hinged head puts the orifice at 690 mm at default tension.
The dimensions of this gallery frame and head define a new design rather than a copy of
the manufacturer's production drawings.

## Front and back

![Operator and rear views](images/front-back.png)

Two splayed uprights carry the wheel axle. The front wheel leg and single head leg carry the treadle shaft.
The head's maiden posts carry the flyer. The open frame carries these parts independently
of any removable cover. The vertical jack tilts the mother-of-all to adjust drive-band tension.
Moving its base along the flyer axis selects a whorl groove while keeping both pulley axes parallel.

![Top and side views of the actual arrangement](images/layout.png)

The footboard extends fore and aft toward the operator. Its hinge shaft crosses the frame,
perpendicular to the wheel axle. A diagonal actuating rail reaches the footman behind the wheel.
An upper gimbal has two rolling axes; a lower compliant connector joins the rocking rail to the footman.
The [reference inventory](../../docs/spinning-wheel-reference-audit.md) records all corrected arrangements.

## Yarn path and take-up

![Flyer, independent bobbin and hollow orifice](images/flyer.png)

The leader enters the 10 mm axial orifice, exits a 6 mm radial port, passes through open guide hooks,
and reaches the bobbin. The flyer guides rotate around the bobbin. The bobbin's two 608 bearings
let it turn at a different speed from the flyer shaft. An orange marker on the teal yarn pack
makes that difference visible.

![Scotch brake and spring](images/brake.png)

The brake cord wraps the bobbin's rear flange.
Both tails run forward below the flyer to two extension springs
and an accessible adjusting screw. Turning the screw extends both springs while preserving cord length.

The drive whorl, wheel hub, crank and treadle rail use removable keys in otherwise round shafts.
The whorl is one piece, between the front maiden and flyer crossbar, with all four grooves and a keyed 16.2 mm bore.
Bearing journals stay circular. The 10 mm yarn orifice requires a 16 mm front journal and
a plain nylon sleeve; the smaller rear journal uses a 608 bearing.

![One-piece whorl and its keyed through-bore](images/whorl.png)

## Motion

![Wheel turn animation](images/motion.gif)

The animation shows one wheel turn in 96 frames. It prescribes 195 mm of yarn feed per wheel turn,
so the flyer makes 7 turns while the bobbin makes 6.
The still images show one solved pose. The frame stays fixed while the treadle rocks and the footman's
bearing centres follow the spatial linkage. The model default prescribes 100 mm of yarn feed per wheel turn.

The wheel can turn in either direction after a hand start. Flywheel inertia carries it
through the crank's dead centres and raises the pedal. Side slides and a pedal return spring
are unnecessary for this rocking treadle. The model prescribes motion; it does not
calculate foot force, bearing load, inertia, belt traction or brake friction.

## Parts and adjustment

The assembly reuses the [bearing envelopes](../ballbearing/) for seven 608-2RS bearings
and three 605ZZ bearings. A custom 16.2 x 26 x 20 mm sleeve carries the front flyer journal.
Other hardware includes 8 mm shafts, 5 mm linkage pins, M6 post/clamp fasteners,
an M8 drive-tension screw and an M4 brake-adjusting screw. Each shaft collar has a radial M2 grub screw.
Bolted end plates retain the gimbal and footman bearings. Screw threads are represented by mounting envelopes.

The rear maiden and its glued foot remove together for bobbin changes. Slacken the drive band and release the brake cord.
Remove the outer rear-flyer collar, unbolt the maiden foot, and slide the post-and-foot unit rearward off the shaft.
Remove the remaining rear-support collar and bobbin retaining collar, then slide the bobbin off the rear shaft.
Leave the modeled 0.2 mm shoulder gaps for bobbin rotation.
Physical brake setting, belt preload, balance, wood joints and shaft retention need checking
on a prototype before using the wheel to spin yarn.

See [construction and verification](../../docs/models/spinningwheel.md) for parameters,
build commands and the limits of the geometry checks.
