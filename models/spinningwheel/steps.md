# Spinning-wheel Decad construction

## Emission

Follow instructions.md as geometry authority. Emit only Go construction under the supplied output directory.
Read no existing target implementation. Read the declared sibling helper source and upstream public APIs as needed.
Compile all dimensions, placements, feature order, motions and checks from the contract; report missing facts.
Use contextual errors and per-call state. Never repair emitted geometry by hand.

## Common geometry API

Retain the supplied sections.go and placement.go implementations unchanged. They own these declarations:
- point2{x,y float64}, hole{x,y,r float64}.
- builder.section(axial bool)*sketch.Sketch creates and records XY or XZ sketches.
- builder.profile(s,holes) solves, fully verifies and selects one valid profile with the specified hole count.
- builder.extrude(s,holes,length,frame) verifies, extrudes along local+Z and places the body.
- builder.panel(x0,z0,x1,z1,length,frame,holes...) creates a rectangular prism with primary circular holes.
- builder.box(x0,x1,y0,y1,z0,z1) creates a world-axis-aligned box.
- builder.cylinder(radius,length,frame), annulus(inner,outer,length,frame).
- builder.dTool(radius,flat,length,frame), dAnnulus(inner,flat,outer,length,frame).
- builder.cut(host,tool), union(parts...), flatShaft(body,caps...).
- builder.span(start,end,radius) makes an exact circular cylinder between endpoints.
- builder.revolve(profile []point2,frame) revolves an XZ radial/axial section about local+Z, then places it.
- builder.torus(radius,wire,start,angle,centre) makes a partial torus in world XZ, normal-Y.
- builder.place(body,frame) makes one checked native rigid transform.
- rectangle(s,x0,y0,x1,y1), circle(s,x,y,r), polygon(s,[]point2), scaled(scale,...point2), dOutline(s,r,flat).
- frame(origin,u,v) returns an orthonormal r3.Frame.
- shaftFrame(x,y,z,t) has U=(cos(t),0,sin(t)),V=(-sin(t),0,cos(t)),N=-Y.
- axisFrame(origin,axis) maps its normal to axis.

All body-returning helpers return *decad.Body. All listed builder helpers use the builder's context and document.
Do not make a second annulus constructor. annulus delegates to models/ballbearing.Build and records its real sketch.
Reuse internal/sketchcheck.VerifyFullyConstrained. Every consumed sketch must report FullyConstrained,
DOF0, no redundant constraints and no conflicts before its profile is used.
Constrain geometry through shared endpoints, H/V, distances and radii. Never Fix computed polygon points.
Primary planar profiles may contain required circle/rectangle holes. Avoid sequentially cutting every through-hole.
Use a single combined flat-cut tool for all shaft D zones, and add no tolerant/faceted replacement geometry.

## Ownership and assembly

Stable new linkage IDs: treadle (footboard), treadle-heel-rail, treadle-actuating-rail,
treadle-rear-crossarm, treadle-connector-block,
heel-pin, heel-bearing-1/2, gimbal-crank-605, gimbal-bridge, gimbal-cheek-1/2,
gimbal-cross-pin, footman (long timber), footman-head-boss, footman-cross-605-1/2,
treadle-connector-lower/upper. Other names must be descriptive and unique.
Keep the existing stable head IDs from instructions.md. Use role groups bearing, pin, gimbal, footman,
treadle, collar, spacer, frame, adjustment, whorl, flyer, yarn, brake-band, spring and related named roles.
The upper main605 and both cross605 are the only three purchased605 envelopes; retain all seven608 envelopes.
head.go owns headFrame,headRing,headDirection. yarn.go owns yarnHook.
Partial emitters must call the declared sibling helpers without redeclaring them.
Signatures: builder.headFrame(y,angle float64)r3.Frame;
builder.headRing(name,group,motion string,inner,outer,y0,y1,angle float64);
headDirection(v r3.Vec,angle float64)r3.Vec.
Frame IDs: head-leg,front-wheel-leg,rear-wheel-leg,front-wheel-bearing-cap,rear-wheel-bearing-cap,
front-bench-rail,rear-bench-rail. Hinge parts use Group hinge and prefix mother-hinge-<1 or2>-.

- motion.go owns Params, State, DefaultParams, validation, Motion and PoseTransform.
- assembly.go owns Part, Assembly, builder and Build/part/fail. builder fields are
  ctx context.Context, doc *decad.Document, p Params, a Assembly, err error, name string, tilt bool.
- drive.go owns wheel parts, wheel crank/pin, gimbal, rod, compliant connector, treadle and heel shaft hardware.
  It also owns the four exact drive-band spans. It does not create the frame's heel608 bearings.
- frame.go frame() owns legs, rails, wedges, wheel bearing caps, wheel608, heel608 and wheel axle collars/spacers.
- frame.go adjustment() owns fixed head base, alignment clamps, hinges, tilting maiden bar,
  jack arm, vertical M8 jack/knob/insert and maiden feet/bolts.
- head.go owns headPoint and all flyer/shaft/whorl/bobbin/post/bearing/retainer geometry.
- yarn.go owns the leader, thirteen open hooks, stems and front stalk.
- brake.go owns both springs, the band, routing eyes/supports, front control rail and M4 hardware.

Build first validates context/document/parameters without mutation. Compute physical State.
Call frame(), drive(), adjustment(), then set tilt=true and call head(), yarn(), brake(). Reset tilt afterwards.
adjustment() must set tilt=false for base, fixed hinge leaves/knuckles/pins and alignment clamps;
set tilt=true for the upper bar, jack/insert/knob, moving leaves and maiden feet/bolts. Restore false before return.
part(name,group,motion string,create func()*decad.Body) records current name and checks construction errors and solidity.
When tilt=true, transform the completed body about State.HeadPivot, axis-Y, angle State.HeadTilt.
Perform this common tilt after all feature cuts, once per final part. Keep only live final bodies.
part records unique named Parts; no hard-coded body/sketch counts may mask missing construction.

All head methods construct at reference headX=-140*Scale,Z690*Scale, adding HeadShift to head-local Y.
headPoint(x,y,z,t) rotates local XZ about that reference axis by the reduced flyer angle and adds HeadShift to Y.
headFrame must reduce its passed angle with math.Remainder(angle,2*pi) before calling shaftFrame.
Do not use the already tilted State.Flyer for reference construction; part applies the one common tilt.
Native body features stay local before placement, especially hollow-shaft cuts and whorl through-tools.

## Motion and rigid placements

Implement instructions' three-dimensional closure with a bounded bisection. Require a finite solution and tight residual.
PedalFrame, RodFrame and GimbalFrame must be orthonormal, with the origins and axes in the contract.
Keep continuous accumulated flyer/bobbin phases; reduce only angles used for trigonometry.
Reject phase multiplication overflow before construction.
Use lowercase error-message openings, even when an exported parameter name appears later in the message.
Compute squares with multiplication rather than math.Pow(value,2), as required by repository Go lint.

Expose PoseTransform(base,state State,motion string,wheelDelta float64)(r3.Transform,error).
For fixed return an identity transform. For wheel rotate about base.Wheel/axis-Y by wheelDelta.
For flyer/bobbin rotate about base.Flyer/axis-Y by the corresponding phase difference.
For treadle map base.PedalFrame to state.PedalFrame; for gimbal and footman map their corresponding frames.
Use r3.FromFrame(base).Inverse followed by r3.FromFrame(target), with the actual public Xform APIs.
Read the dependency declaration before choosing the inverse/composition method signature.
The renderer and geometry sweep both consume this same rigid placement API; tests independently verify joint locations.

## Drive, articulated linkage and frame

Build the six-spoke wheel by native revolve, constrained hub D outline and six rectangular spokes.
Place the crank behind the wheel, with journals round and only contract-defined shaft zones flattened.
Keep the crank's front face at Y78*Scale so its stock collar clears the splayed rear leg at every scale.
Use GimbalFrame for bridge/fork geometry, RodFrame for rod/boss, and PedalFrame for board/actuating rail.
Drill the 605 seats and cross-pin clearances in the appropriate primary section planes before placement.
Fork cheeks have their stock5.2 holes along GimbalFrame.V. Upper rod boss has stock14 through-seat along RodFrame.V.
RodFrame.V equals GimbalFrame.V: plain605 cross bearings have no angular misalignment.
The first605 stays coaxial with the wheel crank pin. Never rotate its bore away from the pin axis.
The lower compliant connector has separate ideal bend legs following the treadle and rod, joined at D.
Model its actual clamp passages and pins; label it as compliant, without a claimed stiffness.

Build heel rail from a YZ rectangle/D-hole section extruded along local+X; rotate all treadle parts with PedalFrame.
For the diagonal rail, use a constrained rectangle in an orthonormal frame along its local centreline.
Join board, rail and diagonal by intended rigid wood overlaps; return separate named final parts.
Return the rear crossarm and connector block as separate timbers too; use no Union for these wood joints.
Make the connector block with a primary YZ rectangle and stock4.4 circle at local(Y,Z)=(450*Scale,8*Scale),
extruded along localX[208,232]*Scale. The lower clamp tool cuts only the compliant leg, not the other timbers.
The two heel608 envelopes belong to the two frame legs. They stay fixed while the D shaft and rail rotate.
Keep shaft journals, bearing clearances and axial collars consistent with the stock widths.
The single head leg and splayed wheel legs are YZ primary profiles extruded along X;
include heel seats in those profiles. Wheel bearing caps are XZ primary profiles extruded along Y.
Sloping rails and matching wedges are constrained XZ quadrilaterals extruded along Y.
Cut the front wedge's two vertical clamp passages with one combined stock6.6 tool.

## Hinged head and front whorl

Build the fixed L base on XY with alignment slots in its primary profile.
Build the narrow upper bar with four stock6.6 maiden-foot holes in its primary profile.
Build the jack arm with a stock8.6 primary screw passage, followed by its blind square insert pocket.
Build the insert as a square annulus. Keep jack flats confined to the keyed knob zone.
Build the jack locally as a full line-meridian revolve of (0,0),(4,4),(4,58*Scale),(0,58*Scale).
Cut the stock flat locally over axial[42,52]*Scale before phase rotation, reference placement and common head tilt.
Knob and shaft share the contract's phase and axial translation. Never change shaft length to simulate screw travel.
Use the contract's exact shelf-contact equation so jack end and fixed base meet for all tension settings.
Build hinge annuli/pins and leaves at the specified alternating axial intervals; never intersect adjacent knuckles.
Make rear maiden removable; foot bolts terminate at the feet, not inside bearing seats.
Construct each M6 bolt plus button head as one stepped full revolution, not a Union of touching cylinders.
For a shank from z0 to z1 with stock head height H, use local meridian
(0,0),(3,0),(3,z1-z0),(6,z1-z0),(6,z1-z0+H),(0,z1-z0+H), then place on its vertical axis.
Use H3 stock for alignment clamps and H4.2 stock for maiden-foot bolts. Both radii are stock, not scaled.

For the stepped shaft use unplaced local axial0 at referenceY=-26*Scale, normal-Y.
Its radius4 section extends to axial179*Scale, radius8 to309*Scale.
The radius5 axial orifice is open at309*Scale and ends at182*Scale.
Cut the stock6 radial port at axial201*Scale toward local+Y before final placement.
Cut flat caps X[6,9],Y[-9,9], axial[182,196]*Scale and[223,257]*Scale in one combined tool.
Place the completed body at shaftFrame(-140*Scale,-26*Scale+HeadShift,690*Scale,FlyerAngle).

For the one-piece whorl, local axial0 is referenceY=-250*Scale with normal-Y.
Traverse groove ratios[17,12,9,7] at axial centres(4+8*i)*Scale, keeping instructions' land ownership.
Each V uses native conical walls. End lands and shoulders match the instructions; no bonded carrier.
Revolve a solid-centre profile, then cut the exact D16.2/flat6.1 tool from axial0 to32*Scale.
Tool and host caps must match exactly. No overrun or prior round bore on this exact axial native route.
Keep eight native Cone faces. Place only after the exact cut, then let part apply the common head tilt.
Build the bobbin/shaft spacer and bearing intervals from instructions, preserving independent rotation and0.2 shoulder gaps.

## Hooks, cords and brake

The thirteen hooks are partial circular tori, round wire1, centreline radius4, wrap300 degrees.
Construct each in a canonical plane with native torus helper, then rigidly orient its radial basis to e,f in instructions.
Keep the opening opposite the bottom attachment. Stems meet the t0 point of the actual wire.
Use the instruction-defined oblique axes at the front and active hooks; actual yarn strands pass through their openings.
Do not keep closed annular guide bodies or change only their captions.

Drive band: u=normalize(Flyer-Wheel),v=(-u.Z,0,u.X),q=(Rw-Rf)/d,k=sqrt(1-q*q).
nPlus=q*u+k*v,nMinus=q*u-k*v. Tangencies are axis+pitch*n.
Wheel wrap is2*pi-2*acos(q), from nPlus through the side away from flyer to nMinus.
Flyer wrap is2*acos(q), from nMinus through the side away from wheel to nPlus.
Use exact native partial tori plus straight circular spans, sharing endpoint caps.
BandLength=Rw*wheelWrap+Rf*flyerWrap+2*sqrt(d*d-(Rw-Rf)^2).

For Scotch springs create native Coil about local+Z from a fully constrained offset circle;
then place so local+Z maps+Y, local+X maps+X, local+Y maps-Z.
Six full turns put both end points at radial+2.1*X. Bridge those endpoints to the corresponding centreline datums.
Use both springs and the two complementary tail lengths; the contract's conserved length excludes spring wire length.
Rear routing eyes bisect the cord directions. Their stems/posts join the rear support bar.
The front rail contains the M4 through-passage in its primary XZ profile.
Cut the square insert pocket from its front face minus1 toY=-334*Scale+4;
insert occupies exactly[-334*Scale,-334*Scale+4]. Add a1 mm front retaining plate,
11x11 stock around the screw axis, with stock4.4 passage, to close the otherwise open nut pocket.
The knob and end eye follow the screw's axial travel and keyed rotational phase from pitch0.7.
Fixed left spring stem ends atY=-310*Scale. Both spring fronts clear the wood for all travel settings.

## Verification boundary

Emit no tests or substitute drawings. Parent first builds and VerifyAll tessellates every real default solid,
then runs parameter branches, physical joint/clearance checks and renderer integration.
Return all real consumed sketches for independent constraint verification.
If a native geometry operation fails, report the smallest real producer case; do not relax tolerances or change dimensions.
