# Pedal yarn spindle construction contract

## Purpose

Build a seated, foot-powered quill spindle. A rocking treadle turns a crank and flywheel;
two external 51/17 spur gear pairs turn the spindle nine times per crank revolution.
Import the involute geometry from `models/spurgear`; do not copy it.
The operator drafts from the point, stops, backs off the leader, and winds onto the shaft.
There is no flyer or automatic take-up. Both rotation directions are available for spinning and plying.

This is a dimensioned mechanical concept for a prototype, not a tested manufacturing release.
Use purchased shaft collars/clamping hubs, 608-2RS bearings and 605ZZ bearings; their envelopes are modeled,
but their clamp screws, threads, fasteners, lubrication, material strength, balancing, noise,
and wear are not certified by the geometry tests.

## Inputs and units

All lengths below are millimeters before frame scaling unless marked stock. Angles are radians.
`Params` contains `Scale` (default 1, inclusive range 0.8 to 1.2) and `CrankAngle`
(default pi/3, any finite value). Reject nonfinite values and invalid scale before adding bodies.
Scale frame dimensions, gear module/thickness, axial shaft lengths, positions and linkage lengths.
Stock bearing dimensions, shaft/pin diameters, mating bore diameters and spindle tip radii never scale.
Teeth and angles do not scale. Keep the purchased bearings unchanged at both scale endpoints.

Expose `DefaultParams() Params`, `Params.Validate() error`, `Motion(Params) (State, error)`,
and `Build(context.Context, *decad.Document, Params) (*Assembly, error)`.
`State` exposes `Input`, `Intermediate`, `Output`, `Heel`, `CrankPin`, `Toe` as `r3.Vec`,
`PedalAngle`, `IntermediateAngle`, `SpindleAngle`, `Ratio`, and `CenterDistance` as float64.
Coordinates include scale. Use Y = -125 for the three linkage point coordinates.
`Part` exposes `Name`, `Group` strings and `Body *decad.Body`.
`Assembly` exposes `Parts []Part`, `Sketches []*sketch.Sketch`, and `State State`.
`Sketches` contains every sketch created by this package, including revolve and holed sections.
Imported gears verify their own sketches through their existing Build entry point.
Import bearing envelopes from models/ballbearing. Append each returned bearing Sketch to Assembly.Sketches.
Build returns48 parts, including10 bearing envelopes, and44 recorded sketches.

## Drive and layout

Use X horizontal, Z vertical, and negative Y toward the operator. All rotating shafts are parallel to Y.
The input center is (-90,0,380). The intermediate is (12.3,0,380).
The output is (12.3,0,482.3). Nominal center distance is 102; add 0.3 for tooth clearance.
Use Module 3, pressure angle 20 degrees, 15 flank samples, 12 thickness and no cap chamfer.
Use stock8.2 bores for the input gear and both intermediate gears, and stock5.2 for the output pinion.
The 51-tooth input and 17-tooth intermediate pinion occupy Y = -30 to -42.
The 51-tooth intermediate gear and 17-tooth output pinion occupy Y = -60 to -72.
Build three gear prototypes:51 teeth/bore8.2,17 teeth/bore8.2 and17 teeth/bore5.2.
Copy the51-tooth prototype for its first occurrence; consume the originals for the remaining occurrences.
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
and removable parts: `frame`, `drive`, `gear`, `steel`, `bearing`, `pedal`, `guard`.
Do not fuse touching assembled parts. Gears and hubs are distinct parts on common shafts.

| Parts | Geometry before scale | Group |
| --- | --- | --- |
| base | Box X[-270,150], Y[-240,110], Z[0,18]. | frame |
| left-leg, right-leg | Boxes X[-180,-156] and [80,104], Y[0,18], Z[18,290]. | frame |
| bearing-board | Rectangular plate X[-185,110], Z[290,525], Y[0,18], stock22-diameter input/intermediate bores and stock14 output bore. | frame |
| input-bearing-front, input-bearing-rear | Stock608-2RS envelopes, bore8/OD22/width7, at input. Front spans Y[0,7]; rear spans Y[18*Scale-7,18*Scale], in actual millimeters. | bearing |
| intermediate-bearing-front, intermediate-bearing-rear | Same two stock608-2RS envelopes at intermediate. | bearing |
| output-bearing-front, output-bearing-rear | Stock605ZZ envelopes, bore5/OD14/width5, at output. Front spans Y[0,5]; rear spans Y[18*Scale-5,18*Scale], actual millimeters. | bearing |
| input-shaft | Cylinder stock diameter8, Y[-116,65], at input. | steel |
| intermediate-shaft | Cylinder stock diameter8, Y[-83,26], at intermediate. | steel |
| spindle | Revolved axial section: stock radius2.5 from Y=30 to -245, taper to stock radius0.3 at Y=-270, close to axis. At output center. | steel |
| input-gear, intermediate-pinion, intermediate-gear, output-pinion | Imported spur gears in the positions and phases above. | gear |
| flywheel-rim | Annulus OD310, ID270, Y[35,53], at input. | drive |
| flywheel-hub | Annulus OD34, stock ID8.2, Y[35,53], at input. | drive |
| flywheel-spoke-0 through -3 | Four radial rectangular bars, radial extent[15,137], tangential width12, Y[39,49], at input, angles a+k*pi/2. Overlap rim and hub; separate assembly solids. | drive |
| input-hub, intermediate-pinion-hub, intermediate-gear-hub, output-hub | Annuli OD24, stock ID8.2 for input/intermediate and stock ID5.2 for output. Y[-44,-52] for first pair and [-74,-82] for second pair. | steel |
| crank | Rectangle in XZ extending -9 to34 along crank direction, width18, thickness8 at Y[-113,-105]. Stock holes diameter8.2 at origin and5.2 at radius25. | drive |
| crank-pin | Cylinder stock diameter5 at P, Y[-132,-105]. | steel |
| treadle | Stepped outline in its rotating XZ frame: (-18,-18),(18,-18),(18,-12),(212,-12),(212,12),(18,12),(18,18),(-18,18). Y[-210,-140]. Stock through bores diameter22 at heel and5.2 at toe. | pedal |
| footman | Rectangle along Q-to-P, length320 plus12 at each end, width24, Y[-129,-121]. Stock through bores diameter14 at Q and P. | pedal |
| heel-pin | Through-shaft stock diameter8 at H, Y[-230,-120]. | steel |
| toe-pin | Cylinder stock diameter5 at Q, Y[-215,-118]. | steel |
| heel-bearing-front, heel-bearing-rear | Stock608-2RS envelopes at H. Front spans Y[-210*Scale,-210*Scale+7]; rear spans Y[-140*Scale-7,-140*Scale], actual millimeters. | bearing |
| footman-bearing-crank, footman-bearing-toe | Stock605ZZ envelopes centered at P and Q, spanning Y[-125*Scale-2.5,-125*Scale+2.5], actual millimeters. | bearing |
| heel-bracket-front, heel-bracket-rear | Rectangles X[-250,-210], Z[18,84], Y[-222,-214] and[-136,-128], stock holes diameter8.2 at H. Both attach to the base. | frame |
| heel-collar-front, heel-collar-rear | Commercial clamping-collar envelopes at H, OD16, stock ID8.2, Y[-228,-222] and[-128,-122]. Abut the outside bracket faces. | steel |
| heel-spacer-front, heel-spacer-rear | Inner-ring spacer envelopes at H, stock OD12 and ID8.2, Y[-214,-210.1] and[-139.9,-136]. Leave total0.2*Scale axial end play at the heel bearings. | steel |
| guard-front | Plate X[-175,100], Z[295,520], Y[-94,-90], stock holes diameter8.8 at input/intermediate and5.8 at output. | guard |
| guard-left, guard-right | Boxes X[-179,-175] and [100,104], Y[-94,0], Z[291,524]. | guard |
| guard-bottom, guard-top | Boxes X[-175,100], Y[-94,0], Z[291,295] and [520,524]. | guard |

Annular clamping-hub envelopes identify the required commercial torque-transmitting parts.
Shafts have0.1 radial nominal clearance in gear/hub bores. Bearing IDs equal nominal stock shaft diameters.
Nominal housing bores equal bearing ODs. Fit tolerances and retainers require the fabrication process to be specified.
Use six608-2RS bearings and four605ZZ bearings. The frame holds the two bearings for each rotating shaft.
Two heel bearings let the treadle rock about its frame-fixed pin. Two footman bearings rotate about the moving crank/toe pins.
The toe pin crosses the treadle and footman. The heel through-shaft crosses the treadle and both base brackets.
The two heel collars prevent shaft withdrawal; purchase actual clamping collars and detail their screws before fabrication.
The two spacers limit pedal end play. Match their contact faces to the actual bearing inner rings, avoiding the seals.
The enlarged heel carries the22mm bearing seats; the24mm pedal blade keeps its existing toe/base clearance.
The supported hinge defines the pedal's arc. Do not add straight vertical guides.
The footman passes in front of the input shaft, with5*Scale axial clearance throughout rotation.
The treadle ends11*Scale in front of the footman. Pins bridge these axial gaps.
Provide shaft collars or circlips on the drive shafts during prototype assembly.
The heel collar/spacer envelopes are included; clamp screws and drive-shaft retention hardware remain omitted.

## Return-stroke screening

Specify the nominal-scale flywheel rim as steel, density7800kg/m3; its existing dimensions give about2.56kg.
Specify the treadle and footman as timber at600kg/m3 for the energy calculation; crank and pins use steel density7800.
Treat moving bearing envelopes as solid steel for conservative potential-energy estimates.
Check the actual nominal assembly with a60rpm entry speed and a30rpm minimum retained speed.
Use only the rim's inertia as stored kinetic energy; omit other rotating inertia from the available budget.
Budget the sum of each moving part's full-cycle gravitational potential range,5N residual downward foot force
at150mm from the heel, and0.05Nm combined load/friction torque referred to the crank over a full revolution.
Require available kinetic energy between60 and30rpm to exceed that conservative budget by at least25 percent.
These are explicit screening assumptions, not measured loads or a rated operating envelope.
Keep the crank/footman return; do not add a spring if this screening passes. A hand start and eased foot pressure remain required.
Scaled variants receive geometry checks; the energy acceptance applies only to Scale=1.

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
Check actual linkage bodies against the shaft, supports, guard and each other throughout a full turn.
Check both heel brackets,8mm shaft,608 bearing seats, collar abutments, spacer gaps and through-shaft engagement.
Check all eight stepped treadle vertices and its bores through actual placed geometry.
Run the nominal return-energy screening against actual Decad volumes, centroids and rim inertia.
Reject positive overlap between independently moving parts; allow only the named flywheel spoke/rim/hub joints.
Include the upper crank position that previously put the footman through the input shaft.

Render real Decad bodies with tools/render. Register `pedalspindle/hero` with the guard removed,
`pedalspindle/guarded` with the guard fitted, and `pedalspindle/drive` as a closer gear/linkage view.
The README must label the removed guard. Produce hero.png at1600x900 and index.png at480x270.
Include guarded.png and drive.png at1600x900, plus a renderer integration test for every preset.
Also render front/rear hero and drive pairs as `hero-pair.png` and `drive-pair.png` at2000x1100,
and isolate each actual meshing pair in front and oblique views in `mesh-pairs.png` at2000x1400.
Drive-only views omit all bearing envelopes along with their hidden supports and linkage.
Also omit all heel collars and spacers in drive-only views so no isolated heel hardware remains.
Render the real treadle and complete heel support in oblique and end-on views in heel-support.png at2000x1100.
Use blue for input, gold for both compound intermediate gears and orange for output.
Render the actual crank, footman, treadle and flywheel at0,90,180,270 degrees in `cycle.png` at2000x1000.
Use rigid copies of real bodies placed from Motion, not redrawn geometry. Label the poses and gear planes.
Add a looping linkage.gif from the real bodies with one constant camera per view. Show the fixed frame
beside the exposed linkage. Mark fixed shaft/heel centers A/B separately from moving crank/toe centers C/D.
Use distinct colors for the fixed support, crank/flywheel, footman and treadle. Label the bearings.
The animation prescribes crank angle; do not claim it simulates foot force, inertia or self-starting.
Explain that flywheel momentum carries the return stroke; no ratchet or clutch enforces direction.
Physical testing must settle pedal effort, flywheel inertia, bearing friction, vibration, gear noise,
guard reach protection, fastening, and usable yarn capacity before manufacturing.

## References

- [Ashford quill instructions](https://www.ashford.co.nz/instructions/QS.pdf) describe drafting from the point
  and changing yarn angle for winding. This design adopts that operating method, not its pulley drive.
- [Ashford double-treadle kit](https://www.ashford.co.nz/instructions/DTKTD.pdf) identifies crank,
  connecting rods and supports in a treadle drive. This design uses one calculated crank-rocker linkage.
