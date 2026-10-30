# Pedal yarn spindle construction contract

## Purpose

Build a seated, foot-powered quill spindle. A rocking treadle turns a crank and flywheel;
two external 51/17 spur gear pairs turn the spindle nine times per crank revolution.
Import the involute geometry from `models/spurgear`; do not copy it.
The operator drafts from the point, stops, backs off the leader, and winds onto the shaft.
There is no flyer or automatic take-up. Both rotation directions are available for spinning and plying.

This is a dimensioned mechanical concept for a prototype, not a tested manufacturing release.
Use purchased shaft collars/clamping hubs and plain bushes; their annular envelopes are modeled,
but their clamp screws, threads, fasteners, lubrication, material strength, balancing, noise,
and wear are not certified by the geometry tests.

## Inputs and units

All lengths below are millimeters before uniform scaling. Angles are radians.
`Params` contains `Scale` (default 1, inclusive range 0.8 to 1.2) and `CrankAngle`
(default pi/3, any finite value). Reject nonfinite values and invalid scale before adding bodies.
Scale every length, including bores, clearances, and linkage lengths. Teeth and angles do not scale.

Expose `DefaultParams() Params`, `Params.Validate() error`, `Motion(Params) (State, error)`,
and `Build(context.Context, *decad.Document, Params) (*Assembly, error)`.
`State` exposes `Input`, `Intermediate`, `Output`, `Heel`, `CrankPin`, `Toe` as `r3.Vec`,
`PedalAngle`, `IntermediateAngle`, `SpindleAngle`, `Ratio`, and `CenterDistance` as float64.
Coordinates include scale. Use Y = -101 for the three linkage point coordinates.
`Part` exposes `Name`, `Group` strings and `Body *decad.Body`.
`Assembly` exposes `Parts []Part`, `Sketches []*sketch.Sketch`, and `State State`.
`Sketches` contains every sketch created by this package, including revolve and holed sections.
Imported gears verify their own sketches through their existing Build entry point.

## Drive and layout

Use X horizontal, Z vertical, and negative Y toward the operator. All rotating shafts are parallel to Y.
The input center is (-90,0,380). The intermediate is (12.3,0,380).
The output is (12.3,0,482.3). Nominal center distance is 102; add 0.3 for tooth clearance.
Use Module 3, pressure angle 20 degrees, 15 flank samples, 12 thickness, 10.2 bore, no cap chamfer.
The 51-tooth input and 17-tooth intermediate pinion occupy Y = -30 to -42.
The 51-tooth intermediate gear and 17-tooth output pinion occupy Y = -60 to -72.
Build only two gear prototypes and transform copies into the four positions.
Leave only assembled parts in the document; do not leave unused prototypes at the origin.
Each pair's tooth-zero direction is local +X before rotation.

For crank angle a, intermediate angle b = pi + pi/17 - 3*a.
Reduce a modulo 2*pi before computing motion and phases, including for very large finite inputs.
Returned angles describe a pose rather than accumulated revolutions.
Output angle c = 3*pi + pi/17 - 3*b. This includes the second pair's +Z line of centers.
The spindle rotates in the crank's direction; the intermediate rotates in the opposite direction.
At 40, 60, and 80 complete pedal cycles/minute the spindle turns at 360, 540, and 720 rpm.
These speeds are calculated, not measured. Required twist and feed rate determine the operator's cadence.

## Treadle linkage

Use a four-bar crank-rocker with heel H=(-230,60), crank center O=(-90,380),
crank radius 25, heel-to-toe distance 200, and toe-to-crank-pin rod length 320 in XZ.
P=O+25*(cos(a),sin(a)). Let d=|P-H| and beta=atan2(Pz-Hz,Px-Hx).
Choose the lower intersection: psi=beta-acos((d*d+200*200-320*320)/(2*d*200)).
Q=H+200*(cos(psi),sin(psi)). Reject a nonclosing linkage instead of clamping a bad acos argument.
At every angle, the 200 and 320 lengths must close and the pedal must clear the base.
The crank turns through a full revolution. A hand nudge starts it away from dead center;
the flywheel carries it through the two dead centers. No ratchet is fitted.

## Parts

Construct separate solids, with unique stable lowercase hyphenated names. Groups select renderer colors
and removable parts: `frame`, `drive`, `gear`, `steel`, `bush`, `pedal`, `guard`.
Do not fuse touching assembled parts. Gears and hubs are distinct parts on common shafts.

| Parts | Geometry before scale | Group |
| --- | --- | --- |
| base | Box X[-270,150], Y[-240,110], Z[0,18]. | frame |
| left-leg, right-leg | Boxes X[-180,-156] and [80,104], Y[0,18], Z[18,290]. | frame |
| bearing-board | Rectangular plate X[-185,110], Z[290,525], Y[0,18], three 16-diameter through holes at the shaft centers. | frame |
| input-bush, intermediate-bush, output-bush | Annuli OD16, ID10.2, Y[-1,19], centered at the corresponding shaft. | bush |
| input-shaft | Cylinder diameter10, Y[-116,65], at input. | steel |
| intermediate-shaft | Cylinder diameter10, Y[-83,26], at intermediate. | steel |
| spindle | Revolved axial section: radius5 from Y=30 to -245, taper to radius0.6 at Y=-270, close to axis. At output center. | steel |
| input-gear, intermediate-pinion, intermediate-gear, output-pinion | Imported spur gears in the positions and phases above. | gear |
| flywheel-rim | Annulus OD310, ID270, Y[35,53], at input. | drive |
| flywheel-hub | Annulus OD34, ID10.2, Y[35,53], at input. | drive |
| flywheel-spoke-0 through -3 | Four radial rectangular bars, radial extent[15,137], tangential width12, Y[39,49], at input, angles a+k*pi/2. Overlap rim and hub; separate assembly solids. | drive |
| input-hub, intermediate-pinion-hub, intermediate-gear-hub, output-hub | Annuli OD24, ID10.2, Y[-44,-52] for first pair and [-74,-82] for second pair. | steel |
| crank | Rectangle in XZ extending -9 to34 along crank direction, width18, thickness8 at Y[-113,-105]. Two holes radius5.1 at origin and4.2 at radius25. | drive |
| crank-pin | Cylinder diameter8 at P, Y[-116,-94]. | steel |
| treadle | Rectangle in its rotating XZ frame from heel coordinate (-12,-8) to (212,8), Y[-210,-120]. Through holes diameter8.4 at heel and toe. | pedal |
| footman | Rectangle along Q-to-P, length320 plus10 at each end, width18, Y[-105,-97]. Through holes diameter8.4 at Q and P. | pedal |
| heel-pin | Cylinder diameter8 at H, Y[-215,-115]. | steel |
| toe-pin | Cylinder diameter8 at Q, Y[-215,-94]. | steel |
| heel-bracket | Rectangle X[-244,-216], Z[18,76], Y[-120,-112], hole diameter8.4 at H. | frame |
| guard-front | Plate X[-175,100], Z[295,520], Y[-94,-90], holes diameter10.8 at all three shafts. | guard |
| guard-left, guard-right | Boxes X[-179,-175] and [100,104], Y[-94,0], Z[291,524]. | guard |
| guard-bottom, guard-top | Boxes X[-175,100], Y[-94,0], Z[291,295] and [520,524]. | guard |

Annular clamping-hub envelopes identify the required commercial torque-transmitting parts.
Shafts have 0.1 radial clearance in gears, bushes, and hubs; gear clamp hubs must bridge that clearance.
The toe pin crosses the treadle and rear-offset footman; the heel pin crosses the treadle and heel bracket.
Provide shaft collars or circlips during prototype assembly. The image omits their detailed retention hardware.

## Sketch and body rules

Build primitive parts locally and transform their completed bodies into world coordinates.
Use centered radius-constrained circles and diameter-constrained holes.
Constrain rectangles by their built-in horizontal/vertical constraints, one anchored corner and two dimensions.
Use shared endpoints, horizontal/vertical relations, and dimensions for the spindle axial polygon.
Never fix every computed point. Solve and require FullyConstrained, DOF zero, no redundant constraints,
and no conflicts before using every sketch. Select a unique valid profile with the expected hole count.
Check every returned part is solid. Return errors naming the part and failed operation.

## Required checks and images

First build the default assembly and tessellate its real parts before expanding coverage.
Then check scale endpoints and another crank quadrant; verify every owned sketch in each build.
Check unique names, gear thickness/placement and bores through the consuming assembly,
solid topology, VerifyAll tessellation, flywheel radius, spindle tip height/length, and scaled bounds.
Check the four-bar closure, full rotation, base clearance and constant 9:1 ratio over a full crank turn.
Check actual meshing gear sections for overlap at several tooth phases and both stages.

Render real Decad bodies with tools/render. Register `pedalspindle/hero` with the guard removed,
`pedalspindle/guarded` with the guard fitted, and `pedalspindle/drive` as a closer gear/linkage view.
The README must label the removed guard. Produce hero.png at1600x900 and index.png at480x270.
Include guarded.png and drive.png at1600x900, plus a renderer integration test for every preset.
Also render front/rear hero and drive pairs as `hero-pair.png` and `drive-pair.png` at2000x1100,
and isolate each actual meshing pair in front and oblique views in `mesh-pairs.png` at2000x1400.
Use blue for input, gold for both compound intermediate gears and orange for output.
Render the actual crank, footman, treadle and flywheel at0,90,180,270 degrees in `cycle.png` at2000x1000.
Use rigid copies of real bodies placed from Motion, not redrawn geometry. Label the poses and gear planes.
Explain that flywheel momentum carries the return stroke; no ratchet or clutch enforces direction.
Physical testing must settle pedal effort, flywheel inertia, bearing friction, vibration, gear noise,
guard reach protection, fastening, and usable yarn capacity before manufacturing.

## References

- [Ashford quill instructions](https://www.ashford.co.nz/instructions/QS.pdf) describe drafting from the point
  and changing yarn angle for winding. This design adopts that operating method, not its pulley drive.
- [Ashford double-treadle kit](https://www.ashford.co.nz/instructions/DTKTD.pdf) identifies crank,
  connecting rods and supports in a treadle drive. This design uses one calculated crank-rocker linkage.
