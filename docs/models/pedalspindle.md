# Pedal spindle construction and verification

The [model page](../../models/pedalspindle/) describes operation and shows the assembly.
[instructions.md](../../models/pedalspindle/instructions.md) owns the dimensions and API;
[steps.md](../../models/pedalspindle/steps.md) maps those requirements to Decad operations.

## API and build

The package `models/pedalspindle` exports `DefaultParams`, `Params.Validate`, `Motion`, and
`Build(ctx, doc, params)`. Build returns an `Assembly` containing named, separate `Parts`,
the package's verified `Sketches`, and the calculated linkage `State`.
Build uses the supplied document. Geometry remains separate at part interfaces.

The renderer colors the input blue, the compound intermediate gears gold and the output orange.
Its paired views show both sides of the assembly and drive. Each gear-pair sheet isolates the actual
placed bodies without exploding or separating their teeth. The four-pose sheet copies the actual
crank, treadle, footman and flywheel into poses calculated by `Motion`; the gear train is hidden there.
The `animation` shot writes a 48-frame looping GIF with one constant camera per view.
The left view includes the fixed grey frame; the right exposes the linkage at an oblique angle.
Both views mark the fixed A/B centers.
Moving C/D centers follow the actual crank and toe pins. Only the joint annotations and reference line are drawn in 2D.
Drive-only views omit bearing envelopes together with the supports and linkage removed from that view.

`Scale` accepts finite values from 0.8 through 1.2; `CrankAngle` accepts any finite radians.
Angles are reduced modulo one turn before calculating a pose. Returned angles describe the pose,
not an accumulated revolution count. Motion calculates linkage positions without creating bodies.

Run the focused construction and geometry checks from the repository root:

```sh
go test ./models/pedalspindle -timeout 5m
```

Build and render all documented views from the actual bodies:

```sh
go -C tools/render run . -model pedalspindle -shot readme -out ../../models/pedalspindle/images
```

Render the animation alone with:

```sh
go -C tools/render run . -model pedalspindle -shot animation -out ../../models/pedalspindle/images/linkage.gif
```

Run renderer integration checks:

```sh
go -C tools/render test -run '^TestPedalSpindle' -timeout 5m
```

## Parts and prototype materials

| Component | Quantity | Intended prototype construction |
| --- | --- | --- |
| 51-tooth Module 3 gears | 2 | Import `spurgear.Build`; machine or print the generated profiles for fit trials. |
| 17-tooth Module 3 pinions | 2 | Import the same construction; the default pressure angle is 20 degrees. |
| Input and intermediate shafts | 2 | 8 mm steel stock, cut to the contract lengths. |
| Quill spindle | 1 | Turn and polish 5 mm steel stock with the specified conical end. |
| 608-2RS bearings | 6 | Two each on the 8 mm input, intermediate and heel shafts, using 8 × 22 × 7 mm envelopes. |
| 605ZZ bearings | 4 | Two for the 5 mm quill and one at each footman end, using 5 × 14 × 5 mm envelopes. |
| Gear clamping hubs | 4 | Use commercial hubs; the modeled rings describe space claims only. |
| Flywheel | 1 assembly | Use a steel rim at 7800 kg/m³; nominal rim mass is about 2.56 kg. Detail its spoke joints. |
| Crank and pins | 1 crank, 3 pins | Use an 8 mm heel shaft and 5 mm moving pins; bearing seats are 22 mm at heel and 14 mm at footman ends. |
| Footman and treadle | 1 each | Use 320 mm and 200 mm pivot spacing respectively. |
| Base, two legs, bearing board, two heel brackets | 6 | Cut plate or timber to the contract; detail bolts and joints before fabrication. |
| Heel collars and spacers | 4 | Two external shaft-clamping collars and two inner-ring spacers limit axial movement. |
| Gear cover | 5 panels | Fit the front and four sides with removable fasteners. |
| Shaft retention and frame fasteners | To be detailed | Select collars, circlips and bolts after the material and fastening trial. |

The base, linkage and gears have no material assignment in the CAD model.
Frame scaling changes the frame, gear module/thickness, axial lengths and linkage spacing.
Purchased bearing sizes, shaft diameters, bearing seats and mating bores keep their stock dimensions.
The construction imports `models/ballbearing` and records all ten returned bearing sketches for verification.
The 48-part assembly contains 44 recorded sketches, including those imported bearing sections.

The frame holds the shaft bearings' outer rings; their inner rings turn with the shafts.
At the fixed heel pin, the treadle moves with the outer bearing rings. At the footman ends, the rod
moves with the outer rings while the inner rings follow the crank/toe pins. CAD envelopes do not distinguish
these internal rings. Bearing IDs and housing bores are nominal; final fit tolerances and retention depend
on the fabrication process and still need detailing.

The round gear bores intentionally clear the shafts. A commercial hub must attach both to the shaft
and to the gear; merely sliding the modeled ring onto the shaft does not transmit torque.
Hub attachment holes and clamp details depend on the purchased hub and must be added before fabrication.
Flywheel spokes overlap the hub and rim as assembled envelopes; their joints need the same treatment.

## Verification boundaries

The geometry checks build the default assembly first and consume its imported spur gears after placement.
They verify every owned sketch has FullyConstrained status, zero DOF, no redundancy and no conflicts.
They tessellate each part with VerifyAll and check the actual placed gear sections, bores and dimensions.
Scale endpoints and another crank quadrant exercise the same construction at different dimensions and poses.
Analytic checks sample a full revolution for linkage closure, pedal clearance, and rotation ratios.
Collision checks sample the real linkage solids every five degrees at scales0.8,1 and1.2.
Each candidate pair with positive axial overlap is checked using real XZ sections.
Nominal bearing seats share circular boundaries; their analytic diameters and centers are checked separately.
An additional Decad interference check covers the footman and input shaft at90 degrees.
The footman runs5*Scale millimeters in front of the shaft end; its pins bridge the gaps to crank and treadle.
These sampled checks do not prove continuous clearance between samples.

The 0.3 mm center-distance allowance changes the operating pressure angle slightly. Section checks sample
tooth phases for both stages; they do not replace a loaded mesh trial, lubrication choice or backlash measurement.
The default assembly check reads the real rim's mass properties and moving parts' volumes and centroids.
At scale1 it compares rim kinetic energy between60 and30rpm with a conservative return-energy budget.
The budget sums each moving part's full-cycle potential-energy range,5N residual downward foot force
at150mm from the heel, and0.05Nm combined load/friction torque over a full crank revolution.
It requires at least25% margin and checks that a1200kg/m³ printed rim fails the same load case.
Steel density is7800kg/m³ and timber density is600kg/m³; moving bearing envelopes count as solid steel.
The calculation omits other rotating inertia from the available energy. These assumptions are not measured loads.
Only the nominal scale receives this energy acceptance; scale variants receive geometry checks.
The nominal result is0.800J available versus0.604J budget, a factor of1.325.
The model does not calculate contact forces, actual foot effort or fatigue life.
The flywheel must carry the crank over dead center and back-drive the pedal on its return stroke.
The mechanism has no one-way clutch; losing momentum can stall or reverse it.
The animation prescribes crank angle and does not establish pedal effort or flywheel mass.
The cover encloses gears only. Physical access testing must determine the remaining guard design.

The construction uses no Fusion-specific features. Downstream Fusion code belongs in
`fusion360-gear-generator`, following this repository's geometry.

## References

[Ashford's quill instructions](https://www.ashford.co.nz/instructions/QS.pdf) describe the yarn angles
for drafting and winding. [Ashford's treadle-kit instructions](https://www.ashford.co.nz/instructions/DTKTD.pdf)
identify the crank, rods and supports used in a treadle mechanism. The dimensions, calculated four-bar
linkage and two spur stages in this model are new construction choices.
