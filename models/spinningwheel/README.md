# Treadle spinning wheel

![Open Saxony spinning wheel](images/hero.png)

The images show all 206 parts of the default build.

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

![Rear of the wheel](images/rear.png)

The rear view shows the drive band running from the wheel rim to the flyer whorl, the bobbin and the mother-of-all.

Two splayed uprights carry the wheel axle. The front wheel leg and single head leg carry the treadle shaft.
The head's maiden posts carry the flyer. The open frame carries these parts independently
of any removable cover. The vertical jack tilts the mother-of-all to adjust drive-band tension.
Moving its base along the flyer axis selects a whorl groove while keeping both pulley axes parallel.

![Top and side views of the actual arrangement](images/layout.png)

The footboard extends fore and aft toward the operator. Its hinge shaft crosses the frame,
perpendicular to the wheel axle. Two bars carry the footboard, and a crossarm behind the wheel reaches the footman.
An upper gimbal has two rolling axes; a lower compliant connector joins the rocking rail to the footman.

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

The animation shows one wheel turn. The left panel views the wheel from the side opposite
the still images (turned 180 degrees about the vertical axis) to show the treadle, footman, crank and connector.
It prescribes 195 mm of yarn feed per wheel turn,
so the flyer makes 7 turns while the bobbin makes 6.
The still images show one pose. The model default is 100 mm of yarn feed per wheel turn.

The wheel can turn in either direction after a hand start. Flywheel inertia carries it
through the crank's dead centres and raises the pedal. Side slides and a pedal return spring
are unnecessary for this rocking treadle. The animation shows set motion; it does not
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

## Exploded

![The wheel pulled apart by sub-assembly](images/exploded.png)

Each sub-assembly moves as one block from the assembled position. The frame stays put. The wheel with its axle and
crank moves to one side, the treadle drops, and the footman and gimbal move to the other. The head with its mother-of-all lifts
above the frame, the flyer and bobbin lift above the head, and the brake and drive band move out to the sides. Parts inside a
block keep their assembled places. The view is built from the 206 bodies of the default build and moved by the
renderer; it is not a derived Decad explode, because the wheel has no joint graph for one.

## Parts table

The 206 bodies have 109 distinct shapes. Identical bodies share a row, and the count says how many the wheel uses.
Each row gives the name of its first body. The bodies sit in the model's containers:
`Frame`, `Wheel`, `Treadle`, `Footman`, `Gimbal`, `Flyer`, `Bobbin`, `Fasteners`, `Collars`, `Bearings` and the
other kind containers. The table below groups the rows by sub-assembly instead, to match the exploded view.

### Frame

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/head-leg.png" alt="Head leg" width="200"> | Head leg (`head-leg`) | 1 |
| <img src="images/parts/front-wheel-leg.png" alt="Front wheel leg" width="200"> | Front wheel leg (`front-wheel-leg`) | 1 |
| <img src="images/parts/rear-wheel-leg.png" alt="Rear wheel leg" width="200"> | Rear wheel leg (`rear-wheel-leg`) | 1 |
| <img src="images/parts/front-bench-rail.png" alt="Front bench rail" width="200"> | Front bench rail (`front-bench-rail`) | 2 |
| <img src="images/parts/front-head-wedge.png" alt="Front head wedge" width="200"> | Front head wedge (`front-head-wedge`) | 1 |
| <img src="images/parts/rear-head-wedge.png" alt="Rear head wedge" width="200"> | Rear head wedge (`rear-head-wedge`) | 1 |
| <img src="images/parts/front-wheel-bearing-cap.png" alt="Front wheel bearing cap" width="200"> | Front wheel bearing cap (`front-wheel-bearing-cap`) | 2 |
| <img src="images/parts/wheel-bearing.png" alt="608-2RS bearing" width="200"> | 608-2RS bearing (`wheel-bearing-1`) | 7 |
| <img src="images/parts/wheel-bearing-ring.png" alt="Bearing retaining ring" width="200"> | Bearing retaining ring (`wheel-bearing-1-ring`) | 5 |

### Wheel, axle and crank

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/wheel-axle-front-collar.png" alt="Shaft collar (8 mm)" width="200"> | Shaft collar (8 mm) (`wheel-axle-front-collar`) | 4 |
| <img src="images/parts/wheel-axle-front-thrust-spacer.png" alt="Wheel axle front thrust spacer" width="200"> | Wheel axle front thrust spacer (`wheel-axle-front-thrust-spacer`) | 2 |
| <img src="images/parts/wheel-rim.png" alt="Wheel rim" width="200"> | Wheel rim (`wheel-rim`) | 1 |
| <img src="images/parts/wheel-hub.png" alt="Wheel hub" width="200"> | Wheel hub (`wheel-hub`) | 1 |
| <img src="images/parts/wheel-spoke.png" alt="Wheel spoke" width="200"> | Wheel spoke (`wheel-spoke-1`) | 6 |
| <img src="images/parts/wheel-axle.png" alt="Wheel axle" width="200"> | Wheel axle (`wheel-axle`) | 1 |
| <img src="images/parts/wheel-hub-key.png" alt="Wheel hub key" width="200"> | Wheel hub key (`wheel-hub-key`) | 1 |
| <img src="images/parts/wheel-crank-key.png" alt="Wheel crank key" width="200"> | Wheel crank key (`wheel-crank-key`) | 1 |
| <img src="images/parts/wheel-crank.png" alt="Wheel crank" width="200"> | Wheel crank (`wheel-crank`) | 1 |
| <img src="images/parts/wheel-hub-front-spacer.png" alt="Wheel hub front spacer" width="200"> | Wheel hub front spacer (`wheel-hub-front-spacer`) | 2 |
| <img src="images/parts/wheel-crank-pin.png" alt="Wheel crank pin" width="200"> | Wheel crank pin (`wheel-crank-pin`) | 1 |
| <img src="images/parts/wheel-crank-pin-collar.png" alt="Wheel crank pin collar" width="200"> | Wheel crank pin collar (`wheel-crank-pin-collar-1`) | 2 |
| <img src="images/parts/wheel-crank-pin-spacer.png" alt="Wheel crank pin spacer" width="200"> | Wheel crank pin spacer (`wheel-crank-pin-spacer`) | 1 |
| <img src="images/parts/wheel-axle-front-collar-setscrew.png" alt="Collar grub screw (axle, crank, rear flyer, bobbin)" width="200"> | Collar grub screw (axle, crank, rear flyer, bobbin) (`wheel-axle-front-collar-setscrew`) | 7 |
| <img src="images/parts/wheel-crank-pin-collar-setscrew.png" alt="Collar grub screw (crank pin, gimbal pin)" width="200"> | Collar grub screw (crank pin, gimbal pin) (`wheel-crank-pin-collar-1-setscrew`) | 4 |

### Treadle

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/treadle.png" alt="Treadle" width="200"> | Treadle (`treadle`) | 1 |
| <img src="images/parts/treadle-heel-rail.png" alt="Treadle heel rail" width="200"> | Treadle heel rail (`treadle-heel-rail`) | 1 |
| <img src="images/parts/treadle-bar-left.png" alt="Treadle bar left" width="200"> | Treadle bar left (`treadle-bar-left`) | 2 |
| <img src="images/parts/treadle-rear-crossarm.png" alt="Treadle rear crossarm" width="200"> | Treadle rear crossarm (`treadle-rear-crossarm`) | 1 |
| <img src="images/parts/treadle-connector-block.png" alt="Treadle connector block" width="200"> | Treadle connector block (`treadle-connector-block`) | 1 |
| <img src="images/parts/heel-pin.png" alt="Heel pin" width="200"> | Heel pin (`heel-pin`) | 1 |
| <img src="images/parts/heel-key.png" alt="Heel key" width="200"> | Heel key (`heel-key`) | 1 |
| <img src="images/parts/heel-collar.png" alt="Heel collar" width="200"> | Heel collar (`heel-collar-1`) | 2 |
| <img src="images/parts/heel-spacer.png" alt="Heel spacer" width="200"> | Heel spacer (`heel-spacer-1`) | 2 |
| <img src="images/parts/treadle-lower-clamp-pin.png" alt="Treadle lower clamp pin" width="200"> | Treadle lower clamp pin (`treadle-lower-clamp-pin`) | 1 |
| <img src="images/parts/treadle-board-bolt-pin.png" alt="Footboard bolt" width="200"> | Footboard bolt (`treadle-board-bolt-1-pin`) | 4 |
| <img src="images/parts/treadle-block-bolt-pin.png" alt="Treadle block bolt pin" width="200"> | Treadle block bolt pin (`treadle-block-bolt-pin`) | 1 |
| <img src="images/parts/treadle-flex-connector.png" alt="Treadle flex connector" width="200"> | Treadle flex connector (`treadle-flex-connector`) | 1 |
| <img src="images/parts/heel-collar-setscrew.png" alt="Collar grub screw (heel pin)" width="200"> | Collar grub screw (heel pin) (`heel-collar-1-setscrew`) | 2 |

### Footman and gimbal

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/gimbal-bridge.png" alt="Gimbal bridge" width="200"> | Gimbal bridge (`gimbal-bridge`) | 1 |
| <img src="images/parts/gimbal-crank.png" alt="605ZZ bearing" width="200"> | 605ZZ bearing (`gimbal-crank-605`) | 3 |
| <img src="images/parts/gimbal-crank-retainer.png" alt="Gimbal crank retainer" width="200"> | Gimbal crank retainer (`gimbal-crank-retainer-1`) | 2 |
| <img src="images/parts/gimbal-retainer-bolt.png" alt="Gimbal retainer bolt" width="200"> | Gimbal retainer bolt (`gimbal-retainer-bolt-1`) | 2 |
| <img src="images/parts/gimbal-retainer-nut.png" alt="M2 nut" width="200"> | M2 nut (`gimbal-retainer-nut-1`) | 3 |
| <img src="images/parts/gimbal-cross-pin.png" alt="Gimbal cross pin" width="200"> | Gimbal cross pin (`gimbal-cross-pin`) | 1 |
| <img src="images/parts/gimbal-cross-pin-collar.png" alt="Gimbal cross pin collar" width="200"> | Gimbal cross pin collar (`gimbal-cross-pin-collar-1`) | 2 |
| <img src="images/parts/gimbal-cross-spacer.png" alt="Gimbal cross spacer" width="200"> | Gimbal cross spacer (`gimbal-cross-spacer-1`) | 2 |
| <img src="images/parts/footman.png" alt="Footman" width="200"> | Footman (`footman`) | 1 |
| <img src="images/parts/footman-head-boss.png" alt="Footman head boss" width="200"> | Footman head boss (`footman-head-boss`) | 1 |
| <img src="images/parts/footman-cross-spacer.png" alt="Footman cross spacer" width="200"> | Footman cross spacer (`footman-cross-spacer`) | 1 |
| <img src="images/parts/footman-cross-retainer.png" alt="Footman cross retainer" width="200"> | Footman cross retainer (`footman-cross-retainer-1`) | 2 |
| <img src="images/parts/footman-retainer-bolt.png" alt="Footman retainer bolt" width="200"> | Footman retainer bolt (`footman-retainer-bolt`) | 1 |
| <img src="images/parts/footman-upper-clamp-pin.png" alt="Footman upper clamp pin" width="200"> | Footman upper clamp pin (`footman-upper-clamp-pin`) | 1 |
| <img src="images/parts/footman-upper-clamp-nut.png" alt="Clamp and bolt nut" width="200"> | Clamp and bolt nut (`footman-upper-clamp-nut`) | 7 |

### Head

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/mother-base.png" alt="Mother base" width="200"> | Mother base (`mother-base`) | 1 |
| <img src="images/parts/mother-alignment-clamp.png" alt="Mother alignment clamp" width="200"> | Mother alignment clamp (`mother-alignment-clamp-1`) | 2 |
| <img src="images/parts/mother-alignment-insert.png" alt="Alignment and foot insert" width="200"> | Alignment and foot insert (`mother-alignment-insert-1`) | 6 |
| <img src="images/parts/mother-hinge-fixed-bracket.png" alt="Mother hinge fixed bracket" width="200"> | Mother hinge fixed bracket (`mother-hinge-1-fixed-bracket-1`) | 4 |
| <img src="images/parts/mother-hinge-fixed-mount.png" alt="Mother hinge fixed mount" width="200"> | Mother hinge fixed mount (`mother-hinge-1-fixed-mount-1`) | 4 |
| <img src="images/parts/mother-hinge-pin.png" alt="Mother hinge pin" width="200"> | Mother hinge pin (`mother-hinge-1-pin`) | 2 |
| <img src="images/parts/mother-hinge-nut.png" alt="Mother hinge nut" width="200"> | Mother hinge nut (`mother-hinge-1-nut`) | 2 |
| <img src="images/parts/mother-hinge-moving-leaf.png" alt="Mother hinge moving leaf" width="200"> | Mother hinge moving leaf (`mother-hinge-1-moving-leaf`) | 2 |
| <img src="images/parts/mother-hinge-moving-mount.png" alt="Mother hinge moving mount" width="200"> | Mother hinge moving mount (`mother-hinge-1-moving-mount`) | 2 |
| <img src="images/parts/mother-upper-bar.png" alt="Mother upper bar" width="200"> | Mother upper bar (`mother-upper-bar`) | 1 |
| <img src="images/parts/mother-jack-arm.png" alt="Mother jack arm" width="200"> | Mother jack arm (`mother-jack-arm`) | 1 |
| <img src="images/parts/mother-jack-square-insert.png" alt="Mother jack square insert" width="200"> | Mother jack square insert (`mother-jack-square-insert`) | 1 |
| <img src="images/parts/mother-jack-screw.png" alt="Mother jack screw" width="200"> | Mother jack screw (`mother-jack-screw`) | 1 |
| <img src="images/parts/mother-jack-knob.png" alt="Mother jack knob" width="200"> | Mother jack knob (`mother-jack-knob`) | 1 |
| <img src="images/parts/maiden-foot.png" alt="Front maiden foot" width="200"> | Front maiden foot (`maiden-foot-1`) | 1 |
| <img src="images/parts/maiden-foot-bolt.png" alt="Maiden foot bolt" width="200"> | Maiden foot bolt (`maiden-foot-1-bolt-1`) | 4 |
| <img src="images/parts/maiden-foot-b.png" alt="Rear maiden foot" width="200"> | Rear maiden foot (`maiden-foot-2`) | 1 |
| <img src="images/parts/front-maiden.png" alt="Front maiden" width="200"> | Front maiden (`front-maiden`) | 1 |
| <img src="images/parts/rear-maiden.png" alt="Rear maiden" width="200"> | Rear maiden (`rear-maiden`) | 1 |
| <img src="images/parts/front-maiden-bush.png" alt="Front maiden bush" width="200"> | Front maiden bush (`front-maiden-bush`) | 1 |

### Flyer and bobbin

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/flyer-shaft.png" alt="Flyer shaft" width="200"> | Flyer shaft (`flyer-shaft`) | 1 |
| <img src="images/parts/flyer-whorl-key.png" alt="Flyer whorl key" width="200"> | Flyer whorl key (`flyer-whorl-key`) | 1 |
| <img src="images/parts/flyer-crossbar-key.png" alt="Flyer crossbar key" width="200"> | Flyer crossbar key (`flyer-crossbar-key`) | 1 |
| <img src="images/parts/flyer-drive-whorl.png" alt="Flyer drive whorl" width="200"> | Flyer drive whorl (`flyer-drive-whorl`) | 1 |
| <img src="images/parts/flyer-crossbar.png" alt="Flyer crossbar" width="200"> | Flyer crossbar (`flyer-crossbar`) | 1 |
| <img src="images/parts/flyer-arm.png" alt="Flyer arm" width="200"> | Flyer arm (`flyer-arm-1`) | 2 |
| <img src="images/parts/rear-flyer-front-collar.png" alt="Flyer shaft collar (long)" width="200"> | Flyer shaft collar (long) (`rear-flyer-front-collar`) | 3 |
| <img src="images/parts/rear-flyer-front-thrust-spacer.png" alt="Rear flyer front thrust spacer" width="200"> | Rear flyer front thrust spacer (`rear-flyer-front-thrust-spacer`) | 1 |
| <img src="images/parts/rear-flyer-rear-thrust-spacer.png" alt="Rear flyer rear thrust spacer" width="200"> | Rear flyer rear thrust spacer (`rear-flyer-rear-thrust-spacer`) | 1 |
| <img src="images/parts/front-sleeve-retainer.png" alt="Shaft collar (flyer shaft)" width="200"> | Shaft collar (flyer shaft) (`front-sleeve-retainer`) | 4 |
| <img src="images/parts/bobbin.png" alt="Bobbin" width="200"> | Bobbin (`bobbin`) | 1 |
| <img src="images/parts/bobbin-inner-race-sleeve.png" alt="Bobbin inner race sleeve" width="200"> | Bobbin inner race sleeve (`bobbin-inner-race-sleeve`) | 1 |
| <img src="images/parts/bobbin-front-thrust-sleeve.png" alt="Bobbin front thrust sleeve" width="200"> | Bobbin front thrust sleeve (`bobbin-front-thrust-sleeve`) | 1 |
| <img src="images/parts/bobbin-yarn-pack.png" alt="Bobbin yarn pack" width="200"> | Bobbin yarn pack (`bobbin-yarn-pack`) | 1 |
| <img src="images/parts/bobbin-witness-bar.png" alt="Bobbin witness bar" width="200"> | Bobbin witness bar (`bobbin-witness-bar`) | 1 |
| <img src="images/parts/yarn-hook.png" alt="Yarn hook" width="200"> | Yarn hook (`yarn-hook-1`) | 11 |
| <img src="images/parts/yarn-hook-b.png" alt="Yarn hook (wide)" width="200"> | Yarn hook (wide) (`yarn-hook-9`) | 1 |
| <img src="images/parts/yarn-front-guide.png" alt="Yarn front guide" width="200"> | Yarn front guide (`yarn-front-guide`) | 1 |
| <img src="images/parts/yarn-leader.png" alt="Yarn leader" width="200"> | Yarn leader (`yarn-leader`) | 1 |
| <img src="images/parts/front-sleeve-retainer-setscrew.png" alt="Collar grub screw (whorl, crossbar, sleeve)" width="200"> | Collar grub screw (whorl, crossbar, sleeve) (`front-sleeve-retainer-setscrew`) | 4 |

### Scotch brake

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/brake-rear-crossbar.png" alt="Brake rear crossbar" width="200"> | Brake rear crossbar (`brake-rear-crossbar`) | 1 |
| <img src="images/parts/brake-front-upright.png" alt="Brake front upright" width="200"> | Brake front upright (`brake-front-upright`) | 1 |
| <img src="images/parts/brake-control-rail.png" alt="Brake control rail" width="200"> | Brake control rail (`brake-control-rail`) | 1 |
| <img src="images/parts/brake-band.png" alt="Brake band" width="200"> | Brake band (`brake-band`) | 1 |
| <img src="images/parts/brake-spring-left.png" alt="Brake spring left" width="200"> | Brake spring left (`brake-spring-left`) | 2 |
| <img src="images/parts/brake-routing-eye-left.png" alt="Brake routing eye left" width="200"> | Brake routing eye left (`brake-routing-eye-left`) | 2 |
| <img src="images/parts/brake-front-pin-left.png" alt="Brake front pin left" width="200"> | Brake front pin left (`brake-front-pin-left`) | 2 |
| <img src="images/parts/brake-left-anchor.png" alt="Brake left anchor" width="200"> | Brake left anchor (`brake-left-anchor`) | 1 |
| <img src="images/parts/brake-takeup-screw.png" alt="Brake takeup screw" width="200"> | Brake takeup screw (`brake-takeup-screw`) | 1 |
| <img src="images/parts/brake-takeup-knob.png" alt="Brake takeup knob" width="200"> | Brake takeup knob (`brake-takeup-knob`) | 1 |
| <img src="images/parts/brake-insert.png" alt="Brake insert" width="200"> | Brake insert (`brake-insert`) | 1 |
| <img src="images/parts/brake-insert-retaining-plate.png" alt="Brake insert retaining plate" width="200"> | Brake insert retaining plate (`brake-insert-retaining-plate`) | 1 |
| <img src="images/parts/brake-insert-mount.png" alt="Brake insert mount" width="200"> | Brake insert mount (`brake-insert-mount-1`) | 2 |

### Drive band

| Image | Part | Count |
| --- | --- | ---: |
| <img src="images/parts/drive-band-wheel-wrap.png" alt="Drive band wheel wrap" width="200"> | Drive band wheel wrap (`drive-band-wheel-wrap`) | 1 |
| <img src="images/parts/drive-band-flyer-wrap.png" alt="Drive band flyer wrap" width="200"> | Drive band flyer wrap (`drive-band-flyer-wrap`) | 1 |
| <img src="images/parts/drive-band-span.png" alt="Drive band span" width="200"> | Drive band span (`drive-band-span-1`) | 2 |

See [construction and verification](../../docs/models/spinningwheel.md) for parameters,
build commands and the limits of the geometry checks.
