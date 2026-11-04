# Stable manufactured-part identifiers

This file owns names shared by construction, joinery, mounts, tests and rendering.
It specifies identifiers only. Geometry remains in instructions.md and mounting.md.
Numeric suffixes are one-based. The parent emitter may correct identifiers against this API without altering geometry.

## Frame and adjustment

- head-leg, front-wheel-leg, rear-wheel-leg.
- front-bench-rail, rear-bench-rail, front-head-wedge, rear-head-wedge.
- front-wheel-bearing-cap, rear-wheel-bearing-cap, wheel-bearing-1/2, heel-bearing-1/2.
- wheel-axle-front-collar, wheel-axle-rear-collar.
- wheel-axle-front-thrust-spacer, wheel-axle-rear-thrust-spacer.
- mother-base, mother-upper-bar, mother-jack-arm, mother-jack-square-insert, mother-jack-screw, mother-jack-knob.
- mother-alignment-clamp-1/2, mother-alignment-insert-1/2.
- maiden-foot-1/2; each has -bolt-1/2 and -insert-1/2 suffixes.
- mother-hinge-<1|2>-fixed-bracket-<1|2>, -fixed-mount-<1|2>, -moving-leaf, -moving-mount, -pin, -nut.

## Drive

- wheel-rim, wheel-hub, wheel-spoke-1 through6, wheel-axle, wheel-crank, wheel-crank-pin.
- wheel-hub-key, wheel-crank-key, wheel-hub-front-spacer, wheel-hub-rear-spacer, wheel-crank-pin-spacer.
- wheel-crank-front-collar, wheel-crank-rear-collar, wheel-crank-pin-collar-1/2.
- gimbal-bridge, gimbal-crank-605, gimbal-crank-retainer-1/2.
- gimbal-retainer-bolt-1/2, gimbal-retainer-nut-1/2.
- gimbal-cross-pin, gimbal-cross-pin-collar-1/2, gimbal-cross-spacer-1/2.
- footman, footman-head-boss, footman-cross-605-1/2, footman-cross-spacer, footman-cross-retainer-1/2.
- footman-retainer-bolt, footman-retainer-nut.
- treadle, treadle-heel-rail, treadle-bar-left/right, treadle-rear-crossarm, treadle-connector-block.
- treadle-board-bolt-1/2/3/4-pin/nut, treadle-block-bolt-pin/nut.
- heel-pin, heel-key, heel-collar-1/2, heel-spacer-1/2, treadle-flex-connector.
- treadle-lower-clamp-pin/nut, footman-upper-clamp-pin/nut.
- drive-band-wheel-wrap, drive-band-flyer-wrap, drive-band-span-1/2.
  These four bodies have tangent, matching cap seams; they are construction spans of the single drive band.
  Keep their shared physical identity in the mount description. They must not overlap material.

## Head

- flyer-shaft, flyer-drive-whorl, flyer-crossbar, flyer-arm-1/2, flyer-whorl-key, flyer-crossbar-key.
- front-maiden, rear-maiden, front-maiden-bush, rear-flyer-bearing.
- rear-flyer-front-collar, rear-flyer-rear-collar.
- rear-flyer-front-thrust-spacer, rear-flyer-rear-thrust-spacer.
- front-sleeve-retainer, whorl-front-collar, whorl-rear-collar, crossbar-rear-collar.
- bobbin, bobbin-yarn-pack, bobbin-witness-bar.
- bobbin-front-bearing, bobbin-rear-bearing, bobbin-inner-race-sleeve, bobbin-front-thrust-sleeve.
- bobbin-rear-removable-collar.

## Guides and brake

- yarn-hook-1 through6 occupy the negative arm in increasing Y order;7through12 occupy the positive arm.
  The positive active guide atY=-160*Scale is yarn-hook-9. Both arrays use[-190,-172,-160,-142,-124,-106].
- yarn-front-guide, yarn-leader.
  Formed hooks and front guide use Group guide; the process leader uses Group yarn.
- brake-band, brake-spring-left, brake-spring-right, brake-routing-eye-left, brake-routing-eye-right.
- brake-rear-crossbar, brake-front-upright, brake-control-rail.
- brake-insert, brake-insert-retaining-plate, brake-insert-mount-1/2.
- brake-left-anchor, brake-takeup-screw, brake-takeup-knob, brake-front-pin-left/right.

## Derived fasteners and support conventions

- Each shaft collar has exactly one <collar-id>-setscrew.
- Two numbered frame/shaft/bearing parts use front/negative as1 and rear/positive as2.
- All output names are unique. Every final part has exactly one Mount record.
- Do not return old separate hook stems, spring links, clamp heads, hinge knuckles or gimbal cheeks.
