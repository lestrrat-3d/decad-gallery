# Spinning-wheel mounting and occupied-volume audit

## Baseline and method

Audit Gallery commit `84493de`, the181actual Decad bodies, and the default parameter case.
The diagnostic overlay is `.tmp/spinningwheel-part-home-overlay.json`.
The bounded log is `.tmp/spinningwheel-part-home-inventory.log`.
Every mesh passes native boundary and volume verification before inspection.
Charge both meshes' displacement bounds before reporting an interior witness.
The probe found92part pairs with material overlap greater than0.05mm beyond those bounds.
This is a lower-bound inventory, not proof that unreported pairs are clear.
A0.3mm proximity graph connects all181bodies, but proximity does not prove a mechanical mount.

## Missing or incomplete mounts

| Parts | Geometry at the audited baseline | Required home |
| --- | --- | --- |
| gimbal-crank-retainer-1/2 | ID10 rings surround a5mm pin and stand0.2mm off the bridge. No screws or groove capture them. | Bolt both end plates through the bridge; clear the crank pin and cover only the outer bearing race. |
| footman-cross-retainer-1/2 | The same unsupported rings stand off the rod boss. | Bolt shaped end plates through the boss and rod tenon; clear the articulating fork. |
| brake-insert-retaining-plate | A thin plate touches timber without modeled fasteners. | Give it two screws outside the insert pocket and matching pilot holes. |
| yarn hooks and brake guide stems | Several shanks stop at wood surfaces; other shanks enter wood without holes. | Extend each shank into a matching drilled seat. |
| Scotch spring attachments | Short cylinders overlap spring wire, anchor stems, cord and eye. They do not form captive joints. | Form one spring with end eyes; retain its front eye with a clevis pin and interlock its rear eye with the cord's terminal eye. |
| hinge leaves and pins | Leaves have no wood fasteners; plain hinge pins have no axial retention. | Use bored mounting leaves, wood screws, headed hinge pins and retaining nuts. |
| lower treadle connector | Two intersecting cylinders impersonate one flexible piece and overlap wood. | Build one native flexible piece with bored end seats and real cross-clamp bolts. Rebuild this deforming body for animation poses. |
| Alignment and maiden-foot bolts | Clearance holes contain the bolt tips, but no nuts or inserts receive their threads. | Add captive square M6 inserts with real seats in the wedge/upper bar. |
| Maiden posts and wheel bearing caps | Flat stock ends merely touch their feet/legs. | Add fitted glued tenons and sockets. |
| Wheel hub, crank, gimbal bearing and rod cross bearings | Several axial gaps have no thrust spacer or stop. | Add the missing spacers and crank-root collars. |
| Shaft collars | Plain annuli have no modeled way to lock onto journals. | Bore each collar and add a recessed radial grub screw. |
| Local D-bore hubs | Round journals block sliding them onto their intended shaft flats. | Use removable keys and bores that clear a round journal. Keep the stock bearing diameters. |

Purchased bearing and clamp-collar envelopes still need explicit seats/shafts in the mounting inventory.
Their internal races, seals and thread details remain outside the purchased-part envelope contract.
The feed yarn has a deliberate free input end; it is process material, not a loose hardware part.

## Confirmed overlap families

| Interface | Required correction |
| --- | --- |
| Four leg/rail intersections | Cut fitted rail pockets into the legs; retain actual wood around each pocket. |
| Six spokes in rim and hub | Cut twelve real spoke sockets, preserving the V groove and positive shaft coupling. |
| Gimbal bridge and cheeks | Produce one machined clevis body, not three interpenetrating objects. |
| Rod and upper boss | Cut the boss's rod socket and provide the retainer cross-bolt passage. |
| Treadle board, heel rail and shaft | Cut a half-lap joint and the actual shaft passage through the board. |
| Rear actuating rail, crossarm and connector block | Cut fitted wood joints without severing a timber. |
| Flexible connector and its wood ends | Drill connector sockets and keep the cross-clamp holes aligned. |
| Clamp pins and heads | Model each bolt as one stepped native solid. |
| Hinge knuckles, brackets and leaves | Model each machined hinge leaf as one solid; cut its recess in the wood. |
| Upper bar, jack arm and rear brake crossbar | Cut matching wood seats instead of overlapping boards. |
| Maiden feet/bolts and brake timbers | Clear the rear support for the foot and bolts; move the front upright clear of the front foot/heads. |
| Flyer arms and crossbar | Cut arm sockets into the thicker crossbar. |
| Front yarn-guide stalk and flyer | Drill a stalk seat in the crossbar. |
| Yarn hooks and stems | Return one formed metal guide per station. |
| Yarn pack and witness marker | Use a thin curved label on the actual pack surface. |
| Feed strand and yarn pack | Put the strand centreline one wire radius outside the represented wound material. |
| Feed-strand and brake-cord bends | Return a continuous native cord body for each physical cord. |
| Spring coils, end links, eyes and anchor stems | Form one spring per side and use physically separate, captured terminal fittings. |

## Acceptance

Complete the real default assembly through the remaining generic native repairs in
[the joinery handoff](handoff-spinning-wheel-joinery.md).
Do not substitute drawn joints or weaken mesh verification.
Emit construction changes from updated instructions and steps with fresh emitters.
Give every returned part a named parent and a mount type, with no unexplained exceptions.
Check the actual seat, bore, fastener, captive-loop or glue-joint geometry for each mount.
Verify that mating parts occupy separate material volumes, including moving and fixed parts.
Repeat the real default and parameter branches, then the full-turn pose checks and renderer integration.
Regenerate all actual-body images and the animation after the corrected assembly passes.
