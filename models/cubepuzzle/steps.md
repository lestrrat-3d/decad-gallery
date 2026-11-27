# Cube Puzzle Decad steps

These draft steps implement `instructions.md`. Expose `Build(ctx context.Context, doc *decad.Document, params Params) (*Assembly, error)`. Define an empty `Params`
and `DefaultParams() Params` for the fixed size. Define `Assembly` with `Hub *decad.Body` and slices `Centers`,
`Edges`, `Corners`, `Disks`, `FlangedSleeves`, `Sleeves`, `Bolts`, and `Nuts`, each holding `*decad.Body`. The
document owns the named body containers.
Expose `NewDocument(opts ...decad.DocumentOption) *decad.Document`, applying `decad.WithChainTolerance` of
0.01 mm before the supplied options. Build uses a document made by this constructor. Return contextual errors. Keep original bodies available with `decad.Copy` before any consuming Boolean operation.
Every sketch, including sketches returned by shared tools, must be fully constrained before use. Use the shared
`internal/sketchcheck`, `internal/fastenertool`, `internal/m2fastener`, and `models/fastener` packages.

1. Build a layer from a frame with origin at zero, U along its axis and V perpendicular. Use the contract's polygon,
   `o`, and `cap`. Constrain its axis line horizontal and on the origin; constrain the three axial-position lines
   vertical and their distances from the origin; constrain the outer radial line horizontal at 43; constrain the
   sloping line by its angle to the axis line and its perpendicular distance from the origin. Use
   `sketch.CreateFillet` on the upper vertical and sloping lines at `(high,rHigh)` with radius `0.75+o`, and on the
   sloping and lower vertical lines at `(low,rLow)` with radius `0.75-o`. Solve, verify, and revolve the single profile 360 degrees around U.
2. Build a sphere from a constrained semicircle and its diameter. Fix the diameter on the origin, constrain its arc's
   radius, and revolve the profile 360 degrees around the diameter. Make a fresh copy before every consuming cut.
3. Create material layers for +X, +Y, +Z. Intersect fresh copies to form the positive corner. Create new +X and +Y
   layers and intersect them for the positive edge. Cut new -0.25 layers along +Z and -Z from the edge. Create a +Z
   material layer and cut new -0.25 layers on +X, -X, +Y and -Y for the positive center. Cut a radius-12 sphere from
   each. Require each result to be solid.
4. Build the +Z center rotor by cutting cylinders from the center: radius 45 from z=-40 to 16; radius 2.9 from z=15
   to 31; radius 6.4 from z=25.6 to 31. Use `fastenertool.Cylinder` with each sketch frame normal +Z. Require a solid.
5. Build a radius-11.5 sphere. For each signed coordinate axis, cut a radius-30 cylinder with its start cap centered
   at 10.5 on that axis, normal along that axis and length 15. These six cuts make the flat-faced hub. Require a solid.
6. For each face, use these outward axis and tangent pairs: (+Z,+X), (-Z,+X), (+X,+Y), (-X,+Y), (+Y,+Z), (-Y,+Z).
   Set the M2 site head to 30 times the outward axis, direction to minus the outward axis, corner to tangent, length
   25 and cavity 19.5. Validate the site. For the pocket cutter copy the site, set its head to the original site's
   `At(CavityFar())`, reverse its direction, and set cavity to zero. Build `RoundedHexPocketTool` of depth 4 and
   round 0.5. Cut the pocket, then cut the original site's `RoundedHoleTool`. Require the hub still solid. Place a
   nominal M2 bolt and nut at the original site. Verify all returned sketches and retain the sites for tests.
7. Build the disk with a radius-6 cylinder from z=26 to z=30. Cut the +Z site's M2 `PocketTool` and `HoleTool` from
   it. Require a solid, and check the bolt's top is flush with the disk top.
8. For each of the two contract profiles, create a frame with origin zero, U=+Z and V=+X. Build a rounded revolution
   with `fastenertool.RoundedRevolve`. The profiles describe custom metal sleeves. Keep the two source bodies to copy.
9. Place eight corners by the combinations of signed X, Y, Z. Set the first frame axes to (signed X, signed Y),
   exchanging them when their sign product with signed Z is negative, to preserve a right-handed frame. Place four
   edges for each of axis pairs (X,Y), (X,Z), and (Y,Z), with all sign combinations as the frame U,V. Place centers,
   disks and sleeves with U=tangent and V=outward-axis cross tangent for the six face pairs above. Use `decad.Copy`
   and the frame transform. Use the already placed bolt and nut bodies from step 6.
10. Put all live bodies in named containers `Hubs`, `Centers`, `Edges`, `Corners`, `Disks`, `Flanged Sleeves`,
    `Sleeves`, `Bolts`, and `Nuts`. Check all body placements and `doc.VerifyContainers()`. Build only the returned
    bodies as live results; do not leave cutter or prototype copies in the document.

Tests must use actual constructed Decad bodies. Verify every sketch and mesh, and test each dimensional requirement,
the six nut and bolt positions, flush heads, pocket separation and the assembled fit. Check every printed part's
qualifying surface edges and print sections, exempting only the documented nut-mating corner pairs. Run the sampled
turn and insertion checks and extraction witnesses in the contract. Do not claim the sampled checks prove arbitrary
continuous paths or physical assembly. Generate real-body images only after these checks pass.
