# Spinning-wheel Decad construction

## Emission

Follow instructions.md as geometry authority. Emit only Go construction under the supplied output directory.
Do not read existing spinningwheel Go. Split by responsibility: motion, sections, drive, frame, head, brake, assembly.
Use repository Go/sketch conventions and contextual errors. Keep per-call state local.
Declared shared dependencies: models/ballbearing public API and internal/sketchcheck.VerifyFullyConstrained.
Read their contracts/helpers and dependency public source as needed, never copy another model construction.

## Geometry primitives and checks

- Create independent sketch.World/XY or XZ planes. Record every sketch in Assembly.Sketches.
- Constrain circle centres to anchored origin or with horizontal/vertical distances; add Radius/Diameter constraints.
- Use CreateRectangle's built-in geometric constraints plus origin offsets and width/height dimensions.
- For orthogonal stepped polygons, share endpoint objects, add H/V per edge, anchor first point,
  then independent segment dimensions, omitting closure-redundant final dimensions.
- For arbitrary axial profiles, share endpoints, constrain H/V where geometrically applicable,
  dimension independent remaining coordinates. Never Fix computed points.
- Solve and VerifyFullyConstrained before profile consumption. Select exactly one valid profile with required hole count.
- Extrude with Distance Along; revolve full sections with FullRevolution; cut/join with native Cut/Union.
- Use exact circle/arc profiles for D bores and native single combined cap tool for disjoint shaft flats.
- D bore construction: extrude D outline using an exact circular arc and straight chord,
  or cut full cylindrical bore tool by cap before cutting host. All arc/chord sketches require geometric constraints.
- Keep stock dimensions exact and return final live bodies only. Do not tessellate as a modeling fallback.
- Place XY extrusions so local u maps world X, v maps world Z, n maps -Y;
  angle t uses u=(cos(t),0,sin(t)),v=(-sin(t),0,cos(t)).
  Extrusion at localZ0..L maps globalY=yBack..yBack-L.
- For sections revolved around localZ, draw radial/axial profile on world.XZ; its world coordinates are (radius,0,axial).
  Axis ConstructionAxis{Origin:0,Dir:(0,0,1)}. Transform completed body to the intended shaft frame.
- Reuse ballbearing.Build for608/605 and custom plain sleeve, record its real Sketch.
- Reject nil ctx/doc and invalid Params before any geometry. Check cancellation. Require all final Body.IsSolid.

## Motion

Implement the exact formulas in instructions. Normalise angle for trig only; keep continuous FlyerAngle/BobbinAngle
for interpreting multiple turns. Pedal closure psi=atan2(dz,dx)-acos((d*d+200²-370²)/(2*d*200)),
where dx=crankX-80,dz=crankZ-60 in nominal units. Fail if cosine leaves[-1,1].
Toe=(80+200*cos(psi),-105,60+200*sin(psi))*Scale. Compute RodAngle from returned pin centres.
Wheel=(220,0,430)*Scale; Flyer=((-240-DriveTension)*Scale,0,690*Scale).
Selected groove determines HeadShift. Brake dimensions use actual BrakeTravel.

## Exact belts

Represent the one physical closed drive cord by four tangent solid spans with shared endpoint caps.
Do not use an unsupported closed sweep or sampled polyline belt.
Let u=(Cf-Cw)/d in XZ, v=(-uZ,uX), q=(Rw-Rf)/d,k=sqrt(1-q*q),nPlus=q*u+k*v,nMinus=q*u-k*v.
Wheel tangent points Aw+-=Cw+Rw*n+-, flyer Af+-=Cf+Rf*n+-.
Wheel wrap CCW from nPlus to nMinus through -u is 2*pi-2*acos(q).
Flyer wrap CCW from nMinus to nPlus through +u is 2*acos(q).
Connect AwMinus to AfMinus and AfPlus to AwPlus using exact circular extrusions in frames along each span.
Create each wrap by revolving a stock-radius circle at (pitchRadius,0) in an axis-containing plane
through AngleExtent{A:units.Radians(wrap),Dir:Along}. Then rotate/translate its starting radial ray
to atan2(nZ,nX), mapping the revolve normal to -Y. Check sense using endpoint coordinates.
Partial torus radial/axial sketch lies in the plane containing pulley axis; revolve is native exact torus.
BeltLength=Rw*wheelWrap+Rf*flyerWrap+2*sqrt(d*d-(Rw-Rf)²).
Scotch cord uses identical method for upper half wrap: angle0..pi in XZ plus two vertical circular extrusions.
Keep the radius offsets sqrt(5)*cordRadius so cords are tangent to both sloping V walls.

## Wheel and linkage

Revolve rim polygon in (radius,axial) coordinates:
(255s,0),(280s,0),(280s,7s),(274s,10s),(280s,13s),(280s,20s),(255s,20s).
Place local axial0 at globalY10s. Build hub D bore then eight rectangular spokes.
Keep wheel/hub/crank/axle as separate named rigidly coupled parts, Motion wheel.
Make full round axle, union the two flat cap tools, then perform one Cut.
Build crank, footman, treadle and support parts to instructions dimensions, drill holes before placing.
For treadle use local outline(-18,-18),(18,-18),(18,-12),(212,-12),(212,12),(18,12),(18,18),(-18,18),scaled.
Stock22 heel and5.2 toe holes. Footman bearings move with footman; toe pin moves with treadle;
crank pin moves with wheel. Heel pin/brackets/spacers fixed. Assign Motion accordingly.

## Frame and adjustment

Extrude each timber rectangle separately, with prescribed holes in posts/brackets before extrusion.
Use box profiles on perpendicular planes where slots/screw bores are required.
Mother-of-all has three independently named plates. Cut the two alignment slots through the maiden bar.
Guide strips lie outside carriage Y[-280,180] with0.2 stock running clearance.
Add leadscrew, knob, block, captive nut, collars and clamp fasteners with named thread-envelope bodies.
Nut travel follows carriage; screw stays axially fixed and knob rotates for selected tension.
Rear maiden is a bolted removable part. Build widened feet and four vertical M6 fastener envelopes
at the foot dimensions in instructions.md. Fasteners and clearance holes stop at the feet;
do not send bolt shanks through the bearing bores. Drill matching foot and bar holes.
Create carriage leadscrew passage and captive nut pocket before returning the carriage.
Use the specified nut/pocket anti-rotation flat, with0.1 stock clearance.
Construct the carriage's primary profile on YZ, extruded along +X from(-325-DriveTension)*Scale.
Draw its stock8.6 circle at local(Y=-260*Scale,Z585*Scale) directly in that primary rectangle sketch.
Consume the annular rectangular profile with one hole, rather than cutting a round bore into an existing box.
Then Cut the short D nut pocket once, followed by the distant vertical brake-rod clearance.
The brake-rod clearance tool starts at Z575*Scale-1 and ends at Z595*Scale+1;
overrun is stock1 mm past both walls. Do not make a later through-cutter's caps coplanar with host walls.
Construct the maiden bar's primary profile on XY, extruded +Z from595*Scale.
Draw both rectangular clamp slots, four circular foot-bolt holes and circular brake-rod clearance in this
single primary sketch. Require exactly one outer profile with seven holes before extrusion.
Only the square insert pocket needs a later blind Cut; do not sequentially cut all seven through features.
That square pocket tool starts at Z595*Scale-1 and ends at Z595*Scale+4,
overrunning the open underside while preserving the exact stock4 pocket depth.
Construct the fixed leadscrew block on YZ too, with its8.6 circle in the primary sketch, extruded along+X.
These feature orders preserve exact primary prisms before blind pockets and avoid a chain of mesh-backed cuts.
Use the forward head crossbeam and shortened negative-Y support platforms from instructions.md.
The descending band must pass behind those platforms with at least0.5 mm stock clearance at minimum scale.
Clamp slots are localY[-140,-90]*Scale and clamp centres globalY=-100*Scale;
use the specified3 mm low heads. Leadscrew/block/nut/knob share Y=-260*Scale.

## Head

Construct all shaft features about local+Z before one final placement to the head frame.
Revolve one stepped shaft from its axial/radial outline: local axial0 is headY8s,
local axial213s is headY-205s, local216s is the blind-bore end at headY-208s,
and local288s is the front orifice at headY-280s.
Cut the stock6 radial outlet with a cylinder from local(0,0,235s) to(0,10,235s).
Local+Y maps to head-local+Z; do not cut a world-placed shaft at each displayed angle.
Cut the two axial D flat cap tools in one combined tool. The front flat only spans[-222,-208]s;
the rear flat spans[-29,8]s. Do not cut the plain front or rear bearing journals.
In the shaft's unplaced coordinates, the front cap tool occupiesX[6,9],Y[-9,9],Z[216s,230s];
the rear cap tool occupiesX[3,5],Y[-5,5],Z[0,37s]. Union both tools before one Cut.
After both cuts, place the completed shaft with shaftFrame(Flyer.X,8s+HeadShift,Flyer.Z,FlyerAngle modulo2*pi).
Build crossbar D16.2 hole and arms as separately named rigid joints; Motion flyer.
Build rear whorl exact stepped full-revolve profile traversing four V grooves in axial order,
close down to axis radius0, then Cut with the exact D8.2/flat3.1 extruded tool.
Use the land/shoulder ownership in instructions.md. In a local+Z revolve with axial0 at headY4s,
and final normal-Y, traverse groove ratios[17,12,9,7] with centres(4+8*i)*s.
First land is axial0..s. Between successive grooves, step up to the next larger outer radius
at axial(8*i-1)*s for i=1,2,3, then continue that larger land to(8*i+1)*s.
Each groove uses two cones over centre+-3s. Last land is axial31s..32s.
Draw the D tool at localZ0 and extrude exactly32s so both tool caps match the host caps.
Do not add overrun to this exact coaxial through-tool. Cut before the common final placement or use
matching shaft frames whose endpoints denote the same two cap planes.
Do not start with a round bore; Cut cannot add the flat's required material inside that round hole.
Motion flyer includes shaft, crossbar, arms, whorl, guide eyes/stems and outward yarn leader.
Incoming yarn on axis may be fixed.

Revolve bobbin using this exact radial/axial section, local axial measured from globalY=-64s towards front:
(11,0),(45s,0),(45s,3.5s),(42s,5s),(45s,6.5s),(45s,10s),
(17s,10s),(17s,130s),(45s,130s),(45s,140s),(11,140s),(11,130s),(6,130s),(6,10s),(11,10s).
This produces stock22 stepped seats and stock12 barrel passage. Assign Motion bobbin.
Pack is exact annulus radius17s..31s over localY[-190,-78]s; Motion bobbin.
Add a small contrasting native solid witness bar attached to pack at radius31s, not intersecting flyer arms.
608 envelopes at specified seat depths. Assign fixed for shaft inner/outer aggregate envelope visualisation;
bearings are mounting envelopes, not separate race kinematics. They need not visually rotate.
Build the inner-race sleeve, front thrust sleeve and rear collar at the explicit Y limits in instructions.md.
The inner sleeve11 OD clears the12 bore. Collars must not bridge bobbin to flyer;
leave0.2 stock axial free clearance at each rotating shoulder.

## Yarn and brake

For all yarn geometry compute angle=math.Remainder(State.FlyerAngle,2*math.Pi) before trig,
matching the shaft/crossbar/arm phase even for very large finite accumulated angles.
Build rings from two concentric circles, oriented by the guide axes in instructions.md.
For the active positive-arm guide use n=normalize((0,1,0)+normalize((Tx-Px,0,Tz-Pz))).
For the front routing guide atY=-224s use n=normalize(normalize((58,3,0))+(0,1,0)).
Its incoming leader segment runs directly from the raised outlet(0,-227s,22s) to(58s,-224s,22s).
Other rings have n=(0,1,0).
Use arbitrary orthonormal placement frames; do not leave rings in localX planes.
Use e=normalize((0,0,-1)-n*((0,0,-1).Dot(n))) for each bottom attachment point centre+4*e.
Build vertical stems from arm top to that point. The front guide also has the horizontal crossbar stalk.
Use exact circular cylinder spans for specified leader waypoints and tangent final strand.
Leader path through radial outlet is clear; clip/space adjoining segments to shared endpoints only.
Represent the Scotch brake upper wrap and tails separately as one named band group.
Create native Coil from constrained offset circle of radius0.4 at radial2.1 on an axis-containing sketch;
pitch=(25s+BrakeTravel-0.8)/6, turns units dimensionless6, coil axis+Z.
Place spring at left tailX,brakeplaneY,Z610s+0.4. Add native short attachment eyes/stems to endpoints.
Build stockM4 adjuster rod/knob/nut with keyed knob connection and band eye on right.
Use E and the insert/pocket dimensions in instructions.md. Drill the carriage clearance and bar clearance.
Insert fits a square pocket open from below; do not represent it as a freely rotating round nut.
Knob lies above the bar and outside the screw eye. Spring length remains25s+BrakeTravel.
Right-tail datum is640s-BrakeTravel and total brake band length is pi*Rb+105s.
Exclude spring/cord/adjuster from flyer Motion. Return the state lengths for checking band conservation.

## Checks before return

Return contextual errors including part name and operation. No hard-coded count masking missing members.
Return unique named solid live parts plus all real sketches. Build must be callable concurrently on independent docs.
Do not emit tests or images; the parent verifies geometry and renders the real output.

## Sibling construction API

For partial re-emission, use this declared internal API without reading existing target Go.
Responsibility ownership:
- drive.go owns wheel rim/hub/spokes/axle/crank, footman/treadle, heel brackets/pin/bearings/collars/spacers,
  linkage pins/collars/sleeves and both footman605 bearings, plus the drive band.
- frame.go frame() owns open-frame timbers, two wheel uprights, two wheel608 envelopes,
  wheel shaft collars and wheel thrust spacers.
- frame.go adjustment() owns head base's adjustment plates/guides, leadscrew/block/nut/knob/collars,
  maiden bar/clamp fasteners and both maiden feet with their four mounting bolts/heads.
- head.go owns both tall maiden posts, front sleeve/rear bearing, flyer/bobbin/whorl,
  head shaft/bobbin retainers/spacers and winding marker. It does not create feet or mounting bolts.
- yarn.go owns leader, guides and stems. brake.go owns Scotch cord/spring/anchors and brake adjuster.
- sections.go owns point2 and hole types and listed geometry helpers except headPoint and place.
- placement.go owns builder.place, the native rigid placement wrapper used for the spring.
- assembly.go owns Part, Assembly, builder and its part/fail/Build functions. head.go owns headPoint.
Types: point2{x,y float64}, hole{x,y,r float64}; builder{ctx context.Context,doc *decad.Document,
p Params,a Assembly,err error,name string}. State/Assembly/Part follow instructions.md.
Methods on *builder:
- part(name,group,motion string,create func()*decad.Body) records context, checks error and final solid.
- fail(operation string,err error) stores first contextual error.
- section(axial bool)*sketch.Sketch creates and records XY or XZ sketch.
- profile(s *sketch.Sketch,holes int)*sketch.Profile solves and verifies before unique selection.
- place(body *decad.Body,f r3.Frame)*decad.Body returns nil on existing builder error;
  otherwise uses r3.FromFrame then decad.Transform(ctx,body,transform), checking contextual errors after each.
- extrude(s *sketch.Sketch,holes int,length float64,f r3.Frame)*decad.Body verifies then extrudes and places.
- panel(x0,z0,x1,z1,length float64,f r3.Frame,holes ...hole)*decad.Body creates rectangular prism with round holes.
- box(x0,x1,y0,y1,z0,z1 float64)*decad.Body creates world-axis-aligned box.
- cylinder(radius,length float64,f r3.Frame)*decad.Body; annulus(inner,outer,length float64,f r3.Frame)*decad.Body.
- dTool(radius,flat,length float64,f r3.Frame)*decad.Body; dAnnulus(inner,flat,outer,length float64,f r3.Frame)*decad.Body.
- cut(host,tool *decad.Body)*decad.Body; union(parts ...*decad.Body)*decad.Body.
- span(a,c r3.Vec,radius float64)*decad.Body constructs exact cylinder between points.
- headPoint(x,y,z,angle float64)r3.Vec rotates head-local XZ by angle, adds Flyer.X/Z and adds HeadShift toY.
- revolve(xy []point2,f r3.Frame)*decad.Body makes full native axial-section revolve then places.
- torus(radius,wire,start,angle float64,centre r3.Vec)*decad.Body makes exact partial wrap in worldXZ.
- flatShaft(body *decad.Body,caps ...*decad.Body)*decad.Body unions cap tools and performs one Cut.
Functions: rectangle(s,x0,y0,x1,y1); circle(s,x,y,r); dOutline(s,r,flat).
Functions: polygon(s *sketch.Sketch,xy []point2); scaled(s float64,xy ...point2)[]point2.
Annulus helper always delegates to ballbearing.Build with the supplied custom dimensions and placement,
and records its actual sketch. Do not duplicate annulus construction or emit unused helper aliases.
For D outlines use centre=Origin, endpoints(flat,+sqrt(r²-flat²)) and(flat,-sqrt(r²-flat²)),
CCW major arc from upper endpoint to lower and closing vertical chord. Constrain arc radius,
upper endpoint horizontal distance from origin and chord verticality; arc endpoint constraints supply the other relations.
Functions: frame(origin,u,v r3.Vec)r3.Frame; shaftFrame(x,y,z,angle float64)r3.Frame maps normal to-Y;
axisFrame(origin,axis r3.Vec)r3.Frame maps normal to axis, using U=normalize(ref.Cross(axis)),
V=normal.Cross(U), ref=+Z unless |axis.Z|>0.9, then ref=+X.
Frame and adjustment functions are methods frame() and adjustment(); motion.go owns Params,State and Motion.
No helper silently fixes geometry; every feature section must still satisfy sketch constraints.
