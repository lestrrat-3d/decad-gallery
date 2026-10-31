# Treadle spinning wheel

## Navigation

This router is non-authoritative; the linked sections own the dimensions.
- API and parameters: validation, coordinates, output types and stable part roles.
- Drive and treadle: spatial closure, gimbal, compliant connector and cross-machine rocker shaft.
- Open frame: three legs, sloping rails and bored bearing supports.
- Head adjustment and drive whorl: hinge, jack, groove selection and front whorl.
- Flyer, bobbin and bearings: hollow tube, posts, seats and shaft retention.
- Yarn route: open hooks and illustrative leader.
- Scotch brake: front controls, two springs and conserved cord length.
- Verification and images: acceptance boundaries.

## Purpose

Build an open Saxony wheel with a flyer, independently rotating bobbin, hollow yarn orifice,
single drive band, Scotch bobbin brake, and crank-driven treadle.
Use the arrangement in [George Weil's labelled wheel](https://www.georgeweil.com/blog/how-does-a-spinning-wheel-work/).
Use [Ashford's spinning manual](https://www.ashford.co.nz/booklet/learn-to-spin.pdf) for yarn routing and Scotch tension.
Use [Ashford's Traditional specifications](https://www.ashford.co.nz/product/traditional-single-drive/)
for the 560 mm wheel, 690 mm orifice height, 10 mm orifice, and 7/9/12/17 ratios.
Other dimensions below define this gallery design; they are not Ashford manufacturing drawings.

The old pedalspindle model is a quill spindle and remains a separate item.
Do not reuse its geared head or guards. Reuse the ballbearing construction by import.

## API and parameters

Package spinningwheel. Expose Params{Scale,WheelAngle,DriveTension,BrakeTravel,YarnFeedPerTurn float64; Ratio int}.
Defaults: 1, pi/3, 6, 3, 100, and ratio 7 respectively.
Scale range [0.8,1.2]; finite WheelAngle; DriveTension [0,12] nominal mm;
BrakeTravel [0,6] actual mm; YarnFeedPerTurn [0,200] actual mm per wheel turn;
Ratio one of 7,9,12,17. Reject invalid inputs before document mutation.
Reject a finite WheelAngle if Ratio*WheelAngle cannot be represented as a finite float64 phase.
Nominal dimensions scale by Scale unless marked stock. Stock shafts, bearings, bores, cord and thread pitches do not scale.
All lengths are mm; angles radians. X runs along the bench, Y away from the operator, Z up.

Expose DefaultParams, Params.Validate, Motion(Params)(State,error), Build(ctx,doc,Params)(*Assembly,error).
State exposes Wheel,Flyer,Heel,CrankPin,Toe,RodHead,HeadPivot r3.Vec;
PedalAngle,RodAngle,FlyerAngle,BobbinAngle,HeadTilt,
WheelPitch,FlyerPitch,BeltLength,BrakeLength,SpringLength,HeadShift float64;
PedalFrame,RodFrame,GimbalFrame r3.Frame.
Wheel/Flyer are physical axis points in belt plane Y0. Other points are physical three-dimensional joint positions.
Assembly exposes Parts []Part, Sketches []*sketch.Sketch, State State.
Part exposes Name,Group,Motion string and Body *decad.Body.
Motion is fixed, wheel, flyer, bobbin, treadle, gimbal or footman; it names the rigid motion for animations.
All parts have unique names, exact native solid bodies, and no leftover tool/prototype bodies.
All consumed sketches, including imported bearings and cut tools, are returned and verified.

Head part IDs and role groups are a stable contract for renderer selection and geometry checks:
- flyer-shaft: Group shaft, Motion flyer; the completed hollow shaft, outlet and flats.
- flyer-drive-whorl: Group whorl, Motion flyer; one grooved body with its D through-bore.
- flyer-crossbar, flyer-arm-1, flyer-arm-2: Group flyer, Motion flyer.
- bobbin: Group bobbin, Motion bobbin; the wooden stepped body.
- bobbin-yarn-pack: Group yarn-pack, Motion bobbin.
- bobbin-witness-bar: Group marker, Motion bobbin.
- bobbin-inner-race-sleeve: Group spacer, Motion flyer; the120*Scale long11 OD,8 ID sleeve.
Use Group bearing/Motion fixed for stock head608 envelopes, bush/fixed for front nylon sleeve,
frame/fixed for maiden posts, spacer/flyer for other shaft sleeves, and collar/flyer for retainers.
Other head IDs must be unique and descriptive. Do not label the shaft or sleeve as a flyer-arm group.

## Drive and treadle

Wheel centre (220,0,580)*Scale; radius280*Scale; inner rim radius255*Scale.
Crank pin C=(220+25*cos(a),105,580+25*sin(a))*Scale, a=WheelAngle.
The treadle axis runs along world X through Y=-190*Scale,Z60*Scale. Heel=(60,-190,60)*Scale.
PedalFrame has origin(0,-190,60)*Scale, U=(1,0,0), V=(0,cos(psi),sin(psi)), N=U.Cross(V).
Its local XY plane carries the footboard and diagonal actuating rail. Positive psi raises their rear edges.
The lower compliant-joint centre D=Toe is PedalFrame applied to (220,450,20)*Scale.
For each candidate psi, W0=normalize((D.X-C.X,0,D.Z-C.Z)), V0=(-W0.Z,0,W0.X).
The upper gimbal cross-pin centre S=RodHead=C+32*Scale*W0.
Solve |D-S|=486*Scale by bisection in psi[-0.12,0.20]; require opposite endpoint signs and residual<1e-8*Scale.
Use the same continuous branch for every wheel angle. Do not replace the skew linkage with a planar four-bar.
GimbalFrame has origin C, U=W0, V=V0, N=U.Cross(V)=(0,-1,0).
RodFrame has origin D, U=normalize(S-D), V=V0, N=U.Cross(V).
RodAngle=atan2(S.Z-D.Z,S.X-D.X) is the projected angle; use the full frames to place/animate rod and gimbal.
The treadle rocks about its cross-machine shaft. Wheel inertia carries the crank through dead centres and raises it.
Start the wheel by hand in the wanted direction. No ratchet, return spring or one-way clutch is required.

Wheel rim spans Y[-10,10]*Scale, with V groove at Y0: radius280 at Y=+-3,
radius274 at Y0; groove slope2. Stock drive cord radius1, pitch radius Rw=274*Scale+sqrt(5).
Hub radius30*Scale, Y[-15,15]*Scale, stock D8.2 bore with flat at X3.1.
Six spokes radial25..260, tangential width18, Y[-6,6], all scaled, rotate with wheel.
Spoke/hub/rim overlaps are named rigid wood joints, not moving interference.
Wheel axle stock diameter8, Y[-60,88]*Scale. D flats X3 over Y[-18,18] and [76,88], scaled axial limits.
Round bearing journals remain full diameter. Crank Y[78,86], outline local [-9,34]x[-9,9],
D8.2 root bore/flat3.1, stock5.2 pin hole at radius25*Scale. Rotate together with wheel.
Stock5 crank pin spans Y[78*Scale-3.5,112*Scale+3.5]. Retain it with stock9 OD,5 ID,3 wide
collars immediately outside the crank's front face and behind the gimbal bearing, with0.2 stock axial gaps.
One stock605ZZ is centred on C, axis Y; its outer race is carried by the upper gimbal bridge.
In GimbalFrame coordinates (u along W0,v along V0,w along -Y), bridge u[-11,10],v[-22,22],
w[-2.5,2.5] stock axial width5, with stock14 through-seat at (u,v)=(0,0).
Two outer-race retaining rings ID10,OD18,width2 stock lie beyond the bridge faces with0.2 gaps.
Gimbal fork cheeks have v[-22,-15] and[15,22],u[8,46],w[-9,9], scaled.
Stock5.2 cross-pin holes run along v through (u,w)=(32*Scale,0).
The stock5 cross-pin runs along V0 through S, v[-25*Scale-3,25*Scale+3].
Its stock9 OD,5 ID,3 wide end collars start0.2 beyond each cheek's outer face.
Bridge, cheeks, crank605, cross-pin and its collars use Motion gimbal; rigid overlaps are named mounting joints.

Footman long timber uses RodFrame coordinates u[20,476],v[-10,10],w[-4,4], scaled.
Its upper boss is u[460,498],v[-12,12],w[-10,10], scaled, with a stock14 bore along v at u486,w0.
The16*Scale overlap joins the wood parts below the bearing seat; the long timber never obstructs that seat.
Two stock605ZZ sit in that boss: v[-12*Scale,-12*Scale+5] and[12*Scale-5,12*Scale].
An8 OD,5 ID spacer fills v[-12*Scale+5,12*Scale-5].
Stock ID10,OD18,width2 outer-race retaining rings lie0.2 outside the boss faces.
These parts use Motion footman. The boss bore axis coincides with the gimbal cross-pin throughout rotation.
The lower wood end stops20*Scale above D. Model the compliant connector as two adjoining radius3 cylinders:
from D-16*Scale*PedalFrame.N to D (Motion treadle) and D to D+32*Scale*RodFrame.U (Motion footman).
They denote an ideal flexible joint, not a plain bearing forced onto skew axes.
The lower8*Scale of the lower leg overlaps the rail block; the last12*Scale of the upper leg overlaps the rod timber.
Lower clamp centre is D-12*Scale*PedalFrame.N, axis PedalFrame.U, pin length30*Scale+4 stock diameter4.
Upper clamp centre is D+26*Scale*RodFrame.U, axis RodFrame.V, pin length24*Scale+4 stock diameter4.
Centre each pin on its clamp point; drill stock4.4 coaxial clearance through both wood and compliant leg.
Add stock7 OD,2 high button heads and7 OD,4 ID,2 wide nut envelopes0.2 outside each timber face.
Lower faces are at+-12*Scale along its pin axis; upper faces at+-10*Scale. Heads/nuts follow their wood's Motion.
Do not claim flexure strain, torque or fatigue verification.

In PedalFrame coordinates, the footboard is x[5,115],y[-30,240],z[-12,12], scaled.
The rotating heel rail is x[-233,193],y[-12,12],z[-12,12], scaled, with stock D8.2/flat3.1 bore along X.
The diagonal actuating rail is a width24,thickness24 scaled prism joining local(-160,0,0) to(170,450,0),
with square ends perpendicular to its centreline. It stays left of the rear wheel leg throughout the stroke.
An offset rear crossarm spans localX[160,220],Y[438,462],Z[-12,0], scaled.
A full-thickness clamp block spans localX[208,232],Y[438,462],Z[-12,12], scaled, below D.
Return three separately named rigid timbers: treadle-actuating-rail (diagonal), treadle-rear-crossarm,
and treadle-connector-block. Their overlaps denote glued/lapped wood joints, as with the wheel spokes.
Do not Union them. The lower clamp passage belongs only to treadle-connector-block.
The half-height crossarm stays below the lower connector's X-directed clamp pin and both of its fasteners.
Do not run the diagonal straight through the splayed leg to reach D.
Board, heel rail and diagonal rail are separate named rigid wood joints, Motion treadle.
An8 mm stock rotating heel shaft spans X[-283,243]*Scale at Y=-190*Scale,Z60*Scale.
Cut its stock flat at local radial coordinate3 over X[-234,194]*Scale only; journals stay round.
Two stock608 bearings have axis X and centres(-260,-190,60)*Scale and(220,-190,60)*Scale;
their stock7 widths are centred on those X coordinates. They seat in the two structural leg bores.
Stock12 OD,8 ID,3 wide collars lie0.2 outside each bearing pair's outer width face.
Stock11 OD,8 ID spacers fill from each bearing's inner face to the heel-rail end, with0.2 end gaps.
Keep the heel shaft axially captured while permitting its rotation; do not put fixed collars across the moving rail.

## Open frame

All coordinates here are nominal, scaled. Use separate solid timbers and real bored bearing posts.
Use a three-leg frame rather than vertical posts on separate sled feet.
Single head leg: YZ polygon(-220,0),(-180,0),(120,500),(-100,500), extruded along X[-280,-240].
Front wheel leg: YZ polygon(-230,0),(-190,0),(-25,550),(-65,550), along X[200,240].
Rear wheel leg: YZ polygon(190,0),(230,0),(65,550),(25,550), along X[200,240].
Head leg and front wheel leg have stock22 heel-bearing bores along X at (Y,Z)=(-190*Scale,60*Scale).
Two sloping rails use XZ polygon(-280,465),(240,375),(240,410),(-280,500),
at Y[-85,-55] and[55,85]. Two wedges at those same Y spans support the head base:
their X[-195,-105], upperZ525, lowerZ=500-(X+280)*90/520. Their lower surfaces touch the rails.
Separate wheel bearing caps X[200,240],Y[-55,-25] and[25,55],Z[550,610], stock22 bores at wheel centre.
Stock608 bearings span Y[-45*Scale,-45*Scale+7] and [45*Scale-7,45*Scale]. Add shaft collars outside posts.
Those stock12 OD,8 ID,3 wide collars spanY[-55*Scale-3,-55*Scale] and[55*Scale,55*Scale+3].
Stock12 OD,8 ID thrust spacers spanY[-55,-45]*Scale and[45,55]*Scale.
No gear enclosure carries a bearing. The wheel is supported on both sides.

## Head adjustment and drive whorl

The head reference axis is X=-140*Scale,Z690*Scale, parallel to the wheel axle.
HeadPivot=(-172,0,555)*Scale. HeadTilt=asin((DriveTension-6)/122).
All upper-head construction is made in this reference pose, then rotated once about HeadPivot and axis -Y by HeadTilt.
State.Flyer is that transformed reference axis point. Default tension6 gives exactly690*Scale orifice height.
Both posts, upper mother-of-all, flyer/bobbin/whorl, guides, brake and controls share this tilt.
The base and fixed hinge leaves do not tilt. Never put the actual tilted State.Flyer into reference-pose construction.
The four front-whorl groove planes, local Y, are [-278,-270,-262,-254]*Scale for ratios [7,9,12,17].
HeadShift is minus the selected plane coordinate; add it to every head part's local Y.
This moves the head along its own axis so the selected groove aligns with the wheel's belt plane Y0.
Whorl selected pitch Rf=Rw/Ratio. Each groove root=(Rw/itsRatio)-sqrt(5), outer=root+6*Scale,
V lip planes at grooveY+-3*Scale. Whorl spans localY[-282,-250]*Scale; connect groove lips by axial/radial steps.
In increasing localY, continue each larger groove's outer radius to the next smaller groove's front
lip plane (nextCentreY-3*Scale), then step radially down at that plane. End lands are1*Scale wide.
This assigns total cylindrical-land lengths[3,2,2,1]*Scale to ratios[7,9,12,17].
Use stock D16.2 bore/flat6.1 and stock16 shaft flat6 over localY[-283,-249]*Scale.
FlyerAngle=Ratio*WheelAngle. Open band gives the same rotational sense.

Define alignment shift A=HeadShift-278*Scale, ranging -24*Scale..0.
The fixed main base is X[-195,-105],Y[-75,280]+A/Scale,Z[525,540], scaled.
A fixed front jack shelf extends X[-195,-30],Y[-85,-45]+A/Scale,Z[525,540], scaled.
Both bases translate in Y for groove selection, but do not tilt for tension.
Two stock6.6 alignment slots run along Y[-75,-25]+A/Scale at X=-180*Scale and X=-120*Scale.
StockM6 vertical clamps are fixed at globalY=-60*Scale,Z[515,540]*Scale, with stock12 OD,height3 heads.
The base overlaps both supporting wedges; slots allow all four groove selections before clamps are tightened.
Make main base and front shelf one L-shaped primary XY profile with those two slots before extrusion.
Drill stock6.6 vertical passages in the front supporting wedge for the two fixed clamp axes.

The tilting maiden bar is reference X[-167,-118],Y[-330,-20]+HeadShift/Scale,Z[555,570], scaled.
Its narrow width clears the lower drive span. Do not restore broad plates across that span.
A front jack arm is X[-172,-35],Y[-342,-318]+HeadShift/Scale,Z[555,570], scaled.
It overlaps the maiden bar. Its vertical stock8.6 screw passage is at X=-50*Scale,Y=-330*Scale+HeadShift.
An8 mm thick stock square M8 insert OD16 has a16.2 square pocket opening below that arm, at referenceZ555*Scale.
The square walls prevent rotation; use a centered stock8 thread envelope through the insert.
Let theta=HeadTilt and h=(15*Scale+122*Scale*sin(theta))/cos(theta).
The jack's reference axis is +Z at that passage, tip B=555*Scale-h, constant total length58*Scale.
Use a conical point, radius0 at axial0 and stock radius4 at axial4 stock, then a round8 mm shaft.
After the common head tilt the conical point touches the fixed shelf's Z540*Scale plane without cap penetration.
Its D-flat3 knob zone is axial[42,52]*Scale measured from B; knob clearance radius4.1 and flat3.1.
Knob outside diameter24*Scale, axial height10*Scale. Cut only that flat span, preserving the8 mm round thread envelope.
Both jack and knob rotate about their local axis by phase2*pi*(h-15*Scale)/1.25, the stock M8 lead.
The shaft length stays constant as its tip and keyed knob translate together along the upper board's normal.
The jack slides at its shelf contact as the upper board tilts. Model its thread and pocket envelope, not cut helices.

Two hinges share axis X=-172*Scale,Z555*Scale, parallel to Y, at reference starts Y=-300*Scale and -90*Scale.
For each start q, fixed knuckles span[q,q+15*Scale] and[q+32*Scale,q+47*Scale];
the moving knuckle spans[q+16*Scale,q+31*Scale]. All receive HeadShift.
Use stock5 bore,10 outside diameter; pin5 stock spans[q-Scale,q+48*Scale].
Fixed brackets X[-190,-175],Z[540,555], at the fixed knuckle Y spans join the base to the knuckles.
Moving leaves X[-169,-158],Z[555,558], at the moving knuckle span join the upper board.
The upper board stays5*Scale away from the pin axis, and leaves3*Scale away; at minimum scale the latter
is2.4 mm, so add a stock2.7 radius circular relief along Y through each moving leaf before placement.
This relief must leave the leaf joined to its moving knuckle but clear of the stock2.5 pin.
Only moving knuckles/leaves receive HeadTilt; all share the same axis and translation.
Pins and both sets of knuckles are separate named native bodies. This is a real revolute hinge, not an annotation.

Maiden feet span reference X=headX+-35*Scale, Y[postFront*Scale-12,postBack*Scale+2], Z[570,585]*Scale.
Each foot has two stock6.6 holes at X=headX+-10*Scale,Y=postFront*Scale-6; matching holes pass through the maiden bar.
StockM6 bolts span referenceZ[560,585]*Scale, with stock12 OD,height4.2 heads above585*Scale.
Feet and posts are rigid wooden joints. Rear support unbolts for bobbin removal.
Keep tall post geometry outside the bolt holes; do not send mounting bolts through bearing bores.

## Flyer, bobbin and bearings

Head local Y values below all receive HeadShift. X/Z offsets rotate with flyer where appropriate.
Front hollow shaft OD16 stock, localY[-335,-205]*Scale; axial bore10 stock opens at -335,
ends at -208*Scale with a solid end wall. Rear shaft OD8 stock extends -205..-26 scaled.
Construct one exact stepped revolved shaft. Cut stock6 radial yarn port towards local+Z,
centreY=-227*Scale, from bore interior to outside tube. This connects to the10 bore.
Add D flats X6 stock over Y[-222,-208]*Scale for crossbar and[-283,-249]*Scale for whorl.
Front maiden X=headX+-20*Scale,Y[-314,-286]*Scale,Z[585,715]*Scale, stock26 bore at axis.
Front plain nylon bush ID16.2,OD26,width20 stock, centred Y=-300*Scale. This is a sleeve,
not a purchased ball bearing. Reuse ballbearing.Build's custom annulus construction, identify group bush.
Rear maiden X=headX+-20*Scale,Y[-54,-30]*Scale,Z[585,715]*Scale, stock22 bore.
Rear stock608 spansY[-48*Scale,-48*Scale+7]. Add axial collars/retainers clear of rotating U arms.
Rear flyer stock12 OD,8 ID,3 wide front collar spansY[-54*Scale-3.2,-54*Scale-0.2].
Its stock12 OD,8 ID thrust spacer spansY[-54*Scale-0.2,-48*Scale].
The rear thrust spacer spansY[-48*Scale+7,-30*Scale].
Place the rear collar atY[-30*Scale+0.2,-30*Scale+3.2]. The rear shaft is at least flush with that collar.
Front sleeve retainer is stock20 OD,16 ID,2 wide, at Y[-314*Scale-2.2,-314*Scale-0.2].
Whorl axial retention uses two stock20 OD,16 ID,2 wide collars:
frontY[-282*Scale-2.2,-282*Scale-0.2], rearY[-250*Scale+0.2,-250*Scale+2.2].
Crossbar localX[-74,74],Z[-16,16],Y[-220,-210], scaled; stock D16.2 bore/flat6.1.
Two arms localX[-74,-56] and[56,74],Z[-10,10],Y[-220,-85], scaled.
Crossbar/arm overlaps are rigid joints. Arms end before the stationary bobbin brake plane.

Bobbin independently rotates on stock8 rear shaft with two608 bearings. One exact revolved wooden body:
front flange localY[-204,-194], radius45; barrelY[-194,-74],radius17;
rear flangeY[-74,-64],radius45, scaled. Barrel passage12 stock clears the11 OD inner-race spacer.
Each flange has stock22 bearing counterbore, depth10*Scale.
Front608 Y[-194*Scale-7,-194*Scale], rear608Y[-74*Scale,-74*Scale+7].
Rear flange V brake groove centreY=-69*Scale, root42*Scale, lip planes+-1.5*Scale, outer45*Scale.
Yarn pack annulus barrel radius17*Scale to31*Scale,Y[-190,-78]*Scale; optional visual winding marker.
The witness bar has local X width4,Z thickness4,Y length8 nominal mm, centred atY=-134*Scale,Z31*Scale,X0,
overlapping the pack by2*Scale. It is a colored winding marker, not a torque-coupling part.
BobbinAngle=(Ratio-YarnFeedPerTurn/(2*pi*31*Scale))*WheelAngle.
This prescribes an example feed for animation. It does not calculate friction, spring force, yarn strength or traction.
Flyer and bobbin must never be rigidly coupled. Relative turns wind yarn onto the bobbin.
Add bearing spacers and removable shaft collars. Bearing outer rings seat in bobbin; inner rings contact round shaft.
An11 OD,8 ID stock sleeve fills the120*Scale distance between the bearing inner faces.
Front thrust sleeve12 OD,8 ID spans from Y=-205*Scale+0.2 to-194*Scale-7-0.2.
Rear removable collar12 OD,8 ID has front faceY=-74*Scale+7+0.2 and stock3 width.
Collars are custom split-clamp retaining envelopes; clamp screws and threads are not manufacturing geometry.
Both0.2 mm gaps preserve rotation without several millimetres of bobbin end play.
Whorl and crossbar D couplings transmit torque; bore clearance is0.1 mm radial and0.1 mm at the flat.

## Yarn route

Model a thin illustrative leader using exact circular extrusions between waypoints, stock radius0.35.
Start on flyer axis at localY=-405*Scale; enter the open end at-335; continue to radial port at-227.
Exit local+Z to22*Scale above axis before turning toward the positive arm.
From the raised outlet run diagonally to the front guide atX58*Scale,Y=-224*Scale,Z22*Scale,
then along the arm toY=-160*Scale.
Open guide hooks centred X=+-58*Scale,Z22*Scale,Y=-190,-172,-160,-142,-124,-106 scaled;
stock centreline radius4,round wire radius1,300-degree wrap, axes along localY except active positive-arm Y=-160*Scale.
That active hook's axis bisects incoming +Y and its outgoing unit tangent-to-pack strand direction.
Its open mouth permits moving yarn between stations without rethreading closed rings.
For each hook choose radial e=normalize(project(local-Z into hook plane)), f=axis.Cross(e).
The round wire follows centre+4*(cos(t)*e+sin(t)*f), t from -150 to150 degrees;
the60-degree opening is opposite e, above the bottom attachment. This keeps the lower stem attached at t0.
The lower stem endpoint has the sameX/Y as centre+4*e at Z=10*Scale, on the arm's mounting face.
Add a front routing guide at X58*Scale,Y=-224*Scale,Z22*Scale.
Its incoming unit direction is normalize((58,3,0)), and its axis bisects that direction and +Y.
The routing hook is now well behind the front maiden and whorl; retain its clearance through a full rotation.
Attach its bottom via a diameter2 vertical stem to Z12*Scale and a horizontal stalk toY=-215*Scale,
inside the front crossbar. Both yarn strands clear the hoop walls.
Final strand goes from selected guide to a tangent point on yarn pack radius31*Scale, in the sameY=-160 plane.
For P=(58,22)*Scale and R=31*Scale, tangent point T=(R²/|P|²)P
minus (R*sqrt(|P|²-R²)/|P|²)*perp(P). Rotate all outward leader segments with flyer.
The axial incoming yarn remains on the rotation axis. Clear the crossbar, tube wall, arms and stock6 guide openings.

## Scotch brake

All following geometry is in the reference head pose and receives the common head tilt.
Stock brake cord radius0.6 wraps the upper semicircle at rear bobbin plane localY=-69*Scale,
pitch Rb=42*Scale+0.6*sqrt(5). Tails at X=headX+-Rb descend to Z600*Scale.
At that level they turn forward along -Y to leftY=-285*Scale+BrakeTravel/2 and rightY=-285*Scale-BrakeTravel/2.
Band centreline length pi*Rb+612*Scale stays constant. Do not confuse spring extension with cord stretch.
Two native six-turn Coil springs run along+Y, wire radius0.4 and centreline radius2.1 stock.
Left spring endpoints areY[-310*Scale,-285*Scale+BrakeTravel/2];
right endpointsY[-310*Scale-BrakeTravel,-285*Scale-BrakeTravel/2], at each tail's X,Z600*Scale.
Each spring has length25*Scale+BrakeTravel/2. Keep0.4 at each axial end for attachment links;
coil pitch=(SpringLength-0.8)/6. Add small radius0.4 axial/radial bridges joining coil ends to the endpoint axes.
The left front endpoint is fixed. The right front endpoint advances towards the operator by BrakeTravel.

A front control rail spans X[headX-55,headX+55], referenceY[-338,-325]+HeadShift/Scale,Z[590,610], scaled.
An upright X[headX-27,headX+22],Y[-335,-325]+HeadShift/Scale,Z[570,590] joins it to the jack arm.
At rear localY[-80,-60], a crossbar X[headX-55,headX+55],Z[565,580] joins the maiden bar above the hinge pin.
Two radius0.8 anchor stems reach the spring/cord datums from these timbers.
Rear routing eyes have stock ID3,OD5,width1.2, centred at each (tailX,-69*Scale,600*Scale).
Their axes bisect the incoming -Z and outgoing -Y directions; attach them with vertical stems to the rear crossbar.
They guide the cord's right-angle bends below the flyer. Keep cords out of the wood and eye walls.

Use stockM4 horizontal take-up screw axis+Y at the right tailX,Z600*Scale, through the front control rail.
Its reference Y range is[-350*Scale-BrakeTravel,-310*Scale-BrakeTravel-2].
End eye centreY=-310*Scale-BrakeTravel, stock OD5,ID2.6,width1.2,axisX.
Knob OD20*Scale,height5 stock occupiesY[-348*Scale-BrakeTravel,-348*Scale-BrakeTravel+5].
Flatten the stock4 screw only there at radial1.3; knob has D4.2 bore/flat1.4.
An8x8x4 stock square M4 insert fits an8.2 square pocket at Y[-334*Scale,-334*Scale+4],
centred on the screw axis and open from the front of its supporting rail. Draw the stock4.2 through-passage first.
Use thread envelopes, not cut threads. The keyed knob turns the screw; the square insert prevents nut rotation.
The front left spring has a fixed stem from the control rail rear face to its endpointY=-310*Scale.
No stationary cord, spring, routing eye or control may cross the rotating flyer-arm swept volume.

## Verification and images

First build default real assembly and VerifyAll tessellate every final body, then expand cases.
Verify all sketches, parameter extremes, each whorl groove selection, belt tangency/ratio,
constant brake length, independent bobbin motion, hollow orifice-to-port passage, bearing seats,
positive torque couplings, spatial linkage closure, gimbal-axis alignment, and swept clearances against the fixed frame.
Keep diagnostics bounded. Distinguish intended bearing/material joints from collisions.
Do not call an analytic planar gap check full 3D contact detection. Document friction and strength limits.
Render actual Decad bodies: hero, front/back pair, flyer/bobbin close-up, Scotch brake detail,
and animation with fixed frame and all moving groups correctly phased.
Show axis/orifice and leader clearly in close-up; show independent bobbin marker motion.
Register renderer integration checks and build entry point. Add images/index.png and hero.png.
