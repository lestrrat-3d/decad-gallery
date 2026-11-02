# Manufactured joints and mounted parts

## Authority and order

This file owns final joinery, hardware retention and physical-part grouping.
instructions.md owns other dimensions. Older timber dimensions describe stock blanks before these cuts.
Apply the changes below before returning Build. Never return interpenetrating stock as separate finished objects.
Use native bodies and fully verified Sketch profiles for every tool. Do not weaken geometric verification.
If an operation refuses, pursue a generic Decad repair; the user explicitly prefers that route over a workaround.
Nominal metal thread envelopes may share their mating cylindrical boundary; do not overlap solid material.
Wood joints below are bonded fitted joints. Describe their assembly as glued joints; do not invent hidden fasteners.

## Positive couplings and axial capture

Keep stock8 round wheel/heel shafts and stock16 front flyer tube. Keep all purchased bearings and round journals.
Use removable parallel keys for their hubs. A keyed bore contains the entire round journal, so it can slide into place.
Do not trap a D-bore hub behind an unflattened journal. The one-piece whorl remains one native revolved body.

For stock8 couplings, shaft key pockets have radialX[2.8,5], transverseZ[-1,1]; key material isX[2.8,4.8],Z[-1,1].
Their through-bores are circleR4.1 united with the rectangleX[2.8,4.9],Z[-1.1,1.1].
For stock16 couplings, pockets areX[7.2,9],Z[-1.5,1.5]; keys areX[7.2,8.7],Z[-1.5,1.5].
Their through-bores are circleR8.1 united with rectangleX[7.2,8.8],Z[-1.6,1.6].
All these cross-section dimensions are stock. X/Z here are the corresponding shaft frame's radial coordinates.
Create one primary line/arc outline for each keyed bore, not a faceted approximation or an extra prior round bore.
Its major circle arc runs fromangle asin(a/R) to2*pi-asin(a/R), with the three keyway lines closing at radialX=m.
Use coincident line/arc endpoints, H/V, origin anchoring, radiusR, halfwidtha and radial extentm.
Do not redundantly dimension a computed intersection. Verify the completed sketch before use.
Bore area is pi*R^2+2*a*m-a*sqrt(R^2-a^2)-R^2*asin(a/R).

Wheel axle spansY[-60*Scale,92*Scale+2], diameter8. It has no D flats.
Its hub key/pocket spansY[-12,12]*Scale; its crank key/pocket spansY[79,85]*Scale.
Keep crankY[78,86]*Scale. Add stock12 OD,8 ID,3-wide root collars:
wheel-crank-front-collar spansY[78*Scale-3.2,78*Scale-0.2]; rear spans[86*Scale+0.2,86*Scale+3.2].
Add wheel-hub-front-spacer overY[-45*Scale+7,-15*Scale] and rear over[15*Scale,45*Scale-7], stock8 ID/12 OD.
Add wheel-crank-pin-spacer, stock5 ID/8 OD, overY[86*Scale,105*Scale-2.5], axis through CrankPin.
These wheel parts move with wheel. Key IDs are wheel-hub-key and wheel-crank-key, Group key.

Heel shaft keeps its original ends and diameter8, with no D flat.
Heel key/pocket spans localX[-220,180]*Scale. Its radial coordinates are PedalFrame.V and PedalFrame.N.
The heel rail has the corresponding keyed8.2 through-bore. Key ID is heel-key, Motion treadle.
The footboard receives that same keyed passage where the shaft crosses it.

Flyer tube keeps its10mm orifice,6mm radial outlet and stock16/8 outer steps.
Replace its D flats with the stock16 key pockets over localY[-281,-251]*Scale and[-219,-211]*Scale.
Keys are flyer-whorl-key and flyer-crossbar-key, Motion flyer. Hub bores use the matching stock16 keyed outline.
Add an integral stock20 OD shoulder over localY[-222,-220]*Scale, outside the tube; it stops the crossbar's front face.
In the unplaced meridian this is outerR10 between axial194*Scale and196*Scale, returning toR8 on both sides.
Keep the hollow bore underneath. Do not cut the shoulder away as a D flat.
Add crossbar-rear-collar, stock16 ID/20 OD,width4, overY[-210*Scale+0.2,-210*Scale+4.2].
Increase front-sleeve-retainer, whorl-front-collar and whorl-rear-collar to stock width4:
front-sleeve Y[-314*Scale-4.2,-314*Scale-0.2], front-whorl[-282*Scale-4.2,-282*Scale-0.2],
rear-whorl[-250*Scale+0.2,-250*Scale+4.2]. Round journals and the port remain unchanged.

Every shaft collar gets a real stockM2 radial grub screw. Do not apply this to bearing end plates.
Choose radial direction opposite the shaft frame's positive radialX, avoiding all keys/slots.
For heel collars this is -PedalFrame.V; cross-pin collars use -GimbalFrame.U.
For wheel/flyer collars use the negative rotated radialX, including head tilt for flyer parts.
At the collar's axial midpoint drill a stockR1 radial thread-envelope passage from the shaft axis through the OD.
Grub screw isR1, from radial distance innerRadius to sqrt(outerRadius^2-1), recessed within the collar OD.
It touches the round shaft at its tip. Name it <collar-id>-setscrew, same Motion, Group fastener.
Keep all original stock-width3 shaft collars except the width4 collars explicitly listed above.

Keep D-shaped jack/brake knobs, but extend their shaft flats to the installation end.
Jack flat starts axial42*Scale and runs through the shaft end58*Scale.
Brake flat starts at its front shaft tip and runs through the knob's rear face.
Bond each knob onto its keyed shaft against the round-root shoulder. Record this as a bonded-keyed mount.

## Upper gimbal and footman

Make gimbal-bridge one machined clevis body incorporating both cheeks; remove the separate cheek Parts.
Cheeks now span GimbalFrame.v[-27,-20] and[20,27], scaled; other u/w dimensions stay unchanged.
The bridge's stock5 thickness and existing605 seat remain unchanged.
Two M2 tie-bolt holes run along GimbalFrame.N at(u,v)=(0,+9) and(0,-9), stock coordinates.
End plates gimbal-crank-retainer-1/2 are stock ID10/OD22,width2, with those twoR1.1 holes.
They directly abut the bridge's w faces: intervals[-4.5,-2.5] and[2.5,4.5], stock.
Two native one-piece headed M2 bolts span shankw[-4.5,6.5], headR2 atw[-5.5,-4.5].
Their nuts are ID2/OD4,width2 atw[4.5,6.5]. IDs gimbal-retainer-bolt-1/2 and gimbal-retainer-nut-1/2.
The end plates cover only the outer race and do not touch the crank pin's retaining collars.

Cross pin now spans v[-30*Scale-3,30*Scale+3]. Its collars lie0.2stock beyond the new cheek outer faces.
Add gimbal-cross-spacer-1/2, stock5 ID/8 OD, over v[-20,-12]*Scale and[12,20]*Scale through RodHead.
They contact the bearing inner races and the clevis inner faces while clearing both retaining plates.

Increase the long footman's thickness fromw[-4,4] to[-5,5], scaled. Keep v[-10,10] and u[20,476], scaled.
Cut its matching socket into the upper boss over u[460,476],v[-10,10],w[-5,5], scaled.
Use an unperforated rod-stock tool; do not leave plugs where the clamp passage is bored.
Keep both605 seat intervals and the cross-pin centre unchanged.
Footman end plates are rectangles u[-18,12],w[-10,10], scaled relative to RodHead, central stockR5 opening.
Their thickness2stock directly abuts the boss faces: v[-12*Scale-2,-12*Scale] and[12*Scale,12*Scale+2].
One stockR1.1 cross-bolt passage lies at u=-12stock,w0 relative to RodHead, through plates, boss and rod.
Bolt shankR1 spansv[-12*Scale-2,12*Scale+4]; headR2,height1 lies immediately before its negative end.
Nut ID2/OD4,width2 startsv=12*Scale+2. IDs footman-retainer-bolt and footman-retainer-nut.
The wider clevis clears the plates and both bolt ends throughout articulation.
Plate IDs remain footman-cross-retainer-1/2. Bearing end plates use Group retainer, not collar.

## One flexible connector and its clamps

Return one treadle-flex-connector, Group compliant, Motion compliant; remove the two separate connector Parts.
Its native volume is the union of the existing two radius3 legs, with both stock4.4 clamp passages.
The leg centres/endpoints and clamp datums remain unchanged.
Expose Connector{Body *decad.Body; Sketches []*sketch.Sketch} and BuildConnector(ctx,doc,Params)(*Connector,error).
This uses the same construction helper as the assembly; all sketches are returned and fully verified.
Validate ctx/doc/parameters before mutation. Compute its pose through Motion.
PoseTransform refuses compliant with a contextual error: that part must be rebuilt, not rigidly moved.
The renderer and pose-clearance tests rebuild this one real native component at each pose.

Drill stockR3.2 sockets into the timbers for its ends:
connector block: PedalFrame normal-axis bore through local(X,Y)=(220,450)*Scale, localZ[3,13]*Scale;
footman: RodFrame.u-axis bore through(v,w)=(0,0), u[19,33]*Scale.
Keep the actual stock4.4 transverse clamp passages.
Each cross-clamp bolt is ONE native stepped revolution, keeping the existing <clamp>-pin ID.
Its stockR2 shank spansaxis[-face,face+2.2]; stockR3.5 head spans[-face-2,-face].
Its nut is stock ID4/OD7,width2 over[face,face+2]. Remove separate -head Parts.
Here face=12*Scale for treadle-lower-clamp,10*Scale for footman-upper-clamp.

## Wood frame, posts and clamp inserts

Extend both wheel bearing-cap blanks down toZ535*Scale. Cut their unperforated envelopes into their legs.
Cut the four sloping-rail envelopes into the corresponding three legs. Preserve remaining wood around each socket.
The wedges remain bonded to the matching sloping rail faces.
Move alignment clamp axes from globalY=-60*Scale to-70*Scale, including their wedge passages.
Add mother-alignment-insert-1/2, stock10x10x8 square M6 inserts, boreR3, immediately below wedge topZ525*Scale.
Their square seats have stock halfwidth5.1, start atZ525*Scale-8, and open through the wedge top.
The base captures them after assembly. The screws retain their original z range/head dimensions.

Extend each maiden post's root down fromZ585*Scale to575*Scale; other post bounds and bearing seats stay unchanged.
Cut that unperforated stock tenon into its foot overZ[575,585]*Scale. Bond the post into its foot.
Move each foot bolt centre from postFront*Scale-6stock to postFront*Scale-6.2stock.
Move the matching bar/foot passages too. Keep bolt/head z ranges unchanged.
Add maiden-foot-<1|2>-insert-<1|2>, stock10x10x8 M6 square inserts immediately below the upper bar's Z570*Scale face.
Their stock10.2 square seats open upward fromZ570*Scale-8. The foot captures them after assembly.
The rear brake crossbar also clears these same insert envelopes where they cross it.
The rear post and its glued foot remove together: undo foot bolts, slide the unit rearward off the shaft, then lift it.

For each hinge, fuse each fixed knuckle with its bracket, keeping the fixed-bracket-<j> ID.
Fuse the moving knuckle with its leaf, keeping moving-leaf. These are individual machined metal hinge leaves.
Use stock5.4 bore/10 OD knuckles and stock5 headed hinge pins; do not return separate overlapping knuckles.
One M3 mounting passage per fixed bracket is atX=-183*Scale and the Y interval's midpoint.
Its metal clearance radius is1.6; its base pilot isR1.5 overZ[532,540]*Scale.
One M3 moving-leaf passage is atX=-162*Scale,Y=q+23.5*Scale, metalR1.6.
Cut the leaf's stock envelope into the bar overZ[555,558]*Scale; drill the barR1.5 overZ[558,566]*Scale.
Fixed screws haveR1.5 shanks Z[532,555]*Scale and stockR3,height1.5 heads above555*Scale.
Moving screws haveR1.5 shanks Z[555,566]*Scale and stockR3,height1.5 heads below555*Scale.
IDs mother-hinge-<n>-fixed-mount-<j> and mother-hinge-<n>-moving-mount, Group fastener, Motion fixed.
The hinge pin hasR2.5 shank overY[q,q+47*Scale+3.2], stockR4,height2 head beforeq.
Add mother-hinge-<n>-nut, stock ID5/OD8,width3, startingY=q+47*Scale.
Fixed leaves/pin/nut do not tilt; moving leaves and their mounting screws share head tilt.

Cut the unperforated upper-bar envelope out of the jack arm, leaving the arm's front strip connected.
Cut the rear brake crossbar's stock envelope out of the upper bar, making its5*Scale deep top seat.
Cut the rear maiden foot's stock envelope out of that crossbar, plus its two M6 insert seats.
Move the front brake upright to localY[-338,-330]*Scale, clearing the maiden foot and bolt heads.

## Treadle half-laps

Work in PedalFrame. Keep the heel rail's lower half inside the footboard's footprint, and the board's upper half.
Cut heel-rail upper-half stock Z[0,12]*Scale by the board footprint; cut board lower half by heel-rail stock.
At heel-rail/diagonal intersection, keep heel-rail upper half and diagonal lower half.
At board/diagonal intersection, keep board upper half and diagonal lower half.
Use unperforated half-thickness tools with the original stock centreline/width, then bore the board's keyed shaft passage.
Cut the rear crossarm's full stock envelope out of the diagonal's end; its upper half remains connected.
Cut the same crossarm stock out of the connector block; its upper half/right side remain connected.
Do not delete a whole overlap that severs the board or a rail into two disconnected pieces.

## Flyer and guide seats

Cut the unperforated arm stock out of the thicker crossbar. The crossbar's outer straps remain connected.
Each ordinary guide is ONE formed metal item containing hook and stem. Front guide also includes its stalk.
Ordinary guide stems start atZ7*Scale, not10*Scale; retain the current hook geometry, mouth and yarn bisectors.
Drill matching stockR1 stem seats into each arm from localZ6*Scale through11*Scale.
Use the actual projected-bottom attachment X/Y from the hook's oblique axis, not just its centre.
Drill a stockR1 front stalk seat alongY[-221,-214]*Scale at localX58*Scale,Z12*Scale in the crossbar.
The front guide keeps its existing12*Scale stem bottom and stalk endpoint atY=-215*Scale.
Record the guide shanks as bonded threaded-envelope seats; no shank may overlap solid wood.

The wound pack's outer radius is31*Scale-0.35stock, so the radius0.35 feed strand is tangent at radius31*Scale.
Keep its other dimensions. The orange marker is a0.2stock thick curved label on that cylinder.
It spans localY[-138,-130]*Scale, centred on radial+Z before bobbin rotation.
Its halfangle is asin(2*Scale/(31*Scale-0.35)); build an exact partial annular Revolve, not a box inside yarn.
Keep bobbin-witness-bar ID/marker Group for renderer compatibility, although its physical shape is now a curved label.
Return yarn-leader as ONE native continuous solid from the existing six radius0.35 spans, with the corrected pack tangency.
Union their adjoining native material; do not return intersecting span Parts as separate pieces.

## Captive Scotch attachments and one cord

Keep rear groove, two tails and complementary travel. Keep spring coil axisY and6native turns per spring.
Spring coil centres span[y0+0.4,y1-0.4], radial2.1, wire0.4. Each spring has closed end eyes radius2.1/wire0.4.
Front eye centre is(x,y0-3,z), rear eye(x,y1+3,z); eye planes areXY before phase rotation.
Straight end legs join each coil endpoint at radial+2.1*X to the eye's radial+2.1*X point.
Return the coil, two legs and two eyes as one formed/welded spring body, IDs brake-spring-left/right.
Left phase is0; right phase is the take-up screw's2*pi*BrakeTravel/0.7 about the spring's Y axis.
Both right eyes and their terminal fittings share that phase. The flexible cord eye accommodates this orientation.

Front spring eyes receive stock3 clevis pins. Define L=(x,y0-3,z).
Clevis blank isX[-3.5,3.5],Y[-6,2.8],Z[-3,3] around L, stock.
Its open-back slot removesX[-4,4],Y[-3,3.8],Z[-0.8,0.8]. Its transverse pin passage isR1.6 alongZ through L.
Pin is one peened rivet: shaftR1.5 overZ[-3,3], headsR2.3 over[-4,-3] and[3,4], stock.
IDs brake-front-pin-left/right. Neither pin nor eye may overlap the clevis cheeks.
The left clevis and its stockR0.8 stem are one brake-left-anchor; its root penetrates6stock into the control rail.
Drill a bonded R0.85 blind seat for that stem. The right clevis is integral with brake-takeup-screw.
Its round shaft ends atY=y0-6, overlapping only its own clevis's front block before native fusion.
Keep the screw's front endpoint and knob placement from instructions.md.

Each cord terminal is a closed radius2.4/wire0.6 eye in the YZ plane, centred(x,y1+6,z).
It links the spring's rear eye, whose centre is3stock farther forward and whose plane is perpendicular.
The two separate loops must have positive wire clearance while remaining interlocked.
Rotate the right terminal's plane by the right spring phase aboutY; its centre stays on that axis.
Each forward cord run ends at(x,y1+8.4,z), spliced into the far point of that eye.
Both eyes and the main cord are ONE physical brake-band body.

Round both rear drop/forward corners at stock radius5, below the flyer.
Construct the main cord with one native circular Sweep, radius0.6, not intersecting cylinder Parts.
For its canonical initial frame use U=worldX,V=-worldZ,N=worldY at(leftX,leftEndY+8.4,600*Scale).
Let h=216*Scale-13.4-BrakeTravel/2 and Rb=42*Scale+0.6*sqrt(5).
Its local path starts(0,0,0). Use LineTo and explicit-center ArcAround segments, in this order:
line to(0,0,h);
quarter arc with Center(0,-5,h), Axis(+1,0,0), End(0,-5,h+5);
line to(0,-90*Scale,h+5);
half arc with Center(Rb,-90*Scale,h+5), Axis(0,0,+1), End(2*Rb,-90*Scale,h+5);
line to(2*Rb,-5,h+5);
quarter arc with Center(2*Rb,-5,h), Axis(-1,0,0), End(2*Rb,0,h);
line to(2*Rb,0,-BrakeTravel).
ArcAround records the exact dyadic center, oriented axis and endpoint. It never infers a center from a through point.
Require nonzero equal endpoint radii and exact perpendicularity to the axis; do not admit near tangency by tolerance.
The axis selects the positive arc direction. Preserve the exact quarter/half turns and tangent rays at all six joints.
Do not replace ArcAround with ArcThrough: rounding the half-turn's through point loses exact line/arc tangency.
The circular section starts at the origin XY plane with positiveZ tangent. Fully verify it before Sweep.
Place the native result, then fuse the two spliced eye bodies. Keep resulting material separate from spring eyes.
State.BrakeLength is the nominal centreline length including eyes: pi*Rb+612*Scale-36.8+14.6*pi.
It excludes splice overlap/compression. Native cord volume must remain constant as BrakeTravel changes.

Routing eyes and their stems are single formed brake-routing-eye-left/right Parts.
Move each eye to the rounded cord corner's midpoint:
Y=-69*Scale+HeadShift-5+5/sqrt(2), Z=600*Scale+5-5/sqrt(2), at that tail's X.
Keep its axis normalize((0,-1,-1)). The stem's upper attachment is eyeCentre+(0,sqrt(2),-sqrt(2)).
Use that same attachmentY for its vertical root and matching timber seat.
Extend each stem4*Scale into the rear crossbar and drill its R0.85 matching bonded seat.
Keep the stock ID3/OD5,width1.2 circular guide opening and45-degree orientation.
The front insert moves flush to the rail's front face Yf=-338*Scale+HeadShift, spanning[Yf,Yf+4stock].
Its square8.2 pocket opens fromYf-1 throughYf+4; retain the central M4 passage.
Increase its retaining plate to stock16x16,width1; put stockR1.1 screw holes atX=rightTailX+-6,Z=600*Scale.
Add brake-insert-mount-1/2 M2 screws: stockR1 shanksY[Yf-1,Yf+7], stockR2,height1 heads beforeYf-1.
Drill corresponding R1 wood pilots behind the plate, outside the insert pocket.

## Mount inventory and checks

Expose Mount{Part,Kind string; Supports,Fasteners []string; Clearance float64} and Assembly.Mounts []Mount.
Return exactly one mount record per final physical Part. Every referenced support/fastener must exist.
The three legs are ground mounts with no support; their actual lower faces meetZ0.
Other mounts must have a path to a ground mount through real declared supports.
Record glue sockets/laps/faces, press-fitted bearings, journals, captive spacers, removable keys,
set-screw collars, bolted end plates, clamp inserts, threaded/bonded knobs, captive loops and process yarn distinctly.
Do not infer a mount just because two bounding boxes are nearby.
Collar grub screws refer to their collar; end plates refer to their boss/clevis and their actual tie bolts.
Hinge leaves refer to the wood they screw into; post roots refer to their machined feet and glued tenons.
Foot and alignment bolts refer to their real square inserts; inserts refer to their square seats and capturing board.
Whorl/crossbar/crank/hub/heel rail refer to their shaft, removable key and axial retaining hardware.
Spring front mounts name their clevis and transverse pin; cord terminal mounts name both spring end eyes.
Clearance records allowed nominal boundary separation at the current pose; geometry checks charge mesh bounds separately.
Both spring mounts use0.2stock clearance: eye inner radius2.1-0.4=1.7 minus pin radius1.5.
This gap permits the closed eye to move around its pin. Check closed-eye topology and rivet-head capture separately.
Use0.5stock clearance for the cord's captive terminal mounts. The two rear eyes specified in
Captive Scotch attachments and one cord have minimum centreline separation1.5stock;
their two wire radii leave0.5stock between material boundaries. Common phase and head rotations preserve that gap.
Check their linked closed-loop topology and material separation independently; proximity alone does not prove capture.
Guide mounts name their bored arm/crossbar. The yarn pack names bobbin; the curved label names the pack.
The leader names its guide path and pack, with a deliberate free input end.
These records describe intended mounts; geometry tests must independently verify seats and material separation.

## Emission ownership

couplings.go owns keyedOutline, builder.keyTool and builder.keyedAnnulus with the signatures in steps.md.
drive.go owns revised wheel/shaft/key/spacer/clevis/rod/clamp/treadle construction.
connector.go owns the native flexible component and its shared exported BuildConnector path.
frame.go owns revised caps, hinges/fasteners, posts' feet, inserts and their raw through-passages.
head.go owns revised hollow shaft/keys/shoulder/crossbar/post roots/pack/label and retains headDirection ownership.
yarn.go owns the single formed guides and single leader. brake.go owns the revised Scotch parts except brakeCord.
brake_cord.go owns builder.brakeCord and its exact-center Sweep plus spliced terminal eyes.
joinery.go owns finishJoinery: all fitted wood cuts, guide/stem/pilot holes, insert seats, collar screws and Mounts.
Use stock tools without their through-holes for wood sockets; do not leave plugs in another part's bolt passage.
Cut all tools for one host together where supported. Copy any donor body before consuming it as a Boolean tool.
assembly.go calls finishJoinery after all initial parts, with tilt=false. Update affected Part.Body handles in place.
Head tools receive the exact common head tilt; heel/rod tools use their full State frames.
Motion and every geometry tool must keep the same physical frames. Return no retired handles or leftover tools.
