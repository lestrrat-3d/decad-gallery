# Treadle spinning wheel

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
State exposes Wheel,Flyer,Heel,CrankPin,Toe r3.Vec; PedalAngle,RodAngle,FlyerAngle,BobbinAngle,
WheelPitch,FlyerPitch,BeltLength,BrakeLength,SpringLength,HeadShift float64.
Wheel/Flyer positions are axis points in belt plane Y0; Heel/CrankPin/Toe use footman midplane Y=-105*Scale.
Assembly exposes Parts []Part, Sketches []*sketch.Sketch, State State.
Part exposes Name,Group,Motion string and Body *decad.Body.
Motion is fixed, wheel, flyer, bobbin, treadle or footman; it names the rigid motion for animations.
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

Wheel centre (220,0,430)*Scale; radius280*Scale; inner rim radius255*Scale.
Heel X80,Z60; crank radius25; pedal pin distance200; footman pin distance370, all scaled.
Crank pin X=220+25*cos(a), Z=430+25*sin(a), a=WheelAngle.
Choose the lower/to-the-right intersection of the heel's radius200 and crank pin's radius370 circles.
PedalAngle=atan2(toeZ-heelZ,toeX-heelX); RodAngle=atan2(crankZ-toeZ,crankX-toeX).
This gives a full revolution without changing assembly branch. The treadle rocks about its heel;
it is not a vertical slider. Wheel inertia carries the crank through dead centres and raises the treadle.
Start the wheel by hand in the wanted direction. No ratchet, return spring or one-way clutch is required.

Wheel rim spans Y[-10,10]*Scale, with V groove at Y0: radius280 at Y=+-3,
radius274 at Y0; groove slope2. Stock drive cord radius1, pitch radius Rw=274*Scale+sqrt(5).
Hub radius30*Scale, Y[-15,15]*Scale, stock D8.2 bore with flat at X3.1.
Eight spokes radial25..260, tangential width18, Y[-6,6], all scaled, rotate with wheel.
Spoke/hub/rim overlaps are named rigid wood joints, not moving interference.
Wheel axle stock diameter8, Y[-78,60]*Scale. D flats X3 over Y[-18,18] and [-78,-66], scaled axial limits.
Round bearing journals remain full diameter. Crank Y[-76,-68], outline local [-9,34]x[-9,9],
D8.2 root bore/flat3.1, stock5.2 pin hole at radius25*Scale. Rotate together with wheel.
Stock5 crank pin spans Y[-109*Scale-3.5,-68*Scale+3.5].
Footman outline local [-12,382]x[-12,12]*Scale, thickness8*Scale, Y[-109,-101]*Scale,
stock14 bearing seats at pin centres0 and370*Scale, stock605ZZ width5 centred at Y=-105*Scale.
Use stock5 toe pin Y[-230*Scale-3.5,-101*Scale+3.5] and end collars for both moving pins.
Treadle extends from heel X=-18..212 in its local axis, thickness24, Y[-230,-150], scaled.
Heel boss thickness36 tapers in steps to board thickness24 at local X18. Through hole stock22 at heel,
stock5.2 at toe200*Scale. Two stock608 bearings seat at Y=-228*Scale and Y=-152*Scale-7.
Heel stock8 pin Y[-253,-128]*Scale, fixed brackets Y[-246,-236] and [-144,-134]*Scale,
bracket X60..100,Z30..84, stock8.2 holes at heel. Add collars and bearing spacers.
Use stock12 OD,8 ID,3 wide heel collars just outside the brackets:
Y[-246*Scale-3,-246*Scale] and[-134*Scale,-134*Scale+3].
Stock11 OD,8 ID heel spacers spanY[-236,-228]*Scale and[-152,-144]*Scale;
the inner spacer spansY[-228*Scale+7,-152*Scale-7].
Stock9 OD,5 ID,3 wide linkage-pin collars use these exact intervals:
crank front[-109*Scale-3.2,-109*Scale-0.2], crank rear[-68*Scale+0.2,-68*Scale+3.2];
toe front[-230*Scale-3.2,-230*Scale-0.2], toe rear[-101*Scale+0.2,-101*Scale+3.2].
Stock8 OD,5 ID pin sleeves use these intervals:
crank front[-109*Scale-0.2,-105*Scale-2.7], crank rear[-105*Scale+2.7,-76*Scale-0.2];
toe front[-150*Scale+0.2,-105*Scale-2.7], toe rear[-105*Scale+2.7,-101*Scale-0.2].
The rear crank collar retains the pin behind the crank; the toe rear collar lies behind the footman.

## Open frame

All coordinates here are nominal, scaled. Use separate solid timbers and real bored bearing posts.
Bench X[-360,420],Y[60,140],Z[500,530]; it is behind the rotating wheel disk.
Left post X[-340,-300],Y[80,120],Z[30,500]. Left foot X[-370,-270],Y[-140,200],Z[0,30].
Head crossbeam X[-300,-220],Y[-100,100],Z[500,530] joins the bench to the forward head support.
Its right edge stays left of the descending drive-band span.
Wheel foot X[150,290],Y[-100,200],Z[0,30]. Bench support X[200,240],Y[100,140],Z[30,500].
Wheel upright pair X[200,240],Y[-55,-25] and [25,55],Z[30,455], stock22 bores at wheel centre.
Stock608 bearings span Y[-45*Scale,-45*Scale+7] and [45*Scale-7,45*Scale]. Add shaft collars outside posts.
Those stock12 OD,8 ID,3 wide collars spanY[-55*Scale-3,-55*Scale] and[55*Scale,55*Scale+3].
Stock12 OD,8 ID thrust spacers spanY[-55,-45]*Scale and[45,55]*Scale.
Heel foot X[35,125],Y[-270,-110],Z[0,30]; connecting rail X[80,220],Y[-140,-110],Z[0,25].
No gear enclosure carries a bearing. The wheel is supported on both sides.

## Head adjustment and drive whorl

Flyer centre X=(-240-DriveTension)*Scale,Z690*Scale. Its axis is parallel to the wheel axle.
The four whorl groove planes, local Y, are [-24,-16,-8,0]*Scale for ratios [7,9,12,17].
HeadShift is minus the selected plane coordinate; add it to every head part's local Y.
This moves the head along its own axis so the selected groove aligns with the wheel's belt plane Y0.
Whorl selected pitch Rf=Rw/Ratio. Each groove root=(Rw/itsRatio)-sqrt(5), outer=root+6*Scale,
V lip planes at grooveY+-3*Scale. Whorl spans localY[-28,4]*Scale; connect groove lips by axial/radial steps.
In increasing localY, continue each larger groove's outer radius to the next smaller groove's front
lip plane (nextCentreY-3*Scale), then step radially down at that plane. End lands are1*Scale wide.
This assigns total cylindrical-land lengths[3,2,2,1]*Scale to ratios[7,9,12,17].
Use stock D8.2 bore/flat3.1 and stock8 shaft flat3 over localY[-29,8]*Scale.
FlyerAngle=Ratio*WheelAngle. Open band gives the same rotational sense.

Fixed head support X[-265,-215],Y[-90,-50],Z[530,560]. Base X[-340,-140],Y[-290,-2],Z[560,575].
Sliding tension carriage X[-325,-155]-DriveTension,Y[-280,-12],Z[575,595].
Maiden bar X[-310,-170]-DriveTension,Y[-280,-30]+HeadShift/Scale,Z[595,610].
All head platforms stop before the belt planeY0; do not extend wood across the descending band.
Add side guide strips on the fixed base at Y=-280 and-12 to constrain the carriage to X motion.
Strips spanX[-340,-140]*Scale,Z[575,595]*Scale,
Y[-290*Scale,-280*Scale-0.2] and[-12*Scale+0.2,-2*Scale].
Add stock8 leadscrew along X at Y=-260*Scale,Z585*Scale, X[-380,-200]*Scale.
Fixed bearing block X[-355,-340],Y[-270,-250],Z[560,605], stock8.6 hole alongX.
Captive nut at X=(-250-DriveTension)*Scale,Y=-260*Scale,Z585*Scale,
thread envelope ID8, OD16, length8, axisX.
Carriage has stock8.6 clearance through-hole along X on the leadscrew axis and a captive16 OD nut pocket.
The nut pocket is an8 mm long cylinder centred at that X, open at the carriage's upper/lower surfaces
only where the stock nut requires it. Add a fixed anti-rotation flat to nut and pocket at radial7 stock.
Knob OD36*Scale, X[-380,-360]*Scale, matching D8.2/flat3.1; flatten screw in knob zone only.
Collars on both sides of fixed block retain screw axially. M8 lead1.25 is a thread envelope, not cut threads.
Use stock12 OD,8 ID,3 wide screw collars immediately outside the block's X faces.
Two stock6.6 rectangular slots in maiden bar at X=headX+-50*Scale,
localY[-140,-90]*Scale; stockM6 clamps at globalY=-100*Scale, Z[587,610]*Scale,
low-profile heads OD12 height3 stock. The low heads clear the flyer's lower swept envelope.
Posts attach with bolts through widened feet to the maiden bar; rear post can be unbolted for bobbin removal.
Each post has a foot X=headX+-40*Scale, its post Y limits, Z[610,625]*Scale.
Foot and upright are separate rigid wooden joints. Two vertical M6 bolts per foot at X=headX+-30*Scale
and the post's Y midpoint span Z[600,625]*Scale, with stock12 heads of4.2 stock height above625*Scale.
Stock6.6 clearance holes pass through feet and maiden bar, not through the tall bearing posts.

## Flyer, bobbin and bearings

Head local Y values below all receive HeadShift. X/Z offsets rotate with flyer where appropriate.
Front hollow shaft OD16 stock, localY[-280,-205]*Scale; axial bore10 stock opens at -280,
ends at -208*Scale with a solid end wall. Rear shaft OD8 stock extends -205..8 scaled.
Construct one exact stepped revolved shaft. Cut stock6 radial yarn port towards local+Z,
centreY=-227*Scale, from bore interior to outside tube. This connects to the10 bore.
Add D flat X6 stock over Y[-222,-208]*Scale for flyer crossbar; rear whorl flat X3 as above.
Front maiden X=headX+-20*Scale,Y[-259,-231]*Scale,Z[610,715]*Scale, stock26 bore at axis.
Front plain nylon bush ID16.2,OD26,width20 stock, centred Y=-245*Scale. This is a sleeve,
not a purchased ball bearing. Reuse ballbearing.Build's custom annulus construction, identify group bush.
Rear maiden X=headX+-20*Scale,Y[-54,-30]*Scale,Z[610,715]*Scale, stock22 bore.
Rear stock608 spansY[-48*Scale,-48*Scale+7]. Add axial collars/retainers clear of rotating U arms.
Rear flyer stock12 OD,8 ID,3 wide front collar spansY[-54*Scale-3.2,-54*Scale-0.2].
Its stock12 OD,8 ID thrust spacer spansY[-54*Scale-0.2,-48*Scale].
The rear thrust spacer spansY[-48*Scale+7,-28*Scale], between the608 inner race and whorl front face.
Place the rear collar beyond the whorl atY[4*Scale+0.1,4*Scale+3.1].
Do not put a3 mm collar into the2*Scale gap between rear post and whorl.
Front sleeve retainer is stock20 OD,16 ID,2 wide, at Y[-259*Scale-2.2,-259*Scale-0.2].
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
The witness bar is4x4x8 nominal mm, centred atY=-134*Scale and radial offset31*Scale,
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
Start on flyer axis at localY=-350*Scale; enter the open end at-280; continue to radial port at-227.
Exit local+Z to22*Scale above axis before turning toward the positive arm.
From the raised outlet run diagonally to the front guide atX58*Scale,Y=-224*Scale,Z22*Scale,
then along the arm toY=-160*Scale.
Guide rings centred X=+-58*Scale,Z22*Scale,Y=-190,-160,-130,-100 scaled;
stock ID6,OD10,width2, axes along localY except the active positive-arm guide atY=-160*Scale.
That active guide's axis bisects incoming +Y and its outgoing unit tangent-to-pack strand direction.
This lets both strands pass through the opening at45 degrees without crossing the hoop wall.
For each ring, project local-Z into its ring plane, normalize to e, and attach a diameter2 stem
to the point centre+4*e (4 stock). The lower stem endpoint has the sameX/Y at Z=10*Scale,
inside the arm. The endpoint at radius4 is inside the stock3..5 hoop material.
Add a front routing guide at X58*Scale,Y=-224*Scale,Z22*Scale.
Its incoming unit direction is normalize((58,3,0)), and its axis bisects that direction and +Y.
The7*Scale gap from the maiden's rear face clears the stock10 OD oblique hoop through a full rotation.
Attach its bottom via a diameter2 vertical stem to Z12*Scale and a horizontal stalk toY=-215*Scale,
inside the front crossbar. Both yarn strands clear the hoop walls.
Final strand goes from selected guide to a tangent point on yarn pack radius31*Scale, in the sameY=-160 plane.
For P=(58,22)*Scale and R=31*Scale, tangent point T=(R²/|P|²)P
minus (R*sqrt(|P|²-R²)/|P|²)*perp(P). Rotate all outward leader segments with flyer.
The axial incoming yarn remains on the rotation axis. Clear the crossbar, tube wall, arms and stock6 guide openings.

## Scotch brake

Stock brake cord radius0.6 in plane localY=-69*Scale. Pitch Rb=42*Scale+0.6*sqrt(5).
It wraps the upper semicircle of the rear bobbin groove. Vertical tails at X=headX+-Rb.
Right tail ends at Z640*Scale-BrakeTravel, left at Z635*Scale+BrakeTravel.
Spring lower anchor Z610*Scale, upper635*Scale+BrakeTravel. Spring length25*Scale+BrakeTravel.
Band centreline length pi*Rb+105*Scale stays constant as screw takes up spring extension.
Use native Coil for the extension spring: stock wire radius0.4, centreline radius2.1,
six turns between Z610*Scale+0.4 and635*Scale+BrakeTravel-0.4.
Connect spring ends to the wooden bar and cord with short eye/hook envelopes.
At both ends use stock0.4 radius vertical links from the coil end (X=leftTailX+2.1)
to the corresponding610*Scale or635*Scale+BrakeTravel datum, then horizontal0.4 radius bridges
to leftTailX. The lower bridge attaches to a stock1.6 diameter anchor at Z[606,610]*Scale.
Right adjusting stockM4 screw moves its band eye down by BrakeTravel through a captive nut in maiden bar.
Use thread envelopes, stock pitch0.7, knob OD20*Scale and positive keyed screw connection.
Let E=640*Scale-BrakeTravel. Rod diameter4 spansZ[590*Scale-BrakeTravel,E-4] with matching flat1.3
only in knob zone Z[E-10,E-5]. Knob height5 stock has D4.2 bore/flat1.4 and occupies that zone.
Band eye axisX, stock OD5,ID2.6,width1.2,centreZ=E-2.5. Its lower edge joins the rod at E-4.
Use an8x8x4 stock square M4 threaded insert in an8.2 square pocket opening from maiden-bar underside
Z595*Scale to595*Scale+4. The carriage closes the pocket; the square walls prevent insert rotation.
Cut stock4.2 rod clearance through both the maiden bar and carriage on the screw axis.
Do not bury the knob in the carriage or put a nut inside uncut wood.
No stationary brake tail or spring may cross the rotating flyer-arm swept volume.

## Verification and images

First build default real assembly and VerifyAll tessellate every final body, then expand cases.
Verify all sketches, parameter extremes, each whorl groove selection, belt tangency/ratio,
constant brake length, independent bobbin motion, hollow orifice-to-port passage, bearing seats,
positive torque couplings, four-bar closure, and swept clearances of moving members against fixed frame.
Keep diagnostics bounded. Distinguish intended bearing/material joints from collisions.
Do not call an analytic planar gap check full 3D contact detection. Document friction and strength limits.
Render actual Decad bodies: hero, front/back pair, flyer/bobbin close-up, Scotch brake detail,
and animation with fixed frame and all moving groups correctly phased.
Show axis/orifice and leader clearly in close-up; show independent bobbin marker motion.
Register renderer integration checks and build entry point. Add images/index.png and hero.png.
