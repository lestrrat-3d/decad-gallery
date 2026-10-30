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
Build returns61 parts, including10 bearing envelopes, and72 recorded sketches.

## Drive and layout

Use X horizontal, Z vertical, and negative Y toward the operator. All rotating shafts are parallel to Y.
The input center is (-90,0,380). The intermediate is (12.3,0,380).
The output is (12.3,0,482.3). Nominal center distance is 102; add 0.3 for tooth clearance.
Use Module 3, pressure angle 20 degrees, 15 flank samples, 12 thickness and no cap chamfer.
Use stock D-shaped bores for positive shaft drive: radius4.1 with flat at local X=3.1 for input/intermediate,
and radius2.6 with flat at local X=1.85 for output. The flat removes the circle's +X cap.
The 51-tooth input and 17-tooth intermediate pinion occupy Y = -30 to -42.
The 51-tooth intermediate gear and 17-tooth output pinion occupy Y = -60 to -72.
Build two solid imported gear prototypes with BoreDiameter=0:51 teeth and17 teeth.
Cut the large prototype's D bore before copying it for input; consume its original for intermediate-gear.
Copy the solid17-tooth prototype for the8.2 D bore; cut the5.2 D bore into the original for output.
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
| bearing-board | Rectangular plate X[-185,110], Z[290,525], Y[0,18], stock22 input/intermediate bores, stock14 output bore, and four stock6.6 tie-bolt holes at the frame corner centers below. | frame |
| front-bearing-frame | One portal plate at Y[-95,-83]. Its outer XZ outline is (-175,18),(-155,18),(-155,298),(80,298),(80,18),(100,18),(100,510),(-175,510). Remove rectangular windows X[-155,80], Z[318,362] and[398,468.3]. Add stock22 input/intermediate bearing bores, stock14 output bore, and four stock6.6 tie-bolt holes. Its two integral legs reach the base. | frame |
| frame-standoff-0 through -3 | Annuli OD20 (scaled), stock ID6.6, Y[-83,0], at XZ corner centers (-165,308),(90,308),(-165,500),(90,500), respectively. | frame |
| frame-bolt-0 through -3 | M6 through-bolt envelopes at the same centers. Stock diameter6 shank Y[-95,30]; stock diameter12 head of stock height4.2 extending forward from Y=-95*Scale. Only shank length and placement scale. | steel |
| frame-nut-0 through -3 | Rear nut envelopes at the same centers, stock OD12/ID6 and stock width5.5, Y[18*Scale,18*Scale+5.5]. Hex flats and threads are omitted. | steel |
| input-bearing-front, input-bearing-rear | Stock608-2RS envelopes, bore8/OD22/width7, at input. Front spans Y[-95*Scale,-95*Scale+7]; rear spans Y[18*Scale-7,18*Scale], in actual millimeters. | bearing |
| intermediate-bearing-front, intermediate-bearing-rear | Same two stock608-2RS envelopes at intermediate. | bearing |
| output-bearing-front, output-bearing-rear | Stock605ZZ envelopes, bore5/OD14/width5, at output. Front spans Y[-95*Scale,-95*Scale+5]; rear spans Y[18*Scale-5,18*Scale], actual millimeters. | bearing |
| input-shaft | Stock diameter8, Y[-116,65], at input. Flat at local X=3 on Y[-116,-104],[-83,-20] and[30,65]; phase a. Keep both intervening bearing journals round. | steel |
| intermediate-shaft | Stock diameter8, Y[-100,26], at intermediate. Flat at local X=3 on Y[-83,-20]; phase b. Keep both bearing journals round. | steel |
| spindle | Revolved axial section: stock radius2.5 from Y=30 to -245, taper to stock radius0.3 at Y=-270, close to axis. Flat at local X=1.75 on Y[-84,-50], phase c. Keep bearing journals and winding surface round. | steel |
| input-gear, intermediate-pinion, intermediate-gear, output-pinion | Imported spur gears in the positions and phases above. | gear |
| flywheel-rim | Annulus OD310, ID270, Y[35,53], at input. | drive |
| flywheel-hub | OD34, stock D bore radius4.1/flat X=3.1, Y[35,53], at input, phase a. | drive |
| flywheel-spoke-0 through -3 | Four radial rectangular bars, radial extent[15,137], tangential width12, Y[39,49], at input, angles a+k*pi/2. Overlap rim and hub; separate assembly solids. | drive |
| input-hub, intermediate-pinion-hub, intermediate-gear-hub, output-hub | Annuli OD24, stock ID8.2 for input/intermediate and stock ID5.2 for output. Y[-44,-52] for first pair and [-74,-82] for second pair. | steel |
| crank | Rectangle in XZ extending -9 to34 along crank direction, width18, thickness8 at Y[-113,-105]. Stock D bore radius4.1/flat X=3.1 at origin; round5.2 pin hole at radius25. | drive |
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
| guard-front | Plate X[-179,104], Z[294,514], Y[-104,-101], stock clearance holes diameter8.8 at input/intermediate and5.8 at output. | guard |
| guard-left, guard-right | Boxes X[-183,-179] and [104,108], Y[-104,0], Z[290,518]. | guard |
| guard-bottom, guard-top | Boxes X[-179,104], Y[-104,0], Z[290,294] and [514,518]. | guard |

Matching shaft/gear D profiles transmit torque directly. The input crank and flywheel hub use the same positive connection.
The four round-bore hub envelopes represent axial retaining collars; their clamp details remain to be selected.
D bores have0.1 radial and flat-face nominal clearance. Flat offsets and diameters are stock dimensions and never scale.
The shaft flat's +X direction rotates with its shaft phase. Both intermediate gears share phase b and the same D shaft.
Only exposed drive regions have flats; bearing journals remain round at their full nominal8mm or5mm diameters.
The heel shaft and three moving linkage pins require no positive rotary coupling and remain round.
Bearing IDs equal nominal stock shaft diameters.
Nominal housing bores equal bearing ODs. Fit tolerances and retainers require the fabrication process to be specified.
Use six608-2RS bearings and four605ZZ bearings. The frame holds the two bearings for each rotating shaft.
The front bearings seat in the permanent portal frame; the rear bearings seat in the rear bearing board.
The gears lie between these supports. Nominal bearing-center spans are106mm for8mm shafts and108mm for the spindle.
Four standoffs and M6 through-bolts clamp the front frame to the rear board. Rear nuts are independent of the guard.
The portal's integral feet meet the base at Z=18*Scale; its two windows leave the drivetrain visible for servicing.
The guard carries no bearing, shaft seat, standoff or frame fastener. All five guard bodies can be removed
without removing or moving any frame, shaft, gear or bearing body. Its shaft holes are clearance holes only.
The bolt-head envelope leaves at least0.6mm clearance from the guard at Scale=0.8; the crank leaves0.8mm.
Bolt shank lengths are100,125 and150mm at Scale0.8,1 and1.2, respectively.
The bolt heads, nuts, thread diameters and bearing sizes never scale. Select matching purchased fasteners.
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
Check the actual D-shaped bores and local shaft flats, including their end positions and round bearing journals.
For every positive connection, section the assembled bodies and require clearance; rotate only the shaft90deg and require overlap.
Check paired rigid rotation at several angles preserves the positive coupling and keeps the gear phases aligned.
Check actual linkage bodies against the shaft, supports, guard and each other throughout a full turn.
Check both heel brackets,8mm shaft,608 bearing seats, collar abutments, spacer gaps and through-shaft engagement.
Check the permanent front frame's eight outer vertices, two windows, seven circular bores and base contact.
Check all four standoff/bolt/nut paths through the front frame and rear board, including stock hardware clearances.
Verify both journals of all three shafts are round at their actual bearing centers and the gears lie between bearings.
Remove all guard bodies and require every structural support, bearing and shaft to remain at its original placement.
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
Omit all frame bolts and nuts in drive-only views as well; show them with the permanent frame in other views.
Render frame-support.png at2000x1100 with the cover removed. Show the complete drivetrain on the permanent frame
in one oblique panel and isolate the actual frame, six shaft bearings and three shafts in the other panel.
Keep all bodies in their assembled positions. Label the front and rear bearing supports and the four through-bolts.
Render coupling.png at2000x1100 with the actual input gear's D bore alone in a front close-up,
and the actual shaft and gear together in an oblique close-up at their assembled positions.
Crop the outer teeth deliberately, label that crop, and keep the bore/shaft large enough to see their flats.
Hide only nonselected bodies; do not draw substitute coupling geometry or move the gear off the shaft.
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

- [ISO4017:2022 head dimensions](https://cdn.standards.iteh.ai/samples/72585/1179998ddd594550a909d6e13c416f3e/ISO-4017-2022.pdf),
  Table1, give M6 head height up to4.15mm and10mm across flats. The modeled head envelope is12mm diameter
  and4.2mm high. Choose the actual long bolt separately; the model does not certify a complete standard fastener.
- [Ashford quill instructions](https://www.ashford.co.nz/instructions/QS.pdf) describe drafting from the point
  and changing yarn angle for winding. This design adopts that operating method, not its pulley drive.
- [Ashford double-treadle kit](https://www.ashford.co.nz/instructions/DTKTD.pdf) identifies crank,
  connecting rods and supports in a treadle drive. This design uses one calculated crank-rocker linkage.
